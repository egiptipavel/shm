package main

import (
	"log/slog"
	"os"
	"shm/internal/broker/rabbitmq"
	"shm/internal/checker"
	"shm/internal/config"
	"shm/internal/lib/logger"
	"shm/internal/storage/sqlite"
)

func main() {
	config := config.New()

	slog.Info("connecting to database")
	db, err := sqlite.New(config.DatabaseFile)
	if err != nil {
		slog.Error("failed to create database", logger.Error(err))
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("connecting to message broker")
	broker, err := rabbitmq.New(config.RabbitMQ)
	if err != nil {
		slog.Error("failed to connect to RabbitMQ", logger.Error(err))
		os.Exit(1)
	}
	defer broker.Close()

	slog.Info("creating checker service")
	checker, err := checker.New(db, broker, config.IntervalMins)
	if err != nil {
		slog.Error("failed to create checker", logger.Error(err))
		os.Exit(1)
	}

	slog.Info("starting checker service")
	checker.Start()
}
