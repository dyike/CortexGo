package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ProjectDir   string `json:"project_dir"`
	ResultsDir   string `json:"results_dir"`
	DataDir      string `json:"data_dir"`
	DataCacheDir string `json:"data_cache_dir"`

	LLMProvider          string `json:"llm_provider"`
	DeepThinkLLM         string `json:"deep_think_llm"`
	QuickThinkLLM        string `json:"quick_think_llm"`
	BackendURL           string `json:"backend_url"`
	MaxDebateRounds      int    `json:"max_debate_rounds"`
	MaxRiskDiscussRounds int    `json:"max_risk_rounds"`
	MaxRecurLimit        int    `json:"max_recursion_limit"`
	OnlineTools          bool   `json:"online_tools"`
	Debug                bool   `json:"debug"`
	RedditUserAgent      string `json:"reddit_user_agent"`

	// Eino Debug configuration
	EinoDebugEnabled bool `json:"eino_debug_enabled"`
	EinoDebugPort    int  `json:"eino_debug_port"`

	CacheEnabled bool `json:"cache_enabled"`

	// Longport API Configuration
	LongportAppKey      string `json:"longport_app_key"`
	LongportAppSecret   string `json:"longport_app_secret"`
	LongportAccessToken string `json:"longport_access_token"`

	// AI Model API Keys
	DeepSeekAPIKey string `json:"deepseek_api_key"`

	// Market/Social data API keys
	FinnhubAPIKey  string `json:"finnhub_api_key"`
	RedditClientID string `json:"reddit_client_id"`
	RedditSecret   string `json:"reddit_secret"`
}

func DefaultConfig() *Config {
	currentDir, _ := os.Getwd()

	cfg := &Config{
		ProjectDir:   currentDir,
		ResultsDir:   filepath.Join(currentDir, "results"),
		DataDir:      filepath.Join(currentDir, "data"),
		DataCacheDir: filepath.Join(currentDir, "data", "cache"),

		LLMProvider:   "deepseek",
		DeepThinkLLM:  "deepseek-chat",
		QuickThinkLLM: "deepseek-chat",
		BackendURL:    "",

		MaxDebateRounds:      3,
		MaxRiskDiscussRounds: 3,
		MaxRecurLimit:        128,
		OnlineTools:          true,
		Debug:                false,
		RedditUserAgent:      "CortexGo/1.0",

		// Eino Debug defaults
		EinoDebugEnabled: false,
		EinoDebugPort:    52538,

		CacheEnabled: true,
	}

	// Load environment variables from .env file
	_ = godotenv.Load()

	// Override with environment variables if they exist
	cfg.loadFromEnv()

	return cfg
}

func (c *Config) loadFromEnv() {
	if val := os.Getenv("PROJECT_DIR"); val != "" {
		c.ProjectDir = val
	}
	if val := os.Getenv("RESULTS_DIR"); val != "" {
		c.ResultsDir = val
	}
	if val := os.Getenv("DATA_DIR"); val != "" {
		c.DataDir = val
	}
	if val := os.Getenv("DATA_CACHE_DIR"); val != "" {
		c.DataCacheDir = val
	}

	if val := os.Getenv("LLM_PROVIDER"); val != "" {
		c.LLMProvider = val
	}
	if val := os.Getenv("DEEP_THINK_LLM"); val != "" {
		c.DeepThinkLLM = val
	}
	if val := os.Getenv("QUICK_THINK_LLM"); val != "" {
		c.QuickThinkLLM = val
	}
	if val := os.Getenv("BACKEND_URL"); val != "" {
		c.BackendURL = val
	}

	if val := os.Getenv("CACHE_ENABLED"); val != "" {
		if cache, err := strconv.ParseBool(val); err == nil {
			c.CacheEnabled = cache
		}
	}

	if val := os.Getenv("ONLINE_TOOLS"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			c.OnlineTools = enabled
		}
	}

	if val := os.Getenv("MAX_DEBATE_ROUNDS"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			c.MaxDebateRounds = v
		}
	}
	if val := os.Getenv("MAX_RISK_ROUNDS"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			c.MaxRiskDiscussRounds = v
		}
	}
	if val := os.Getenv("MAX_RECURSION_LIMIT"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			c.MaxRecurLimit = v
		}
	}

	if val := os.Getenv("CORTEXGO_DEBUG"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			c.Debug = enabled
		}
	}

	if val := os.Getenv("EINO_DEBUG_ENABLED"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			c.EinoDebugEnabled = enabled
		}
	}
	if val := os.Getenv("EINO_DEBUG_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			c.EinoDebugPort = port
		}
	}

	if val := os.Getenv("LONGPORT_APP_KEY"); val != "" {
		c.LongportAppKey = val
	}
	if val := os.Getenv("LONGPORT_APP_SECRET"); val != "" {
		c.LongportAppSecret = val
	}
	if val := os.Getenv("LONGPORT_ACCESS_TOKEN"); val != "" {
		c.LongportAccessToken = val
	}

	if val := os.Getenv("DEEPSEEK_API_KEY"); val != "" {
		c.DeepSeekAPIKey = val
	}
	if val := os.Getenv("CORTEXGO_FINNHUB_API_KEY"); val != "" {
		c.FinnhubAPIKey = val
	}
	if val := os.Getenv("CORTEXGO_REDDIT_CLIENT_ID"); val != "" {
		c.RedditClientID = val
	}
	if val := os.Getenv("CORTEXGO_REDDIT_SECRET"); val != "" {
		c.RedditSecret = val
	}
	if val := os.Getenv("REDDIT_USER_AGENT"); val != "" {
		c.RedditUserAgent = val
	}
}

func (c *Config) EnsureDirectories() error {
	dirs := []string{c.ProjectDir, c.ResultsDir, c.DataDir, c.DataCacheDir}
	for _, dir := range dirs {
		path := strings.TrimSpace(dir)
		if path == "" {
			continue
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", path, err)
		}
	}
	return nil
}
