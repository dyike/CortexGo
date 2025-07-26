package main

import (
	"fmt"
	"log"

	"github.com/dyike/CortexGo/pkg/config"
	"github.com/dyike/CortexGo/pkg/graph"
)

func main() {
	cfg := config.DefaultConfig()

	tradingGraph := graph.NewTradingAgentsGraph(true, cfg)

	state, decision, err := tradingGraph.Propagate("AAPL", "2024-01-15")
	if err != nil {
		log.Fatalf("Trading analysis failed: %v", err)
	}

	fmt.Printf("\n=== ANALYST TEAM RESULTS ===\n")
	fmt.Printf("Symbol: %s\n", state.CurrentSymbol)
	fmt.Printf("Number of Reports: %d\n", len(state.Reports))

	for _, report := range state.Reports {
		fmt.Printf("\n--- %s Analysis ---\n", report.Analyst)
		fmt.Printf("Rating: %s (Confidence: %.2f)\n", report.Rating, report.Confidence)
		fmt.Printf("Priority: %d\n", report.Priority)
		fmt.Printf("Analysis: %s\n", report.Analysis)
		if len(report.KeyFindings) > 0 {
			fmt.Printf("Key Findings: %v\n", report.KeyFindings)
		}
		if len(report.Concerns) > 0 {
			fmt.Printf("Concerns: %v\n", report.Concerns)
		}
	}

	if state.TeamConsensus != nil {
		fmt.Printf("\n=== TEAM CONSENSUS ===\n")
		fmt.Printf("Final Rating: %s\n", state.TeamConsensus.FinalRating)
		fmt.Printf("Agreement Level: %.2f\n", state.TeamConsensus.AgreementLevel)
		fmt.Printf("Confidence: %.2f\n", state.TeamConsensus.Confidence)
		if len(state.TeamConsensus.MainArguments) > 0 {
			fmt.Printf("Main Arguments: %v\n", state.TeamConsensus.MainArguments)
		}
		if len(state.TeamConsensus.Dissents) > 0 {
			fmt.Printf("Dissents: %v\n", state.TeamConsensus.Dissents)
		}
	}

	if len(state.Discussions) > 0 {
		fmt.Printf("\n=== TEAM DISCUSSIONS ===\n")
		for _, discussion := range state.Discussions {
			fmt.Printf("Topic: %s\n", discussion.Topic)
			fmt.Printf("Participants: %v\n", discussion.Participants)
			fmt.Printf("Debate Points: %d\n", len(discussion.DebatePoints))
			for _, point := range discussion.DebatePoints {
				fmt.Printf("  - %s: %s (%s)\n", point.Analyst, point.Position, point.Response)
			}
		}
	}

	fmt.Printf("\n=== FINAL DECISION ===\n")
	if decision != nil {
		fmt.Printf("Action: %s\n", decision.Action)
		fmt.Printf("Confidence: %.2f\n", decision.Confidence)
		fmt.Printf("Risk: %.2f\n", decision.Risk)
		fmt.Printf("Reason: %s\n", decision.Reason)
	}
}
