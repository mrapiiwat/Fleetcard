package main

import (
	"log"
	"time"

	"github.com/fleetcard/config"
	"github.com/fleetcard/controllers"
	"github.com/fleetcard/db"
	"github.com/go-co-op/gocron"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db.DB = db.Connect(cfg)

	s := gocron.NewScheduler(time.Local)

	// รันทุกวันเวลา 05:00 AM
	// _, err = s.Every(1).Day().Every(1).Minute().Do(func() {   ->   "สำหรับเทส"
	_, err = s.Every(1).Day().At("05:00").Do(func() {
		log.Println("Running scheduled task at 05:00 AM")
		controllers.ProcessAllInboundFiles(cfg.DateFormat)
	})
	if err != nil {
		log.Fatalf("Scheduler setup failed: %v", err)
	}

	log.Println("Cron job scheduled. Waiting...")
	s.StartBlocking()
}
