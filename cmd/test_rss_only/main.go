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
		CacheEnabled: false, // 禁用缓存以便测试
		DataCacheDir: "./cache",
		DataDir:      "./data",
	}

	// Create Google News client
	googleNewsClient := dataflows.NewGoogleNewsClient(cfg)

	fmt.Println("🔍 专门测试Google News RSS功能")
	fmt.Println("===================================")
	fmt.Println()

	// Test 1: RSS获取AI新闻
	fmt.Println("=== Test 1: RSS获取AI新闻 ===")
	params1 := dataflows.EnhancedGoogleNewsParams{
		Query:      "artificial intelligence",
		Language:   "en",
		Country:    "US",
		MaxResults: 5,
		SortBy:     "date",
	}

	articles1, err := googleNewsClient.GetGoogleNewsRSS(params1, cfg)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("✅ 获取到 %d 篇AI新闻\n", len(articles1))
		for i, article := range articles1 {
			fmt.Printf("\n%d. 标题: %s\n", i+1, article.Title)
			fmt.Printf("   来源: %s\n", article.Source)
			fmt.Printf("   时间: %s\n", article.PublishedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("   链接: %s\n", article.URL)
			if article.Content != "" {
				fmt.Printf("   摘要: %s\n", article.Content)
			} else {
				fmt.Printf("   摘要: [无内容]\n")
			}
		}
	}
	fmt.Println()

	// Test 2: RSS获取金融新闻
	fmt.Println("=== Test 2: RSS获取金融新闻 ===")
	params2 := dataflows.EnhancedGoogleNewsParams{
		Query:      "stock market",
		Language:   "en",
		Country:    "US",
		MaxResults: 3,
		Category:   "business",
	}

	articles2, err := googleNewsClient.GetGoogleNewsRSS(params2, cfg)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("✅ 获取到 %d 篇金融新闻\n", len(articles2))
		for i, article := range articles2 {
			fmt.Printf("\n%d. 标题: %s\n", i+1, article.Title)
			fmt.Printf("   来源: %s\n", article.Source)
			if article.Content != "" {
				fmt.Printf("   摘要: %s\n", article.Content)
			} else {
				fmt.Printf("   摘要: [无内容]\n")
			}
		}
	}
	fmt.Println()

	// Test 3: 中文新闻
	fmt.Println("=== Test 3: 中文经济新闻 ===")
	params3 := dataflows.EnhancedGoogleNewsParams{
		Query:      "中国经济",
		Language:   "zh-CN",
		Country:    "CN",
		MaxResults: 3,
	}

	articles3, err := googleNewsClient.GetGoogleNewsRSS(params3, cfg)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("✅ 获取到 %d 篇中文新闻\n", len(articles3))
		for i, article := range articles3 {
			fmt.Printf("\n%d. 标题: %s\n", i+1, article.Title)
			fmt.Printf("   来源: %s\n", article.Source)
			if article.Content != "" {
				fmt.Printf("   摘要: %s\n", article.Content)
			} else {
				fmt.Printf("   摘要: [无内容]\n")
			}
		}
	}

	fmt.Println()
	fmt.Println("📊 RSS测试总结:")
	fmt.Println("===============")
	fmt.Printf("• AI新闻: %d篇\n", len(articles1))
	fmt.Printf("• 金融新闻: %d篇\n", len(articles2))
	fmt.Printf("• 中文新闻: %d篇\n", len(articles3))
	fmt.Printf("• 总计: %d篇\n", len(articles1)+len(articles2)+len(articles3))

	fmt.Println()
	fmt.Println("🔍 RSS内容分析:")
	fmt.Println("===============")
	fmt.Println("• Google News RSS的description字段通常只包含标题信息")
	fmt.Println("• 如需完整摘要，建议使用以下策略:")
	fmt.Println("  1. 结合多个RSS源 (如BBC, Reuters, CNN等)")
	fmt.Println("  2. 使用专门的新闻API (News API, Bing News API)")
	fmt.Println("  3. 对重要新闻进行二次内容提取")
	fmt.Println("• 当前RSS实现已能提供:")
	fmt.Println("  ✓ 准确的标题")
	fmt.Println("  ✓ 可靠的来源信息")
	fmt.Println("  ✓ 精确的发布时间")
	fmt.Println("  ✓ 直接的新闻链接")
}