package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wrale/wrale-fleet-metal/internal/server"
)

func main() {
	// Parse command line flags
	deviceID := flag.String("device-id", "", "Unique device identifier")
	httpAddr := flag.String("http-addr", ":8080", "HTTP API address")
	flag.Parse()

	if *deviceID == "" {
		log.Fatal("device-id is required")
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create server with config
	srv, err := server.New(server.Config{
		DeviceID: *deviceID,
		HTTPAddr: *httpAddr,
	})
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle shutdown gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	// Run server
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}