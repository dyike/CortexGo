package models

// type ToolResp struct {
// 	ID   string         `json:"id,omitempty" form:"id,omitempty"`
// 	Type string         `json:"type,omitempty" form:"type,omitempty"`
// 	Name string         `json:"name,omitempty" form:"name,omitempty"`
// 	Args map[string]any `json:"args,omitempty" form:"args,omitempty"`
// }

// type ToolChunkResp struct {
// 	ID   string `json:"id,omitempty" form:"id,omitempty"`
// 	Type string `json:"type,omitempty" form:"type,omitempty"`
// 	Name string `json:"name,omitempty" form:"name,omitempty"`
// 	Args string `json:"args,omitempty" form:"args,omitempty"`
// }

type ChatResp struct {
	Role       string      `json:"role"`
	Content    string      `json:"content"`
	ToolCalls  []*ToolCall `json:"tool_calls,omitempty"`
	ToolCallId string      `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Message struct {
	Role         string      `json:"role"`
	Content      string      `json:"content"`
	ToolCalls    []*ToolCall `json:"tool_calls"`
	ResponseMeta *struct {
		FinishReason string `json:"finish_reason"`
	} `json:"response_meta"`
}
