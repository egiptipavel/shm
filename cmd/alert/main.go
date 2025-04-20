package main

import (
	"log/slog"
	"os"
	"shm/internal/alert"
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

	alert := alert.New(db, broker)
	slog.Info("starting alert service")
	if err := alert.Start(); err != nil {
		slog.Error("error from alert service", sl.Error(err))
		os.Exit(1)
	}
}
