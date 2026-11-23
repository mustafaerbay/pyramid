package config

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
)

// Config holds runtime configuration, sourced from environment variables.
type Config struct {
	Port        string
	DatabaseURL string
	Env         string
	CORSOrigin  string

	// SessionSecret signs session cookies. From SESSION_SECRET; when unset a
	// random one is made at start, so sessions end on every restart.
	SessionSecret []byte
	// CookieSecure marks the session cookie Secure (HTTPS only).
	CookieSecure bool
	// AdminEmail/AdminPassword create the first admin account on start if
	// no account has that email yet (ADMIN_EMAIL, ADMIN_PASSWORD).
	AdminEmail    string
	AdminPassword string
	// AuthOff skips sign-in checks. Only tests set it; Load never does.
	AuthOff bool

	// Email for notifications; off unless SMTP_HOST and SMTP_FROM are set.
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	// PublicURL makes links in emails absolute (PUBLIC_URL).
	PublicURL string
}

// Load reads configuration from the environment, applying sensible defaults
// for local development.
func Load() Config {
	return Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: getenv("DATABASE_URL", defaultDatabaseURL()),
		Env:         getenv("APP_ENV", "development"),
		CORSOrigin:  getenv("CORS_ORIGIN", "http://localhost:5173"),

		SessionSecret: sessionSecret(),
		CookieSecure:  getenv("COOKIE_SECURE", "") == "true",
		AdminEmail:    os.Getenv("ADMIN_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),

		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     getenv("SMTP_PORT", "587"),
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     os.Getenv("SMTP_FROM"),
		PublicURL:    os.Getenv("PUBLIC_URL"),
	}
}

func sessionSecret() []byte {
	if s := os.Getenv("SESSION_SECRET"); len(s) >= 32 {
		return []byte(s)
	} else if s != "" {
		log.Fatal("SESSION_SECRET must be at least 32 characters")
	}
	log.Print("SESSION_SECRET not set: using a random one, sessions end when the server restarts")
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("random session secret: %v", err)
	}
	return b
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
