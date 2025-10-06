package reasoning

import "fmt"

// ============================================================================
// THINKING MODE - SHOW INTERNAL REASONING (CLAUDE-STYLE)
// ============================================================================
//
// This provides grayed-out "thinking blocks" similar to how Claude's
// reasoning appears in Cursor - showing meta-cognition without cluttering
// the main output.
//
// Example:
//   [Thinking: Extracting goals from query...]
//   [Thinking: Goal identified - get weather and analyze mood impact]
//   [Thinking: Confidence 70% - need to research mood impact]
//   [Thinking: Analysis complete - confidence 95%]
//
// These blocks are:
// - Grayed out (ANSI dim/gray color)
// - Optional (controlled by show_thinking config)
// - Show the internal reasoning process
// ============================================================================

const (
	// ANSI color codes for thinking blocks
	colorReset = "\033[0m"
	colorGray  = "\033[90m" // Bright black (gray)
	colorDim   = "\033[2m"  // Dim text
)

// ThinkingBlock formats text as a grayed-out thinking block
func ThinkingBlock(text string) string {
	return fmt.Sprintf("%s%s[Thinking: %s]%s\n", colorGray, colorDim, text, colorReset)
}

// ThinkingGoalExtraction shows goal extraction thinking
func ThinkingGoalExtraction(mainGoal string, subGoals []string) string {
	block := ThinkingBlock(fmt.Sprintf("Goal identified: %s", mainGoal))
	if len(subGoals) > 0 {
		for i, sg := range subGoals {
			block += ThinkingBlock(fmt.Sprintf("  Sub-goal %d: %s", i+1, sg))
		}
	}
	return block
}

// ThinkingReflection shows meta-cognitive reflection
func ThinkingReflection(accomplished, missing string, confidence float64) string {
	block := ThinkingBlock(fmt.Sprintf("Reflection: %s", accomplished))
	if missing != "" && missing != "nothing" {
		block += ThinkingBlock(fmt.Sprintf("  Still need: %s", missing))
	}
	block += ThinkingBlock(fmt.Sprintf("  Confidence: %.0f%%", confidence*100))
	return block
}

// ThinkingQualityCheck shows quality evaluation
func ThinkingQualityCheck(confidence float64, shouldContinue bool, reason string) string {
	block := ThinkingBlock(fmt.Sprintf("Quality check: %.0f%% confident", confidence*100))
	if shouldContinue {
		block += ThinkingBlock(fmt.Sprintf("  Decision: Continue - %s", reason))
	} else {
		block += ThinkingBlock(fmt.Sprintf("  Decision: Ready to answer - %s", reason))
	}
	return block
}

// ThinkingProgress shows progress update
func ThinkingProgress(iteration, maxIter int, action string) string {
	return ThinkingBlock(fmt.Sprintf("Iteration %d/%d: %s", iteration, maxIter, action))
}
