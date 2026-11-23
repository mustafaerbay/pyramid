package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/config"
	"github.com/mustafaerbay/pyramid/backend/internal/database"
	"github.com/mustafaerbay/pyramid/backend/internal/handlers"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	if cfg.Env != "development" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	// Only the internal router/proxies (private networks) may tell us the
	// client's address via X-Forwarded-For.
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}); err != nil {
		log.Fatalf("trusted proxies: %v", err)
	}
	h := handlers.New(db, cfg)
	if err := h.EnsureAdmin(); err != nil {
		log.Fatalf("creating the first admin failed: %v", err)
	}
	h.RegisterRoutes(r)

	log.Printf("Pyramid API listening on :%s (env=%s)", cfg.Port, cfg.Env)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
