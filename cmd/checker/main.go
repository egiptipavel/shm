package main

import (
	"log/slog"
	"os"
	"shm/internal/checker"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/lib/sl"
)

func main() {
	config := config.New()

	db := setup.ConnectToSQLite(config.DatabaseFile)
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(
		config.RabbitMQUser,
		config.RabbitMQPass,
		config.RabbitMQHost,
		config.RabbitMQPort,
	)
	defer broker.Close()

	slog.Info("creating checker service")
	checker, err := checker.New(db, broker, config.IntervalMins)
	if err != nil {
		slog.Error("failed to create checker", sl.Error(err))
		os.Exit(1)
	}

	slog.Info("starting checker service")
	checker.Start()
}
