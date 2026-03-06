package store

import (
	"context"
	"sync"

	"github.com/dcm-project/3-tier-demo-service-provider/api/v1alpha1"
)

// StackStore holds 3-tier stacks in memory.
type StackStore interface {
	Create(ctx context.Context, stack v1alpha1.Stack) (v1alpha1.Stack, error)
	Get(ctx context.Context, id string) (v1alpha1.Stack, bool)
	List(ctx context.Context, maxPageSize, offset int) ([]v1alpha1.Stack, bool)
	Delete(ctx context.Context, id string) (bool, error)
}

type memoryStore struct {
	mu     sync.RWMutex
	stacks map[string]v1alpha1.Stack
}

// NewMemoryStore returns an in-memory stack store.
func NewMemoryStore() StackStore {
	return &memoryStore{stacks: make(map[string]v1alpha1.Stack)}
}

func (s *memoryStore) Create(ctx context.Context, stack v1alpha1.Stack) (v1alpha1.Stack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stacks[*stack.Id]; ok {
		return stack, ErrStackExists
	}
	s.stacks[*stack.Id] = stack
	return stack, nil
}

func (s *memoryStore) Get(ctx context.Context, id string) (v1alpha1.Stack, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stack, ok := s.stacks[id]
	return stack, ok
}

func (s *memoryStore) List(ctx context.Context, maxPageSize, offset int) ([]v1alpha1.Stack, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []v1alpha1.Stack
	for _, s := range s.stacks {
		list = append(list, s)
	}
	if offset >= len(list) {
		return nil, false
	}
	end := offset + maxPageSize
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end], end < len(list)
}

func (s *memoryStore) Delete(ctx context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stacks[id]; !ok {
		return false, nil
	}
	delete(s.stacks, id)
	return true, nil
}
