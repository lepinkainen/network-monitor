package main

import (
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"network-monitor/internal/config"
	"network-monitor/internal/database"
	"network-monitor/internal/monitor"
	"network-monitor/internal/ping"
	"network-monitor/internal/web"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	// Parse configuration
	cfg := config.ParseFlags()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize schema
	if err := db.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	// Initialize components
	pinger := ping.New()
	mon := monitor.New(cfg, db, pinger)
	webServer := web.New(db, cfg.Port, staticFiles)

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := mon.Start(); err != nil {
			log.Fatalf("Failed to start monitor: %v", err)
		}
	}()

	go func() {
		if err := webServer.Start(); err != nil {
			log.Fatalf("Failed to start web server: %v", err)
		}
	}()

	log.Printf("Monitor started. Pinging %v every %v", cfg.Targets, cfg.Interval)
	log.Printf("Web interface available at http://localhost:%d", cfg.Port)

	<-sigChan
	log.Println("Shutting down...")
	mon.Stop()
	mon.Wait()
}
