package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"sync"
	"unsafe"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/display"
	"github.com/dyike/CortexGo/internal/graph"
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

var (
	cfgMu     sync.RWMutex
	activeCfg *config.Config
)

func ensureConfig() (*config.Config, error) {
	cfgMu.RLock()
	if activeCfg != nil {
		cfg := activeCfg
		cfgMu.RUnlock()
		return cfg, nil
	}
	cfgMu.RUnlock()

	cfgMu.Lock()
	defer cfgMu.Unlock()

	if activeCfg == nil {
		cfg := config.DefaultConfig()
		if err := cfg.EnsureDirectories(); err != nil {
			return nil, err
		}
		activeCfg = cfg
	}

	return activeCfg, nil
}

func runAnalysis(symbol, date string) (*analysisResponse, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}
	if date == "" {
		return nil, fmt.Errorf("date cannot be empty")
	}

	cfg, err := ensureConfig()
	if err != nil {
		return nil, err
	}

	tradingGraph := graph.NewTradingAgentsGraph(cfg.Debug, cfg)
	state, err := tradingGraph.Propagate(symbol, date)
	if err != nil {
		return nil, err
	}

	if state != nil {
		state.Config = nil
	}

	summaryJSON, err := display.NewResultsDisplay(symbol, date).SerializeResults(state)
	if err != nil {
		return nil, err
	}

	return &analysisResponse{
		Success: true,
		Symbol:  symbol,
		Date:    date,
		Summary: json.RawMessage(summaryJSON),
		State:   state,
	}, nil
}

//export CortexGoAnalyze
func CortexGoAnalyze(symbol *C.char, date *C.char) *C.char {
	goSymbol := C.GoString(symbol)
	goDate := C.GoString(date)

	resp, err := runAnalysis(goSymbol, goDate)
	if err != nil {
		resp = &analysisResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	payload, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		fallback := map[string]any{
			"success": false,
			"error":   marshalErr.Error(),
		}
		data, _ := json.Marshal(fallback)
		return C.CString(string(data))
	}

	return C.CString(string(payload))
}

//export CortexGoFree
func CortexGoFree(ptr *C.char) {
	if ptr != nil {
		C.free(unsafe.Pointer(ptr))
	}
}

func main() {}
