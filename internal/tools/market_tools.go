package tools

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	t_utils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/cache"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/pkg/dataflows"
	"github.com/longportapp/openapi-go/quote"
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

			// 首先检查缓存
			cacheManager := cache.GetMarketDataCache()
			if cachedData, found := cacheManager.Get(ctx, input.Symbol, count); found {
				log.Printf("Using cached market data for %s (count: %d)", input.Symbol, count)
				return &models.MarketDataOutput{Data: cachedData}, nil
			}

			// 缓存未命中，获取真实数据
			longportConf := dataflows.LongportConfig{
				AppKey:      cfg.LongportAppKey,
				AppSecret:   cfg.LongportAppSecret,
				AccessToken: cfg.LongportAccessToken,
			}
			longportClient, err := dataflows.NewLongportClient(longportConf)
			if err != nil {
				log.Printf("Failed to create Longport client, using mock data: %v", err)
				return getMockMarketData(input.Symbol), nil
			}

			// Try to get real market data from Longport
			sticks, err := longportClient.GetSticksWithDay(ctx, input.Symbol, count)
			if err == nil && len(sticks) > 0 {
				marketData := convertSticksToMarketData(sticks, input.Symbol)

				// 缓存数据
				cacheManager.Set(ctx, input.Symbol, count, marketData)
				log.Printf("Fetched and cached market data for %s (count: %d)", input.Symbol, count)

				return &models.MarketDataOutput{Data: marketData}, nil
			}
			log.Printf("Failed to get real market data for %s: %v", input.Symbol, err)

			return getMockMarketData(input.Symbol), nil
		},
	)
}

// bestIndParams contains descriptions for all supported technical indicators
var bestIndParams = map[string]string{
	"close_50_sma":  "50 SMA: A medium-term trend indicator. Usage: Identify trend direction and serve as dynamic support/resistance. Tips: It lags price; combine with faster indicators for timely signals.",
	"close_200_sma": "200 SMA: A long-term trend benchmark. Usage: Confirm overall market trend and identify golden/death cross setups. Tips: It reacts slowly; best for strategic trend confirmation rather than frequent trading entries.",
	"close_10_ema":  "10 EMA: A responsive short-term average. Usage: Capture quick shifts in momentum and potential entry points. Tips: Prone to noise in choppy markets; use alongside longer averages for filtering false signals.",
	"vwma":          "VWMA: A moving average weighted by volume. Usage: Confirm trends by integrating price action with volume data. Tips: Watch for skewed results from volume spikes; use in combination with other volume analyses.",
	"macd":          "MACD: Computes momentum via differences of EMAs. Usage: Look for crossovers and divergence as signals of trend changes. Tips: Confirm with other indicators in low-volatility or sideways markets.",
	"macds":         "MACD Signal: An EMA smoothing of the MACD line. Usage: Use crossovers with the MACD line to trigger trades. Tips: Should be part of a broader strategy to avoid false positives.",
	"macdh":         "MACD Histogram: Shows the gap between the MACD line and its signal. Usage: Visualize momentum strength and spot divergence early. Tips: Can be volatile; complement with additional filters in fast-moving markets.",
	"rsi":           "RSI: Measures momentum to flag overbought/oversold conditions. Usage: Apply 70/30 thresholds and watch for divergence to signal reversals. Tips: In strong trends, RSI may remain extreme; always cross-check with trend analysis.",
	"mfi":           "MFI: The Money Flow Index is a momentum indicator that uses both price and volume to measure buying and selling pressure. Usage: Identify overbought (>80) or oversold (<20) conditions and confirm the strength of trends or reversals. Tips: Use alongside RSI or MACD to confirm signals; divergence between price and MFI can indicate potential reversals.",
	"boll":          "Bollinger Middle: A 20 SMA serving as the basis for Bollinger Bands. Usage: Acts as a dynamic benchmark for price movement. Tips: Combine with the upper and lower bands to effectively spot breakouts or reversals.",
	"boll_ub":       "Bollinger Upper Band: Typically 2 standard deviations above the middle line. Usage: Signals potential overbought conditions and breakout zones. Tips: Confirm signals with other tools; prices may ride the band in strong trends.",
	"boll_lb":       "Bollinger Lower Band: Typically 2 standard deviations below the middle line. Usage: Indicates potential oversold conditions. Tips: Use additional analysis to avoid false reversal signals.",
	"atr":           "ATR: Averages true range to measure volatility. Usage: Set stop-loss levels and adjust position sizes based on current market volatility. Tips: It's a reactive measure, so use it as part of a broader risk management strategy.",
}

// NewStockIndicatorTool creates a new technical indicator analysis tool
func NewStockIndicatorTool(cfg *config.Config) tool.BaseTool {
	return t_utils.NewTool[models.StockIndicatorInput, *models.StockIndicatorOutput](
		&schema.ToolInfo{
			Name: "get_stock_stats_indicators_window",
			Desc: "Get comprehensive technical indicator analysis for a stock with all major indicators calculated at once",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"symbol": {
					Type:     "string",
					Desc:     "Ticker symbol of the company",
					Required: true,
				},
				"curr_date": {
					Type:     "string",
					Desc:     "The current trading date you are trading on, YYYY-mm-dd",
					Required: true,
				},
				"look_back_days": {
					Type:     "integer",
					Desc:     "How many days to look back for analysis",
					Required: true,
				},
				"online": {
					Type:     "boolean",
					Desc:     "To fetch data online or offline",
					Required: false,
				},
			}),
		},
		func(ctx context.Context, input models.StockIndicatorInput) (*models.StockIndicatorOutput, error) {
			log.Printf("Stock indicator tool called with input: %+v", input)

			// Parse current date
			currDate, err := time.Parse("2006-01-02", input.CurrDate)
			if err != nil {
				return nil, fmt.Errorf("invalid date format: %s", input.CurrDate)
			}

			// Calculate start date
			startDate := currDate.AddDate(0, 0, -input.LookBackDays)

			// Get market data once with sufficient buffer for all indicators
			// 200 SMA needs 200 days, MACD Signal needs 26+9=35 days, so we use 250 as buffer
			bufferDays := 250
			if input.LookBackDays < 30 {
				bufferDays = 300 // More buffer for short look-back periods
			}

			var marketData []*models.MarketData
			marketData, err = getOnlineMarketDataForIndicator(ctx, cfg, input.Symbol, input.LookBackDays+bufferDays)
			if err != nil {
				return nil, fmt.Errorf("failed to get market data: %v", err)
			}

			if len(marketData) == 0 {
				return nil, fmt.Errorf("no market data available for symbol %s", input.Symbol)
			}

			log.Printf("Retrieved %d market data points for %s (requested %d + %d buffer)",
				len(marketData), input.Symbol, input.LookBackDays, bufferDays)

			// Calculate all indicators at once
			allIndicators := dataflows.CalculateAllIndicators(marketData, startDate, currDate)

			// 保存指标结果到CSV
			cacheManager := cache.GetMarketDataCache()
			go func() {
				if err := cacheManager.SaveIndicatorsToCSV(input.Symbol, allIndicators); err != nil {
					log.Printf("Failed to save indicators to CSV: %v", err)
				} else {
					log.Printf("Successfully saved indicators to CSV for %s", input.Symbol)
				}
			}()

			// Format comprehensive result
			var resultBuilder strings.Builder
			resultBuilder.WriteString(fmt.Sprintf("# Technical Analysis for %s from %s to %s\n\n",
				input.Symbol, startDate.Format("2006-01-02"), input.CurrDate))

			// Group indicators by category
			categories := map[string][]string{
				"Moving Averages":       {"close_10_ema", "close_50_sma", "close_200_sma", "vwma"},
				"Momentum Indicators":   {"rsi", "macd", "macds", "macdh", "mfi"},
				"Volatility Indicators": {"boll", "boll_ub", "boll_lb", "atr"},
			}

			for category, indicators := range categories {
				resultBuilder.WriteString(fmt.Sprintf("## %s\n\n", category))

				categoryHasData := false
				for _, indicator := range indicators {
					if values, exists := allIndicators[indicator]; exists && len(values) > 0 {
						categoryHasData = true
						// Show latest value and description
						latestValue := values[len(values)-1]
						resultBuilder.WriteString(fmt.Sprintf("### %s\n", indicator))
						resultBuilder.WriteString(fmt.Sprintf("**Latest Value (%s):** %.4f\n\n", latestValue.Date, latestValue.Value))

						// Show recent 5 values
						resultBuilder.WriteString("**Recent Values:**\n")
						start := len(values) - 5
						if start < 0 {
							start = 0
						}
						for i := start; i < len(values); i++ {
							resultBuilder.WriteString(fmt.Sprintf("- %s: %.4f\n", values[i].Date, values[i].Value))
						}

						// Add description
						if desc, exists := bestIndParams[indicator]; exists {
							resultBuilder.WriteString(fmt.Sprintf("\n*%s*\n\n", desc))
						}
					} else {
						// Indicator failed to calculate
						resultBuilder.WriteString(fmt.Sprintf("### %s\n", indicator))
						resultBuilder.WriteString("**Status:** Data insufficient for calculation\n\n")
						if desc, exists := bestIndParams[indicator]; exists {
							resultBuilder.WriteString(fmt.Sprintf("*%s*\n\n", desc))
						}
					}
				}

				if !categoryHasData {
					resultBuilder.WriteString("*No indicators in this category could be calculated with the available data.*\n\n")
				}
			}

			// Add summary section
			resultBuilder.WriteString("## Summary\n\n")
			resultBuilder.WriteString(generateTechnicalSummary(allIndicators))

			return &models.StockIndicatorOutput{
				Result: resultBuilder.String(),
			}, nil
		},
	)
}

// getOnlineMarketDataForIndicator fetches market data online for indicator calculations with caching
func getOnlineMarketDataForIndicator(ctx context.Context, cfg *config.Config, symbol string, count int) ([]*models.MarketData, error) {
	// 首先检查缓存
	cacheManager := cache.GetMarketDataCache()
	if cachedData, found := cacheManager.Get(ctx, symbol, count); found {
		log.Printf("Using cached market data for indicators %s (count: %d)", symbol, count)
		return cachedData, nil
	}
	longportConf := dataflows.LongportConfig{
		AppKey:      cfg.LongportAppKey,
		AppSecret:   cfg.LongportAppSecret,
		AccessToken: cfg.LongportAccessToken,
	}
	longportClient, err := dataflows.NewLongportClient(longportConf)
	if err != nil {
		return nil, err
	}

	sticks, err := longportClient.GetSticksWithDay(ctx, symbol, count)
	if err != nil {
		return nil, err
	}

	marketData := convertSticksToMarketData(sticks, symbol)

	// 缓存数据
	cacheManager.Set(ctx, symbol, count, marketData)
	log.Printf("Fetched and cached market data for indicators %s (count: %d)", symbol, count)

	return marketData, nil
}

// generateTechnicalSummary generates a summary of technical analysis
func generateTechnicalSummary(indicators map[string][]models.IndicatorValue) string {
	var summary strings.Builder

	// Get latest values for analysis
	getLatestValue := func(indicator string) (float64, bool) {
		if values, exists := indicators[indicator]; exists && len(values) > 0 {
			return values[len(values)-1].Value, true
		}
		return 0, false
	}

	// Trend Analysis
	summary.WriteString("**Trend Analysis:**\n")
	if ema10, exists1 := getLatestValue("close_10_ema"); exists1 {
		if sma50, exists2 := getLatestValue("close_50_sma"); exists2 {
			if ema10 > sma50 {
				summary.WriteString("- Short-term trend is BULLISH (10 EMA > 50 SMA)\n")
			} else {
				summary.WriteString("- Short-term trend is BEARISH (10 EMA < 50 SMA)\n")
			}
		}
	}

	if sma50, exists1 := getLatestValue("close_50_sma"); exists1 {
		if sma200, exists2 := getLatestValue("close_200_sma"); exists2 {
			if sma50 > sma200 {
				summary.WriteString("- Long-term trend is BULLISH (50 SMA > 200 SMA)\n")
			} else {
				summary.WriteString("- Long-term trend is BEARISH (50 SMA < 200 SMA)\n")
			}
		}
	}

	summary.WriteString("\n")

	// Momentum Analysis
	summary.WriteString("**Momentum Analysis:**\n")
	if rsi, exists := getLatestValue("rsi"); exists {
		if rsi > 70 {
			summary.WriteString(fmt.Sprintf("- RSI (%.2f) indicates OVERBOUGHT conditions\n", rsi))
		} else if rsi < 30 {
			summary.WriteString(fmt.Sprintf("- RSI (%.2f) indicates OVERSOLD conditions\n", rsi))
		} else {
			summary.WriteString(fmt.Sprintf("- RSI (%.2f) is in NEUTRAL range\n", rsi))
		}
	}

	if mfi, exists := getLatestValue("mfi"); exists {
		if mfi > 80 {
			summary.WriteString(fmt.Sprintf("- MFI (%.2f) indicates OVERBOUGHT with strong selling pressure\n", mfi))
		} else if mfi < 20 {
			summary.WriteString(fmt.Sprintf("- MFI (%.2f) indicates OVERSOLD with strong buying pressure\n", mfi))
		} else {
			summary.WriteString(fmt.Sprintf("- MFI (%.2f) shows balanced money flow\n", mfi))
		}
	}

	summary.WriteString("\n")

	// Volatility Analysis
	summary.WriteString("**Volatility Analysis:**\n")
	if bollMiddle, exists1 := getLatestValue("boll"); exists1 {
		if bollUpper, exists2 := getLatestValue("boll_ub"); exists2 {
			if bollLower, exists3 := getLatestValue("boll_lb"); exists3 {
				bandWidth := (bollUpper - bollLower) / bollMiddle * 100
				summary.WriteString(fmt.Sprintf("- Bollinger Band width: %.2f%% ", bandWidth))
				if bandWidth > 20 {
					summary.WriteString("(HIGH volatility)\n")
				} else if bandWidth < 10 {
					summary.WriteString("(LOW volatility)\n")
				} else {
					summary.WriteString("(MODERATE volatility)\n")
				}
			}
		}
	}

	if atr, exists := getLatestValue("atr"); exists {
		summary.WriteString(fmt.Sprintf("- Average True Range: %.4f (daily volatility measure)\n", atr))
	}

	summary.WriteString("\n")

	// MACD Analysis
	summary.WriteString("**MACD Analysis:**\n")
	if macd, exists1 := getLatestValue("macd"); exists1 {
		if macdSignal, exists2 := getLatestValue("macds"); exists2 {
			if macd > macdSignal {
				summary.WriteString("- MACD line above signal line: BULLISH momentum\n")
			} else {
				summary.WriteString("- MACD line below signal line: BEARISH momentum\n")
			}
		}
	}

	if macdHist, exists := getLatestValue("macdh"); exists {
		if macdHist > 0 {
			summary.WriteString("- MACD Histogram positive: Increasing bullish momentum\n")
		} else {
			summary.WriteString("- MACD Histogram negative: Increasing bearish momentum\n")
		}
	}
	return summary.String()
}

// convertSticksToMarketData 提取公共的数据转换函数
func convertSticksToMarketData(sticks []*quote.Candlestick, symbol string) []*models.MarketData {
	marketData := make([]*models.MarketData, 0, len(sticks))
	for _, stick := range sticks {
		date := time.Unix(stick.Timestamp, 0).Format("2006-01-02")
		open, _ := stick.Open.Float64()
		high, _ := stick.High.Float64()
		low, _ := stick.Low.Float64()
		close, _ := stick.Close.Float64()

		marketData = append(marketData, &models.MarketData{
			Symbol: symbol,
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: stick.Volume,
		})
	}
	return marketData
}

// getMockMarketData 获取模拟市场数据
func getMockMarketData(symbol string) *models.MarketDataOutput {
	return &models.MarketDataOutput{
		Data: []*models.MarketData{
			{
				Symbol: symbol,
				Date:   time.Now().Format("2006-01-02"),
				Open:   100.0,
				High:   101.0,
				Low:    99.0,
				Close:  100.5,
				Volume: int64(1000000),
			},
		},
	}
}
