package storage

import (
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

const dbPath = ""

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sqlite.Open(dbPath)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	s.InitTable()
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// InitTable 初始化消息表
// message_role: assistant, tool
// tool_calls_json: 存储 ToolCallAggregate 数组的 JSON 字符串
func (p *Store) InitTable() {
	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		message_role TEXT NOT NULL,
		content TEXT,
		tool_calls_json TEXT,
		tool_call_id TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := p.db.Exec(query)
	if err != nil {
		log.Fatalf("Error creating messages table: %v", err)
	}
	fmt.Println("SQLite table 'messages' initialized successfully.")
}

// SaveMessage 将聚合后的消息写入 SQLite
func (p *Store) SaveMessage(event string, msg *models.ChatResp) {
	// 仅持久化完整的 *_final 消息，忽略流式片段
	if !strings.HasSuffix(event, "_final") {
		return
	}

	// 1. 序列化 ToolCalls 字段为 JSON 字符串
	var toolCallsJSON string
	if len(msg.ToolCalls) > 0 {
		data, err := json.Marshal(msg.ToolCalls)
		if err != nil {
			log.Printf("Error marshalling tool calls for persistence: %v", err)
			// 失败则存空，保证数据不会阻塞
		} else {
			toolCallsJSON = string(data)
		}
	}

	// TODO: 您需要从上下文或外部获取一个真实的 session ID
	sessionID := "current_user_session_abc"

	// 2. 准备 SQL 语句
	stmt, err := p.db.Prepare(`
		INSERT INTO messages
		(session_id, message_role, content, tool_calls_json, tool_call_id)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Printf("Error preparing SQL statement: %v", err)
		return
	}
	defer stmt.Close()

	// 3. 执行 SQL 插入
	_, err = stmt.Exec(
		sessionID,
		msg.Role,
		msg.Content,
		toolCallsJSON,
		msg.ToolCallId,
	)

	if err != nil {
		log.Printf("Error inserting message into DB: %v", err)
	} else {
		fmt.Printf("✅ DB Persistence: [Event: %s, Role: %s] | Content Snippet: %s...\n",
			event, msg.Role, strings.TrimSpace(msg.Content[:min(len(msg.Content), 30)]))
	}
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
