package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/ishee11/poc/internal/usecase"
)

// Register godoc
// @Summary Register
// @Description Creates a regular user and sets an HttpOnly session cookie.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Register request"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	defer r.Body.Close()

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	if err := h.registerUserUC.Execute(r.Context(), usecase.RegisterUserCommand{
		Email:    req.Email,
		Password: req.Password,
	}); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.authUC.Login(r.Context(), usecase.LoginCommand{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: r.UserAgent(),
		IP:        clientIP(r),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	h.setSessionCookie(w, result.Token, result.ExpiresAt)
	slog.InfoContext(
		r.Context(),
		"auth_register_success",
		"request_id", GetRequestID(r.Context()),
		"user_id", result.User.UserID,
		"ip", clientIP(r),
		"user_agent", r.UserAgent(),
	)

	writeJSON(w, http.StatusOK, LoginResponse{
		User:      authUserResponse(result.User),
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
	})
}

// Login godoc
// @Summary Login
// @Description Authenticates a system user and sets an HttpOnly session cookie.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login request"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 429 {object} ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}
	defer r.Body.Close()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, r, http.StatusBadRequest, "bad_request", nil)
		return
	}

	result, err := h.authUC.Login(r.Context(), usecase.LoginCommand{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: r.UserAgent(),
		IP:        clientIP(r),
	})
	if err != nil {
		slog.WarnContext(
			r.Context(),
			"auth_login_failed",
			"request_id", GetRequestID(r.Context()),
			"ip", clientIP(r),
			"user_agent", r.UserAgent(),
		)
		writeError(w, r, err)
		return
	}

	h.setSessionCookie(w, result.Token, result.ExpiresAt)
	slog.InfoContext(
		r.Context(),
		"auth_login_success",
		"request_id", GetRequestID(r.Context()),
		"user_id", result.User.UserID,
		"role", result.User.Role,
		"ip", clientIP(r),
		"user_agent", r.UserAgent(),
	)

	writeJSON(w, http.StatusOK, LoginResponse{
		User:      authUserResponse(result.User),
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
	})
}

// Logout godoc
// @Summary Logout
// @Description Revokes the current auth session and clears the session cookie.
// @Tags auth
// @Produce json
// @Success 204
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}

	token := h.sessionToken(r)
	if err := h.authUC.Logout(r.Context(), token); err != nil {
		writeError(w, r, err)
		return
	}

	h.clearSessionCookie(w)
	slog.InfoContext(
		r.Context(),
		"auth_logout",
		"request_id", GetRequestID(r.Context()),
		"ip", clientIP(r),
		"user_agent", r.UserAgent(),
	)

	w.WriteHeader(http.StatusNoContent)
}

// Me godoc
// @Summary Current user
// @Description Returns the authenticated system user.
// @Tags auth
// @Produce json
// @Success 200 {object} MeResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, r, http.StatusMethodNotAllowed, "method_not_allowed", nil)
		return
	}

	principal, err := h.authUC.CurrentUser(r.Context(), h.sessionToken(r))
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, MeResponse{
		User: authUserResponse(*principal),
	})
}

func (h *AuthHandler) sessionToken(r *http.Request) string {
	cookie, err := r.Cookie(h.cookie.Name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookie.Name,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: h.cookie.SameSite,
		Expires:  expiresAt,
		MaxAge:   int(h.cookie.MaxAge.Seconds()),
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookie.Name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: h.cookie.SameSite,
		MaxAge:   -1,
	})
}

func authUserResponse(principal usecase.AuthPrincipal) AuthUserResponse {
	return AuthUserResponse{
		ID:    principal.UserID,
		Email: principal.Email,
		Role:  principal.Role,
	}
}

func clientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		return forwardedFor
	}
	return r.RemoteAddr
}
