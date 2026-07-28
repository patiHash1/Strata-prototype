package config

import (
	"github.com/patiHash1/Strata-prototype/internal/env"
)

// DBConfig holds database connection settings.
type DBConfig struct {
	DSN string
}

// Config holds all application configuration values.
type Config struct {
	Port          int
	BaseURL       string
	EnableSwagger bool
	DB            DBConfig
	JWTSecret     string
	JWTIssuer     string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		Port:          env.GetInt("PORT", 8080),
		BaseURL:       env.GetString("BASE_URL", "http://localhost:8080"),
		EnableSwagger: env.GetBool("ENABLE_SWAGGER", true),
		DB: DBConfig{
			DSN: env.GetString("DATABASE_URL", ""),
		},
		JWTSecret: env.GetString("JWT_SECRET", "dev-secret-change-in-production"),
		JWTIssuer: env.GetString("JWT_ISSUER", "strata"),
	}
}
