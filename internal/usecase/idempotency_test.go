package usecase

import (
	"errors"
	"testing"

	"github.com/ishee11/poc/internal/entity"
)

func TestIdempotent(t *testing.T) {

	t.Run("empty requestID", func(t *testing.T) {
		err := Idempotent(nil, nil, "", func() error {
			return nil
		})

		if !errors.Is(err, entity.ErrInvalidRequestID) {
			t.Fatalf("expected ErrInvalidRequestID, got %v", err)
		}
	})

	t.Run("already processed", func(t *testing.T) {
		repo := &operationRepoMock{
			getByRequestIDFn: func(tx Tx, requestID string) (*entity.Operation, error) {
				return &entity.Operation{}, nil
			},
		}

		called := false

		err := Idempotent(nil, repo, "req-1", func() error {
			called = true
			return nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if called {
			t.Fatal("fn should not be called for idempotent request")
		}
	})

	t.Run("first execution", func(t *testing.T) {
		repo := &operationRepoMock{
			getByRequestIDFn: func(tx Tx, requestID string) (*entity.Operation, error) {
				return nil, nil
			},
		}

		called := false

		err := Idempotent(nil, repo, "req-1", func() error {
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

	t.Run("repo error", func(t *testing.T) {
		repo := &operationRepoMock{
			getByRequestIDFn: func(tx Tx, requestID string) (*entity.Operation, error) {
				return nil, errors.New("db error")
			},
		}

		err := Idempotent(nil, repo, "req-1", func() error {
			return nil
		})

		if err == nil {
			t.Fatal("expected error")
		}
	})
}
