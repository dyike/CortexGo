package utils

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dyike/CortexGo/models"
)

type CSVManager struct {
	basePath string
}

func NewCSVManager(basePath string) *CSVManager {
	return &CSVManager{
		basePath: basePath,
	}
}

// WriteMarketDataToCSV 将市场数据写入CSV文件
func (c *CSVManager) WriteMarketDataToCSV(symbol string, data []*models.MarketData) error {
	// 创建目录结构: data/csv/market/{symbol}/
	dirPath := filepath.Join(c.basePath, "csv", "market", symbol)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// 文件名包含时间戳和数据量
	filename := fmt.Sprintf("%s_market_data_%d_records_%s.csv",
		symbol, len(data), time.Now().Format("20060102_150405"))
	filePath := filepath.Join(dirPath, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入标题行
	headers := []string{
		"Symbol", "Date", "Open", "High", "Low", "Close", "Volume",
		"Timestamp", // 用于缓存过期检查
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// 写入数据行
	timestamp := time.Now().Unix()
	for _, record := range data {
		row := []string{
			record.Symbol,
			record.Date,
			strconv.FormatFloat(record.Open, 'f', 4, 64),
			strconv.FormatFloat(record.High, 'f', 4, 64),
			strconv.FormatFloat(record.Low, 'f', 4, 64),
			strconv.FormatFloat(record.Close, 'f', 4, 64),
			strconv.FormatInt(record.Volume, 10),
			strconv.FormatInt(timestamp, 10),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %v", err)
		}
	}

	return nil
}

// ReadMarketDataFromCSV 从CSV文件读取市场数据
func (c *CSVManager) ReadMarketDataFromCSV(filePath string) ([]*models.MarketData, time.Time, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to read CSV: %v", err)
	}

	if len(records) <= 1 {
		return nil, time.Time{}, fmt.Errorf("no data in CSV file")
	}

	var marketData []*models.MarketData
	var fileTimestamp time.Time

	// 跳过标题行，处理数据行
	for i, record := range records[1:] {
		if len(record) < 8 {
			continue // 跳过格式不正确的行
		}

		open, _ := strconv.ParseFloat(record[2], 64)
		high, _ := strconv.ParseFloat(record[3], 64)
		low, _ := strconv.ParseFloat(record[4], 64)
		close, _ := strconv.ParseFloat(record[5], 64)
		volume, _ := strconv.ParseInt(record[6], 10, 64)

		if i == 0 {
			// 从第一行获取文件时间戳
			if ts, err := strconv.ParseInt(record[7], 10, 64); err == nil {
				fileTimestamp = time.Unix(ts, 0)
			}
		}

		marketData = append(marketData, &models.MarketData{
			Symbol: record[0],
			Date:   record[1],
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		})
	}

	return marketData, fileTimestamp, nil
}

// FindLatestCSV 查找最新的CSV文件
func (c *CSVManager) FindLatestCSV(symbol string, minRecords int) (string, error) {
	dirPath := filepath.Join(c.basePath, "csv", "market", symbol)

	// 检查目录是否存在
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return "", fmt.Errorf("no CSV directory for symbol %s", symbol)
	}

	files, err := filepath.Glob(filepath.Join(dirPath, fmt.Sprintf("%s_market_data_*_records_*.csv", symbol)))
	if err != nil {
		return "", fmt.Errorf("failed to search CSV files: %v", err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no CSV files found for symbol %s", symbol)
	}

	// 找到最新且记录数满足要求的文件
	var bestFile string
	var latestTime time.Time

	for _, file := range files {
		// 从文件名解析记录数
		base := filepath.Base(file)
		parts := strings.Split(base, "_")
		if len(parts) < 4 {
			continue
		}

		recordCount, err := strconv.Atoi(parts[3]) // 假设格式为 SYMBOL_market_data_COUNT_records_TIMESTAMP.csv
		if err != nil {
			continue
		}

		// 只考虑记录数足够的文件
		if recordCount < minRecords {
			continue
		}

		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			bestFile = file
		}
	}

	if bestFile == "" {
		return "", fmt.Errorf("no suitable CSV file found for symbol %s with at least %d records", symbol, minRecords)
	}

	return bestFile, nil
}

// WriteIndicatorResultToCSV 将技术指标结果写入CSV
func (c *CSVManager) WriteIndicatorResultToCSV(symbol string, indicators map[string][]models.IndicatorValue) error {
	dirPath := filepath.Join(c.basePath, "csv", "indicators", symbol)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	filename := fmt.Sprintf("%s_indicators_%s.csv", symbol, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(dirPath, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 收集所有日期和指标名称
	allDates := make(map[string]bool)
	indicatorNames := make([]string, 0, len(indicators))

	for name, values := range indicators {
		indicatorNames = append(indicatorNames, name)
		for _, value := range values {
			allDates[value.Date] = true
		}
	}

	// 对指标名称排序
	sort.Strings(indicatorNames)

	// 转换为排序的日期列表
	dates := make([]string, 0, len(allDates))
	for date := range allDates {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	// 写入标题行
	headers := []string{"Date"}
	headers = append(headers, indicatorNames...)
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// 按日期写入数据
	for _, date := range dates {
		row := []string{date}
		for _, indicatorName := range indicatorNames {
			value := ""
			if values, exists := indicators[indicatorName]; exists {
				for _, v := range values {
					if v.Date == date {
						value = strconv.FormatFloat(v.Value, 'f', 6, 64)
						break
					}
				}
			}
			row = append(row, value)
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %v", err)
		}
	}

	return nil
}

// CleanOldCSVFiles 清理过期的CSV文件
func (c *CSVManager) CleanOldCSVFiles(maxAge time.Duration) error {
	marketDir := filepath.Join(c.basePath, "csv", "market")
	indicatorDir := filepath.Join(c.basePath, "csv", "indicators")

	dirs := []string{marketDir, indicatorDir}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && strings.HasSuffix(info.Name(), ".csv") {
				if time.Since(info.ModTime()) > maxAge {
					if err := os.Remove(path); err != nil {
						return fmt.Errorf("failed to remove old file %s: %v", path, err)
					}
				}
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to clean directory %s: %v", dir, err)
		}
	}

	return nil
}
