package http

import "github.com/gin-gonic/gin"

func NewRouter(h *SessionHandler) *gin.Engine {
	r := gin.New()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	sessions := r.Group("/sessions")
	{
		sessions.POST("", h.CreateSession)
		sessions.POST("/:id/start", h.StartSession)
		sessions.POST("/:id/close", h.CloseSession)

		sessions.POST("/:id/buyin", h.BuyIn)
		sessions.POST("/:id/cashout", h.CashOut)

		sessions.GET("/:id/result/:player_id", h.GetResult)
	}

	return r
}
