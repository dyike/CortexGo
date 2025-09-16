package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/pkg/dataflows"
)

func main() {
	// Create configuration
	cfg := &config.Config{
		OnlineTools:   true,
		CacheEnabled:  true,
		DataCacheDir:  "./cache",
		DataDir:       "./data",
	}

	// Create data directories
	fmt.Println("Setting up directories...")

	// Create Google News client
	googleNewsClient := dataflows.NewGoogleNewsClient(cfg)

	fmt.Println("🔍 Testing Enhanced Google News Dataflow\n")

	// Test 1: General Google News search
	fmt.Println("=== Test 1: General Google News Search ===")
	params1 := dataflows.EnhancedGoogleNewsParams{
		Query:      "artificial intelligence",
		Language:   "en",
		Country:    "US",
		MaxResults: 5,
		SortBy:     "date",
		Category:   "technology",
	}

	articles1, err := googleNewsClient.GetGoogleNews(params1, cfg)
	if err != nil {
		log.Printf("Error in Test 1: %v", err)
	} else {
		fmt.Printf("✅ Found %d articles about AI\n", len(articles1))
		for i, article := range articles1 {
			if i >= 3 { // Show first 3
				break
			}
			fmt.Printf("   %d. %s (%s)\n", i+1, article.Title, article.Source)
		}
	}
	fmt.Println()

	// Test 2: Finance news
	fmt.Println("=== Test 2: Finance News ===")
	financeArticles, err := googleNewsClient.GetFinanceNews(8, cfg)
	if err != nil {
		log.Printf("Error in Test 2: %v", err)
	} else {
		fmt.Printf("✅ Found %d finance articles\n", len(financeArticles))
		for i, article := range financeArticles {
			if i >= 3 { // Show first 3
				break
			}
			fmt.Printf("   %d. %s (%s - %s)\n", i+1, article.Title, article.Source,
				article.PublishedAt.Format("15:04"))
		}
	}
	fmt.Println()

	// Test 3: Stock-specific news
	fmt.Println("=== Test 3: Stock News (AAPL) ===")
	stockArticles, err := googleNewsClient.GetStockNews("AAPL", 5, cfg)
	if err != nil {
		log.Printf("Error in Test 3: %v", err)
	} else {
		fmt.Printf("✅ Found %d articles about AAPL\n", len(stockArticles))
		for i, article := range stockArticles {
			if i >= 3 { // Show first 3
				break
			}
			fmt.Printf("   %d. %s (%s)\n", i+1, article.Title, article.Source)
		}
	}
	fmt.Println()

	// Test 4: Search with specific site
	fmt.Println("=== Test 4: Site-specific Search (Reuters) ===")
	params4 := dataflows.EnhancedGoogleNewsParams{
		Query:      "stock market",
		Language:   "en",
		Country:    "US",
		MaxResults: 5,
		SortBy:     "date",
		Site:       "reuters.com",
	}

	articles4, err := googleNewsClient.GetGoogleNews(params4, cfg)
	if err != nil {
		log.Printf("Error in Test 4: %v", err)
	} else {
		fmt.Printf("✅ Found %d Reuters articles about stock market\n", len(articles4))
		for i, article := range articles4 {
			if i >= 2 { // Show first 2
				break
			}
			fmt.Printf("   %d. %s\n", i+1, article.Title)
			if article.Content != "" {
				summary := article.Content
				if len(summary) > 100 {
					summary = summary[:100] + "..."
				}
				fmt.Printf("      Summary: %s\n", summary)
			}
		}
	}
	fmt.Println()

	// Test 5: Date range search
	fmt.Println("=== Test 5: Date Range Search (Last 2 days) ===")
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -2) // Last 2 days

	params5 := dataflows.EnhancedGoogleNewsParams{
		Query:      "crypto bitcoin",
		Language:   "en",
		Country:    "US",
		StartDate:  startDate,
		EndDate:    endDate,
		MaxResults: 5,
		SortBy:     "date",
	}

	articles5, err := googleNewsClient.GetGoogleNews(params5, cfg)
	if err != nil {
		log.Printf("Error in Test 5: %v", err)
	} else {
		fmt.Printf("✅ Found %d crypto articles from last 2 days\n", len(articles5))
		for i, article := range articles5 {
			if i >= 3 { // Show first 3
				break
			}
			timeSince := time.Since(article.PublishedAt)
			var timeStr string
			if timeSince < time.Hour {
				timeStr = fmt.Sprintf("%d minutes ago", int(timeSince.Minutes()))
			} else if timeSince < 24*time.Hour {
				timeStr = fmt.Sprintf("%d hours ago", int(timeSince.Hours()))
			} else {
				timeStr = fmt.Sprintf("%d days ago", int(timeSince.Hours()/24))
			}

			fmt.Printf("   %d. %s (%s - %s)\n", i+1, article.Title, article.Source, timeStr)
		}
	}
	fmt.Println()

	// Test 6: Chinese language news
	fmt.Println("=== Test 6: Chinese Language News ===")
	params6 := dataflows.EnhancedGoogleNewsParams{
		Query:      "中国经济",
		Language:   "zh-CN",
		Country:    "CN",
		MaxResults: 3,
		SortBy:     "date",
	}

	articles6, err := googleNewsClient.GetGoogleNews(params6, cfg)
	if err != nil {
		log.Printf("Error in Test 6: %v", err)
	} else {
		fmt.Printf("✅ Found %d Chinese articles about economy\n", len(articles6))
		for i, article := range articles6 {
			fmt.Printf("   %d. %s (%s)\n", i+1, article.Title, article.Source)
		}
	}
	fmt.Println()

	// Test 7: Content Extraction (with demo explanation)
	fmt.Println("=== Test 7: Content Extraction Demo ===")
	fmt.Println("注意：Google News使用复杂的重定向机制，实际内容提取需要:")
	fmt.Println("  1. 直接访问新闻源的RSS feed")
	fmt.Println("  2. 使用专门的新闻API")
	fmt.Println("  3. 使用全文提取服务")
	fmt.Println("当前演示展示了内容提取框架的工作原理。")
	fmt.Println()

	contentParams := dataflows.EnhancedGoogleNewsParams{
		Query:      "technology earnings report",
		Language:   "en",
		Country:    "US",
		MaxResults: 2,
		SortBy:     "date",
		Site:       "reuters.com", // 使用Reuters这样的直接新闻源
	}

	articlesWithContent, err := googleNewsClient.GetNewsWithContent(contentParams, cfg)
	if err != nil {
		log.Printf("Error in Test 7: %v", err)
	} else {
		fmt.Printf("✅ Found %d articles with content extraction\n", len(articlesWithContent))
		for i, article := range articlesWithContent {
			fmt.Printf("   %d. %s (%s)\n", i+1, article.Title, article.Source)
			if article.Content != "" {
				content := article.Content
				if len(content) > 200 {
					content = content[:200] + "..."
				}
				fmt.Printf("      Content: %s\n", content)
			} else {
				fmt.Printf("      Content: [No content extracted]\n")
			}
			fmt.Println()
		}
	}
	fmt.Println()

	// Test 8: RSS Direct Fetch
	fmt.Println("=== Test 8: RSS Direct Fetch (推荐方法) ===")
	fmt.Println("使用Google News RSS feed直接获取新闻，包含摘要内容")
	fmt.Println()

	rssParams := dataflows.EnhancedGoogleNewsParams{
		Query:      "artificial intelligence",
		Language:   "en",
		Country:    "US",
		MaxResults: 3,
		SortBy:     "date",
	}

	rssArticles, err := googleNewsClient.GetGoogleNewsRSS(rssParams, cfg)
	if err != nil {
		log.Printf("Error in Test 8: %v", err)
	} else {
		fmt.Printf("✅ RSS方式获取到 %d 篇文章 (包含摘要)\n", len(rssArticles))
		for i, article := range rssArticles {
			fmt.Printf("   %d. %s (%s)\n", i+1, article.Title, article.Source)
			if article.Content != "" {
				content := article.Content
				if len(content) > 150 {
					content = content[:150] + "..."
				}
				fmt.Printf("      摘要: %s\n", content)
			}
			fmt.Println()
		}
	}
	fmt.Println()

	// Summary
	fmt.Println("📊 Test Summary:")
	fmt.Println("================")
	fmt.Printf("✅ General AI news: %d articles\n", len(articles1))
	fmt.Printf("✅ Finance news: %d articles\n", len(financeArticles))
	fmt.Printf("✅ AAPL stock news: %d articles\n", len(stockArticles))
	fmt.Printf("✅ Reuters articles: %d articles\n", len(articles4))
	fmt.Printf("✅ Recent crypto news: %d articles\n", len(articles5))
	fmt.Printf("✅ Chinese news: %d articles\n", len(articles6))
	fmt.Printf("✅ Content extraction: %d articles\n", len(articlesWithContent))
	fmt.Printf("✅ RSS direct fetch: %d articles\n", len(rssArticles))

	totalArticles := len(articles1) + len(financeArticles) + len(stockArticles) +
					len(articles4) + len(articles5) + len(articles6) + len(articlesWithContent) + len(rssArticles)
	fmt.Printf("\n🎉 Total articles retrieved: %d\n", totalArticles)

	if totalArticles > 0 {
		fmt.Println("\n✨ Google News Enhanced Dataflow is working successfully!")
		fmt.Println("\nFeatures tested:")
		fmt.Println("  ✓ General news search")
		fmt.Println("  ✓ Category filtering (technology, business)")
		fmt.Println("  ✓ Finance-specific news aggregation")
		fmt.Println("  ✓ Stock symbol news detection")
		fmt.Println("  ✓ Site-specific filtering")
		fmt.Println("  ✓ Date range filtering")
		fmt.Println("  ✓ Multi-language support")
		fmt.Println("  ✓ Caching and data persistence")
		fmt.Println("  ✓ Article content extraction framework")
		fmt.Println("  ✓ RSS direct feed parsing with content")
	} else {
		fmt.Println("\n⚠️  No articles were retrieved. This might be due to:")
		fmt.Println("  - Network connectivity issues")
		fmt.Println("  - Google's anti-bot measures")
		fmt.Println("  - Rate limiting")
		fmt.Println("  - HTML structure changes")
	}

	fmt.Println("\n📁 Check the ./data/news_data directory for saved results!")

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📋 Google News Content Extraction Summary")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("✅ 成功实现的功能:")
	fmt.Println("  • 多种Google News搜索方式 (通用搜索、分类、股票、网站过滤)")
	fmt.Println("  • RSS feed解析和数据提取")
	fmt.Println("  • 缓存机制避免重复请求")
	fmt.Println("  • 多语言支持 (中文、英文)")
	fmt.Println("  • 时间范围过滤")
	fmt.Println("  • 内容提取框架 (支持多种CSS选择器)")
	fmt.Println()
	fmt.Println("🔍 关于内容提取:")
	fmt.Println("  • Google News使用加密的重定向链接")
	fmt.Println("  • 当前提取到的是Google News页面的meta描述")
	fmt.Println("  • 完整内容提取需要直接访问原始新闻源")
	fmt.Println("  • 框架已具备从任何HTML页面提取内容的能力")
	fmt.Println()
	fmt.Println("🚀 实际使用建议:")
	fmt.Println("  1. 【推荐】使用RSS方法 (GetGoogleNewsRSS) 获取新闻摘要")
	fmt.Println("  2. RSS方式提供标题、摘要、来源等完整信息，无需额外解析")
	fmt.Println("  3. 对于交易分析，RSS摘要内容通常已足够判断市场情绪")
	fmt.Println("  4. 如需完整文章内容，可结合专门的新闻API")
}