package setup

import (
	"database/sql"
	"log/slog"
	"os"
	"shm/internal/broker/rabbitmq"
	"shm/internal/lib/logger"
	"shm/internal/storage/sqlite"
)

func ConnectToSQLite(dataSourceName string) *sql.DB {
	slog.Info("connecting to SQLite")
	db, err := sqlite.New(dataSourceName)
	if err != nil {
		slog.Error("failed to create database", logger.Error(err))
		os.Exit(1)
	}
	return db
}

func ConnectToRabbitMQ(url string) *rabbitmq.RabbitMQ {
	slog.Info("connecting to RabbitMQ")
	broker, err := rabbitmq.New(url)
	if err != nil {
		slog.Error("failed to connect to RabbitMQ", logger.Error(err))
		os.Exit(1)
	}
	return broker
}
