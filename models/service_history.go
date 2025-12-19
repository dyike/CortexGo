package models

// HistoryParams 描述查询历史列表的参数（书签分页，基于 sqlite rowid 游标）
type HistoryParams struct {
	Cursor string `json:"cursor"` // rowid 形式的书签，空表示第一页
	Limit  int    `json:"limit"`  // 每页数量，默认 50，最大 200
}

// HistorySession 表示一次会话的概要信息
type HistorySession struct {
	SessionID string `json:"session_id"`
	Symbol    string `json:"symbol"`
	TradeDate string `json:"trade_date"`
	Prompt    string `json:"prompt"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// HistoryInfoParams 描述按 session_id 读取历史详情的参数
type HistoryInfoParams struct {
	SessionID string `json:"session_id"` // 必填
}

// HistoryMessage 表示某个会话中的单条消息
type HistoryMessage struct {
	ID           string      `json:"id"`
	Role         string      `json:"role"`
	Agent        string      `json:"agent"`
	Content      string      `json:"content"`
	ToolCallId   string      `json:"tool_call_id,omitempty"`
	ToolName     string      `json:"tool_name,omitempty"`
	ToolCalls    []*ToolCall `json:"tool_calls,omitempty"`
	Status       string      `json:"status"`
	FinishReason string      `json:"finish_reason"`
	Seq          int         `json:"seq"`
	CreatedAt    string      `json:"created_at"`
	UpdatedAt    string      `json:"updated_at"`
}

// HistoryListResponse 历史会话列表返回
type HistoryListResponse struct {
	Items      []HistorySession `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
	HasMore    bool             `json:"has_more"`
}

// HistoryInfoResponse 单个会话的详情
type HistoryInfoResponse struct {
	Session  HistorySession   `json:"session"`
	Messages []HistoryMessage `json:"messages"`
}
