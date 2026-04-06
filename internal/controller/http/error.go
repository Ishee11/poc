package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ishee11/poc/internal/entity"
)

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entity.ErrSessionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

	case errors.Is(err, entity.ErrSessionAlreadyExists):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	case errors.Is(err, entity.ErrNotEnoughChips):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	case errors.Is(err, entity.ErrSessionNotFinished):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	case errors.Is(err, entity.ErrPlayerNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
