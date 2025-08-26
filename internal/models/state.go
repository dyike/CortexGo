package models

import (
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/internal/config"
)

type InvestDebateState struct {
	BullHistory     string `json:"bull_history"`
	BearHistory     string `json:"bear_history"`
	History         string `json:"history"`
	CurrentResponse string `json:"current_response"`
	JudgeDecision   string `json:"judge_decision"`
	Count           int    `json:"count"`
	MaxRounds       int    `json:"max_rounds"`
	CurrentRound    int    `json:"current_round"`
}

type RiskDebateState struct {
	RiskyHistory           string `json:"risky_history"`
	SafeHistory            string `json:"safe_history"`
	NeutralHistory         string `json:"neutral_history"`
	History                string `json:"history"`
	CurrentRiskyResponse   string `json:"current_risky_response"`
	CurrentSafeResponse    string `json:"current_safe_response"`
	CurrentNeutralResponse string `json:"current_neutral_response"`
	JudgeDecision          string `json:"judge_decision"`
	LatestSpeaker          string `json:"latest_speaker"`
	Count                  int    `json:"count"`
	MaxRounds              int    `json:"max_rounds"`
	CurrentRound           int    `json:"current_round"`
}

type TradingState struct {
	Messages          []*schema.Message `json:"messages"` // User messages
	CompanyOfInterest string            `json:"company_of_interest"`
	TradeDate         string            `json:"trade_date"`
	MarketData        []*MarketData     `json:"market_data"`

	MarketReport       string `json:"market_report"`
	FundamentalsReport string `json:"fundamentals_report"`
	NewsReport         string `json:"news_report"`
	SocialReport       string `json:"social_report"`

	SentimentReport       string             `json:"sentiment_report"`
	InvestmentDebateState *InvestDebateState `json:"investment_debate_state"`
	RiskDebateState       *RiskDebateState   `json:"risk_debate_state"`
	TraderInvestmentPlan  string             `json:"trader_investment_plan"`
	InvestmentPlan        string             `json:"investment_plan"`
	FinalTradeDecision    string             `json:"final_trade_decision"`
	Decision              *TradingDecision   `json:"decision"`
	Goto                  string             `json:"goto"`
	MaxIterations         int                `json:"max_iterations"`
	CurrentIteration      int                `json:"current_iteration"`
	Config                *config.Config     `json:"config"` // Configuration for dynamic behavior

	// Enhanced fields to match Python version
	Phase                 string `json:"phase"`                   // Current workflow phase: analysis, debate, trading, risk
	WorkflowComplete      bool   `json:"workflow_complete"`       // Whether the workflow has finished
	AnalysisPhaseComplete bool   `json:"analysis_phase_complete"` // Whether all 4 analysts have completed
	DebatePhaseComplete   bool   `json:"debate_phase_complete"`   // Whether debate phase is complete
	TradingPhaseComplete  bool   `json:"trading_phase_complete"`  // Whether trading phase is complete
	RiskPhaseComplete     bool   `json:"risk_phase_complete"`     // Whether risk phase is complete

	// Agent completion tracking
	SocialAnalystComplete       bool `json:"social_analyst_complete"`
	NewsAnalystComplete         bool `json:"news_analyst_complete"`
	FundamentalsAnalystComplete bool `json:"fundamentals_analyst_complete"`

	// Memory and reflection data
	PreviousDecisions []TradingDecision `json:"previous_decisions"` // Historical decisions for learning
	ReflectionNotes   string            `json:"reflection_notes"`   // Reflections on past performance
}

func NewTradingState(symbol string, date time.Time, userPrompt string, cfg *config.Config) *TradingState {
	return &TradingState{
		Messages: []*schema.Message{
			schema.UserMessage(userPrompt),
		},
		CompanyOfInterest: symbol,
		TradeDate:         date.Format("2006-01-02"),
		MarketData:        make([]*MarketData, 0),
		InvestmentDebateState: &InvestDebateState{
			History:         "",
			CurrentResponse: "",
			Count:           0,
			MaxRounds:       2, // Default 2 rounds of debate
			CurrentRound:    0,
		},
		RiskDebateState: &RiskDebateState{
			History:                "",
			CurrentRiskyResponse:   "",
			CurrentSafeResponse:    "",
			CurrentNeutralResponse: "",
			Count:                  0,
			MaxRounds:              2, // Default 2 rounds of risk discussion
			CurrentRound:           0,
		},
		MarketReport:       "",
		FundamentalsReport: "",
		SentimentReport:    "",
		NewsReport:         "",
		MaxIterations:      20,
		CurrentIteration:   0,
		Goto:               "market_analyst",
		Config:             cfg, // Store configuration for dynamic behavior

		// Initialize enhanced fields
		Phase:                       "analysis",
		WorkflowComplete:            false,
		AnalysisPhaseComplete:       false,
		DebatePhaseComplete:         false,
		TradingPhaseComplete:        false,
		RiskPhaseComplete:           false,
		SocialAnalystComplete:       false,
		NewsAnalystComplete:         false,
		FundamentalsAnalystComplete: false,
		PreviousDecisions:           []TradingDecision{},
		ReflectionNotes:             "",
	}
}
