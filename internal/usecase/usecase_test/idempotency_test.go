package usecase_test

import (
	"errors"
	"testing"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func TestIdempotent(t *testing.T) {

	t.Run("empty requestID", func(t *testing.T) {
		repo := &idempotencyRepoMock{}

		err := usecase.Idempotent(nil, repo, "", func() error {
			return nil
		})

		if !errors.Is(err, entity.ErrInvalidRequestID) {
			t.Fatalf("expected ErrInvalidRequestID, got %v", err)
		}
	})

	t.Run("success execution", func(t *testing.T) {
		called := false

		repo := &idempotencyRepoMock{
			saveFn: func(tx usecase.Tx, requestID string) error {
				return nil
			},
		}

		err := usecase.Idempotent(nil, repo, "req-1", func() error {
			called = true
			return nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !called {
			t.Fatal("fn should be called")
		}
	})

	t.Run("duplicate request treated as success", func(t *testing.T) {
		called := false

		repo := &idempotencyRepoMock{
			saveFn: func(tx usecase.Tx, requestID string) error {
				return entity.ErrDuplicateRequest
			},
		}

		err := usecase.Idempotent(nil, repo, "req-1", func() error {
			called = true
			return nil
		})

		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}

		// 🔥 важно: fn НЕ должен вызываться
		if called {
			t.Fatal("fn should NOT be called on duplicate")
		}
	})

	t.Run("propagates error from repo", func(t *testing.T) {
		expectedErr := errors.New("db error")

		repo := &idempotencyRepoMock{
			saveFn: func(tx usecase.Tx, requestID string) error {
				return expectedErr
			},
		}

		err := usecase.Idempotent(nil, repo, "req-1", func() error {
			return nil
		})

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("propagates error from fn", func(t *testing.T) {
		expectedErr := errors.New("business error")

		repo := &idempotencyRepoMock{
			saveFn: func(tx usecase.Tx, requestID string) error {
				return nil
			},
		}

		err := usecase.Idempotent(nil, repo, "req-1", func() error {
			return expectedErr
		})

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}
