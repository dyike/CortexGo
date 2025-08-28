package processing

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/dyike/CortexGo/internal/models"
)

// SignalProcessor extracts actionable decisions from complex analysis text
type SignalProcessor struct {
	buyPatterns  []*regexp.Regexp
	sellPatterns []*regexp.Regexp
	holdPatterns []*regexp.Regexp
}

// TradingSignal represents a processed trading signal
type TradingSignal struct {
	Action       string  `json:"action"`        // BUY, SELL, HOLD
	Confidence   float64 `json:"confidence"`    // 0.0 to 1.0
	Reasoning    string  `json:"reasoning"`     // Extracted reasoning
	EntryPrice   float64 `json:"entry_price"`   // Suggested entry price
	StopLoss     float64 `json:"stop_loss"`     // Stop loss level
	TakeProfit   float64 `json:"take_profit"`   // Take profit level
	PositionSize float64 `json:"position_size"` // Suggested position size
}

// NewSignalProcessor creates a new signal processor with predefined patterns
func NewSignalProcessor() *SignalProcessor {
	return &SignalProcessor{
		buyPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(buy|purchase|long|bullish|positive|upward|invest)\b`),
			regexp.MustCompile(`(?i)\b(strong buy|recommended buy|buy recommendation)\b`),
			regexp.MustCompile(`(?i)\b(undervalued|oversold|growth potential|opportunity)\b`),
		},
		sellPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(sell|short|bearish|negative|downward|divest)\b`),
			regexp.MustCompile(`(?i)\b(strong sell|sell recommendation|avoid)\b`),
			regexp.MustCompile(`(?i)\b(overvalued|overbought|risk|decline)\b`),
		},
		holdPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(hold|maintain|neutral|wait|sideways)\b`),
			regexp.MustCompile(`(?i)\b(no action|stay put|keep position)\b`),
		},
	}
}

// ProcessTradingDecision extracts a trading signal from the final state
func (sp *SignalProcessor) ProcessTradingDecision(ctx context.Context, state *models.TradingState) (*TradingSignal, error) {
	// Combine all analysis text for signal extraction
	combinedText := strings.Join([]string{
		state.MarketReport,
		state.SentimentReport,
		state.NewsReport,
		state.FundamentalsReport,
		state.InvestmentDebateState.JudgeDecision,
		state.TraderInvestmentPlan,
		state.FinalTradeDecision,
	}, " ")

	// Extract action based on pattern matching
	action := sp.extractAction(combinedText)
	confidence := sp.calculateConfidence(combinedText, action)
	reasoning := sp.extractReasoning(combinedText, action)

	// Extract numerical values if available
	entryPrice := sp.extractPrice(combinedText, "entry")
	stopLoss := sp.extractPrice(combinedText, "stop")
	takeProfit := sp.extractPrice(combinedText, "target")
	positionSize := sp.extractPositionSize(combinedText)

	return &TradingSignal{
		Action:       action,
		Confidence:   confidence,
		Reasoning:    reasoning,
		EntryPrice:   entryPrice,
		StopLoss:     stopLoss,
		TakeProfit:   takeProfit,
		PositionSize: positionSize,
	}, nil
}

// extractAction determines the primary trading action from text
func (sp *SignalProcessor) extractAction(text string) string {
	text = strings.ToLower(text)

	buyScore := 0
	sellScore := 0
	holdScore := 0

	// Count pattern matches
	for _, pattern := range sp.buyPatterns {
		buyScore += len(pattern.FindAllString(text, -1))
	}

	for _, pattern := range sp.sellPatterns {
		sellScore += len(pattern.FindAllString(text, -1))
	}

	for _, pattern := range sp.holdPatterns {
		holdScore += len(pattern.FindAllString(text, -1))
	}

	// Determine action based on highest score
	if buyScore > sellScore && buyScore > holdScore {
		return "BUY"
	} else if sellScore > buyScore && sellScore > holdScore {
		return "SELL"
	}

	return "HOLD"
}

// calculateConfidence calculates confidence based on signal strength
func (sp *SignalProcessor) calculateConfidence(text string, action string) float64 {
	text = strings.ToLower(text)
	totalWords := len(strings.Fields(text))

	if totalWords == 0 {
		return 0.5
	}

	var relevantPatterns []*regexp.Regexp
	switch action {
	case "BUY":
		relevantPatterns = sp.buyPatterns
	case "SELL":
		relevantPatterns = sp.sellPatterns
	case "HOLD":
		relevantPatterns = sp.holdPatterns
	}

	matchCount := 0
	for _, pattern := range relevantPatterns {
		matchCount += len(pattern.FindAllString(text, -1))
	}

	// Calculate confidence as percentage of relevant words
	confidence := float64(matchCount) / float64(totalWords) * 10
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.1 {
		confidence = 0.1
	}

	return confidence
}

// extractReasoning extracts key reasoning points from the text
func (sp *SignalProcessor) extractReasoning(text string, action string) string {
	sentences := strings.Split(text, ".")
	relevantSentences := []string{}

	actionWords := map[string][]string{
		"BUY":  {"buy", "bullish", "positive", "growth", "opportunity", "undervalued"},
		"SELL": {"sell", "bearish", "negative", "risk", "decline", "overvalued"},
		"HOLD": {"hold", "neutral", "wait", "maintain", "uncertain"},
	}

	words := actionWords[action]
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) < 10 {
			continue
		}

		for _, word := range words {
			if strings.Contains(strings.ToLower(sentence), word) {
				relevantSentences = append(relevantSentences, sentence)
				break
			}
		}

		if len(relevantSentences) >= 3 {
			break
		}
	}

	if len(relevantSentences) == 0 {
		return "Decision based on comprehensive analysis of market conditions."
	}

	return strings.Join(relevantSentences, ". ")
}

// extractPrice extracts price values from text
func (sp *SignalProcessor) extractPrice(text string, priceType string) float64 {
	patterns := map[string]*regexp.Regexp{
		"entry":  regexp.MustCompile(`(?i)entry[^$]*\$?(\d+\.?\d*)`),
		"stop":   regexp.MustCompile(`(?i)stop[^$]*\$?(\d+\.?\d*)`),
		"target": regexp.MustCompile(`(?i)target[^$]*\$?(\d+\.?\d*)`),
	}

	pattern := patterns[priceType]
	if pattern == nil {
		return 0.0
	}

	matches := pattern.FindStringSubmatch(text)
	if len(matches) > 1 {
		var price float64
		if json.Unmarshal([]byte(matches[1]), &price) == nil {
			return price
		}
	}

	return 0.0
}

// extractPositionSize extracts position sizing information
func (sp *SignalProcessor) extractPositionSize(text string) float64 {
	pattern := regexp.MustCompile(`(?i)position[^0-9]*(\d+\.?\d*)`)
	matches := pattern.FindStringSubmatch(text)

	if len(matches) > 1 {
		var size float64
		if json.Unmarshal([]byte(matches[1]), &size) == nil {
			return size
		}
	}

	// Default position size
	return 0.1 // 10% of portfolio
}

// ProcessSignal creates a structured trading decision from the current state
func ProcessSignal(ctx context.Context, state *models.TradingState) (*models.TradingDecision, error) {
	processor := NewSignalProcessor()
	signal, err := processor.ProcessTradingDecision(ctx, state)
	if err != nil {
		return nil, err
	}

	return &models.TradingDecision{
		Symbol:       state.CompanyOfInterest,
		Action:       signal.Action,
		Confidence:   signal.Confidence,
		Reasoning:    signal.Reasoning,
		EntryPrice:   signal.EntryPrice,
		StopLoss:     signal.StopLoss,
		TakeProfit:   signal.TakeProfit,
		PositionSize: signal.PositionSize,
		Timestamp:    state.TradeDate,
	}, nil
}
