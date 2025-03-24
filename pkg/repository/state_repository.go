package repository

import (
	"fmt"
	"sync"

	"github.com/dskvich/ai-bot/pkg/domain"
)

type stateRepository struct {
	mu    sync.RWMutex
	state map[string]domain.State
}

func NewStateRepository() *stateRepository {
	return &stateRepository{
		state: make(map[string]domain.State),
	}
}

func (s *stateRepository) key(chatID int64, topicID int) string {
	return fmt.Sprintf("%d:%d", chatID, topicID)
}

func (s *stateRepository) Save(chatID int64, topicID int, state domain.State) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.key(chatID, topicID)
	s.state[key] = state
}

func (s *stateRepository) Get(chatID int64, topicID int) (domain.State, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.key(chatID, topicID)
	state, exists := s.state[key]
	return state, exists
}

func (s *stateRepository) Clear(chatID int64, topicID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.key(chatID, topicID)
	delete(s.state, key)
}
