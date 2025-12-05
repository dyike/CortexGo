package dataflows

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/dyike/CortexGo/models"
)

func CalculateAllIndicators(data []*models.MarketData, startDate, endDate time.Time) map[string][]models.IndicatorValue {
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
