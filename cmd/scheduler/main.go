package main

import (
	"log/slog"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/scheduler"
)

func main() {
	config := config.New()

	db := setup.ConnectToSQLite(config.DatabaseFile)
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(config.RabbitMQ)
	defer broker.Close()

	slog.Info("starting scheduler service")
	scheduler.New(db, broker, config.IntervalMins).Start()
}
