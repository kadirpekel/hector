// Package dev provides self-development capabilities for Hector
// Git operations for autonomous commits and branch management
package dev

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// GIT MANAGER
// ============================================================================

// GitManager handles git operations for self-development
type GitManager struct {
	ProjectRoot string
	AuthorName  string
	AuthorEmail string
}

// NewGitManager creates a new git manager
func NewGitManager(projectRoot string) *GitManager {
	return &GitManager{
		ProjectRoot: projectRoot,
		AuthorName:  "Hector Dev Agent",
		AuthorEmail: "hector-dev@localhost",
	}
}

// ============================================================================
// BRANCH MANAGEMENT
// ============================================================================

// CreateDevBranch creates a new development branch
func (g *GitManager) CreateDevBranch(category, description string) (string, error) {
	// Generate branch name: dev/{category}-{timestamp}
	timestamp := time.Now().Format("20060102-150405")
	branchName := fmt.Sprintf("dev/%s-%s", category, timestamp)

	// Create and checkout branch
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = g.ProjectRoot

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to create branch: %w, output: %s", err, string(output))
	}

	return branchName, nil
}

// CheckoutBranch checks out an existing branch
func (g *GitManager) CheckoutBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = g.ProjectRoot

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout branch: %w, output: %s", err, string(output))
	}

	return nil
}

// GetCurrentBranch returns the current branch name
func (g *GitManager) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = g.ProjectRoot

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// ListDevBranches lists all dev/* branches
func (g *GitManager) ListDevBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "--list", "dev/*")
	cmd.Dir = g.ProjectRoot

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var branches []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branch != "" {
			branches = append(branches, branch)
		}
	}

	return branches, nil
}

// ============================================================================
// COMMIT OPERATIONS
// ============================================================================

// CommitChange creates a commit with detailed message
func (g *GitManager) CommitChange(commitMsg *CommitMessage) error {
	// Stage all changes
	if err := g.stageChanges(); err != nil {
		return err
	}

	// Check if there are changes to commit
	if !g.hasChanges() {
		return fmt.Errorf("no changes to commit")
	}

	// Format commit message
	message := commitMsg.Format()

	// Create commit
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = g.ProjectRoot
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("GIT_AUTHOR_NAME=%s", g.AuthorName),
		fmt.Sprintf("GIT_AUTHOR_EMAIL=%s", g.AuthorEmail),
		fmt.Sprintf("GIT_COMMITTER_NAME=%s", g.AuthorName),
		fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", g.AuthorEmail),
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit: %w, output: %s", err, string(output))
	}

	return nil
}

func (g *GitManager) stageChanges() error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = g.ProjectRoot

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage changes: %w, output: %s", err, string(output))
	}

	return nil
}

func (g *GitManager) hasChanges() bool {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = g.ProjectRoot

	// Returns non-zero exit code if there are changes
	return cmd.Run() != nil
}

// ============================================================================
// COMMIT MESSAGE STRUCTURE
// ============================================================================

// CommitMessage represents a structured commit message for self-dev
type CommitMessage struct {
	Title        string
	Category     string
	Description  string
	KPIBefore    *HectorKPIs
	KPIAfter     *HectorKPIs
	Comparison   *KPIComparison
	FilesChanged []string
	TestsPassing bool
}

// Format formats the commit message according to the standard
func (c *CommitMessage) Format() string {
	var msg strings.Builder

	// Title with category prefix
	msg.WriteString(fmt.Sprintf("[hector-dev] %s\n\n", c.Title))

	// Category
	msg.WriteString(fmt.Sprintf("Category: %s\n", c.Category))

	// Description
	if c.Description != "" {
		msg.WriteString(fmt.Sprintf("\n%s\n", c.Description))
	}

	// KPI Improvements
	if c.Comparison != nil && len(c.Comparison.Improvements) > 0 {
		msg.WriteString("\nKPI Improvements:\n")
		for metric, improvement := range c.Comparison.Improvements {
			msg.WriteString(fmt.Sprintf("  • %s: +%.1f%%\n", metric, improvement))
		}
	}

	// KPI Regressions (if any)
	if c.Comparison != nil && len(c.Comparison.Regressions) > 0 {
		msg.WriteString("\nKPI Regressions:\n")
		for metric, regression := range c.Comparison.Regressions {
			msg.WriteString(fmt.Sprintf("  • %s: -%.1f%%\n", metric, regression))
		}
	}

	// Overall Score
	if c.Comparison != nil {
		msg.WriteString(fmt.Sprintf("\nOverall Score: %.1f/100 (%s)\n",
			c.Comparison.OverallScore,
			c.Comparison.getScoreLabel()))
	}

	// Key Metrics Summary
	if c.KPIAfter != nil {
		msg.WriteString("\nKey Metrics:\n")
		msg.WriteString(fmt.Sprintf("  • Tests: %d/%d passing (%.1f%%)\n",
			c.KPIAfter.Functional.TestsPassed,
			c.KPIAfter.Functional.TestsTotal,
			c.KPIAfter.Functional.TestPassRate))
		msg.WriteString(fmt.Sprintf("  • Avg Response Time: %dms\n",
			c.KPIAfter.Performance.AvgResponseTime))
		msg.WriteString(fmt.Sprintf("  • Token Efficiency: %.2f\n",
			c.KPIAfter.Efficiency.TokenEfficiency))
		msg.WriteString(fmt.Sprintf("  • Linter Issues: %d\n",
			c.KPIAfter.Quality.LinterIssues))
	}

	// Files Changed
	if len(c.FilesChanged) > 0 {
		msg.WriteString("\nFiles Modified:\n")
		for _, file := range c.FilesChanged {
			msg.WriteString(fmt.Sprintf("  • %s\n", file))
		}
	}

	// Tests Status
	if c.TestsPassing {
		msg.WriteString("\n✅ All tests passing\n")
	} else {
		msg.WriteString("\n⚠️  Some tests failing - review required\n")
	}

	return msg.String()
}

// ============================================================================
// COMMIT HISTORY ANALYSIS
// ============================================================================

// GetRecentCommits gets recent dev commits for learning
func (g *GitManager) GetRecentCommits(count int) ([]*DevCommit, error) {
	cmd := exec.Command("git", "log",
		fmt.Sprintf("-n%d", count),
		"--grep=^\\[hector-dev\\]",
		"--format=%H|%an|%ae|%at|%s|%b",
		"--all")
	cmd.Dir = g.ProjectRoot

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	return g.parseCommits(string(output)), nil
}

// DevCommit represents a development commit
type DevCommit struct {
	Hash         string
	Author       string
	Email        string
	Timestamp    time.Time
	Title        string
	Category     string
	Description  string
	KPIScore     float64
	Improvements map[string]float64
	Regressions  map[string]float64
}

func (g *GitManager) parseCommits(output string) []*DevCommit {
	var commits []*DevCommit

	// Split by commit separator (empty line between commits)
	commitBlocks := strings.Split(output, "\n\n")

	for _, block := range commitBlocks {
		if strings.TrimSpace(block) == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		if len(lines) == 0 {
			continue
		}

		// Parse first line: hash|author|email|timestamp|subject
		fields := strings.Split(lines[0], "|")
		if len(fields) < 5 {
			continue
		}

		commit := &DevCommit{
			Hash:         fields[0],
			Author:       fields[1],
			Email:        fields[2],
			Title:        fields[4],
			Improvements: make(map[string]float64),
			Regressions:  make(map[string]float64),
		}

		// Parse timestamp
		if ts, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
			commit.Timestamp = time.Unix(ts, 0)
		}

		// Parse body for category and KPIs
		body := strings.Join(lines[1:], "\n")
		commit.parseBody(body)

		commits = append(commits, commit)
	}

	return commits
}

func (c *DevCommit) parseBody(body string) {
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse category
		if strings.HasPrefix(line, "Category:") {
			c.Category = strings.TrimSpace(strings.TrimPrefix(line, "Category:"))
		}

		// Parse overall score
		if strings.Contains(line, "Overall Score:") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "Score:" && i+1 < len(fields) {
					scoreStr := strings.TrimSuffix(fields[i+1], "/100")
					if score, err := strconv.ParseFloat(scoreStr, 64); err == nil {
						c.KPIScore = score
					}
				}
			}
		}

		// Parse improvements
		if strings.HasPrefix(line, "•") || strings.HasPrefix(line, "-") {
			c.parseMetricLine(line)
		}
	}
}

func (c *DevCommit) parseMetricLine(line string) {
	// Format: "• metric_name: +X.X%" or "• metric_name: -X.X%"
	line = strings.TrimPrefix(line, "•")
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimSpace(line)

	parts := strings.Split(line, ":")
	if len(parts) != 2 {
		return
	}

	metric := strings.TrimSpace(parts[0])
	valueStr := strings.TrimSpace(parts[1])

	// Check if improvement (+) or regression (-)
	isImprovement := strings.HasPrefix(valueStr, "+")
	valueStr = strings.TrimPrefix(valueStr, "+")
	valueStr = strings.TrimPrefix(valueStr, "-")
	valueStr = strings.TrimSuffix(valueStr, "%")

	if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
		if isImprovement {
			c.Improvements[metric] = value
		} else {
			c.Regressions[metric] = value
		}
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// GetChangedFiles returns list of files changed in working directory
func (g *GitManager) GetChangedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = g.ProjectRoot

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	var files []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			files = append(files, strings.TrimSpace(line))
		}
	}

	return files, nil
}

// IsWorkingTreeClean checks if working tree is clean
func (g *GitManager) IsWorkingTreeClean() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = g.ProjectRoot

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == ""
}
