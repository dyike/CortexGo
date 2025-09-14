package dataflows

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// RedditClient handles Reddit API operations
type RedditClient struct {
	client *resty.Client
	cache  *CacheManager
}

// NewRedditClient creates a new Reddit client
func NewRedditClient(config *Config) *RedditClient {
	cacheDir := filepath.Join(config.DataCacheDir, "reddit")
	cache := NewCacheManager(cacheDir, 1*time.Hour, config.CacheEnabled) // 1 hour cache for Reddit

	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("User-Agent", "CortexGo/1.0 (by /u/cortexgo)")

	return &RedditClient{
		client: client,
		cache:  cache,
	}
}

// RedditSearchParams represents parameters for Reddit search
type RedditSearchParams struct {
	Query      string    `json:"query"`
	Subreddit  string    `json:"subreddit"`
	Sort       string    `json:"sort"`       // relevance, hot, top, new, comments
	Time       string    `json:"time"`       // hour, day, week, month, year, all
	Limit      int       `json:"limit"`
	After      string    `json:"after"`
	MaxResults int       `json:"max_results"`
}

// RedditResponse represents the API response structure
type RedditResponse struct {
	Kind string `json:"kind"`
	Data struct {
		After     string `json:"after"`
		Before    string `json:"before"`
		Children  []RedditChild `json:"children"`
		Dist      int    `json:"dist"`
		Modhash   string `json:"modhash"`
	} `json:"data"`
}

// RedditChild represents a Reddit post wrapper
type RedditChild struct {
	Kind string    `json:"kind"`
	Data RedditPostData `json:"data"`
}

// RedditPostData represents Reddit post data from API
type RedditPostData struct {
	ID                string  `json:"id"`
	Title             string  `json:"title"`
	Selftext          string  `json:"selftext"`
	URL               string  `json:"url"`
	Permalink         string  `json:"permalink"`
	Subreddit         string  `json:"subreddit"`
	Author            string  `json:"author"`
	Score             int     `json:"score"`
	NumComments       int     `json:"num_comments"`
	Created           float64 `json:"created"`
	CreatedUTC        float64 `json:"created_utc"`
	Stickied          bool    `json:"stickied"`
	Locked            bool    `json:"locked"`
	Over18            bool    `json:"over_18"`
	Distinguished     *string `json:"distinguished"`
	IsSelf            bool    `json:"is_self"`
	Domain            string  `json:"domain"`
	Thumbnail         string  `json:"thumbnail"`
	SelftextHTML      *string `json:"selftext_html"`
}

// GetSubredditPosts retrieves posts from a specific subreddit
func (rc *RedditClient) GetSubredditPosts(subreddit string, sort string, limit int, config *Config) ([]*RedditPost, error) {
	if strings.TrimSpace(subreddit) == "" {
		return nil, fmt.Errorf("subreddit cannot be empty")
	}

	// Set defaults
	if sort == "" {
		sort = "hot"
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s_%s_%d", subreddit, sort, limit)
	var cached []*RedditPost
	if rc.cache.Get("subreddit", "posts", cacheKey, &cached) {
		return cached, nil
	}

	// Build Reddit URL
	redditURL := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d", subreddit, sort, limit)

	var result []*RedditPost
	err := WithRetry(DefaultRetryConfig(), func() error {
		resp, err := rc.client.R().Get(redditURL)
		if err != nil {
			return fmt.Errorf("failed to fetch Reddit posts: %w", err)
		}

		if resp.StatusCode() != 200 {
			return fmt.Errorf("HTTP error %d when fetching Reddit posts", resp.StatusCode())
		}

		var redditResp RedditResponse
		if err := json.Unmarshal(resp.Body(), &redditResp); err != nil {
			return fmt.Errorf("failed to parse Reddit JSON: %w", err)
		}

		result = rc.convertToRedditPosts(redditResp.Data.Children)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Cache the result
	rc.cache.Set("subreddit", "posts", cacheKey, result)

	// Save to file
	filePath := filepath.Join(config.DataDir, "reddit_data",
		fmt.Sprintf("r_%s_%s_%s.json",
			subreddit, sort, time.Now().Format("2006-01-02")))
	SaveDataToFile(result, filePath)

	return result, nil
}

// SearchReddit searches Reddit for posts matching a query
func (rc *RedditClient) SearchReddit(params RedditSearchParams, config *Config) ([]*RedditPost, error) {
	if strings.TrimSpace(params.Query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// Set defaults
	if params.Sort == "" {
		params.Sort = "relevance"
	}
	if params.Time == "" {
		params.Time = "week"
	}
	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 25
	}
	if params.MaxResults <= 0 {
		params.MaxResults = 50
	}

	// Check cache first
	var cached []*RedditPost
	if rc.cache.Get("search", "query", params, &cached) {
		return cached, nil
	}

	var allResults []*RedditPost
	after := params.After

	for len(allResults) < params.MaxResults {
		// Build search URL
		searchURL := rc.buildSearchURL(params, after)

		var redditResp RedditResponse
		err := WithRetry(DefaultRetryConfig(), func() error {
			resp, err := rc.client.R().Get(searchURL)
			if err != nil {
				return fmt.Errorf("failed to search Reddit: %w", err)
			}

			if resp.StatusCode() != 200 {
				return fmt.Errorf("HTTP error %d when searching Reddit", resp.StatusCode())
			}

			if err := json.Unmarshal(resp.Body(), &redditResp); err != nil {
				return fmt.Errorf("failed to parse Reddit JSON: %w", err)
			}

			return nil
		})

		if err != nil {
			return nil, err
		}

		// Convert and add results
		posts := rc.convertToRedditPosts(redditResp.Data.Children)
		allResults = append(allResults, posts...)

		// Check if we should continue pagination
		if redditResp.Data.After == "" || len(posts) == 0 {
			break
		}
		after = redditResp.Data.After

		// Don't exceed max results
		if len(allResults) >= params.MaxResults {
			allResults = allResults[:params.MaxResults]
			break
		}
	}

	// Cache the result
	rc.cache.Set("search", "query", params, allResults)

	// Save to file
	filePath := filepath.Join(config.DataDir, "reddit_data",
		fmt.Sprintf("search_%s_%s.json",
			strings.ReplaceAll(params.Query, " ", "_"),
			time.Now().Format("2006-01-02")))
	SaveDataToFile(allResults, filePath)

	return allResults, nil
}

// GetPopularFinancePosts gets posts from popular finance-related subreddits
func (rc *RedditClient) GetPopularFinancePosts(limit int, config *Config) ([]*RedditPost, error) {
	financeSubreddits := []string{
		"wallstreetbets", "investing", "stocks", "SecurityAnalysis",
		"ValueInvesting", "options", "Bogleheads", "financialindependence",
		"personalfinance", "SecurityAnalysis", "StockMarket", "pennystocks",
	}

	var allPosts []*RedditPost
	postsPerSub := limit / len(financeSubreddits)
	if postsPerSub < 1 {
		postsPerSub = 1
	}

	for _, subreddit := range financeSubreddits {
		posts, err := rc.GetSubredditPosts(subreddit, "hot", postsPerSub, config)
		if err != nil {
			// Log error but continue with other subreddits
			continue
		}
		allPosts = append(allPosts, posts...)
	}

	// Sort by score and limit results
	if len(allPosts) > limit {
		// Sort by score (descending)
		for i := 0; i < len(allPosts)-1; i++ {
			for j := i + 1; j < len(allPosts); j++ {
				if allPosts[i].Score < allPosts[j].Score {
					allPosts[i], allPosts[j] = allPosts[j], allPosts[i]
				}
			}
		}
		allPosts = allPosts[:limit]
	}

	return allPosts, nil
}

// GetStockMentions searches for mentions of a specific stock symbol
func (rc *RedditClient) GetStockMentions(symbol string, config *Config) ([]*RedditPost, error) {
	if strings.TrimSpace(symbol) == "" {
		return nil, fmt.Errorf("stock symbol cannot be empty")
	}

	// Clean and format symbol
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// Create search queries for the symbol
	queries := []string{
		fmt.Sprintf("$%s", symbol),      // $AAPL format
		fmt.Sprintf("%s stock", symbol), // AAPL stock
		symbol,                          // Just AAPL
	}

	var allResults []*RedditPost
	seen := make(map[string]bool) // To avoid duplicates

	for _, query := range queries {
		params := RedditSearchParams{
			Query:      query,
			Subreddit:  "wallstreetbets+stocks+investing+SecurityAnalysis+StockMarket",
			Sort:       "relevance",
			Time:       "week",
			Limit:      25,
			MaxResults: 25,
		}

		posts, err := rc.SearchReddit(params, config)
		if err != nil {
			continue // Skip this query if it fails
		}

		// Filter for posts that actually mention the symbol and avoid duplicates
		for _, post := range posts {
			if !seen[post.ID] && rc.containsStockSymbol(post, symbol) {
				seen[post.ID] = true
				allResults = append(allResults, post)
			}
		}
	}

	return allResults, nil
}

// buildSearchURL constructs the Reddit search URL
func (rc *RedditClient) buildSearchURL(params RedditSearchParams, after string) string {
	baseURL := "https://www.reddit.com/search.json"

	values := url.Values{}
	values.Set("q", params.Query)
	values.Set("sort", params.Sort)
	values.Set("t", params.Time)
	values.Set("limit", fmt.Sprintf("%d", params.Limit))

	if params.Subreddit != "" {
		values.Set("q", fmt.Sprintf("%s subreddit:%s", params.Query, params.Subreddit))
	}

	if after != "" {
		values.Set("after", after)
	}

	return fmt.Sprintf("%s?%s", baseURL, values.Encode())
}

// convertToRedditPosts converts Reddit API response to RedditPost structs
func (rc *RedditClient) convertToRedditPosts(children []RedditChild) []*RedditPost {
	var posts []*RedditPost

	for _, child := range children {
		if child.Kind != "t3" { // t3 is the Reddit kind for posts
			continue
		}

		data := child.Data
		createdAt := time.Unix(int64(data.CreatedUTC), 0)

		// Construct full URL
		fullURL := data.URL
		if data.IsSelf {
			fullURL = fmt.Sprintf("https://www.reddit.com%s", data.Permalink)
		}

		// Get content - prefer selftext for text posts
		content := data.Selftext
		if content == "" && data.SelftextHTML != nil {
			content = *data.SelftextHTML
		}

		post := &RedditPost{
			ID:         data.ID,
			Title:      data.Title,
			Content:    content,
			URL:        fullURL,
			Subreddit:  data.Subreddit,
			Author:     data.Author,
			Score:      data.Score,
			Comments:   data.NumComments,
			CreatedAt:  createdAt,
			IsStickied: data.Stickied,
			IsLocked:   data.Locked,
		}

		posts = append(posts, post)
	}

	return posts
}

// containsStockSymbol checks if a post contains mentions of a stock symbol
func (rc *RedditClient) containsStockSymbol(post *RedditPost, symbol string) bool {
	text := strings.ToUpper(post.Title + " " + post.Content)

	// Check for various formats
	patterns := []string{
		fmt.Sprintf("$%s", symbol),          // $AAPL
		fmt.Sprintf(" %s ", symbol),         // " AAPL "
		fmt.Sprintf("(%s)", symbol),         // (AAPL)
		fmt.Sprintf("%s stock", symbol),     // AAPL stock
		fmt.Sprintf("%s shares", symbol),    // AAPL shares
	}

	for _, pattern := range patterns {
		if strings.Contains(text, strings.ToUpper(pattern)) {
			return true
		}
	}

	// Use regex for more sophisticated matching
	// Match symbol with word boundaries
	regex := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(symbol)))
	return regex.MatchString(text)
}