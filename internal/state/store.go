package state

import "sync"

// Store keeps tiny-app serialized state in memory.
type Store struct {
	mu    sync.RWMutex
	state map[string][]byte
}

func NewStore() *Store {
	return &Store{
		state: make(map[string][]byte),
	}
}

func (s *Store) Save(appID string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := make([]byte, len(data))
	copy(cp, data)
	s.state[appID] = cp
}

func (s *Store) Load(appID string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.state[appID]
	if !ok {
		return nil, false
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, true
}
