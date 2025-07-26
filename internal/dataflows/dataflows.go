package dataflows

import (
	"time"
)

// Package-level interface instance for easy access
var DefaultInterface *DataFlowInterface

// Initialize sets up the default dataflows interface with provided config
func Initialize(config *Config) error {
	DefaultInterface = NewDataFlowInterface(config)
	return nil
}

// GetInterface returns the default dataflows interface
func GetInterface() *DataFlowInterface {
	if DefaultInterface == nil {
		panic("dataflows not initialized - call Initialize(config) first")
	}
	return DefaultInterface
}

// Package-level convenience functions that use the default interface

// Market Data

func GetYFinData(symbol string, start, end time.Time) ([]*MarketData, error) {
	return GetInterface().GetYFinData(symbol, start, end)
}

func GetYFinDataOnline(symbol string) (*MarketData, error) {
	return GetInterface().GetYFinDataOnline(symbol)
}

func GetYFinDataWindow(symbol string, days int) ([]*MarketData, error) {
	return GetInterface().GetYFinDataWindow(symbol, days)
}

func GetCompanyInfo(symbol string) (map[string]interface{}, error) {
	return GetInterface().GetCompanyInfo(symbol)
}

// News

func GetFinnhubNews(symbol string, from, to time.Time) ([]*NewsArticle, error) {
	return GetInterface().GetFinnhubNews(symbol, from, to)
}

func GetFinnhubGeneralNews(category string) ([]*NewsArticle, error) {
	return GetInterface().GetFinnhubGeneralNews(category)
}

func GetGoogleNews(query string, startDate, endDate time.Time, maxResults int) ([]*NewsArticle, error) {
	return GetInterface().GetGoogleNews(query, startDate, endDate, maxResults)
}

func GetGlobalNews() ([]*NewsArticle, error) {
	return GetInterface().GetGlobalNews()
}

// Insider Trading

func GetFinnhubCompanyInsiderSentiment(symbol string, from, to time.Time) ([]*InsiderSentiment, error) {
	return GetInterface().GetFinnhubCompanyInsiderSentiment(symbol, from, to)
}

func GetFinnhubCompanyInsiderTransactions(symbol string, from, to time.Time) ([]*InsiderTransaction, error) {
	return GetInterface().GetFinnhubCompanyInsiderTransactions(symbol, from, to)
}

// Utilities

func GetNewsFromURL(url string) (*NewsArticle, error) {
	return GetInterface().GetNewsFromURL(url)
}

func ValidateAndNormalizeSymbol(symbol string) (string, error) {
	return GetInterface().ValidateAndNormalizeSymbol(symbol)
}

func SearchSymbols(query string) ([]string, error) {
	return GetInterface().SearchSymbols(query)
}

// Convenience functions

func GetRecentNews(symbol string) ([]*NewsArticle, error) {
	return GetInterface().GetRecentNews(symbol)
}

func GetRecentMarketData(symbol string) ([]*MarketData, error) {
	return GetInterface().GetRecentMarketData(symbol)
}

func GetRecentInsiderActivity(symbol string) ([]*InsiderTransaction, error) {
	return GetInterface().GetRecentInsiderActivity(symbol)
}

func GetMarketOverview(symbol string) (map[string]interface{}, error) {
	return GetInterface().GetMarketOverview(symbol)
}

// Batch operations

func GetMultipleSymbolsData(symbols []string, days int) (map[string][]*MarketData, error) {
	return GetInterface().GetMultipleSymbolsData(symbols, days)
}

func GetMultipleSymbolsNews(symbols []string) (map[string][]*NewsArticle, error) {
	return GetInterface().GetMultipleSymbolsNews(symbols)
}