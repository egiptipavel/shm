package main

import (
	"log/slog"
	"os"
	"shm/internal/checker"
	"shm/internal/config"
	"shm/internal/lib/logger"
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

	checker, err := checker.New(db, config.IntervalMins)
	if err != nil {
		slog.Error("failed to create checker", logger.Error(err))
		os.Exit(1)
	}
	defer checker.Close()

	slog.Info("starting checker service")
	checker.Start()
}
