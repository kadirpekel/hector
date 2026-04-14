package output

import (
	"context"
	"regexp"
	"strings"

	"github.com/verikod/hector/pkg/guardrails"
)

// ContentFilter blocks or warns about harmful content.
type ContentFilter struct {
	blockedKeywords []string
	blockedPatterns []*regexp.Regexp
	action          guardrails.Action
	severity        guardrails.Severity
}

// NewContentFilter creates a new content filter.
func NewContentFilter() *ContentFilter {
	return &ContentFilter{
		action:   guardrails.ActionBlock,
		severity: guardrails.SeverityHigh,
	}
}

// BlockKeywords adds keywords to block (case-insensitive).
func (f *ContentFilter) BlockKeywords(keywords ...string) *ContentFilter {
	for _, kw := range keywords {
		f.blockedKeywords = append(f.blockedKeywords, strings.ToLower(kw))
	}
	return f
}

// BlockPatterns adds regex patterns to block.
func (f *ContentFilter) BlockPatterns(patterns ...string) *ContentFilter {
	for _, p := range patterns {
		if re, err := regexp.Compile("(?i)" + p); err == nil {
			f.blockedPatterns = append(f.blockedPatterns, re)
		}
	}
	return f
}

// WithAction sets the action to take on match.
func (f *ContentFilter) WithAction(action guardrails.Action) *ContentFilter {
	f.action = action
	return f
}

// WithSeverity sets the severity of matches.
func (f *ContentFilter) WithSeverity(severity guardrails.Severity) *ContentFilter {
	f.severity = severity
	return f
}

// Name returns the guardrail name.
func (f *ContentFilter) Name() string {
	return "content_filter"
}

// Check scans output for blocked content.
func (f *ContentFilter) Check(_ context.Context, output string) (*guardrails.Result, error) {
	lowerOutput := strings.ToLower(output)

	// Check keywords
	for _, keyword := range f.blockedKeywords {
		if strings.Contains(lowerOutput, keyword) {
			return &guardrails.Result{
				Action:        f.action,
				Severity:      f.severity,
				Reason:        "blocked content detected",
				GuardrailName: f.Name(),
				Details: map[string]any{
					"match_type": "keyword",
					"matched":    keyword,
				},
			}, nil
		}
	}

	// Check patterns
	for _, re := range f.blockedPatterns {
		if match := re.FindString(output); match != "" {
			return &guardrails.Result{
				Action:        f.action,
				Severity:      f.severity,
				Reason:        "blocked content pattern detected",
				GuardrailName: f.Name(),
				Details: map[string]any{
					"match_type": "pattern",
					"pattern":    re.String(),
					"matched":    match,
				},
			}, nil
		}
	}

	return guardrails.Allow(f.Name()), nil
}

// Ensure interface compliance.
var _ guardrails.OutputGuardrail = (*ContentFilter)(nil)
