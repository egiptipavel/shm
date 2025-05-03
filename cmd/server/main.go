package main

import (
	"log/slog"
	"net/http"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/lib/sl"
	"shm/internal/repository/sqlite"
	"shm/internal/server"
	"shm/internal/service"
)

func main() {
	cfg := config.NewServerConfig()

	db := setup.ConnectToPostgreSQL(config.NewPostgreSQLConfig())
	defer db.Close()

	sitesRepo := sqlite.NewSitesRepo(db)
	sites := service.NewSitesService(sitesRepo, cfg.CommonConfig)

	server := server.New(sites, cfg)
	slog.Info("starting http server", slog.String("address", cfg.Address))
	if err := server.Start(); err != http.ErrServerClosed {
		slog.Error("error from http server", sl.Error(err))
	}
}
