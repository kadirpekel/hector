package reasoning

import (
	"context"
	"fmt"
	"strings"
)

// ============================================================================
// EXTENSION SERVICE - END-TO-END LIKE TOOLS
// ============================================================================

// ExtensionDefinition defines an extension that can be registered
type ExtensionDefinition struct {
	Name         string                                                             // Extension name (e.g., "tools", "memory")
	Description  string                                                             // Description for LLM prompt
	OpenTag      string                                                             // Opening tag (e.g., "[[[tools]]]")
	CloseTag     string                                                             // Closing tag (e.g., "[[[/tools]]]")
	Processor    func(content string) (userDisplay string, rawData string)          // How to process matched content
	Executor     func(ctx context.Context, rawData string) (ExtensionResult, error) // How to execute the extension
	PromptFormat string                                                             // How to format this extension in prompts
}

// ExtensionResult represents the result of extension execution
type ExtensionResult struct {
	Name        string                 // Extension name
	Success     bool                   // Whether execution succeeded
	Content     string                 // Result content
	Error       string                 // Error message if failed
	UserDisplay string                 // What user saw during processing
	RawData     string                 // Original raw data
	Metadata    map[string]interface{} // Additional metadata
}

// ExtensionService manages extensions end-to-end like tools
type ExtensionService interface {
	// Register an extension definition
	RegisterExtension(ext ExtensionDefinition) error

	// Get available extensions
	GetAvailableExtensions() []ExtensionDefinition

	// Process LLM response and extract extension calls
	ProcessResponse(response string) (maskedResponse string, extractedExtensions []ExtensionCall)

	// Execute extracted extensions
	ExecuteExtensions(ctx context.Context, extensions []ExtensionCall) (map[string]ExtensionResult, error)

	// Execute single extension
	ExecuteExtension(ctx context.Context, name string, rawData string) (ExtensionResult, error)

	// Format extensions for prompt inclusion
	FormatForPrompt() string

	// Streaming support - detect extension markers in partial text
	ContainsMarker(text string) (found bool, markerPos int, markerLen int)
}

// ExtensionCall represents an extracted extension call from LLM response
type ExtensionCall struct {
	Name        string // Extension name
	RawData     string // Raw data to process
	UserDisplay string // What user sees
}

// DefaultExtensionService implements ExtensionService
type DefaultExtensionService struct {
	extensions map[string]ExtensionDefinition
}

// NewExtensionService creates a new extension service
func NewExtensionService() ExtensionService {
	return &DefaultExtensionService{
		extensions: make(map[string]ExtensionDefinition),
	}
}

// RegisterExtension implements ExtensionService
func (s *DefaultExtensionService) RegisterExtension(ext ExtensionDefinition) error {
	if ext.Name == "" {
		return fmt.Errorf("extension name cannot be empty")
	}
	if ext.OpenTag == "" {
		return fmt.Errorf("extension open tag cannot be empty")
	}
	// CloseTag can be empty for marker-based extensions
	if ext.Processor == nil {
		return fmt.Errorf("extension processor cannot be nil")
	}

	// Store the definition
	s.extensions[ext.Name] = ext
	return nil
}

// GetAvailableExtensions implements ExtensionService
func (s *DefaultExtensionService) GetAvailableExtensions() []ExtensionDefinition {
	extensions := make([]ExtensionDefinition, 0, len(s.extensions))
	for _, ext := range s.extensions {
		extensions = append(extensions, ext)
	}
	return extensions
}

// ProcessResponse implements ExtensionService
func (s *DefaultExtensionService) ProcessResponse(response string) (string, []ExtensionCall) {
	// First, find all extension boundaries without modifying the response
	boundaries := s.findExtensionBoundaries(response)

	// Validate boundaries for conflicts
	if err := s.validateBoundaries(boundaries); err != nil {
		// If validation fails, fall back to single extension processing
		return s.processSingleExtension(response)
	}

	// Process extensions in order of appearance
	return s.processMultipleExtensions(response, boundaries)
}

// processExtension processes a single extension using tag-based boundaries
func (s *DefaultExtensionService) processExtension(response string, ext ExtensionDefinition) (string, []ExtensionCall) {
	var calls []ExtensionCall
	currentResponse := response

	for {
		openPos := strings.Index(currentResponse, ext.OpenTag)
		if openPos == -1 {
			break
		}

		// Handle marker-based extensions (no closing tag)
		if ext.CloseTag == "" {
			// Extract everything after the marker
			contentStart := openPos + len(ext.OpenTag)
			content := strings.TrimSpace(currentResponse[contentStart:])

			// Find the end of the marker content using heuristic
			// This extracts JSON-like content after the marker
			contentEnd := s.findMarkerContentEnd(content)
			content = strings.TrimSpace(content[:contentEnd])

			if content != "" {
				userDisplay, rawData := ext.Processor(content)

				// Create extension call
				calls = append(calls, ExtensionCall{
					Name:        ext.Name,
					RawData:     rawData,
					UserDisplay: userDisplay,
				})

				// Replace the entire marker and content with user display
				beforeMarker := currentResponse[:openPos]
				afterContent := currentResponse[openPos+len(ext.OpenTag)+len(content):]
				currentResponse = beforeMarker + userDisplay + afterContent
			} else {
				break
			}
		} else {
			// Standard tag-based processing (with closing tag)
			closePos := strings.Index(currentResponse[openPos:], ext.CloseTag)
			if closePos == -1 {
				// No closing tag found - skip this malformed extension
				break
			}

			closePos += openPos + len(ext.CloseTag)

			// Extract content between tags
			contentStart := openPos + len(ext.OpenTag)
			content := currentResponse[contentStart : closePos-len(ext.CloseTag)]

			userDisplay, rawData := ext.Processor(content)

			// Create extension call
			calls = append(calls, ExtensionCall{
				Name:        ext.Name,
				RawData:     rawData,
				UserDisplay: userDisplay,
			})

			// Replace the entire tag block with user display
			beforeTags := currentResponse[:openPos]
			afterTags := currentResponse[closePos:]
			currentResponse = beforeTags + userDisplay + afterTags
		}
	}

	return currentResponse, calls
}

// findMarkerContentEnd finds the end of marker-based content (heuristic for JSON-like content)
func (s *DefaultExtensionService) findMarkerContentEnd(content string) int {
	lines := strings.Split(content, "\n")
	endPos := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			// Add the length of the original line (with whitespace) + newline
			if i < len(lines)-1 {
				endPos += len(line) + 1 // +1 for newline
			} else {
				endPos += len(line) // Last line, no newline
			}
			continue
		}
		// Stop at first non-JSON line
		if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "```") {
			break
		}
		// Add the length of the original line (with whitespace) + newline
		if i < len(lines)-1 {
			endPos += len(line) + 1 // +1 for newline
		} else {
			endPos += len(line) // Last line, no newline
		}
	}

	// Clamp to content length
	if endPos > len(content) {
		endPos = len(content)
	}

	return endPos
}

// ExtensionBoundary represents a found extension boundary in the response
type ExtensionBoundary struct {
	ExtensionName string
	OpenTag       string
	CloseTag      string
	StartPos      int
	EndPos        int
	Content       string
}

// findExtensionBoundaries finds all extension boundaries in the response
func (s *DefaultExtensionService) findExtensionBoundaries(response string) []ExtensionBoundary {
	var boundaries []ExtensionBoundary

	for _, ext := range s.extensions {
		// Find all occurrences of this extension's open tag
		startPos := 0
		for {
			openPos := strings.Index(response[startPos:], ext.OpenTag)
			if openPos == -1 {
				break
			}
			openPos += startPos

			// Handle marker-based extensions (like TOOL_CALLS:)
			if ext.CloseTag == "" {
				boundary := s.findMarkerBoundary(response, ext, openPos)
				if boundary != nil {
					boundaries = append(boundaries, *boundary)
				}
			} else {
				// Handle tag-based extensions
				closePos := strings.Index(response[openPos:], ext.CloseTag)
				if closePos != -1 {
					closePos += openPos + len(ext.CloseTag)
					contentStart := openPos + len(ext.OpenTag)
					content := response[contentStart : closePos-len(ext.CloseTag)]

					boundaries = append(boundaries, ExtensionBoundary{
						ExtensionName: ext.Name,
						OpenTag:       ext.OpenTag,
						CloseTag:      ext.CloseTag,
						StartPos:      openPos,
						EndPos:        closePos,
						Content:       content,
					})
				}
			}

			startPos = openPos + 1
		}
	}

	// Sort boundaries by start position
	for i := 0; i < len(boundaries); i++ {
		for j := i + 1; j < len(boundaries); j++ {
			if boundaries[i].StartPos > boundaries[j].StartPos {
				boundaries[i], boundaries[j] = boundaries[j], boundaries[i]
			}
		}
	}

	return boundaries
}

// findMarkerBoundary finds boundary for marker-based extensions
func (s *DefaultExtensionService) findMarkerBoundary(response string, ext ExtensionDefinition, startPos int) *ExtensionBoundary {
	contentStart := startPos + len(ext.OpenTag)
	content := strings.TrimSpace(response[contentStart:])

	// Find the end of the marker content using generic heuristic
	contentEnd := s.findMarkerContentEnd(content)
	extractedContent := strings.TrimSpace(content[:contentEnd])

	// Adjust endPos to be relative to original response
	endPos := contentStart + len(extractedContent)

	return &ExtensionBoundary{
		ExtensionName: ext.Name,
		OpenTag:       ext.OpenTag,
		CloseTag:      ext.CloseTag,
		StartPos:      startPos,
		EndPos:        endPos,
		Content:       extractedContent,
	}
}

// validateBoundaries checks for overlapping or conflicting boundaries
func (s *DefaultExtensionService) validateBoundaries(boundaries []ExtensionBoundary) error {
	for i := 0; i < len(boundaries); i++ {
		for j := i + 1; j < len(boundaries); j++ {
			// Check for overlap
			if boundaries[i].StartPos < boundaries[j].EndPos && boundaries[j].StartPos < boundaries[i].EndPos {
				return fmt.Errorf("overlapping extension boundaries: %s (%d-%d) and %s (%d-%d)",
					boundaries[i].ExtensionName, boundaries[i].StartPos, boundaries[i].EndPos,
					boundaries[j].ExtensionName, boundaries[j].StartPos, boundaries[j].EndPos)
			}
		}
	}
	return nil
}

// processSingleExtension processes response with single extension (fallback)
func (s *DefaultExtensionService) processSingleExtension(response string) (string, []ExtensionCall) {
	var calls []ExtensionCall
	currentResponse := response

	// Process each registered extension
	for _, ext := range s.extensions {
		processed, extracted := s.processExtension(currentResponse, ext)
		currentResponse = processed
		calls = append(calls, extracted...)
	}

	return currentResponse, calls
}

// processMultipleExtensions processes multiple extensions with proper boundary handling
func (s *DefaultExtensionService) processMultipleExtensions(response string, boundaries []ExtensionBoundary) (string, []ExtensionCall) {
	var calls []ExtensionCall
	var result strings.Builder

	lastPos := 0

	for _, boundary := range boundaries {
		// Add content before this boundary
		if boundary.StartPos > lastPos {
			result.WriteString(response[lastPos:boundary.StartPos])
		}

		// Process this extension
		ext, exists := s.extensions[boundary.ExtensionName]
		if exists {
			userDisplay, rawData := ext.Processor(boundary.Content)
			result.WriteString(userDisplay)

			calls = append(calls, ExtensionCall{
				Name:        boundary.ExtensionName,
				RawData:     rawData,
				UserDisplay: userDisplay,
			})
		}

		lastPos = boundary.EndPos
	}

	// Add remaining content after last boundary
	if lastPos < len(response) {
		result.WriteString(response[lastPos:])
	}

	return result.String(), calls
}

// ExecuteExtensions implements ExtensionService
func (s *DefaultExtensionService) ExecuteExtensions(ctx context.Context, extensions []ExtensionCall) (map[string]ExtensionResult, error) {
	results := make(map[string]ExtensionResult)

	for _, ext := range extensions {
		result, err := s.ExecuteExtension(ctx, ext.Name, ext.RawData)
		if err != nil {
			// Store error result
			results[ext.Name] = ExtensionResult{
				Name:        ext.Name,
				Success:     false,
				Error:       err.Error(),
				UserDisplay: ext.UserDisplay,
				RawData:     ext.RawData,
			}
		} else {
			// Store success result
			result.UserDisplay = ext.UserDisplay
			result.RawData = ext.RawData
			results[ext.Name] = result
		}
	}

	return results, nil
}

// ExecuteExtension implements ExtensionService
func (s *DefaultExtensionService) ExecuteExtension(ctx context.Context, name string, rawData string) (ExtensionResult, error) {
	ext, exists := s.extensions[name]
	if !exists {
		return ExtensionResult{}, fmt.Errorf("extension '%s' not found", name)
	}

	if ext.Executor == nil {
		// No executor - just return the raw data as content
		return ExtensionResult{
			Name:    name,
			Success: true,
			Content: rawData,
		}, nil
	}

	// Execute the extension
	return ext.Executor(ctx, rawData)
}

// FormatForPrompt implements ExtensionService
func (s *DefaultExtensionService) FormatForPrompt() string {
	if len(s.extensions) == 0 {
		return ""
	}

	var formatted strings.Builder
	formatted.WriteString("Available extensions:\n")

	for _, ext := range s.extensions {
		formatted.WriteString(fmt.Sprintf("- %s: %s\n", ext.Name, ext.Description))
	}

	formatted.WriteString("\nTo use extensions, wrap your content in the appropriate tags:\n\n")

	for _, ext := range s.extensions {
		if ext.PromptFormat != "" {
			formatted.WriteString(ext.PromptFormat)
		} else {
			formatted.WriteString(fmt.Sprintf("For %s:\n%s\n{your content here}\n%s\n\n", ext.Name, ext.OpenTag, ext.CloseTag))
		}
	}

	return formatted.String()
}

// ContainsMarker implements ExtensionService - detects any extension marker in text
func (s *DefaultExtensionService) ContainsMarker(text string) (found bool, markerPos int, markerLen int) {
	// Check all registered extensions for their open tags
	earliestPos := -1
	earliestLen := 0

	for _, ext := range s.extensions {
		pos := strings.Index(text, ext.OpenTag)
		if pos != -1 && (earliestPos == -1 || pos < earliestPos) {
			earliestPos = pos
			earliestLen = len(ext.OpenTag)
		}
	}

	if earliestPos != -1 {
		return true, earliestPos, earliestLen
	}

	return false, -1, 0
}
