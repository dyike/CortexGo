package dataflows

import (
	"time"

	"github.com/dyike/CortexGo/config"
	"github.com/shopspring/decimal"
)

// Config is an alias for the main application config
type Config = config.Config

// MarketData represents stock price data
type MarketData struct {
	Symbol    string          `json:"symbol"`
	Date      time.Time       `json:"date"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
	AdjClose  decimal.Decimal `json:"adj_close"`
	Volume    int64           `json:"volume"`
	Timestamp time.Time       `json:"timestamp"`
}

// NewsArticle represents a news article
type NewsArticle struct {
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	URL         string            `json:"url"`
	Source      string            `json:"source"`
	PublishedAt time.Time         `json:"published_at"`
	Sentiment   float64           `json:"sentiment,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// RedditPost represents a Reddit post
type RedditPost struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	URL        string    `json:"url"`
	Subreddit  string    `json:"subreddit"`
	Author     string    `json:"author"`
	Score      int       `json:"score"`
	Comments   int       `json:"comments"`
	CreatedAt  time.Time `json:"created_at"`
	Sentiment  float64   `json:"sentiment,omitempty"`
	IsStickied bool      `json:"is_stickied"`
	IsLocked   bool      `json:"is_locked"`
}
