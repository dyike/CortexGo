package main

import (
	"fmt"
	"log"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/pkg/dataflows"
)

func main() {
	// Create a basic config
	cfg := &config.Config{
		OnlineTools:   true,
		CacheEnabled:  true,
		DataCacheDir:  "./cache",
		DataDir:       "./data",
	}

	// Test Reddit client
	redditClient := dataflows.NewRedditClient(cfg)

	fmt.Println("Testing Reddit News Tools...")

	// Test 1: Get posts from a subreddit
	fmt.Println("\n=== Test 1: Get posts from r/stocks ===")
	posts, err := redditClient.GetSubredditPosts("stocks", "hot", 5, cfg)
	if err != nil {
		log.Printf("Error getting subreddit posts: %v", err)
	} else {
		fmt.Printf("Found %d posts from r/stocks\n", len(posts))
		for i, post := range posts {
			fmt.Printf("%d. %s (Score: %d)\n", i+1, post.Title, post.Score)
		}
	}

	// Test 2: Search for stock mentions
	fmt.Println("\n=== Test 2: Search for AAPL mentions ===")
	stockPosts, err := redditClient.GetStockMentions("AAPL", cfg)
	if err != nil {
		log.Printf("Error getting stock mentions: %v", err)
	} else {
		fmt.Printf("Found %d posts mentioning AAPL\n", len(stockPosts))
		for i, post := range stockPosts {
			if i >= 3 { // Limit to first 3
				break
			}
			fmt.Printf("%d. %s (r/%s, Score: %d)\n", i+1, post.Title, post.Subreddit, post.Score)
		}
	}

	// Test 3: Get finance posts
	fmt.Println("\n=== Test 3: Get popular finance posts ===")
	financePosts, err := redditClient.GetPopularFinancePosts(10, cfg)
	if err != nil {
		log.Printf("Error getting finance posts: %v", err)
	} else {
		fmt.Printf("Found %d finance posts\n", len(financePosts))
		for i, post := range financePosts {
			if i >= 5 { // Limit to first 5
				break
			}
			fmt.Printf("%d. %s (r/%s, Score: %d)\n", i+1, post.Title, post.Subreddit, post.Score)
		}
	}

	// Test 4: General Reddit search
	fmt.Println("\n=== Test 4: Search Reddit for 'stock market' ===")
	searchParams := dataflows.RedditSearchParams{
		Query:      "stock market",
		Sort:       "hot",
		Time:       "day",
		Limit:      10,
		MaxResults: 10,
	}

	searchPosts, err := redditClient.SearchReddit(searchParams, cfg)
	if err != nil {
		log.Printf("Error searching Reddit: %v", err)
	} else {
		fmt.Printf("Found %d posts for 'stock market'\n", len(searchPosts))
		for i, post := range searchPosts {
			if i >= 3 { // Limit to first 3
				break
			}
			fmt.Printf("%d. %s (r/%s, Score: %d)\n", i+1, post.Title, post.Subreddit, post.Score)
		}
	}

	fmt.Println("\nReddit News Tools test completed!")
}