package config

import (
	"log/slog"
	"os"
	"shm/internal/lib/sl"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var drivers = []string{"postgres", "sqlite"}

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
	DbDriver               string
	DbQueryTimeoutSec      time.Duration
	BrokerTimeoutSec       time.Duration
	SiteResponseTimeoutSec time.Duration
}

func NewCommonConfig() CommonConfig {
	return CommonConfig{
		DbDriver:               getEnvFrom("DATABASE_DRIVER", drivers, "postgres"),
		DbQueryTimeoutSec:      time.Duration(getEnvAsInt("DATABASE_QUERY_TIMEOUT_SEC", 5)) * time.Second,
		BrokerTimeoutSec:       time.Duration(getEnvAsInt("BROKER_TIMEOUT_SEC", 5)) * time.Second,
		SiteResponseTimeoutSec: time.Duration(getEnvAsInt("SITE_RESPONSE_TIMEOUT_SEC", 5)) * time.Second,
	}
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultVal
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		slog.Error(
			"failed to parse env variable as int",
			slog.String("env_var", key),
			slog.String("given", key),
		)
		os.Exit(1)
	}

	return value
}

func getEnvFrom(key string, values []string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		if !slices.Contains(values, value) {
			slog.Error(
				"unexpected value of env variable",
				slog.String("env_var", key),
				slog.String("value", value),
				slog.Any("expected", strings.Join(values, " | ")),
			)
			os.Exit(1)
		}
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
