package main

import (
	"log/slog"
	"net/http"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/lib/sl"
	"shm/internal/server"
)

func main() {
	config := config.New()

	db := setup.ConnectToSQLite(config.DatabaseFile)
	defer db.Close()

	server := server.New(db, config.ServerAddress)
	slog.Info("starting http server", slog.String("address", config.ServerAddress))
	if err := server.Start(); err != http.ErrServerClosed {
		slog.Error("error from http server", sl.Error(err))
	}
}
