package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/graph"
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

	genFunc := func(ctx context.Context) *models.TradingState {
		return models.NewTradingState(params.Symbol, parsedDate, params.Prompt, &cfg)
	}

	orchestrator := graph.NewTradingOrchestrator[string, string, *models.TradingState](ctx, genFunc, &cfg)

	go func() {
		_, streamErr := orchestrator.Stream(ctx, params.Prompt,
			compose.WithCallbacks(&graph.LoggerCallback{
				Emit: func(event string, data *models.ChatResp) {
					if data == nil {
						return
					}
					payload, _ := json.Marshal(data)
					bridge.Notify("agent."+event, string(payload))
				},
			}),
		)

		if streamErr != nil {
			errPayload, _ := json.Marshal(map[string]string{"error": streamErr.Error()})
			bridge.Notify("agent.error", string(errPayload))
			return
		}

		bridge.Notify("agent.finished", `{"status":"completed"}`)
	}()

	return map[string]string{"status": "started"}, nil
}
