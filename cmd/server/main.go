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
	cfg := config.NewServerConfig()

	db := setup.ConnectToSQLite(config.NewSQLiteConfig())
	defer db.Close()

	server := server.New(db, cfg)
	slog.Info("starting http server", slog.String("address", cfg.Address))
	if err := server.Start(); err != http.ErrServerClosed {
		slog.Error("error from http server", sl.Error(err))
	}
}
