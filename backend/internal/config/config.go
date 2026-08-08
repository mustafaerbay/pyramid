package config

import (
	"fmt"
	"os"
)

// Config holds runtime configuration, sourced from environment variables.
type Config struct {
	Port        string
	DatabaseURL string
	Env         string
	CORSOrigin  string
}

// Load reads configuration from the environment, applying sensible defaults
// for local development.
func Load() Config {
	return Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: getenv("DATABASE_URL", defaultDatabaseURL()),
		Env:         getenv("APP_ENV", "development"),
		CORSOrigin:  getenv("CORS_ORIGIN", "http://localhost:5173"),
	}
}

func defaultDatabaseURL() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		getenv("DB_HOST", "localhost"),
		getenv("DB_PORT", "5432"),
		getenv("DB_USER", "postgres"),
		getenv("DB_PASSWORD", "postgres"),
		getenv("DB_NAME", "pyramid"),
		getenv("DB_SSLMODE", "disable"),
	)
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
