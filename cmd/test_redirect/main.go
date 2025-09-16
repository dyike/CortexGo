package main

import (
	"fmt"
	"log"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/pkg/dataflows"
)

func main() {
	// Create configuration
	cfg := &config.Config{
		OnlineTools:  true,
		CacheEnabled: true,
		DataCacheDir: "./cache",
		DataDir:      "./data",
	}

	// Create Google News client
	googleNewsClient := dataflows.NewGoogleNewsClient(cfg)

	// Test with a specific Google News URL
	testURL := "https://news.google.com/read/CBMiS0FVX3lxTE5mYUtjcnRiMXRCVXo0c2R3VDlYdGptLUREdzFRRDdGbDBLX0FVaGlFaXhMX3AwY3dVb1IzbHk2MGRpYTYwWkJpbTRtQQ?hl=en-US&gl=US&ceid=US%3Aen"

	fmt.Println("🔍 Testing Google News Redirect and Content Extraction")
	fmt.Printf("Testing URL: %s\n\n", testURL)

	content, err := googleNewsClient.GetArticleContent(testURL)
	if err != nil {
		log.Printf("Error extracting content: %v", err)
	} else {
		fmt.Printf("✅ Successfully extracted content (%d chars):\n", len(content))
		if len(content) > 300 {
			fmt.Printf("%s...\n", content[:300])
		} else {
			fmt.Printf("%s\n", content)
		}
	}
}