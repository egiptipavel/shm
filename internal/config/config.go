package config

import (
	"fmt"
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
var brokers = []string{"rabbitmq"}

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
	MessageBroker          string
	BrokerTimeoutSec       time.Duration
	SiteResponseTimeoutSec time.Duration
}

func NewCommonConfig() CommonConfig {
	return CommonConfig{
		DbDriver:               getEnvFrom("DATABASE_DRIVER", drivers, "postgres"),
		DbQueryTimeoutSec:      getEnvAsDuration("DATABASE_QUERY_TIMEOUT_SEC", 5*time.Second),
		MessageBroker:          getEnvFrom("MESSAGE_BROKER", brokers, "rabbitmq"),
		BrokerTimeoutSec:       getEnvAsDuration("BROKER_TIMEOUT_SEC", 5*time.Second),
		SiteResponseTimeoutSec: getEnvAsDuration("SITE_RESPONSE_TIMEOUT_SEC", 5*time.Second),
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
		slog.Error("failed to parse env variable as int", slog.String("env_var", key), sl.Error(err))
		os.Exit(1)
	}

	return value
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valueInt := getEnvAsInt(key, -1)
	if valueInt == -1 {
		return defaultVal
	}

	words := strings.Split(key, "_")
	lastWord := words[len(words)-1]
	timeUnit, err := toTimeUnit(lastWord)
	if err != nil {
		slog.Error("invalid env variable", slog.String("env_var", key))
	}

	return time.Duration(valueInt) * timeUnit
}

func toTimeUnit(s string) (time.Duration, error) {
	var timeUnit time.Duration
	switch s {
	case "NS":
		timeUnit = time.Nanosecond
	case "US":
		timeUnit = time.Microsecond
	case "MS":
		timeUnit = time.Millisecond
	case "SEC":
		timeUnit = time.Second
	case "MIN":
		timeUnit = time.Minute
	case "HOUR":
		timeUnit = time.Hour
	default:
		return 0, fmt.Errorf("unknown time unit: %s", s)
	}
	return timeUnit, nil
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
