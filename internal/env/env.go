package env

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// LoadDotenv reads the .env file in the project root and loads its
// variables into the process environment. It is safe to call multiple
// times — subsequent calls are no-ops.
func LoadDotenv() {
	// Ignore error — .env file is optional in production.
	_ = godotenv.Load()
}

// GetString returns the value of the environment variable named by key,
// or fallback if it is empty or unset.
func GetString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// GetInt returns the value of the environment variable named by key
// parsed as an int, or fallback on error.
func GetInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// GetBool returns the value of the environment variable named by key
// parsed as a bool, or fallback on error.
func GetBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
