package tools

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	t_utils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/models"
	"github.com/dyike/CortexGo/pkg/dataflows"
)

// NewGoogleNewsSearchTool creates a tool for searching Google News
func NewGoogleNewsSearchTool(cfg *config.Config) tool.BaseTool {
	return t_utils.NewTool(
		&schema.ToolInfo{
			Name: "search_google_news",
			Desc: "Search Google News for articles using enhanced search capabilities",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"query": {
					Type:     "string",
					Desc:     "Search query for news articles",
					Required: true,
				},
				"language": {
					Type:     "string",
					Desc:     "Language code (e.g., 'en', 'zh-CN', 'zh-TW'). Default: 'en'",
					Required: false,
				},
				"country": {
					Type:     "string",
					Desc:     "Country code (e.g., 'US', 'CN', 'TW'). Default: 'US'",
					Required: false,
				},
				"max_results": {
					Type:     "integer",
					Desc:     "Maximum number of results to return (1-50, default: 20)",
					Required: false,
				},
				"sort_by": {
					Type:     "string",
					Desc:     "Sort method: 'date' or 'relevance' (default: 'date')",
					Required: false,
				},
				"category": {
					Type:     "string",
					Desc:     "News category: 'business', 'technology', 'health', 'sports', etc.",
					Required: false,
				},
				"site": {
					Type:     "string",
					Desc:     "Specific news site to search (e.g., 'reuters.com', 'bloomberg.com')",
					Required: false,
				},
				"days_back": {
					Type:     "integer",
					Desc:     "Number of days to look back for news (default: 7)",
					Required: false,
				},
			}),
		},
		func(ctx context.Context, input models.GoogleNewsSearchInput) (*models.NewsOutput, error) {
			if input.Query == "" {
				return nil, fmt.Errorf("query parameter is required")
			}

			// Set defaults
			language := input.Language
			if language == "" {
				language = "en"
			}

			country := input.Country
			if country == "" {
				country = "US"
			}

			maxResults := input.MaxResults
			if maxResults <= 0 {
				maxResults = 10
			}
			if maxResults > 10 {
				maxResults = 20
			}

			sortBy := input.SortBy
			if sortBy == "" {
				sortBy = "date"
			}

			daysBack := input.DaysBack
			if daysBack <= 0 {
				daysBack = 7
			}

			// Calculate date range
			endDate := time.Now()
			startDate := endDate.AddDate(0, 0, -daysBack)

			// Set up search parameters
			params := dataflows.EnhancedGoogleNewsParams{
				Query:      input.Query,
				Language:   language,
				Country:    country,
				StartDate:  startDate,
				EndDate:    endDate,
				MaxResults: maxResults,
				SortBy:     sortBy,
				Category:   input.Category,
				Site:       input.Site,
			}

			// Create Google News client
			googleNewsClient := dataflows.NewGoogleNewsClient(cfg)

			// Search for news
			articles, err := googleNewsClient.GetGoogleNews(params, cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to search Google News: %v", err)
			}

			log.Printf("Found %d Google News articles for query: %s", len(articles), input.Query)

			// Format results
			var result strings.Builder
			result.WriteString(fmt.Sprintf("# Google News Search Results for \"%s\"\n\n", input.Query))
			result.WriteString(fmt.Sprintf("*Found %d articles (past %d days)*\n\n", len(articles), daysBack))

			if len(articles) == 0 {
				result.WriteString("No articles found matching your search criteria.\n")
			} else {
				for i, article := range articles {
					result.WriteString(fmt.Sprintf("## %d. %s\n", i+1, article.Title))
					result.WriteString(fmt.Sprintf("**Source:** %s | **Published:** %s\n",
						article.Source, article.PublishedAt.Format("2006-01-02 15:04")))
					result.WriteString(fmt.Sprintf("**URL:** %s\n", article.URL))

					if article.Content != "" && len(article.Content) > 200 {
						result.WriteString(fmt.Sprintf("**Summary:** %s...\n", article.Content[:200]))
					} else if article.Content != "" {
						result.WriteString(fmt.Sprintf("**Summary:** %s\n", article.Content))
					}

					result.WriteString("\n---\n\n")
				}

				// Add search metadata
				result.WriteString("## Search Parameters\n\n")
				result.WriteString(fmt.Sprintf("- **Query:** %s\n", input.Query))
				result.WriteString(fmt.Sprintf("- **Language:** %s\n", language))
				result.WriteString(fmt.Sprintf("- **Country:** %s\n", country))
				result.WriteString(fmt.Sprintf("- **Sort by:** %s\n", sortBy))
				if input.Category != "" {
					result.WriteString(fmt.Sprintf("- **Category:** %s\n", input.Category))
				}
				if input.Site != "" {
					result.WriteString(fmt.Sprintf("- **Site:** %s\n", input.Site))
				}
			}

			return &models.NewsOutput{
				Articles: convertToModelsNewsArticles(articles),
				Result:   result.String(),
			}, nil
		},
	)
}

// NewGoogleFinanceNewsTool creates a tool for getting finance news from Google
func NewGoogleFinanceNewsTool(cfg *config.Config) tool.BaseTool {
	return t_utils.NewTool(
		&schema.ToolInfo{
			Name: "get_google_finance_news",
			Desc: "Get latest finance and business news from Google News",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"max_results": {
					Type:     "integer",
					Desc:     "Maximum number of articles to return (default: 20)",
					Required: false,
				},
			}),
		},
		func(ctx context.Context, input models.FinanceNewsInput) (*models.NewsOutput, error) {
			maxResults := input.Limit
			if maxResults <= 0 {
				maxResults = 10
			}
			if maxResults > 10 {
				maxResults = 20
			}

			// Create Google News client
			googleNewsClient := dataflows.NewGoogleNewsClient(cfg)

			// Get finance news
			articles, err := googleNewsClient.GetFinanceNews(maxResults, cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to get finance news: %v", err)
			}

			log.Printf("Retrieved %d finance articles from Google News", len(articles))

			// Format results
			var result strings.Builder
			result.WriteString("# Latest Finance News from Google News\n\n")
			result.WriteString(fmt.Sprintf("*%d articles from major financial sources*\n\n", len(articles)))

			if len(articles) == 0 {
				result.WriteString("No finance news found.\n")
			} else {
				// Group by recency
				now := time.Now()
				var recent, older []*dataflows.NewsArticle

				for _, article := range articles {
					hoursSince := now.Sub(article.PublishedAt).Hours()
					if hoursSince <= 6 {
						recent = append(recent, article)
					} else {
						older = append(older, article)
					}
				}

				if len(recent) > 0 {
					result.WriteString("## 🔥 Breaking News (Last 6 Hours)\n\n")
					for i, article := range recent {
						result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, article.Title))
						result.WriteString(fmt.Sprintf("**%s** - %s\n",
							article.Source, article.PublishedAt.Format("15:04")))
						result.WriteString(fmt.Sprintf("**URL:** %s\n", article.URL))

						if article.Content != "" && len(article.Content) > 150 {
							result.WriteString(fmt.Sprintf("**Summary:** %s...\n", article.Content[:150]))
						} else if article.Content != "" {
							result.WriteString(fmt.Sprintf("**Summary:** %s\n", article.Content))
						}
						result.WriteString("\n")
					}
					result.WriteString("---\n\n")
				}

				if len(older) > 0 {
					result.WriteString("## 📈 Recent Financial News\n\n")
					for i, article := range older {
						result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, article.Title))
						result.WriteString(fmt.Sprintf("**%s** - %s\n",
							article.Source, article.PublishedAt.Format("2006-01-02 15:04")))
						result.WriteString(fmt.Sprintf("**URL:** %s\n", article.URL))

						if article.Content != "" && len(article.Content) > 100 {
							result.WriteString(fmt.Sprintf("**Summary:** %s...\n", article.Content[:100]))
						} else if article.Content != "" {
							result.WriteString(fmt.Sprintf("**Summary:** %s\n", article.Content))
						}
						result.WriteString("\n")
					}
				}

				// Add market insights
				result.WriteString("## 💡 Market Insights\n\n")
				result.WriteString("**Key Topics Trending:**\n")

				// Simple keyword analysis
				keywords := make(map[string]int)
				for _, article := range articles {
					words := strings.Fields(strings.ToLower(article.Title))
					for _, word := range words {
						if len(word) > 4 && !isCommonWord(word) {
							keywords[word]++
						}
					}
				}

				// Show top keywords
				count := 0
				for word, freq := range keywords {
					if freq >= 2 && count < 5 {
						result.WriteString(fmt.Sprintf("- %s (mentioned %d times)\n", word, freq))
						count++
					}
				}
			}

			return &models.NewsOutput{
				Articles: convertToModelsNewsArticles(articles),
				Result:   result.String(),
			}, nil
		},
	)
}

// NewGoogleStockNewsTool creates a tool for getting stock-specific news from Google
func NewGoogleStockNewsTool(cfg *config.Config) tool.BaseTool {
	return t_utils.NewTool(
		&schema.ToolInfo{
			Name: "get_google_stock_news",
			Desc: "Get news articles for a specific stock symbol from Google News",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"symbol": {
					Type:     "string",
					Desc:     "Stock symbol (e.g., AAPL, TSLA, GOOGL)",
					Required: true,
				},
				"max_results": {
					Type:     "integer",
					Desc:     "Maximum number of articles to return (default: 15)",
					Required: false,
				},
			}),
		},
		func(ctx context.Context, input models.StockNewsInput) (*models.NewsOutput, error) {
			if input.Symbol == "" {
				return nil, fmt.Errorf("symbol parameter is required")
			}

			maxResults := input.MaxResults
			if maxResults <= 0 {
				maxResults = 10
			}
			if maxResults > 10 {
				maxResults = 20
			}

			// Create Google News client
			googleNewsClient := dataflows.NewGoogleNewsClient(cfg)

			// Get stock news
			articles, err := googleNewsClient.GetStockNews(input.Symbol, maxResults, cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to get stock news: %v", err)
			}

			symbol := strings.ToUpper(input.Symbol)
			log.Printf("Found %d news articles for %s", len(articles), symbol)

			// Format results
			var result strings.Builder
			result.WriteString(fmt.Sprintf("# News Articles for $%s\n\n", symbol))

			if len(articles) == 0 {
				result.WriteString(fmt.Sprintf("No recent news found for %s.\n", symbol))
				result.WriteString("\n**Suggestions:**\n")
				result.WriteString("- Check if the symbol is correct\n")
				result.WriteString("- Try searching for the company name instead\n")
				result.WriteString("- The stock might not have recent news coverage\n")
			} else {
				result.WriteString(fmt.Sprintf("*Found %d relevant articles*\n\n", len(articles)))

				// Categorize news by recency
				now := time.Now()
				var breaking, recent, older []*dataflows.NewsArticle

				for _, article := range articles {
					hoursSince := now.Sub(article.PublishedAt).Hours()
					if hoursSince <= 2 {
						breaking = append(breaking, article)
					} else if hoursSince <= 24 {
						recent = append(recent, article)
					} else {
						older = append(older, article)
					}
				}

				if len(breaking) > 0 {
					result.WriteString("## 🚨 Breaking News (Last 2 Hours)\n\n")
					for i, article := range breaking {
						result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, article.Title))
						result.WriteString(fmt.Sprintf("**%s** - %s ago\n",
							article.Source, formatTimeSince(article.PublishedAt)))
						result.WriteString(fmt.Sprintf("**URL:** %s\n", article.URL))
						if article.Content != "" {
							result.WriteString(fmt.Sprintf("**Summary:** %s\n", article.Content))
						}
						result.WriteString("\n")
					}
					result.WriteString("---\n\n")
				}

				if len(recent) > 0 {
					result.WriteString("## 📰 Today's News\n\n")
					for i, article := range recent {
						result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, article.Title))
						result.WriteString(fmt.Sprintf("**%s** - %s\n",
							article.Source, article.PublishedAt.Format("15:04")))
						result.WriteString(fmt.Sprintf("**URL:** %s\n", article.URL))
						if article.Content != "" && len(article.Content) > 150 {
							result.WriteString(fmt.Sprintf("**Summary:** %s...\n", article.Content[:150]))
						} else if article.Content != "" {
							result.WriteString(fmt.Sprintf("**Summary:** %s\n", article.Content))
						}
						result.WriteString("\n")
					}
					result.WriteString("---\n\n")
				}

				if len(older) > 0 {
					result.WriteString("## 📅 Recent Coverage\n\n")
					for i, article := range older {
						if i >= 5 { // Limit older news to 5 items
							result.WriteString(fmt.Sprintf("... and %d more articles\n", len(older)-5))
							break
						}
						result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, article.Title))
						result.WriteString(fmt.Sprintf("**%s** - %s\n",
							article.Source, article.PublishedAt.Format("2006-01-02")))
						result.WriteString(fmt.Sprintf("**URL:** %s\n", article.URL))
						result.WriteString("\n")
					}
				}

				// Add analysis
				result.WriteString("## 📊 News Analysis\n\n")
				result.WriteString(fmt.Sprintf("- **Total Articles:** %d\n", len(articles)))
				result.WriteString(fmt.Sprintf("- **Breaking News:** %d\n", len(breaking)))
				result.WriteString(fmt.Sprintf("- **Today's Coverage:** %d\n", len(recent)))
				result.WriteString(fmt.Sprintf("- **Recent Coverage:** %d\n", len(older)))

				if len(articles) > 0 {
					// Find most active source
					sources := make(map[string]int)
					for _, article := range articles {
						sources[article.Source]++
					}

					maxCount := 0
					topSource := ""
					for source, count := range sources {
						if count > maxCount {
							maxCount = count
							topSource = source
						}
					}

					if topSource != "" {
						result.WriteString(fmt.Sprintf("- **Most Active Source:** %s (%d articles)\n", topSource, maxCount))
					}
				}
			}

			return &models.NewsOutput{
				Articles: convertToModelsNewsArticles(articles),
				Result:   result.String(),
			}, nil
		},
	)
}

// Helper functions

// convertToModelsNewsArticles converts dataflows.NewsArticle to models.NewsArticle
func convertToModelsNewsArticles(articles []*dataflows.NewsArticle) []*models.NewsArticle {
	modelArticles := make([]*models.NewsArticle, len(articles))
	for i, a := range articles {
		modelArticles[i] = &models.NewsArticle{
			Title:       a.Title,
			Content:     a.Content,
			URL:         a.URL,
			Source:      a.Source,
			PublishedAt: a.PublishedAt,
			Keywords:    a.Keywords,
		}
	}
	return modelArticles
}

// formatTimeSince formats time duration in human-readable format
func formatTimeSince(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
}

// isCommonWord checks if a word is a common word to filter out
func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true, "not": true,
		"you": true, "all": true, "can": true, "had": true, "her": true, "was": true,
		"one": true, "our": true, "out": true, "day": true, "get": true, "has": true,
		"him": true, "how": true, "man": true, "new": true, "now": true, "old": true,
		"see": true, "two": true, "way": true, "who": true, "boy": true, "did": true,
		"its": true, "let": true, "put": true, "say": true, "she": true, "too": true,
		"use": true, "will": true, "with": true, "this": true, "that": true, "from": true,
		"they": true, "know": true, "want": true, "been": true, "good": true, "much": true,
		"some": true, "time": true, "very": true, "when": true, "come": true, "here": true,
		"just": true, "like": true, "long": true, "make": true, "many": true, "over": true,
		"such": true, "take": true, "than": true, "them": true, "well": true, "were": true,
	}
	return commonWords[strings.ToLower(word)]
}
