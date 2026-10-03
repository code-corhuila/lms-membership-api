// Package config loads configuration from environment variables — the only source of
// environment-specific configuration (library-docs/05-architecture/cross-cutting.md).
package config

import (
	"crypto/rsa"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	Port string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// JWTPublicKey validates a real Administrator session token, issued by
	// lms-access-api and signed RS256 — this service never holds the private
	// key (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.7).
	JWTPublicKey *rsa.PublicKey
	// InternalJWTSecret is a second, deliberately separate secret: only for
	// the short-lived HS256 tokens services mint for each other
	// (internal/adapter/out/circulationclient). Keeping it apart from the
	// RS256 user-session key — rather than accepting "whatever alg the token
	// declares" against one shared key — is what closes the classic
	// RS256-to-HS256 key-confusion hole: internal/adapter/in/httpapi/middleware/auth.go
	// picks which key to use from the token's declared algorithm, and an
	// RS256 token can never be re-verified as HS256 using the public key's
	// own bytes as a forged secret, because that secret is never this one.
	InternalJWTSecret string
	JWTExpiry         time.Duration

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

	publicKeyPEM := getEnv("JWT_PUBLIC_KEY", "")
	if publicKeyPEM == "" {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY must be set")
	}
	// .env stores a PEM as one line with literal \n — rules/2-anexos/C-api-hexagonal.md,
	// numeral 5.3.7.
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(strings.ReplaceAll(publicKeyPEM, `\n`, "\n")))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_PUBLIC_KEY: %w", err)
	}

	cfg := &Config{
		Port: getEnv("PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "lms_user"),
		DBPassword: getEnv("DB_PASSWORD", "lms_password"),
		DBName:     getEnv("DB_NAME", "lms_db"),

		JWTPublicKey:      publicKey,
		InternalJWTSecret: getEnv("INTERNAL_JWT_SECRET", ""),
		JWTExpiry:         jwtExpiry,

		LogLevel:   getEnv("LOG_LEVEL", "info"),
		CORSOrigin: getEnv("CORS_ORIGIN", "*"),

		CirculationServiceURL: getEnv("CIRCULATION_SERVICE_URL", "http://circulation-service:8080"),
	}

	if cfg.InternalJWTSecret == "" {
		return nil, fmt.Errorf("INTERNAL_JWT_SECRET must be set")
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
