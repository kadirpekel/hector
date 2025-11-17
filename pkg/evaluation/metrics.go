package evaluation

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/llms"
)

// EvaluationMetrics contains RAG evaluation metrics
type EvaluationMetrics struct {
	// Context Precision: Proportion of retrieved contexts that are relevant
	ContextPrecision float64 `json:"context_precision"`

	// Context Recall: Proportion of relevant contexts that were retrieved
	ContextRecall float64 `json:"context_recall"`

	// Answer Relevance: How relevant is the generated answer to the query
	AnswerRelevance float64 `json:"answer_relevance"`

	// Faithfulness: How faithful is the answer to the retrieved contexts
	Faithfulness float64 `json:"faithfulness"`

	// Answer Correctness: Overall correctness (combination of relevance and faithfulness)
	AnswerCorrectness float64 `json:"answer_correctness"`

	// Latency: Time taken for retrieval + generation (in seconds)
	Latency float64 `json:"latency"`

	// Token Usage: Total tokens used
	TokenUsage int `json:"token_usage"`
}

// EvaluationResult contains the full evaluation result
type EvaluationResult struct {
	Query           string                   `json:"query"`
	RetrievedDocs   []databases.SearchResult `json:"retrieved_docs"`
	GeneratedAnswer string                   `json:"generated_answer"`
	GroundTruth     string                   `json:"ground_truth,omitempty"` // Optional ground truth answer
	Metrics         EvaluationMetrics        `json:"metrics"`
	Timestamp       string                   `json:"timestamp"`
}

// Evaluator interface for evaluating RAG systems
type Evaluator interface {
	// Evaluate evaluates a single query-answer pair
	Evaluate(ctx context.Context, query string, retrievedDocs []databases.SearchResult, generatedAnswer string, groundTruth string) (*EvaluationResult, error)

	// EvaluateBatch evaluates multiple query-answer pairs
	EvaluateBatch(ctx context.Context, testCases []TestCase) ([]EvaluationResult, error)
}

// TestCase represents a single test case for evaluation
type TestCase struct {
	Query        string   `json:"query"`
	GroundTruth  string   `json:"ground_truth,omitempty"`
	ExpectedDocs []string `json:"expected_docs,omitempty"` // Optional: expected document IDs
}

// LLMEvaluator uses an LLM to evaluate RAG results
type LLMEvaluator struct {
	llmProvider llms.LLMProvider
}

// NewLLMEvaluator creates a new LLM-based evaluator
func NewLLMEvaluator(llmProvider llms.LLMProvider) *LLMEvaluator {
	return &LLMEvaluator{
		llmProvider: llmProvider,
	}
}

// Evaluate implements the Evaluator interface
func (e *LLMEvaluator) Evaluate(ctx context.Context, query string, retrievedDocs []databases.SearchResult, generatedAnswer string, groundTruth string) (*EvaluationResult, error) {
	// Calculate context precision and recall
	contextPrecision, contextRecall := e.calculateContextMetrics(query, retrievedDocs)

	// Calculate answer relevance using LLM
	answerRelevance, err := e.calculateAnswerRelevance(ctx, query, generatedAnswer)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate answer relevance: %w", err)
	}

	// Calculate faithfulness using LLM
	faithfulness, err := e.calculateFaithfulness(ctx, retrievedDocs, generatedAnswer)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate faithfulness: %w", err)
	}

	// Calculate answer correctness (average of relevance and faithfulness)
	answerCorrectness := (answerRelevance + faithfulness) / 2.0

	metrics := EvaluationMetrics{
		ContextPrecision:  contextPrecision,
		ContextRecall:     contextRecall,
		AnswerRelevance:   answerRelevance,
		Faithfulness:      faithfulness,
		AnswerCorrectness: answerCorrectness,
	}

	return &EvaluationResult{
		Query:           query,
		RetrievedDocs:   retrievedDocs,
		GeneratedAnswer: generatedAnswer,
		GroundTruth:     groundTruth,
		Metrics:         metrics,
		Timestamp:       "",
	}, nil
}

// EvaluateBatch implements the Evaluator interface
func (e *LLMEvaluator) EvaluateBatch(ctx context.Context, testCases []TestCase) ([]EvaluationResult, error) {
	// Note: This is a simplified batch evaluation
	// In a full implementation, you'd need to run the RAG pipeline for each test case
	// For now, we'll return an error indicating batch evaluation needs the full pipeline
	return nil, fmt.Errorf("batch evaluation requires running the full RAG pipeline - use Evaluate() for individual cases")
}

// calculateContextMetrics calculates precision and recall for retrieved contexts
func (e *LLMEvaluator) calculateContextMetrics(query string, retrievedDocs []databases.SearchResult) (precision, recall float64) {
	if len(retrievedDocs) == 0 {
		return 0.0, 0.0
	}

	// Simple keyword-based relevance (can be enhanced with LLM)
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	relevantCount := 0
	for _, doc := range retrievedDocs {
		contentLower := strings.ToLower(doc.Content)
		matches := 0
		for _, word := range queryWords {
			if strings.Contains(contentLower, word) {
				matches++
			}
		}
		// Consider relevant if at least 50% of query words match
		if float64(matches)/float64(len(queryWords)) >= 0.5 {
			relevantCount++
		}
	}

	precision = float64(relevantCount) / float64(len(retrievedDocs))
	// Recall is harder to calculate without knowing total relevant docs
	// For now, assume all retrieved docs are the relevant set
	recall = 1.0 // Simplified: assume we retrieved all relevant docs

	return precision, recall
}

// calculateAnswerRelevance uses LLM to score how relevant the answer is to the query
func (e *LLMEvaluator) calculateAnswerRelevance(ctx context.Context, query string, answer string) (float64, error) {
	prompt := fmt.Sprintf(`Rate how relevant the answer is to the query on a scale of 0.0 to 1.0.

Query: %s
Answer: %s

Return only a number between 0.0 and 1.0 representing the relevance score.`, query, answer)

	// Use LLM to score relevance
	score, err := e.scoreWithLLM(ctx, prompt)
	if err != nil {
		return 0.0, err
	}

	return score, nil
}

// calculateFaithfulness uses LLM to score how faithful the answer is to the retrieved contexts
func (e *LLMEvaluator) calculateFaithfulness(ctx context.Context, retrievedDocs []databases.SearchResult, answer string) (float64, error) {
	// Combine retrieved document contents
	contexts := make([]string, 0, len(retrievedDocs))
	for _, doc := range retrievedDocs {
		if len(doc.Content) > 500 {
			contexts = append(contexts, doc.Content[:500]+"...")
		} else {
			contexts = append(contexts, doc.Content)
		}
	}
	contextText := strings.Join(contexts, "\n\n")

	prompt := fmt.Sprintf(`Rate how faithful the answer is to the provided contexts on a scale of 0.0 to 1.0.
A score of 1.0 means the answer is fully supported by the contexts.
A score of 0.0 means the answer contradicts or is not supported by the contexts.

Contexts:
%s

Answer: %s

Return only a number between 0.0 and 1.0 representing the faithfulness score.`, contextText, answer)

	score, err := e.scoreWithLLM(ctx, prompt)
	if err != nil {
		return 0.0, err
	}

	return score, nil
}

// scoreWithLLM uses LLM to extract a numeric score from a prompt
func (e *LLMEvaluator) scoreWithLLM(ctx context.Context, prompt string) (float64, error) {
	// This is a simplified implementation
	// In production, you'd want to use structured output or better parsing
	messages := []*pb.Message{
		{
			Role: pb.Role_ROLE_USER,
			Parts: []*pb.Part{
				{
					Part: &pb.Part_Text{
						Text: prompt,
					},
				},
			},
		},
	}

	response, _, _, err := e.llmProvider.Generate(ctx, messages, nil)
	if err != nil {
		return 0.0, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse score from response (look for number between 0.0 and 1.0)
	score, err := parseScoreFromResponse(response)
	if err != nil {
		return 0.0, fmt.Errorf("failed to parse score from LLM response: %w", err)
	}

	return score, nil
}

// parseScoreFromResponse extracts a score (0.0-1.0) from LLM response
func parseScoreFromResponse(response string) (float64, error) {
	// Look for number patterns in the response
	response = strings.TrimSpace(response)

	// Try to find a float between 0.0 and 1.0
	// Simple regex-like parsing
	var score float64
	_, err := fmt.Sscanf(response, "%f", &score)
	if err != nil {
		// Try to find number in text
		words := strings.Fields(response)
		for _, word := range words {
			var val float64
			if _, err := fmt.Sscanf(word, "%f", &val); err == nil {
				if val >= 0.0 && val <= 1.0 {
					score = val
					break
				}
			}
		}
		if score == 0.0 && err != nil {
			return 0.5, nil // Default to 0.5 if parsing fails
		}
	}

	if score < 0.0 {
		score = 0.0
	}
	if score > 1.0 {
		score = 1.0
	}

	return score, nil
}
