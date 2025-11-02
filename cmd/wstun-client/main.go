package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"cnb.cool/yankeguo/wstun"
)

func main() {
	// Define flags
	configPath := flag.String("c", "", "path to configuration file (required)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -c <config-file>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "WebSocket tunnel client\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -c /etc/wstun/client.yaml\n", os.Args[0])
	}
	flag.Parse()

	// Validate required flag
	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: configuration file is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	config, err := wstun.LoadClientConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Configure logger based on log level
	logPrefix := fmt.Sprintf("[wstun-client] ")
	logger := log.New(os.Stdout, logPrefix, log.LstdFlags)
	logger.Printf("Starting client with config from: %s", *configPath)
	logger.Printf("Server URL: %s", config.ServerURL)
	logger.Printf("Local address: %s", config.LocalAddr)
	logger.Printf("Log level: %s", config.LogLevel)

	if config.TLSInsecure {
		logger.Printf("TLS verification: disabled (insecure)")
	} else {
		logger.Printf("TLS verification: enabled")
	}

	logger.Printf("Reconnect interval: %v", config.ReconnectInterval)

	if config.MaxReconnectAttempts > 0 {
		logger.Printf("Max reconnect attempts: %d", config.MaxReconnectAttempts)
	} else {
		logger.Printf("Max reconnect attempts: unlimited")
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// TODO: Start actual client implementation here
	// For now, just wait for shutdown signal
	logger.Printf("Client started successfully (implementation pending)")

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Printf("Received signal %v, initiating graceful shutdown...", sig)

	// TODO: Implement graceful shutdown of connections
	logger.Printf("Shutdown complete")
}

