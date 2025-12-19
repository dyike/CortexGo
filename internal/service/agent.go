package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/graph"
	"github.com/dyike/CortexGo/internal/storage"
	"github.com/dyike/CortexGo/models"
	"github.com/dyike/CortexGo/pkg/bridge"
)

// StartAgentStream 启动交易编排流，并通过回调推送流式事件
func StartAgentStream(paramsJson string) (any, error) {
	var params models.AgentInitParams
	if err := json.Unmarshal([]byte(paramsJson), &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	params.Symbol = strings.TrimSpace(params.Symbol)
	if params.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	if strings.TrimSpace(params.TradeDate) == "" {
		params.TradeDate = time.Now().Format("2006-01-02")
	}
	parsedDate, err := time.Parse("2006-01-02", params.TradeDate)
	if err != nil {
		return nil, fmt.Errorf("invalid trade_date: %w", err)
	}

	if strings.TrimSpace(params.Prompt) == "" {
		params.Prompt = fmt.Sprintf("Analyze trading opportunities for %s on %s", params.Symbol, params.TradeDate)
	}

	cfg := config.Get()
	if cfg.DeepSeekAPIKey == "" {
		return nil, fmt.Errorf("deepseek api key is required")
	}

	ctx := context.Background()
	if err := agents.InitChatModel(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("init chat model: %w", err)
	}

	store, err := storage.GetSQLiteStore()
	if err != nil {
		return nil, fmt.Errorf("init sqlite: %w", err)
	}
	sessionRec := &models.SessionRecord{
		Symbol:    params.Symbol,
		TradeDate: params.TradeDate,
		Prompt:    params.Prompt,
		Status:    storage.StatusInit,
	}
	if _, err := store.CreateSession(ctx, sessionRec); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	if err := store.SaveMessage(ctx, &models.MessageRecord{
		SessionId: sessionRec.Id,
		Role:      "user",
		Content:   params.Prompt,
		Status:    storage.StatusDone,
	}); err != nil {
		return nil, fmt.Errorf("record user prompt: %w", err)
	}

	genFunc := func(ctx context.Context) *models.TradingState {
		return models.NewTradingState(params.Symbol, parsedDate, params.Prompt, &cfg)
	}

	orchestrator := graph.NewTradingOrchestrator[string, string, *models.TradingState](ctx, genFunc, &cfg)
	sessionID := sessionRec.Id
	sessionIDStr := strconv.FormatInt(sessionID, 10)
	persistStreamEvent := func(event string, data *models.ChatResp) {
		if data == nil {
			return
		}
		var (
			role         string
			content      string
			status       = storage.StatusDone
			finishReason string
			toolCallID   string
			toolName     string
			toolCalls    string
		)
		if len(data.ToolCalls) > 0 {
			if encoded, err := json.Marshal(data.ToolCalls); err == nil {
				toolCalls = string(encoded)
			} else {
				fmt.Printf("marshal tool calls err=%v\n", err)
			}
		}
		switch event {
		case "text_final":
			role = "assistant"
			content = data.Content
		case "tool_call_result_final":
			role = data.Role
			if strings.TrimSpace(role) == "" {
				role = "tool"
			}
			content = data.Content
			toolCallID = data.ToolCallId
			toolName = data.ToolName
		case "error":
			role = "system"
			content = data.Content
			status = storage.StatusError
		default:
			return
		}
		msg := &models.MessageRecord{
			SessionId:    sessionID,
			Role:         role,
			Agent:        data.AgentName,
			Content:      content,
			Status:       status,
			ToolCalls:    toolCalls,
			ToolCallId:   toolCallID,
			ToolName:     toolName,
			FinishReason: finishReason,
		}
		if err := store.SaveMessage(ctx, msg); err != nil {
			fmt.Printf("persist stream message event=%s err=%v\n", event, err)
		}
	}

	go func() {
		_, streamErr := orchestrator.Stream(ctx, params.Prompt,
			compose.WithCallbacks(&graph.LoggerCallback{
				Emit: func(event string, data *models.ChatResp) {
					persistStreamEvent(event, data)
					if data == nil {
						return
					}
					payload, _ := json.Marshal(data)
					bridge.Notify("agent."+event, string(payload))
				},
			}),
		)
		status := storage.StatusDone
		if streamErr != nil {
			status = storage.StatusError
			if err := store.SaveMessage(ctx, &models.MessageRecord{
				SessionId: sessionID,
				Role:      "system",
				Content:   streamErr.Error(),
				Status:    storage.StatusError,
			}); err != nil {
				fmt.Printf("record stream error message err=%v\n", err)
			}
		}
		if err := store.UpdateSessionStatus(ctx, sessionID, status); err != nil {
			fmt.Printf("update session status err=%v\n", err)
		}

		if streamErr != nil {
			errPayload, _ := json.Marshal(map[string]string{"error": streamErr.Error()})
			bridge.Notify("agent.error", string(errPayload))
			return
		}

		bridge.Notify("agent.finished", `{"status":"completed"}`)
	}()

	return map[string]string{"status": "started", "session_id": sessionIDStr}, nil
}
