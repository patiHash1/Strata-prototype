package config

import (
	"github.com/patiHash1/Strata-prototype/internal/env"
	"github.com/patiHash1/Strata-prototype/internal/logger"
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
	AllowedOrigins  string
}

// Load reads configuration from environment variables.
// It loads the .env file before reading so local development works.
// It fatally exits if required security-sensitive variables are missing.
func Load() Config {
	env.LoadDotenv()

	jwtSecret := env.GetString("JWT_SECRET", "")
	if jwtSecret == "" {
		logger.Error("JWT_SECRET environment variable is required")
		panic("JWT_SECRET environment variable is required")
	}

	superAdminUname := env.GetString("SUPERADMIN_UNAME", "")
	superAdminPword := env.GetString("SUPERADMIN_PWORD", "")

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
		JWTSecret:       jwtSecret,
		JWTIssuer:       env.GetString("JWT_ISSUER", "strata"),
		SuperAdminUname: superAdminUname,
		SuperAdminPword: superAdminPword,
		AllowedOrigins:  env.GetString("ALLOWED_ORIGINS", ""),
	}
}
