package setup

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"shm/internal/broker/rabbitmq"
	"shm/internal/config"
	"shm/internal/db/postgres"
	"shm/internal/db/sqlite"
	"shm/internal/lib/sl"
)

func ConnectToSQLite(config config.SQLiteConfig) *sql.DB {
	slog.Info("connecting to SQLite")
	db, err := sqlite.New(config.File)
	if err != nil {
		slog.Error("failed to create database", sl.Error(err))
		os.Exit(1)
	}
	return db
}

func ConnectToPostgreSQL(config config.PostgreSQLConfig) *sql.DB {
	slog.Info("connecting to PostgreSQL")
	db, err := postgres.New(config)
	if err != nil {
		slog.Error("failed to create database", sl.Error(err))
		os.Exit(1)
	}
	return db
}

func ConnectToRabbitMQ(config config.RabbitMQConfig) *rabbitmq.RabbitMQ {
	slog.Info("connecting to RabbitMQ")
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", config.User, config.Pass, config.Host, config.Port)
	broker, err := rabbitmq.New(url)
	if err != nil {
		slog.Error("failed to connect to RabbitMQ", sl.Error(err))
		os.Exit(1)
	}
	return broker
}
