package streams

import (
	"sync"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/exp/metrics/identity"
)

func SyncMap[T any](items Map[T]) Map[T] {
	return &MutexMap[T]{items: items}
}

type MutexMap[T any] struct {
	items Map[T]
	mu    sync.RWMutex
}

func (m *MutexMap[T]) Load(id identity.Stream) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.items.Load(id)
}

func (m *MutexMap[T]) Store(id identity.Stream, v T) {
	m.mu.Lock()
	m.Store(id, v)
	m.mu.Unlock()
}

func (m *MutexMap[T]) Delete(id identity.Stream) {
	m.mu.Lock()
	m.Delete(id)
	m.mu.Unlock()
}

func (m *MutexMap[T]) Items() func(yield func(identity.Stream, T) bool) bool {
	return func(yield func(identity.Stream, T) bool) bool {
		m.mu.RLock()
		ok := m.items.Items()(yield)
		m.mu.RUnlock()
		return ok
	}
}
