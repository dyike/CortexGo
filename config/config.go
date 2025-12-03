package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"

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
	CacheEnabled     bool `json:"cache_enabled"`

	// Longport API Configuration
	LongportAppKey      string `json:"longport_app_key"`
	LongportAppSecret   string `json:"longport_app_secret"`
	LongportAccessToken string `json:"longport_access_token"`

	// AI Model API Keys
	DeepSeekAPIKey string `json:"deepseek_api_key"`
}

var (
	globalCfg Config
	mu        sync.RWMutex
	ConfigDir string
)

func Initialize(dir string, jsonStr string) error {
	ConfigDir = dir
	return Update(jsonStr)
}

func Update(jsonStr string) error {
	var newCfg Config
	if err := json.Unmarshal([]byte(jsonStr), &newCfg); err != nil {
		return err
	}
	mu.Lock()
	globalCfg = newCfg
	mu.Unlock()
	return nil
}

func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	return globalCfg
}

func LoadConfigFromEnv() *Config {
	cfg := &Config{}
	_ = godotenv.Load()
	cfg.loadFromEnv()
	return cfg
}

func LoadConfigFromJsonFile(path string) *Config {
	cfg := &Config{}
	if err := loadConfigFromFile(path, cfg); err != nil {
		panic(err)
	}
	return cfg
}

func LoadConfigFromJsonContent(content string) *Config {
	cfg := &Config{}
	if err := json.Unmarshal([]byte(content), cfg); err != nil {
		panic(err)
	}
	return cfg
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

func loadConfigFromFile(filePath string, cfg *Config) error {
	if _, err := os.Stat(filePath); err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return err
	}

	return nil
}
