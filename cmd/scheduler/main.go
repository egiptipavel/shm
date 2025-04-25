package main

import (
	"log/slog"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/scheduler"
)

func main() {
	cfg := config.NewSchedulerConfig()

	db := setup.ConnectToSQLite(config.NewSQLiteConfig())
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(config.NewRabbitMQConfig())
	defer broker.Close()

	slog.Info("starting scheduler service")
	scheduler.New(db, broker, cfg).Start()
}
