package main

import (
	"log/slog"
	"net/http"
	"os"
	"shm/internal/config"
	"shm/internal/server"
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

	server := server.New(db, config.Address)
	slog.Info("starting http server", slog.String("address", config.Address))
	if err := server.Start(); err != http.ErrServerClosed {
		slog.Error("error from http server", slog.String("error", err.Error()))
	}
}
