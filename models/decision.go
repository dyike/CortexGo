package models

// TradingDecision represents a trading decision made by the system
type TradingDecision struct {
	Symbol       string  `json:"symbol"`
	Date         string  `json:"date"`
	Timestamp    string  `json:"timestamp"`
	Action       string  `json:"action"`
	Quantity     float64 `json:"quantity"`
	Price        float64 `json:"price"`
	Reason       string  `json:"reason"`
	Reasoning    string  `json:"reasoning"`
	Confidence   float64 `json:"confidence"`
	Risk         float64 `json:"risk"`
	EntryPrice   float64 `json:"entry_price"`
	StopLoss     float64 `json:"stop_loss"`
	TakeProfit   float64 `json:"take_profit"`
	PositionSize float64 `json:"position_size"`
}
