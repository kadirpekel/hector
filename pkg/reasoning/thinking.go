package reasoning

import "fmt"

const (
	colorReset = "\033[0m"
	colorGray  = "\033[90m"
	colorDim   = "\033[2m"
)

func ThinkingBlock(text string) string {
	return fmt.Sprintf("%s%s[Thinking: %s]%s\n", colorGray, colorDim, text, colorReset)
}

func ThinkingGoalExtraction(mainGoal string, subGoals []string) string {
	block := ThinkingBlock(fmt.Sprintf("Goal identified: %s", mainGoal))
	if len(subGoals) > 0 {
		for i, sg := range subGoals {
			block += ThinkingBlock(fmt.Sprintf("  Sub-goal %d: %s", i+1, sg))
		}
	}
	return block
}

func ThinkingReflection(accomplished, missing string, confidence float64) string {
	block := ThinkingBlock(fmt.Sprintf("Reflection: %s", accomplished))
	if missing != "" && missing != "nothing" {
		block += ThinkingBlock(fmt.Sprintf("  Still need: %s", missing))
	}
	block += ThinkingBlock(fmt.Sprintf("  Confidence: %.0f%%", confidence*100))
	return block
}

func ThinkingQualityCheck(confidence float64, shouldContinue bool, reason string) string {
	block := ThinkingBlock(fmt.Sprintf("Quality check: %.0f%% confident", confidence*100))
	if shouldContinue {
		block += ThinkingBlock(fmt.Sprintf("  Decision: Continue - %s", reason))
	} else {
		block += ThinkingBlock(fmt.Sprintf("  Decision: Ready to answer - %s", reason))
	}
	return block
}

func ThinkingProgress(iteration, maxIter int, action string) string {
	return ThinkingBlock(fmt.Sprintf("Iteration %d/%d: %s", iteration, maxIter, action))
}
