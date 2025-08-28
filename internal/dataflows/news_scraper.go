package dataflows

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

// NewsScraperClient handles news scraping operations
type NewsScraperClient struct {
	client *resty.Client
	cache  *CacheManager
}

// NewNewsScraperClient creates a new news scraper client
func NewNewsScraperClient(config *Config) *NewsScraperClient {
	cacheDir := filepath.Join(config.DataCacheDir, "news_scraper")
	cache := NewCacheManager(cacheDir, 2*time.Hour, config.CacheEnabled) // 2 hour cache for news

	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("User-Agent", "Mozilla/5.0 (compatible; CortexGo/1.0)")

	return &NewsScraperClient{
		client: client,
		cache:  cache,
	}
}

// GoogleNewsParams represents parameters for Google News search
type GoogleNewsParams struct {
	Query      string    `json:"query"`
	Language   string    `json:"language"`
	Country    string    `json:"country"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	MaxResults int       `json:"max_results"`
}

// GetGoogleNews scrapes Google News for articles
func (ns *NewsScraperClient) GetGoogleNews(params GoogleNewsParams, config *Config) ([]*NewsArticle, error) {
	if strings.TrimSpace(params.Query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// Set defaults
	if params.Language == "" {
		params.Language = "en"
	}
	if params.Country == "" {
		params.Country = "US"
	}
	if params.MaxResults <= 0 {
		params.MaxResults = 20
	}

	// Check cache first
	var cached []*NewsArticle
	if ns.cache.Get("google_news", "search", params, &cached) {
		return cached, nil
	}

	// Build Google News URL
	googleURL := ns.buildGoogleNewsURL(params)

	var result []*NewsArticle
	err := WithRetry(DefaultRetryConfig(), func() error {
		resp, err := ns.client.R().Get(googleURL)
		if err != nil {
			return fmt.Errorf("failed to fetch Google News: %w", err)
		}

		if resp.StatusCode() != 200 {
			return fmt.Errorf("HTTP error %d when fetching Google News", resp.StatusCode())
		}

		// Parse HTML response
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))
		if err != nil {
			return fmt.Errorf("failed to parse HTML: %w", err)
		}

		result = ns.parseGoogleNewsHTML(doc, params.Query)

		// Limit results
		if len(result) > params.MaxResults {
			result = result[:params.MaxResults]
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Cache the result
	ns.cache.Set("google_news", "search", params, result)

	// Save to file
	filePath := filepath.Join(config.DataDir, "news_data",
		fmt.Sprintf("google_news_%s_%s.json",
			strings.ReplaceAll(params.Query, " ", "_"),
			time.Now().Format("2006-01-02")))
	SaveDataToFile(result, filePath)

	return result, nil
}

// buildGoogleNewsURL constructs the Google News search URL
func (ns *NewsScraperClient) buildGoogleNewsURL(params GoogleNewsParams) string {
	baseURL := "https://news.google.com/search"

	query := url.QueryEscape(params.Query)

	// Add date range if specified
	if !params.StartDate.IsZero() && !params.EndDate.IsZero() {
		// Google News uses a specific date format
		dateQuery := fmt.Sprintf(" after:%s before:%s",
			params.StartDate.Format("2006-01-02"),
			params.EndDate.Format("2006-01-02"))
		query += url.QueryEscape(dateQuery)
	}

	return fmt.Sprintf("%s?q=%s&hl=%s&gl=%s&ceid=%s:%s",
		baseURL, query, params.Language, params.Country, params.Country, params.Language)
}

// parseGoogleNewsHTML extracts articles from Google News HTML
func (ns *NewsScraperClient) parseGoogleNewsHTML(doc *goquery.Document, query string) []*NewsArticle {
	var articles []*NewsArticle

	// Google News structure may change, this is a basic implementation
	doc.Find("article").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find("h3").Text())
		if title == "" {
			title = strings.TrimSpace(s.Find("h4").Text())
		}

		if title == "" {
			return // Skip if no title found
		}

		// Extract URL
		link := s.Find("a").First()
		href, exists := link.Attr("href")
		if !exists {
			return
		}

		// Clean up Google News URL
		articleURL := ns.cleanGoogleNewsURL(href)

		// Extract source
		source := strings.TrimSpace(s.Find("div[data-n-tid]").Text())
		if source == "" {
			source = "Google News"
		}

		// Extract time (this is tricky with Google News, often relative)
		timeText := strings.TrimSpace(s.Find("time").Text())
		publishedAt := ns.parseRelativeTime(timeText)

		// Extract snippet/content
		content := strings.TrimSpace(s.Find("span").Last().Text())

		article := &NewsArticle{
			Title:       title,
			Content:     content,
			URL:         articleURL,
			Source:      source,
			PublishedAt: publishedAt,
			Keywords:    []string{query},
			Metadata: map[string]string{
				"scraper":      "google_news",
				"original_url": href,
				"time_text":    timeText,
			},
		}

		articles = append(articles, article)
	})

	return articles
}

// cleanGoogleNewsURL removes Google News redirect wrapper
func (ns *NewsScraperClient) cleanGoogleNewsURL(googleURL string) string {
	// Google News URLs are often wrapped, try to extract the real URL
	if strings.Contains(googleURL, "url=") {
		parts := strings.Split(googleURL, "url=")
		if len(parts) > 1 {
			decoded, err := url.QueryUnescape(parts[1])
			if err == nil {
				return decoded
			}
		}
	}

	// If it's a relative URL, make it absolute
	if strings.HasPrefix(googleURL, "./") {
		return "https://news.google.com" + googleURL[1:]
	}

	if strings.HasPrefix(googleURL, "/") {
		return "https://news.google.com" + googleURL
	}

	return googleURL
}

// parseRelativeTime converts relative time strings to actual time
func (ns *NewsScraperClient) parseRelativeTime(timeText string) time.Time {
	now := time.Now()
	timeText = strings.ToLower(strings.TrimSpace(timeText))

	// Handle common patterns
	patterns := map[string]time.Duration{
		"just now":      0,
		"1 minute ago":  1 * time.Minute,
		"2 minutes ago": 2 * time.Minute,
		"3 minutes ago": 3 * time.Minute,
		"4 minutes ago": 4 * time.Minute,
		"5 minutes ago": 5 * time.Minute,
		"1 hour ago":    1 * time.Hour,
		"2 hours ago":   2 * time.Hour,
		"3 hours ago":   3 * time.Hour,
		"4 hours ago":   4 * time.Hour,
		"5 hours ago":   5 * time.Hour,
		"6 hours ago":   6 * time.Hour,
		"1 day ago":     24 * time.Hour,
		"2 days ago":    48 * time.Hour,
		"3 days ago":    72 * time.Hour,
		"1 week ago":    7 * 24 * time.Hour,
	}

	if duration, exists := patterns[timeText]; exists {
		return now.Add(-duration)
	}

	// Try to extract numbers using regex
	minuteRegex := regexp.MustCompile(`(\d+)\s*minutes?\s*ago`)
	if matches := minuteRegex.FindStringSubmatch(timeText); len(matches) > 1 {
		if minutes := parseNumber(matches[1]); minutes > 0 {
			return now.Add(-time.Duration(minutes) * time.Minute)
		}
	}

	hourRegex := regexp.MustCompile(`(\d+)\s*hours?\s*ago`)
	if matches := hourRegex.FindStringSubmatch(timeText); len(matches) > 1 {
		if hours := parseNumber(matches[1]); hours > 0 {
			return now.Add(-time.Duration(hours) * time.Hour)
		}
	}

	dayRegex := regexp.MustCompile(`(\d+)\s*days?\s*ago`)
	if matches := dayRegex.FindStringSubmatch(timeText); len(matches) > 1 {
		if days := parseNumber(matches[1]); days > 0 {
			return now.Add(-time.Duration(days) * 24 * time.Hour)
		}
	}

	// If we can't parse it, assume it's recent
	return now.Add(-1 * time.Hour)
}

// parseNumber safely converts string to int
func parseNumber(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// GetNewsFromURL scrapes a specific news article URL
func (ns *NewsScraperClient) GetNewsFromURL(articleURL string) (*NewsArticle, error) {
	if strings.TrimSpace(articleURL) == "" {
		return nil, fmt.Errorf("article URL cannot be empty")
	}

	// Check cache first
	var cached NewsArticle
	if ns.cache.Get("article", "content", articleURL, &cached) {
		return &cached, nil
	}

	var result *NewsArticle
	err := WithRetry(DefaultRetryConfig(), func() error {
		resp, err := ns.client.R().Get(articleURL)
		if err != nil {
			return fmt.Errorf("failed to fetch article: %w", err)
		}

		if resp.StatusCode() != 200 {
			return fmt.Errorf("HTTP error %d when fetching article", resp.StatusCode())
		}

		// Parse HTML response
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))
		if err != nil {
			return fmt.Errorf("failed to parse HTML: %w", err)
		}

		result = ns.extractArticleContent(doc, articleURL)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Cache the result
	ns.cache.Set("article", "content", articleURL, result)

	return result, nil
}

// extractArticleContent extracts article content from HTML
func (ns *NewsScraperClient) extractArticleContent(doc *goquery.Document, url string) *NewsArticle {
	// Try to extract title
	title := ""
	titleSelectors := []string{"h1", "title", ".headline", ".article-title", ".entry-title"}
	for _, selector := range titleSelectors {
		if t := strings.TrimSpace(doc.Find(selector).First().Text()); t != "" {
			title = t
			break
		}
	}

	// Try to extract content
	content := ""
	contentSelectors := []string{
		".article-content", ".entry-content", ".post-content",
		".content", "article p", ".article-body", ".story-body",
	}
	for _, selector := range contentSelectors {
		if c := strings.TrimSpace(doc.Find(selector).Text()); c != "" {
			content = c
			break
		}
	}

	// Extract meta information
	source := ""
	if meta := doc.Find("meta[property='og:site_name']"); meta.Length() > 0 {
		source, _ = meta.Attr("content")
	}
	if source == "" {
		// Try to extract from URL
		if u, err := parseURL(url); err == nil {
			source = u.Host
		}
	}

	// Try to extract publish date
	publishedAt := time.Now()
	if meta := doc.Find("meta[property='article:published_time']"); meta.Length() > 0 {
		if dateStr, exists := meta.Attr("content"); exists {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				publishedAt = t
			}
		}
	}

	return &NewsArticle{
		Title:       title,
		Content:     content,
		URL:         url,
		Source:      source,
		PublishedAt: publishedAt,
		Metadata: map[string]string{
			"scraper": "url_content",
		},
	}
}

// parseURL safely parses a URL
func parseURL(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}
