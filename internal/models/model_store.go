package model

import "sync"

type Store interface {
	Save(model Model)
	Get(id string) (Model, bool)
}

type InMemoryStore struct {
	data map[string]Model
	mu   sync.RWMutex
}

// FindByID implements Repository.
func (s *InMemoryStore) FindByID(id string) (Model, bool) {
	m, ok := s.data[id]
	if !ok {
		return Model{}, false
	}
	return m, true
}

// GetByOwner implements Repository.
func (s *InMemoryStore) GetByOwner(ownerID string) []Model {
	var result []Model

	for _, m := range s.data {
		if m.OwnerID == ownerID {
			result = append(result, m)
		}
	}

	return result
}

// UpdateStatus implements Repository.
func (s *InMemoryStore) UpdateStatus(id string, status Status) {
	m, ok := s.data[id]
	if !ok {
		return
	}
	m.Status = status
	s.data[id] = m
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		data: make(map[string]Model),
	}
}

func (s *InMemoryStore) Save(m Model) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[m.ID] = m
}

func (s *InMemoryStore) Get(id string) (Model, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.data[id]
	return m, ok
}
