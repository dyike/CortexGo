package cache

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/utils"
)

type MarketDataCache struct {
	memoryCache map[string]*CachedData
	csvManager  *utils.CSVManager
	mu          sync.RWMutex
	basePath    string
}

type CachedData struct {
	Data      []*models.MarketData
	Symbol    string
	Count     int
	Timestamp time.Time
	TTL       time.Duration
}

var (
	globalCache *MarketDataCache
	once        sync.Once
)

func GetMarketDataCache() *MarketDataCache {
	once.Do(func() {
		basePath := "data" // 可以从配置文件读取
		globalCache = &MarketDataCache{
			memoryCache: make(map[string]*CachedData),
			csvManager:  utils.NewCSVManager(basePath),
			basePath:    basePath,
		}
	})
	return globalCache
}

func (c *MarketDataCache) Get(ctx context.Context, symbol string, count int) ([]*models.MarketData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := fmt.Sprintf("%s-%d", symbol, count)

	// 1. 先检查内存缓存
	if cached, exists := c.memoryCache[key]; exists {
		if time.Since(cached.Timestamp) <= cached.TTL {
			log.Printf("Using memory cache for %s (count: %d)", symbol, count)
			return cached.Data, true
		}
		// 内存缓存过期，删除
		delete(c.memoryCache, key)
	}

	// 2. 检查CSV文件缓存
	if csvFile, err := c.csvManager.FindLatestCSV(symbol, count); err == nil {
		if data, fileTime, err := c.csvManager.ReadMarketDataFromCSV(csvFile); err == nil {
			// 检查CSV文件是否过期（30分钟TTL）
			if time.Since(fileTime) <= 30*time.Minute && len(data) >= count {
				log.Printf("Using CSV cache for %s (count: %d) from file: %s", symbol, count, filepath.Base(csvFile))

				// 将数据加载到内存缓存
				c.memoryCache[key] = &CachedData{
					Data:      data[:min(count, len(data))],
					Symbol:    symbol,
					Count:     count,
					Timestamp: fileTime,
					TTL:       5 * time.Minute,
				}

				return data[:min(count, len(data))], true
			} else if time.Since(fileTime) > 30*time.Minute {
				log.Printf("CSV cache expired for %s (age: %v)", symbol, time.Since(fileTime))
			} else if len(data) < count {
				log.Printf("CSV cache has insufficient data for %s (has: %d, need: %d)", symbol, len(data), count)
			}
		} else {
			log.Printf("Failed to read CSV cache for %s: %v", symbol, err)
		}
	}

	return nil, false
}

func (c *MarketDataCache) Set(ctx context.Context, symbol string, count int, data []*models.MarketData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s-%d", symbol, count)
	now := time.Now()

	// 1. 保存到内存缓存
	c.memoryCache[key] = &CachedData{
		Data:      data,
		Symbol:    symbol,
		Count:     count,
		Timestamp: now,
		TTL:       5 * time.Minute,
	}
	log.Printf("Saved to memory cache: %s (count: %d)", symbol, count)

	// 2. 异步保存到CSV文件
	go func() {
		if err := c.csvManager.WriteMarketDataToCSV(symbol, data); err != nil {
			log.Printf("Failed to write market data to CSV for %s: %v", symbol, err)
		} else {
			log.Printf("Successfully saved market data to CSV for %s (count: %d)", symbol, count)
		}
	}()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *MarketDataCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.memoryCache = make(map[string]*CachedData)
	log.Printf("Cleared memory cache")
}

// SaveIndicatorsToCSV 保存技术指标到CSV
func (c *MarketDataCache) SaveIndicatorsToCSV(symbol string, indicators map[string][]models.IndicatorValue) error {
	return c.csvManager.WriteIndicatorResultToCSV(symbol, indicators)
}

// CleanExpiredFiles 清理过期文件
func (c *MarketDataCache) CleanExpiredFiles(maxAge time.Duration) error {
	return c.csvManager.CleanOldCSVFiles(maxAge)
}

// GetCacheStats 获取缓存统计信息
func (c *MarketDataCache) GetCacheStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"memory_cache_size": len(c.memoryCache),
		"base_path":         c.basePath,
	}

	// 统计内存缓存中的项目
	memoryItems := make([]string, 0, len(c.memoryCache))
	for key := range c.memoryCache {
		memoryItems = append(memoryItems, key)
	}
	stats["memory_cache_keys"] = memoryItems

	return stats
}

// WarmupCache 预热缓存，从CSV文件加载常用数据
func (c *MarketDataCache) WarmupCache(ctx context.Context, symbols []string, counts []int) {
	for _, symbol := range symbols {
		for _, count := range counts {
			if _, found := c.Get(ctx, symbol, count); found {
				log.Printf("Warmed up cache for %s (count: %d)", symbol, count)
			}
		}
	}
}
