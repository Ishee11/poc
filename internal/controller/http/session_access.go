package http

import (
	"errors"
	"net/http"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type sessionAccessAuthorizer struct {
	service *usecase.SessionAccessService
	authUC  *usecase.AuthService
	cookie  AuthCookieConfig
}

func (a *sessionAccessAuthorizer) viewer(r *http.Request) (*entity.AuthUserID, bool, error) {
	if !a.cookie.Enabled {
		return nil, true, nil
	}

	cookie, err := r.Cookie(a.cookie.Name)
	if err != nil || cookie.Value == "" {
		return nil, false, nil
	}

	principal, err := a.authUC.CurrentUser(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, entity.ErrUnauthorized) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &principal.UserID, principal.Role == entity.AuthRoleAdmin, nil
}

func (a *sessionAccessAuthorizer) principal(r *http.Request) (*usecase.AuthPrincipal, error) {
	if !a.cookie.Enabled {
		return nil, nil
	}
	cookie, err := r.Cookie(a.cookie.Name)
	if err != nil || cookie.Value == "" {
		return nil, entity.ErrUnauthorized
	}
	return a.authUC.CurrentUser(r.Context(), cookie.Value)
}

func (a *sessionAccessAuthorizer) requireAuthenticated(w http.ResponseWriter, r *http.Request) (*usecase.AuthPrincipal, bool) {
	principal, err := a.principal(r)
	if err != nil {
		writeError(w, r, err)
		return nil, false
	}
	return principal, true
}

func (a *sessionAccessAuthorizer) requireView(w http.ResponseWriter, r *http.Request, sessionID entity.SessionID) bool {
	viewerUserID, viewerIsAdmin, err := a.viewer(r)
	if err != nil {
		writeError(w, r, err)
		return false
	}
	err = a.service.RequireView(r.Context(), usecase.SessionAccessQuery{
		SessionID:     sessionID,
		ViewerUserID:  viewerUserID,
		ViewerIsAdmin: viewerIsAdmin,
		GuestPlayerID: entity.PlayerID(r.URL.Query().Get("guest_player_id")),
	})
	if err != nil {
		writeError(w, r, err)
		return false
	}
	return true
}

func (a *sessionAccessAuthorizer) requireMutation(w http.ResponseWriter, r *http.Request, sessionID entity.SessionID) bool {
	if !a.cookie.Enabled {
		return true
	}
	principal, ok := a.requireAuthenticated(w, r)
	if !ok {
		return false
	}
	viewerUserID := principal.UserID
	err := a.service.RequireMutation(r.Context(), usecase.SessionAccessQuery{
		SessionID:     sessionID,
		ViewerUserID:  &viewerUserID,
		ViewerIsAdmin: principal.Role == entity.AuthRoleAdmin,
	})
	if err != nil {
		writeError(w, r, err)
		return false
	}
	return true
}

func (a *sessionAccessAuthorizer) canMutate(r *http.Request, sessionID entity.SessionID) (bool, error) {
	if !a.cookie.Enabled {
		return true, nil
	}
	principal, err := a.principal(r)
	if err != nil {
		if errors.Is(err, entity.ErrUnauthorized) {
			return false, nil
		}
		return false, err
	}
	viewerUserID := principal.UserID
	return a.service.CanMutate(r.Context(), usecase.SessionAccessQuery{
		SessionID:     sessionID,
		ViewerUserID:  &viewerUserID,
		ViewerIsAdmin: principal.Role == entity.AuthRoleAdmin,
	})
}
