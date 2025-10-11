package a2a

import (
	"fmt"
	"strings"
)

// ============================================================================
// A2A MESSAGE HELPER FUNCTIONS
// Utilities for working with A2A protocol messages
// ============================================================================

// ExtractTextFromMessage extracts text content from an A2A message
// Returns the first text part found, or empty string if none
func ExtractTextFromMessage(msg Message) string {
	for _, part := range msg.Parts {
		if part.Type == PartTypeText {
			return part.Text
		}
	}
	return ""
}

// ExtractAllTextFromMessage extracts all text content from an A2A message
// Concatenates all text parts with newlines
func ExtractAllTextFromMessage(msg Message) string {
	var texts []string
	for _, part := range msg.Parts {
		if part.Type == PartTypeText {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// CreateUserMessage creates a user message with text content
func CreateUserMessage(text string) Message {
	return CreateTextMessage(MessageRoleUser, text)
}

// CreateAssistantMessage creates an assistant message with text content
func CreateAssistantMessage(text string) Message {
	return CreateTextMessage(MessageRoleAssistant, text)
}

// AddTextPart adds a text part to an existing message
func AddTextPart(msg *Message, text string) {
	msg.Parts = append(msg.Parts, Part{
		Type: PartTypeText,
		Text: text,
	})
}

// HasTextContent checks if a message contains any text content
func HasTextContent(msg Message) bool {
	for _, part := range msg.Parts {
		if part.Type == PartTypeText && part.Text != "" {
			return true
		}
	}
	return false
}

// GetTextParts returns all text parts from a message
func GetTextParts(msg Message) []string {
	var texts []string
	for _, part := range msg.Parts {
		if part.Type == PartTypeText {
			texts = append(texts, part.Text)
		}
	}
	return texts
}

// MessageToText converts an A2A message to a simple text representation
// Used for logging, debugging, and simple text-based operations
func MessageToText(msg Message) string {
	role := string(msg.Role)
	text := ExtractTextFromMessage(msg)

	if text == "" {
		return fmt.Sprintf("[%s: <no text>]", role)
	}

	// Truncate long messages for readability
	if len(text) > 100 {
		return fmt.Sprintf("[%s: %s...]", role, text[:100])
	}

	return fmt.Sprintf("[%s: %s]", role, text)
}

// MessagesToText converts a slice of A2A messages to text representation
func MessagesToText(messages []Message) []string {
	texts := make([]string, len(messages))
	for i, msg := range messages {
		texts[i] = MessageToText(msg)
	}
	return texts
}

// FilterMessagesByRole filters messages by role
func FilterMessagesByRole(messages []Message, role MessageRole) []Message {
	var filtered []Message
	for _, msg := range messages {
		if msg.Role == role {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// GetLastUserMessage returns the last user message from a conversation
func GetLastUserMessage(messages []Message) (Message, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == MessageRoleUser {
			return messages[i], true
		}
	}
	return Message{}, false
}

// GetLastAssistantMessage returns the last assistant message from a conversation
func GetLastAssistantMessage(messages []Message) (Message, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == MessageRoleAssistant {
			return messages[i], true
		}
	}
	return Message{}, false
}
