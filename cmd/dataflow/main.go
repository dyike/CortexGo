package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/dataflows"
)

func main() {
	ctx := context.Background()
	// Init eino devops server
	cfg := config.DefaultConfig()

	symbol := "UI.US"

	longbridge, err := dataflows.NewLongportClient(cfg)
	if err != nil {
		panic(err)
	}

	infos, err := longbridge.GetStaticInfo(ctx, []string{symbol})
	if err != nil {
		panic(err)
	}

	payload, _ := json.Marshal(infos)
	fmt.Println(string(payload))
}
