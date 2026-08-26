package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"logalert/internal/config"
	"logalert/internal/model"
)

// PanicGuardFn is a function type for guarding against panics during operations.
type PanicGuardFn func(code, rawURL string) bool

// URLStore stores and manages shortened URLs.
type URLStore struct {
	mu          sync.RWMutex
	config      *config.Config
	urls        map[string]model.ShortURL
	panicGuard  PanicGuardFn
	dataLoaded  bool
}

// NewURLStore creates a new URLStore.
func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return &URLStore{
		config: cfg,
		urls:   make(map[string]model.ShortURL),
	}, nil
}

// Load loads URL data from file.
func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dataDir := s.config.Storage.DataDir
	urlFile := dataDir + "/urls.json"

	data, err := os.ReadFile(urlFile)
	if err != nil {
		if os.IsNotExist(err) {
			s.dataLoaded = true
			return nil
		}
		return fmt.Errorf("failed to read URL file: %w", err)
	}

	var urls map[string]model.ShortURL
	if err := json.Unmarshal(data, &urls); err != nil {
		return fmt.Errorf("failed to parse URL file: %w", err)
	}

	s.urls = urls
	s.dataLoaded = true
	return nil
}

// Close releases resources.
func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dataDir := s.config.Storage.DataDir
	os.MkdirAll(dataDir, 0755)

	data, err := json.MarshalIndent(s.urls, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize URLs: %w", err)
	}

	urlFile := dataDir + "/urls.json"
	if err := os.WriteFile(urlFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write URL file: %w", err)
	}

	return nil
}

// SetPanicGuard sets a guard function that can prevent certain operations from panicking.
func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

// Save saves a ShortURL to the store.
func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return fmt.Errorf("short URL is nil")
	}
	if err := u.Validate(); err != nil {
		return fmt.Errorf("invalid short URL: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			return fmt.Errorf("code already exists: %s", u.Code)
		}
	}

	if s.panicGuard != nil && s.panicGuard(u.Code, u.RawURL) {
		panic("panicking on guard for code: " + u.Code)
	}

	s.urls[u.Code] = *u
	return nil
}

// Get retrieves a ShortURL by code.
func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.urls[code]
	if !ok {
		return nil, fmt.Errorf("code not found: %s", code)
	}
	result := u
	return &result, nil
}

// RawSnapshot returns a raw snapshot of all stored URLs.
func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]model.ShortURL, len(s.urls))
	for k, v := range s.urls {
		result[k] = v
	}
	return result
}

// AccessLogEntry represents a single access log entry.
type AccessLogEntry struct {
	Code      string `json:"code"`
	RawURL    string `json:"raw_url"`
	Timestamp string `json:"timestamp"`
	UserAgent string `json:"user_agent"`
}

// AccessLogStore stores and manages access logs.
type AccessLogStore struct {
	mu      sync.Mutex
	config  *config.Config
	logs    []AccessLogEntry
	opened  bool
}

// NewAccessLogStore creates a new AccessLogStore.
func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return &AccessLogStore{
		config: cfg,
		logs:   make([]AccessLogEntry, 0),
	}, nil
}

// Open opens the access log store.
func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.opened {
		return nil
	}

	dataDir := s.config.Storage.DataDir
	logFile := dataDir + "/access_logs.json"

	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			s.opened = true
			return nil
		}
		return fmt.Errorf("failed to read access log file: %w", err)
	}

	var logs []AccessLogEntry
	if err := json.Unmarshal(data, &logs); err != nil {
		return fmt.Errorf("failed to parse access log file: %w", err)
	}

	s.logs = logs
	s.opened = true
	return nil
}

// Close closes the access log store.
func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.opened {
		return nil
	}

	dataDir := s.config.Storage.DataDir
	os.MkdirAll(dataDir, 0755)

	logFile := dataDir + "/access_logs.json"

	data, err := json.MarshalIndent(s.logs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize access logs: %w", err)
	}

	if err := os.WriteFile(logFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write access log file: %w", err)
	}

	s.opened = false
	return nil
}

// AppendLog appends a log entry to the access log.
func (s *AccessLogStore) AppendLog(entry AccessLogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, entry)
}
