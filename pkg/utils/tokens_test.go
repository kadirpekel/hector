package utils

import (
	"testing"
)

func TestNewTokenCounter(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		wantError bool
	}{
		{
			name:      "GPT-4o model",
			model:     "gpt-4o",
			wantError: false,
		},
		{
			name:      "GPT-4 model",
			model:     "gpt-4",
			wantError: false,
		},
		{
			name:      "GPT-3.5-turbo model",
			model:     "gpt-3.5-turbo",
			wantError: false,
		},
		{
			name:      "Claude model (uses fallback)",
			model:     "claude-3-5-sonnet",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter, err := NewTokenCounter(tt.model)
			if (err != nil) != tt.wantError {
				t.Errorf("NewTokenCounter() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && counter == nil {
				t.Error("NewTokenCounter() returned nil counter")
			}
			if counter != nil && counter.GetModel() != tt.model {
				t.Errorf("NewTokenCounter() model = %v, want %v", counter.GetModel(), tt.model)
			}
		})
	}
}

func TestTokenCounter_Count(t *testing.T) {
	counter, err := NewTokenCounter("gpt-4o")
	if err != nil {
		t.Fatalf("Failed to create token counter: %v", err)
	}

	tests := []struct {
		name      string
		text      string
		minTokens int // Minimum expected tokens
		maxTokens int // Maximum expected tokens
	}{
		{
			name:      "Empty string",
			text:      "",
			minTokens: 0,
			maxTokens: 0,
		},
		{
			name:      "Simple sentence",
			text:      "Hello, world!",
			minTokens: 3,
			maxTokens: 5,
		},
		{
			name:      "Longer text",
			text:      "This is a longer sentence with more words to count tokens accurately.",
			minTokens: 12,
			maxTokens: 18,
		},
		{
			name:      "Code snippet",
			text:      "func main() { fmt.Println(\"Hello\") }",
			minTokens: 8,
			maxTokens: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := counter.Count(tt.text)
			if count < tt.minTokens || count > tt.maxTokens {
				t.Errorf("Count() = %v, want between %v and %v for text: %q",
					count, tt.minTokens, tt.maxTokens, tt.text)
			}
		})
	}
}

func TestTokenCounter_CountMessages(t *testing.T) {
	counter, err := NewTokenCounter("gpt-4o")
	if err != nil {
		t.Fatalf("Failed to create token counter: %v", err)
	}

	tests := []struct {
		name      string
		messages  []Message
		minTokens int
		maxTokens int
	}{
		{
			name:      "Empty messages",
			messages:  []Message{},
			minTokens: 3, // Reply priming tokens
			maxTokens: 3,
		},
		{
			name: "Single message",
			messages: []Message{
				{Role: "user", Content: "Hello"},
			},
			minTokens: 5,
			maxTokens: 10,
		},
		{
			name: "Conversation",
			messages: []Message{
				{Role: "user", Content: "What is AI?"},
				{Role: "assistant", Content: "AI stands for Artificial Intelligence."},
				{Role: "user", Content: "Tell me more."},
			},
			minTokens: 15,
			maxTokens: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := counter.CountMessages(tt.messages)
			if count < tt.minTokens || count > tt.maxTokens {
				t.Errorf("CountMessages() = %v, want between %v and %v",
					count, tt.minTokens, tt.maxTokens)
			}
		})
	}
}

func TestTokenCounter_FitWithinLimit(t *testing.T) {
	counter, err := NewTokenCounter("gpt-4o")
	if err != nil {
		t.Fatalf("Failed to create token counter: %v", err)
	}

	messages := []Message{
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Message 2"},
		{Role: "assistant", Content: "Response 2"},
		{Role: "user", Content: "Message 3"},
	}

	tests := []struct {
		name         string
		messages     []Message
		maxTokens    int
		expectEmpty  bool
		expectAllFit bool
	}{
		{
			name:         "Very low limit",
			messages:     messages,
			maxTokens:    5,
			expectEmpty:  true,
			expectAllFit: false,
		},
		{
			name:         "Moderate limit",
			messages:     messages,
			maxTokens:    50,
			expectEmpty:  false,
			expectAllFit: false,
		},
		{
			name:         "High limit",
			messages:     messages,
			maxTokens:    1000,
			expectEmpty:  false,
			expectAllFit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fitted := counter.FitWithinLimit(tt.messages, tt.maxTokens)

			if tt.expectEmpty && len(fitted) > 0 {
				t.Errorf("FitWithinLimit() expected empty result, got %d messages", len(fitted))
			}

			if tt.expectAllFit && len(fitted) != len(tt.messages) {
				t.Errorf("FitWithinLimit() expected all messages to fit, got %d/%d",
					len(fitted), len(tt.messages))
			}

			// Verify fitted messages are within limit
			if len(fitted) > 0 {
				tokenCount := counter.CountMessages(fitted)
				if tokenCount > tt.maxTokens {
					t.Errorf("FitWithinLimit() result has %d tokens, exceeds limit of %d",
						tokenCount, tt.maxTokens)
				}
			}

			// Verify messages are from the end (most recent)
			if len(fitted) > 0 && len(fitted) < len(tt.messages) {
				lastOriginal := tt.messages[len(tt.messages)-1]
				lastFitted := fitted[len(fitted)-1]
				if lastOriginal.Content != lastFitted.Content {
					t.Error("FitWithinLimit() should preserve most recent messages")
				}
			}
		})
	}
}

func TestTokenCounter_Caching(t *testing.T) {
	// Create first counter
	counter1, err := NewTokenCounter("gpt-4o")
	if err != nil {
		t.Fatalf("Failed to create first counter: %v", err)
	}

	// Create second counter with same model
	counter2, err := NewTokenCounter("gpt-4o")
	if err != nil {
		t.Fatalf("Failed to create second counter: %v", err)
	}

	// Both should work correctly
	text := "Test caching"
	count1 := counter1.Count(text)
	count2 := counter2.Count(text)

	if count1 != count2 {
		t.Errorf("Cached counters produced different results: %d vs %d", count1, count2)
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{
			name: "Empty string",
			text: "",
			want: 0,
		},
		{
			name: "4 characters",
			text: "test",
			want: 1,
		},
		{
			name: "8 characters",
			text: "testtest",
			want: 2,
		},
		{
			name: "10 characters",
			text: "hellohello",
			want: 2, // 10 / 4 = 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			if got != tt.want {
				t.Errorf("EstimateTokens() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEncodingForModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{
			name:  "GPT-4o",
			model: "gpt-4o",
			want:  "o200k_base",
		},
		{
			name:  "GPT-4",
			model: "gpt-4",
			want:  "cl100k_base",
		},
		{
			name:  "GPT-3.5-turbo",
			model: "gpt-3.5-turbo",
			want:  "cl100k_base",
		},
		{
			name:  "Claude",
			model: "claude-3-5-sonnet",
			want:  "cl100k_base",
		},
		{
			name:  "Unknown model",
			model: "unknown-model",
			want:  "cl100k_base", // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEncodingForModel(tt.model)
			if got != tt.want {
				t.Errorf("GetEncodingForModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkTokenCounter_Count(b *testing.B) {
	counter, err := NewTokenCounter("gpt-4o")
	if err != nil {
		b.Fatalf("Failed to create counter: %v", err)
	}

	text := "This is a benchmark test for token counting performance with a moderately long sentence."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.Count(text)
	}
}

func BenchmarkTokenCounter_CountMessages(b *testing.B) {
	counter, err := NewTokenCounter("gpt-4o")
	if err != nil {
		b.Fatalf("Failed to create counter: %v", err)
	}

	messages := []Message{
		{Role: "user", Content: "What is machine learning?"},
		{Role: "assistant", Content: "Machine learning is a subset of AI..."},
		{Role: "user", Content: "Can you give me an example?"},
		{Role: "assistant", Content: "Sure! Image recognition is a common example..."},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.CountMessages(messages)
	}
}

func BenchmarkEstimateTokens(b *testing.B) {
	text := "This is a benchmark test for the legacy token estimation function."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateTokens(text)
	}
}
