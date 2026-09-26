// Package config loads configuration from environment variables — the only source of
// environment-specific configuration (library-docs/05-architecture/cross-cutting.md).
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	JWTSecret string
	JWTExpiry time.Duration

	LogLevel   string
	CORSOrigin string

	// CirculationServiceURL is where DeactivateStudent asks whether a student
	// has active loans — Membership no longer has DB access to that table.
	CirculationServiceURL string
}

func Load() (*Config, error) {
	jwtExpiry, err := time.ParseDuration(getEnv("JWT_EXPIRY", "1h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY: %w", err)
	}

	cfg := &Config{
		Port: getEnv("PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "lms_user"),
		DBPassword: getEnv("DB_PASSWORD", "lms_password"),
		DBName:     getEnv("DB_NAME", "lms_db"),

		JWTSecret: getEnv("JWT_SECRET", ""),
		JWTExpiry: jwtExpiry,

		LogLevel:   getEnv("LOG_LEVEL", "info"),
		CORSOrigin: getEnv("CORS_ORIGIN", "*"),

		CirculationServiceURL: getEnv("CIRCULATION_SERVICE_URL", "http://circulation-service:8080"),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET must be set")
	}

	return cfg, nil
}

// DSN builds the PostgreSQL connection string (pgx).
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
