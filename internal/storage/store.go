package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dyike/CortexGo/models"
	"github.com/dyike/CortexGo/pkg/sqlite"
)

const (
	StatusDone  = "done"
	StatusError = "error"
	StatusInit  = "init"
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

	// Enable foreign keys (SQLite default is OFF unless enabled)
	if _, err := s.db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = s.db.Close()
		return nil, fmt.Errorf("enable foreign_keys: %w", err)
	}

	if err := s.InitTable(); err != nil {
		_ = s.db.Close()
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

// InitTable 初始化会话和消息表。
func (s *Store) InitTable() error {
	sessionDDL := `
	CREATE TABLE IF NOT EXISTS sessions (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
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
	  session_id INTEGER NOT NULL,
	  role TEXT NOT NULL,
	  agent TEXT DEFAULT '',
	  content TEXT,
	  tool_calls TEXT DEFAULT '',
	  tool_call_id TEXT DEFAULT '',
	  tool_name TEXT DEFAULT '',
	  status TEXT,
	  finish_reason TEXT,
	  seq INTEGER,
	  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	  FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE,
	  UNIQUE(session_id, seq)
	);`

	if _, err := s.db.Exec(sessionDDL); err != nil {
		return fmt.Errorf("create sessions table: %w", err)
	}
	if _, err := s.db.Exec(messageDDL); err != nil {
		return fmt.Errorf("create messages table: %w", err)
	}

	// 常用查询索引
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_session_seq ON messages(session_id, seq);`); err != nil {
		return fmt.Errorf("create messages index: %w", err)
	}

	return nil
}

// CreateSession 创建会话（使用自增 id），并回填 rec.Id。
func (s *Store) CreateSession(ctx context.Context, rec *models.SessionRecord) (int64, error) {
	if rec == nil {
		return 0, fmt.Errorf("session record is nil")
	}

	if strings.TrimSpace(rec.Status) == "" {
		rec.Status = StatusInit
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (symbol, trade_date, prompt, status, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, rec.Symbol, rec.TradeDate, rec.Prompt, rec.Status)
	if err != nil {
		return 0, fmt.Errorf("insert session: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}
	rec.Id = id
	return id, nil
}

// UpsertSession：
// - 如果 rec.Id == 0：视为新建，会走 CreateSession，并回填 Id
// - 如果 rec.Id != 0：按主键冲突做 upsert
func (s *Store) UpsertSession(ctx context.Context, rec *models.SessionRecord) error {
	if rec == nil {
		return fmt.Errorf("session record is nil")
	}
	if rec.Id == 0 {
		_, err := s.CreateSession(ctx, rec)
		return err
	}

	if strings.TrimSpace(rec.Status) == "" {
		rec.Status = StatusInit
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, symbol, trade_date, prompt, status, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			symbol = excluded.symbol,
			trade_date = excluded.trade_date,
			prompt = excluded.prompt,
			status = excluded.status,
			updated_at = CURRENT_TIMESTAMP
	`, rec.Id, rec.Symbol, rec.TradeDate, rec.Prompt, rec.Status)
	if err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}
	return nil
}

// UpdateSessionStatus 将会话标记为完成或错误。
func (s *Store) UpdateSessionStatus(ctx context.Context, sessionID int64, status string) error {
	if sessionID <= 0 {
		return fmt.Errorf("invalid session id: %d", sessionID)
	}
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("status is required")
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

// SaveMessage 写入一条消息：
// - 自动生成 seq（同 session 内递增）
// - 并发冲突时（UNIQUE(session_id, seq)）做有限次重试
func (s *Store) SaveMessage(ctx context.Context, msg *models.MessageRecord) error {
	if msg == nil {
		return nil
	}
	if strings.TrimSpace(msg.Role) == "" {
		return fmt.Errorf("message role is required")
	}

	status := msg.Status
	if strings.TrimSpace(status) == "" {
		status = StatusDone
	}

	const maxRetry = 5
	for i := 0; i < maxRetry; i++ {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO messages
				(session_id, role, agent, content, tool_calls, tool_call_id, tool_name, status, finish_reason, seq, updated_at)
			SELECT
				?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE(MAX(seq), 0) + 1, CURRENT_TIMESTAMP
			FROM messages
			WHERE session_id = ?
		`, msg.SessionId, msg.Role, msg.Agent, msg.Content, msg.ToolCalls, msg.ToolCallId, msg.ToolName, status, msg.FinishReason, msg.SessionId)

		if err == nil {
			return nil
		}
		// 并发下可能两个写入都算出同一个 seq：靠 UNIQUE 约束兜底，然后重试
		if isUniqueConstraintErr(err) {
			// 小退避，减少热冲突（不引入额外依赖，直接 sleep）
			time.Sleep(5 * time.Millisecond)
			continue
		}
		return fmt.Errorf("insert message: %w", err)
	}

	return fmt.Errorf("insert message: too many retries due to seq conflicts")
}

// ListSessions 使用 id 游标倒序分页列出会话。
func (s *Store) ListSessions(ctx context.Context, cursor int64, limit int, symbol string, tradeDate string, query string) ([]models.SessionRecord, error) {
	if limit <= 0 {
		limit = 20
	}

	args := make([]any, 0, 4)
	conds := make([]string, 0, 3)
	orConds := make([]string, 0, 2)
	queryStr := `
		SELECT id, symbol, trade_date, prompt, status, created_at, updated_at
		FROM sessions
	`
	if cursor > 0 {
		conds = append(conds, "id < ?")
		args = append(args, cursor)
	}
	if query != "" {
		orConds = append(orConds, "symbol LIKE ?")
		args = append(args, likePattern(query))
		orConds = append(orConds, "trade_date LIKE ?")
		args = append(args, likePattern(query))
	} else {
		if symbol != "" {
			orConds = append(orConds, "symbol LIKE ?")
			args = append(args, likePattern(symbol))
		}
		if tradeDate != "" {
			orConds = append(orConds, "trade_date LIKE ?")
			args = append(args, likePattern(tradeDate))
		}
	}
	if len(orConds) > 0 {
		conds = append(conds, "("+strings.Join(orConds, " OR ")+")")
	}
	if len(conds) > 0 {
		queryStr += "WHERE " + strings.Join(conds, " AND ") + " "
	}
	queryStr += "ORDER BY id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var items []models.SessionRecord
	for rows.Next() {
		var rec models.SessionRecord
		if err := rows.Scan(&rec.Id, &rec.Symbol, &rec.TradeDate, &rec.Prompt, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		items = append(items, rec)
	}
	return items, nil
}

func likePattern(input string) string {
	if strings.ContainsAny(input, "%_") {
		return input
	}
	return "%" + input + "%"
}

// GetSession 读取单个会话详情。
func (s *Store) GetSession(ctx context.Context, sessionID int64) (*models.SessionRecord, error) {
	if sessionID <= 0 {
		return nil, fmt.Errorf("invalid session id: %d", sessionID)
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT id, symbol, trade_date, prompt, status, created_at, updated_at
		FROM sessions
		WHERE id = ?
	`, sessionID)

	var rec models.SessionRecord
	if err := row.Scan(&rec.Id, &rec.Symbol, &rec.TradeDate, &rec.Prompt, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &rec, nil
}

func (s *Store) ListMessages(ctx context.Context, sessionID int64) ([]models.MessageRecord, error) {
	if sessionID <= 0 {
		return nil, fmt.Errorf("invalid session id: %d", sessionID)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			session_id,
			role,
			agent,
			content,
			tool_calls,
			tool_call_id,
			tool_name,
			status,
			finish_reason,
			seq,
			created_at,
			updated_at
		FROM messages
		WHERE session_id = ?
		ORDER BY seq ASC, id ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var items []models.MessageRecord
	for rows.Next() {
		var rec models.MessageRecord
		if err := rows.Scan(
			&rec.Id,
			&rec.SessionId,
			&rec.Role,
			&rec.Agent,
			&rec.Content,
			&rec.ToolCalls,
			&rec.ToolCallId,
			&rec.ToolName,
			&rec.Status,
			&rec.FinishReason,
			&rec.Seq,
			&rec.CreatedAt,
			&rec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		items = append(items, rec)
	}
	return items, nil
}

// DeleteSession 删除会话及其消息（依赖外键 ON DELETE CASCADE）。
func (s *Store) DeleteSession(ctx context.Context, sessionID int64) error {
	if sessionID <= 0 {
		return fmt.Errorf("invalid session id: %d", sessionID)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	res, err := tx.ExecContext(ctx, `
		DELETE FROM messages
		WHERE session_id = ?
	`, sessionID)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete session messages: %w", err)
	}
	_, _ = res.RowsAffected()

	res, err = tx.ExecContext(ctx, `
		DELETE FROM sessions
		WHERE id = ?
	`, sessionID)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete session: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete session: %w", err)
	}
	if rows == 0 {
		_ = tx.Rollback()
		return sql.ErrNoRows
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	// sqlite3 常见报错文本： "UNIQUE constraint failed: messages.session_id, messages.seq"
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed")
}

func isDuplicateColumnErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column name")
}
