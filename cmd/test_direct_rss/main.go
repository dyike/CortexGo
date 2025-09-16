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
		CacheEnabled: false,
		DataCacheDir: "./cache",
		DataDir:      "./data",
	}

	// Create Google News client
	googleNewsClient := dataflows.NewGoogleNewsClient(cfg)

	fmt.Println("🔍 测试直接RSS源获取真实摘要")
	fmt.Println("==============================")
	fmt.Println()

	// Test 1: 获取AI相关新闻
	fmt.Println("=== Test 1: AI新闻 (直接RSS源) ===")
	articles1, err := googleNewsClient.GetDirectNewsRSS("artificial intelligence", 5, cfg)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		for i, article := range articles1 {
			fmt.Printf("\n%d. 标题: %s\n", i+1, article.Title)
			fmt.Printf("   来源: %s\n", article.Source)
			fmt.Printf("   时间: %s\n", article.PublishedAt.Format("2006-01-02 15:04"))
			if len(article.Content) > 200 {
				fmt.Printf("   摘要: %s...\n", article.Content[:200])
			} else {
				fmt.Printf("   摘要: %s\n", article.Content)
			}
		}
	}
	fmt.Println()

	// Test 2: 获取股市新闻
	fmt.Println("=== Test 2: 股市新闻 (直接RSS源) ===")
	articles2, err := googleNewsClient.GetDirectNewsRSS("stock market", 3, cfg)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		for i, article := range articles2 {
			fmt.Printf("\n%d. 标题: %s\n", i+1, article.Title)
			fmt.Printf("   来源: %s\n", article.Source)
			if len(article.Content) > 150 {
				fmt.Printf("   摘要: %s...\n", article.Content[:150])
			} else {
				fmt.Printf("   摘要: %s\n", article.Content)
			}
		}
	}
	fmt.Println()

	// Test 3: 获取科技新闻
	fmt.Println("=== Test 3: 科技新闻 (直接RSS源) ===")
	articles3, err := googleNewsClient.GetDirectNewsRSS("technology", 3, cfg)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		for i, article := range articles3 {
			fmt.Printf("\n%d. 标题: %s\n", i+1, article.Title)
			fmt.Printf("   来源: %s\n", article.Source)
			if len(article.Content) > 150 {
				fmt.Printf("   摘要: %s...\n", article.Content[:150])
			} else {
				fmt.Printf("   摘要: %s\n", article.Content)
			}
		}
	}

	fmt.Println()
	fmt.Println("📊 直接RSS源测试总结:")
	fmt.Println("====================")
	fmt.Printf("• AI新闻: %d篇\n", len(articles1))
	fmt.Printf("• 股市新闻: %d篇\n", len(articles2))
	fmt.Printf("• 科技新闻: %d篇\n", len(articles3))
	fmt.Printf("• 总计: %d篇\n", len(articles1)+len(articles2)+len(articles3))

	fmt.Println()
	fmt.Println("✅ 直接RSS源的优势:")
	fmt.Println("=================")
	fmt.Println("• 真实的文章摘要内容")
	fmt.Println("• 来自权威新闻源")
	fmt.Println("• 更丰富的上下文信息")
	fmt.Println("• 适合AI情感分析和内容理解")
	fmt.Println("• 无需额外的内容提取步骤")
}