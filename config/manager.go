package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Manager struct {
	path         string
	mu           sync.RWMutex
	cfg          Config
	watcher      *fsnotify.Watcher
	debounce     time.Duration
	onChange     func(Config)
	suppressSelf atomic.Bool
}

type managerOptions struct {
	configPath    string
	initialConfig *Config
	debounce      time.Duration
}

type ManagerOption func(*managerOptions)

var (
	defaultManager *Manager
	managerMu      sync.Mutex
)

func NewManager(opts ...ManagerOption) (*Manager, error) {
	options := managerOptions{
		debounce: 300 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(&options)
	}

	configPath := options.configPath
	if configPath == "" {
		var err error
		configPath, err = defaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	cfg, err := loadOrCreateConfig(configPath, options)
	if err != nil {
		return nil, err
	}

	return &Manager{
		path:     configPath,
		cfg:      cfg,
		debounce: options.debounce,
	}, nil
}

func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) Path() string {
	return m.path
}

func (m *Manager) UpdateFromJSON(jsonStr string) error {
	var cfg Config
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		return fmt.Errorf("parse config json: %w", err)
	}
	return m.Update(cfg)
}

func (m *Manager) Update(newCfg Config) error {
	if err := newCfg.Validate(); err != nil {
		return err
	}

	m.mu.RLock()
	current := m.cfg
	m.mu.RUnlock()
	if reflect.DeepEqual(current, newCfg) {
		return nil
	}

	m.suppressSelf.Store(true)
	defer time.AfterFunc(m.debounce, func() { m.suppressSelf.Store(false) })

	if err := writeConfigFile(m.path, newCfg); err != nil {
		m.suppressSelf.Store(false)
		return err
	}

	m.applyConfig(newCfg)
	return nil
}

func (m *Manager) Watch(ctx context.Context, onChange func(Config)) error {
	m.mu.Lock()
	m.onChange = onChange
	if m.watcher != nil {
		m.mu.Unlock()
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		m.mu.Unlock()
		return err
	}
	m.watcher = watcher
	debounce := m.debounce
	configPath := m.path
	m.mu.Unlock()

	configDir := filepath.Dir(configPath)
	if err := watcher.Add(configDir); err != nil {
		return fmt.Errorf("watch config dir: %w", err)
	}

	go m.watchLoop(ctx, watcher, configPath, debounce)
	return nil
}

func (m *Manager) watchLoop(ctx context.Context, watcher *fsnotify.Watcher, configPath string, debounce time.Duration) {
	defer watcher.Close()

	var timerMu sync.Mutex
	var timer *time.Timer
	trigger := func() {
		timerMu.Lock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(debounce, m.reloadFromDisk)
		timerMu.Unlock()
	}

	for {
		select {
		case evt, ok := <-watcher.Events:
			if !ok {
				return
			}
			if !m.isConfigEvent(evt, configPath) {
				continue
			}
			if m.suppressSelf.Load() {
				continue
			}
			trigger()
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			if err != nil {
				log.Printf("config watcher error: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) isConfigEvent(evt fsnotify.Event, configPath string) bool {
	if filepath.Clean(evt.Name) != filepath.Clean(configPath) {
		return false
	}
	return evt.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0
}

func (m *Manager) reloadFromDisk() {
	var cfg Config
	if err := loadConfigFromFile(m.path, &cfg); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg = *DefaultConfigWithRoot(filepath.Dir(m.path))
			if err := writeConfigFile(m.path, cfg); err != nil {
				log.Printf("config recreate failed: %v", err)
				return
			}
		} else {
			log.Printf("config reload failed: %v", err)
			return
		}
	}
	if err := cfg.Validate(); err != nil {
		log.Printf("config validation failed: %v", err)
		return
	}

	m.mu.RLock()
	current := m.cfg
	m.mu.RUnlock()
	if reflect.DeepEqual(current, cfg) {
		return
	}
	m.applyConfig(cfg)
}

func (m *Manager) applyConfig(cfg Config) {
	m.mu.Lock()
	m.cfg = cfg
	cb := m.onChange
	m.mu.Unlock()

	if cb != nil {
		cb(cfg)
	}
}

func loadOrCreateConfig(path string, options managerOptions) (Config, error) {
	var cfg Config
	if _, err := os.Stat(path); err == nil {
		if err := loadConfigFromFile(path, &cfg); err != nil {
			return Config{}, fmt.Errorf("load config: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return Config{}, err
		}
		return cfg, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("stat config: %w", err)
	}

	switch {
	case options.initialConfig != nil:
		cfg = *options.initialConfig
	default:
		cfg = *DefaultConfigWithRoot(filepath.Dir(path))
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	if err := writeConfigFile(path, cfg); err != nil {
		return Config{}, fmt.Errorf("write initial config: %w", err)
	}

	return cfg, nil
}

func defaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	dir = filepath.Join(dir, "CortexGo")
	return filepath.Join(dir, "config.json"), nil
}

func writeConfigFile(path string, cfg Config) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(path), "cfg-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(&cfg); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return fmt.Errorf("encode config: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return fmt.Errorf("flush config: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return fmt.Errorf("close temp config: %w", err)
	}
	return os.Rename(tmpFile.Name(), path)
}

func WithConfigDir(dir string) ManagerOption {
	return func(o *managerOptions) {
		if dir == "" {
			return
		}
		o.configPath = filepath.Join(dir, "config.json")
	}
}

func WithConfigPath(path string) ManagerOption {
	return func(o *managerOptions) {
		if path != "" {
			o.configPath = path
		}
	}
}

func WithDebounce(d time.Duration) ManagerOption {
	return func(o *managerOptions) {
		if d > 0 {
			o.debounce = d
		}
	}
}

func WithInitialConfig(cfg *Config) ManagerOption {
	return func(o *managerOptions) {
		o.initialConfig = cfg
	}
}

func DefaultManager() *Manager {
	managerMu.Lock()
	defer managerMu.Unlock()
	if defaultManager != nil {
		return defaultManager
	}
	mgr, err := NewManager()
	if err != nil {
		log.Printf("config: failed to create default manager: %v", err)
		return nil
	}
	defaultManager = mgr
	return defaultManager
}

func SetDefaultManager(mgr *Manager) {
	managerMu.Lock()
	defer managerMu.Unlock()
	defaultManager = mgr
}
