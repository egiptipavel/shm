package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found")
	}
}

type Config struct {
	DatabaseFile  string
	TelegramToken string
	Address       string
	IntervalMins  int
}

func New() Config {
	return Config{
		DatabaseFile:  getEnv("DATABASE_FILE", "storage/shm.db"),
		TelegramToken: getEnv("TELEGRAM_TOKEN", ""),
		Address:       getEnv("ADDRESS", "localhost:8080"),
		IntervalMins:  getEnvAsInt("REQUEST_INTERVAL_MINS", 1),
	}
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultVal
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}
