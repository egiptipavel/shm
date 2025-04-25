package config

import (
	"log/slog"
	"os"
	"shm/internal/lib/sl"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	var envFiles []string
	if rabbitmqEnvFile, exists := os.LookupEnv("RABBITMQ_ENV_FILE"); exists {
		envFiles = append(envFiles, rabbitmqEnvFile)
	}
	envFiles = append(envFiles, ".env")
	if err := godotenv.Load(envFiles...); err != nil {
		slog.Warn("error from loading env variables", sl.Error(err))
	}
}

type CommonConfig struct {
	DbQueryTimeoutSec      time.Duration
	BrokerTimeoutSec       time.Duration
	SiteResponseTimeoutSec time.Duration
}

func NewCommonConfig() CommonConfig {
	return CommonConfig{
		DbQueryTimeoutSec:      time.Duration(getEnvAsInt("DATABASE_QUERY_TIMEOUT_SEC", 5)) * time.Second,
		BrokerTimeoutSec:       time.Duration(getEnvAsInt("BROKER_TIMEOUT_SEC", 5)) * time.Second,
		SiteResponseTimeoutSec: time.Duration(getEnvAsInt("SITE_RESPONSE_TIMEOUT_SEC", 5)) * time.Second,
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

func getEnvFromFile(key string, defaultVal string) string {
	if !strings.HasSuffix(key, "_FILE") {
		slog.Error("env variable with _FILE at the end was expected", slog.String("given", key))
		os.Exit(1)
	}

	if value, exists := os.LookupEnv(key); exists {
		content, err := os.ReadFile(value)
		if err != nil {
			slog.Error("failed to read file", sl.Error(err))
			os.Exit(1)
		}
		return string(content)
	}

	return defaultVal
}
