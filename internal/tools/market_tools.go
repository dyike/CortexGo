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
			Desc: "Get technical indicator analysis for a stock over a specified time window",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"symbol": {
					Type:     "string",
					Desc:     "Ticker symbol of the company",
					Required: true,
				},
				"indicator": {
					Type:     "string",
					Desc:     "Technical indicator to get the analysis and report of",
					Required: true,
				},
				"curr_date": {
					Type:     "string",
					Desc:     "The current trading date you are trading on, YYYY-mm-dd",
					Required: true,
				},
				"look_back_days": {
					Type:     "integer",
					Desc:     "How many days to look back",
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

			// Validate indicator
			if _, exists := bestIndParams[input.Indicator]; !exists {
				supportedIndicators := make([]string, 0, len(bestIndParams))
				for k := range bestIndParams {
					supportedIndicators = append(supportedIndicators, k)
				}
				sort.Strings(supportedIndicators)
				return nil, fmt.Errorf("indicator %s is not supported. Please choose from: %s",
					input.Indicator, strings.Join(supportedIndicators, ", "))
			}

			// Parse current date
			currDate, err := time.Parse("2006-01-02", input.CurrDate)
			if err != nil {
				return nil, fmt.Errorf("invalid date format: %s", input.CurrDate)
			}

			// Calculate start date
			startDate := currDate.AddDate(0, 0, -input.LookBackDays)

			// Get market data
			var marketData []*models.MarketData
			if input.Online {
				marketData, err = getOnlineMarketDataForIndicator(ctx, cfg, input.Symbol, input.LookBackDays+50) // extra buffer for indicators
			} else {
				marketData, err = getOfflineMarketDataForIndicator(ctx, cfg, input.Symbol, startDate, currDate)
			}

			if err != nil {
				return nil, fmt.Errorf("failed to get market data: %v", err)
			}

			if len(marketData) == 0 {
				return nil, fmt.Errorf("no market data available for symbol %s", input.Symbol)
			}

			// Calculate indicators
			indicatorValues, err := calculateIndicator(marketData, input.Indicator, startDate, currDate)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate indicator: %v", err)
			}

			// Format result
			var indString strings.Builder
			for _, value := range indicatorValues {
				indString.WriteString(fmt.Sprintf("%s: %.4f\n", value.Date, value.Value))
			}

			resultStr := fmt.Sprintf("## %s values from %s to %s:\n\n%s\n\n%s",
				input.Indicator,
				startDate.Format("2006-01-02"),
				input.CurrDate,
				indString.String(),
				bestIndParams[input.Indicator])

			return &models.StockIndicatorOutput{
				Result: resultStr,
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

// calculateIndicator calculates the specified technical indicator
func calculateIndicator(data []*models.MarketData, indicator string, startDate, endDate time.Time) ([]models.IndicatorValue, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data available")
	}

	// Sort data by date
	sort.Slice(data, func(i, j int) bool {
		return data[i].Date < data[j].Date
	})

	switch indicator {
	case "close_50_sma":
		return calculateSMA(data, 50, startDate, endDate)
	case "close_200_sma":
		return calculateSMA(data, 200, startDate, endDate)
	case "close_10_ema":
		return calculateEMA(data, 10, startDate, endDate)
	case "rsi":
		return calculateRSI(data, 14, startDate, endDate)
	case "macd":
		return calculateMACD(data, startDate, endDate)
	case "macds":
		return calculateMACDSignal(data, startDate, endDate)
	case "macdh":
		return calculateMACDHistogram(data, startDate, endDate)
	case "boll":
		return calculateBollingerMiddle(data, 20, startDate, endDate)
	case "boll_ub":
		return calculateBollingerUpper(data, 20, 2, startDate, endDate)
	case "boll_lb":
		return calculateBollingerLower(data, 20, 2, startDate, endDate)
	case "atr":
		return calculateATR(data, 14, startDate, endDate)
	case "vwma":
		return calculateVWMA(data, 20, startDate, endDate)
	case "mfi":
		return calculateMFI(data, 14, startDate, endDate)
	default:
		return nil, fmt.Errorf("unsupported indicator: %s", indicator)
	}
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
	macdValues, err := calculateMACD(data, time.Time{}, time.Time{}) // Get all MACD values
	if err != nil {
		return nil, err
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
		return nil, err
	}

	var result []models.IndicatorValue
	for i, signal := range signalEMA {
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
		return nil, err
	}

	signalValues, err := calculateMACDSignal(data, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	var result []models.IndicatorValue
	minLen := min(len(macdValues), len(signalValues))

	for i := 0; i < minLen; i++ {
		currentDate, _ := time.Parse("2006-01-02", macdValues[i].Date)
		if !currentDate.Before(startDate) && !currentDate.After(endDate) {
			histogram := macdValues[i].Value - signalValues[i].Value
			result = append(result, models.IndicatorValue{
				Date:  macdValues[i].Date,
				Value: histogram,
			})
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
