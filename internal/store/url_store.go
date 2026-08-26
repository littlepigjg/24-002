package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu         sync.RWMutex
	urls       map[string]model.ShortURL
	cfg        *config.Config
	panicGuard PanicGuardFn
	synced     bool
	dirty      bool
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return &URLStore{
		urls: make(map[string]model.ShortURL),
		cfg:  cfg,
	}, nil
}

func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.cfg.Storage.GetURLFilePath()
	if path == "" {
		s.synced = true
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s.synced = true
			return nil
		}
		return fmt.Errorf("failed to read url store file: %w", err)
	}

	var urls map[string]model.ShortURL
	if err := json.Unmarshal(data, &urls); err != nil {
		return fmt.Errorf("failed to unmarshal url store: %w", err)
	}

	if urls == nil {
		urls = make(map[string]model.ShortURL)
	}
	s.urls = urls
	s.synced = true
	return nil
}

func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cfg.Storage.GetFlushOnWrite() && s.dirty {
		if err := s.flushLocked(); err != nil {
			return err
		}
	}
	s.urls = make(map[string]model.ShortURL)
	return nil
}

func (s *URLStore) flushLocked() error {
	path := s.cfg.Storage.GetURLFilePath()
	if path == "" {
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(s.urls, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal url store: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write url store: %w", err)
	}
	s.dirty = false
	return nil
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if u == nil {
		return fmt.Errorf("short url is nil")
	}

	if s.panicGuard != nil && s.panicGuard(u.Code, u.RawURL) {
		return nil
	}

	if _, exists := s.urls[u.Code]; exists && !overwrite {
		return fmt.Errorf("code already exists: %s", u.Code)
	}
	s.urls[u.Code] = *u
	s.dirty = true

	if s.cfg.Storage.GetFlushOnWrite() {
		return s.flushLocked()
	}
	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.urls[code]
	if !ok {
		return nil, fmt.Errorf("url not found: %s", code)
	}
	return &u, nil
}

func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]model.ShortURL, len(s.urls))
	for k, v := range s.urls {
		result[k] = v
	}
	return result
}

func (s *URLStore) SyncInterval() time.Duration {
	return s.cfg.Storage.GetSyncInterval()
}

type AccessLogStore struct {
	mu     sync.Mutex
	path   string
	opened bool
	cfg    *config.Config
	lines  []string
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return &AccessLogStore{
		cfg: cfg,
	}, nil
}

func (a *AccessLogStore) Open(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.opened {
		return nil
	}

	path := a.cfg.Storage.GetLogFilePath()
	if path == "" {
		a.opened = true
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	a.path = path
	a.opened = true
	return nil
}

func (a *AccessLogStore) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.opened {
		return nil
	}

	if a.path != "" && len(a.lines) > 0 {
		f, err := os.OpenFile(a.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open access log: %w", err)
		}
		defer f.Close()

		for _, line := range a.lines {
			if _, err := f.WriteString(line + "\n"); err != nil {
				return fmt.Errorf("failed to write access log: %w", err)
			}
		}
	}

	a.opened = false
	return nil
}

func (a *AccessLogStore) WriteLog(code, rawURL string, status int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	timestamp := time.Now().Format(time.RFC3339)
	line := fmt.Sprintf("%s %s %d %s", timestamp, code, status, rawURL)
	a.lines = append(a.lines, line)
}
