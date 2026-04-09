package usecase

import (
	"errors"
	"testing"

	"github.com/ishee11/poc/internal/entity"
)

func TestIdempotent(t *testing.T) {

	t.Run("empty requestID", func(t *testing.T) {
		err := Idempotent(nil, "", func() error {
			return nil
		})

		if !errors.Is(err, entity.ErrInvalidRequestID) {
			t.Fatalf("expected ErrInvalidRequestID, got %v", err)
		}
	})

	t.Run("success execution", func(t *testing.T) {
		called := false

		err := Idempotent(nil, "req-1", func() error {
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

		err := Idempotent(nil, "req-1", func() error {
			called = true
			return entity.ErrDuplicateRequest
		})

		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}

		if !called {
			t.Fatal("fn should be called")
		}
	})

	t.Run("propagates error", func(t *testing.T) {
		expectedErr := errors.New("db error")

		err := Idempotent(nil, "req-1", func() error {
			return expectedErr
		})

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}
