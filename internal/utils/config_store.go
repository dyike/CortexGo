package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ConfigStore keeps track of a configuration file path and provides
// helpers for reading/writing JSON blobs safely.
type ConfigStore struct {
	mu              sync.RWMutex
	path            string
	defaultFilename string
}

// NewConfigStore creates a store with a default filename used when no
// explicit path is provided.
func NewConfigStore(defaultFilename string) *ConfigStore {
	return &ConfigStore{defaultFilename: defaultFilename}
}

// Path returns the current absolute config file path or empty if unset.
func (s *ConfigStore) Path() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

// SetPath normalises and stores the provided path. An empty path clears
// the stored value. The resolved absolute path is returned.
func (s *ConfigStore) SetPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		s.mu.Lock()
		s.path = ""
		s.mu.Unlock()
		return "", nil
	}

	absPath := path
	if !filepath.IsAbs(absPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("determine working directory: %w", err)
		}
		absPath = filepath.Join(cwd, absPath)
	}

	absPath = filepath.Clean(absPath)

	s.mu.Lock()
	s.path = absPath
	s.mu.Unlock()
	return absPath, nil
}

// DetectDefault checks the current working directory for an existing
// config file using the default filename.
func (s *ConfigStore) DetectDefault() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	path := filepath.Join(cwd, s.defaultFilename)
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	return "", false
}

// Resolve returns the stored path if available, otherwise constructs one
// from the provided base directory (or current working directory when
// baseDir is empty) using the default filename. The resolved path is also
// stored internally.
func (s *ConfigStore) Resolve(baseDir string) (string, error) {
	if existing := s.Path(); existing != "" {
		return existing, nil
	}

	root := strings.TrimSpace(baseDir)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		root = cwd
	}

	resolved := filepath.Join(root, s.defaultFilename)
	if _, err := s.SetPath(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

// Read loads the file content, creating parent directories if needed.
func (s *ConfigStore) Read(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("config path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}
	return os.ReadFile(path)
}

// Write persists the given bytes atomically to the target path.
func (s *ConfigStore) Write(path string, data []byte) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("config path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
