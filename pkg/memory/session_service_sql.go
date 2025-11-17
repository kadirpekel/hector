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

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type SQLSessionService struct {
	db      *sql.DB
	dialect string
	agentID string
	mu      sync.RWMutex
}

type sessionRow struct {
	ID        string
	AgentID   string
	Metadata  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

const (
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
)

func NewSQLSessionService(db *sql.DB, dialect string, agentID string) (*SQLSessionService, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	switch dialect {
	case "postgres", "mysql", "sqlite":

	default:
		return nil, fmt.Errorf("unsupported dialect: %s (supported: postgres, mysql, sqlite)", dialect)
	}

	s := &SQLSessionService{
		db:      db,
		dialect: dialect,
		agentID: agentID,
	}

	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

func NewSQLSessionServiceFromConfig(cfg *config.SessionSQLConfig, agentID string) (*SQLSessionService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("SQL configuration is required")
	}

	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	driverName := cfg.Driver
	if driverName == "sqlite" {
		driverName = "sqlite3"
	}

	db, err := sql.Open(driverName, cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to %s database '%s' at %s:%d: %w\n"+
			"  💡 Troubleshooting:\n"+
			"     - Ensure the database server is running\n"+
			"     - Check that the host and port are correct\n"+
			"     - Verify network connectivity\n"+
			"     - Confirm database credentials are correct\n"+
			"     - For Docker: ensure the container is running (docker ps)",
			cfg.Driver, cfg.Database, cfg.Host, cfg.Port, err)
	}

	return NewSQLSessionService(db, cfg.Driver, agentID)
}

func (s *SQLSessionService) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sessionsSQL := createSessionsTableSQL
	messagesSQL := createMessagesTableSQL

	if s.dialect == "postgres" {

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

	if _, err := s.db.ExecContext(ctx, sessionsSQL); err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, messagesSQL); err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	return nil
}

func (s *SQLSessionService) AppendMessage(sessionID string, message *pb.Message) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}

	ctx := context.Background()

	if _, err := s.GetOrCreateSessionMetadata(sessionID); err != nil {
		return fmt.Errorf("failed to ensure session exists: %w", err)
	}

	sequenceNum, err := s.getNextSequenceNum(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get sequence number: %w", err)
	}

	messageJSON, err := protojson.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

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

	if err := s.updateSessionTimestamp(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to update session timestamp: %w", err)
	}

	return nil
}

func (s *SQLSessionService) AppendMessages(sessionID string, messages []*pb.Message) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}
	if len(messages) == 0 {
		return nil
	}

	ctx := context.Background()

	if _, err := s.GetOrCreateSessionMetadata(sessionID); err != nil {
		return fmt.Errorf("failed to ensure session exists: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var startSequenceNum int64
	countQuery := `SELECT COALESCE(MAX(sequence_num), 0) FROM session_messages WHERE session_id = ?`
	if s.dialect == "postgres" {
		countQuery = `SELECT COALESCE(MAX(sequence_num), 0) FROM session_messages WHERE session_id = $1`
	}

	if err = tx.QueryRowContext(ctx, countQuery, sessionID).Scan(&startSequenceNum); err != nil {
		return fmt.Errorf("failed to get sequence number: %w", err)
	}

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

	for i, message := range messages {
		if message == nil {
			err = fmt.Errorf("message at index %d is nil", i)
			return err
		}

		messageJSON, marshalErr := protojson.Marshal(message)
		if marshalErr != nil {
			err = fmt.Errorf("failed to marshal message at index %d: %w", i, marshalErr)
			return err
		}

		sequenceNum := startSequenceNum + int64(i) + 1

		_, execErr := tx.ExecContext(ctx, insertQuery,
			sessionID, message.MessageId, message.ContextId, message.TaskId,
			message.Role.String(), string(messageJSON), sequenceNum, now,
		)
		if execErr != nil {
			err = fmt.Errorf("failed to insert message at index %d: %w", i, execErr)
			return err
		}
	}

	updateQuery := `UPDATE sessions SET updated_at = ? WHERE id = ?`
	if s.dialect == "postgres" {
		updateQuery = `UPDATE sessions SET updated_at = $1 WHERE id = $2`
	}

	_, err = tx.ExecContext(ctx, updateQuery, now, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session timestamp: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *SQLSessionService) GetMessages(sessionID string, limit int) ([]*pb.Message, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	ctx := context.Background()

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

	if limit > 0 {

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

func (s *SQLSessionService) GetMessagesWithOptions(sessionID string, opts reasoning.LoadOptions) ([]*pb.Message, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	ctx := context.Background()

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

	if opts.FromMessageID != "" {
		if s.dialect == "postgres" {
			query += ` AND message_id > $2`
			args = append(args, opts.FromMessageID)
		} else {
			query += ` AND message_id > ?`
			args = append(args, opts.FromMessageID)
		}
	}

	if len(opts.Roles) > 0 {
		roleStrings := make([]string, len(opts.Roles))
		for i, role := range opts.Roles {
			roleStrings[i] = role.String()
		}

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

	if opts.Limit > 0 {

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

func (s *SQLSessionService) GetOrCreateSessionMetadata(sessionID string) (*reasoning.SessionMetadata, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	ctx := context.Background()

	s.mu.Lock()
	defer s.mu.Unlock()

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

func (s *SQLSessionService) UpdateSessionMetadata(sessionID string, metadata map[string]interface{}) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}

	ctx := context.Background()

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `UPDATE sessions SET metadata = ?, updated_at = ? WHERE id = ? AND agent_id = ?`
	if s.dialect == "postgres" {
		query = `UPDATE sessions SET metadata = $1, updated_at = $2 WHERE id = $3 AND agent_id = $4`
	}

	_, err = s.db.ExecContext(ctx, query, string(metadataJSON), time.Now(), sessionID, s.agentID)
	if err != nil {
		return fmt.Errorf("failed to update session metadata: %w", err)
	}

	return nil
}

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

	return nil
}

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

func (s *SQLSessionService) Close() error {
	return s.db.Close()
}

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
