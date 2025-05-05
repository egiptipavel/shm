package main

import (
	"log/slog"
	"net/http"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/lib/sl"
	"shm/internal/server"
	"shm/internal/service"
)

func main() {
	cfg := config.NewServerConfig()

	db := setup.ConnectToDatabase(cfg.DbDriver)
	defer db.Close()

	sitesRepo := db.SitesRepo()
	sites := service.NewSitesService(sitesRepo, cfg.CommonConfig)

	server := server.New(sites, cfg)
	slog.Info("starting http server", slog.String("address", cfg.Address))
	if err := server.Start(); err != http.ErrServerClosed {
		slog.Error("error from http server", sl.Error(err))
	}
}
