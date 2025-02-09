package main

import (
	"flag"
	"log"
	"shm/internal/config"
	"shm/internal/monitor"
	"shm/internal/storage"
)

var configPath = flag.String("config", "configs/config.json", "path to config file")

func main() {
	flag.Parse()

	config, err := config.ParseConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to parse config file: %s", err)
	}

	storage, err := storage.NewStorage(config.DBFile)
	if err != nil {
		log.Fatalf("failed to create storage: %s", err)
	}
	defer storage.Close()

	monitor.New(storage, *config).Start()
}
