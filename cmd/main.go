package main

import (
	"github.com/fleetcard/config"
	"github.com/fleetcard/db"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}
	db.Connect(cfg)
}
