package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ProjectDir   string `json:"project_dir"`
	ResultsDir   string `json:"results_dir"`
	DataDir      string `json:"data_dir"`
	DataCacheDir string `json:"data_cache_dir"`

	LLMProvider   string `json:"llm_provider"`
	DeepThinkLLM  string `json:"deep_think_llm"`
	QuickThinkLLM string `json:"quick_think_llm"`
	BackendURL    string `json:"backend_url"`

	MaxDebateRounds      int `json:"max_debate_rounds"`
	MaxRiskDiscussRounds int `json:"max_risk_discuss_rounds"`
	MaxRecurLimit        int `json:"max_recur_limit"`

	OnlineTools bool `json:"online_tools"`
	Debug       bool `json:"debug"`

	// Dataflows configuration
	FinnhubAPIKey   string `json:"finnhub_api_key"`
	RedditClientID  string `json:"reddit_client_id"`
	RedditSecret    string `json:"reddit_secret"`
	RedditUserAgent string `json:"reddit_user_agent"`
	CacheEnabled    bool   `json:"cache_enabled"`
}

func DefaultConfig() *Config {
	currentDir, _ := os.Getwd()

	cfg := &Config{
		ProjectDir:   currentDir,
		ResultsDir:   filepath.Join(currentDir, "results"),
		DataDir:      filepath.Join(currentDir, "data"),
		DataCacheDir: filepath.Join(currentDir, "data", "cache"),

		LLMProvider:   "openai",
		DeepThinkLLM:  "o1-mini",
		QuickThinkLLM: "gpt-4o-mini",
		BackendURL:    "https://api.openai.com/v1",

		MaxDebateRounds:      1,
		MaxRiskDiscussRounds: 1,
		MaxRecurLimit:        100,

		OnlineTools: true,
		Debug:       false,

		// Dataflows defaults
		FinnhubAPIKey:   "",
		RedditClientID:  "",
		RedditSecret:    "",
		RedditUserAgent: "CortexGo/1.0",
		CacheEnabled:    true,
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

	if val := os.Getenv("MAX_DEBATE_ROUNDS"); val != "" {
		if rounds, err := strconv.Atoi(val); err == nil {
			c.MaxDebateRounds = rounds
		}
	}
	if val := os.Getenv("MAX_RISK_DISCUSS_ROUNDS"); val != "" {
		if rounds, err := strconv.Atoi(val); err == nil {
			c.MaxRiskDiscussRounds = rounds
		}
	}
	if val := os.Getenv("MAX_RECUR_LIMIT"); val != "" {
		if limit, err := strconv.Atoi(val); err == nil {
			c.MaxRecurLimit = limit
		}
	}

	if val := os.Getenv("ONLINE_TOOLS"); val != "" {
		if online, err := strconv.ParseBool(val); err == nil {
			c.OnlineTools = online
		}
	}
	if val := os.Getenv("DEBUG"); val != "" {
		if debug, err := strconv.ParseBool(val); err == nil {
			c.Debug = debug
		}
	}

	if val := os.Getenv("FINNHUB_API_KEY"); val != "" {
		c.FinnhubAPIKey = val
	}
	if val := os.Getenv("REDDIT_CLIENT_ID"); val != "" {
		c.RedditClientID = val
	}
	if val := os.Getenv("REDDIT_SECRET"); val != "" {
		c.RedditSecret = val
	}
	if val := os.Getenv("REDDIT_USER_AGENT"); val != "" {
		c.RedditUserAgent = val
	}
	if val := os.Getenv("CACHE_ENABLED"); val != "" {
		if cache, err := strconv.ParseBool(val); err == nil {
			c.CacheEnabled = cache
		}
	}
}

func (c *Config) EnsureDirectories() error {
	dirs := []string{c.ResultsDir, c.DataDir, c.DataCacheDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create dataflows subdirectories
	dataflowsSubdirs := []string{
		"market_data/price_data",
		"finnhub_data",
		"reddit_data",
		"news_data",
		"fundamental_data",
	}
	
	for _, subdir := range dataflowsSubdirs {
		fullPath := filepath.Join(c.DataDir, subdir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return err
		}
	}

	return nil
}
