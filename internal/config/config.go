package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseFile  string
	TelegramToken string
	IntervalMins  int
}

func New() *Config {
	return &Config{
		DatabaseFile:  getEnv("DATABASE_FILE", "shm.db"),
		TelegramToken: getEnv("TELEGRAM_TOKEN", ""),
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
