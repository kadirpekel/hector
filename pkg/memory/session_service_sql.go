package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"google.golang.org/protobuf/encoding/protojson"

	// Database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// SQLSessionService implements reasoning.SessionService with SQL backend
// Supports PostgreSQL, MySQL, and SQLite via database/sql
type SQLSessionService struct {
	db      *sql.DB
	dialect string // "postgres", "mysql", or "sqlite"
	agentID string // Agent ID that owns this service (for multi-agent support)
	mu      sync.RWMutex
}

// sessionRow represents the database schema for sessions
type sessionRow struct {
	ID        string
	AgentID   string
	Metadata  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// messageRow represents the database schema for messages
type messageRow struct {
	ID          int64
	SessionID   string
	MessageID   string
	ContextID   string
	TaskID      string
	Role        string
	MessageJSON string
	SequenceNum int64
	CreatedAt   time.Time
}

const (
	// SQL schema for sessions
	// IMPORTANT: For multi-agent isolation, PRIMARY KEY is (id, agent_id)
	// This allows different agents to use the same session ID
	createSessionsTableSQL = `
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(255) NOT NULL,
    agent_id VARCHAR(255) NOT NULL,
    metadata TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (id, agent_id)
);

CREATE INDEX IF NOT EXISTS idx_sessions_agent_id ON sessions(agent_id);
CREATE INDEX IF NOT EXISTS idx_sessions_updated_at ON sessions(updated_at);
`

	// SQL schema for messages
	createMessagesTableSQL = `
CREATE TABLE IF NOT EXISTS session_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    context_id VARCHAR(255),
    task_id VARCHAR(255),
    role VARCHAR(50) NOT NULL,
    message_json TEXT NOT NULL,
    sequence_num INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_session_id ON session_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_message_id ON session_messages(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_sequence ON session_messages(session_id, sequence_num);
`
	// Note: For MySQL, change AUTOINCREMENT to AUTO_INCREMENT
	// For PostgreSQL, use SERIAL instead of INTEGER AUTOINCREMENT
)

// NewSQLSessionService creates a new SQL-backed session service
func NewSQLSessionService(db *sql.DB, dialect string, agentID string) (*SQLSessionService, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	// Validate dialect
	switch dialect {
	case "postgres", "mysql", "sqlite":
		// Valid
	default:
		return nil, fmt.Errorf("unsupported dialect: %s (supported: postgres, mysql, sqlite)", dialect)
	}

	s := &SQLSessionService{
		db:      db,
		dialect: dialect,
		agentID: agentID,
	}

	// Initialize schema (shared across all agents using this database)
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

// NewSQLSessionServiceFromConfig creates a new SQL session service from configuration
func NewSQLSessionServiceFromConfig(cfg *config.SessionSQLConfig, agentID string) (*SQLSessionService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("SQL configuration is required")
	}

	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	// Set defaults and validate
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Map driver name to actual SQL driver name
	driverName := cfg.Driver
	if driverName == "sqlite" {
		driverName = "sqlite3"
	}

	// Open database connection
	db, err := sql.Open(driverName, cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create service
	return NewSQLSessionService(db, cfg.Driver, agentID)
}

// initSchema creates the sessions and messages tables if they don't exist
func (s *SQLSessionService) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Adjust schema for different dialects
	sessionsSQL := createSessionsTableSQL
	messagesSQL := createMessagesTableSQL

	if s.dialect == "postgres" {
		// PostgreSQL uses SERIAL for auto-increment
		messagesSQL = `
CREATE TABLE IF NOT EXISTS session_messages (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    context_id VARCHAR(255),
    task_id VARCHAR(255),
    role VARCHAR(50) NOT NULL,
    message_json TEXT NOT NULL,
    sequence_num BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_session_id ON session_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_message_id ON session_messages(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_sequence ON session_messages(session_id, sequence_num);
`
	} else if s.dialect == "mysql" {
		// MySQL uses AUTO_INCREMENT
		messagesSQL = `
CREATE TABLE IF NOT EXISTS session_messages (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    session_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    context_id VARCHAR(255),
    task_id VARCHAR(255),
    role VARCHAR(50) NOT NULL,
    message_json TEXT NOT NULL,
    sequence_num BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_session_id ON session_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_message_id ON session_messages(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_sequence ON session_messages(session_id, sequence_num);
`
	}

	// Create sessions table
	if _, err := s.db.ExecContext(ctx, sessionsSQL); err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	// Create messages table
	if _, err := s.db.ExecContext(ctx, messagesSQL); err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	return nil
}

// AppendMessage appends a message to a session
func (s *SQLSessionService) AppendMessage(sessionID string, message *pb.Message) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}

	ctx := context.Background()

	// Ensure session exists
	if _, err := s.GetOrCreateSessionMetadata(sessionID); err != nil {
		return fmt.Errorf("failed to ensure session exists: %w", err)
	}

	// Get next sequence number
	sequenceNum, err := s.getNextSequenceNum(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get sequence number: %w", err)
	}

	// Serialize message using protojson
	messageJSON, err := protojson.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Prepare insert query based on dialect
	query := `
INSERT INTO session_messages (session_id, message_id, context_id, task_id, role, message_json, sequence_num, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`
	if s.dialect == "postgres" {
		query = `
INSERT INTO session_messages (session_id, message_id, context_id, task_id, role, message_json, sequence_num, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`
	}

	now := time.Now()
	_, err = s.db.ExecContext(ctx, query,
		sessionID, message.MessageId, message.ContextId, message.TaskId,
		message.Role.String(), string(messageJSON), sequenceNum, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	// Update session timestamp
	if err := s.updateSessionTimestamp(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to update session timestamp: %w", err)
	}

	return nil
}

// AppendMessages appends multiple messages atomically using a transaction
// If any message fails, the entire batch is rolled back
// This is the PREFERRED method for batch operations
func (s *SQLSessionService) AppendMessages(sessionID string, messages []*pb.Message) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}
	if len(messages) == 0 {
		return nil // Nothing to append
	}

	ctx := context.Background()

	// Ensure session exists
	if _, err := s.GetOrCreateSessionMetadata(sessionID); err != nil {
		return fmt.Errorf("failed to ensure session exists: %w", err)
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Get starting sequence number (do this inside transaction for consistency)
	var startSequenceNum int64
	countQuery := `SELECT COALESCE(MAX(sequence_num), 0) FROM session_messages WHERE session_id = ?`
	if s.dialect == "postgres" {
		countQuery = `SELECT COALESCE(MAX(sequence_num), 0) FROM session_messages WHERE session_id = $1`
	}

	if err = tx.QueryRowContext(ctx, countQuery, sessionID).Scan(&startSequenceNum); err != nil {
		return fmt.Errorf("failed to get sequence number: %w", err)
	}

	// Prepare insert query based on dialect
	insertQuery := `
INSERT INTO session_messages (session_id, message_id, context_id, task_id, role, message_json, sequence_num, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`
	if s.dialect == "postgres" {
		insertQuery = `
INSERT INTO session_messages (session_id, message_id, context_id, task_id, role, message_json, sequence_num, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`
	}

	now := time.Now()

	// Insert all messages within transaction
	for i, message := range messages {
		if message == nil {
			err = fmt.Errorf("message at index %d is nil", i)
			return err
		}

		// Serialize message
		messageJSON, marshalErr := protojson.Marshal(message)
		if marshalErr != nil {
			err = fmt.Errorf("failed to marshal message at index %d: %w", i, marshalErr)
			return err
		}

		sequenceNum := startSequenceNum + int64(i) + 1

		// Execute insert within transaction
		_, execErr := tx.ExecContext(ctx, insertQuery,
			sessionID, message.MessageId, message.ContextId, message.TaskId,
			message.Role.String(), string(messageJSON), sequenceNum, now,
		)
		if execErr != nil {
			err = fmt.Errorf("failed to insert message at index %d: %w", i, execErr)
			return err
		}
	}

	// Update session timestamp within transaction
	updateQuery := `UPDATE sessions SET updated_at = ? WHERE id = ?`
	if s.dialect == "postgres" {
		updateQuery = `UPDATE sessions SET updated_at = $1 WHERE id = $2`
	}

	_, err = tx.ExecContext(ctx, updateQuery, now, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session timestamp: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetMessages returns the most recent messages from a session
func (s *SQLSessionService) GetMessages(sessionID string, limit int) ([]*pb.Message, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	ctx := context.Background()

	// Build query with JOIN to filter by agent_id (multi-agent isolation)
	query := `
SELECT sm.message_json
FROM session_messages sm
JOIN sessions s ON sm.session_id = s.id
WHERE sm.session_id = ? AND s.agent_id = ?
ORDER BY sm.sequence_num ASC
`
	args := []interface{}{sessionID, s.agentID}

	if s.dialect == "postgres" {
		query = `
SELECT sm.message_json
FROM session_messages sm
JOIN sessions s ON sm.session_id = s.id
WHERE sm.session_id = $1 AND s.agent_id = $2
ORDER BY sm.sequence_num ASC
`
	}

	// Add limit if specified
	if limit > 0 {
		// For limit, we want the LAST N messages, so we need a subquery
		if s.dialect == "postgres" {
			query = `
SELECT message_json FROM (
    SELECT sm.message_json, sm.sequence_num
    FROM session_messages sm
    JOIN sessions s ON sm.session_id = s.id
    WHERE sm.session_id = $1 AND s.agent_id = $2
    ORDER BY sm.sequence_num DESC
    LIMIT $3
) sub ORDER BY sequence_num ASC
`
		} else {
			query = `
SELECT message_json FROM (
    SELECT sm.message_json, sm.sequence_num
    FROM session_messages sm
    JOIN sessions s ON sm.session_id = s.id
    WHERE sm.session_id = ? AND s.agent_id = ?
    ORDER BY sm.sequence_num DESC
    LIMIT ?
) sub ORDER BY sequence_num ASC
`
		}
		args = append(args, limit)
	}

	var rows *sql.Rows
	var err error

	rows, err = s.db.QueryContext(ctx, query, args...)

	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*pb.Message
	for rows.Next() {
		var messageJSON string
		if err := rows.Scan(&messageJSON); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		message := &pb.Message{}
		if err := protojson.Unmarshal([]byte(messageJSON), message); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message: %w", err)
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// GetMessagesWithOptions retrieves messages with advanced filtering options
// This allows strategies to load messages efficiently (e.g., from checkpoint)
func (s *SQLSessionService) GetMessagesWithOptions(sessionID string, opts reasoning.LoadOptions) ([]*pb.Message, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	ctx := context.Background()

	// Build query based on dialect - JOIN with sessions to filter by agent_id for isolation
	query := `
		SELECT sm.message_id, sm.message_json 
		FROM session_messages sm 
		JOIN sessions s ON sm.session_id = s.id 
		WHERE sm.session_id = ? AND s.agent_id = ?
	`
	args := []interface{}{sessionID, s.agentID}

	if s.dialect == "postgres" {
		query = `
			SELECT sm.message_id, sm.message_json 
			FROM session_messages sm 
			JOIN sessions s ON sm.session_id = s.id 
			WHERE sm.session_id = $1 AND s.agent_id = $2
		`
	}

	// Add FromMessageID filter (for checkpoint loading)
	if opts.FromMessageID != "" {
		if s.dialect == "postgres" {
			query += ` AND message_id > $2`
			args = append(args, opts.FromMessageID)
		} else {
			query += ` AND message_id > ?`
			args = append(args, opts.FromMessageID)
		}
	}

	// Add role filter
	if len(opts.Roles) > 0 {
		roleStrings := make([]string, len(opts.Roles))
		for i, role := range opts.Roles {
			roleStrings[i] = role.String()
		}
		// Build IN clause
		placeholders := ""
		for i := range roleStrings {
			if i > 0 {
				placeholders += ", "
			}
			if s.dialect == "postgres" {
				placeholders += fmt.Sprintf("$%d", len(args)+1+i)
			} else {
				placeholders += "?"
			}
		}
		query += ` AND role IN (` + placeholders + `)`
		for _, rs := range roleStrings {
			args = append(args, rs)
		}
	}

	query += ` ORDER BY sequence_num ASC`

	// Add limit (get last N messages)
	if opts.Limit > 0 {
		// Wrap in subquery to get last N
		if s.dialect == "postgres" {
			query = `SELECT message_id, message_json FROM (` + query + ` ORDER BY sequence_num DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1) + `) sub ORDER BY sequence_num ASC`
		} else {
			query = `SELECT message_id, message_json FROM (` + query + ` ORDER BY sequence_num DESC LIMIT ?) sub ORDER BY sequence_num ASC`
		}
		args = append(args, opts.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*pb.Message
	for rows.Next() {
		var messageID, messageJSON string
		if err := rows.Scan(&messageID, &messageJSON); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		message := &pb.Message{}
		if err := protojson.Unmarshal([]byte(messageJSON), message); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message: %w", err)
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// GetMessageCount returns the number of messages in a session
func (s *SQLSessionService) GetMessageCount(sessionID string) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("sessionID cannot be empty")
	}

	ctx := context.Background()

	query := `SELECT COUNT(*) FROM session_messages WHERE session_id = ?`
	if s.dialect == "postgres" {
		query = `SELECT COUNT(*) FROM session_messages WHERE session_id = $1`
	}

	var count int
	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}

// GetOrCreateSessionMetadata returns or creates session metadata
func (s *SQLSessionService) GetOrCreateSessionMetadata(sessionID string) (*reasoning.SessionMetadata, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	ctx := context.Background()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Try to get existing session FOR THIS AGENT (multi-agent isolation)
	query := `
SELECT id, agent_id, metadata, created_at, updated_at
FROM sessions
WHERE id = ? AND agent_id = ?
`
	if s.dialect == "postgres" {
		query = `
SELECT id, agent_id, metadata, created_at, updated_at
FROM sessions
WHERE id = $1 AND agent_id = $2
`
	}

	var row sessionRow
	err := s.db.QueryRowContext(ctx, query, sessionID, s.agentID).Scan(
		&row.ID, &row.AgentID, &row.Metadata, &row.CreatedAt, &row.UpdatedAt,
	)

	if err == nil {
		// Session exists, parse and return
		metadata := &reasoning.SessionMetadata{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			Metadata:  make(map[string]interface{}),
		}

		if row.Metadata != "" && row.Metadata != "{}" {
			if err := json.Unmarshal([]byte(row.Metadata), &metadata.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		return metadata, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	// Session doesn't exist, create it
	now := time.Now()
	metadata := &reasoning.SessionMetadata{
		ID:        sessionID,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]interface{}),
	}

	metadataJSON, err := json.Marshal(metadata.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	insertQuery := `
INSERT INTO sessions (id, agent_id, metadata, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
`
	if s.dialect == "postgres" {
		insertQuery = `
INSERT INTO sessions (id, agent_id, metadata, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
`
	}

	_, err = s.db.ExecContext(ctx, insertQuery,
		sessionID, s.agentID, string(metadataJSON), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert session: %w", err)
	}

	return metadata, nil
}

// DeleteSession deletes a session and all its messages
func (s *SQLSessionService) DeleteSession(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}

	ctx := context.Background()

	query := `DELETE FROM sessions WHERE id = ?`
	if s.dialect == "postgres" {
		query = `DELETE FROM sessions WHERE id = $1`
	}

	_, err := s.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Messages will be cascade deleted due to foreign key constraint

	return nil
}

// SessionCount returns the number of active sessions
func (s *SQLSessionService) SessionCount() int {
	ctx := context.Background()

	query := `SELECT COUNT(*) FROM sessions`

	var count int
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0
	}

	return count
}

// Close closes the database connection
func (s *SQLSessionService) Close() error {
	return s.db.Close()
}

// Helper methods

func (s *SQLSessionService) getNextSequenceNum(ctx context.Context, sessionID string) (int64, error) {
	query := `SELECT COALESCE(MAX(sequence_num), 0) + 1 FROM session_messages WHERE session_id = ?`
	if s.dialect == "postgres" {
		query = `SELECT COALESCE(MAX(sequence_num), 0) + 1 FROM session_messages WHERE session_id = $1`
	}

	var sequenceNum int64
	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(&sequenceNum)
	if err != nil {
		return 0, fmt.Errorf("failed to get next sequence number: %w", err)
	}

	return sequenceNum, nil
}

func (s *SQLSessionService) updateSessionTimestamp(ctx context.Context, sessionID string) error {
	query := `UPDATE sessions SET updated_at = ? WHERE id = ?`
	if s.dialect == "postgres" {
		query = `UPDATE sessions SET updated_at = $1 WHERE id = $2`
	}

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, now, sessionID)
	return err
}
