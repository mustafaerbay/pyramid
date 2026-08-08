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
	handlers.New(db, cfg).RegisterRoutes(r)

	log.Printf("Pyramid API listening on :%s (env=%s)", cfg.Port, cfg.Env)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
