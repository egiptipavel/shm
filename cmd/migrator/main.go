package main

import (
	"log/slog"
	"os"
	"shm/internal/config"
	"shm/internal/lib/setup"
	"shm/internal/lib/sl"

	"github.com/pressly/goose/v3"
)

func main() {
	cfg := config.NewMigratorConfig()

	db := setup.ConnectToDatabase(cfg.DbDriver)
	defer db.Close()

	if err := goose.SetDialect(cfg.DbDriver); err != nil {
		slog.Error("failed to set dialect", sl.Error(err))
		os.Exit(1)
	}

	if err := goose.Up(db.DB(), cfg.MigrationsFolder); err != nil {
		slog.Error("failed to applies all available migrations", sl.Error(err))
		os.Exit(1)
	}
}
