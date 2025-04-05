package main

import (
	"log/slog"
	"os"
	"shm/internal/config"
	"shm/internal/scheduler"
	"shm/internal/storage/sqlite"
)

func main() {
	config := config.New()

	db, err := sqlite.New(config.DatabaseFile)
	if err != nil {
		slog.Error("failed to create database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	scheduler.New(db).Start()
}
