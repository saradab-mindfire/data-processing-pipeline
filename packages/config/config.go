package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds every value the app needs from the environment.
type Config struct {
	ServerAddr string
	APIKey     string

	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
	DBPort     string
	DBSSLMode  string

	WorkerAddr string
	WorkerURL string
}

// Load reads a local .env file (if present) and then the process
// environment, falling back to local-dev defaults for anything unset.
func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on process environment variables")
	}
	fmt.Println("XYZ", getEnv("API_KEY", ""))

	return Config{
		ServerAddr: getEnv("SERVER_ADDR", "localhost:9090"),
		APIKey:     getEnv("API_KEY", ""),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBUser:     getEnv("DB_USER", "admin"),
		DBPassword: getEnv("DB_PASSWORD", "admin123"),
		DBName:     getEnv("DB_NAME", "data-processing-pipeline"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		WorkerAddr: getEnv("WORKER_ADDR", "localhost:9091"),
		WorkerURL:  getEnv("WORKER_URL", "http://localhost:9091"),
	}
}

func (c Config) DATABASEURL() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
