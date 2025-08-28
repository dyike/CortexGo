package cli

import "time"

// AnalystType represents different types of analysts
type AnalystType string

const (
	MarketAnalyst       AnalystType = "market"
	SocialAnalyst       AnalystType = "social"
	NewsAnalyst         AnalystType = "news"
	FundamentalsAnalyst AnalystType = "fundamentals"
)

// ResearchDepth represents the depth of analysis
type ResearchDepth string

const (
	ShallowResearch ResearchDepth = "shallow" // 1 round
	MediumResearch  ResearchDepth = "medium"  // 3 rounds
	DeepResearch    ResearchDepth = "deep"    // 5 rounds
)

// LLMProvider represents available LLM providers
type LLMProvider string

const (
	OpenAIProvider     LLMProvider = "openai"
	AnthropicProvider  LLMProvider = "anthropic"
	GoogleProvider     LLMProvider = "google"
	OpenRouterProvider LLMProvider = "openrouter"
	OllamaProvider     LLMProvider = "ollama"
)

// AgentStatus represents the status of an agent
type AgentStatus string

const (
	StatusPending    AgentStatus = "pending"
	StatusInProgress AgentStatus = "in_progress"
	StatusCompleted  AgentStatus = "completed"
	StatusError      AgentStatus = "error"
)

// UserSelections holds all user choices for the analysis
type UserSelections struct {
	Ticker        string        `json:"ticker"`
	AnalysisDate  time.Time     `json:"analysis_date"`
	Analysts      []AnalystType `json:"analysts"`
	ResearchDepth ResearchDepth `json:"research_depth"`
	LLMProvider   LLMProvider   `json:"llm_provider"`
	QuickModel    string        `json:"quick_model"`
	DeepModel     string        `json:"deep_model"`
}

// AnalysisProgress tracks the progress of all agents
type AnalysisProgress struct {
	// Analyst Team
	MarketAnalyst       AgentStatus `json:"market_analyst"`
	SocialAnalyst       AgentStatus `json:"social_analyst"`
	NewsAnalyst         AgentStatus `json:"news_analyst"`
	FundamentalsAnalyst AgentStatus `json:"fundamentals_analyst"`

	// Research Team
	BullResearcher  AgentStatus `json:"bull_researcher"`
	BearResearcher  AgentStatus `json:"bear_researcher"`
	ResearchManager AgentStatus `json:"research_manager"`

	// Trading Team
	Trader AgentStatus `json:"trader"`

	// Risk Management Team
	RiskyAnalyst   AgentStatus `json:"risky_analyst"`
	SafeAnalyst    AgentStatus `json:"safe_analyst"`
	NeutralAnalyst AgentStatus `json:"neutral_analyst"`
	RiskManager    AgentStatus `json:"risk_manager"`
}

// AnalysisStats tracks statistics during analysis
type AnalysisStats struct {
	ToolCallsCount   int           `json:"tool_calls_count"`
	LLMCallsCount    int           `json:"llm_calls_count"`
	ReportsGenerated int           `json:"reports_generated"`
	CurrentPhase     string        `json:"current_phase"`
	StartTime        time.Time     `json:"start_time"`
	ElapsedTime      time.Duration `json:"elapsed_time"`
}

// LogMessage represents a log entry in the analysis
type LogMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Agent     string    `json:"agent"`
	Type      string    `json:"type"` // "tool_call", "reasoning", "report", "error"
	Content   string    `json:"content"`
	Level     string    `json:"level"` // "info", "warn", "error"
}

// ReportSection represents a generated report section
type ReportSection struct {
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Agent     string    `json:"agent"`
	Timestamp time.Time `json:"timestamp"`
	FilePath  string    `json:"file_path"`
}

// AnalysisSession holds the complete state of an analysis session
type AnalysisSession struct {
	Selections UserSelections   `json:"selections"`
	Progress   AnalysisProgress `json:"progress"`
	Stats      AnalysisStats    `json:"stats"`
	Messages   []LogMessage     `json:"messages"`
	Reports    []ReportSection  `json:"reports"`
	ResultsDir string           `json:"results_dir"`
	SessionID  string           `json:"session_id"`
}

// GetAnalystDisplayName returns a user-friendly name for the analyst type
func (a AnalystType) GetDisplayName() string {
	switch a {
	case MarketAnalyst:
		return "Market Analyst"
	case SocialAnalyst:
		return "Social Media Analyst"
	case NewsAnalyst:
		return "News Analyst"
	case FundamentalsAnalyst:
		return "Fundamentals Analyst"
	default:
		return string(a)
	}
}

// GetResearchRounds returns the number of rounds for each research depth
func (r ResearchDepth) GetResearchRounds() int {
	switch r {
	case ShallowResearch:
		return 1
	case MediumResearch:
		return 3
	case DeepResearch:
		return 5
	default:
		return 1
	}
}

// GetProviderModels returns available models for each provider
func (p LLMProvider) GetProviderModels() ([]string, []string) {
	switch p {
	case OpenAIProvider:
		quick := []string{"gpt-4o-mini", "gpt-3.5-turbo", "gpt-4o"}
		deep := []string{"o4-mini", "o1-preview", "gpt-4o", "gpt-4-turbo"}
		return quick, deep
	case AnthropicProvider:
		quick := []string{"claude-3-haiku", "claude-3-sonnet"}
		deep := []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"}
		return quick, deep
	case GoogleProvider:
		quick := []string{"gemini-1.5-flash", "gemini-1.5-pro"}
		deep := []string{"gemini-1.5-pro", "gemini-1.5-flash"}
		return quick, deep
	case OpenRouterProvider:
		quick := []string{"openai/gpt-4o-mini", "anthropic/claude-3-haiku"}
		deep := []string{"openai/o1-preview", "anthropic/claude-3-opus"}
		return quick, deep
	case OllamaProvider:
		quick := []string{"llama3.2", "qwen2.5", "mistral"}
		deep := []string{"llama3.1:70b", "qwen2.5:32b", "deepseek-coder"}
		return quick, deep
	default:
		return []string{}, []string{}
	}
}

// IsCompleted checks if all selected analysts have completed
func (p *AnalysisProgress) IsCompleted(selectedAnalysts []AnalystType) bool {
	// Check analyst team
	for _, analyst := range selectedAnalysts {
		switch analyst {
		case MarketAnalyst:
			if p.MarketAnalyst != StatusCompleted {
				return false
			}
		case SocialAnalyst:
			if p.SocialAnalyst != StatusCompleted {
				return false
			}
		case NewsAnalyst:
			if p.NewsAnalyst != StatusCompleted {
				return false
			}
		case FundamentalsAnalyst:
			if p.FundamentalsAnalyst != StatusCompleted {
				return false
			}
		}
	}

	// Check other teams
	return p.ResearchManager == StatusCompleted &&
		p.Trader == StatusCompleted &&
		p.RiskManager == StatusCompleted
}

// GetCompletedCount returns the number of completed agents
func (p *AnalysisProgress) GetCompletedCount() int {
	count := 0
	statuses := []AgentStatus{
		p.MarketAnalyst, p.SocialAnalyst, p.NewsAnalyst, p.FundamentalsAnalyst,
		p.BullResearcher, p.BearResearcher, p.ResearchManager,
		p.Trader,
		p.RiskyAnalyst, p.SafeAnalyst, p.NeutralAnalyst, p.RiskManager,
	}

	for _, status := range statuses {
		if status == StatusCompleted {
			count++
		}
	}

	return count
}

// GetTotalAgents returns the total number of agents
func (p *AnalysisProgress) GetTotalAgents() int {
	return 12 // Total number of agents in the system
}
