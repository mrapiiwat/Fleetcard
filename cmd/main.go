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
	
	db.DB = db.Connect(cfg)

	controllers.ProcessAllInboundFiles(cfg.DateFormat)
}
