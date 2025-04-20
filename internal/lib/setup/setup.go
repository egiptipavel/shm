package setup

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"shm/internal/broker/rabbitmq"
	"shm/internal/lib/sl"
	"shm/internal/storage/sqlite"
)

func ConnectToSQLite(dataSourceName string) *sql.DB {
	slog.Info("connecting to SQLite")
	db, err := sqlite.New(dataSourceName)
	if err != nil {
		slog.Error("failed to create database", sl.Error(err))
		os.Exit(1)
	}
	return db
}

func ConnectToRabbitMQ(user, pass, host, port string) *rabbitmq.RabbitMQ {
	slog.Info("connecting to RabbitMQ")
	broker, err := rabbitmq.New(fmt.Sprintf("amqp://%s:%s@%s:%s/", user, pass, host, port))
	if err != nil {
		slog.Error("failed to connect to RabbitMQ", sl.Error(err))
		os.Exit(1)
	}
	return broker
}
