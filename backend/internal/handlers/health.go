package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

// GetHealth implements GET /health.
func (h *Handler) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, api.Health{
		Status: "ok",
		Time:   time.Now().UTC(),
	})
}
