package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/pkg/cache"
	"logalert/pkg/logger"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu          sync.RWMutex
	urls        map[string]model.ShortURL
	cfg         *config.Config
	persistence *FilePersistence
	cache       *cache.Cache
	panicGuard  PanicGuardFn
	logger      logger.Logger
	closed      bool
	cacheSize   int
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	return NewURLStoreWithCacheSize(cfg, 1000)
}

func NewURLStoreWithCacheSize(cfg *config.Config, cacheSize int) (*URLStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	log := logger.Default()
	persistence, err := NewFilePersistence(cfg.Storage.DataDir, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create persistence: %w", err)
	}

	return &URLStore{
		urls:        make(map[string]model.ShortURL),
		cfg:         cfg,
		persistence: persistence,
		cache:       cache.New(cacheSize),
		logger:      log.WithField("component", "url_store"),
		cacheSize:   cacheSize,
	}, nil
}

func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.cfg.Storage.GetURLFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s.logger.Info("url file not found, starting with empty store")
			return nil
		}
		return fmt.Errorf("failed to read url file: %w", err)
	}

	var urls []model.ShortURL
	if err := json.Unmarshal(data, &urls); err != nil {
		return fmt.Errorf("failed to unmarshal urls: %w", err)
	}

	for _, u := range urls {
		s.urls[u.Code] = u
	}

	s.logger.Info("urls loaded from file", "count", len(urls))
	return nil
}

func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	err := s.persistence.SaveLogs(context.Background(), nil)
	if err != nil {
		s.logger.Error("failed to persist urls on close", "error", err)
	}

	s.logger.Info("url store closed")
	return nil
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return fmt.Errorf("short url is nil")
	}
	if err := u.Validate(); err != nil {
		return fmt.Errorf("invalid short url: %w", err)
	}

	s.mu.Lock()

	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			s.mu.Unlock()
			return fmt.Errorf("url code already exists: %s", u.Code)
		}
	}

	s.urls[u.Code] = *u
	s.cache.Set(u.Code, *u, 0)

	entries := make([]*model.LogEntry, 0)
	for _, url := range s.urls {
		entry := model.NewLogEntry(url.RawURL, model.LevelInfo, fmt.Sprintf("url:%s", url.Code))
		entries = append(entries, entry)
	}

	s.persistence.SaveLogs(context.Background(), entries)

	s.mu.Unlock()

	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	url, ok := s.urls[code]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("url not found: %s", code)
	}

	return &url, nil
}

func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]model.ShortURL, len(s.urls))
	for k, v := range s.urls {
		snapshot[k] = v
	}
	return snapshot
}

func (s *URLStore) getCodes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	codes := make([]string, 0, len(s.urls))
	for k := range s.urls {
		codes = append(codes, k)
	}
	return codes
}

var _ = filepath.Join
