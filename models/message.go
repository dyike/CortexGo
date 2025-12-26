package models

// ChatResp represents a chat response from an agent
type ChatResp struct {
	AgentName  string      `json:"agent_name"`
	Role       string      `json:"role"`
	Content    string      `json:"content"`
	ToolCalls  []*ToolCall `json:"tool_calls,omitempty"`
	ToolCallId string      `json:"tool_call_id,omitempty"`
	ToolName   string      `json:"tool_name,omitempty"`
}

// ToolCall represents a tool call made by an agent
type ToolCall struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
