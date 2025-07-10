package models

import "time"

type TradingDecision struct {
	Symbol    string    `json:"symbol"`
	Date      time.Time `json:"date"`
	Action    string    `json:"action"`
	Quantity  float64   `json:"quantity"`
	Price     float64   `json:"price"`
	Reason    string    `json:"reason"`
	Confidence float64  `json:"confidence"`
	Risk      float64   `json:"risk"`
}

type MarketData struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    int64     `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Open      float64   `json:"open"`
	Close     float64   `json:"close"`
}

type AnalysisReport struct {
	Analyst   string    `json:"analyst"`
	Symbol    string    `json:"symbol"`
	Date      time.Time `json:"date"`
	Analysis  string    `json:"analysis"`
	Rating    string    `json:"rating"`
	Confidence float64  `json:"confidence"`
	Metrics   map[string]interface{} `json:"metrics"`
}

type AgentState struct {
	CurrentSymbol string                 `json:"current_symbol"`
	CurrentDate   time.Time              `json:"current_date"`
	MarketData    *MarketData            `json:"market_data"`
	Reports       []AnalysisReport       `json:"reports"`
	Decision      *TradingDecision       `json:"decision"`
	Metadata      map[string]interface{} `json:"metadata"`
}