package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	sessionuc "github.com/ishee11/poc/internal/usecase/session"
)

type SessionUseCase interface {
	BuyIn(ctx context.Context, cmd sessionuc.BuyInCommand) error
	CashOut(ctx context.Context, cmd sessionuc.CashOutCommand) error
	StartSession(ctx context.Context, cmd sessionuc.StartSessionCommand) error
	CloseSession(ctx context.Context, cmd sessionuc.CloseSessionCommand) error
	GetResult(ctx context.Context, q sessionuc.GetResultQuery) (valueobject.Money, error)
	CreateSession(ctx context.Context, cmd sessionuc.CreateSessionCommand) error
	GetSession(ctx context.Context, q sessionuc.GetSessionQuery) (*entity.Session, error)
}

type SessionHandler struct {
	uc SessionUseCase
}

func NewSessionHandler(uc SessionUseCase) *SessionHandler {
	return &SessionHandler{uc: uc}
}

type createSessionRequest struct {
	SessionID string `json:"session_id"`
	Rate      int64  `json:"rate"`
}

type buyInRequest struct {
	OperationID string `json:"operation_id"`
	PlayerID    string `json:"player_id"`
	Money       int64  `json:"money"`
}

type cashOutRequest struct {
	OperationID string `json:"operation_id"`
	PlayerID    string `json:"player_id"`
	Chips       int64  `json:"chips"`
}

func (h *SessionHandler) CreateSession(c *gin.Context) {
	var req createSessionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	cmd := sessionuc.CreateSessionCommand{
		SessionID: req.SessionID,
		Rate:      req.Rate,
	}

	if err := h.uc.CreateSession(c.Request.Context(), cmd); err != nil {
		handleError(c, err)
		return
	}

	respondOK(c)
}

func (h *SessionHandler) StartSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty session id"})
		return
	}

	cmd := sessionuc.StartSessionCommand{
		SessionID: sessionID,
	}

	if err := h.uc.StartSession(c.Request.Context(), cmd); err != nil {
		handleError(c, err)
		return
	}

	respondOK(c)
}

func (h *SessionHandler) CloseSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty session id"})
		return
	}

	cmd := sessionuc.CloseSessionCommand{
		SessionID: sessionID,
	}

	if err := h.uc.CloseSession(c.Request.Context(), cmd); err != nil {
		handleError(c, err)
		return
	}

	respondOK(c)
}

func (h *SessionHandler) BuyIn(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty session id"})
		return
	}

	var req buyInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	cmd := sessionuc.BuyInCommand{
		SessionID:   sessionID,
		OperationID: req.OperationID,
		PlayerID:    req.PlayerID,
		Money:       req.Money,
	}

	if err := h.uc.BuyIn(c.Request.Context(), cmd); err != nil {
		handleError(c, err)
		return
	}

	respondOK(c)
}

func (h *SessionHandler) CashOut(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty session id"})
		return
	}

	var req cashOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	cmd := sessionuc.CashOutCommand{
		SessionID:   sessionID,
		OperationID: req.OperationID,
		PlayerID:    req.PlayerID,
		Chips:       req.Chips,
	}

	if err := h.uc.CashOut(c.Request.Context(), cmd); err != nil {
		handleError(c, err)
		return
	}

	respondOK(c)
}

func (h *SessionHandler) GetResult(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty session id"})
		return
	}

	playerID := c.Param("player_id")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty player id"})
		return
	}

	q := sessionuc.GetResultQuery{
		SessionID: sessionID,
		PlayerID:  playerID,
	}

	result, err := h.uc.GetResult(c.Request.Context(), q)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"amount": result.Amount(),
	})
}

func respondOK(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *SessionHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty session id"})
		return
	}

	session, err := h.uc.GetSession(c.Request.Context(), sessionuc.GetSessionQuery{
		SessionID: sessionID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapSession(session))
}
