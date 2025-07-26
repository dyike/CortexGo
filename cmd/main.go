package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
)

func main() {
	err := agents.InitModel()
	if err != nil {
		fmt.Printf("Failed to initialize model: %v\n", err)
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "-s" {
		runServer()
		return
	}

	runConsole()
}

func runConsole() {
	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("请输入交易符号 (例如: AAPL): ")
	symbol, _ := reader.ReadString('\n')
	symbol = strings.TrimSpace(symbol)

	fmt.Print("请输入您的交易需求: ")
	userPrompt, _ := reader.ReadString('\n')
	userPrompt = strings.TrimSpace(userPrompt)

	genFunc := func(ctx context.Context) *agents.TradingState {
		return agents.NewTradingState(symbol, time.Now(), userPrompt)
	}

	orchestrator := agents.NewTradingOrchestrator[string, string, *agents.TradingState](ctx, genFunc)

	fmt.Printf("\n开始处理 %s 的交易分析...\n\n", symbol)

	result, err := orchestrator.Invoke(ctx, consts.Coordinator)
	if err != nil {
		fmt.Printf("执行失败: %v\n", err)
		return
	}

	fmt.Printf("交易分析完成: %s\n", result)
}

func runServer() {
	fmt.Println("Server mode not implemented yet")
}
