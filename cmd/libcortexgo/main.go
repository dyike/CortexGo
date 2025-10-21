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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

type configResponse struct {
	Success bool           `json:"success"`
	Error   string         `json:"error,omitempty"`
	Config  *config.Config `json:"config,omitempty"`
}

var (
	cfgMu     sync.RWMutex
	activeCfg *config.Config

	logCbMu sync.RWMutex
	logCb   C.log_callback_t
	logCtx  unsafe.Pointer

	configPathMu   sync.RWMutex
	configFilePath string
)

const defaultConfigFilename = "cortexgo.json"

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

	envelope := map[string]any{
		"event": event,
	}
	if data != nil {
		envelope["data"] = data
	} else {
		envelope["data"] = map[string]any{}
	}

	bytes, err := json.Marshal(envelope)
	if err != nil {
		fallback := map[string]any{
			"event": "log_error",
			"error": err.Error(),
		}
		bytes, _ = json.Marshal(fallback)
	}

	cstr := C.CString(string(bytes))
	defer C.free(unsafe.Pointer(cstr))
	C.call_log_callback(cb, cstr, ctx)
}

func goString(ptr *C.char) string {
	if ptr == nil {
		return ""
	}
	return C.GoString(ptr)
}

func ensureConfig() (*config.Config, error) {
	cfgMu.RLock()
	if activeCfg != nil {
		cfg := activeCfg
		cfgMu.RUnlock()
		return cfg, nil
	}
	cfgMu.RUnlock()

	cfg, err := loadInitialConfig()
	if err != nil {
		return nil, err
	}

	storeActiveConfig(cfg)
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

	if err := cfg.EnsureDirectories(); err != nil {
		return nil, err
	}

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

func storeActiveConfig(cfg *config.Config) {
	cfgMu.Lock()
	activeCfg = cfg
	cfgMu.Unlock()

	if err := persistActiveConfig(cfg); err != nil {
		fmt.Printf("failed to persist config: %v\n", err)
	}
}

func currentConfigCopy() (*config.Config, error) {
	if _, err := ensureConfig(); err != nil {
		return nil, err
	}

	cfgMu.RLock()
	defer cfgMu.RUnlock()

	if activeCfg == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}

	clone := *activeCfg
	return &clone, nil
}

func loadInitialConfig() (*config.Config, error) {
	if path := getConfigFilePath(); strings.TrimSpace(path) != "" {
		return loadConfigFromFile(path)
	}

	if envPath := strings.TrimSpace(os.Getenv("CORTEXGO_CONFIG_PATH")); envPath != "" {
		setConfigFilePathInternal(envPath)
		return loadConfigFromFile(envPath)
	}

	if defaultPath, ok := detectDefaultConfigFile(); ok {
		setConfigFilePathInternal(defaultPath)
		return loadConfigFromFile(defaultPath)
	}

	return buildConfig(nil, "")
}

func loadConfigFromFile(path string) (*config.Config, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("config path is empty")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			base := config.DefaultConfig()
			if base != nil {
				base.ProjectDir = filepath.Dir(path)
			}
			cfg, buildErr := buildConfig(base, "")
			if buildErr != nil {
				return nil, buildErr
			}
			if persistErr := persistConfigToPath(cfg, path); persistErr != nil {
				return nil, persistErr
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	base := config.DefaultConfig()
	if base != nil {
		base.ProjectDir = filepath.Dir(path)
	}
	return buildConfig(base, string(data))
}

func detectDefaultConfigFile() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	path := filepath.Join(cwd, defaultConfigFilename)
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	return "", false
}

func persistActiveConfig(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}

	path := getConfigFilePath()
	if strings.TrimSpace(path) == "" {
		resolved, err := resolveConfigPath(cfg)
		if err != nil {
			return err
		}
		path = resolved
	}

	return persistConfigToPath(cfg, path)
}

func persistConfigToPath(cfg *config.Config, path string) error {
	if cfg == nil {
		return nil
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("config path is empty")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func resolveConfigPath(cfg *config.Config) (string, error) {
	path := getConfigFilePath()
	if strings.TrimSpace(path) != "" {
		return path, nil
	}

	baseDir := strings.TrimSpace(cfg.ProjectDir)
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	resolved := filepath.Join(baseDir, defaultConfigFilename)
	setConfigFilePathInternal(resolved)
	return resolved, nil
}

func getConfigFilePath() string {
	configPathMu.RLock()
	path := configFilePath
	configPathMu.RUnlock()
	return path
}

func setConfigFilePathInternal(path string) {
	configPathMu.Lock()
	configFilePath = strings.TrimSpace(path)
	configPathMu.Unlock()
}

func setConfigFilePath(path string) (*config.Config, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		setConfigFilePathInternal("")
		cfg, err := ensureConfig()
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	absPath := trimmed
	if !filepath.IsAbs(absPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("determine working directory: %w", err)
		}
		absPath = filepath.Join(cwd, trimmed)
	}

	setConfigFilePathInternal(absPath)
	cfg, err := loadConfigFromFile(absPath)
	if err != nil {
		return nil, err
	}
	storeActiveConfig(cfg)
	return cfg, nil
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

	tradingGraph := graph.NewTradingAgentsGraphWithEmitter(cfg.Debug, cfg, emitToRegisteredCallback)
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

func encodeConfigResponse(resp configResponse) *C.char {
	data, err := json.Marshal(resp)
	if err != nil {
		fallback := map[string]any{
			"success": false,
			"error":   err.Error(),
		}
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
		fallback := map[string]any{
			"success": false,
			"error":   err.Error(),
		}
		payload, _ = json.Marshal(fallback)
	}

	return C.CString(string(payload))
}

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
	cfg, err := setConfigFilePath(goPath)
	if err != nil {
		return encodeConfigResponse(configResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	return encodeConfigResponse(configResponse{
		Success: true,
		Config:  cfg,
	})
}

//export CortexGoAnalyzeWithConfig
func CortexGoAnalyzeWithConfig(symbol *C.char, date *C.char, configJSON *C.char) *C.char {
	jsonPayload := goString(configJSON)
	if strings.TrimSpace(jsonPayload) != "" {
		base, err := currentConfigCopy()
		if err != nil {
			cfg, buildErr := buildConfig(nil, jsonPayload)
			if buildErr != nil {
				return encodeAnalysisPayload(&analysisResponse{
					Success: false,
					Error:   buildErr.Error(),
				})
			}
			storeActiveConfig(cfg)
		} else {
			cfg, err := buildConfig(base, jsonPayload)
			if err != nil {
				return encodeAnalysisPayload(&analysisResponse{
					Success: false,
					Error:   err.Error(),
				})
			}
			storeActiveConfig(cfg)
		}
	}

	return CortexGoAnalyze(symbol, date)
}

//export CortexGoSetConfigJSON
func CortexGoSetConfigJSON(configJSON *C.char) *C.char {
	jsonPayload := goString(configJSON)

	cfg, err := buildConfig(nil, jsonPayload)
	if err != nil {
		return encodeConfigResponse(configResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	storeActiveConfig(cfg)

	return encodeConfigResponse(configResponse{
		Success: true,
		Config:  cfg,
	})
}

//export CortexGoUpdateConfigJSON
func CortexGoUpdateConfigJSON(configJSON *C.char) *C.char {
	jsonPayload := goString(configJSON)

	current, err := currentConfigCopy()
	if err != nil {
		return encodeConfigResponse(configResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	cfg, err := buildConfig(current, jsonPayload)
	if err != nil {
		return encodeConfigResponse(configResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	storeActiveConfig(cfg)

	return encodeConfigResponse(configResponse{
		Success: true,
		Config:  cfg,
	})
}

//export CortexGoResetConfig
func CortexGoResetConfig() *C.char {
	cfg, err := buildConfig(nil, "")
	if err != nil {
		return encodeConfigResponse(configResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	storeActiveConfig(cfg)

	return encodeConfigResponse(configResponse{
		Success: true,
		Config:  cfg,
	})
}

//export CortexGoGetConfigJSON
func CortexGoGetConfigJSON() *C.char {
	cfg, err := currentConfigCopy()
	if err != nil {
		return encodeConfigResponse(configResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	return encodeConfigResponse(configResponse{
		Success: true,
		Config:  cfg,
	})
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

	return encodeAnalysisPayload(resp)
}

//export CortexGoFree
func CortexGoFree(ptr *C.char) {
	if ptr != nil {
		C.free(unsafe.Pointer(ptr))
	}
}

//export CortexGoVersion
func CortexGoVersion() *C.char {
	return C.CString("v0.0.1")
}

func main() {}
