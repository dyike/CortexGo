package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/display"
)

// ConfigManager handles advanced configuration management
type ConfigManager struct {
	config     *config.Config
	configPath string
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(cfg *config.Config) *ConfigManager {
	return &ConfigManager{
		config:     cfg,
		configPath: filepath.Join(cfg.ProjectDir, "cortexgo.json"),
	}
}

// SaveConfig saves the current configuration to file
func (cm *ConfigManager) SaveConfig() error {
	// Create a serializable config structure
	configData := map[string]interface{}{
		"llm_provider":            cm.config.LLMProvider,
		"deep_think_llm":         cm.config.DeepThinkLLM,
		"quick_think_llm":        cm.config.QuickThinkLLM,
		"backend_url":            cm.config.BackendURL,
		"max_debate_rounds":      cm.config.MaxDebateRounds,
		"max_risk_rounds":        cm.config.MaxRiskDiscussRounds,
		"max_recursion_limit":    cm.config.MaxRecurLimit,
		"online_tools":           cm.config.OnlineTools,
		"cache_enabled":          cm.config.CacheEnabled,
		"debug":                  cm.config.Debug,
		"eino_debug_enabled":     cm.config.EinoDebugEnabled,
		"eino_debug_port":        cm.config.EinoDebugPort,
		"reddit_user_agent":      cm.config.RedditUserAgent,
	}

	jsonData, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return ioutil.WriteFile(cm.configPath, jsonData, 0644)
}

// LoadConfig loads configuration from file
func (cm *ConfigManager) LoadConfig() error {
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		display.DisplayInfo("No configuration file found, using defaults")
		return nil
	}

	data, err := ioutil.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply loaded configuration
	if val, ok := configData["llm_provider"].(string); ok {
		cm.config.LLMProvider = val
	}
	if val, ok := configData["deep_think_llm"].(string); ok {
		cm.config.DeepThinkLLM = val
	}
	if val, ok := configData["quick_think_llm"].(string); ok {
		cm.config.QuickThinkLLM = val
	}
	if val, ok := configData["backend_url"].(string); ok {
		cm.config.BackendURL = val
	}
	if val, ok := configData["max_debate_rounds"].(float64); ok {
		cm.config.MaxDebateRounds = int(val)
	}
	if val, ok := configData["max_risk_rounds"].(float64); ok {
		cm.config.MaxRiskDiscussRounds = int(val)
	}
	if val, ok := configData["max_recursion_limit"].(float64); ok {
		cm.config.MaxRecurLimit = int(val)
	}
	if val, ok := configData["online_tools"].(bool); ok {
		cm.config.OnlineTools = val
	}
	if val, ok := configData["cache_enabled"].(bool); ok {
		cm.config.CacheEnabled = val
	}
	if val, ok := configData["debug"].(bool); ok {
		cm.config.Debug = val
	}
	if val, ok := configData["eino_debug_enabled"].(bool); ok {
		cm.config.EinoDebugEnabled = val
	}
	if val, ok := configData["eino_debug_port"].(float64); ok {
		cm.config.EinoDebugPort = int(val)
	}
	if val, ok := configData["reddit_user_agent"].(string); ok {
		cm.config.RedditUserAgent = val
	}

	display.DisplaySuccess("Configuration loaded from file")
	return nil
}

// ResetConfig resets configuration to defaults
func (cm *ConfigManager) ResetConfig() error {
	defaultConfig := config.DefaultConfig()
	*cm.config = *defaultConfig
	display.DisplaySuccess("Configuration reset to defaults")
	return nil
}

// ExportConfig exports configuration to a specified file
func (cm *ConfigManager) ExportConfig(filename string) error {
	configData := map[string]interface{}{
		"metadata": map[string]string{
			"version":      "1.0.0",
			"exported_at":  fmt.Sprintf("%d", os.Getpid()),
			"description":  "CortexGo Configuration Export",
		},
		"llm_provider":            cm.config.LLMProvider,
		"deep_think_llm":         cm.config.DeepThinkLLM,
		"quick_think_llm":        cm.config.QuickThinkLLM,
		"backend_url":            cm.config.BackendURL,
		"max_debate_rounds":      cm.config.MaxDebateRounds,
		"max_risk_rounds":        cm.config.MaxRiskDiscussRounds,
		"max_recursion_limit":    cm.config.MaxRecurLimit,
		"online_tools":           cm.config.OnlineTools,
		"cache_enabled":          cm.config.CacheEnabled,
		"debug":                  cm.config.Debug,
		"eino_debug_enabled":     cm.config.EinoDebugEnabled,
		"eino_debug_port":        cm.config.EinoDebugPort,
		"reddit_user_agent":      cm.config.RedditUserAgent,
		"directories": map[string]string{
			"project_dir":     cm.config.ProjectDir,
			"results_dir":     cm.config.ResultsDir,
			"data_dir":        cm.config.DataDir,
			"data_cache_dir":  cm.config.DataCacheDir,
		},
	}

	jsonData, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return ioutil.WriteFile(filename, jsonData, 0644)
}

// ImportConfig imports configuration from a specified file
func (cm *ConfigManager) ImportConfig(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate config version if available
	if metadata, ok := configData["metadata"].(map[string]interface{}); ok {
		if version, ok := metadata["version"].(string); ok {
			display.DisplayInfo(fmt.Sprintf("Importing configuration version: %s", version))
		}
	}

	// Apply imported configuration (same logic as LoadConfig)
	return cm.LoadConfig()
}

// GetConfigValue gets a configuration value by key
func (cm *ConfigManager) GetConfigValue(key string) (interface{}, error) {
	switch strings.ToLower(key) {
	case "llm_provider":
		return cm.config.LLMProvider, nil
	case "deep_think_llm":
		return cm.config.DeepThinkLLM, nil
	case "quick_think_llm":
		return cm.config.QuickThinkLLM, nil
	case "backend_url":
		return cm.config.BackendURL, nil
	case "max_debate_rounds":
		return cm.config.MaxDebateRounds, nil
	case "max_risk_rounds":
		return cm.config.MaxRiskDiscussRounds, nil
	case "max_recursion_limit":
		return cm.config.MaxRecurLimit, nil
	case "online_tools":
		return cm.config.OnlineTools, nil
	case "cache_enabled":
		return cm.config.CacheEnabled, nil
	case "debug":
		return cm.config.Debug, nil
	case "eino_debug_enabled":
		return cm.config.EinoDebugEnabled, nil
	case "eino_debug_port":
		return cm.config.EinoDebugPort, nil
	case "reddit_user_agent":
		return cm.config.RedditUserAgent, nil
	case "project_dir":
		return cm.config.ProjectDir, nil
	case "results_dir":
		return cm.config.ResultsDir, nil
	case "data_dir":
		return cm.config.DataDir, nil
	case "data_cache_dir":
		return cm.config.DataCacheDir, nil
	default:
		return nil, fmt.Errorf("unknown configuration key: %s", key)
	}
}

// SetConfigValue sets a configuration value by key
func (cm *ConfigManager) SetConfigValue(key, value string) error {
	switch strings.ToLower(key) {
	case "llm_provider":
		validProviders := []string{"openai", "deepseek", "anthropic"}
		if !contains(validProviders, value) {
			return fmt.Errorf("invalid LLM provider. Valid options: %s", strings.Join(validProviders, ", "))
		}
		cm.config.LLMProvider = value
		
	case "deep_think_llm":
		cm.config.DeepThinkLLM = value
		
	case "quick_think_llm":
		cm.config.QuickThinkLLM = value
		
	case "backend_url":
		cm.config.BackendURL = value
		
	case "max_debate_rounds":
		if i, err := strconv.Atoi(value); err == nil && i >= 1 && i <= 10 {
			cm.config.MaxDebateRounds = i
		} else {
			return fmt.Errorf("max_debate_rounds must be between 1-10")
		}
		
	case "max_risk_rounds":
		if i, err := strconv.Atoi(value); err == nil && i >= 1 && i <= 10 {
			cm.config.MaxRiskDiscussRounds = i
		} else {
			return fmt.Errorf("max_risk_rounds must be between 1-10")
		}
		
	case "max_recursion_limit":
		if i, err := strconv.Atoi(value); err == nil && i >= 10 && i <= 1000 {
			cm.config.MaxRecurLimit = i
		} else {
			return fmt.Errorf("max_recursion_limit must be between 10-1000")
		}
		
	case "online_tools":
		if b, err := strconv.ParseBool(value); err == nil {
			cm.config.OnlineTools = b
		} else {
			return fmt.Errorf("online_tools must be true or false")
		}
		
	case "cache_enabled":
		if b, err := strconv.ParseBool(value); err == nil {
			cm.config.CacheEnabled = b
		} else {
			return fmt.Errorf("cache_enabled must be true or false")
		}
		
	case "debug":
		if b, err := strconv.ParseBool(value); err == nil {
			cm.config.Debug = b
		} else {
			return fmt.Errorf("debug must be true or false")
		}
		
	case "eino_debug_enabled":
		if b, err := strconv.ParseBool(value); err == nil {
			cm.config.EinoDebugEnabled = b
		} else {
			return fmt.Errorf("eino_debug_enabled must be true or false")
		}
		
	case "eino_debug_port":
		if i, err := strconv.Atoi(value); err == nil && i >= 1024 && i <= 65535 {
			cm.config.EinoDebugPort = i
		} else {
			return fmt.Errorf("eino_debug_port must be between 1024-65535")
		}
		
	case "reddit_user_agent":
		cm.config.RedditUserAgent = value
		
	default:
		return fmt.Errorf("unknown or read-only configuration key: %s", key)
	}
	
	return nil
}

// ValidateConfiguration validates the current configuration
func (cm *ConfigManager) ValidateConfiguration() []string {
	var warnings []string
	
	// Check required environment variables based on provider
	switch cm.config.LLMProvider {
	case "openai":
		if os.Getenv("OPENAI_API_KEY") == "" {
			warnings = append(warnings, "OPENAI_API_KEY environment variable not set")
		}
	case "deepseek":
		if cm.config.DeepSeekAPIKey == "" && os.Getenv("DEEPSEEK_API_KEY") == "" {
			warnings = append(warnings, "DEEPSEEK_API_KEY not configured")
		}
	case "anthropic":
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			warnings = append(warnings, "ANTHROPIC_API_KEY environment variable not set")
		}
	}
	
	// Check API keys for data sources
	if cm.config.FinnhubAPIKey == "" && os.Getenv("CORTEXGO_FINNHUB_API_KEY") == "" {
		warnings = append(warnings, "Finnhub API key not configured - market data may be limited")
	}
	
	if (cm.config.RedditClientID == "" || cm.config.RedditSecret == "") && 
	   (os.Getenv("CORTEXGO_REDDIT_CLIENT_ID") == "" || os.Getenv("CORTEXGO_REDDIT_SECRET") == "") {
		warnings = append(warnings, "Reddit API credentials not configured - social sentiment may be limited")
	}
	
	// Check directory permissions
	dirs := []string{cm.config.ResultsDir, cm.config.DataDir, cm.config.DataCacheDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			warnings = append(warnings, fmt.Sprintf("Cannot create/access directory: %s", dir))
		}
	}
	
	// Validate numeric ranges
	if cm.config.MaxDebateRounds < 1 || cm.config.MaxDebateRounds > 10 {
		warnings = append(warnings, "max_debate_rounds should be between 1-10")
	}
	
	if cm.config.MaxRiskDiscussRounds < 1 || cm.config.MaxRiskDiscussRounds > 10 {
		warnings = append(warnings, "max_risk_rounds should be between 1-10")
	}
	
	if cm.config.EinoDebugPort < 1024 || cm.config.EinoDebugPort > 65535 {
		warnings = append(warnings, "eino_debug_port should be between 1024-65535")
	}
	
	return warnings
}

// ListAvailableKeys returns all available configuration keys
func (cm *ConfigManager) ListAvailableKeys() []string {
	return []string{
		"llm_provider", "deep_think_llm", "quick_think_llm", "backend_url",
		"max_debate_rounds", "max_risk_rounds", "max_recursion_limit",
		"online_tools", "cache_enabled", "debug", 
		"eino_debug_enabled", "eino_debug_port", "reddit_user_agent",
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}