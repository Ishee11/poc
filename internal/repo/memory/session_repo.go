package memory

import (
	"context"
	"sync"

	"github.com/ishee11/poc/internal/entity"
)

type SessionRepository struct {
	mu       sync.RWMutex
	sessions map[string]*entity.Session
}

func NewSessionRepository() *SessionRepository {
	return &SessionRepository{
		sessions: make(map[string]*entity.Session),
	}
}

func (r *SessionRepository) GetByID(ctx context.Context, id string) (*entity.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.sessions[id]
	if !ok {
		return nil, entity.ErrSessionNotFound
	}

	return s.Copy(), nil
}

func (r *SessionRepository) Save(ctx context.Context, s *entity.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[s.ID()] = s.Copy()
	return nil
}

func (r *SessionRepository) Create(ctx context.Context, s *entity.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[s.ID()]; exists {
		return entity.ErrSessionAlreadyExists
	}

	r.sessions[s.ID()] = s.Copy()
	return nil
}
