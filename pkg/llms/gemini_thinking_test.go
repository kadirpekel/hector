package llms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiProvider_GenerateStreaming_Thinking(t *testing.T) {
	// Mock Gemini API response with thinking content
	// According to Gemini docs, thinking parts have "text" field with content
	// and "thought": true boolean to mark it as thinking
	// See: https://ai.google.dev/gemini-api/docs/thinking
	mockResponse := []GeminiResponse{
		{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{"text": "I need to think about this...", "thought": true},
						},
					},
				},
			},
		},
		{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{"text": "Here is the answer."},
						},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Stream the response chunks in SSE format
		for _, resp := range mockResponse {
			data, _ := json.Marshal(resp)
			_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
		}
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		APIKey: "test-key",
		Model:  "gemini-2.0-flash-thinking-exp",
		Host:   server.URL,
	}

	provider, err := NewGeminiProviderFromConfig(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	messages := []*pb.Message{
		{
			Role: pb.Role_ROLE_USER,
			Parts: []*pb.Part{
				{Part: &pb.Part_Text{Text: "Hello"}},
			},
		},
	}

	chunks, err := provider.GenerateStreaming(ctx, messages, nil)
	require.NoError(t, err)

	var receivedChunks []StreamChunk
	for chunk := range chunks {
		receivedChunks = append(receivedChunks, chunk)
	}

	// Verify chunks: thinking, thinking_complete, text, done
	require.Len(t, receivedChunks, 4)

	// First chunk should be thinking (streaming)
	assert.Equal(t, "thinking", receivedChunks[0].Type)
	assert.Equal(t, "I need to think about this...", receivedChunks[0].Text)

	// Second chunk should be thinking_complete (closes the thinking block)
	assert.Equal(t, "thinking_complete", receivedChunks[1].Type)
	assert.Equal(t, "I need to think about this...", receivedChunks[1].Text)

	// Third chunk should be text
	assert.Equal(t, "text", receivedChunks[2].Type)
	assert.Equal(t, "Here is the answer.", receivedChunks[2].Text)

	// Fourth chunk should be done
	assert.Equal(t, "done", receivedChunks[3].Type)
}

func TestGeminiProvider_GenerateStreaming_ThinkingAfterToolCall(t *testing.T) {
	// Test case for the bug: Gemini marks post-tool-call responses as thinking
	// when they should be regular text. This causes duplicate greetings/confirmations.
	// See: https://ai.google.dev/gemini-api/docs/thinking
	mockResponse := []GeminiResponse{
		{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{"text": "Of course, John! Let me check the weather in Tokyo for you.", "thought": true},
						},
					},
				},
			},
		},
		{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{"functionCall": map[string]interface{}{
								"name": "WEATHERMAP_WEATHER",
								"args": map[string]interface{}{
									"location": "Tokyo",
								},
							}},
						},
					},
				},
			},
		},
		{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							// This is incorrectly marked as thinking by Gemini, but should be regular text
							{"text": "No problem, John! I've just checked the weather for you in Tokyo. Here's what it looks like:", "thought": true},
						},
					},
				},
			},
		},
		{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{"text": " The weather in Tokyo is currently seeing few clouds."},
						},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Stream the response chunks in SSE format
		for _, resp := range mockResponse {
			data, _ := json.Marshal(resp)
			_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
		}
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		APIKey: "test-key",
		Model:  "gemini-2.0-flash-thinking-exp",
		Host:   server.URL,
	}

	provider, err := NewGeminiProviderFromConfig(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	messages := []*pb.Message{
		{
			Role: pb.Role_ROLE_USER,
			Parts: []*pb.Part{
				{Part: &pb.Part_Text{Text: "what is weather like in tokyo today my friend. i am john, i am your friend"}},
			},
		},
	}

	chunks, err := provider.GenerateStreaming(ctx, messages, nil)
	require.NoError(t, err)

	var receivedChunks []StreamChunk
	for chunk := range chunks {
		receivedChunks = append(receivedChunks, chunk)
	}

	// Expected sequence:
	// 1. thinking: "Of course, John! Let me check..."
	// 2. thinking_complete: closes thinking block
	// 3. tool_call: WEATHERMAP_WEATHER
	// 4. text: "No problem, John!..." (should be text, NOT thinking, even though Gemini marks it as thought:true)
	// 5. text: " The weather in Tokyo..."
	// 6. done

	require.GreaterOrEqual(t, len(receivedChunks), 5, "Should have at least 5 chunks")

	// First chunk should be thinking (before tool call)
	assert.Equal(t, "thinking", receivedChunks[0].Type)
	assert.Contains(t, receivedChunks[0].Text, "Of course, John!")

	// Second chunk should be thinking_complete
	assert.Equal(t, "thinking_complete", receivedChunks[1].Type)

	// Third chunk should be tool_call
	assert.Equal(t, "tool_call", receivedChunks[2].Type)
	if receivedChunks[2].ToolCall != nil {
		assert.Equal(t, "WEATHERMAP_WEATHER", receivedChunks[2].ToolCall.Name)
	}

	// Fourth chunk should be TEXT (not thinking), even though Gemini marked it as thought:true
	// This is the key fix: post-tool-call "thinking" should be treated as regular text
	assert.Equal(t, "text", receivedChunks[3].Type)
	assert.Contains(t, receivedChunks[3].Text, "No problem, John!")

	// Last chunk should be done
	assert.Equal(t, "done", receivedChunks[len(receivedChunks)-1].Type)
}
