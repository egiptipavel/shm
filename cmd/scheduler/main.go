package main

import (
	"log/slog"
	"os"
	"shm/internal/broker/rabbitmq"
	"shm/internal/config"
	"shm/internal/lib/logger"
	"shm/internal/scheduler"
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

	slog.Info("starting scheduler service")
	scheduler.New(db, broker, config.IntervalMins).Start()
}
