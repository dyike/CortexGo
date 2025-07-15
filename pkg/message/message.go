package message

import (
	"container/list"
	"time"

	"github.com/dyike/CortexGo/consts"
)

type Message struct {
	Timestamp string
	Type      string
	Content   string
}

type ToolCall struct {
	Timestamp string
	Name      string
	Args      string
}

type MessageBuffer struct {
	messages       *list.List
	toolCalls      *list.List
	maxLength      int
	currentReport  string
	finalReport    string
	agentStatus    map[string]string
	currentAgent   string
	reportSections map[string]string
	sectionOrder   []string
}

func NewMessageBuffer(maxLength int) *MessageBuffer {
	agentStatus := map[string]string{
		// Analyst Team
		consts.Agent_MarketAnalyst:       consts.State_Pending,
		consts.Agent_SocialAnalyst:       consts.State_Pending,
		consts.Agent_NewsAnalyst:         consts.State_Pending,
		consts.Agent_FundamentalsAnalyst: consts.State_Pending,
		// Research Team
		consts.Agent_BullResearcher:  consts.State_Pending,
		consts.Agent_BearResearcher:  consts.State_Pending,
		consts.Agent_ResearchManager: consts.State_Pending,
		// Trading Team
		consts.Agent_Trader: consts.State_Pending,
		// Risk Management Team
		consts.Agent_RiskyAnalyst:   consts.State_Pending,
		consts.Agent_NeutralAnalyst: consts.State_Pending,
		consts.Agent_SafeAnalyst:    consts.State_Pending,
		// Portfolio Management Team
		consts.Agent_PortfolioManager: consts.State_Pending,
	}
	sectionOrder := []string{
		"market_report",
		"sentiment_report",
		"news_report",
		"fundamentals_report",
		"investment_plan",
		"trader_investment_plan",
		"final_trade_decision",
	}
	return &MessageBuffer{
		messages:       list.New(),
		toolCalls:      list.New(),
		maxLength:      maxLength,
		agentStatus:    agentStatus,
		reportSections: make(map[string]string),
		sectionOrder:   sectionOrder,
	}
}

func (m *MessageBuffer) AddMessage(msgType, content string) {
	timestamp := time.Now().Format("15:05:05")
	m.messages.PushBack(Message{
		Timestamp: timestamp,
		Type:      msgType,
		Content:   content,
	})

}

func (m *MessageBuffer) AddToolCall() {

}

func (m *MessageBuffer) UpdateAgentStatus() {

}

func (m *MessageBuffer) UpdateReportSection() {

}

func (m *MessageBuffer) updateCurrentReport() {

}

func (m *MessageBuffer) updateFinalReport() {

}
