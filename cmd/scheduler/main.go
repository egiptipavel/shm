package main

import (
	"log/slog"
	"os"
	"shm/internal/config"
	"shm/internal/lib/logger"
	"shm/internal/scheduler"
	"shm/internal/storage/sqlite"
)

func main() {
	config := config.New()

	db, err := sqlite.New(config.DatabaseFile)
	if err != nil {
		slog.Error("failed to create database", logger.Error(err))
		os.Exit(1)
	}
	defer db.Close()

	scheduler, err := scheduler.New(db, config.IntervalMins)
	if err != nil {
		slog.Error("failed to create scheduler", logger.Error(err))
		os.Exit(1)
	}
	defer scheduler.Close()

	slog.Info("starting scheduler service")
	scheduler.Start()
}
