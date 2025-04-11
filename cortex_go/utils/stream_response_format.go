package utils

// StreamResponse represents the response structure for streaming messages
type StreamResponse struct {
	AgentName    string   `json:"agent_name"`
	Instructions string   `json:"instructions"`
	Steps        []string `json:"steps"`
	StatusCode   int      `json:"status_code"`
	Output       string   `json:"output"`
	LiveUrl      *string  `json:"live_url"`
}
