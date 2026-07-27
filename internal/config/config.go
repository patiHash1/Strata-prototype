package config

import (
	"github.com/patiHash1/Strata-prototype/internal/env"
)

// Config holds all application configuration values.
// Add fields here as the API grows (DB, auth, etc.).
type Config struct {
	Port          int
	BaseURL       string
	EnableSwagger bool
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		Port:          env.GetInt("PORT", 8080),
		BaseURL:       env.GetString("BASE_URL", "http://localhost:8080"),
		EnableSwagger: env.GetBool("ENABLE_SWAGGER", true),
	}
}
