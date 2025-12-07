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

// GetAgentHistory 列出 results 目录下所有 Markdown 报告（仅目录信息，不包含内容），支持书签分页
func GetAgentHistory(paramsJson string) (any, error) {
	var params models.HistoryParams
	if strings.TrimSpace(paramsJson) != "" {
		if err := json.Unmarshal([]byte(paramsJson), &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	cfg := config.Get()
	resultsDirAbs, err := filepath.Abs(strings.TrimSpace(cfg.ResultsDir))
	if err != nil || resultsDirAbs == "" {
		return nil, errors.New("results_dir is not configured")
	}

	return listAllHistory(resultsDirAbs, params.Cursor, params.Limit)
}

// listAllHistory 遍历 results 目录，列出所有 markdown 报告，支持简单书签分页
func listAllHistory(resultsDir string, cursor string, limit int) (any, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	resultsDirAbs, err := filepath.Abs(resultsDir)
	if err != nil {
		return nil, fmt.Errorf("resolve results dir: %w", err)
	}

	var items []models.HistoryListItem
	if err := filepath.WalkDir(resultsDirAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			return nil
		}
		items = append(items, models.HistoryListItem{
			Name: d.Name(),
			Path: filepath.ToSlash(path),
		})
		return nil
	}); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("results directory %s not found", resultsDir)
		}
		return nil, fmt.Errorf("walk results dir: %w", err)
	}

	// 固定排序，便于书签定位
	sort.Slice(items, func(i, j int) bool {
		return items[i].Path < items[j].Path
	})

	start := 0
	if cursor != "" {
		for i, f := range items {
			if f.Path == cursor {
				start = i + 1
				break
			}
		}
	}
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	if len(items) < end {
		end = len(items)
	}

	page := items[start:end]
	nextCursor := ""
	if end < len(items) {
		nextCursor = items[end-1].Path
	}

	return map[string]any{
		"items":       page,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	}, nil
}

// GetHistoryInfo 根据相对路径读取 markdown 内容；路径可指向目录（递归读取其中的 md）或单个 md 文件
func GetHistoryInfo(paramsJson string) (any, error) {
	var params models.HistoryInfoParams
	if err := json.Unmarshal([]byte(paramsJson), &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	relPath := strings.TrimSpace(params.Path)
	if relPath == "" {
		return nil, errors.New("path is required")
	}

	absPath, err := filepath.Abs(filepath.Clean(relPath))
	if err != nil || !filepath.IsAbs(absPath) {
		return nil, errors.New("invalid path")
	}

	cfg := config.Get()
	resultsDirAbs, err := filepath.Abs(strings.TrimSpace(cfg.ResultsDir))
	if err != nil || resultsDirAbs == "" {
		return nil, errors.New("results_dir is not configured")
	}

	relToRoot, err := filepath.Rel(resultsDirAbs, absPath)
	if err != nil || strings.HasPrefix(relToRoot, "..") {
		return nil, errors.New("path is outside results_dir")
	}

	target := absPath
	info, err := os.Stat(target)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("path not found: %s", relPath)
		}
		return nil, fmt.Errorf("stat path: %w", err)
	}

	var files []models.HistoryFile
	readFile := func(fullPath string, name string) error {
		content, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			return fmt.Errorf("read file %s: %w", fullPath, readErr)
		}
		if _, relErr := filepath.Rel(resultsDirAbs, fullPath); relErr != nil {
			return relErr
		}
		files = append(files, models.HistoryFile{
			Name:    name,
			Path:    filepath.ToSlash(fullPath),
			Content: string(content),
		})
		return nil
	}

	if info.IsDir() {
		err = filepath.WalkDir(target, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
				return nil
			}
			return readFile(path, d.Name())
		})
		if err != nil {
			return nil, fmt.Errorf("walk dir: %w", err)
		}
	} else {
		if !strings.EqualFold(filepath.Ext(info.Name()), ".md") {
			return nil, errors.New("path is not a markdown file")
		}
		if err := readFile(target, info.Name()); err != nil {
			return nil, err
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return map[string]any{
		"path":  filepath.ToSlash(target),
		"files": files,
	}, nil
}
