package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/dyike/CortexGo/pkg/config"
	"github.com/dyike/CortexGo/pkg/models"
)

type Researcher struct {
	*BaseAgent
}

func NewResearcher(config *config.Config) *Researcher {
	return &Researcher{
		BaseAgent: NewBaseAgent("researcher", config),
	}
}

func (r *Researcher) Process(ctx context.Context, state *models.AgentState) (*models.AgentState, error) {
	if len(state.Reports) == 0 {
		return state, fmt.Errorf("no analyst reports available for research")
	}

	var ratings []string
	var confidences []float64
	var analyses []string

	for _, report := range state.Reports {
		ratings = append(ratings, report.Rating)
		confidences = append(confidences, report.Confidence)
		analyses = append(analyses, fmt.Sprintf("%s: %s", report.Analyst, report.Analysis))
	}

	avgConfidence := r.calculateAverageConfidence(confidences)
	consensusRating := r.determineConsensusRating(ratings)

	researchSummary := fmt.Sprintf(
		"Research Summary for %s:\n"+
			"Analyst Reports: %d\n"+
			"Consensus Rating: %s\n"+
			"Average Confidence: %.2f\n"+
			"Key Insights: %s\n"+
			"Risk Assessment: %s",
		state.CurrentSymbol,
		len(state.Reports),
		consensusRating,
		avgConfidence,
		strings.Join(analyses, "; "),
		r.assessRisk(ratings, confidences),
	)

	researchReport := models.AnalysisReport{
		Analyst:    r.Name(),
		Symbol:     state.CurrentSymbol,
		Date:       state.CurrentDate,
		Analysis:   researchSummary,
		Rating:     consensusRating,
		Confidence: avgConfidence,
		Metrics: map[string]interface{}{
			"num_reports":       len(state.Reports),
			"consensus_rating":  consensusRating,
			"avg_confidence":    avgConfidence,
			"risk_level":        r.assessRisk(ratings, confidences),
		},
	}

	state.Reports = append(state.Reports, researchReport)
	return state, nil
}

func (r *Researcher) calculateAverageConfidence(confidences []float64) float64 {
	if len(confidences) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, conf := range confidences {
		sum += conf
	}
	return sum / float64(len(confidences))
}

func (r *Researcher) determineConsensusRating(ratings []string) string {
	ratingCounts := make(map[string]int)
	for _, rating := range ratings {
		ratingCounts[rating]++
	}

	maxCount := 0
	consensusRating := "NEUTRAL"
	
	for rating, count := range ratingCounts {
		if count > maxCount {
			maxCount = count
			consensusRating = rating
		}
	}

	return consensusRating
}

func (r *Researcher) assessRisk(ratings []string, confidences []float64) string {
	avgConfidence := r.calculateAverageConfidence(confidences)
	
	ratingVariance := r.calculateRatingVariance(ratings)
	
	if avgConfidence < 0.5 || ratingVariance > 0.7 {
		return "HIGH"
	} else if avgConfidence < 0.7 || ratingVariance > 0.4 {
		return "MEDIUM"
	}
	
	return "LOW"
}

func (r *Researcher) calculateRatingVariance(ratings []string) float64 {
	if len(ratings) <= 1 {
		return 0
	}
	
	ratingCounts := make(map[string]int)
	for _, rating := range ratings {
		ratingCounts[rating]++
	}
	
	if len(ratingCounts) == 1 {
		return 0
	}
	
	return float64(len(ratingCounts)) / float64(len(ratings))
}