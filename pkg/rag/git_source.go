package rag

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/verikod/hector/pkg/utils"
)

// GitSourceConfig configures git-backed indexing.
type GitSourceConfig struct {
	URLs        []string
	Ref         string
	Depth       int
	SparsePaths []string
	Include     []string
	Exclude     []string
	MaxFileSize int64
	CacheDir    string
	AuthTokenEnv string
	RefreshMode string
	RefreshInterval time.Duration
}

// GitSource materializes git repositories to local cache and indexes files via DirectorySource.
type GitSource struct {
	cfg      GitSourceConfig
	cacheDir string
	dirSrc   *DirectorySource
	rootDir  string
	repoMeta map[string]gitRepoMeta
}

type gitRepoMeta struct {
	URL    string
	Commit string
}

var (
	gitSourceAllowedReposMu sync.RWMutex
	gitSourceAllowedRepos   []string
)

// SetGitAllowedRepos configures an optional allowlist for git source URLs.
// If empty, all URLs are allowed.
func SetGitAllowedRepos(repos []string) {
	gitSourceAllowedReposMu.Lock()
	defer gitSourceAllowedReposMu.Unlock()

	gitSourceAllowedRepos = gitSourceAllowedRepos[:0]
	for _, r := range repos {
		trimmed := strings.TrimSpace(r)
		if trimmed != "" {
			gitSourceAllowedRepos = append(gitSourceAllowedRepos, trimmed)
		}
	}
}

func isGitURLAllowed(url string) bool {
	gitSourceAllowedReposMu.RLock()
	defer gitSourceAllowedReposMu.RUnlock()

	if len(gitSourceAllowedRepos) == 0 {
		return true
	}

	for _, allowed := range gitSourceAllowedRepos {
		if strings.HasPrefix(url, allowed) || url == allowed {
			return true
		}
	}

	return false
	
}

// NewGitSource creates a lazy git-backed data source.
func NewGitSource(cfg GitSourceConfig) (*GitSource, error) {
	if len(cfg.URLs) == 0 {
		return nil, fmt.Errorf("at least one git url is required")
	}
	if cfg.Depth <= 0 {
		cfg.Depth = 1
	}
	if cfg.RefreshMode == "" {
		cfg.RefreshMode = "manual"
	}
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = time.Hour
	}

	for _, u := range cfg.URLs {
		if !isGitURLAllowed(u) {
			return nil, fmt.Errorf("git repository not allowed by policy: %s", u)
		}
	}

	cacheDir := cfg.CacheDir
	if cacheDir == "" {
		hectorDir, err := utils.EnsureHectorDir(".")
		if err != nil {
			return nil, err
		}
		cacheDir = filepath.Join(hectorDir, "git-sources")
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create git cache dir: %w", err)
	}

	materializedRoot, repoMeta, err := materializeGitRepos(cacheDir, cfg)
	if err != nil {
		return nil, err
	}

	if len(cfg.Include) == 0 {
		cfg.Include = DefaultDirectorySourceConfig(materializedRoot).Include
	}
	if len(cfg.Exclude) == 0 {
		cfg.Exclude = DefaultDirectorySourceConfig(materializedRoot).Exclude
	}
	if cfg.MaxFileSize <= 0 {
		cfg.MaxFileSize = 10 * 1024 * 1024
	}

	ds, err := NewDirectorySourceFromConfig(DirectorySourceConfig{
		Path:        materializedRoot,
		Include:     cfg.Include,
		Exclude:     cfg.Exclude,
		MaxFileSize: cfg.MaxFileSize,
	})
	if err != nil {
		return nil, err
	}

	dirSrc, ok := ds.(*DirectorySource)
	if !ok {
		return nil, fmt.Errorf("expected directory source")
	}

	return &GitSource{
		cfg:      cfg,
		cacheDir: cacheDir,
		dirSrc:   dirSrc,
		rootDir:  materializedRoot,
		repoMeta: repoMeta,
	}, nil
}

func (g *GitSource) Type() string {
	return "git"
}

func (g *GitSource) DiscoverDocuments(ctx context.Context) (<-chan Document, <-chan error) {
	baseDocs, baseErrs := g.dirSrc.DiscoverDocuments(ctx)
	outDocs := make(chan Document, 100)
	outErrs := make(chan error, 10)

	go func() {
		defer close(outDocs)
		defer close(outErrs)

		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-baseErrs:
				if !ok {
					baseErrs = nil
					continue
				}
				outErrs <- err
			case doc, ok := <-baseDocs:
				if !ok {
					baseDocs = nil
					continue
				}
				annotated := g.annotateDocument(doc)
				outDocs <- annotated
			}

			if baseDocs == nil && baseErrs == nil {
				return
			}
		}
	}()

	return outDocs, outErrs
}

func (g *GitSource) ReadDocument(ctx context.Context, id string) (*Document, error) {
	doc, err := g.dirSrc.ReadDocument(ctx, id)
	if err != nil {
		return nil, err
	}
	annotated := g.annotateDocument(*doc)
	return &annotated, nil
}

func (g *GitSource) SupportsIncrementalIndexing() bool {
	return g.dirSrc.SupportsIncrementalIndexing()
}

func (g *GitSource) GetLastModified(ctx context.Context, id string) (time.Time, error) {
	return g.dirSrc.GetLastModified(ctx, id)
}

func (g *GitSource) Close() error {
	return g.dirSrc.Close()
}

func materializeGitRepos(cacheDir string, cfg GitSourceConfig) (string, map[string]gitRepoMeta, error) {
	h := sha1.New()
	h.Write([]byte(strings.Join(cfg.URLs, "|")))
	h.Write([]byte("|" + cfg.Ref))
	h.Write([]byte("|" + strings.Join(cfg.SparsePaths, "|")))
	h.Write([]byte(fmt.Sprintf("|%d", cfg.Depth)))
	h.Write([]byte("|" + cfg.RefreshMode))
	h.Write([]byte("|" + cfg.RefreshInterval.String()))
	key := hex.EncodeToString(h.Sum(nil))[:16]

	root := filepath.Join(cacheDir, key)
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", nil, err
	}

	repoMeta := make(map[string]gitRepoMeta)

	for idx, url := range cfg.URLs {
		repoKey := fmt.Sprintf("repo_%d", idx+1)
		repoDir := filepath.Join(root, repoKey)
		if err := ensureRepoMaterialized(repoDir, url, cfg); err != nil {
			return "", nil, err
		}
		commit, err := currentCommit(repoDir)
		if err != nil {
			return "", nil, err
		}
		repoMeta[repoKey] = gitRepoMeta{URL: url, Commit: commit}
	}

	return root, repoMeta, nil
}

func ensureRepoMaterialized(repoDir, url string, cfg GitSourceConfig) error {
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
		if !shouldRefresh(repoDir, cfg) {
			return nil
		}
		if err := refreshRepo(repoDir, cfg); err != nil {
			return err
		}
		return writeRefreshStamp(repoDir)
	}

	if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
		return err
	}

	args := []string{"clone", "--depth", fmt.Sprintf("%d", cfg.Depth)}
	if cfg.Ref != "" {
		args = append(args, "--branch", cfg.Ref)
	}
	if len(cfg.SparsePaths) > 0 {
		args = append(args, "--no-checkout", "--filter=blob:none", "--sparse")
	}
	args = append(args, url, repoDir)

	if err := runGit(filepath.Dir(repoDir), cfg.AuthTokenEnv, args...); err != nil {
		return fmt.Errorf("git clone failed for %s: %w", url, err)
	}

	if len(cfg.SparsePaths) > 0 {
		sparseArgs := append([]string{"sparse-checkout", "set"}, cfg.SparsePaths...)
		if err := runGit(repoDir, cfg.AuthTokenEnv, sparseArgs...); err != nil {
			return fmt.Errorf("git sparse-checkout failed: %w", err)
		}
		if err := runGit(repoDir, cfg.AuthTokenEnv, "checkout"); err != nil {
			return fmt.Errorf("git checkout failed: %w", err)
		}
	}

	return writeRefreshStamp(repoDir)
}

func shouldRefresh(repoDir string, cfg GitSourceConfig) bool {
	switch cfg.RefreshMode {
	case "startup":
		return true
	case "interval":
		stamp, err := readRefreshStamp(repoDir)
		if err != nil {
			return true
		}
		return time.Since(stamp) >= cfg.RefreshInterval
	case "manual", "":
		return false
	default:
		return false
	}
}

func refreshRepo(repoDir string, cfg GitSourceConfig) error {
	if cfg.Ref != "" {
		if err := runGit(repoDir, cfg.AuthTokenEnv, "fetch", "--depth", fmt.Sprintf("%d", cfg.Depth), "origin", cfg.Ref); err != nil {
			return fmt.Errorf("git fetch failed: %w", err)
		}
		if err := runGit(repoDir, cfg.AuthTokenEnv, "checkout", "--detach", "FETCH_HEAD"); err != nil {
			return fmt.Errorf("git checkout failed: %w", err)
		}
		return nil
	}

	if err := runGit(repoDir, cfg.AuthTokenEnv, "fetch", "--depth", fmt.Sprintf("%d", cfg.Depth), "origin"); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}
	if err := runGit(repoDir, cfg.AuthTokenEnv, "checkout", "--detach", "FETCH_HEAD"); err != nil {
		return fmt.Errorf("git checkout failed: %w", err)
	}

	return nil
}

func currentCommit(repoDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to resolve git commit: %s", strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func refreshStampPath(repoDir string) string {
	return filepath.Join(repoDir, ".hector_refresh_stamp")
}

func writeRefreshStamp(repoDir string) error {
	now := strconv.FormatInt(time.Now().Unix(), 10)
	return os.WriteFile(refreshStampPath(repoDir), []byte(now), 0644)
}

func readRefreshStamp(repoDir string) (time.Time, error) {
	data, err := os.ReadFile(refreshStampPath(repoDir))
	if err != nil {
		return time.Time{}, err
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(n, 0), nil
}

func (g *GitSource) annotateDocument(doc Document) Document {
	if doc.Metadata == nil {
		doc.Metadata = map[string]any{}
	}

	repoKey := strings.Split(strings.TrimPrefix(doc.SourcePath, string(os.PathSeparator)), string(os.PathSeparator))[0]
	if repoKey == "" {
		repoKey = strings.Split(strings.TrimPrefix(doc.SourcePath, "/"), "/")[0]
	}
	if meta, ok := g.repoMeta[repoKey]; ok {
		doc.Metadata["git_repo_url"] = meta.URL
		doc.Metadata["git_commit"] = meta.Commit
		doc.Metadata["git_ref"] = g.cfg.Ref
	}

	doc.Metadata["git_refresh_mode"] = g.cfg.RefreshMode
	if g.cfg.RefreshInterval > 0 {
		doc.Metadata["git_refresh_interval"] = g.cfg.RefreshInterval.String()
	}

	return doc
}

func runGit(repoDir, authTokenEnv string, args ...string) error {
	cmd := exec.Command("git", args...)
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	cmd.Env = os.Environ()
	if authTokenEnv != "" {
		token := os.Getenv(authTokenEnv)
		if token != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_TOKEN=%s", token))
		}
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("git command failed", "args", strings.Join(args, " "), "output", string(output))
		return fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	return nil
}

var _ DataSource = (*GitSource)(nil)
