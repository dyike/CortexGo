package models

type ToolResp struct {
	ID   string         `json:"id,omitempty" form:"id,omitempty"`
	Type string         `json:"type,omitempty" form:"type,omitempty"`
	Name string         `json:"name,omitempty" form:"name,omitempty"`
	Args map[string]any `json:"args,omitempty" form:"args,omitempty"`
}

type ToolChunkResp struct {
	ID   string `json:"id,omitempty" form:"id,omitempty"`
	Type string `json:"type,omitempty" form:"type,omitempty"`
	Name string `json:"name,omitempty" form:"name,omitempty"`
	Args string `json:"args,omitempty" form:"args,omitempty"`
}

type ChatResp struct {
	ThreadID       string           `json:"thread_id,omitempty" form:"thread_id"`
	Agent          string           `json:"agent,omitempty" form:"agent"`
	ID             string           `json:"id,omitempty" form:"id"`
	Role           string           `json:"role,omitempty" form:"role"`
	Content        string           `json:"content,omitempty" form:"content"`
	FinishReason   string           `json:"finish_reason,omitempty" form:"finish_reason"`
	Options        []map[string]any `json:"options,omitempty" form:"options"`
	ToolCallID     string           `json:"tool_call_id,omitempty" form:"tool_call_id"`
	ToolCalls      []ToolResp       `json:"tool_calls,omitempty" form:"tool_calls"`
	ToolCallChunks []ToolChunkResp  `json:"tool_call_chunks,omitempty" form:"tool_call_chunks"`
	MessageChunks  any              `json:"message_chunks,omitempty" form:"message_chunks"`
}
