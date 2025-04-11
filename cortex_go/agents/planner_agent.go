package agents

import (
	"fmt"
	"os"
)

// PlannerAgent represents the planner agent
type PlannerAgent struct {
	systemPrompt string
	modelName    string
}

// PlannerResponse represents the response from the planner agent
type PlannerResponse struct {
	Plan string `json:"plan"`
}

// NewPlannerAgent creates a new instance of PlannerAgent
func NewPlannerAgent() *PlannerAgent {
	systemPrompt := `You are a planning agent responsible for breaking down complex tasks into actionable steps.
Your job is to analyze the task and create a detailed plan of action.`

	return &PlannerAgent{
		systemPrompt: systemPrompt,
		modelName:    os.Getenv("ANTHROPIC_MODEL_NAME"),
	}
}

// Run executes the planner agent with the given prompt
func (pa *PlannerAgent) Run(userPrompt string) (*PlannerResponse, error) {
	// In a real implementation, this would call the AI model
	// For now, we'll just return a simple plan
	planText := fmt.Sprintf("1. Analyze task: %s\n2. Break down requirements\n3. Assign subtasks", userPrompt)
	return &PlannerResponse{
		Plan: planText,
	}, nil
}
