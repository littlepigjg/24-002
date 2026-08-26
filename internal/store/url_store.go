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

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu              sync.RWMutex
	entries         map[string]*model.ShortURL
	consumedEntries map[string]*model.ShortURL
	pendingTasks    []*urlAnalysisTask
	guards          []PanicGuardFn
	logger          logger.Logger
	config          *config.Config
}

type urlAnalysisTask struct {
	code      string
	rawURL    string
	timestamp time.Time
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	return &URLStore{
		entries:         make(map[string]*model.ShortURL),
		consumedEntries: make(map[string]*model.ShortURL),
		pendingTasks:    make([]*urlAnalysisTask, 0),
		logger:          logger.Default().WithField("store", "url"),
		config:          cfg,
	}, nil
}

func (s *URLStore) Load(ctx context.Context) error {
	return nil
}

func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = make(map[string]*model.ShortURL)
	s.consumedEntries = make(map[string]*model.ShortURL)
	s.pendingTasks = make([]*urlAnalysisTask, 0)
	return nil
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fn != nil {
		s.guards = append(s.guards, fn)
	}
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return fmt.Errorf("short URL is nil")
	}
	if err := u.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.entries[u.Code]; exists && !overwrite {
		return fmt.Errorf("code already exists: %s", u.Code)
	}

	for _, guard := range s.guards {
		if guard(u.Code, u.RawURL) {
			return fmt.Errorf("panic guard triggered for code: %s", u.Code)
		}
	}

	s.entries[u.Code] = u

	task := &urlAnalysisTask{
		code:      u.Code,
		rawURL:    u.RawURL,
		timestamp: time.Now(),
	}
	s.pendingTasks = append(s.pendingTasks, task)

	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[code]
	if !ok {
		return nil, fmt.Errorf("short URL not found: %s", code)
	}
	return entry, nil
}

func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]model.ShortURL, len(s.entries))
	for k, v := range s.entries {
		result[k] = *v
	}
	return result
}

func (s *URLStore) IncrVisits(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[code]
	if !ok {
		return fmt.Errorf("short URL not found: %s", code)
	}
	entry.Visits++
	return nil
}

func (s *URLStore) ConsumeEntry(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.entries[code]; ok {
		s.consumedEntries[code] = entry
		delete(s.entries, code)
	}
}

func (s *URLStore) PendingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pendingTasks)
}

func (s *URLStore) ConsumedCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.consumedEntries)
}
