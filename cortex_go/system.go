package cortexgo

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dyike/CortexGo/cortex_go/agents"
	"github.com/dyike/CortexGo/cortex_go/utils"
	"github.com/gorilla/websocket"
)

// DateTimeEncoder handles custom JSON encoding for time.Time
type DateTimeEncoder struct {
	*json.Encoder
}

func (e *DateTimeEncoder) EncodeTime(t time.Time) error {
	return e.Encode(t.Format(time.RFC3339))
}

// SystemInstructor represents the main orchestrator class
type SystemInstructor struct {
	websocket            *websocket.Conn
	streamOutput         *utils.StreamResponse
	orchestratorResponse []utils.StreamResponse
	logger               *Logger
	orchestratorAgent    *agents.OrchestratorAgent
}

// NewSystemInstructor creates a new instance of SystemInstructor
func NewSystemInstructor() *SystemInstructor {
	logger := &Logger{}
	logger.configure()

	return &SystemInstructor{
		logger:               logger,
		orchestratorResponse: make([]utils.StreamResponse, 0),
		orchestratorAgent:    agents.NewOrchestratorAgent(),
	}
}

// safeWebSocketSend safely sends a message through the websocket
func (s *SystemInstructor) safeWebSocketSend(message interface{}) error {
	if s.websocket != nil {
		jsonData, err := json.Marshal(message)
		if err != nil {
			s.logger.error(fmt.Sprintf("Failed to marshal message: %v", err))
			return err
		}

		if err := s.websocket.WriteMessage(websocket.TextMessage, jsonData); err != nil {
			s.logger.error(fmt.Sprintf("WebSocket send failed: %v", err))
			return err
		}

		s.logger.debug(fmt.Sprintf("WebSocket message sent: %v", message))
		return nil
	}
	return nil
}

// SetWebSocket sets the websocket connection for the instructor
func (s *SystemInstructor) SetWebSocket(ws *websocket.Conn) {
	s.websocket = ws
}

// detectTaskType determines the type of task
func (s *SystemInstructor) detectTaskType(task string) string {
	taskLower := strings.ToLower(task)

	if strings.Contains(taskLower, "code") ||
		strings.Contains(taskLower, "implement") ||
		strings.Contains(taskLower, "develop") ||
		strings.Contains(taskLower, "programming") {
		return "coding"
	}

	if strings.Contains(taskLower, "browse") ||
		strings.Contains(taskLower, "search") ||
		strings.Contains(taskLower, "find") ||
		strings.Contains(taskLower, "web") ||
		strings.Contains(taskLower, "internet") {
		return "web_surfing"
	}

	return "general"
}

// processTask processes the task based on the plan
func (s *SystemInstructor) processTask(deps *agents.OrchestratorDeps, task string, planResult string) (string, error) {
	taskType := s.detectTaskType(task)

	s.streamOutput.Steps = append(s.streamOutput.Steps, fmt.Sprintf("Detected task type: %s", taskType))
	s.safeWebSocketSend(s.streamOutput)

	var result string
	var err error

	switch taskType {
	case "coding":
		s.streamOutput.Steps = append(s.streamOutput.Steps, "Executing coding task...")
		s.safeWebSocketSend(s.streamOutput)

		result, err = s.orchestratorAgent.CoderTask(deps, task)
		if err != nil {
			s.streamOutput.Steps = append(s.streamOutput.Steps, fmt.Sprintf("Error in coding task: %v", err))
			s.safeWebSocketSend(s.streamOutput)
			return "", err
		}

		s.streamOutput.Steps = append(s.streamOutput.Steps, "Coding task completed")

	case "web_surfing":
		s.streamOutput.Steps = append(s.streamOutput.Steps, "Executing web surfing task...")
		s.safeWebSocketSend(s.streamOutput)

		result, err = s.orchestratorAgent.WebSurferTask(deps, task)
		if err != nil {
			s.streamOutput.Steps = append(s.streamOutput.Steps, fmt.Sprintf("Error in web surfing task: %v", err))
			s.safeWebSocketSend(s.streamOutput)
			return "", err
		}

		s.streamOutput.Steps = append(s.streamOutput.Steps, "Web surfing task completed")

	default:
		// For general tasks, we'll use the plan result directly
		result = planResult
		s.streamOutput.Steps = append(s.streamOutput.Steps, "General task completed based on plan")
	}

	s.safeWebSocketSend(s.streamOutput)
	return result, nil
}

// Run executes the main orchestration loop
func (s *SystemInstructor) Run(task string) ([]string, error) {
	s.streamOutput = &utils.StreamResponse{
		AgentName:    "Orchestrator",
		Instructions: task,
		Steps:        make([]string, 0),
		Output:       "",
		StatusCode:   0,
	}
	s.orchestratorResponse = append(s.orchestratorResponse, *s.streamOutput)

	// Send initial state
	if err := s.safeWebSocketSend(s.streamOutput); err != nil {
		return nil, err
	}

	// Initialize system
	s.streamOutput.Steps = append(s.streamOutput.Steps, "Agents initialized successfully")
	if err := s.safeWebSocketSend(s.streamOutput); err != nil {
		return nil, err
	}

	// Create orchestrator dependencies
	orchestratorDeps := agents.NewOrchestratorDeps(
		s.websocket,
		s.streamOutput,
		s.orchestratorResponse,
	)

	// STEP 1: Plan the task using the orchestrator agent
	s.logger.info("Starting task planning")
	planResult, err := s.orchestratorAgent.PlanTask(orchestratorDeps, task)
	if err != nil {
		s.logger.error(fmt.Sprintf("Error planning task: %v", err))
		s.streamOutput.StatusCode = 500
		s.streamOutput.Output = fmt.Sprintf("Error planning task: %v", err)
		s.safeWebSocketSend(s.streamOutput)
		return nil, err
	}

	// STEP 2: Process the task based on the plan
	s.logger.info("Starting task processing")
	result, err := s.processTask(orchestratorDeps, task, planResult)
	if err != nil {
		s.logger.error(fmt.Sprintf("Error processing task: %v", err))
		s.streamOutput.StatusCode = 500
		s.streamOutput.Output = fmt.Sprintf("Error processing task: %v", err)
		s.safeWebSocketSend(s.streamOutput)
		return nil, err
	}

	// Update final result
	s.streamOutput.Output = result
	s.streamOutput.StatusCode = 200
	s.logger.debug(fmt.Sprintf("Final response: %v", result))
	s.streamOutput.Steps = append(s.streamOutput.Steps, "Task completed successfully")

	if err := s.safeWebSocketSend(s.streamOutput); err != nil {
		return nil, err
	}

	s.logger.info("Task completed successfully")

	// Convert responses to string array
	var results []string
	for _, resp := range s.orchestratorResponse {
		jsonData, err := json.Marshal(resp)
		if err != nil {
			continue
		}
		results = append(results, string(jsonData))
	}

	return results, nil
}

// Shutdown performs a clean shutdown of the orchestrator
func (s *SystemInstructor) Shutdown() error {
	if s.websocket != nil {
		if err := s.websocket.Close(); err != nil {
			s.logger.error(fmt.Sprintf("Error during websocket shutdown: %v", err))
			return err
		}
	}

	// Clear responses
	s.orchestratorResponse = make([]utils.StreamResponse, 0)
	s.logger.info("Orchestrator shutdown complete")
	return nil
}
