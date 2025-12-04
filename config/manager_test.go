package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagerCreatesAndUpdates(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(WithConfigDir(dir))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	path := filepath.Join(dir, "config.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	cfg := mgr.Get()
	cfg.ProjectDir = filepath.Join(dir, "project")
	cfg.ResultsDir = filepath.Join(dir, "results")
	cfg.DataDir = filepath.Join(dir, "data")
	cfg.DataCacheDir = filepath.Join(dir, "cache")

	data, _ := json.Marshal(cfg)
	if err := mgr.UpdateFromJSON(string(data)); err != nil {
		t.Fatalf("UpdateFromJSON: %v", err)
	}

	updated := mgr.Get()
	if updated.ProjectDir != cfg.ProjectDir {
		t.Fatalf("expected project dir %s, got %s", cfg.ProjectDir, updated.ProjectDir)
	}
}

func TestManagerWatchReloads(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(WithConfigDir(dir))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reloaded := make(chan struct{}, 1)
	if err := mgr.Watch(ctx, func(cfg Config) {
		reloaded <- struct{}{}
	}); err != nil {
		t.Fatalf("Watch: %v", err)
	}

	cfg := mgr.Get()
	cfg.ProjectDir = filepath.Join(dir, "changed")
	cfg.ResultsDir = filepath.Join(dir, "results")
	cfg.DataDir = filepath.Join(dir, "data")
	cfg.DataCacheDir = filepath.Join(dir, "cache")

	if err := writeConfigFile(mgr.Path(), cfg); err != nil {
		t.Fatalf("writeConfigFile: %v", err)
	}

	select {
	case <-reloaded:
	case <-time.After(2 * time.Second):
		t.Fatalf("watcher did not fire on config change")
	}
}
