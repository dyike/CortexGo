package dataflows

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

// RSS 结构体定义
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	Source      Source    `xml:"source"`
	GUID        string    `xml:"guid"`
}

type Source struct {
	URL  string `xml:"url,attr"`
	Text string `xml:",chardata"`
}

// GoogleNewsClient handles Google News operations
type GoogleNewsClient struct {
	client *resty.Client
	cache  *CacheManager
}

// NewGoogleNewsClient creates a new Google News client
func NewGoogleNewsClient(config *Config) *GoogleNewsClient {
	cacheDir := filepath.Join(config.DataCacheDir, "google_news")
	cache := NewCacheManager(cacheDir, 30*time.Minute, config.CacheEnabled) // 30 minute cache for news

	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	return &GoogleNewsClient{
		client: client,
		cache:  cache,
	}
}

// EnhancedGoogleNewsParams represents enhanced parameters for Google News search
type EnhancedGoogleNewsParams struct {
	Query      string    `json:"query"`
	Language   string    `json:"language"`   // en, zh-CN, zh-TW, etc.
	Country    string    `json:"country"`    // US, CN, TW, etc.
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	MaxResults int       `json:"max_results"`
	SortBy     string    `json:"sort_by"`    // date, relevance
	Category   string    `json:"category"`   // business, technology, health, etc.
	Site       string    `json:"site"`       // specific site like "reuters.com"
}

// GetGoogleNews performs enhanced Google News search
func (gnc *GoogleNewsClient) GetGoogleNews(params EnhancedGoogleNewsParams, config *Config) ([]*NewsArticle, error) {
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
	if params.SortBy == "" {
		params.SortBy = "date"
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s_%s_%s_%s_%d", params.Query, params.Language, params.Country, params.SortBy, params.MaxResults)
	var cached []*NewsArticle
	if gnc.cache.Get("enhanced_search", "query", cacheKey, &cached) {
		return cached, nil
	}

	// Try multiple search strategies
	var allResults []*NewsArticle

	// Strategy 1: Direct Google News search
	newsResults, err := gnc.searchGoogleNewsDirect(params)
	if err == nil {
		allResults = append(allResults, newsResults...)
	}

	// Strategy 2: Google Search with news filter
	if len(allResults) < params.MaxResults {
		googleResults, err := gnc.searchGoogleWithNewsFilter(params)
		if err == nil {
			allResults = append(allResults, googleResults...)
		}
	}

	// Remove duplicates and limit results
	allResults = gnc.removeDuplicates(allResults)
	if len(allResults) > params.MaxResults {
		allResults = allResults[:params.MaxResults]
	}

	// Cache the result
	gnc.cache.Set("enhanced_search", "query", cacheKey, allResults)

	// Save to file
	filePath := filepath.Join(config.DataDir, "news_data",
		fmt.Sprintf("google_news_enhanced_%s_%s.json",
			strings.ReplaceAll(params.Query, " ", "_"),
			time.Now().Format("2006-01-02")))
	SaveDataToFile(allResults, filePath)

	return allResults, nil
}

// searchGoogleNewsDirect searches Google News directly
func (gnc *GoogleNewsClient) searchGoogleNewsDirect(params EnhancedGoogleNewsParams) ([]*NewsArticle, error) {
	searchURL := gnc.buildGoogleNewsURL(params)

	var result []*NewsArticle
	err := WithRetry(DefaultRetryConfig(), func() error {
		resp, err := gnc.client.R().Get(searchURL)
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

		result = gnc.parseGoogleNewsHTML(doc, params.Query)
		return nil
	})

	return result, err
}

// searchGoogleWithNewsFilter searches Google with news filter
func (gnc *GoogleNewsClient) searchGoogleWithNewsFilter(params EnhancedGoogleNewsParams) ([]*NewsArticle, error) {
	searchURL := gnc.buildGoogleSearchNewsURL(params)

	var result []*NewsArticle
	err := WithRetry(DefaultRetryConfig(), func() error {
		resp, err := gnc.client.R().Get(searchURL)
		if err != nil {
			return fmt.Errorf("failed to fetch Google Search News: %w", err)
		}

		if resp.StatusCode() != 200 {
			return fmt.Errorf("HTTP error %d when fetching Google Search News", resp.StatusCode())
		}

		// Parse HTML response
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))
		if err != nil {
			return fmt.Errorf("failed to parse HTML: %w", err)
		}

		result = gnc.parseGoogleSearchNewsHTML(doc, params.Query)
		return nil
	})

	return result, err
}

// buildGoogleNewsURL constructs the Google News search URL
func (gnc *GoogleNewsClient) buildGoogleNewsURL(params EnhancedGoogleNewsParams) string {
	baseURL := "https://news.google.com/search"

	query := url.QueryEscape(params.Query)

	// Add category filter
	if params.Category != "" {
		query += url.QueryEscape(fmt.Sprintf(" category:%s", params.Category))
	}

	// Add site filter
	if params.Site != "" {
		query += url.QueryEscape(fmt.Sprintf(" site:%s", params.Site))
	}

	// Add date range if specified
	if !params.StartDate.IsZero() && !params.EndDate.IsZero() {
		dateQuery := fmt.Sprintf(" after:%s before:%s",
			params.StartDate.Format("2006-01-02"),
			params.EndDate.Format("2006-01-02"))
		query += url.QueryEscape(dateQuery)
	}

	return fmt.Sprintf("%s?q=%s&hl=%s&gl=%s&ceid=%s:%s",
		baseURL, query, params.Language, params.Country, params.Country, params.Language)
}

// buildGoogleSearchNewsURL constructs the Google Search with news filter URL
func (gnc *GoogleNewsClient) buildGoogleSearchNewsURL(params EnhancedGoogleNewsParams) string {
	baseURL := "https://www.google.com/search"

	query := url.QueryEscape(params.Query)

	// Add site filter
	if params.Site != "" {
		query += url.QueryEscape(fmt.Sprintf(" site:%s", params.Site))
	}

	values := url.Values{}
	values.Set("q", params.Query)
	values.Set("tbm", "nws") // News search
	values.Set("hl", params.Language)
	values.Set("gl", params.Country)
	values.Set("num", fmt.Sprintf("%d", params.MaxResults))

	// Add sort parameter
	if params.SortBy == "date" {
		values.Set("tbs", "sbd:1") // Sort by date
	}

	return fmt.Sprintf("%s?%s", baseURL, values.Encode())
}

// parseGoogleNewsHTML extracts articles from Google News HTML
func (gnc *GoogleNewsClient) parseGoogleNewsHTML(doc *goquery.Document, query string) []*NewsArticle {
	var articles []*NewsArticle

	// Try multiple selectors for Google News structure
	selectors := []string{
		"article",
		"[data-n-tid]",
		".JtKRv",
		".WwrzSb",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			article := gnc.extractGoogleNewsArticle(s, query)
			if article != nil {
				articles = append(articles, article)
			}
		})

		if len(articles) > 0 {
			break // Use the first successful selector
		}
	}

	return articles
}

// parseGoogleSearchNewsHTML extracts articles from Google Search News HTML
func (gnc *GoogleNewsClient) parseGoogleSearchNewsHTML(doc *goquery.Document, query string) []*NewsArticle {
	var articles []*NewsArticle

	doc.Find(".SoaBEf, .WlydOe, .g").Each(func(i int, s *goquery.Selection) {
		article := gnc.extractGoogleSearchNewsArticle(s, query)
		if article != nil {
			articles = append(articles, article)
		}
	})

	return articles
}

// extractGoogleNewsArticle extracts a single article from Google News
func (gnc *GoogleNewsClient) extractGoogleNewsArticle(s *goquery.Selection, query string) *NewsArticle {
	// Extract title
	title := ""
	titleSelectors := []string{"h3", "h4", "[role='heading']", ".JtKRv"}
	for _, sel := range titleSelectors {
		if t := strings.TrimSpace(s.Find(sel).Text()); t != "" {
			title = t
			break
		}
	}

	if title == "" {
		return nil
	}

	// Extract URL
	link := s.Find("a").First()
	href, exists := link.Attr("href")
	if !exists {
		return nil
	}

	articleURL := gnc.cleanGoogleURL(href)

	// Extract source
	source := strings.TrimSpace(s.Find("[data-n-tid], .wEwyrc").Text())
	if source == "" {
		source = "Google News"
	}

	// Extract time
	timeText := strings.TrimSpace(s.Find("time, .WW6dff").Text())
	publishedAt := gnc.parseTimeText(timeText)

	// Extract content/snippet
	content := strings.TrimSpace(s.Find(".st, .Y3v8qd").Text())

	return &NewsArticle{
		Title:       title,
		Content:     content,
		URL:         articleURL,
		Source:      source,
		PublishedAt: publishedAt,
		Keywords:    []string{query},
		Metadata: map[string]string{
			"scraper":      "google_news_enhanced",
			"original_url": href,
			"time_text":    timeText,
		},
	}
}

// extractGoogleSearchNewsArticle extracts a single article from Google Search News
func (gnc *GoogleNewsClient) extractGoogleSearchNewsArticle(s *goquery.Selection, query string) *NewsArticle {
	// Extract title
	title := strings.TrimSpace(s.Find("h3, .LC20lb").Text())
	if title == "" {
		return nil
	}

	// Extract URL
	link := s.Find("a").First()
	href, exists := link.Attr("href")
	if !exists {
		return nil
	}

	articleURL := gnc.cleanGoogleURL(href)

	// Extract source and time
	sourceTimeText := strings.TrimSpace(s.Find(".fG8Fp, .slp").Text())
	source, timeText := gnc.parseSourceTime(sourceTimeText)

	publishedAt := gnc.parseTimeText(timeText)

	// Extract snippet
	content := strings.TrimSpace(s.Find(".st, .s3v9rd").Text())

	return &NewsArticle{
		Title:       title,
		Content:     content,
		URL:         articleURL,
		Source:      source,
		PublishedAt: publishedAt,
		Keywords:    []string{query},
		Metadata: map[string]string{
			"scraper":         "google_search_news",
			"original_url":    href,
			"source_time_raw": sourceTimeText,
		},
	}
}

// cleanGoogleURL cleans Google redirect URLs
func (gnc *GoogleNewsClient) cleanGoogleURL(googleURL string) string {
	// Handle Google News redirects
	if strings.Contains(googleURL, "/url?") {
		parts := strings.Split(googleURL, "url=")
		if len(parts) > 1 {
			decoded, err := url.QueryUnescape(parts[1])
			if err == nil {
				// Remove additional parameters
				if idx := strings.Index(decoded, "&"); idx != -1 {
					decoded = decoded[:idx]
				}
				return decoded
			}
		}
	}

	// Handle relative URLs
	if strings.HasPrefix(googleURL, "./") {
		return "https://news.google.com" + googleURL[1:]
	}

	if strings.HasPrefix(googleURL, "/") {
		return "https://news.google.com" + googleURL
	}

	return googleURL
}

// parseSourceTime parses source and time from combined text
func (gnc *GoogleNewsClient) parseSourceTime(text string) (source, timeText string) {
	// Common patterns: "Source - 3 hours ago", "Source · 1 day ago"
	separators := []string{" - ", " · ", " — ", " | "}

	for _, sep := range separators {
		if parts := strings.Split(text, sep); len(parts) >= 2 {
			source = strings.TrimSpace(parts[0])
			timeText = strings.TrimSpace(parts[len(parts)-1])
			return
		}
	}

	// If no separator found, assume the whole text is source
	source = text
	return
}

// parseTimeText converts time text to actual time
func (gnc *GoogleNewsClient) parseTimeText(timeText string) time.Time {
	now := time.Now()
	timeText = strings.ToLower(strings.TrimSpace(timeText))

	// Handle various time formats
	patterns := map[*regexp.Regexp]func([]string) time.Duration{
		regexp.MustCompile(`(\d+)\s*minutes?\s*ago`): func(matches []string) time.Duration {
			if len(matches) > 1 {
				if mins := parseNumber(matches[1]); mins > 0 {
					return time.Duration(mins) * time.Minute
				}
			}
			return 0
		},
		regexp.MustCompile(`(\d+)\s*hours?\s*ago`): func(matches []string) time.Duration {
			if len(matches) > 1 {
				if hours := parseNumber(matches[1]); hours > 0 {
					return time.Duration(hours) * time.Hour
				}
			}
			return 0
		},
		regexp.MustCompile(`(\d+)\s*days?\s*ago`): func(matches []string) time.Duration {
			if len(matches) > 1 {
				if days := parseNumber(matches[1]); days > 0 {
					return time.Duration(days) * 24 * time.Hour
				}
			}
			return 0
		},
	}

	for pattern, handler := range patterns {
		if matches := pattern.FindStringSubmatch(timeText); len(matches) > 0 {
			if duration := handler(matches); duration > 0 {
				return now.Add(-duration)
			}
		}
	}

	// Default to recent time if can't parse
	return now.Add(-1 * time.Hour)
}

// removeDuplicates removes duplicate articles based on URL and title
func (gnc *GoogleNewsClient) removeDuplicates(articles []*NewsArticle) []*NewsArticle {
	seen := make(map[string]bool)
	var unique []*NewsArticle

	for _, article := range articles {
		key := fmt.Sprintf("%s|%s", article.URL, article.Title)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, article)
		}
	}

	return unique
}

// GetFinanceNews gets finance-related news from Google News
func (gnc *GoogleNewsClient) GetFinanceNews(maxResults int, config *Config) ([]*NewsArticle, error) {
	financeQueries := []string{
		"stock market",
		"financial news",
		"economy",
		"trading",
		"investment",
	}

	var allArticles []*NewsArticle
	articlesPerQuery := maxResults / len(financeQueries)
	if articlesPerQuery < 1 {
		articlesPerQuery = 1
	}

	for _, query := range financeQueries {
		params := EnhancedGoogleNewsParams{
			Query:      query,
			Language:   "en",
			Country:    "US",
			MaxResults: articlesPerQuery,
			SortBy:     "date",
			Category:   "business",
		}

		articles, err := gnc.GetGoogleNews(params, config)
		if err != nil {
			continue // Skip failed queries
		}

		allArticles = append(allArticles, articles...)
	}

	// Remove duplicates and sort by time
	allArticles = gnc.removeDuplicates(allArticles)

	// Sort by published time (newest first)
	for i := 0; i < len(allArticles)-1; i++ {
		for j := i + 1; j < len(allArticles); j++ {
			if allArticles[i].PublishedAt.Before(allArticles[j].PublishedAt) {
				allArticles[i], allArticles[j] = allArticles[j], allArticles[i]
			}
		}
	}

	if len(allArticles) > maxResults {
		allArticles = allArticles[:maxResults]
	}

	return allArticles, nil
}

// GetStockNews gets news for a specific stock symbol
func (gnc *GoogleNewsClient) GetStockNews(symbol string, maxResults int, config *Config) ([]*NewsArticle, error) {
	if strings.TrimSpace(symbol) == "" {
		return nil, fmt.Errorf("stock symbol cannot be empty")
	}

	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// Try multiple search queries for the stock
	queries := []string{
		fmt.Sprintf("%s stock news", symbol),
		fmt.Sprintf("%s earnings", symbol),
		fmt.Sprintf("%s financial", symbol),
		symbol, // Just the symbol
	}

	var allArticles []*NewsArticle
	articlesPerQuery := maxResults / len(queries)
	if articlesPerQuery < 1 {
		articlesPerQuery = 1
	}

	for _, query := range queries {
		params := EnhancedGoogleNewsParams{
			Query:      query,
			Language:   "en",
			Country:    "US",
			MaxResults: articlesPerQuery,
			SortBy:     "date",
			Category:   "business",
		}

		articles, err := gnc.GetGoogleNews(params, config)
		if err != nil {
			continue
		}

		// Filter articles that actually mention the stock symbol
		var relevantArticles []*NewsArticle
		for _, article := range articles {
			if gnc.containsStockSymbol(article, symbol) {
				relevantArticles = append(relevantArticles, article)
			}
		}

		allArticles = append(allArticles, relevantArticles...)
	}

	// Remove duplicates and limit results
	allArticles = gnc.removeDuplicates(allArticles)
	if len(allArticles) > maxResults {
		allArticles = allArticles[:maxResults]
	}

	return allArticles, nil
}

// containsStockSymbol checks if an article mentions a stock symbol
func (gnc *GoogleNewsClient) containsStockSymbol(article *NewsArticle, symbol string) bool {
	text := strings.ToUpper(article.Title + " " + article.Content)

	// Check for various formats
	patterns := []string{
		fmt.Sprintf(" %s ", symbol),      // " AAPL "
		fmt.Sprintf("(%s)", symbol),      // (AAPL)
		fmt.Sprintf("%s:", symbol),       // AAPL:
		fmt.Sprintf("$%s", symbol),       // $AAPL
	}

	for _, pattern := range patterns {
		if strings.Contains(text, strings.ToUpper(pattern)) {
			return true
		}
	}

	// Use regex for word boundary matching
	regex := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(symbol)))
	return regex.MatchString(text)
}

// parseNumber safely converts string to int
func parseNumber(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// GetArticleContent 获取文章的具体内容
func (gnc *GoogleNewsClient) GetArticleContent(articleURL string) (string, error) {
	if strings.TrimSpace(articleURL) == "" {
		return "", fmt.Errorf("article URL cannot be empty")
	}

	// 检查缓存
	var cached string
	if gnc.cache.Get("article_content", "url", articleURL, &cached) {
		return cached, nil
	}

	// 如果是Google News链接，尝试获取实际目标URL
	actualURL := articleURL
	if strings.Contains(articleURL, "news.google.com") {
		if redirectURL, err := gnc.followRedirect(articleURL); err == nil && redirectURL != "" {
			actualURL = redirectURL
			fmt.Printf("  跟随重定向到: %s\n", actualURL)
		}
	}

	var content string
	err := WithRetry(DefaultRetryConfig(), func() error {
		resp, err := gnc.client.R().Get(actualURL)
		if err != nil {
			return fmt.Errorf("failed to fetch article: %w", err)
		}

		if resp.StatusCode() != 200 {
			return fmt.Errorf("HTTP error %d when fetching article", resp.StatusCode())
		}

		// 解析HTML内容
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))
		if err != nil {
			return fmt.Errorf("failed to parse HTML: %w", err)
		}

		content = gnc.extractArticleContent(doc)
		return nil
	})

	if err != nil {
		return "", err
	}

	// 缓存结果
	gnc.cache.Set("article_content", "url", articleURL, content)

	return content, nil
}

// followRedirect 跟随Google News重定向获取实际URL
func (gnc *GoogleNewsClient) followRedirect(googleURL string) (string, error) {
	// 对于Google News的read链接，尝试解析页面中的实际链接
	resp, err := gnc.client.R().Get(googleURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Google News page: %w", err)
	}

	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("HTTP error %d when fetching Google News page", resp.StatusCode())
	}

	// 解析HTML内容查找实际的新闻链接
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))
	if err != nil {
		return "", fmt.Errorf("failed to parse Google News HTML: %w", err)
	}

	// 查找可能的原始新闻链接
	var actualURL string

	// 尝试从各种可能的选择器中找到原始链接
	selectors := []string{
		"a[href*='http']",
		"article a",
		".article a",
		"[data-url]",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			if actualURL != "" {
				return // 已经找到了
			}

			href := s.AttrOr("href", "")
			if href == "" {
				href = s.AttrOr("data-url", "")
			}

			// 检查是否是有效的新闻网站URL
			if href != "" && !strings.Contains(href, "google.com") &&
			   (strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://")) {
				actualURL = href
			}
		})

		if actualURL != "" {
			break
		}
	}

	if actualURL == "" {
		return googleURL, nil // 如果找不到，返回原URL
	}

	return actualURL, nil
}

// extractArticleContent 从HTML中提取文章内容
func (gnc *GoogleNewsClient) extractArticleContent(doc *goquery.Document) string {
	var contentParts []string

	// 尝试多种内容选择器，按优先级排序
	contentSelectors := []string{
		// 通用文章内容选择器
		"article p",
		".article-content p",
		".entry-content p",
		".post-content p",
		".content p",
		".story-body p",
		".article-body p",
		// 更通用的段落选择器
		"main p",
		".main p",
		"#content p",
		// 最后尝试所有段落
		"p",
	}

	for _, selector := range contentSelectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if len(text) > 20 && !gnc.isNavigationText(text) {
				contentParts = append(contentParts, text)
			}
		})

		// 如果找到足够的内容，就停止
		if len(contentParts) >= 3 {
			break
		}
	}

	// 合并内容段落
	fullContent := strings.Join(contentParts, "\n\n")

	// 如果内容太短，尝试其他方法
	if len(fullContent) < 50 {
		// 尝试获取meta description
		if desc := doc.Find("meta[name='description']").AttrOr("content", ""); desc != "" {
			return desc
		}

		// 尝试获取og:description
		if ogDesc := doc.Find("meta[property='og:description']").AttrOr("content", ""); ogDesc != "" {
			return ogDesc
		}
	}

	return fullContent
}

// isNavigationText 判断是否为导航文本（需要过滤掉）
func (gnc *GoogleNewsClient) isNavigationText(text string) bool {
	text = strings.ToLower(text)

	// 常见的导航和无关文本
	navigationPatterns := []string{
		"subscribe", "sign in", "menu", "search", "home", "about",
		"contact", "privacy", "terms", "cookie", "advertisement",
		"share", "tweet", "facebook", "linkedin", "email",
		"read more", "continue reading", "related articles",
		"you may also like", "recommended", "trending",
	}

	for _, pattern := range navigationPatterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}

	return false
}

// GetNewsWithContent 获取新闻并同时提取内容
func (gnc *GoogleNewsClient) GetNewsWithContent(params EnhancedGoogleNewsParams, config *Config) ([]*NewsArticle, error) {
	// 首先获取新闻列表
	articles, err := gnc.GetGoogleNews(params, config)
	if err != nil {
		return nil, err
	}

	// 为每篇文章获取内容（限制数量避免过多请求）
	maxContentArticles := params.MaxResults
	if maxContentArticles > 5 {
		maxContentArticles = 5 // 最多5篇
	}
	if len(articles) < maxContentArticles {
		maxContentArticles = len(articles)
	}

	fmt.Printf("正在获取 %d 篇文章的具体内容...\n", maxContentArticles)

	for i := 0; i < maxContentArticles; i++ {
		article := articles[i]

		fmt.Printf("获取文章内容 %d/%d: %s\n", i+1, maxContentArticles, article.Title)

		content, err := gnc.GetArticleContent(article.URL)
		if err != nil {
			fmt.Printf("  获取内容失败: %v\n", err)
			continue
		}

		if len(content) > 50 {
			article.Content = content
			fmt.Printf("  成功获取 %d 字符内容\n", len(content))
		} else {
			fmt.Printf("  内容太短: %d 字符 - %s\n", len(content), content)
		}

		// 添加延迟避免请求太频繁
		time.Sleep(1 * time.Second)
	}

	return articles, nil
}

// GetGoogleNewsRSS 通过RSS feed获取Google News
func (gnc *GoogleNewsClient) GetGoogleNewsRSS(params EnhancedGoogleNewsParams, config *Config) ([]*NewsArticle, error) {
	rssURL := gnc.buildGoogleNewsRSSURL(params)

	fmt.Printf("📡 正在通过RSS获取Google News: %s\n", params.Query)

	// 检查缓存
	cacheKey := fmt.Sprintf("rss_%s_%s_%s", params.Query, params.Language, params.Country)
	var cached []*NewsArticle
	if gnc.cache.Get("google_news_rss", "query", cacheKey, &cached) {
		fmt.Printf("✅ 从缓存获取到 %d 篇RSS文章\n", len(cached))
		return cached, nil
	}

	var articles []*NewsArticle
	err := WithRetry(DefaultRetryConfig(), func() error {
		resp, err := gnc.client.R().Get(rssURL)
		if err != nil {
			return fmt.Errorf("failed to fetch RSS feed: %w", err)
		}

		if resp.StatusCode() != 200 {
			return fmt.Errorf("HTTP error %d when fetching RSS feed", resp.StatusCode())
		}

		// 解析RSS XML
		var rss RSS
		if err := xml.Unmarshal(resp.Body(), &rss); err != nil {
			return fmt.Errorf("failed to parse RSS XML: %w", err)
		}

		// 转换RSS项目为NewsArticle
		for i, item := range rss.Channel.Items {
			if params.MaxResults > 0 && i >= params.MaxResults {
				break
			}

			article := gnc.convertRSSItemToNewsArticle(item, params.Query)
			articles = append(articles, article)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 缓存结果
	gnc.cache.Set("google_news_rss", "query", cacheKey, articles)

	// 保存到文件
	if config.DataDir != "" {
		gnc.saveRSSArticlesToFile(articles, params, config.DataDir)
	}

	fmt.Printf("✅ RSS模式获取到 %d 篇文章\n", len(articles))
	return articles, nil
}

// buildGoogleNewsRSSURL 构建Google News RSS URL
func (gnc *GoogleNewsClient) buildGoogleNewsRSSURL(params EnhancedGoogleNewsParams) string {
	baseURL := "https://news.google.com/rss"

	// 构建查询参数
	v := url.Values{}

	// 添加搜索查询
	if params.Query != "" {
		if params.Site != "" {
			// 如果指定了网站，使用site:语法
			v.Set("q", fmt.Sprintf("%s site:%s", params.Query, params.Site))
		} else {
			v.Set("q", params.Query)
		}
	}

	// 添加语言和地区
	if params.Language != "" {
		v.Set("hl", params.Language)
	}
	if params.Country != "" {
		v.Set("gl", params.Country)
		v.Set("ceid", fmt.Sprintf("%s:%s", params.Country, strings.Split(params.Language, "-")[0]))
	}

	// 构建最终URL
	if len(v) > 0 {
		return baseURL + "/search?" + v.Encode()
	}

	// 如果没有搜索参数，使用分类URL
	if params.Category != "" {
		switch params.Category {
		case "business":
			return baseURL + "/topics/CAAqJggKIiBDQkFTRWdvSUwyMHZNRFZ4ZERBU0FtVnVHZ0pWVXlnQVAB"
		case "technology":
			return baseURL + "/topics/CAAqJggKIiBDQkFTRWdvSUwyMHZNRGx6TVdZU0FtVnVHZ0pWVXlnQVAB"
		case "health":
			return baseURL + "/topics/CAAqJQgKIh9DQkFTRVFvSUwyMHZNR3QwTlRFU0FtVnVHZ0pWVXlnQVAB"
		}
	}

	return baseURL
}

// convertRSSItemToNewsArticle 将RSS项目转换为NewsArticle
func (gnc *GoogleNewsClient) convertRSSItemToNewsArticle(item Item, query string) *NewsArticle {
	// 解析发布时间
	pubTime, err := time.Parse(time.RFC1123Z, item.PubDate)
	if err != nil {
		// 尝试其他时间格式
		pubTime, _ = time.Parse("Mon, 02 Jan 2006 15:04:05 MST", item.PubDate)
	}
	if pubTime.IsZero() {
		pubTime = time.Now()
	}

	// 提取来源信息
	source := item.Source.Text
	if source == "" && item.Source.URL != "" {
		// 从URL提取域名作为来源
		if u, err := url.Parse(item.Source.URL); err == nil {
			source = u.Host
		}
	}

	// 清理HTML标签从description中获取纯文本内容
	cleanContent := gnc.cleanHTMLContent(item.Description)

	return &NewsArticle{
		Title:       strings.TrimSpace(item.Title),
		Content:     cleanContent,
		URL:         item.Link,
		Source:      source,
		PublishedAt: pubTime,
		Keywords:    []string{query},
		Metadata: map[string]string{
			"scraper":     "google_news_rss",
			"guid":        item.GUID,
			"source_url":  item.Source.URL,
		},
	}
}

// saveRSSArticlesToFile 保存RSS文章到文件
func (gnc *GoogleNewsClient) saveRSSArticlesToFile(articles []*NewsArticle, params EnhancedGoogleNewsParams, dataDir string) {
	newsDir := filepath.Join(dataDir, "news_data")

	// 创建目录
	if err := os.MkdirAll(newsDir, 0755); err != nil {
		fmt.Printf("无法创建目录 %s: %v\n", newsDir, err)
		return
	}

	// 生成文件名
	today := time.Now().Format("2006-01-02")
	querySlug := strings.ReplaceAll(strings.ToLower(params.Query), " ", "_")
	filename := fmt.Sprintf("google_news_rss_%s_%s.json", querySlug, today)
	filePath := filepath.Join(newsDir, filename)

	// 保存文章为JSON
	data, err := json.MarshalIndent(articles, "", "  ")
	if err != nil {
		fmt.Printf("无法序列化文章数据: %v\n", err)
		return
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		fmt.Printf("无法写入文件 %s: %v\n", filePath, err)
		return
	}

	fmt.Printf("📁 RSS文章已保存到: %s\n", filePath)
}

// cleanHTMLContent 清理HTML标签并提取纯文本内容
func (gnc *GoogleNewsClient) cleanHTMLContent(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}

	// 使用goquery解析HTML并提取文本
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		// 如果解析失败，使用正则表达式简单去除HTML标签
		return gnc.stripHTMLTags(htmlContent)
	}

	// 提取所有文本内容
	text := strings.TrimSpace(doc.Text())

	// 如果提取的文本为空，尝试正则表达式方法
	if text == "" {
		return gnc.stripHTMLTags(htmlContent)
	}

	return text
}

// stripHTMLTags 使用正则表达式去除HTML标签
func (gnc *GoogleNewsClient) stripHTMLTags(content string) string {
	// 去除HTML标签
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	content = htmlTagRegex.ReplaceAllString(content, "")

	// 解码HTML实体
	content = strings.ReplaceAll(content, "&nbsp;", " ")
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&quot;", "\"")
	content = strings.ReplaceAll(content, "&#39;", "'")

	// 清理多余的空白字符
	spaceRegex := regexp.MustCompile(`\s+`)
	content = spaceRegex.ReplaceAllString(content, " ")

	return strings.TrimSpace(content)
}

// GetDirectNewsRSS 直接从各大新闻源RSS获取带真实摘要的新闻
func (gnc *GoogleNewsClient) GetDirectNewsRSS(query string, maxResults int, config *Config) ([]*NewsArticle, error) {
	fmt.Printf("📡 正在从多个直接新闻源获取: %s\n", query)

	// 主要新闻源RSS列表
	rssFeeds := []struct {
		Name string
		URL  string
	}{
		{"BBC News", "https://feeds.bbci.co.uk/news/rss.xml"},
		{"Reuters", "https://www.reuters.com/tools/rss"},
		{"CNN", "http://rss.cnn.com/rss/edition.rss"},
		{"TechCrunch", "https://techcrunch.com/feed/"},
		{"Yahoo Finance", "https://finance.yahoo.com/rss/"},
	}

	var allArticles []*NewsArticle

	for _, feed := range rssFeeds {
		fmt.Printf("  正在获取 %s...\n", feed.Name)

		articles, err := gnc.fetchDirectRSSFeed(feed.URL, feed.Name, query, maxResults/len(rssFeeds))
		if err != nil {
			fmt.Printf("  ⚠️ %s 获取失败: %v\n", feed.Name, err)
			continue
		}

		allArticles = append(allArticles, articles...)
		if len(allArticles) >= maxResults {
			break
		}
	}

	// 限制结果数量
	if len(allArticles) > maxResults {
		allArticles = allArticles[:maxResults]
	}

	fmt.Printf("✅ 从直接RSS源获取到 %d 篇文章\n", len(allArticles))
	return allArticles, nil
}

// fetchDirectRSSFeed 从单个RSS源获取新闻
func (gnc *GoogleNewsClient) fetchDirectRSSFeed(rssURL, sourceName, query string, maxResults int) ([]*NewsArticle, error) {
	var articles []*NewsArticle

	err := WithRetry(DefaultRetryConfig(), func() error {
		resp, err := gnc.client.R().Get(rssURL)
		if err != nil {
			return fmt.Errorf("failed to fetch RSS feed: %w", err)
		}

		if resp.StatusCode() != 200 {
			return fmt.Errorf("HTTP error %d", resp.StatusCode())
		}

		// 解析RSS XML
		var rss RSS
		if err := xml.Unmarshal(resp.Body(), &rss); err != nil {
			return fmt.Errorf("failed to parse RSS XML: %w", err)
		}

		// 过滤与查询相关的文章
		count := 0
		for _, item := range rss.Channel.Items {
			if count >= maxResults {
				break
			}

			// 简单的关键词匹配
			if query != "" && !gnc.containsKeyword(item.Title+item.Description, query) {
				continue
			}

			article := gnc.convertDirectRSSItem(item, sourceName, query)
			articles = append(articles, article)
			count++
		}

		return nil
	})

	return articles, err
}

// containsKeyword 检查文本是否包含关键词
func (gnc *GoogleNewsClient) containsKeyword(text, keyword string) bool {
	text = strings.ToLower(text)
	keyword = strings.ToLower(keyword)

	// 支持多个关键词
	keywords := strings.Fields(keyword)
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// convertDirectRSSItem 转换直接RSS项目为NewsArticle
func (gnc *GoogleNewsClient) convertDirectRSSItem(item Item, sourceName, query string) *NewsArticle {
	// 解析发布时间
	pubTime, _ := time.Parse(time.RFC1123Z, item.PubDate)
	if pubTime.IsZero() {
		pubTime, _ = time.Parse("Mon, 02 Jan 2006 15:04:05 MST", item.PubDate)
	}
	if pubTime.IsZero() {
		pubTime = time.Now()
	}

	// 清理描述内容
	cleanDescription := gnc.cleanHTMLContent(item.Description)

	return &NewsArticle{
		Title:       strings.TrimSpace(item.Title),
		Content:     cleanDescription, // 这里是真正的摘要内容
		URL:         item.Link,
		Source:      sourceName,
		PublishedAt: pubTime,
		Keywords:    []string{query},
		Metadata: map[string]string{
			"scraper":    "direct_rss",
			"guid":       item.GUID,
			"rss_source": sourceName,
		},
	}
}