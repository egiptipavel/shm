package config

import (
	"log/slog"
	"os"
	"shm/internal/lib/sl"
	"strconv"
	"strings"

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

type Config struct {
	DatabaseFile  string
	RabbitMQUser  string
	RabbitMQPass  string
	RabbitMQHost  string
	RabbitMQPort  string
	TelegramToken string
	ServerAddress string
	IntervalMins  int
}

func New() Config {
	return Config{
		DatabaseFile:  getEnv("DATABASE_FILE", "storage/shm.db"),
		RabbitMQUser:  getEnv("RABBITMQ_DEFAULT_USER", "guest"),
		RabbitMQPass:  getEnv("RABBITMQ_DEFAULT_PASS", "guest"),
		RabbitMQHost:  getEnv("RABBITMQ_NODE_IP_ADDRESS", "rabbitmq"),
		RabbitMQPort:  getEnv("RABBITMQ_NODE_PORT", "5672"),
		TelegramToken: getEnv("TELEGRAM_TOKEN_FILE", ""),
		ServerAddress: getEnv("SERVER_ADDRESS", "server:8080"),
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
		if strings.HasSuffix(key, "_FILE") {
			content, err := os.ReadFile(value)
			if err != nil {
				slog.Error("failed to read file", sl.Error(err))
				os.Exit(1)
			}
			return string(content)
		}
		return value
	}

	return defaultVal
}
