package tools

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	t_utils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/pkg/dataflows"
)

// NewRedditSubredditTool creates a tool for fetching posts from a specific subreddit
func NewRedditSubredditTool(cfg *config.Config) tool.BaseTool {
	return t_utils.NewTool(
		&schema.ToolInfo{
			Name: "get_reddit_subreddit_posts",
			Desc: "Get hot, new, or top posts from a specific subreddit",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"subreddit": {
					Type:     "string",
					Desc:     "The subreddit name (without r/ prefix)",
					Required: true,
				},
				"sort": {
					Type:     "string",
					Desc:     "Sort method: hot, new, top, rising (default: hot)",
					Required: false,
				},
				"limit": {
					Type:     "integer",
					Desc:     "Number of posts to retrieve (1-100, default: 25)",
					Required: false,
				},
			}),
		},
		func(ctx context.Context, input models.RedditSubredditInput) (*models.RedditOutput, error) {
			if input.Subreddit == "" {
				return nil, fmt.Errorf("subreddit parameter is required")
			}

			// Set defaults
			sort := input.Sort
			if sort == "" {
				sort = "hot"
			}

			limit := input.Limit
			if limit <= 0 {
				limit = 25
			}
			if limit > 100 {
				limit = 100
			}

			// Create Reddit client
			redditClient := dataflows.NewRedditClient(cfg)

			// Get posts
			posts, err := redditClient.GetSubredditPosts(input.Subreddit, sort, limit, cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to get Reddit posts: %v", err)
			}

			log.Printf("Retrieved %d posts from r/%s (%s)", len(posts), input.Subreddit, sort)

			// Format results
			var result strings.Builder
			result.WriteString(fmt.Sprintf("# Posts from r/%s (%s)\n\n", input.Subreddit, sort))

			for i, post := range posts {
				result.WriteString(fmt.Sprintf("## %d. %s\n", i+1, post.Title))
				result.WriteString(fmt.Sprintf("**Author:** u/%s | **Score:** %d | **Comments:** %d\n",
					post.Author, post.Score, post.Comments))
				result.WriteString(fmt.Sprintf("**Created:** %s\n", post.CreatedAt.Format("2006-01-02 15:04")))
				result.WriteString(fmt.Sprintf("**URL:** %s\n", post.URL))

				if post.Content != "" && len(post.Content) > 200 {
					result.WriteString(fmt.Sprintf("**Content Preview:** %s...\n", post.Content[:200]))
				} else if post.Content != "" {
					result.WriteString(fmt.Sprintf("**Content:** %s\n", post.Content))
				}

				if post.IsStickied {
					result.WriteString("*[STICKIED]*\n")
				}
				if post.IsLocked {
					result.WriteString("*[LOCKED]*\n")
				}

				result.WriteString("\n---\n\n")
			}

			// Convert dataflows.RedditPost to models.RedditPost
			modelPosts := make([]*models.RedditPost, len(posts))
			for i, p := range posts {
				modelPosts[i] = &models.RedditPost{
					ID:         p.ID,
					Title:      p.Title,
					Content:    p.Content,
					URL:        p.URL,
					Subreddit:  p.Subreddit,
					Author:     p.Author,
					Score:      p.Score,
					Comments:   p.Comments,
					CreatedAt:  p.CreatedAt,
					Sentiment:  p.Sentiment,
					IsStickied: p.IsStickied,
					IsLocked:   p.IsLocked,
				}
			}

			return &models.RedditOutput{
				Posts:  modelPosts,
				Result: result.String(),
			}, nil
		},
	)
}

// NewRedditSearchTool creates a tool for searching Reddit posts
func NewRedditSearchTool(cfg *config.Config) tool.BaseTool {
	return t_utils.NewTool(
		&schema.ToolInfo{
			Name: "search_reddit_posts",
			Desc: "Search Reddit posts across all subreddits or within specific subreddits",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"query": {
					Type:     "string",
					Desc:     "Search query terms",
					Required: true,
				},
				"subreddit": {
					Type:     "string",
					Desc:     "Limit search to specific subreddit(s). Use + to separate multiple subreddits (e.g., 'stocks+investing')",
					Required: false,
				},
				"sort": {
					Type:     "string",
					Desc:     "Sort method: relevance, hot, top, new, comments (default: relevance)",
					Required: false,
				},
				"time": {
					Type:     "string",
					Desc:     "Time period: hour, day, week, month, year, all (default: week)",
					Required: false,
				},
				"limit": {
					Type:     "integer",
					Desc:     "Maximum number of results (default: 25)",
					Required: false,
				},
			}),
		},
		func(ctx context.Context, input models.RedditSearchInput) (*models.RedditOutput, error) {
			if input.Query == "" {
				return nil, fmt.Errorf("query parameter is required")
			}

			// Set up search parameters
			params := dataflows.RedditSearchParams{
				Query:     input.Query,
				Subreddit: input.Subreddit,
				Sort:      input.Sort,
				Time:      input.Time,
				Limit:     25,
			}

			if input.Limit > 0 {
				params.MaxResults = input.Limit
			} else {
				params.MaxResults = 25
			}

			// Create Reddit client
			redditClient := dataflows.NewRedditClient(cfg)

			// Search posts
			posts, err := redditClient.SearchReddit(params, cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to search Reddit: %v", err)
			}

			log.Printf("Found %d posts for query: %s", len(posts), input.Query)

			// Format results
			var result strings.Builder
			subredditInfo := "all subreddits"
			if input.Subreddit != "" {
				subredditInfo = fmt.Sprintf("r/%s", strings.ReplaceAll(input.Subreddit, "+", ", r/"))
			}

			result.WriteString(fmt.Sprintf("# Reddit Search Results for \"%s\" in %s\n\n", input.Query, subredditInfo))

			if len(posts) == 0 {
				result.WriteString("No posts found matching your search criteria.\n")
			} else {
				for i, post := range posts {
					result.WriteString(fmt.Sprintf("## %d. %s\n", i+1, post.Title))
					result.WriteString(fmt.Sprintf("**Subreddit:** r/%s | **Author:** u/%s\n", post.Subreddit, post.Author))
					result.WriteString(fmt.Sprintf("**Score:** %d | **Comments:** %d\n", post.Score, post.Comments))
					result.WriteString(fmt.Sprintf("**Created:** %s\n", post.CreatedAt.Format("2006-01-02 15:04")))
					result.WriteString(fmt.Sprintf("**URL:** %s\n", post.URL))

					if post.Content != "" && len(post.Content) > 300 {
						result.WriteString(fmt.Sprintf("**Content Preview:** %s...\n", post.Content[:300]))
					} else if post.Content != "" {
						result.WriteString(fmt.Sprintf("**Content:** %s\n", post.Content))
					}

					result.WriteString("\n---\n\n")
				}
			}

			// Convert dataflows.RedditPost to models.RedditPost
			modelPosts := make([]*models.RedditPost, len(posts))
			for i, p := range posts {
				modelPosts[i] = &models.RedditPost{
					ID:         p.ID,
					Title:      p.Title,
					Content:    p.Content,
					URL:        p.URL,
					Subreddit:  p.Subreddit,
					Author:     p.Author,
					Score:      p.Score,
					Comments:   p.Comments,
					CreatedAt:  p.CreatedAt,
					Sentiment:  p.Sentiment,
					IsStickied: p.IsStickied,
					IsLocked:   p.IsLocked,
				}
			}

			return &models.RedditOutput{
				Posts:  modelPosts,
				Result: result.String(),
			}, nil
		},
	)
}

// NewRedditStockMentionsTool creates a tool for finding stock mentions on Reddit
func NewRedditStockMentionsTool(cfg *config.Config) tool.BaseTool {
	return t_utils.NewTool(
		&schema.ToolInfo{
			Name: "get_reddit_stock_mentions",
			Desc: "Find Reddit posts mentioning a specific stock symbol across finance-related subreddits",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"symbol": {
					Type:     "string",
					Desc:     "Stock symbol to search for (e.g., AAPL, TSLA)",
					Required: true,
				},
			}),
		},
		func(ctx context.Context, input models.StockMentionsInput) (*models.RedditOutput, error) {
			if input.Symbol == "" {
				return nil, fmt.Errorf("symbol parameter is required")
			}

			// Create Reddit client
			redditClient := dataflows.NewRedditClient(cfg)

			// Get stock mentions
			posts, err := redditClient.GetStockMentions(input.Symbol, cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to get stock mentions: %v", err)
			}

			log.Printf("Found %d posts mentioning %s", len(posts), input.Symbol)

			// Format results
			var result strings.Builder
			result.WriteString(fmt.Sprintf("# Reddit Posts Mentioning $%s\n\n", strings.ToUpper(input.Symbol)))

			if len(posts) == 0 {
				result.WriteString(fmt.Sprintf("No recent posts found mentioning %s.\n", input.Symbol))
			} else {
				// Group by subreddit for better organization
				subredditGroups := make(map[string][]*dataflows.RedditPost)
				for _, post := range posts {
					subredditGroups[post.Subreddit] = append(subredditGroups[post.Subreddit], post)
				}

				for subreddit, subredditPosts := range subredditGroups {
					result.WriteString(fmt.Sprintf("## r/%s (%d posts)\n\n", subreddit, len(subredditPosts)))

					for i, post := range subredditPosts {
						result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, post.Title))
						result.WriteString(fmt.Sprintf("**Author:** u/%s | **Score:** %d | **Comments:** %d\n",
							post.Author, post.Score, post.Comments))
						result.WriteString(fmt.Sprintf("**Created:** %s\n", post.CreatedAt.Format("2006-01-02 15:04")))
						result.WriteString(fmt.Sprintf("**URL:** %s\n", post.URL))

						if post.Content != "" && len(post.Content) > 250 {
							result.WriteString(fmt.Sprintf("**Content Preview:** %s...\n", post.Content[:250]))
						} else if post.Content != "" {
							result.WriteString(fmt.Sprintf("**Content:** %s\n", post.Content))
						}

						result.WriteString("\n")
					}
					result.WriteString("---\n\n")
				}

				// Add sentiment analysis suggestion
				result.WriteString("## Analysis Suggestions\n\n")
				result.WriteString("- Review post scores and comments to gauge community interest\n")
				result.WriteString("- Check recent posts for breaking news or events affecting the stock\n")
				result.WriteString("- Look for posts with high engagement (score + comments) for significant discussions\n")
				result.WriteString("- Consider the credibility of sources and authors in investment discussions\n")
			}

			// Convert dataflows.RedditPost to models.RedditPost
			modelPosts := make([]*models.RedditPost, len(posts))
			for i, p := range posts {
				modelPosts[i] = &models.RedditPost{
					ID:         p.ID,
					Title:      p.Title,
					Content:    p.Content,
					URL:        p.URL,
					Subreddit:  p.Subreddit,
					Author:     p.Author,
					Score:      p.Score,
					Comments:   p.Comments,
					CreatedAt:  p.CreatedAt,
					Sentiment:  p.Sentiment,
					IsStickied: p.IsStickied,
					IsLocked:   p.IsLocked,
				}
			}

			return &models.RedditOutput{
				Posts:  modelPosts,
				Result: result.String(),
			}, nil
		},
	)
}

// NewRedditFinanceNewsTool creates a tool for getting popular finance posts
func NewRedditFinanceNewsTool(cfg *config.Config) tool.BaseTool {
	return t_utils.NewTool(
		&schema.ToolInfo{
			Name: "get_reddit_finance_news",
			Desc: "Get popular posts from major finance-related subreddits for market sentiment and news",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"limit": {
					Type:     "integer",
					Desc:     "Maximum number of posts to retrieve (default: 50)",
					Required: false,
				},
			}),
		},
		func(ctx context.Context, input models.FinanceNewsInput) (*models.RedditOutput, error) {
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}

			// Create Reddit client
			redditClient := dataflows.NewRedditClient(cfg)

			// Get popular finance posts
			posts, err := redditClient.GetPopularFinancePosts(limit, cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to get finance posts: %v", err)
			}

			log.Printf("Retrieved %d popular finance posts", len(posts))

			// Format results
			var result strings.Builder
			result.WriteString("# Popular Finance Posts from Reddit\n\n")
			result.WriteString(fmt.Sprintf("*Aggregated from popular finance subreddits - %d posts*\n\n", len(posts)))

			if len(posts) == 0 {
				result.WriteString("No finance posts found.\n")
			} else {
				// Group by subreddit
				subredditGroups := make(map[string][]*dataflows.RedditPost)
				for _, post := range posts {
					subredditGroups[post.Subreddit] = append(subredditGroups[post.Subreddit], post)
				}

				// Show top posts first, then group by subreddit
				result.WriteString("## 🔥 Top Posts (by score)\n\n")

				// Sort posts by score and show top 10
				topPosts := make([]*dataflows.RedditPost, len(posts))
				copy(topPosts, posts)

				// Simple bubble sort by score
				for i := 0; i < len(topPosts)-1; i++ {
					for j := i + 1; j < len(topPosts); j++ {
						if topPosts[i].Score < topPosts[j].Score {
							topPosts[i], topPosts[j] = topPosts[j], topPosts[i]
						}
					}
				}

				// Show top 10
				maxTop := 10
				if len(topPosts) < maxTop {
					maxTop = len(topPosts)
				}

				for i := 0; i < maxTop; i++ {
					post := topPosts[i]
					result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, post.Title))
					result.WriteString(fmt.Sprintf("**r/%s** | **u/%s** | **Score:** %d | **Comments:** %d\n",
						post.Subreddit, post.Author, post.Score, post.Comments))
					result.WriteString(fmt.Sprintf("**Created:** %s | **URL:** %s\n",
						post.CreatedAt.Format("2006-01-02 15:04"), post.URL))

					if post.Content != "" && len(post.Content) > 200 {
						result.WriteString(fmt.Sprintf("**Preview:** %s...\n", post.Content[:200]))
					}

					result.WriteString("\n")
				}

				result.WriteString("\n## 📊 Posts by Subreddit\n\n")
				for subreddit, subredditPosts := range subredditGroups {
					result.WriteString(fmt.Sprintf("### r/%s (%d posts)\n\n", subreddit, len(subredditPosts)))

					for i, post := range subredditPosts {
						result.WriteString(fmt.Sprintf("- **%s** (Score: %d, Comments: %d) - %s\n",
							post.Title, post.Score, post.Comments, post.URL))

						if i >= 2 && len(subredditPosts) > 3 {
							result.WriteString(fmt.Sprintf("- ... and %d more posts\n", len(subredditPosts)-3))
							break
						}
					}
					result.WriteString("\n")
				}

				// Add market sentiment indicators
				result.WriteString("## 📈 Market Sentiment Indicators\n\n")
				result.WriteString("**High Engagement Topics:**\n")

				// Find posts with high comment-to-score ratio (controversy indicator)
				var highEngagement []*dataflows.RedditPost
				for _, post := range posts {
					if post.Comments > 50 && post.Score > 100 {
						ratio := float64(post.Comments) / float64(post.Score)
						if ratio > 0.3 { // High comment-to-score ratio
							highEngagement = append(highEngagement, post)
						}
					}
				}

				if len(highEngagement) > 0 {
					for i, post := range highEngagement {
						if i >= 5 { // Limit to top 5
							break
						}
						ratio := float64(post.Comments) / float64(post.Score)
						result.WriteString(fmt.Sprintf("- %s (r/%s) - Engagement ratio: %.2f\n",
							post.Title, post.Subreddit, ratio))
					}
				} else {
					result.WriteString("- No highly controversial posts detected\n")
				}
			}

			// Convert dataflows.RedditPost to models.RedditPost
			modelPosts := make([]*models.RedditPost, len(posts))
			for i, p := range posts {
				modelPosts[i] = &models.RedditPost{
					ID:         p.ID,
					Title:      p.Title,
					Content:    p.Content,
					URL:        p.URL,
					Subreddit:  p.Subreddit,
					Author:     p.Author,
					Score:      p.Score,
					Comments:   p.Comments,
					CreatedAt:  p.CreatedAt,
					Sentiment:  p.Sentiment,
					IsStickied: p.IsStickied,
					IsLocked:   p.IsLocked,
				}
			}

			return &models.RedditOutput{
				Posts:  modelPosts,
				Result: result.String(),
			}, nil
		},
	)
}