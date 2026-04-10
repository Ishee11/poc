package http

import (
	"errors"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
)

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, entity.ErrSessionNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)

	case errors.Is(err, entity.ErrInvalidChips),
		errors.Is(err, entity.ErrInvalidCashOut),
		errors.Is(err, entity.ErrInvalidOperation),
		errors.Is(err, entity.ErrInvalidRequestID):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, entity.ErrDuplicateRequest):
		// идемпотентность → успех
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
