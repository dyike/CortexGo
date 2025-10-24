package main

import (
	"encoding/json"

	"github.com/dyike/CortexGo/internal/models"
)

type analysisResponse struct {
	Success bool                 `json:"success"`
	Error   string               `json:"error,omitempty"`
	Symbol  string               `json:"symbol,omitempty"`
	Date    string               `json:"date,omitempty"`
	Summary json.RawMessage      `json:"summary,omitempty"`
	State   *models.TradingState `json:"state,omitempty"`
}
