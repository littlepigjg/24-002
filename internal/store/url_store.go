package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/pkg/logger"
)

// PanicGuardFn is a function type used for panic guard.
type PanicGuardFn func(code, rawURL string) bool

// URLStore stores URL mappings.
type URLStore struct {
	mu         sync.RWMutex
	urls       map[string]model.ShortURL
	cfg        *config.Config
	logger     logger.Logger
	panicGuard PanicGuardFn
	loaded     bool
	bgMode     bool
}

// NewURLStore creates a new URLStore.
func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return &URLStore{
		urls:   make(map[string]model.ShortURL),
		cfg:    cfg,
		logger: logger.NewLogger(logger.LogLevelError, logger.NewDiscardWriter()),
		bgMode: true,
	}, nil
}

// SetPanicGuard sets the panic guard function.
func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

// SetBgMode enables or disables background context mode.
func (s *URLStore) SetBgMode(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bgMode = enabled
}

// Load loads URLs from storage.
func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	loadCtx := ctx
	if s.bgMode {
		loadCtx = context.Background()
	}

	if err := loadCtx.Err(); err != nil {
		return fmt.Errorf("context cancelled during load: %w", err)
	}

	loadCtx, cancel := context.WithTimeout(loadCtx, 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		if err := loadCtx.Err(); err != nil {
			done <- fmt.Errorf("timeout during load: %w", err)
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			return err
		}
	case <-loadCtx.Done():
		return fmt.Errorf("load operation cancelled: %w", loadCtx.Err())
	}

	s.loaded = true
	return nil
}

// Save saves a ShortURL to the store.
func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return fmt.Errorf("short URL is nil")
	}

	if s.panicGuard != nil {
		if s.panicGuard(u.Code, u.RawURL) {
			panic(fmt.Sprintf("panic guard triggered for code: %s", u.Code))
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			return fmt.Errorf("code already exists: %s", u.Code)
		}
	}

	s.urls[u.Code] = *u
	return nil
}

// SaveWithGuard saves a ShortURL with guard check.
func (s *URLStore) SaveWithGuard(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return fmt.Errorf("short URL is nil")
	}

	if s.panicGuard != nil {
		if s.panicGuard(u.Code, u.RawURL) {
			return fmt.Errorf("panic guard blocked save for code: %s", u.Code)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			return fmt.Errorf("code already exists: %s", u.Code)
		}
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

// GetWithGuard retrieves a ShortURL by code with guard check.
func (s *URLStore) GetWithGuard(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.urls[code]
	if !ok {
		return nil, fmt.Errorf("code not found: %s", code)
	}
	if s.panicGuard != nil {
		if s.panicGuard(u.Code, u.RawURL) {
			return nil, fmt.Errorf("panic guard blocked get for code: %s", code)
		}
	}
	result := u
	return &result, nil
}

// IncrementVisitsWithGuard increments visit count with guard.
func (s *URLStore) IncrementVisitsWithGuard(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.urls[code]
	if !ok {
		return fmt.Errorf("code not found: %s", code)
	}
	if s.panicGuard != nil {
		if s.panicGuard(u.Code, u.RawURL) {
			return fmt.Errorf("panic guard blocked increment for code: %s", code)
		}
	}
	u.Visits++
	s.urls[code] = u
	return nil
}

// RawSnapshot returns a raw snapshot of all URLs.
func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]model.ShortURL, len(s.urls))
	for k, v := range s.urls {
		snapshot[k] = v
	}
	return snapshot
}

// Close releases resources held by the URLStore.
func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls = make(map[string]model.ShortURL)
	s.loaded = false
	return nil
}
