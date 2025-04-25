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
	cfg := config.NewCheckerConfig()

	db := setup.ConnectToSQLite(config.NewSQLiteConfig())
	defer db.Close()

	broker := setup.ConnectToRabbitMQ(config.NewRabbitMQConfig())
	defer broker.Close()

	slog.Info("creating checker service")
	checker, err := checker.New(db, broker, cfg)
	if err != nil {
		slog.Error("failed to create checker", sl.Error(err))
		os.Exit(1)
	}

	slog.Info("starting checker service")
	checker.Start()
}
