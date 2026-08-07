package config

import (
	"github.com/patiHash1/Strata-prototype/internal/env"
)

// DBConfig holds database connection settings.
type DBConfig struct {
	DSN string
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// Config holds all application configuration values.
type Config struct {
	Port            int
	EnableSwagger   bool
	DB              DBConfig
	Redis           RedisConfig
	JWTSecret       string
	JWTIssuer       string
	SuperAdminUname string
	SuperAdminPword string
}

// Load reads configuration from environment variables.
// It loads the .env file before reading so local development works.
func Load() Config {
	env.LoadDotenv()

	return Config{
		Port:          env.GetInt("PORT", 8080),
		EnableSwagger: env.GetBool("ENABLE_SWAGGER", true),
		DB: DBConfig{
			DSN: env.GetString("DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			Addr:     env.GetString("REDIS_ADDR", ""),
			Password: env.GetString("REDIS_PASSWORD", ""),
			DB:       env.GetInt("REDIS_DB", 0),
		},
		JWTSecret:       env.GetString("JWT_SECRET", "dev-secret-change-in-production"),
		JWTIssuer:       env.GetString("JWT_ISSUER", "strata"),
		SuperAdminUname: env.GetString("SUPERADMIN_UNAME", "admin@strata.local"),
		SuperAdminPword: env.GetString("SUPERADMIN_PWORD", "SuperAdmin123!"),
	}
}
