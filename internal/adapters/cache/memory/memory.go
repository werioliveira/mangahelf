package memory

import (
	"context"
	"sync"
	"time"

	"github.com/mangashelf/mangashelf/internal/ports"
)

type Memory struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
}

type cacheEntry struct {
	value []byte
	exp   time.Time
}

func New() *Memory { return &Memory{data: make(map[string]cacheEntry)} }

func (m *Memory) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.data[key]
	if !ok || !e.exp.IsZero() && time.Now().After(e.exp) {
		return nil, nil
	}
	return e.value, nil
}

func (m *Memory) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	m.data[key] = cacheEntry{value: value, exp: exp}
	return nil
}

func (m *Memory) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

var _ ports.Cache = (*Memory)(nil)
