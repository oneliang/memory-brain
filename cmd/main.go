package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/oneliang/memory-brain/internal/api"
	"github.com/oneliang/memory-brain/internal/config"
)

func main() {
	// Parse command line arguments
	command := flag.String("command", "server", "Command to run (server)")
	port := flag.Int("port", 0, "Server port (0 = use config)")
	flag.Parse()

	switch *command {
	case "server":
		runServer(*port)
	default:
		fmt.Printf("Unknown command: %s\n", *command)
		os.Exit(1)
	}
}

func runServer(portOverride int) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Use command line port if specified, otherwise use config
	port := cfg.Server.Port
	if portOverride > 0 {
		port = portOverride
	}

	// Create server with config
	server := api.NewServer(port, cfg)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v, shutting down...", sig)
		cancel()
	}()

	// Start server
	log.Printf("Memory Brain server starting on port %d...", port)
	if err := server.Start(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}