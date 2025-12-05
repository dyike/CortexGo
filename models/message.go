package models

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

type ToolResp struct {
	ID   string         `json:"id,omitempty" form:"id,omitempty"`
	Type string         `json:"type,omitempty" form:"type,omitempty"`
	Name string         `json:"name,omitempty" form:"name,omitempty"`
	Args map[string]any `json:"args,omitempty" form:"args,omitempty"`
}

type ToolChunkResp struct {
	ID   string `json:"id,omitempty" form:"id,omitempty"`
	Type string `json:"type,omitempty" form:"type,omitempty"`
	Name string `json:"name,omitempty" form:"name,omitempty"`
	Args string `json:"args,omitempty" form:"args,omitempty"`
}

type ChatResp struct {
	ThreadID       string                   `json:"thread_id,omitempty" form:"thread_id"`
	Agent          string                   `json:"agent,omitempty" form:"agent"`
	ID             string                   `json:"id,omitempty" form:"id"`
	Role           string                   `json:"role,omitempty" form:"role"`
	Content        string                   `json:"content,omitempty" form:"content"`
	FinishReason   string                   `json:"finish_reason,omitempty" form:"finish_reason"`
	Options        []map[string]interface{} `json:"options,omitempty" form:"options"`
	ToolCallID     string                   `json:"tool_call_id,omitempty" form:"tool_call_id"`
	ToolCalls      []ToolResp               `json:"tool_calls,omitempty" form:"tool_calls"`
	ToolCallChunks []ToolChunkResp          `json:"tool_call_chunks,omitempty" form:"tool_call_chunks"`
	MessageChunks  any                      `json:"message_chunks,omitempty" form:"message_chunks"`
}
