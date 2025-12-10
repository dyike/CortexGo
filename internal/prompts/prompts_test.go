package prompts

import (
	"testing"
)

func TestLoadPrompt(t *testing.T) {
	val, err := LoadPrompt("analysts/market_analyst")
	t.Log("val", val)
	t.Log("err", err)
}
