package main

/*
#include <stdlib.h>

typedef void (*log_callback_t)(const char *msg, void *user_data);
static inline void call_log_callback(log_callback_t cb, const char *msg, void *user_data) {
	if (cb != NULL) {
		cb(msg, user_data);
	}
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/graph"
	"github.com/dyike/CortexGo/internal/models"
)

type configResponse struct {
	Success bool           `json:"success"`
	Error   string         `json:"error,omitempty"`
	Config  *config.Config `json:"config,omitempty"`
}

var (
	cfgMu     sync.RWMutex
	activeCfg *config.Config

	cfgPathMu sync.RWMutex
	cfgPath   string

	logCbMu sync.RWMutex
	logCb   C.log_callback_t
	logCtx  unsafe.Pointer
)

// Helpers -------------------------------------------------------------------

func goString(ptr *C.char) string {
	if ptr == nil {
		return ""
	}
	return C.GoString(ptr)
}

func setActiveConfig(cfg *config.Config) {
	cfgMu.Lock()
	activeCfg = cfg
	cfgMu.Unlock()
}

func ensureConfig() (*config.Config, error) {
	cfgMu.RLock()
	cfg := activeCfg
	cfgMu.RUnlock()
	if cfg == nil {
		return nil, fmt.Errorf("configuration not initialized; call CortexGoSetConfigPath first")
	}
	return cfg, nil
}

func buildConfig(base *config.Config, payload string) (*config.Config, error) {
	var cfg *config.Config
	if base == nil {
		cfg = config.DefaultConfig()
	} else {
		clone := *base
		cfg = &clone
	}

	if trimmed := strings.TrimSpace(payload); trimmed != "" {
		if err := json.Unmarshal([]byte(trimmed), cfg); err != nil {
			return nil, err
		}
	}

	normalizeConfig(cfg)
	return cfg, nil
}

func currentConfigCopy() (*config.Config, error) {
	cfg, err := ensureConfig()
	if err != nil {
		return nil, err
	}
	clone := *cfg
	return &clone, nil
}

func currentConfigPath() string {
	cfgPathMu.RLock()
	defer cfgPathMu.RUnlock()
	return cfgPath
}

func updateConfigPath(path string) {
	cfgPathMu.Lock()
	cfgPath = path
	cfgPathMu.Unlock()
}

func loadConfigFromPath(path string) (*config.Config, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, fmt.Errorf("config path is empty")
	}
	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return nil, err
	}
	cfg, loadErr := safeLoadConfigFromFile(absPath)
	if loadErr != nil {
		return nil, loadErr
	}
	normalizeConfig(cfg)
	return cfg, nil
}

func safeLoadConfigFromFile(path string) (cfg *config.Config, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("load config: %v", r)
			cfg = nil
		}
	}()
	cfg = config.LoadConfigFromJsonFile(path)
	return cfg, nil
}

func normalizeConfig(cfg *config.Config) {
	if cfg == nil {
		return
	}

	if strings.TrimSpace(cfg.ProjectDir) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		cfg.ProjectDir = cwd
	}

	if strings.TrimSpace(cfg.ResultsDir) == "" {
		cfg.ResultsDir = filepath.Join(cfg.ProjectDir, "results")
	}

	if strings.TrimSpace(cfg.DataDir) == "" {
		cfg.DataDir = filepath.Join(cfg.ProjectDir, "data")
	}

	if strings.TrimSpace(cfg.DataCacheDir) == "" {
		cfg.DataCacheDir = filepath.Join(cfg.DataDir, "cache")
	}

	if cfg.EinoDebugPort == 0 {
		cfg.EinoDebugPort = 52538
	}
}

func persistConfigToDisk(cfg *config.Config) error {
	path := currentConfigPath()
	if cfg == nil || strings.TrimSpace(path) == "" {
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(abs, data, 0o644)
}

// Logging --------------------------------------------------------------------

func emitToRegisteredCallback(event string, data *models.ChatResp) {
	logCbMu.RLock()
	cb := logCb
	ctx := logCtx
	logCbMu.RUnlock()

	if cb == nil {
		if data == nil {
			return
		}
		if strings.TrimSpace(data.Content) != "" {
			fmt.Print(data.Content)
		}
		return
	}

	envelope := map[string]any{"event": event}
	if data != nil {
		envelope["data"] = data
	} else {
		envelope["data"] = map[string]any{}
	}

	bytes, err := json.Marshal(envelope)
	if err != nil {
		fallback := map[string]any{"event": "log_error", "error": err.Error()}
		bytes, _ = json.Marshal(fallback)
	}

	cstr := C.CString(string(bytes))
	defer C.free(unsafe.Pointer(cstr))
	C.call_log_callback(cb, cstr, ctx)
}

// Analysis -------------------------------------------------------------------

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

	tradingGraph := graph.NewTradingAgentsGraphWithEmitter(cfg, emitToRegisteredCallback)
	if tradingGraph == nil {
		return nil, fmt.Errorf("failed to initialize trading graph")
	}

	state, err := tradingGraph.Propagate(symbol, date)
	if err != nil {
		return nil, err
	}

	if state != nil {
		state.Config = nil
	}

	summaryJSON, err := json.Marshal(state)
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

func encodeConfigResponse(resp configResponse) *C.char {
	data, err := json.Marshal(resp)
	if err != nil {
		fallback := map[string]any{"success": false, "error": err.Error()}
		data, _ = json.Marshal(fallback)
	}
	return C.CString(string(data))
}

func encodeAnalysisPayload(resp *analysisResponse) *C.char {
	if resp == nil {
		resp = &analysisResponse{Success: false, Error: "nil response"}
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		fallback := map[string]any{"success": false, "error": err.Error()}
		payload, _ = json.Marshal(fallback)
	}
	return C.CString(string(payload))
}

// Exported APIs --------------------------------------------------------------

//export CortexGoRegisterLogCallback
func CortexGoRegisterLogCallback(cb C.log_callback_t, user unsafe.Pointer) {
	logCbMu.Lock()
	logCb = cb
	logCtx = user
	logCbMu.Unlock()
}

//export CortexGoSetConfigPath
func CortexGoSetConfigPath(path *C.char) *C.char {
	goPath := goString(path)
	cfg, err := loadConfigFromPath(goPath)
	if err != nil {
		return encodeConfigResponse(configResponse{Success: false, Error: err.Error()})
	}
	absPath, _ := filepath.Abs(strings.TrimSpace(goPath))
	updateConfigPath(absPath)
	setActiveConfig(cfg)
	return encodeConfigResponse(configResponse{Success: true, Config: cfg})
}

//export CortexGoUpdateConfigJSON
func CortexGoUpdateConfigJSON(configJSON *C.char) *C.char {
	jsonPayload := goString(configJSON)
	if strings.TrimSpace(jsonPayload) == "" {
		return encodeConfigResponse(configResponse{Success: false, Error: "config payload is empty"})
	}

	current, err := currentConfigCopy()
	if err != nil {
		current = config.DefaultConfig()
	}

	cfg, err := buildConfig(current, jsonPayload)
	if err != nil {
		return encodeConfigResponse(configResponse{Success: false, Error: err.Error()})
	}

	setActiveConfig(cfg)
	if err := persistConfigToDisk(cfg); err != nil {
		return encodeConfigResponse(configResponse{Success: false, Error: err.Error()})
	}

	return encodeConfigResponse(configResponse{Success: true, Config: cfg})
}

//export CortexGoAnalyze
func CortexGoAnalyze(symbol *C.char, date *C.char) *C.char {
	resp, err := runAnalysis(C.GoString(symbol), C.GoString(date))
	if err != nil {
		resp = &analysisResponse{Success: false, Error: err.Error()}
	}
	return encodeAnalysisPayload(resp)
}

func main() {}
