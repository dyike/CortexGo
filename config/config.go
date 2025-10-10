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
}

func DefaultConfig() *Config {
	currentDir, _ := os.Getwd()

	cfg := &Config{
		ProjectDir:   currentDir,
		ResultsDir:   filepath.Join(currentDir, "results"),
		DataDir:      filepath.Join(currentDir, "data"),
		DataCacheDir: filepath.Join(currentDir, "data", "cache"),

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

	if val := os.Getenv("CACHE_ENABLED"); val != "" {
		if cache, err := strconv.ParseBool(val); err == nil {
			c.CacheEnabled = cache
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
}
