package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	StatusStreaming = "streaming"
	StatusDone      = "done"
	StatusError     = "error"
)

type Store struct {
	db *sql.DB
}

type SessionRecord struct {
	ID        string
	Symbol    string
	TradeDate string
	Prompt    string
	Status    string
}

type MessageRecord struct {
	ID           string
	SessionID    string
	Role         string
	Agent        string
	Content      string
	Status       string
	FinishReason string
	Seq          int
}

type SessionWithMeta struct {
	SessionRecord
	RowID     int64
	CreatedAt string
	UpdatedAt string
}

type MessageWithMeta struct {
	MessageRecord
	CreatedAt string
	UpdatedAt string
}

func Open(dbPath string) (*Store, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, fmt.Errorf("db path is required")
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=3000;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("set pragma %s: %w", p, err)
		}
	}

	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func initSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    symbol TEXT,
    trade_date TEXT,
    prompt TEXT,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    agent TEXT,
    content TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    finish_reason TEXT,
    seq INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(session_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_messages_session_created ON messages(session_id, created_at);
`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

func (s *Store) CreateSession(ctx context.Context, session SessionRecord) error {
	if session.Status == "" {
		session.Status = StatusStreaming
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO sessions (id, symbol, trade_date, prompt, status)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    symbol=excluded.symbol,
    trade_date=excluded.trade_date,
    prompt=excluded.prompt,
    status=excluded.status,
    updated_at=CURRENT_TIMESTAMP
`, session.ID, session.Symbol, session.TradeDate, session.Prompt, session.Status)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *Store) InsertMessage(ctx context.Context, msg MessageRecord) error {
	if msg.Status == "" {
		msg.Status = StatusStreaming
	}
	if msg.Seq <= 0 {
		return fmt.Errorf("message seq must be positive")
	}
	if strings.TrimSpace(msg.Role) == "" {
		return fmt.Errorf("message role is required")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO messages (id, session_id, role, agent, content, status, finish_reason, seq)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO NOTHING
`, msg.ID, msg.SessionID, msg.Role, msg.Agent, msg.Content, msg.Status, msg.FinishReason, msg.Seq)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

func (s *Store) AppendMessageContent(ctx context.Context, msgID string, delta string) error {
	if strings.TrimSpace(msgID) == "" || delta == "" {
		return nil
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE messages
SET content = content || ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, delta, msgID)
	if err != nil {
		return fmt.Errorf("append message content: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("append message content: message %s not found", msgID)
	}
	return nil
}

func (s *Store) ReplaceMessageContent(ctx context.Context, msgID string, content string) error {
	if strings.TrimSpace(msgID) == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE messages
SET content = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, content, msgID)
	if err != nil {
		return fmt.Errorf("replace message content: %w", err)
	}
	return nil
}

func (s *Store) MarkMessageStatus(ctx context.Context, msgID, status, finishReason string) error {
	if strings.TrimSpace(msgID) == "" {
		return nil
	}
	if status == "" {
		status = StatusDone
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE messages
SET status = ?, 
    finish_reason = CASE WHEN ? <> '' THEN ? ELSE finish_reason END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, status, finishReason, finishReason, msgID)
	if err != nil {
		return fmt.Errorf("update message status: %w", err)
	}
	return nil
}

func (s *Store) UpdateSessionStatus(ctx context.Context, sessionID, status string) error {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(status) == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE sessions
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, status, sessionID)
	if err != nil {
		return fmt.Errorf("update session status: %w", err)
	}
	return nil
}

func (s *Store) FinalizeOpenMessages(ctx context.Context, sessionID, status, finishReason string) error {
	if status == "" {
		status = StatusDone
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE messages
SET status = ?,
    finish_reason = CASE WHEN ? <> '' THEN ? ELSE finish_reason END,
    updated_at = CURRENT_TIMESTAMP
WHERE session_id = ? AND status = ?
`, status, finishReason, finishReason, sessionID, StatusStreaming)
	if err != nil {
		return fmt.Errorf("finalize messages: %w", err)
	}
	return nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

// ListSessions 按 rowid 倒序分页列出会话
func (s *Store) ListSessions(ctx context.Context, cursor int64, limit int) ([]SessionWithMeta, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT rowid, id, symbol, trade_date, prompt, status, created_at, updated_at
FROM sessions
WHERE (? = 0 OR rowid < ?)
ORDER BY rowid DESC
LIMIT ?
`, cursor, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionWithMeta
	for rows.Next() {
		var rec SessionWithMeta
		if err := rows.Scan(&rec.RowID, &rec.ID, &rec.Symbol, &rec.TradeDate, &rec.Prompt, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sessions rows: %w", err)
	}
	return sessions, nil
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (*SessionWithMeta, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("session id is required")
	}
	row := s.db.QueryRowContext(ctx, `
SELECT rowid, id, symbol, trade_date, prompt, status, created_at, updated_at
FROM sessions
WHERE id = ?
LIMIT 1
`, sessionID)

	var rec SessionWithMeta
	if err := row.Scan(&rec.RowID, &rec.ID, &rec.Symbol, &rec.TradeDate, &rec.Prompt, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &rec, nil
}

func (s *Store) ListMessages(ctx context.Context, sessionID string) ([]MessageWithMeta, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("session id is required")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, session_id, role, agent, content, status, finish_reason, seq, created_at, updated_at
FROM messages
WHERE session_id = ?
ORDER BY seq ASC
`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var msgs []MessageWithMeta
	for rows.Next() {
		var rec MessageWithMeta
		if err := rows.Scan(&rec.ID, &rec.SessionID, &rec.Role, &rec.Agent, &rec.Content, &rec.Status, &rec.FinishReason, &rec.Seq, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list messages rows: %w", err)
	}
	return msgs, nil
}
