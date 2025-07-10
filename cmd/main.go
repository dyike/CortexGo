package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/dyike/CortexGo/pkg/config"
	"github.com/dyike/CortexGo/pkg/graph"
)

func main() {
	cfg := config.DefaultConfig()
	cfg.Debug = true
	
	if err := cfg.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to create directories: %v", err)
	}

	ta := graph.NewTradingAgentsGraph(true, cfg)

	fmt.Println("Starting trading analysis for NVDA on 2024-05-10...")
	
	state, decision, err := ta.Propagate("NVDA", "2024-05-10")
	if err != nil {
		log.Fatalf("Trading analysis failed: %v", err)
	}

	fmt.Println("\n=== TRADING DECISION ===")
	decisionJSON, err := json.MarshalIndent(decision, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal decision: %v", err)
	}
	fmt.Println(string(decisionJSON))

	fmt.Println("\n=== ANALYSIS REPORTS ===")
	for i, report := range state.Reports {
		fmt.Printf("\nReport %d (%s):\n", i+1, report.Analyst)
		fmt.Printf("Rating: %s\n", report.Rating)
		fmt.Printf("Confidence: %.2f\n", report.Confidence)
		fmt.Printf("Analysis: %s\n", report.Analysis)
	}

	if err := ta.ReflectAndRemember(1000.0); err != nil {
		log.Printf("Reflection failed: %v", err)
	}

	fmt.Println("\nTrading analysis completed successfully!")
}