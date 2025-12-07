package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/models"
)

// GetAgentHistory 读取 results 目录下指定标的与交易日的 Markdown 报告
func GetAgentHistory(paramsJson string) (any, error) {
	var params models.HistoryParams
	if strings.TrimSpace(paramsJson) != "" {
		if err := json.Unmarshal([]byte(paramsJson), &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	params.Symbol = strings.TrimSpace(params.Symbol)
	params.TradeDate = strings.TrimSpace(params.TradeDate)

	cfg := config.Get()
	resultsDir := strings.TrimSpace(cfg.ResultsDir)
	if resultsDir == "" {
		return nil, errors.New("results_dir is not configured")
	}

	// 如果没有传 symbol / trade_date，列出全部 Markdown 报告（支持书签分页）
	if params.Symbol == "" && params.TradeDate == "" {
		return listAllHistory(resultsDir, params.Cursor, params.Limit)
	}

	if params.Symbol == "" || params.TradeDate == "" {
		return nil, errors.New("symbol and trade_date are required when fetching a specific run")
	}

	dirPath := filepath.Join(resultsDir, params.Symbol, params.TradeDate)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("results not found for %s on %s", params.Symbol, params.TradeDate)
		}
		return nil, fmt.Errorf("read results dir: %w", err)
	}

	var files []models.HistoryFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}

		fullPath := filepath.Join(dirPath, entry.Name())
		content, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			return nil, fmt.Errorf("read file %s: %w", entry.Name(), readErr)
		}

		files = append(files, models.HistoryFile{
			Name:    entry.Name(),
			Path:    filepath.ToSlash(filepath.Join(params.Symbol, params.TradeDate, entry.Name())),
			Content: string(content),
		})
	}

	// 为了响应稳定性，对文件名排序
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return map[string]any{
		"symbol":     params.Symbol,
		"trade_date": params.TradeDate,
		"files":      files,
	}, nil
}

// listAllHistory 遍历 results 目录，列出所有 markdown 报告，支持简单书签分页
func listAllHistory(resultsDir string, cursor string, limit int) (any, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var files []models.HistoryFile
	if err := filepath.WalkDir(resultsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			return nil
		}
		rel, relErr := filepath.Rel(resultsDir, path)
		if relErr != nil {
			return relErr
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read file %s: %w", path, readErr)
		}
		files = append(files, models.HistoryFile{
			Name:    d.Name(),
			Path:    filepath.ToSlash(rel),
			Content: string(content),
		})
		return nil
	}); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("results directory %s not found", resultsDir)
		}
		return nil, fmt.Errorf("walk results dir: %w", err)
	}

	// 固定排序，便于书签定位
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	start := 0
	if cursor != "" {
		for i, f := range files {
			if f.Path == cursor {
				start = i + 1
				break
			}
		}
	}
	if start > len(files) {
		start = len(files)
	}
	end := start + limit
	if end > len(files) {
		end = len(files)
	}

	page := files[start:end]
	nextCursor := ""
	if end < len(files) {
		nextCursor = files[end-1].Path
	}

	return map[string]any{
		"files":       page,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	}, nil
}
