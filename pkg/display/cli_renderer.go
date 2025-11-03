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

	// Render header if title provided
	if metadata.Title != "" {
		r.printHeader(metadata.Title, color, icon)
	}

	// Render content with appropriate styling
	content := block.Thinking

	r.printContent(content, color, metadata.Collapsible)

	fmt.Println() // Add spacing after block
	return nil
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
