package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// AppConfig holds all configuration values for the application.
type AppConfig struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	Addr       string
}

// Load reads the given .env file and returns an AppConfig populated from its
// values. Environment variables already set in the shell take precedence over
// values in the file. If the file does not exist, Load falls back to OS
// environment variables and built-in defaults.
func Load(file string) (*AppConfig, error) {
	if err := godotenv.Load(file); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load config %q: %w", file, err)
	}

	return &AppConfig{
		DBHost:     getenv("DB_HOST", "localhost"),
		DBPort:     getenv("DB_PORT", "5432"),
		DBUser:     getenv("DB_USER", "postgres"),
		DBPassword: getenv("DB_PASSWORD", ""),
		DBName:     getenv("DB_NAME", "pokemon"),
		DBSSLMode:  getenv("DB_SSLMODE", "disable"),
		Addr:       getenv("ADDR", ":8080"),
	}, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
