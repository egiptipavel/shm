package main

import (
	"log/slog"
	"net/http"
	"shm/internal/config"
	"shm/internal/lib/logger"
	"shm/internal/lib/setup"
	"shm/internal/server"
)

func main() {
	config := config.New()

	db := setup.ConnectToSQLite(config.DatabaseFile)
	defer db.Close()

	server := server.New(db, config.Address)
	slog.Info("starting http server", slog.String("address", config.Address))
	if err := server.Start(); err != http.ErrServerClosed {
		slog.Error("error from http server", logger.Error(err))
	}
}
