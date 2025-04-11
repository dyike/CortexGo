package agents

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dyike/CortexGo/cortex_go/prompt"
	"github.com/dyike/CortexGo/cortex_go/utils"
	"github.com/gorilla/websocket"
)

// OrchestratorDeps represents dependencies for the orchestrator agent
type OrchestratorDeps struct {
	WebSocket      *websocket.Conn
	StreamOutput   *utils.StreamResponse
	AgentResponses []utils.StreamResponse
}

// NewOrchestratorDeps creates a new instance of OrchestratorDeps
func NewOrchestratorDeps(ws *websocket.Conn, streamOutput *utils.StreamResponse, agentResponses []utils.StreamResponse) *OrchestratorDeps {
	return &OrchestratorDeps{
		WebSocket:      ws,
		StreamOutput:   streamOutput,
		AgentResponses: agentResponses,
	}
}

// OrchestratorAgent represents the orchestrator agent
type OrchestratorAgent struct {
	systemPrompt string
	modelName    string
	apiClient    interface{} // Replace with actual client type
	plannerAgent *PlannerAgent
	coderAgent   *CoderAgent
}

// NewOrchestratorAgent creates a new instance of OrchestratorAgent
func NewOrchestratorAgent() *OrchestratorAgent {
	systemPrompt := prompt.OrchestratorSystemPrompt

	return &OrchestratorAgent{
		systemPrompt: systemPrompt,
		modelName:    os.Getenv("MODEL_NAME"),
		plannerAgent: NewPlannerAgent(),
		coderAgent:   NewCoderAgent(),
	}
}

// SafeWebSocketSend safely sends a message through the websocket
func SafeWebSocketSend(ws *websocket.Conn, message interface{}) error {
	if ws != nil {
		jsonData, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("[ERROR] Failed to marshal message: %v\n", err)
			return err
		}

		if err := ws.WriteMessage(websocket.TextMessage, jsonData); err != nil {
			fmt.Printf("[ERROR] WebSocket send failed: %v\n", err)
			return err
		}

		fmt.Printf("[DEBUG] WebSocket message sent: %v\n", message)
		return nil
	}
	return nil
}

// PlanTask plans the task and assigns it to the appropriate agents
func (oa *OrchestratorAgent) PlanTask(deps *OrchestratorDeps, task string) (string, error) {
	fmt.Printf("[INFO] Planning task: %s\n", task)

	// Create a new StreamResponse for Planner Agent
	plannerStreamOutput := utils.StreamResponse{
		AgentName:    "Planner Agent",
		Instructions: task,
		Steps:        make([]string, 0),
		Output:       "",
		StatusCode:   0,
	}

	// Add to orchestrator's response collection if available
	if len(deps.AgentResponses) > 0 {
		deps.AgentResponses = append(deps.AgentResponses, plannerStreamOutput)
	}

	// Send initial update for Planner Agent
	SafeWebSocketSend(deps.WebSocket, plannerStreamOutput)

	// Update planner stream
	plannerStreamOutput.Steps = append(plannerStreamOutput.Steps, "Planning task...")
	SafeWebSocketSend(deps.WebSocket, plannerStreamOutput)

	// Run planner agent
	plannerResponse, err := oa.plannerAgent.Run(task)
	if err != nil {
		errMsg := fmt.Sprintf("Error planning task: %v", err)
		fmt.Printf("[ERROR] %s\n", errMsg)

		plannerStreamOutput.Steps = append(plannerStreamOutput.Steps, "Planning failed: "+errMsg)
		plannerStreamOutput.StatusCode = 500
		SafeWebSocketSend(deps.WebSocket, plannerStreamOutput)

		return "", fmt.Errorf(errMsg)
	}

	// Update planner stream with results
	planText := plannerResponse.Plan
	plannerStreamOutput.Steps = append(plannerStreamOutput.Steps, "Task planned successfully")
	plannerStreamOutput.Output = planText
	plannerStreamOutput.StatusCode = 200
	SafeWebSocketSend(deps.WebSocket, plannerStreamOutput)

	// Also update orchestrator stream
	deps.StreamOutput.Steps = append(deps.StreamOutput.Steps, "Task planned successfully")
	SafeWebSocketSend(deps.WebSocket, deps.StreamOutput)

	return fmt.Sprintf("Task planned successfully\nTask: %s", planText), nil
}

// CoderTask assigns coding tasks to the coder agent
func (oa *OrchestratorAgent) CoderTask(deps *OrchestratorDeps, task string) (string, error) {
	fmt.Printf("[INFO] Assigning coding task: %s\n", task)

	// Create a new StreamResponse for Coder Agent
	coderStreamOutput := utils.StreamResponse{
		AgentName:    "Coder Agent",
		Instructions: task,
		Steps:        make([]string, 0),
		Output:       "",
		StatusCode:   0,
	}

	// Add to orchestrator's response collection if available
	if len(deps.AgentResponses) > 0 {
		deps.AgentResponses = append(deps.AgentResponses, coderStreamOutput)
	}

	// Create deps for coder agent
	coderDeps := NewCoderAgentDeps(deps.WebSocket, &coderStreamOutput)

	// Run coder agent
	coderResponse, err := oa.coderAgent.Run(task, coderDeps)
	if err != nil {
		errMsg := fmt.Sprintf("Error executing coding task: %v", err)
		fmt.Printf("[ERROR] %s\n", errMsg)

		coderStreamOutput.Steps = append(coderStreamOutput.Steps, "Coding failed: "+errMsg)
		coderStreamOutput.StatusCode = 500
		SafeWebSocketSend(deps.WebSocket, coderStreamOutput)

		return "", fmt.Errorf(errMsg)
	}

	return coderResponse.Content, nil
}

// WebSurferTask assigns web surfing tasks to the web surfer agent
func (oa *OrchestratorAgent) WebSurferTask(deps *OrchestratorDeps, task string) (string, error) {
	fmt.Printf("[INFO] Assigning web surfing task: %s\n", task)

	// Create a new StreamResponse for WebSurfer
	webSurferStreamOutput := utils.StreamResponse{
		AgentName:    "Web Surfer",
		Instructions: task,
		Steps:        make([]string, 0),
		Output:       "",
		StatusCode:   0,
	}

	// Add to orchestrator's response collection if available
	if len(deps.AgentResponses) > 0 {
		deps.AgentResponses = append(deps.AgentResponses, webSurferStreamOutput)
	}

	// Initialize WebSurfer agent
	webSurfer := NewWebSurferAgent("http://localhost:8000/api/v1/web/stream")

	// Run web surfer agent
	success, message, _, err := webSurfer.GenerateReply(task, deps.WebSocket, &webSurferStreamOutput)
	if err != nil {
		errMsg := fmt.Sprintf("Error executing web surfing task: %v", err)
		fmt.Printf("[ERROR] %s\n", errMsg)

		webSurferStreamOutput.Steps = append(webSurferStreamOutput.Steps, "Web surfing failed: "+errMsg)
		webSurferStreamOutput.StatusCode = 500
		SafeWebSocketSend(deps.WebSocket, webSurferStreamOutput)

		return "", fmt.Errorf(errMsg)
	}

	if !success {
		errMsg := "Web surfing completed with issues"
		fmt.Printf("[ERROR] %s\n", errMsg)

		webSurferStreamOutput.Steps = append(webSurferStreamOutput.Steps, errMsg)
		webSurferStreamOutput.StatusCode = 500
		SafeWebSocketSend(deps.WebSocket, webSurferStreamOutput)

		return message, fmt.Errorf(errMsg)
	}

	return message, nil
}
