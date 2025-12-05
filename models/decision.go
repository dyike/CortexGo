package models

import "time"

type TradingDecision struct {
	Symbol       string  `json:"symbol"`
	Date         string  `json:"date"`      // Changed to string to match usage
	Timestamp    string  `json:"timestamp"` // Added for signal processor
	Action       string  `json:"action"`
	Quantity     float64 `json:"quantity"`
	Price        float64 `json:"price"`
	Reason       string  `json:"reason"`
	Reasoning    string  `json:"reasoning"` // Added for signal processor
	Confidence   float64 `json:"confidence"`
	Risk         float64 `json:"risk"`
	EntryPrice   float64 `json:"entry_price"`   // Added for signal processor
	StopLoss     float64 `json:"stop_loss"`     // Added for signal processor
	TakeProfit   float64 `json:"take_profit"`   // Added for signal processor
	PositionSize float64 `json:"position_size"` // Added for signal processor
}

type AnalysisReport struct {
	Analyst     string                 `json:"analyst"`
	Symbol      string                 `json:"symbol"`
	Date        time.Time              `json:"date"`
	Analysis    string                 `json:"analysis"`
	Rating      string                 `json:"rating"`
	Confidence  float64                `json:"confidence"`
	Metrics     map[string]interface{} `json:"metrics"`
	KeyFindings []string               `json:"key_findings"`
	Concerns    []string               `json:"concerns"`
	Priority    int                    `json:"priority"`
}

type AnalystDiscussion struct {
	Participants []string      `json:"participants"`
	Topic        string        `json:"topic"`
	DebatePoints []DebatePoint `json:"debate_points"`
	Consensus    *Consensus    `json:"consensus"`
	Timestamp    time.Time     `json:"timestamp"`
}

type DebatePoint struct {
	Analyst   string    `json:"analyst"`
	Position  string    `json:"position"`
	Evidence  []string  `json:"evidence"`
	Response  string    `json:"response"`
	Timestamp time.Time `json:"timestamp"`
}

type Consensus struct {
	FinalRating    string   `json:"final_rating"`
	AgreementLevel float64  `json:"agreement_level"`
	MainArguments  []string `json:"main_arguments"`
	Dissents       []string `json:"dissents"`
	Confidence     float64  `json:"confidence"`
}

type AgentState struct {
	CurrentSymbol string                 `json:"current_symbol"`
	CurrentDate   time.Time              `json:"current_date"`
	MarketData    *MarketData            `json:"market_data"`
	Reports       []AnalysisReport       `json:"reports"`
	Decision      *TradingDecision       `json:"decision"`
	Metadata      map[string]interface{} `json:"metadata"`
	Discussions   []AnalystDiscussion    `json:"discussions"`
	TeamConsensus *Consensus             `json:"team_consensus"`
}
