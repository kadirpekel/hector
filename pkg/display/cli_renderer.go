package display

import (
	"fmt"
	"os"
	"strings"

	pb "github.com/kadirpekel/hector/pkg/a2a/pb"
)

// Color codes
const (
	colorReset   = "\033[0m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"
	colorItalic  = "\033[3m"
	colorGray    = "\033[90m"
	colorCyan    = "\033[36m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorMagenta = "\033[35m"
	colorRed     = "\033[31m"
	colorBlue    = "\033[34m"
)

// Icons for different block types
const (
	iconThinking   = "💭"
	iconPlanning   = "📋"
	iconReflection = "🔍"
	iconProgress   = "⏳"
	iconDebug      = "🐛"
	iconAnalysis   = "📊"
	iconTool       = "🔧"
	iconSuccess    = "✓"
	iconFailed     = "✗"
)

// CLIRenderer renders content blocks for terminal output
type CLIRenderer struct {
	showThinking bool
	verbose      bool
	useColors    bool
	useIcons     bool
	indentLevel  int
}

// NewCLIRenderer creates a new CLI renderer
func NewCLIRenderer(showThinking, verbose bool) *CLIRenderer {
	return &CLIRenderer{
		showThinking: showThinking,
		verbose:      verbose,
		useColors:    true,
		useIcons:     true,
		indentLevel:  0,
	}
}

// RenderText renders plain text
func (r *CLIRenderer) RenderText(text string) error {
	fmt.Print(text)
	return nil
}

// RenderContentBlock renders a content block with appropriate styling
func (r *CLIRenderer) RenderContentBlock(block *pb.ContentBlock) error {
	switch block.Type {
	case pb.ContentBlockType_CONTENT_BLOCK_TYPE_TEXT:
		return r.renderTextBlock(block.GetTextBlock())

	case pb.ContentBlockType_CONTENT_BLOCK_TYPE_THINKING,
		pb.ContentBlockType_CONTENT_BLOCK_TYPE_REASONING:
		return r.renderThinkingBlock(block.GetThinkingBlock(), "thinking")

	case pb.ContentBlockType_CONTENT_BLOCK_TYPE_REFLECTION:
		return r.renderThinkingBlock(block.GetThinkingBlock(), "reflection")

	case pb.ContentBlockType_CONTENT_BLOCK_TYPE_PLANNING:
		return r.renderThinkingBlock(block.GetThinkingBlock(), "planning")

	case pb.ContentBlockType_CONTENT_BLOCK_TYPE_PROGRESS:
		return r.renderThinkingBlock(block.GetThinkingBlock(), "progress")

	case pb.ContentBlockType_CONTENT_BLOCK_TYPE_DEBUG:
		return r.renderThinkingBlock(block.GetThinkingBlock(), "debug")

	case pb.ContentBlockType_CONTENT_BLOCK_TYPE_ANALYSIS:
		return r.renderThinkingBlock(block.GetThinkingBlock(), "analysis")

	case pb.ContentBlockType_CONTENT_BLOCK_TYPE_TOOL_CALL:
		return r.renderToolCall(block.GetToolCallBlock())

	case pb.ContentBlockType_CONTENT_BLOCK_TYPE_TOOL_RESULT:
		return r.renderToolResult(block.GetToolResultBlock())

	default:
		return nil
	}
}

// renderTextBlock renders a text content block
func (r *CLIRenderer) renderTextBlock(block *pb.TextBlock) error {
	if block == nil {
		return nil
	}
	fmt.Print(block.Text)
	return nil
}

// renderThinkingBlock renders thinking/reasoning/reflection blocks
func (r *CLIRenderer) renderThinkingBlock(block *pb.ThinkingBlock, blockType string) error {
	if block == nil {
		return nil
	}

	// Get metadata
	metadata := block.Metadata
	if metadata == nil {
		metadata = &pb.ThinkingMetadata{}
	}

	// Determine if we should show this block
	if !r.shouldShowThinking(metadata, blockType) {
		return nil
	}

	// Get styling based on block type
	color, icon := r.getStyleForType(blockType)

	// Render with specialized layout based on block type
	switch blockType {
	case "thinking", "reasoning":
		return r.renderThinkingSpecial(block, metadata, color, icon)
	case "planning":
		return r.renderPlanningSpecial(block, metadata, color, icon)
	case "progress":
		return r.renderProgressSpecial(block, metadata, color, icon)
	case "reflection":
		return r.renderReflectionSpecial(block, metadata, color, icon)
	case "analysis":
		return r.renderAnalysisSpecial(block, metadata, color, icon)
	case "debug":
		return r.renderDebugSpecial(block, metadata, color, icon)
	default:
		// Fallback to standard rendering
		if metadata.Title != "" {
			r.printHeader(metadata.Title, color, icon)
		}
		r.printContent(block.Thinking, color, metadata.Collapsible)
		fmt.Println()
		return nil
	}
}

// renderToolCall renders a tool call block
func (r *CLIRenderer) renderToolCall(block *pb.ToolCallBlock) error {
	if block == nil {
		return nil
	}

	icon := iconTool
	if r.useIcons {
		fmt.Printf("\n%s ", icon)
	}

	if r.useColors {
		fmt.Printf("%s%s%s%s", colorCyan, colorBold, block.Name, colorReset)
	} else {
		fmt.Printf("%s", block.Name)
	}

	fmt.Println()
	return nil
}

// renderToolResult renders a tool result block
func (r *CLIRenderer) renderToolResult(block *pb.ToolResultBlock) error {
	if block == nil {
		return nil
	}

	icon := iconSuccess
	color := colorGreen
	if block.IsError {
		icon = iconFailed
		color = colorRed
	}

	if r.useIcons {
		fmt.Printf("%s ", icon)
	}

	if r.useColors {
		fmt.Printf("%s%s%s\n", color, "Tool completed", colorReset)
	} else {
		status := "success"
		if block.IsError {
			status = "failed"
		}
		fmt.Printf("Tool %s\n", status)
	}

	return nil
}

// shouldShowThinking determines if a thinking block should be displayed
func (r *CLIRenderer) shouldShowThinking(metadata *pb.ThinkingMetadata, blockType string) bool {
	// Always show high-priority and planning blocks
	if metadata.Priority >= 2 || blockType == "planning" {
		return true
	}

	// Skip ephemeral blocks in non-verbose mode
	if metadata.Ephemeral && !r.verbose {
		return false
	}

	// Skip if thinking is disabled, except for high-priority
	if !r.showThinking && metadata.Priority < 2 {
		return false
	}

	return true
}

// getStyleForType returns color and icon for a block type
func (r *CLIRenderer) getStyleForType(blockType string) (string, string) {
	switch blockType {
	case "thinking", "reasoning":
		return colorYellow, iconThinking
	case "reflection":
		return colorCyan, iconReflection
	case "planning":
		return colorMagenta, iconPlanning
	case "progress":
		return colorBlue, iconProgress
	case "debug":
		return colorGray, iconDebug
	case "analysis":
		return colorGreen, iconAnalysis
	default:
		return colorReset, "💡"
	}
}

// printHeader prints a styled header
func (r *CLIRenderer) printHeader(title string, color, icon string) {
	fmt.Println()

	if r.useIcons {
		fmt.Printf("%s ", icon)
	}

	if r.useColors {
		fmt.Printf("%s%s%s%s\n", color, colorBold, title, colorReset)
	} else {
		fmt.Printf("%s\n", title)
	}

	// Separator line
	lineLength := len(title) + 2
	if r.useColors {
		fmt.Printf("%s%s%s%s\n", color, colorDim, strings.Repeat("─", lineLength), colorReset)
	} else {
		fmt.Printf("%s\n", strings.Repeat("-", lineLength))
	}
}

// printContent prints styled content
func (r *CLIRenderer) printContent(content, color string, collapsible bool) {
	indent := strings.Repeat("  ", r.indentLevel)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if r.useColors {
			if collapsible {
				fmt.Printf("%s%s%s%s%s\n", indent, color, colorItalic, line, colorReset)
			} else {
				fmt.Printf("%s%s%s%s\n", indent, color, line, colorReset)
			}
		} else {
			fmt.Printf("%s%s\n", indent, line)
		}
	}
}

// Flush ensures all output is written
func (r *CLIRenderer) Flush() error {
	return os.Stdout.Sync()
}

// SetIndentLevel sets the indentation level
func (r *CLIRenderer) SetIndentLevel(level int) {
	r.indentLevel = level
}

// DisableColors disables color output
func (r *CLIRenderer) DisableColors() {
	r.useColors = false
}

// DisableIcons disables icon output
func (r *CLIRenderer) DisableIcons() {
	r.useIcons = false
}

// renderThinkingSpecial renders thinking blocks with grayed-out, dimmed style
func (r *CLIRenderer) renderThinkingSpecial(block *pb.ThinkingBlock, metadata *pb.ThinkingMetadata, color, icon string) error {
	// Thinking blocks should be dimmed/grayed out
	if metadata.Title != "" {
		fmt.Println()
		if r.useIcons {
			fmt.Printf("%s ", icon)
		}
		if r.useColors {
			fmt.Printf("%s%s%s%s\n", colorGray, colorDim, metadata.Title, colorReset)
		} else {
			fmt.Printf("%s\n", metadata.Title)
		}
	}

	// Print content with gray, dimmed styling
	lines := strings.Split(block.Thinking, "\n")
	for _, line := range lines {
		if r.useColors {
			if metadata.Collapsible {
				fmt.Printf("  %s%s%s[thinking] %s%s\n", colorGray, colorDim, colorItalic, line, colorReset)
			} else {
				fmt.Printf("  %s%s%s%s\n", colorGray, colorDim, line, colorReset)
			}
		} else {
			fmt.Printf("  %s\n", line)
		}
	}

	fmt.Println()
	return nil
}

// renderPlanningSpecial renders planning blocks with todo/checklist format
func (r *CLIRenderer) renderPlanningSpecial(block *pb.ThinkingBlock, metadata *pb.ThinkingMetadata, color, icon string) error {
	if metadata.Title != "" {
		r.printHeader(metadata.Title, color, icon)
	}

	// Parse content for todo items (lines starting with -, *, •, or numbers)
	lines := strings.Split(block.Thinking, "\n")
	inList := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is a list item
		isListItem := false
		checkbox := "☐"

		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") ||
			strings.HasPrefix(trimmed, "• ") || strings.HasPrefix(trimmed, "+ ") {
			isListItem = true
			trimmed = strings.TrimPrefix(trimmed, "- ")
			trimmed = strings.TrimPrefix(trimmed, "* ")
			trimmed = strings.TrimPrefix(trimmed, "• ")
			trimmed = strings.TrimPrefix(trimmed, "+ ")
		} else if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' && trimmed[1] == '.' {
			isListItem = true
			// Find the end of the number
			idx := strings.Index(trimmed, ". ")
			if idx > 0 {
				trimmed = trimmed[idx+2:]
			}
		}

		if isListItem {
			inList = true
			if r.useColors {
				fmt.Printf("  %s%s %s%s%s\n", color, checkbox, colorBold, trimmed, colorReset)
			} else {
				fmt.Printf("  %s %s\n", checkbox, trimmed)
			}
		} else if trimmed != "" {
			if inList {
				// End of list, add spacing
				fmt.Println()
				inList = false
			}
			// Regular content
			if r.useColors {
				fmt.Printf("  %s%s%s\n", color, trimmed, colorReset)
			} else {
				fmt.Printf("  %s\n", trimmed)
			}
		} else if !inList {
			// Empty line outside list
			fmt.Println()
		}
	}

	fmt.Println()
	return nil
}

// renderProgressSpecial renders progress blocks as inline status indicators
func (r *CLIRenderer) renderProgressSpecial(block *pb.ThinkingBlock, metadata *pb.ThinkingMetadata, color, icon string) error {
	// Progress should be compact, inline indicators
	if r.useIcons {
		fmt.Printf("%s ", icon)
	}

	if r.useColors {
		fmt.Printf("%s%s%s ", color, block.Thinking, colorReset)
	} else {
		fmt.Printf("%s ", block.Thinking)
	}

	return nil
}

// renderReflectionSpecial renders reflection blocks with emphasized borders
func (r *CLIRenderer) renderReflectionSpecial(block *pb.ThinkingBlock, metadata *pb.ThinkingMetadata, color, icon string) error {
	// Reflection blocks get a boxed treatment
	fmt.Println()

	width := 60
	topBorder := "╭" + strings.Repeat("─", width-2) + "╮"
	bottomBorder := "╰" + strings.Repeat("─", width-2) + "╯"

	if r.useColors {
		fmt.Printf("%s%s%s\n", color, topBorder, colorReset)
	} else {
		fmt.Println(topBorder)
	}

	// Header
	if metadata.Title != "" {
		headerText := fmt.Sprintf("%s %s", icon, metadata.Title)
		padding := width - len(headerText) - 4
		if padding < 0 {
			padding = 0
		}

		if r.useColors {
			fmt.Printf("%s│ %s%s%s%s │%s\n", color, colorBold, headerText, strings.Repeat(" ", padding), color, colorReset)
			fmt.Printf("%s│%s%s│%s\n", color, strings.Repeat(" ", width-2), color, colorReset)
		} else {
			fmt.Printf("│ %s%s │\n", headerText, strings.Repeat(" ", padding))
			fmt.Printf("│%s│\n", strings.Repeat(" ", width-2))
		}
	}

	// Content
	lines := strings.Split(block.Thinking, "\n")
	for _, line := range lines {
		// Wrap long lines
		if len(line) > width-6 {
			// Simple wrapping
			for len(line) > width-6 {
				chunk := line[:width-6]
				line = line[width-6:]

				if r.useColors {
					fmt.Printf("%s│ %s%s │%s\n", color, chunk, strings.Repeat(" ", width-len(chunk)-4), colorReset)
				} else {
					fmt.Printf("│ %s%s │\n", chunk, strings.Repeat(" ", width-len(chunk)-4))
				}
			}
			if line != "" {
				padding := width - len(line) - 4
				if r.useColors {
					fmt.Printf("%s│ %s%s │%s\n", color, line, strings.Repeat(" ", padding), colorReset)
				} else {
					fmt.Printf("│ %s%s │\n", line, strings.Repeat(" ", padding))
				}
			}
		} else {
			padding := width - len(line) - 4
			if padding < 0 {
				padding = 0
			}
			if r.useColors {
				fmt.Printf("%s│ %s%s │%s\n", color, line, strings.Repeat(" ", padding), colorReset)
			} else {
				fmt.Printf("│ %s%s │\n", line, strings.Repeat(" ", padding))
			}
		}
	}

	if r.useColors {
		fmt.Printf("%s%s%s\n", color, bottomBorder, colorReset)
	} else {
		fmt.Println(bottomBorder)
	}

	fmt.Println()
	return nil
}

// renderAnalysisSpecial renders analysis blocks with data-focused layout
func (r *CLIRenderer) renderAnalysisSpecial(block *pb.ThinkingBlock, metadata *pb.ThinkingMetadata, color, icon string) error {
	if metadata.Title != "" {
		r.printHeader(metadata.Title, color, icon)
	}

	// Analysis blocks emphasize structure and data
	lines := strings.Split(block.Thinking, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Highlight metrics and numbers
		if strings.Contains(trimmed, ":") || strings.Contains(trimmed, "=") {
			// This looks like a key-value pair
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				if r.useColors {
					fmt.Printf("  %s%s:%s %s%s%s\n", color, key, colorReset, colorBold, value, colorReset)
				} else {
					fmt.Printf("  %s: %s\n", key, value)
				}
				continue
			}
		}

		// Regular line
		if r.useColors {
			fmt.Printf("  %s%s%s\n", color, trimmed, colorReset)
		} else {
			fmt.Printf("  %s\n", trimmed)
		}
	}

	fmt.Println()
	return nil
}

// renderDebugSpecial renders debug blocks with technical/monospace styling
func (r *CLIRenderer) renderDebugSpecial(block *pb.ThinkingBlock, metadata *pb.ThinkingMetadata, color, icon string) error {
	if metadata.Title != "" {
		fmt.Println()
		if r.useIcons {
			fmt.Printf("%s ", icon)
		}
		if r.useColors {
			fmt.Printf("%s%s[DEBUG]%s %s\n", colorGray, colorDim, colorReset, metadata.Title)
		} else {
			fmt.Printf("[DEBUG] %s\n", metadata.Title)
		}

		// Separator
		fmt.Printf("%s%s%s\n", colorGray, strings.Repeat(".", 40), colorReset)
	}

	// Debug content in monospace-style with line numbers
	lines := strings.Split(block.Thinking, "\n")
	for i, line := range lines {
		if r.useColors {
			fmt.Printf("%s%3d │ %s%s\n", colorGray, i+1, line, colorReset)
		} else {
			fmt.Printf("%3d | %s\n", i+1, line)
		}
	}

	fmt.Println()
	return nil
}
