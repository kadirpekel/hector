package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/app"
	"github.com/verikod/hector/pkg/auth"
	"github.com/verikod/hector/pkg/config"
	"github.com/verikod/hector/pkg/session"

	"github.com/verikod/hector/pkg/execution/native"
	"github.com/verikod/hector/pkg/utils"
	"github.com/verikod/hector/pkg/vector"
)

// TaskDeleter defines the interface for deleting app-specific tasks.
type TaskDeleter interface {
	DeleteByApp(ctx context.Context, appName string) error
}

// AdminHandler handles admin API endpoints for app management.
type AdminHandler struct {
	appStore       app.Store
	tokenIssuer    *auth.TokenIssuer
	appManager     *AppManager
	sessionService session.Service
	vectorProvider vector.Provider
	taskDeleter    TaskDeleter
	taskQueue      native.Queue // Optional: for queue admin endpoints
	adminKey       string       // CLI-provided admin key for authentication
	rootDir        string
	reloadFunc     func()       // Optional: triggers config reload (same as SIGHUP)
}

// AdminHandlerConfig configures the admin handler.
type AdminHandlerConfig struct {
	// AppStore is the app storage backend.
	AppStore app.Store

	// TokenIssuer issues JWTs for apps.
	TokenIssuer *auth.TokenIssuer

	// AppManager manages app runtimes.
	AppManager *AppManager

	// SessionService manages app sessions.
	SessionService session.Service

	// VectorProvider manages app vector data.
	VectorProvider vector.Provider

	// TaskDeleter manages task cleanup (optional).
	TaskDeleter TaskDeleter

	// TaskQueue is the task queue for queue admin endpoints (optional).
	TaskQueue native.Queue

	// AdminKey is the secret key for admin authentication (from --admin-key CLI arg).
	AdminKey string

	// RootDir is the root directory for app data (default: .).
	RootDir string

	// ReloadFunc triggers a full config reload (same as SIGHUP). Optional.
	ReloadFunc func()
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(cfg AdminHandlerConfig) (*AdminHandler, error) {
	if cfg.AppStore == nil {
		return nil, fmt.Errorf("app store is required")
	}
	if cfg.TokenIssuer == nil {
		return nil, fmt.Errorf("token issuer is required")
	}
	if cfg.AppManager == nil {
		return nil, fmt.Errorf("app manager is required")
	}
	if cfg.SessionService == nil {
		return nil, fmt.Errorf("session service is required")
	}
	if cfg.VectorProvider == nil {
		return nil, fmt.Errorf("vector provider is required")
	}
	if cfg.AdminKey == "" {
		return nil, fmt.Errorf("admin key is required")
	}

	if cfg.RootDir == "" {
		cfg.RootDir = "."
	}

	return &AdminHandler{
		appStore:       cfg.AppStore,
		tokenIssuer:    cfg.TokenIssuer,
		appManager:     cfg.AppManager,
		sessionService: cfg.SessionService,
		vectorProvider: cfg.VectorProvider,
		taskDeleter:    cfg.TaskDeleter,
		taskQueue:      cfg.TaskQueue,
		adminKey:       cfg.AdminKey,
		rootDir:        cfg.RootDir,
		reloadFunc:     cfg.ReloadFunc,
	}, nil
}

// ServeHTTP routes admin requests.
func (h *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Authenticate admin requests
	if !h.authenticateAdmin(r) {
		h.writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Route based on path
	path := strings.TrimPrefix(r.URL.Path, "/admin")
	path = strings.TrimPrefix(path, "/")

	switch {
	case path == "apps" || path == "apps/":
		h.handleApps(w, r)
	case strings.HasPrefix(path, "apps/"):
		h.handleApp(w, r, strings.TrimPrefix(path, "apps/"))
	case path == "jwks" || path == "jwks.json":
		h.handleJWKS(w, r)
	case path == "sessions" || path == "sessions/":
		h.handleSessions(w, r)
	case strings.HasPrefix(path, "sessions/"):
		h.handleSession(w, r, strings.TrimPrefix(path, "sessions/"))
	case path == "queue" || path == "queue/":
		h.handleQueue(w, r)
	case strings.HasPrefix(path, "queue/"):
		h.handleQueueSubpath(w, r, strings.TrimPrefix(path, "queue/"))
	case path == "reload":
		h.handleReload(w, r)
	default:
		h.writeError(w, "Not found", http.StatusNotFound)
	}
}

// authenticateAdmin checks if the request has valid admin credentials.
func (h *AdminHandler) authenticateAdmin(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	// Support "Bearer <token>" format
	token := strings.TrimPrefix(authHeader, "Bearer ")
	return token == h.adminKey
}

// handleApps handles /admin/apps (list, create).
func (h *AdminHandler) handleApps(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listApps(w, r)
	case http.MethodPost:
		h.createApp(w, r)
	default:
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleApp handles /admin/apps/{id} (get, update, delete).
func (h *AdminHandler) handleApp(w http.ResponseWriter, r *http.Request, path string) {
	// Parse app ID from path
	parts := strings.Split(path, "/")
	appID := parts[0]

	if len(parts) > 1 {
		// Handle sub-resources like /admin/apps/{id}/token
		subResource := parts[1]
		switch subResource {
		case "token":
			if r.Method == http.MethodPost {
				h.regenerateToken(w, r, appID)
				return
			}
		}
		h.writeError(w, "Not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getApp(w, r, appID)
	case http.MethodPut:
		h.updateApp(w, r, appID)
	case http.MethodDelete:
		h.deleteApp(w, r, appID)
	default:
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listApps returns all apps.
func (h *AdminHandler) listApps(w http.ResponseWriter, r *http.Request) {
	apps, err := h.appStore.List(r.Context())
	if err != nil {
		slog.Error("Failed to list apps", "error", err)
		h.writeError(w, "Failed to list apps", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"apps": apps,
	})
}

// createApp creates a new app and returns its JWT.
func (h *AdminHandler) createApp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID         string `json:"id,omitempty"`
		Name       string `json:"name"`
		ConfigJSON string `json:"config_json,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		h.writeError(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Use central config generator if none provided
	configJSON := req.ConfigJSON
	if configJSON == "" {
		result, err := config.GenerateLeanConfig(config.CLIOptions{}, "")
		if err != nil {
			slog.Error("Failed to generate default config", "error", err)
			h.writeError(w, "Failed to generate config", http.StatusInternalServerError)
			return
		}
		jsonBytes, err := json.Marshal(result.AppConfig)
		if err != nil {
			slog.Error("Failed to marshal generated config", "error", err)
			h.writeError(w, "Failed to generate config", http.StatusInternalServerError)
			return
		}
		configJSON = string(jsonBytes)
	}

	// Validate config by parsing it (applies defaults + validates)
	if _, err := config.ParseAppConfigJSON([]byte(configJSON)); err != nil {
		h.writeError(w, fmt.Sprintf("Invalid config: %v", err), http.StatusBadRequest)
		return
	}

	// Create app
	newApp := &app.App{
		ID:         req.ID,
		Name:       req.Name,
		ConfigJSON: configJSON,
	}

	createdApp, err := h.appStore.Create(r.Context(), newApp)
	if err != nil {
		slog.Error("Failed to create app", "error", err)
		h.writeError(w, "Failed to create app", http.StatusInternalServerError)
		return
	}

	// Issue JWT for the new app
	token, err := h.tokenIssuer.IssueToken(createdApp.ID, 365*24*time.Hour) // 1 year
	if err != nil {
		slog.Error("Failed to issue token", "error", err)
		h.writeError(w, "Failed to issue token", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]any{
		"app":          createdApp,
		"access_token": token,
		"token_type":   "bearer",
		"issuer":       h.tokenIssuer.Issuer(),
	})
}

// getApp returns an app by ID.
func (h *AdminHandler) getApp(w http.ResponseWriter, r *http.Request, appID string) {
	foundApp, err := h.appStore.Get(r.Context(), appID)
	if err == app.ErrAppNotFound {
		h.writeError(w, "App not found", http.StatusNotFound)
		return
	}
	if err != nil {
		slog.Error("Failed to get app", "error", err, "app", appID)
		h.writeError(w, "Failed to get app", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, foundApp)
}

// updateApp updates an app's configuration.
func (h *AdminHandler) updateApp(w http.ResponseWriter, r *http.Request, appID string) {
	var req struct {
		Name       string `json:"name"`
		ConfigJSON string `json:"config_json"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing app
	existingApp, err := h.appStore.Get(r.Context(), appID)
	if err == app.ErrAppNotFound {
		h.writeError(w, "App not found", http.StatusNotFound)
		return
	}
	if err != nil {
		slog.Error("Failed to get app", "error", err, "app", appID)
		h.writeError(w, "Failed to get app", http.StatusInternalServerError)
		return
	}

	// Update fields
	if req.Name != "" {
		existingApp.Name = req.Name
	}
	if req.ConfigJSON != "" {
		// Validate config before persisting
		if _, err := config.ParseAppConfigJSON([]byte(req.ConfigJSON)); err != nil {
			h.writeError(w, fmt.Sprintf("Invalid config: %v", err), http.StatusBadRequest)
			return
		}
		existingApp.ConfigJSON = req.ConfigJSON
	}

	if err := h.appStore.Update(r.Context(), existingApp); err != nil {
		slog.Error("Failed to update app", "error", err, "app", appID)
		h.writeError(w, "Failed to update app", http.StatusInternalServerError)
		return
	}

	// Invalidate runtime cache to force reload on next request
	h.appManager.Invalidate(appID)

	h.writeJSON(w, http.StatusOK, existingApp)
}

// deleteApp deletes an app and its resources.
func (h *AdminHandler) deleteApp(w http.ResponseWriter, r *http.Request, appID string) {
	// Protect the "default" app from deletion
	if appID == "default" {
		h.writeError(w, "Cannot delete the default app", http.StatusForbidden)
		return
	}

	// 1. Unload app if running (stop runtime, close resources)
	if err := h.appManager.UnloadApp(r.Context(), appID); err != nil {
		slog.Warn("Failed to unload app before deletion", "error", err, "app", appID)
		// Proceed anyway, but warn
	}

	// 2. Delete app resources (sessions, vectors, files)
	if err := h.deleteAppResources(r.Context(), appID); err != nil {
		slog.Error("Failed to delete app resources", "error", err, "app", appID)
		// We continue to try to delete the app record even if resource cleanup fails partially,
		// but ideally we should probably stop?
		// Decision: Continue to ensure we don't end up with "ghost" apps that can't be deleted.
		// Users can retry delete to clean up leftovers if implemented effectively.
	}

	// 3. Delete app record
	if err := h.appStore.Delete(r.Context(), appID); err == app.ErrAppNotFound {
		h.writeError(w, "App not found", http.StatusNotFound)
		return
	} else if err != nil {
		slog.Error("Failed to delete app from store", "error", err, "app", appID)
		h.writeError(w, "Failed to delete app", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// deleteAppResources cleans up all resources associated with an app.
func (h *AdminHandler) deleteAppResources(ctx context.Context, appID string) error {
	var errs []string

	// 1. Delete sessions and states
	if err := h.sessionService.DeleteByApp(ctx, appID); err != nil {
		errs = append(errs, fmt.Sprintf("failed to delete sessions: %v", err))
	}

	// 2. Delete tasks
	if h.taskDeleter != nil {
		if err := h.taskDeleter.DeleteByApp(ctx, appID); err != nil {
			errs = append(errs, fmt.Sprintf("failed to delete tasks: %v", err))
		}
	} else {
		slog.Warn("Task deletion skipped: taskDeleter not configured", "app", appID)
	}

	// 3. Delete vector data
	if err := h.vectorProvider.DeleteApp(ctx, appID); err != nil {
		errs = append(errs, fmt.Sprintf("failed to delete vector resources: %v", err))
	}

	// 3. Delete app directory (.hector/apps/<app_id>)
	appDir, err := utils.GetAppDir(h.rootDir, appID)
	// If getting dir fails (e.g. permission), we can't remove it.
	if err == nil {
		// Caution: os.RemoveAll is dangerous. Ensure appDir is correct.
		// GetAppDir construction is reliable (.hector/apps/id), so it should be safe.
		// Ensure we don't delete root if something is empty.
		if strings.HasSuffix(appDir, "/.hector/apps/"+appID) || strings.HasSuffix(appDir, "\\.hector\\apps\\"+appID) {
			if err := os.RemoveAll(appDir); err != nil {
				errs = append(errs, fmt.Sprintf("failed to delete app directory: %v", err))
			} else {
				slog.Info("Deleted app directory", "path", appDir)
			}
		} else {
			// This should theoretically never happen with GetAppDir, but extra safety check.
			slog.Warn("Skipping directory deletion for safety: path pattern mismatch", "path", appDir)
		}
	} else {
		errs = append(errs, fmt.Sprintf("failed to resolve app dir: %v", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// regenerateToken creates a new JWT for an app.
func (h *AdminHandler) regenerateToken(w http.ResponseWriter, r *http.Request, appID string) {
	// Verify app exists
	_, err := h.appStore.Get(r.Context(), appID)
	if err == app.ErrAppNotFound {
		h.writeError(w, "App not found", http.StatusNotFound)
		return
	}
	if err != nil {
		slog.Error("Failed to get app", "error", err, "app", appID)
		h.writeError(w, "Failed to get app", http.StatusInternalServerError)
		return
	}

	// Issue new token
	token, err := h.tokenIssuer.IssueToken(appID, 365*24*time.Hour)
	if err != nil {
		slog.Error("Failed to issue token", "error", err, "app", appID)
		h.writeError(w, "Failed to issue token", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "bearer",
		"issuer":       h.tokenIssuer.Issuer(),
	})
}

// handleJWKS serves the public JWKS for token verification.
func (h *AdminHandler) handleJWKS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jwks := h.tokenIssuer.PublicJWKS()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jwks); err != nil {
		slog.Error("Failed to encode JWKS", "error", err)
		h.writeError(w, "Failed to encode JWKS", http.StatusInternalServerError)
	}
}

// writeJSON writes a JSON response.
func (h *AdminHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// writeError writes a JSON error response.
func (h *AdminHandler) writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// handleSessions handles /admin/sessions (list).
func (h *AdminHandler) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.listSessions(w, r)
}

// handleSession handles /admin/sessions/{id} (get, delete).
func (h *AdminHandler) handleSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	if sessionID == "" || len(sessionID) > 128 {
		h.writeError(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getSession(w, r, sessionID)
	case http.MethodDelete:
		h.deleteSession(w, r, sessionID)
	default:
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listSessions returns sessions for the current app with pagination.
// GET /admin/sessions?app_id=X&user_id=Y&page_size=N&page_token=T
func (h *AdminHandler) listSessions(w http.ResponseWriter, r *http.Request) {
	if h.sessionService == nil {
		h.writeError(w, "Session service not configured", http.StatusServiceUnavailable)
		return
	}

	// Parse query params
	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		appID = "default"
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default"
	}
	pageSize := 50 // Default page size
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if val, err := fmt.Sscanf(ps, "%d", &pageSize); err == nil && val > 0 {
			if pageSize > 100 {
				pageSize = 100 // Cap at 100
			}
		}
	}
	pageToken := r.URL.Query().Get("page_token")

	resp, err := h.sessionService.List(r.Context(), &session.ListRequest{
		AppName:   appID,
		UserID:    userID,
		PageSize:  pageSize,
		PageToken: pageToken,
	})
	if err != nil {
		slog.Error("Failed to list sessions", "error", err, "app", appID)
		h.writeError(w, "Failed to list sessions", http.StatusInternalServerError)
		return
	}

	// Build response with session metadata
	type sessionMeta struct {
		ID         string `json:"id"`
		AppName    string `json:"app_name"`
		UserID     string `json:"user_id"`
		LastUpdate string `json:"last_update"`
		EventCount int    `json:"event_count"`
	}

	sessions := make([]sessionMeta, 0, len(resp.Sessions))
	for _, s := range resp.Sessions {
		sessions = append(sessions, sessionMeta{
			ID:         s.ID(),
			AppName:    s.AppName(),
			UserID:     s.UserID(),
			LastUpdate: s.LastUpdateTime().Format(time.RFC3339),
			EventCount: s.Events().Len(),
		})
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"sessions":        sessions,
		"next_page_token": resp.NextPageToken,
	})
}

// getSession returns a session by ID with its events (merged with xray).
// GET /admin/sessions/{id}?app_id=X&limit=N&after_event_id=ID
func (h *AdminHandler) getSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	if h.sessionService == nil {
		h.writeError(w, "Session service not configured", http.StatusServiceUnavailable)
		return
	}

	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		appID = "default"
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default"
	}

	// Parse limit - default to 0 (all events) if not specified, or use provided limit
	limit := 0 // 0 means return all events
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	afterEventID := r.URL.Query().Get("after_event_id")

	resp, err := h.sessionService.Get(r.Context(), &session.GetRequest{
		AppName:         appID,
		UserID:          userID,
		SessionID:       sessionID,
		NumRecentEvents: limit, // 0 = all events
		AfterEventID:    afterEventID,
	})
	if err != nil {
		if err == session.ErrSessionNotFound {
			h.writeError(w, fmt.Sprintf("Session %q not found", sessionID), http.StatusNotFound)
		} else {
			h.writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Serialize events
	events := make([]*agent.Event, 0)
	for evt := range resp.Session.Events().All() {
		events = append(events, evt)
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":          resp.Session.ID(),
		"app_name":    resp.Session.AppName(),
		"user_id":     resp.Session.UserID(),
		"last_update": resp.Session.LastUpdateTime().Format(time.RFC3339),
		"event_count": len(events),
		"events":      events,
	})
}

// deleteSession removes a session.
// DELETE /admin/sessions/{id}?app_id=X&user_id=Y
func (h *AdminHandler) deleteSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	if h.sessionService == nil {
		h.writeError(w, "Session service not configured", http.StatusServiceUnavailable)
		return
	}

	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		appID = "default"
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default"
	}

	err := h.sessionService.Delete(r.Context(), &session.DeleteRequest{
		AppName:   appID,
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		if err == session.ErrSessionNotFound {
			h.writeError(w, fmt.Sprintf("Session %q not found", sessionID), http.StatusNotFound)
		} else {
			slog.Error("Failed to delete session", "error", err, "session_id", sessionID)
			h.writeError(w, "Failed to delete session", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleQueue handles /admin/queue (stats).
func (h *AdminHandler) handleQueue(w http.ResponseWriter, r *http.Request) {
	if h.taskQueue == nil {
		h.writeError(w, "Queue not configured", http.StatusServiceUnavailable)
		return
	}

	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get queue stats
	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		appID = "default"
	}

	stats, err := h.taskQueue.Stats(r.Context(), appID)
	if err != nil {
		slog.Error("Failed to get queue stats", "error", err, "app", appID)
		h.writeError(w, "Failed to get queue stats", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, stats)
}

// handleQueueSubpath handles /admin/queue/* routes.
func (h *AdminHandler) handleQueueSubpath(w http.ResponseWriter, r *http.Request, path string) {
	if h.taskQueue == nil {
		h.writeError(w, "Queue not configured", http.StatusServiceUnavailable)
		return
	}

	switch {
	case path == "stats":
		h.handleQueueStats(w, r)
	case path == "dlq" || path == "dlq/":
		h.handleDLQ(w, r)
	case strings.HasPrefix(path, "dlq/"):
		h.handleDLQItem(w, r, strings.TrimPrefix(path, "dlq/"))
	default:
		h.writeError(w, "Not found", http.StatusNotFound)
	}
}

// handleQueueStats returns queue statistics.
// GET /admin/queue/stats?app_id=X
func (h *AdminHandler) handleQueueStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		appID = "default"
	}

	stats, err := h.taskQueue.Stats(r.Context(), appID)
	if err != nil {
		slog.Error("Failed to get queue stats", "error", err, "app", appID)
		h.writeError(w, "Failed to get queue stats", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, stats)
}

// handleDLQ lists dead-letter queue items.
// GET /admin/queue/dlq?app_id=X&limit=N
func (h *AdminHandler) handleDLQ(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		appID = "default"
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > 100 {
				limit = 100
			}
		}
	}

	items, err := h.taskQueue.ListDLQ(r.Context(), appID, limit)
	if err != nil {
		slog.Error("Failed to list DLQ", "error", err, "app", appID)
		h.writeError(w, "Failed to list dead-letter queue", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"count": len(items),
	})
}

// handleDLQItem handles operations on individual DLQ items.
// POST /admin/queue/dlq/{id}/requeue - Requeue item
func (h *AdminHandler) handleDLQItem(w http.ResponseWriter, r *http.Request, path string) {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		h.writeError(w, "Invalid path", http.StatusBadRequest)
		return
	}

	itemID := parts[0]
	action := parts[1]

	switch action {
	case "requeue":
		if r.Method != http.MethodPost {
			h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.requeueDLQItem(w, r, itemID)
	default:
		h.writeError(w, "Not found", http.StatusNotFound)
	}
}

// requeueDLQItem moves an item from DLQ back to pending.
// POST /admin/queue/dlq/{id}/requeue
func (h *AdminHandler) requeueDLQItem(w http.ResponseWriter, r *http.Request, itemID string) {
	err := h.taskQueue.Requeue(r.Context(), itemID)
	if err != nil {
		slog.Error("Failed to requeue DLQ item", "error", err, "item_id", itemID)
		h.writeError(w, "Failed to requeue item", http.StatusInternalServerError)
		return
	}

	slog.Info("Requeued DLQ item", "item_id", itemID)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"requeued": itemID,
		"status":   "pending",
	})
}

// handleReload triggers a full config reload (equivalent to SIGHUP).
// POST /admin/reload
// Optional JSON body: { "env": { "KEY": "VALUE", ... } }
// Environment variables in the body are set before triggering the reload.
func (h *AdminHandler) handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.reloadFunc == nil {
		h.writeError(w, "Reload not configured", http.StatusServiceUnavailable)
		return
	}

	// Parse optional env overrides
	if r.Body != nil && r.ContentLength > 0 {
		var req struct {
			Env map[string]string `json:"env,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Apply env overrides before reload
		for key, value := range req.Env {
			if err := os.Setenv(key, value); err != nil {
				slog.Warn("Failed to set environment variable", "key", key, "error", err)
			}
		}

		if len(req.Env) > 0 {
			slog.Info("Applied environment overrides before reload", "count", len(req.Env))
		}
	}

	// Trigger reload (same path as SIGHUP)
	slog.Info("Admin API triggered config reload")
	h.reloadFunc()

	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":  "reloaded",
		"message": "Configuration reload triggered successfully",
	})
}
