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
		fmt.Fprintf(os.Stderr, "WebSocket tunnel server\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -c /etc/wstun/server.yaml\n", os.Args[0])
	}
	flag.Parse()

	// Validate required flag
	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: configuration file is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	config, err := wstun.LoadServerConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Configure logger based on log level
	logPrefix := fmt.Sprintf("[wstun-server] ")
	logger := log.New(os.Stdout, logPrefix, log.LstdFlags)
	logger.Printf("Starting server with config from: %s", *configPath)
	logger.Printf("Listen address: %s", config.ListenAddr)
	logger.Printf("Log level: %s", config.LogLevel)

	if config.TLSCert != "" && config.TLSKey != "" {
		logger.Printf("TLS enabled (cert: %s, key: %s)", config.TLSCert, config.TLSKey)
	}

	if config.MaxConnections > 0 {
		logger.Printf("Max connections: %d", config.MaxConnections)
	} else {
		logger.Printf("Max connections: unlimited")
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// TODO: Start actual server implementation here
	// For now, just wait for shutdown signal
	logger.Printf("Server started successfully (implementation pending)")

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Printf("Received signal %v, initiating graceful shutdown...", sig)

	// TODO: Implement graceful shutdown of connections
	logger.Printf("Shutdown complete")
}
