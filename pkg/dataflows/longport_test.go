package dataflows

import (
	"context"
	"testing"

	"github.com/dyike/CortexGo/config"
)

func TestLongportClient_GetStaticInfo(t *testing.T) {
	// Load configuration from environment
	cfg := config.DefaultConfig()
	longportConf := LongportConfig{
		AppKey:      cfg.LongportAppKey,
		AppSecret:   cfg.LongportAppSecret,
		AccessToken: cfg.LongportAccessToken,
	}

	// Create Longport client
	client, err := NewLongportClient(longportConf)
	if err != nil {
		t.Skipf("Skipping test due to missing Longport API credentials: %v", err)
	}

	// Test symbols (Hong Kong stock codes)
	symbols := []string{"700.HK", "9988.HK", "1299.HK"} // Tencent, Alibaba, AIA

	// Test GetStaticInfo
	ctx := context.Background()
	staticInfos, err := client.GetStaticInfo(ctx, symbols)
	if err != nil {
		t.Fatalf("GetStaticInfo failed: %v", err)
	}

	// Verify results
	if len(staticInfos) == 0 {
		t.Error("Expected non-empty static info results")
	}

	for i, info := range staticInfos {
		if info == nil {
			t.Errorf("Static info at index %d is nil", i)
			continue
		}

		t.Logf("Symbol: %s", info.Symbol)
		t.Logf("NameCn: %s", info.NameCn)
		t.Logf("NameEn: %s", info.NameEn)
		t.Logf("Exchange: %s", info.Exchange)
		t.Logf("Currency: %s", info.Currency)
		t.Logf("LotSize: %d", info.LotSize)
	}
}
