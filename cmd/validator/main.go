package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/klimeurt/secflow-collector/internal/config"
	"github.com/klimeurt/secflow-collector/internal/validator"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create validator service
	v, err := validator.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	// Start the validator service
	if err := v.Start(); err != nil {
		log.Fatalf("Failed to start validator: %v", err)
	}
	defer v.Stop()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Received shutdown signal, stopping validator...")
}