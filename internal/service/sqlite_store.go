package service

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/storage/sqlite"
)

var (
	sqliteStoreOnce sync.Once
	sqliteStoreInst *sqlite.Store
	sqliteStoreErr  error
	// ErrDataDirNotConfigured indicates config.DataDir is empty.
	ErrDataDirNotConfigured = errors.New("data_dir is not configured")
)

// getSQLiteStore returns a shared sqlite store handle for reuse.
func getSQLiteStore() (*sqlite.Store, error) {
	sqliteStoreOnce.Do(func() {
		cfg := config.Get()
		dataDir := strings.TrimSpace(cfg.DataDir)
		if dataDir == "" {
			sqliteStoreErr = ErrDataDirNotConfigured
			return
		}
		dbPath := filepath.Join(dataDir, "agent.db")
		sqliteStoreInst, sqliteStoreErr = sqlite.Open(dbPath)
	})
	return sqliteStoreInst, sqliteStoreErr
}
