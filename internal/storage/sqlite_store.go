package storage

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dyike/CortexGo/config"
)

var (
	sqliteStoreOnce sync.Once
	sqliteStoreInst *Store
	sqliteStoreErr  error
	// ErrDataDirNotConfigured indicates config.DataDir is empty.
	ErrDataDirNotConfigured = errors.New("data_dir is not configured")
)

// GetSQLiteStore returns a shared sqlite store handle for reuse.
func GetSQLiteStore() (*Store, error) {
	sqliteStoreOnce.Do(func() {
		cfg := config.Get()
		dataDir := strings.TrimSpace(cfg.DataDir)
		if dataDir == "" {
			sqliteStoreErr = ErrDataDirNotConfigured
			return
		}
		dbPath := filepath.Join(dataDir, "agent.db")
		sqliteStoreInst, sqliteStoreErr = Open(dbPath)
	})
	return sqliteStoreInst, sqliteStoreErr
}
