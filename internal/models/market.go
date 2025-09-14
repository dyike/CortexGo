package models

type MarketDataInput struct {
	Symbol string `json:"symbol"`
	Count  int    `json:"count"`
}

type MarketDataOutput struct {
	Data []*MarketData `json:"data"`
}

type MarketData struct {
	Symbol string  `json:"symbol"`
	Volume int64   `json:"volume"`
	Date   string  `json:"date"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
}

// StockIndicatorInput represents the input for technical indicator analysis
type StockIndicatorInput struct {
	Symbol       string `json:"symbol"`
	Indicator    string `json:"indicator"`
	CurrDate     string `json:"curr_date"`
	LookBackDays int    `json:"look_back_days"`
	Online       bool   `json:"online"`
}

// StockIndicatorOutput represents the output of technical indicator analysis
type StockIndicatorOutput struct {
	Result string `json:"result"`
}

// IndicatorValue represents a single indicator value at a specific date
type IndicatorValue struct {
	Date  string
	Value float64
}

