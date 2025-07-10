package agents

import (
	"context"
	"fmt"

	"github.com/dyike/CortexGo/pkg/config"
	"github.com/dyike/CortexGo/pkg/models"
)

type FundamentalAnalyst struct {
	*BaseAgent
}

func NewFundamentalAnalyst(config *config.Config) *FundamentalAnalyst {
	return &FundamentalAnalyst{
		BaseAgent: NewBaseAgent("fundamental_analyst", config),
	}
}

func (f *FundamentalAnalyst) Process(ctx context.Context, state *models.AgentState) (*models.AgentState, error) {
	analysis := fmt.Sprintf("Fundamental analysis for %s: Analyzing P/E ratio, revenue growth, and market position.", state.CurrentSymbol)
	
	report := models.AnalysisReport{
		Analyst:    f.Name(),
		Symbol:     state.CurrentSymbol,
		Date:       state.CurrentDate,
		Analysis:   analysis,
		Rating:     "BUY",
		Confidence: 0.75,
		Metrics: map[string]interface{}{
			"pe_ratio":       15.2,
			"revenue_growth": 12.5,
			"debt_ratio":     0.3,
		},
	}
	
	state.Reports = append(state.Reports, report)
	return state, nil
}

type SentimentAnalyst struct {
	*BaseAgent
}

func NewSentimentAnalyst(config *config.Config) *SentimentAnalyst {
	return &SentimentAnalyst{
		BaseAgent: NewBaseAgent("sentiment_analyst", config),
	}
}

func (s *SentimentAnalyst) Process(ctx context.Context, state *models.AgentState) (*models.AgentState, error) {
	analysis := fmt.Sprintf("Sentiment analysis for %s: Analyzing social media sentiment and market mood.", state.CurrentSymbol)
	
	report := models.AnalysisReport{
		Analyst:    s.Name(),
		Symbol:     state.CurrentSymbol,
		Date:       state.CurrentDate,
		Analysis:   analysis,
		Rating:     "HOLD",
		Confidence: 0.65,
		Metrics: map[string]interface{}{
			"social_sentiment": 0.6,
			"news_sentiment":   0.7,
			"volume_trend":     "increasing",
		},
	}
	
	state.Reports = append(state.Reports, report)
	return state, nil
}

type TechnicalAnalyst struct {
	*BaseAgent
}

func NewTechnicalAnalyst(config *config.Config) *TechnicalAnalyst {
	return &TechnicalAnalyst{
		BaseAgent: NewBaseAgent("technical_analyst", config),
	}
}

func (t *TechnicalAnalyst) Process(ctx context.Context, state *models.AgentState) (*models.AgentState, error) {
	analysis := fmt.Sprintf("Technical analysis for %s: Analyzing charts, indicators, and price patterns.", state.CurrentSymbol)
	
	report := models.AnalysisReport{
		Analyst:    t.Name(),
		Symbol:     state.CurrentSymbol,
		Date:       state.CurrentDate,
		Analysis:   analysis,
		Rating:     "BUY",
		Confidence: 0.80,
		Metrics: map[string]interface{}{
			"rsi":            45.2,
			"macd":           0.8,
			"moving_avg_50":  123.45,
			"moving_avg_200": 118.32,
		},
	}
	
	state.Reports = append(state.Reports, report)
	return state, nil
}

type NewsAnalyst struct {
	*BaseAgent
}

func NewNewsAnalyst(config *config.Config) *NewsAnalyst {
	return &NewsAnalyst{
		BaseAgent: NewBaseAgent("news_analyst", config),
	}
}

func (n *NewsAnalyst) Process(ctx context.Context, state *models.AgentState) (*models.AgentState, error) {
	analysis := fmt.Sprintf("News analysis for %s: Analyzing recent news and economic indicators.", state.CurrentSymbol)
	
	report := models.AnalysisReport{
		Analyst:    n.Name(),
		Symbol:     state.CurrentSymbol,
		Date:       state.CurrentDate,
		Analysis:   analysis,
		Rating:     "NEUTRAL",
		Confidence: 0.70,
		Metrics: map[string]interface{}{
			"news_count":      15,
			"positive_news":   8,
			"negative_news":   3,
			"neutral_news":    4,
			"impact_score":    0.6,
		},
	}
	
	state.Reports = append(state.Reports, report)
	return state, nil
}