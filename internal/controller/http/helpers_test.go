package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ishee11/poc/internal/entity"
)

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "not found", err: entity.ErrSessionNotFound, wantStatus: http.StatusNotFound, wantCode: "session_not_found"},
		{name: "invalid chips", err: entity.ErrInvalidChips, wantStatus: http.StatusBadRequest, wantCode: "invalid_chips"},
		{name: "duplicate request", err: entity.ErrDuplicateRequest, wantStatus: http.StatusOK, wantCode: ""},
		{name: "unknown", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeError(rec, tc.err)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
			if tc.wantCode == "" {
				if rec.Body.Len() != 0 {
					t.Fatalf("expected empty body, got %q", rec.Body.String())
				}
				return
			}

			var body ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Error != tc.wantCode {
				t.Fatalf("expected code %s, got %s", tc.wantCode, body.Error)
			}
		})
	}
}

func TestWriteError_SessionNotBalancedDetails(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, &entity.SessionNotBalancedError{RemainingChips: 150})

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Code)
	}

	var body struct {
		Error   string `json:"error"`
		Details struct {
			RemainingChips int64 `json:"remaining_chips"`
		} `json:"details"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != "session_not_balanced" || body.Details.RemainingChips != 150 {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestParseDateRange(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/stats/sessions?from=2026-04-01&to=2026-04-02", nil)
	from, to, err := parseDateRange(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from == nil || from.Value != "2026-04-01T00:00:00Z" {
		t.Fatalf("unexpected from: %+v", from)
	}
	if to == nil || to.Value != "2026-04-03T00:00:00Z" {
		t.Fatalf("unexpected to: %+v", to)
	}

	badReq := httptest.NewRequest(http.MethodGet, "/stats/sessions?from=bad-date", nil)
	if _, _, err := parseDateRange(badReq); err == nil {
		t.Fatal("expected error for bad date")
	}
}
