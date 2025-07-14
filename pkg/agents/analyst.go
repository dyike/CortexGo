package agents

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

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
	pe_ratio := 15.2
	revenue_growth := 12.5
	debt_ratio := 0.3
	
	analysis := fmt.Sprintf("Fundamental analysis for %s: P/E ratio %.1f suggests %s valuation. Revenue growth of %.1f%% indicates %s momentum. Debt ratio of %.1f shows %s financial health.", 
		state.CurrentSymbol, pe_ratio, f.evaluatePE(pe_ratio), revenue_growth, f.evaluateGrowth(revenue_growth), debt_ratio, f.evaluateDebt(debt_ratio))

	rating, confidence := f.calculateFundamentalRating(pe_ratio, revenue_growth, debt_ratio)
	
	report := models.AnalysisReport{
		Analyst:    f.Name(),
		Symbol:     state.CurrentSymbol,
		Date:       state.CurrentDate,
		Analysis:   analysis,
		Rating:     rating,
		Confidence: confidence,
		Metrics: map[string]interface{}{
			"pe_ratio":       pe_ratio,
			"revenue_growth": revenue_growth,
			"debt_ratio":     debt_ratio,
		},
		KeyFindings: f.getKeyFindings(pe_ratio, revenue_growth, debt_ratio),
		Concerns:   f.getConcerns(pe_ratio, revenue_growth, debt_ratio),
		Priority:   f.getPriority(confidence),
	}

	state.Reports = append(state.Reports, report)
	return state, nil
}

func (f *FundamentalAnalyst) evaluatePE(pe float64) string {
	if pe < 15 { return "attractive" }
	if pe < 25 { return "fair" }
	return "expensive"
}

func (f *FundamentalAnalyst) evaluateGrowth(growth float64) string {
	if growth > 15 { return "strong" }
	if growth > 5 { return "moderate" }
	return "weak"
}

func (f *FundamentalAnalyst) evaluateDebt(debt float64) string {
	if debt < 0.3 { return "strong" }
	if debt < 0.6 { return "acceptable" }
	return "concerning"
}

func (f *FundamentalAnalyst) calculateFundamentalRating(pe, growth, debt float64) (string, float64) {
	score := 0.0
	if pe < 15 { score += 0.4 } else if pe < 25 { score += 0.2 }
	if growth > 15 { score += 0.4 } else if growth > 5 { score += 0.2 }
	if debt < 0.3 { score += 0.2 } else if debt < 0.6 { score += 0.1 }
	
	confidence := math.Min(0.9, 0.5 + score)
	if score > 0.7 { return "BUY", confidence }
	if score > 0.4 { return "HOLD", confidence }
	return "SELL", confidence
}

func (f *FundamentalAnalyst) getKeyFindings(pe, growth, debt float64) []string {
	findings := []string{}
	if pe < 15 { findings = append(findings, "Attractive valuation with low P/E ratio") }
	if growth > 15 { findings = append(findings, "Strong revenue growth trajectory") }
	if debt < 0.3 { findings = append(findings, "Conservative debt management") }
	return findings
}

func (f *FundamentalAnalyst) getConcerns(pe, growth, debt float64) []string {
	concerns := []string{}
	if pe > 25 { concerns = append(concerns, "High valuation multiple may limit upside") }
	if growth < 5 { concerns = append(concerns, "Slowing revenue growth momentum") }
	if debt > 0.6 { concerns = append(concerns, "High debt levels pose financial risk") }
	return concerns
}

func (f *FundamentalAnalyst) getPriority(confidence float64) int {
	if confidence > 0.8 { return 1 }
	if confidence > 0.6 { return 2 }
	return 3
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
	social_sentiment := 0.6
	news_sentiment := 0.7
	volume_trend := "increasing"
	
	analysis := fmt.Sprintf("Sentiment analysis for %s: Social sentiment %.1f (%s), news sentiment %.1f (%s), volume trend %s. Overall market mood appears %s.", 
		state.CurrentSymbol, social_sentiment, s.interpretSentiment(social_sentiment), news_sentiment, s.interpretSentiment(news_sentiment), 
		volume_trend, s.getOverallMood(social_sentiment, news_sentiment))

	rating, confidence := s.calculateSentimentRating(social_sentiment, news_sentiment, volume_trend)
	
	report := models.AnalysisReport{
		Analyst:    s.Name(),
		Symbol:     state.CurrentSymbol,
		Date:       state.CurrentDate,
		Analysis:   analysis,
		Rating:     rating,
		Confidence: confidence,
		Metrics: map[string]interface{}{
			"social_sentiment": social_sentiment,
			"news_sentiment":   news_sentiment,
			"volume_trend":     volume_trend,
		},
		KeyFindings: s.getKeyFindings(social_sentiment, news_sentiment, volume_trend),
		Concerns:   s.getConcerns(social_sentiment, news_sentiment),
		Priority:   s.getPriority(confidence),
	}

	state.Reports = append(state.Reports, report)
	return state, nil
}

func (s *SentimentAnalyst) interpretSentiment(sentiment float64) string {
	if sentiment > 0.7 { return "very positive" }
	if sentiment > 0.5 { return "positive" }
	if sentiment > 0.3 { return "negative" }
	return "very negative"
}

func (s *SentimentAnalyst) getOverallMood(social, news float64) string {
	avg := (social + news) / 2
	if avg > 0.6 { return "optimistic" }
	if avg > 0.4 { return "neutral" }
	return "pessimistic"
}

func (s *SentimentAnalyst) calculateSentimentRating(social, news float64, volume string) (string, float64) {
	score := (social + news) / 2
	confidence := 0.6 + (math.Abs(score-0.5) * 0.4)
	
	if volume == "increasing" { score += 0.1 }
	
	if score > 0.65 { return "BUY", confidence }
	if score > 0.35 { return "HOLD", confidence }
	return "SELL", confidence
}

func (s *SentimentAnalyst) getKeyFindings(social, news float64, volume string) []string {
	findings := []string{}
	if social > 0.7 { findings = append(findings, "Strong positive social media sentiment") }
	if news > 0.7 { findings = append(findings, "Favorable news coverage") }
	if volume == "increasing" { findings = append(findings, "Rising trading volume supports sentiment") }
	return findings
}

func (s *SentimentAnalyst) getConcerns(social, news float64) []string {
	concerns := []string{}
	if social < 0.3 { concerns = append(concerns, "Negative social media sentiment") }
	if news < 0.3 { concerns = append(concerns, "Unfavorable news coverage") }
	return concerns
}

func (s *SentimentAnalyst) getPriority(confidence float64) int {
	if confidence > 0.75 { return 2 }
	return 3
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
	rsi := 45.2
	macd := 0.8
	ma50 := 123.45
	ma200 := 118.32
	currentPrice := state.MarketData.Price
	
	analysis := fmt.Sprintf("Technical analysis for %s: RSI %.1f (%s), MACD %.2f (%s), MA50 %.2f vs MA200 %.2f (%s). Current price %.2f shows %s pattern.", 
		state.CurrentSymbol, rsi, t.interpretRSI(rsi), macd, t.interpretMACD(macd), ma50, ma200, 
		t.interpretMovingAverages(ma50, ma200), currentPrice, t.getTrend(currentPrice, ma50, ma200))

	rating, confidence := t.calculateTechnicalRating(rsi, macd, ma50, ma200, currentPrice)
	
	report := models.AnalysisReport{
		Analyst:    t.Name(),
		Symbol:     state.CurrentSymbol,
		Date:       state.CurrentDate,
		Analysis:   analysis,
		Rating:     rating,
		Confidence: confidence,
		Metrics: map[string]interface{}{
			"rsi":            rsi,
			"macd":           macd,
			"moving_avg_50":  ma50,
			"moving_avg_200": ma200,
			"current_price":  currentPrice,
		},
		KeyFindings: t.getKeyFindings(rsi, macd, ma50, ma200, currentPrice),
		Concerns:   t.getConcerns(rsi, macd),
		Priority:   t.getPriority(confidence),
	}

	state.Reports = append(state.Reports, report)
	return state, nil
}

func (t *TechnicalAnalyst) interpretRSI(rsi float64) string {
	if rsi > 70 { return "overbought" }
	if rsi < 30 { return "oversold" }
	return "neutral"
}

func (t *TechnicalAnalyst) interpretMACD(macd float64) string {
	if macd > 0.5 { return "strong bullish" }
	if macd > 0 { return "bullish" }
	if macd > -0.5 { return "bearish" }
	return "strong bearish"
}

func (t *TechnicalAnalyst) interpretMovingAverages(ma50, ma200 float64) string {
	if ma50 > ma200 { return "golden cross pattern" }
	return "death cross pattern"
}

func (t *TechnicalAnalyst) getTrend(price, ma50, ma200 float64) string {
	if price > ma50 && ma50 > ma200 { return "strong uptrend" }
	if price > ma50 { return "uptrend" }
	if price < ma50 && ma50 < ma200 { return "strong downtrend" }
	return "downtrend"
}

func (t *TechnicalAnalyst) calculateTechnicalRating(rsi, macd, ma50, ma200, price float64) (string, float64) {
	score := 0.0
	
	if rsi < 30 { score += 0.3 } else if rsi < 70 { score += 0.1 }
	if macd > 0 { score += 0.3 }
	if price > ma50 { score += 0.2 }
	if ma50 > ma200 { score += 0.2 }
	
	confidence := 0.7 + (math.Abs(score-0.5) * 0.3)
	
	if score > 0.7 { return "BUY", confidence }
	if score > 0.3 { return "HOLD", confidence }
	return "SELL", confidence
}

func (t *TechnicalAnalyst) getKeyFindings(rsi, macd, ma50, ma200, price float64) []string {
	findings := []string{}
	if rsi < 30 { findings = append(findings, "RSI indicates oversold conditions") }
	if macd > 0.5 { findings = append(findings, "Strong MACD bullish signal") }
	if price > ma50 && ma50 > ma200 { findings = append(findings, "Price above both moving averages - strong uptrend") }
	return findings
}

func (t *TechnicalAnalyst) getConcerns(rsi, macd float64) []string {
	concerns := []string{}
	if rsi > 70 { concerns = append(concerns, "RSI indicates overbought conditions") }
	if macd < -0.5 { concerns = append(concerns, "Strong bearish MACD signal") }
	return concerns
}

func (t *TechnicalAnalyst) getPriority(confidence float64) int {
	if confidence > 0.8 { return 1 }
	if confidence > 0.65 { return 2 }
	return 3
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
	newsCount := 15
	positiveNews := 8
	negativeNews := 3
	neutralNews := 4
	impactScore := 0.6
	
	analysis := fmt.Sprintf("News analysis for %s: %d total news items (%d positive, %d negative, %d neutral). Impact score %.1f suggests %s market influence. Recent developments show %s sentiment shift.", 
		state.CurrentSymbol, newsCount, positiveNews, negativeNews, neutralNews, impactScore, 
		n.interpretImpact(impactScore), n.getSentimentShift(positiveNews, negativeNews))

	rating, confidence := n.calculateNewsRating(positiveNews, negativeNews, impactScore)
	
	report := models.AnalysisReport{
		Analyst:    n.Name(),
		Symbol:     state.CurrentSymbol,
		Date:       state.CurrentDate,
		Analysis:   analysis,
		Rating:     rating,
		Confidence: confidence,
		Metrics: map[string]interface{}{
			"news_count":    newsCount,
			"positive_news": positiveNews,
			"negative_news": negativeNews,
			"neutral_news":  neutralNews,
			"impact_score":  impactScore,
		},
		KeyFindings: n.getKeyFindings(positiveNews, negativeNews, impactScore),
		Concerns:   n.getConcerns(negativeNews, impactScore),
		Priority:   n.getPriority(confidence),
	}

	state.Reports = append(state.Reports, report)
	return state, nil
}

func (n *NewsAnalyst) interpretImpact(impact float64) string {
	if impact > 0.7 { return "high" }
	if impact > 0.4 { return "moderate" }
	return "low"
}

func (n *NewsAnalyst) getSentimentShift(positive, negative int) string {
	ratio := float64(positive) / float64(positive + negative)
	if ratio > 0.7 { return "strongly positive" }
	if ratio > 0.5 { return "positive" }
	if ratio < 0.3 { return "negative" }
	return "mixed"
}

func (n *NewsAnalyst) calculateNewsRating(positive, negative int, impact float64) (string, float64) {
	total := positive + negative
	if total == 0 { return "NEUTRAL", 0.5 }
	
	sentiment := float64(positive) / float64(total)
	score := (sentiment * 0.7) + (impact * 0.3)
	confidence := 0.6 + (math.Abs(score-0.5) * 0.4)
	
	if score > 0.65 { return "BUY", confidence }
	if score > 0.35 { return "HOLD", confidence }
	return "SELL", confidence
}

func (n *NewsAnalyst) getKeyFindings(positive, negative int, impact float64) []string {
	findings := []string{}
	if positive > negative*2 { findings = append(findings, "Predominantly positive news coverage") }
	if impact > 0.7 { findings = append(findings, "High-impact news events detected") }
	return findings
}

func (n *NewsAnalyst) getConcerns(negative int, impact float64) []string {
	concerns := []string{}
	if negative > 5 { concerns = append(concerns, "Significant negative news coverage") }
	if impact < 0.3 { concerns = append(concerns, "Low news impact may indicate weak market interest") }
	return concerns
}

func (n *NewsAnalyst) getPriority(confidence float64) int {
	if confidence > 0.75 { return 2 }
	return 3
}

type AnalystTeam struct {
	analysts []Agent
	config   *config.Config
}

func NewAnalystTeam(config *config.Config) *AnalystTeam {
	return &AnalystTeam{
		analysts: []Agent{
			NewFundamentalAnalyst(config),
			NewSentimentAnalyst(config),
			NewTechnicalAnalyst(config),
			NewNewsAnalyst(config),
		},
		config: config,
	}
}

func (team *AnalystTeam) ConductAnalysis(ctx context.Context, state *models.AgentState) (*models.AgentState, error) {
	for _, analyst := range team.analysts {
		var err error
		state, err = analyst.Process(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("analyst %s failed: %v", analyst.Name(), err)
		}
	}
	
	discussion, err := team.FacilitateDiscussion(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("team discussion failed: %v", err)
	}
	
	state.Discussions = append(state.Discussions, *discussion)
	state.TeamConsensus = discussion.Consensus
	
	return state, nil
}

func (team *AnalystTeam) FacilitateDiscussion(ctx context.Context, state *models.AgentState) (*models.AnalystDiscussion, error) {
	if len(state.Reports) == 0 {
		return nil, fmt.Errorf("no reports available for discussion")
	}
	
	participants := make([]string, len(state.Reports))
	for i, report := range state.Reports {
		participants[i] = report.Analyst
	}
	
	discussion := &models.AnalystDiscussion{
		Participants: participants,
		Topic:        fmt.Sprintf("Investment decision for %s", state.CurrentSymbol),
		DebatePoints: []models.DebatePoint{},
		Timestamp:    time.Now(),
	}
	
	conflictingReports := team.findConflictingViews(state.Reports)
	for _, conflict := range conflictingReports {
		debatePoint := models.DebatePoint{
			Analyst:   conflict.Analyst,
			Position:  conflict.Rating,
			Evidence:  conflict.KeyFindings,
			Response:  team.generateResponse(conflict, state.Reports),
			Timestamp: time.Now(),
		}
		discussion.DebatePoints = append(discussion.DebatePoints, debatePoint)
	}
	
	consensus := team.reachConsensus(state.Reports)
	discussion.Consensus = consensus
	
	return discussion, nil
}

func (team *AnalystTeam) findConflictingViews(reports []models.AnalysisReport) []models.AnalysisReport {
	ratingCounts := make(map[string]int)
	for _, report := range reports {
		ratingCounts[report.Rating]++
	}
	
	if len(ratingCounts) <= 1 {
		return []models.AnalysisReport{}
	}
	
	conflicting := []models.AnalysisReport{}
	for _, report := range reports {
		if ratingCounts[report.Rating] == 1 || len(report.Concerns) > 0 {
			conflicting = append(conflicting, report)
		}
	}
	
	return conflicting
}

func (team *AnalystTeam) generateResponse(analyst models.AnalysisReport, allReports []models.AnalysisReport) string {
	responses := []string{}
	
	for _, other := range allReports {
		if other.Analyst != analyst.Analyst && other.Rating != analyst.Rating {
			response := fmt.Sprintf("Disagrees with %s's %s rating based on %s analysis", 
				other.Analyst, other.Rating, strings.ToLower(analyst.Analyst))
			responses = append(responses, response)
		}
	}
	
	if len(responses) == 0 {
		return fmt.Sprintf("Supports team consensus with %s rating", analyst.Rating)
	}
	
	return strings.Join(responses, "; ")
}

func (team *AnalystTeam) reachConsensus(reports []models.AnalysisReport) *models.Consensus {
	if len(reports) == 0 {
		return &models.Consensus{
			FinalRating:    "NEUTRAL",
			AgreementLevel: 0.0,
			MainArguments:  []string{},
			Dissents:       []string{},
			Confidence:     0.5,
		}
	}
	
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Priority < reports[j].Priority
	})
	
	ratingWeights := make(map[string]float64)
	totalWeight := 0.0
	arguments := []string{}
	dissents := []string{}
	
	for _, report := range reports {
		weight := report.Confidence / float64(report.Priority)
		ratingWeights[report.Rating] += weight
		totalWeight += weight
		
		arguments = append(arguments, report.KeyFindings...)
		if len(report.Concerns) > 0 {
			dissents = append(dissents, report.Concerns...)
		}
	}
	
	var finalRating string
	var maxWeight float64
	for rating, weight := range ratingWeights {
		if weight > maxWeight {
			maxWeight = weight
			finalRating = rating
		}
	}
	
	agreementLevel := maxWeight / totalWeight
	avgConfidence := 0.0
	for _, report := range reports {
		avgConfidence += report.Confidence
	}
	avgConfidence /= float64(len(reports))
	
	confidence := avgConfidence * agreementLevel
	
	return &models.Consensus{
		FinalRating:    finalRating,
		AgreementLevel: agreementLevel,
		MainArguments:  arguments,
		Dissents:       dissents,
		Confidence:     confidence,
	}
}
