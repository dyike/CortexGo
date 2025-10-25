package config

import (
	"fmt"
	"testing"
)

func TestConfigFromEnv(t *testing.T) {
	cfg := LoadConfigFromEnv()
	fmt.Println(cfg)
}
