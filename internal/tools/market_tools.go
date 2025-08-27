package tools

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
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
			if input.Online {
				marketData, err = getOnlineMarketDataForIndicator(ctx, cfg, input.Symbol, input.LookBackDays+bufferDays)
			} else {
				marketData, err = getOfflineMarketDataForIndicator(ctx, cfg, input.Symbol, startDate, currDate)
			}

			if err != nil {
				return nil, fmt.Errorf("failed to get market data: %v", err)
			}

			if len(marketData) == 0 {
				return nil, fmt.Errorf("no market data available for symbol %s", input.Symbol)
			}

			log.Printf("Retrieved %d market data points for %s (requested %d + %d buffer)", 
				len(marketData), input.Symbol, input.LookBackDays, bufferDays)

			// Calculate all indicators at once
			allIndicators := calculateAllIndicators(marketData, startDate, currDate)

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

// getOnlineMarketDataForIndicator fetches market data online for indicator calculations
func getOnlineMarketDataForIndicator(ctx context.Context, cfg *config.Config, symbol string, count int) ([]*models.MarketData, error) {
	longportClient, err := dataflows.NewLongportClient(cfg)
	if err != nil {
		return nil, err
	}

	sticks, err := longportClient.GetSticksWithDay(ctx, symbol, count)
	if err != nil {
		return nil, err
	}

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

	return marketData, nil
}

// getOfflineMarketDataForIndicator fetches market data offline (placeholder implementation)
func getOfflineMarketDataForIndicator(ctx context.Context, cfg *config.Config, symbol string, startDate, endDate time.Time) ([]*models.MarketData, error) {
	// For now, fallback to online data
	// You can implement CSV reading logic here if needed
	return getOnlineMarketDataForIndicator(ctx, cfg, symbol, int(endDate.Sub(startDate).Hours()/24)+30)
}

// calculateAllIndicators calculates all technical indicators at once
func calculateAllIndicators(data []*models.MarketData, startDate, endDate time.Time) map[string][]models.IndicatorValue {
	if len(data) == 0 {
		return make(map[string][]models.IndicatorValue)
	}

	// Sort data by date once
	sort.Slice(data, func(i, j int) bool {
		return data[i].Date < data[j].Date
	})

	results := make(map[string][]models.IndicatorValue)

	// Calculate all indicators
	indicators := map[string]func() ([]models.IndicatorValue, error){
		"close_10_ema":  func() ([]models.IndicatorValue, error) { return calculateEMA(data, 10, startDate, endDate) },
		"close_50_sma":  func() ([]models.IndicatorValue, error) { return calculateSMA(data, 50, startDate, endDate) },
		"close_200_sma": func() ([]models.IndicatorValue, error) { return calculateSMA(data, 200, startDate, endDate) },
		"vwma":          func() ([]models.IndicatorValue, error) { return calculateVWMA(data, 20, startDate, endDate) },
		"rsi":           func() ([]models.IndicatorValue, error) { return calculateRSI(data, 14, startDate, endDate) },
		"macd":          func() ([]models.IndicatorValue, error) { return calculateMACD(data, startDate, endDate) },
		"macds":         func() ([]models.IndicatorValue, error) { return calculateMACDSignal(data, startDate, endDate) },
		"macdh":         func() ([]models.IndicatorValue, error) { return calculateMACDHistogram(data, startDate, endDate) },
		"mfi":           func() ([]models.IndicatorValue, error) { return calculateMFI(data, 14, startDate, endDate) },
		"boll":          func() ([]models.IndicatorValue, error) { return calculateBollingerMiddle(data, 20, startDate, endDate) },
		"boll_ub": func() ([]models.IndicatorValue, error) {
			return calculateBollingerUpper(data, 20, 2, startDate, endDate)
		},
		"boll_lb": func() ([]models.IndicatorValue, error) {
			return calculateBollingerLower(data, 20, 2, startDate, endDate)
		},
		"atr": func() ([]models.IndicatorValue, error) { return calculateATR(data, 14, startDate, endDate) },
	}

	// Calculate all indicators
	successCount := 0
	totalCount := len(indicators)
	
	for name, calcFunc := range indicators {
		if values, err := calcFunc(); err == nil {
			results[name] = values
			successCount++
		} else {
			log.Printf("Failed to calculate %s: %v", name, err)
		}
	}

	log.Printf("Successfully calculated %d/%d indicators", successCount, totalCount)
	return results
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

// calculateSMA calculates Simple Moving Average
func calculateSMA(data []*models.MarketData, period int, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	var result []models.IndicatorValue

	for i := period - 1; i < len(data); i++ {
		currentDate, _ := time.Parse("2006-01-02", data[i].Date)
		if currentDate.Before(startDate) || currentDate.After(endDate) {
			continue
		}

		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += data[j].Close
		}
		avg := sum / float64(period)

		result = append(result, models.IndicatorValue{
			Date:  data[i].Date,
			Value: avg,
		})
	}

	return result, nil
}

// calculateEMA calculates Exponential Moving Average
func calculateEMA(data []*models.MarketData, period int, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	if len(data) < period {
		return nil, fmt.Errorf("insufficient data for EMA calculation")
	}

	multiplier := 2.0 / (float64(period) + 1.0)
	var result []models.IndicatorValue

	// Calculate initial SMA for first EMA value
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += data[i].Close
	}
	ema := sum / float64(period)

	for i := period - 1; i < len(data); i++ {
		currentDate, _ := time.Parse("2006-01-02", data[i].Date)

		if i > period-1 {
			ema = (data[i].Close * multiplier) + (ema * (1 - multiplier))
		}

		if !currentDate.Before(startDate) && !currentDate.After(endDate) {
			result = append(result, models.IndicatorValue{
				Date:  data[i].Date,
				Value: ema,
			})
		}
	}

	return result, nil
}

// calculateRSI calculates Relative Strength Index
func calculateRSI(data []*models.MarketData, period int, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	if len(data) < period+1 {
		return nil, fmt.Errorf("insufficient data for RSI calculation")
	}

	var result []models.IndicatorValue
	var gains, losses []float64

	// Calculate initial gains and losses
	for i := 1; i <= period; i++ {
		change := data[i].Close - data[i-1].Close
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	// Calculate initial averages
	avgGain := sum(gains) / float64(period)
	avgLoss := sum(losses) / float64(period)

	for i := period; i < len(data); i++ {
		currentDate, _ := time.Parse("2006-01-02", data[i].Date)

		change := data[i].Close - data[i-1].Close
		var gain, loss float64
		if change > 0 {
			gain = change
			loss = 0
		} else {
			gain = 0
			loss = -change
		}

		// Smoothed averages
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)

		var rsi float64
		if avgLoss == 0 {
			rsi = 100
		} else {
			rs := avgGain / avgLoss
			rsi = 100 - (100 / (1 + rs))
		}

		if !currentDate.Before(startDate) && !currentDate.After(endDate) {
			result = append(result, models.IndicatorValue{
				Date:  data[i].Date,
				Value: rsi,
			})
		}
	}

	return result, nil
}

// calculateMACD calculates MACD line
func calculateMACD(data []*models.MarketData, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	ema12, err := calculateEMAValues(data, 12)
	if err != nil {
		return nil, err
	}

	ema26, err := calculateEMAValues(data, 26)
	if err != nil {
		return nil, err
	}

	var result []models.IndicatorValue
	minLen := min(len(ema12), len(ema26))

	for i := 0; i < minLen; i++ {
		currentDate, _ := time.Parse("2006-01-02", data[25+i].Date) // 26-1 offset
		if !currentDate.Before(startDate) && !currentDate.After(endDate) {
			macd := ema12[i] - ema26[i]
			result = append(result, models.IndicatorValue{
				Date:  data[25+i].Date,
				Value: macd,
			})
		}
	}

	return result, nil
}

// calculateMACDSignal calculates MACD Signal line
func calculateMACDSignal(data []*models.MarketData, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	// Need more data for MACD calculation, so get all MACD values without date filtering
	macdValues, err := calculateMACD(data, time.Time{}, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("failed to calculate MACD for signal: %v", err)
	}

	if len(macdValues) < 9 {
		return nil, fmt.Errorf("insufficient MACD data for signal calculation: need at least 9 values, got %d", len(macdValues))
	}

	// Create temporary data for EMA calculation on MACD
	macdData := make([]*models.MarketData, len(macdValues))
	for i, val := range macdValues {
		macdData[i] = &models.MarketData{
			Date:  val.Date,
			Close: val.Value,
		}
	}

	signalEMA, err := calculateEMAValues(macdData, 9)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate EMA for MACD signal: %v", err)
	}

	var result []models.IndicatorValue
	for i, signal := range signalEMA {
		if i+8 >= len(macdData) {
			break
		}
		currentDate, _ := time.Parse("2006-01-02", macdData[8+i].Date) // 9-1 offset
		if !currentDate.Before(startDate) && !currentDate.After(endDate) {
			result = append(result, models.IndicatorValue{
				Date:  macdData[8+i].Date,
				Value: signal,
			})
		}
	}

	return result, nil
}

// calculateMACDHistogram calculates MACD Histogram
func calculateMACDHistogram(data []*models.MarketData, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	macdValues, err := calculateMACD(data, time.Time{}, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("failed to calculate MACD for histogram: %v", err)
	}

	signalValues, err := calculateMACDSignal(data, time.Time{}, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("failed to calculate MACD signal for histogram: %v", err)
	}

	if len(macdValues) == 0 || len(signalValues) == 0 {
		return nil, fmt.Errorf("insufficient data for MACD histogram: MACD=%d, Signal=%d", len(macdValues), len(signalValues))
	}

	// Create a map for signal values by date for easier lookup
	signalMap := make(map[string]float64)
	for _, sig := range signalValues {
		signalMap[sig.Date] = sig.Value
	}

	var result []models.IndicatorValue
	for _, macdVal := range macdValues {
		if signalVal, exists := signalMap[macdVal.Date]; exists {
			currentDate, _ := time.Parse("2006-01-02", macdVal.Date)
			if !currentDate.Before(startDate) && !currentDate.After(endDate) {
				histogram := macdVal.Value - signalVal
				result = append(result, models.IndicatorValue{
					Date:  macdVal.Date,
					Value: histogram,
				})
			}
		}
	}

	return result, nil
}

// calculateBollingerMiddle calculates Bollinger Band middle line (SMA)
func calculateBollingerMiddle(data []*models.MarketData, period int, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	return calculateSMA(data, period, startDate, endDate)
}

// calculateBollingerUpper calculates Bollinger Band upper line
func calculateBollingerUpper(data []*models.MarketData, period int, multiplier float64, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	var result []models.IndicatorValue

	for i := period - 1; i < len(data); i++ {
		currentDate, _ := time.Parse("2006-01-02", data[i].Date)
		if currentDate.Before(startDate) || currentDate.After(endDate) {
			continue
		}

		// Calculate SMA
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += data[j].Close
		}
		sma := sum / float64(period)

		// Calculate standard deviation
		var variance float64
		for j := i - period + 1; j <= i; j++ {
			diff := data[j].Close - sma
			variance += diff * diff
		}
		variance /= float64(period)
		stdDev := math.Sqrt(variance)

		upperBand := sma + (multiplier * stdDev)
		result = append(result, models.IndicatorValue{
			Date:  data[i].Date,
			Value: upperBand,
		})
	}

	return result, nil
}

// calculateBollingerLower calculates Bollinger Band lower line
func calculateBollingerLower(data []*models.MarketData, period int, multiplier float64, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	var result []models.IndicatorValue

	for i := period - 1; i < len(data); i++ {
		currentDate, _ := time.Parse("2006-01-02", data[i].Date)
		if currentDate.Before(startDate) || currentDate.After(endDate) {
			continue
		}

		// Calculate SMA
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += data[j].Close
		}
		sma := sum / float64(period)

		// Calculate standard deviation
		var variance float64
		for j := i - period + 1; j <= i; j++ {
			diff := data[j].Close - sma
			variance += diff * diff
		}
		variance /= float64(period)
		stdDev := math.Sqrt(variance)

		lowerBand := sma - (multiplier * stdDev)
		result = append(result, models.IndicatorValue{
			Date:  data[i].Date,
			Value: lowerBand,
		})
	}

	return result, nil
}

// calculateATR calculates Average True Range
func calculateATR(data []*models.MarketData, period int, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	if len(data) < period+1 {
		return nil, fmt.Errorf("insufficient data for ATR calculation")
	}

	var trueRanges []float64
	for i := 1; i < len(data); i++ {
		tr1 := data[i].High - data[i].Low
		tr2 := math.Abs(data[i].High - data[i-1].Close)
		tr3 := math.Abs(data[i].Low - data[i-1].Close)
		tr := math.Max(tr1, math.Max(tr2, tr3))
		trueRanges = append(trueRanges, tr)
	}

	var result []models.IndicatorValue
	for i := period - 1; i < len(trueRanges); i++ {
		currentDate, _ := time.Parse("2006-01-02", data[i+1].Date)
		if currentDate.Before(startDate) || currentDate.After(endDate) {
			continue
		}

		atr := 0.0
		for j := i - period + 1; j <= i; j++ {
			atr += trueRanges[j]
		}
		atr /= float64(period)

		result = append(result, models.IndicatorValue{
			Date:  data[i+1].Date,
			Value: atr,
		})
	}

	return result, nil
}

// calculateVWMA calculates Volume Weighted Moving Average
func calculateVWMA(data []*models.MarketData, period int, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	var result []models.IndicatorValue

	for i := period - 1; i < len(data); i++ {
		currentDate, _ := time.Parse("2006-01-02", data[i].Date)
		if currentDate.Before(startDate) || currentDate.After(endDate) {
			continue
		}

		var totalVolume, weightedSum float64
		for j := i - period + 1; j <= i; j++ {
			totalVolume += float64(data[j].Volume)
			weightedSum += data[j].Close * float64(data[j].Volume)
		}

		var vwma float64
		if totalVolume > 0 {
			vwma = weightedSum / totalVolume
		}

		result = append(result, models.IndicatorValue{
			Date:  data[i].Date,
			Value: vwma,
		})
	}

	return result, nil
}

// calculateMFI calculates Money Flow Index
func calculateMFI(data []*models.MarketData, period int, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	if len(data) < period+1 {
		return nil, fmt.Errorf("insufficient data for MFI calculation")
	}

	var result []models.IndicatorValue

	for i := period; i < len(data); i++ {
		currentDate, _ := time.Parse("2006-01-02", data[i].Date)
		if currentDate.Before(startDate) || currentDate.After(endDate) {
			continue
		}

		var positiveFlow, negativeFlow float64

		for j := i - period + 1; j <= i; j++ {
			if j == 0 {
				continue
			}

			typicalPrice := (data[j].High + data[j].Low + data[j].Close) / 3
			prevTypicalPrice := (data[j-1].High + data[j-1].Low + data[j-1].Close) / 3
			rawMoneyFlow := typicalPrice * float64(data[j].Volume)

			if typicalPrice > prevTypicalPrice {
				positiveFlow += rawMoneyFlow
			} else if typicalPrice < prevTypicalPrice {
				negativeFlow += rawMoneyFlow
			}
		}

		var mfi float64
		if negativeFlow == 0 {
			mfi = 100
		} else {
			moneyRatio := positiveFlow / negativeFlow
			mfi = 100 - (100 / (1 + moneyRatio))
		}

		result = append(result, models.IndicatorValue{
			Date:  data[i].Date,
			Value: mfi,
		})
	}

	return result, nil
}

// Helper functions

// calculateEMAValues calculates EMA values for internal use
func calculateEMAValues(data []*models.MarketData, period int) ([]float64, error) {
	if len(data) < period {
		return nil, fmt.Errorf("insufficient data for EMA calculation")
	}

	multiplier := 2.0 / (float64(period) + 1.0)
	var result []float64

	// Calculate initial SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += data[i].Close
	}
	ema := sum / float64(period)
	result = append(result, ema)

	for i := period; i < len(data); i++ {
		ema = (data[i].Close * multiplier) + (ema * (1 - multiplier))
		result = append(result, ema)
	}

	return result, nil
}

// sum calculates the sum of a slice of float64
func sum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
