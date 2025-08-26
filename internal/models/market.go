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
