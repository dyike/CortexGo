package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/dyike/CortexGo/models"
	"github.com/dyike/CortexGo/pkg/sqlite"
)

const (
	StatusStreaming = "streaming"
	StatusDone      = "done"
	StatusError     = "error"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sqlite.Open(dbPath)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.InitTable(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// InitTable 初始化会话和消息表，并在需要时补充缺失列。
func (p *Store) InitTable() error {
	sessionDDL := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		symbol TEXT,
		trade_date TEXT,
		prompt TEXT,
		status TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	messageDDL := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		message_role TEXT NOT NULL,
		agent TEXT DEFAULT '',
		content TEXT,
		tool_calls_json TEXT,
		tool_call_id TEXT,
		status TEXT,
		finish_reason TEXT,
		seq INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(session_id) REFERENCES sessions(id)
	);`

	if _, err := p.db.Exec(sessionDDL); err != nil {
		return fmt.Errorf("create sessions table: %w", err)
	}

	if _, err := p.db.Exec(messageDDL); err != nil {
		return fmt.Errorf("create messages table: %w", err)
	}

	if err := p.ensureSessionsSchema(); err != nil {
		return err
	}
	if err := p.ensureMessagesSchema(); err != nil {
		return err
	}

	if _, err := p.db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_rowid ON sessions(rowid);`); err != nil {
		return fmt.Errorf("create sessions index: %w", err)
	}
	if _, err := p.db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_session_seq ON messages(session_id, seq);`); err != nil {
		return fmt.Errorf("create messages index: %w", err)
	}
	return nil
}

// UpsertSession 创建或更新会话元信息。
func (p *Store) UpsertSession(ctx context.Context, rec models.SessionRecord) error {
	if strings.TrimSpace(rec.ID) == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(rec.Status) == "" {
		rec.Status = StatusStreaming
	}
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO sessions (id, symbol, trade_date, prompt, status)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			symbol = excluded.symbol,
			trade_date = excluded.trade_date,
			prompt = excluded.prompt,
			status = excluded.status,
			updated_at = CURRENT_TIMESTAMP
	`, rec.ID, rec.Symbol, rec.TradeDate, rec.Prompt, rec.Status)
	return err
}

// UpdateSessionStatus 将会话标记为完成或错误。
func (p *Store) UpdateSessionStatus(ctx context.Context, sessionID, status string) error {
	_, err := p.db.ExecContext(ctx, `
		UPDATE sessions
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, sessionID)
	return err
}

// SaveMessage 将聚合后的消息写入 SQLite。
func (p *Store) SaveMessage(ctx context.Context, sessionID, event string, msg *models.ChatResp) error {
	if msg == nil {
		return nil
	}
	// 仅持久化完整的 *_final 消息，忽略流式片段
	if !strings.HasSuffix(event, "_final") {
		return nil
	}
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session id is required for saving messages")
	}

	seq, err := p.nextSeq(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get next seq: %w", err)
	}

	var toolCallsJSON string
	if len(msg.ToolCalls) > 0 {
		data, err := json.Marshal(msg.ToolCalls)
		if err != nil {
			log.Printf("Error marshalling tool calls for persistence: %v", err)
		} else {
			toolCallsJSON = string(data)
		}
	}

	status := StatusDone
	finishReason := ""

	_, err = p.db.ExecContext(ctx, `
		INSERT INTO messages
		(session_id, message_role, agent, content, tool_calls_json, tool_call_id, status, finish_reason, seq, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, sessionID, msg.Role, "", msg.Content, toolCallsJSON, msg.ToolCallId, status, finishReason, seq)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

// ListSessions 使用 rowid 游标倒序分页列出会话。
func (p *Store) ListSessions(ctx context.Context, cursor int64, limit int) ([]models.SessionWithMeta, error) {
	args := make([]any, 0, 2)
	query := `
	SELECT rowid, id, symbol, trade_date, prompt, status, created_at, updated_at
	FROM sessions
	`
	if cursor > 0 {
		query += "WHERE rowid < ? "
		args = append(args, cursor)
	}
	query += "ORDER BY rowid DESC LIMIT ?"
	args = append(args, limit)

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var items []models.SessionWithMeta
	for rows.Next() {
		var rec models.SessionWithMeta
		if err := rows.Scan(&rec.RowID, &rec.ID, &rec.Symbol, &rec.TradeDate, &rec.Prompt, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		items = append(items, rec)
	}
	return items, nil
}

// GetSession 读取单个会话详情。
func (p *Store) GetSession(ctx context.Context, sessionID string) (*models.SessionWithMeta, error) {
	row := p.db.QueryRowContext(ctx, `
		SELECT rowid, id, symbol, trade_date, prompt, status, created_at, updated_at
		FROM sessions
		WHERE id = ?
	`, sessionID)

	var rec models.SessionWithMeta
	if err := row.Scan(&rec.RowID, &rec.ID, &rec.Symbol, &rec.TradeDate, &rec.Prompt, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &rec, nil
}

// ListMessages 按 seq 升序返回指定会话的消息。
func (p *Store) ListMessages(ctx context.Context, sessionID string) ([]models.MessageWithMeta, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, session_id, message_role, agent, content, status, finish_reason, seq, created_at, updated_at
		FROM messages
		WHERE session_id = ?
		ORDER BY seq ASC, id ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var items []models.MessageWithMeta
	for rows.Next() {
		var rec models.MessageWithMeta
		if err := rows.Scan(&rec.ID, &rec.SessionID, &rec.Role, &rec.Agent, &rec.Content, &rec.Status, &rec.FinishReason, &rec.Seq, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		items = append(items, rec)
	}
	return items, nil
}

func (p *Store) nextSeq(ctx context.Context, sessionID string) (int, error) {
	row := p.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(seq), 0) + 1 FROM messages WHERE session_id = ?
	`, sessionID)
	var seq sql.NullInt64
	if err := row.Scan(&seq); err != nil {
		return 0, err
	}
	if seq.Valid {
		return int(seq.Int64), nil
	}
	return 1, nil
}

func (p *Store) tableColumns(table string) (map[string]struct{}, error) {
	rows, err := p.db.Query(fmt.Sprintf("PRAGMA table_info(%s);", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := make(map[string]struct{})
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols[name] = struct{}{}
	}
	return cols, nil
}

func (p *Store) addColumnIfMissing(table, column, def string) error {
	cols, err := p.tableColumns(table)
	if err != nil {
		return err
	}
	if _, exists := cols[column]; exists {
		return nil
	}
	alter := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table, column, def)
	if _, err := p.db.Exec(alter); err != nil {
		return fmt.Errorf("add column %s to %s: %w", column, table, err)
	}
	return nil
}

func (p *Store) ensureMessagesSchema() error {
	extraCols := map[string]string{
		"agent":         "TEXT DEFAULT ''",
		"status":        "TEXT",
		"finish_reason": "TEXT",
		"seq":           "INTEGER",
		"created_at":    "DATETIME DEFAULT CURRENT_TIMESTAMP",
		"updated_at":    "DATETIME DEFAULT CURRENT_TIMESTAMP",
	}
	for col, def := range extraCols {
		if err := p.addColumnIfMissing("messages", col, def); err != nil {
			return err
		}
	}
	return nil
}

func (p *Store) ensureSessionsSchema() error {
	extraCols := map[string]string{
		"created_at": "DATETIME DEFAULT CURRENT_TIMESTAMP",
		"updated_at": "DATETIME DEFAULT CURRENT_TIMESTAMP",
	}
	for col, def := range extraCols {
		if err := p.addColumnIfMissing("sessions", col, def); err != nil {
			return err
		}
	}
	return nil
}
