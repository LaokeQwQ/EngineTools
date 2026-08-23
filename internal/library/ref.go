// Package library serializes concurrent access to one Engine DJ library.
// Readers run concurrently; writers exclude everyone. Release closures are
// idempotent so deferred cleanup can call them twice without corrupting
// state.
package library

import "sync"

// Manager tracks one read/write lock per library identity.
type Manager struct {
	mu    sync.Mutex
	locks map[string]*sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{locks: make(map[string]*sync.RWMutex)}
}

func (m *Manager) lockFor(id string) *sync.RWMutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.locks[id]
	if !ok {
		l = &sync.RWMutex{}
		m.locks[id] = l
	}
	return l
}

// AcquireRead takes a shared lock on the library and returns an idempotent
// release closure.
func (m *Manager) AcquireRead(id string) func() {
	l := m.lockFor(id)
	l.RLock()
	var once sync.Once
	return func() { once.Do(l.RUnlock) }
}

// AcquireWrite takes the exclusive lock on the library and returns an
// idempotent release closure.
func (m *Manager) AcquireWrite(id string) func() {
	l := m.lockFor(id)
	l.Lock()
	var once sync.Once
	return func() { once.Do(l.Unlock) }
}
