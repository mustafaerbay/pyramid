package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/config"
	"gorm.io/gorm"
)

// Handler carries shared dependencies for all HTTP handlers.
type Handler struct {
	db  *gorm.DB
	cfg config.Config
}

// New builds a Handler.
func New(db *gorm.DB, cfg config.Config) *Handler {
	return &Handler{db: db, cfg: cfg}
}

// RegisterRoutes wires every route. Route paths mirror openapi/openapi.yaml.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.Use(h.cors())

	r.GET("/health", h.GetHealth)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/people", h.ListPeople)
		v1.POST("/people", h.CreatePerson)
		v1.GET("/people/:id", h.GetPerson)
		v1.PATCH("/people/:id", h.UpdatePerson)
		v1.DELETE("/people/:id", h.DeletePerson)
	}
}

// cors is a minimal CORS middleware for the local Svelte dev server.
func (h *Handler) cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", h.cfg.CORSOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// --- small shared helpers ---

func strptr(s string) *string { return &s }

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
