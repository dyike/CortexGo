package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/eino-ext/devops"
	"github.com/cloudwego/eino/compose"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/graph"
	"github.com/dyike/CortexGo/models"
)

func main() {
	ctx := context.Background()
	// Init eino devops server
	err := devops.Init(ctx)
	if err != nil {
		return
	}
	cfg := config.DefaultConfig()

	if err := agents.InitChatModel(ctx, cfg); err != nil {
		panic(err)
	}

	symbol := "CRCL.US"
	tradeDate := "2025-12-12"

	parsedDate, err := time.Parse("2006-01-02", tradeDate)
	if err != nil {
		panic(err)
	}
	userPrompt := fmt.Sprintf("Analyze trading opportunities for %s on %s", symbol, tradeDate)

	genFunc := func(ctx context.Context) *models.TradingState {
		state := models.NewTradingState(symbol, parsedDate, userPrompt, cfg)
		return state
	}

	to := graph.NewTradingOrchestrator[string, string, *models.TradingState](ctx, genFunc, cfg)
	_, err = to.Stream(ctx, "Analyze trading opportunities for UI on 2025-09-23",
		compose.WithCallbacks(&graph.LoggerCallback{
			Emit: func(event string, data *models.ChatResp) {
				if data == nil {
					fmt.Printf("[event=%s] <nil>\n", event)
					return
				}
				payload, _ := json.Marshal(data)
				fmt.Printf("[event=%s] %s\n", event, string(payload))
			},
		}),
	)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Blocking process exits
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}
