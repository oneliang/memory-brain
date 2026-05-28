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
)

func main() {
	// Parse command line arguments
	command := flag.String("command", "server", "Command to run (server)")
	port := flag.Int("port", 12321, "Server port")
	flag.Parse()

	switch *command {
	case "server":
		runServer(*port)
	default:
		fmt.Printf("Unknown command: %s\n", *command)
		os.Exit(1)
	}
}

func runServer(port int) {
	// Create server
	server := api.NewServer(port)

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