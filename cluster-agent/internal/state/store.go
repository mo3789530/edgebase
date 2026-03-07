package state

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type Store interface {
	LastSyncID() uuid.UUID
	SetLastSyncID(id uuid.UUID)
	LastGeneration() int64
	SetLastGeneration(generation int64)
	LastInventoryAt() time.Time
	SetLastInventoryAt(ts time.Time)
}

type MemoryStore struct {
	mu              sync.RWMutex
	lastSyncID      uuid.UUID
	lastGeneration  int64
	lastInventoryAt time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) LastSyncID() uuid.UUID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSyncID
}

func (s *MemoryStore) SetLastSyncID(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSyncID = id
}

func (s *MemoryStore) LastGeneration() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastGeneration
}

func (s *MemoryStore) SetLastGeneration(generation int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastGeneration = generation
}

func (s *MemoryStore) LastInventoryAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastInventoryAt
}

func (s *MemoryStore) SetLastInventoryAt(ts time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastInventoryAt = ts
}
