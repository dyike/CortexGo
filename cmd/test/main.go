package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudwego/eino-ext/devops"
	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/graph"
)

func main() {
	ctx := context.Background()
	// Init eino devops server
	err := devops.Init(ctx)
	if err != nil {
		return
	}

	cfg := config.DefaultConfig()
	tag := graph.NewTradingAgentsGraph(true, cfg)
	if tag != nil {
		res, _, _ := tag.Propagate("UI", "2025-08-06")
		fmt.Println(res)
	}

	// Blocking process exits
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}
