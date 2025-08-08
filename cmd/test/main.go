package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/eino-ext/devops"
	"github.com/cloudwego/eino/compose"
	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/graph"
	"github.com/dyike/CortexGo/internal/models"
)

func main() {
	ctx := context.Background()
	// Init eino devops server
	err := devops.Init(ctx)
	if err != nil {
		return
	}
	cfg := config.DefaultConfig()

	symbol := "UI"
	tradeDate := "2025-08-06"

	parsedDate, err := time.Parse("2006-01-02", tradeDate)
	if err != nil {
		panic(err)
	}
	userPrompt := fmt.Sprintf("Analyze trading opportunities for %s on %s", symbol, tradeDate)

	// Add logger callback
	outChan := make(chan string)
	go func() {
		for out := range outChan {
			fmt.Print(out)
		}
	}()

	genFunc := func(ctx context.Context) *models.TradingState {
		state := models.NewTradingState(symbol, parsedDate, userPrompt, cfg)
		return state
	}

	to := graph.NewTradingOrchestrator[string, string, *models.TradingState](ctx, genFunc, cfg)
	_, err = to.Stream(ctx, "Analyze trading opportunities for UI on 2025-08-06",
		compose.WithCallbacks(&graph.LoggerCallback{
			Out: outChan,
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
