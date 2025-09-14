package models

import "time"

// RedditPost represents a Reddit post (duplicated to avoid import cycle)
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

// Reddit tool input/output models
type RedditSubredditInput struct {
	Subreddit string `json:"subreddit"`
	Sort      string `json:"sort"`
	Limit     int    `json:"limit"`
}

type RedditSearchInput struct {
	Query     string `json:"query"`
	Subreddit string `json:"subreddit"`
	Sort      string `json:"sort"`
	Time      string `json:"time"`
	Limit     int    `json:"limit"`
}

type StockMentionsInput struct {
	Symbol string `json:"symbol"`
}

type FinanceNewsInput struct {
	Limit int `json:"limit"`
}

type RedditOutput struct {
	Posts  []*RedditPost `json:"posts"`
	Result string        `json:"result"`
}