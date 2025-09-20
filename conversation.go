package hector

import (
	"fmt"
	"strings"
	"time"
)

// ============================================================================
// CONVERSATION HISTORY & MEMORY
// ============================================================================

// ConversationHistory manages conversation state and history
type ConversationHistory struct {
	SessionID   string                 `json:"session_id"`
	Messages    []Message              `json:"messages"`
	Context     map[string]interface{} `json:"context"`
	UserProfile UserProfile            `json:"user_profile"`
	LastUpdated time.Time              `json:"last_updated"`
	MaxMessages int                    `json:"max_messages"`
}

// Message represents a single message in the conversation
type Message struct {
	ID          string                 `json:"id"`
	Role        string                 `json:"role"` // "user", "assistant", "system"
	Content     string                 `json:"content"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ToolCalls   []ToolCall             `json:"tool_calls,omitempty"`
	ToolResults []ToolResult           `json:"tool_results,omitempty"`
}

// ToolCall represents a tool call made by the agent
type ToolCall struct {
	ID        string                 `json:"id"`
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	Timestamp time.Time              `json:"timestamp"`
}

// UserProfile stores user preferences and context
type UserProfile struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name,omitempty"`
	Preferences map[string]interface{} `json:"preferences"`
	Context     map[string]interface{} `json:"context"`
	LastSeen    time.Time              `json:"last_seen"`
}

// AgentMemory manages different types of memory
type AgentMemory struct {
	ShortTerm map[string]interface{} `json:"short_term"` // Current session context
	LongTerm  map[string]interface{} `json:"long_term"`  // Persistent user preferences
	Episodic  []Episode              `json:"episodic"`   // Conversation episodes
	Semantic  map[string]interface{} `json:"semantic"`   // Knowledge and facts
}

// Episode represents a conversation episode
type Episode struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	Summary   string                 `json:"summary"`
	Messages  []Message              `json:"messages"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Outcome   string                 `json:"outcome"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ============================================================================
// CONVERSATION HISTORY METHODS
// ============================================================================

// NewConversationHistory creates a new conversation history
func NewConversationHistory(sessionID string) *ConversationHistory {
	return &ConversationHistory{
		SessionID:   sessionID,
		Messages:    make([]Message, 0),
		Context:     make(map[string]interface{}),
		UserProfile: UserProfile{ID: sessionID, Preferences: make(map[string]interface{}), Context: make(map[string]interface{})},
		LastUpdated: time.Now(),
		MaxMessages: 100, // Keep last 100 messages
	}
}

// AddMessage adds a message to the conversation history
func (ch *ConversationHistory) AddMessage(role, content string, metadata map[string]interface{}) *Message {
	message := Message{
		ID:        generateMessageID(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	ch.Messages = append(ch.Messages, message)
	ch.LastUpdated = time.Now()

	// Trim messages if we exceed the limit
	if len(ch.Messages) > ch.MaxMessages {
		ch.Messages = ch.Messages[len(ch.Messages)-ch.MaxMessages:]
	}

	return &message
}

// AddToolCall adds a tool call to the last message
func (ch *ConversationHistory) AddToolCall(messageID string, toolName string, arguments map[string]interface{}) {
	if len(ch.Messages) == 0 {
		return
	}

	// Find the message and add tool call
	for i := range ch.Messages {
		if ch.Messages[i].ID == messageID {
			toolCall := ToolCall{
				ID:        generateToolCallID(),
				ToolName:  toolName,
				Arguments: arguments,
				Timestamp: time.Now(),
			}
			ch.Messages[i].ToolCalls = append(ch.Messages[i].ToolCalls, toolCall)
			break
		}
	}
}

// AddToolResult adds a tool result to the last message
func (ch *ConversationHistory) AddToolResult(messageID string, result ToolResult) {
	if len(ch.Messages) == 0 {
		return
	}

	// Find the message and add tool result
	for i := range ch.Messages {
		if ch.Messages[i].ID == messageID {
			ch.Messages[i].ToolResults = append(ch.Messages[i].ToolResults, result)
			break
		}
	}
}

// GetRecentMessages returns the last N messages
func (ch *ConversationHistory) GetRecentMessages(n int) []Message {
	if n <= 0 || len(ch.Messages) == 0 {
		return []Message{}
	}

	start := len(ch.Messages) - n
	if start < 0 {
		start = 0
	}

	return ch.Messages[start:]
}

// GetContextForLLM returns formatted context for LLM
func (ch *ConversationHistory) GetContextForLLM(maxMessages int) string {
	recentMessages := ch.GetRecentMessages(maxMessages)

	var context strings.Builder
	context.WriteString("Conversation History:\n")

	for _, msg := range recentMessages {
		context.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))

		// Add tool calls if any
		for _, toolCall := range msg.ToolCalls {
			context.WriteString(fmt.Sprintf("  [Tool Call: %s]\n", toolCall.ToolName))
		}

		// Add tool results if any
		for _, toolResult := range msg.ToolResults {
			if toolResult.Success {
				context.WriteString(fmt.Sprintf("  [Tool Result: %s]\n", toolResult.Content))
			}
		}
	}

	return context.String()
}

// UpdateUserProfile updates user profile information
func (ch *ConversationHistory) UpdateUserProfile(updates map[string]interface{}) {
	for key, value := range updates {
		ch.UserProfile.Preferences[key] = value
	}
	ch.UserProfile.LastSeen = time.Now()
}

// SetContext sets conversation context
func (ch *ConversationHistory) SetContext(key string, value interface{}) {
	ch.Context[key] = value
	ch.LastUpdated = time.Now()
}

// GetContext gets conversation context
func (ch *ConversationHistory) GetContext(key string) (interface{}, bool) {
	value, exists := ch.Context[key]
	return value, exists
}

// ============================================================================
// AGENT MEMORY METHODS
// ============================================================================

// NewAgentMemory creates a new agent memory
func NewAgentMemory() *AgentMemory {
	return &AgentMemory{
		ShortTerm: make(map[string]interface{}),
		LongTerm:  make(map[string]interface{}),
		Episodic:  make([]Episode, 0),
		Semantic:  make(map[string]interface{}),
	}
}

// StoreShortTerm stores information in short-term memory
func (am *AgentMemory) StoreShortTerm(key string, value interface{}) {
	am.ShortTerm[key] = value
}

// StoreLongTerm stores information in long-term memory
func (am *AgentMemory) StoreLongTerm(key string, value interface{}) {
	am.LongTerm[key] = value
}

// StoreSemantic stores semantic knowledge
func (am *AgentMemory) StoreSemantic(key string, value interface{}) {
	am.Semantic[key] = value
}

// AddEpisode adds a conversation episode
func (am *AgentMemory) AddEpisode(episode Episode) {
	am.Episodic = append(am.Episodic, episode)

	// Keep only last 50 episodes
	if len(am.Episodic) > 50 {
		am.Episodic = am.Episodic[len(am.Episodic)-50:]
	}
}

// GetRelevantEpisodes returns episodes relevant to current context
func (am *AgentMemory) GetRelevantEpisodes(query string, limit int) []Episode {
	// Simple implementation - in practice, you'd use semantic search
	if limit <= 0 || len(am.Episodic) == 0 {
		return []Episode{}
	}

	start := len(am.Episodic) - limit
	if start < 0 {
		start = 0
	}

	return am.Episodic[start:]
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// generateToolCallID generates a unique tool call ID
func generateToolCallID() string {
	return fmt.Sprintf("tool_%d", time.Now().UnixNano())
}
