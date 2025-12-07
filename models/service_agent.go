package models

// AgentInitParams 描述交易 agent 初始化参数
type AgentInitParams struct {
	Symbol    string `json:"symbol"`
	TradeDate string `json:"trade_date"`
	Prompt    string `json:"prompt"`
}
