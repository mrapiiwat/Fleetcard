package main

import (
	"log"

	"github.com/fleetcard/config"
	"github.com/fleetcard/controllers"
	"github.com/fleetcard/db"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	connectedDB := db.Connect(cfg)
	db.DB = connectedDB

	controllers.DownloadAllGPGFromSFTP()
}
