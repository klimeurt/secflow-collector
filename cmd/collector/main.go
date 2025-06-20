package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/klimeurt/secflow-collector/internal/collector"
	"github.com/klimeurt/secflow-collector/internal/config"
	"github.com/robfig/cron/v3"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create scanner
	scanner, err := collector.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Create cron scheduler
	c := cron.New()

	// Add job
	_, err = c.AddFunc(cfg.CronSchedule, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := scanner.ScanRepositories(ctx); err != nil {
			log.Printf("Scan failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to add cron job: %v", err)
	}

	// Start cron scheduler
	c.Start()
	log.Printf("Cron scheduler started with schedule: %s", cfg.CronSchedule)

	// Run immediately on startup if configured
	if cfg.RunOnStartup {
		log.Println("Running initial scan on startup...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := scanner.ScanRepositories(ctx); err != nil {
			log.Printf("Initial scan failed: %v", err)
		}
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	c.Stop()
}