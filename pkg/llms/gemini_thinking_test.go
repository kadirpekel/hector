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
	mockResponse := []GeminiResponse{
		{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{"thought": "I need to think about this..."},
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

	// Verify chunks
	require.Len(t, receivedChunks, 3)

	// First chunk should be thinking
	assert.Equal(t, "thinking", receivedChunks[0].Type)
	assert.Equal(t, "I need to think about this...", receivedChunks[0].Text)

	// Second chunk should be text
	assert.Equal(t, "text", receivedChunks[1].Type)
	assert.Equal(t, "Here is the answer.", receivedChunks[1].Text)

	// Third chunk should be done
	assert.Equal(t, "done", receivedChunks[2].Type)
}
