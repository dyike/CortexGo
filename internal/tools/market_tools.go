package tools

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino/components/tool"
	t_utils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/dataflows"
	"github.com/dyike/CortexGo/internal/models"
)

// createMarketDataTool creates the market data tool using proper generic types
func NewMarketool(cfg *config.Config) tool.BaseTool {
	return t_utils.NewTool(
		&schema.ToolInfo{
			Name: "get_market_data",
			Desc: "Get market data for a specific symbol and date range",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"symbol": {
					Type:     "string",
					Desc:     "The stock symbol",
					Required: true,
				},
				"count": {
					Type:     "integer",
					Desc:     "Number of days to retrieve (default: 30)",
					Required: false,
				},
			}),
		},
		func(ctx context.Context, input models.MarketDataInput) (*models.MarketDataOutput, error) {
			if input.Symbol == "" {
				return nil, fmt.Errorf("symbol parameter is required")
			}

			count := input.Count
			if count <= 0 {
				count = 30 // default
			}

			// Create Longport client for real market data
			longportClient, err := dataflows.NewLongportClient(cfg)
			if err != nil {
				log.Printf("Failed to create Longport client, using mock data: %v", err)
				longportClient = nil
			}

			// Try to get real market data from Longport
			if longportClient != nil {
				sticks, err := longportClient.GetSticksWithDay(ctx, input.Symbol, count)
				if err == nil && len(sticks) > 0 {
					marketData := make([]*models.MarketData, 0, len(sticks))
					for _, stick := range sticks {
						// Convert Unix timestamp to date string
						date := time.Unix(stick.Timestamp, 0).Format("2006-01-02")
						// Convert decimal values to float64
						open, _ := stick.Open.Float64()
						high, _ := stick.High.Float64()
						low, _ := stick.Low.Float64()
						close, _ := stick.Close.Float64()
						marketData = append(marketData, &models.MarketData{
							Symbol: input.Symbol,
							Date:   date,
							Open:   open,
							High:   high,
							Low:    low,
							Close:  close,
							Volume: stick.Volume,
						})
					}
					// log.Printf("Successfully retrieved %d market data records for %s", len(marketData), input.Symbol)
					return &models.MarketDataOutput{Data: marketData}, nil
				}
				log.Printf("Failed to get real market data for %s: %v", input.Symbol, err)
			}

			return &models.MarketDataOutput{
				Data: []*models.MarketData{
					{
						Symbol: input.Symbol,
						Date:   time.Now().Format("2006-01-02"),
						Open:   100.0,
						High:   101.0,
						Low:    99.0,
						Close:  100.5,
						Volume: int64(1000000),
					},
				},
			}, nil
		},
	)
}
