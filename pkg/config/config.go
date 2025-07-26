package config

import (
	"os"
	"path/filepath"
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
}

func DefaultConfig() *Config {
	currentDir, _ := os.Getwd()

	return &Config{
		ProjectDir:   currentDir,
		ResultsDir:   filepath.Join(currentDir, "results"),
		DataDir:      filepath.Join(currentDir, "data"),
		DataCacheDir: filepath.Join(currentDir, "data", "cache"),

		LLMProvider:   "openai",
		DeepThinkLLM:  "o4-mini",
		QuickThinkLLM: "gpt-4o-mini",
		BackendURL:    "https://api.openai.com/v1",

		MaxDebateRounds:      1,
		MaxRiskDiscussRounds: 1,
		MaxRecurLimit:        100,

		OnlineTools: true,
		Debug:       false,
	}
}

func (c *Config) EnsureDirectories() error {
	dirs := []string{c.ResultsDir, c.DataDir, c.DataCacheDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
