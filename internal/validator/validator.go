package validator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/klimeurt/secflow-collector/internal/config"
	"github.com/nats-io/nats.go"
)

// Validator is the main validator service
type Validator struct {
	config    *config.Config
	checker   *Checker
	processor *Processor
	nc        *nats.Conn
	sub       *nats.Subscription
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// New creates a new Validator instance
func New(cfg *config.Config) (*Validator, error) {
	// Connect to NATS
	nc, err := nats.Connect(cfg.NATSUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create checker
	checker, err := NewChecker(cfg)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create checker: %w", err)
	}

	// Create processor
	processor := NewProcessor(cfg, checker, nc)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	return &Validator{
		config:    cfg,
		checker:   checker,
		processor: processor,
		nc:        nc,
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// Start begins processing messages from the source queue
func (v *Validator) Start() error {
	log.Printf("Starting validator service...")
	log.Printf("Subscribing to subject: %s", v.config.SourceSubject)
	log.Printf("Valid repos will be sent to: %s", v.config.ValidReposSubject)
	log.Printf("Invalid repos will be sent to: %s", v.config.InvalidReposSubject)

	// Process any existing messages in the queue first
	if err := v.ProcessExistingMessages(); err != nil {
		return fmt.Errorf("failed to process existing messages: %w", err)
	}

	// Subscribe to the source subject
	sub, err := v.nc.Subscribe(v.config.SourceSubject, func(msg *nats.Msg) {
		v.wg.Add(1)
		go func() {
			defer v.wg.Done()
			
			// Process the message
			if err := v.processor.ProcessMessage(v.ctx, msg); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		}()
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", v.config.SourceSubject, err)
	}

	v.sub = sub
	log.Printf("Validator service started successfully")
	return nil
}

// ProcessExistingMessages processes any existing messages in the queue at startup
func (v *Validator) ProcessExistingMessages() error {
	if !v.config.ProcessStartupMessages {
		log.Printf("Startup message processing disabled, skipping...")
		return nil
	}

	log.Printf("Processing existing messages from queue: %s", v.config.SourceSubject)
	
	// Create a synchronous subscription for startup processing
	sub, err := v.nc.SubscribeSync(v.config.SourceSubject)
	if err != nil {
		return fmt.Errorf("failed to create sync subscription for startup processing: %w", err)
	}
	defer func() {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("Warning: failed to unsubscribe during startup processing: %v", err)
		}
	}()

	processedCount := 0
	timeout := 1 * time.Second // Short timeout to detect empty queue

	for {
		// Try to get next message with timeout
		msg, err := sub.NextMsg(timeout)
		if err != nil {
			if err == nats.ErrTimeout {
				// No more messages available, queue is empty
				log.Printf("No more existing messages found. Processed %d messages during startup.", processedCount)
				break
			}
			// Other error occurred
			return fmt.Errorf("error receiving message during startup processing: %w", err)
		}

		// Process the message using existing processor logic
		if err := v.processor.ProcessMessage(v.ctx, msg); err != nil {
			log.Printf("Error processing startup message: %v", err)
			// Continue processing other messages even if one fails
		} else {
			processedCount++
		}
	}

	log.Printf("Startup message processing completed. Processed %d messages.", processedCount)
	return nil
}

// Stop gracefully shuts down the validator service
func (v *Validator) Stop() {
	log.Printf("Stopping validator service...")
	
	// Cancel the context to signal shutdown
	v.cancel()
	
	// Unsubscribe from NATS
	if v.sub != nil {
		if err := v.sub.Unsubscribe(); err != nil {
			log.Printf("Warning: failed to unsubscribe: %v", err)
		}
	}
	
	// Wait for all goroutines to finish
	v.wg.Wait()
	
	// Close NATS connection
	if v.nc != nil {
		v.nc.Close()
	}
	
	log.Printf("Validator service stopped")
}

// Wait blocks until the service is stopped
func (v *Validator) Wait() {
	<-v.ctx.Done()
}