package agents

import (
	"fmt"
	"os"

	"github.com/dyike/CortexGo/cortex_go/utils"
	"github.com/gorilla/websocket"
)

// CoderAgentDeps represents dependencies for the coder agent
type CoderAgentDeps struct {
	WebSocket    *websocket.Conn
	StreamOutput *utils.StreamResponse
}

// NewCoderAgentDeps creates a new instance of CoderAgentDeps
func NewCoderAgentDeps(ws *websocket.Conn, streamOutput *utils.StreamResponse) *CoderAgentDeps {
	return &CoderAgentDeps{
		WebSocket:    ws,
		StreamOutput: streamOutput,
	}
}

// CoderResponse represents the response from the coder agent
type CoderResponse struct {
	Content string `json:"content"`
}

// CoderAgent represents the coder agent
type CoderAgent struct {
	systemPrompt string
	modelName    string
}

// NewCoderAgent creates a new instance of CoderAgent
func NewCoderAgent() *CoderAgent {
	systemPrompt := `You are a coding agent responsible for implementing technical solutions.
Your job is to write code based on the given requirements.`

	return &CoderAgent{
		systemPrompt: systemPrompt,
		modelName:    os.Getenv("ANTHROPIC_MODEL_NAME"),
	}
}

// Run executes the coder agent with the given prompt and dependencies
func (ca *CoderAgent) Run(userPrompt string, deps *CoderAgentDeps) (*CoderResponse, error) {
	// Update stream output if available
	if deps != nil && deps.StreamOutput != nil {
		deps.StreamOutput.Steps = append(deps.StreamOutput.Steps, "Processing coding task...")
		SafeWebSocketSend(deps.WebSocket, deps.StreamOutput)
	}

	// In a real implementation, this would call the AI model
	// For now, we'll just return a simple response
	content := fmt.Sprintf("Code implementation for: %s\n\n```go\nfunc main() {\n  fmt.Println(\"Implementation for: %s\")\n}\n```",
		userPrompt, userPrompt)

	// Update stream output with results
	if deps != nil && deps.StreamOutput != nil {
		deps.StreamOutput.Steps = append(deps.StreamOutput.Steps, "Code implementation generated")
		deps.StreamOutput.Output = content
		SafeWebSocketSend(deps.WebSocket, deps.StreamOutput)
	}

	return &CoderResponse{
		Content: content,
	}, nil
}
