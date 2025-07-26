package models

import (
	"time"

	"github.com/cloudwego/eino/schema"
)

type InvestDebateState struct {
	BullHistory     string `json:"bull_history"`
	BearHistory     string `json:"bear_history"`
	History         string `json:"history"`
	CurrentResponse string `json:"current_response"`
	JudgeDecision   string `json:"judge_decision"`
	Count           int    `json:"count"`
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
}

type TradingState struct {
	Messages              []*schema.Message  `json:"messages"`
	CompanyOfInterest     string             `json:"company_of_interest"`
	TradeDate             string             `json:"trade_date"`
	MarketData            *MarketData        `json:"market_data"`
	MarketReport          string             `json:"market_report"`
	FundamentalsReport    string             `json:"fundamentals_report"`
	SentimentReport       string             `json:"sentiment_report"`
	NewsReport            string             `json:"news_report"`
	InvestmentDebateState *InvestDebateState `json:"investment_debate_state"`
	RiskDebateState       *RiskDebateState   `json:"risk_debate_state"`
	TraderInvestmentPlan  string             `json:"trader_investment_plan"`
	InvestmentPlan        string             `json:"investment_plan"`
	FinalTradeDecision    string             `json:"final_trade_decision"`
	Decision              *TradingDecision   `json:"decision"`
	Goto                  string             `json:"goto"`
	MaxIterations         int                `json:"max_iterations"`
	CurrentIteration      int                `json:"current_iteration"`
}

func NewTradingState(symbol string, date time.Time, userPrompt string) *TradingState {
	return &TradingState{
		Messages: []*schema.Message{
			schema.UserMessage(userPrompt),
		},
		CompanyOfInterest: symbol,
		TradeDate:         date.Format("2006-01-02"),
		MarketData: &MarketData{
			Symbol:    symbol,
			Timestamp: date,
		},
		InvestmentDebateState: &InvestDebateState{
			History:         "",
			CurrentResponse: "",
			Count:           0,
		},
		RiskDebateState: &RiskDebateState{
			History:                "",
			CurrentRiskyResponse:   "",
			CurrentSafeResponse:    "",
			CurrentNeutralResponse: "",
			Count:                  0,
		},
		MarketReport:       "",
		FundamentalsReport: "",
		SentimentReport:    "",
		NewsReport:         "",
		MaxIterations:      20,
		CurrentIteration:   0,
		Goto:               "market_analyst",
	}
}
