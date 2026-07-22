package main

import (
	"flag"
	"fmt"
	"log/slog"
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
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Configure structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: config.SlogLevel(),
	}))

	logger.Info("starting client",
		"config", *configPath,
		"server_url", config.ServerURL,
		"service_name", config.ServiceName,
		"target_addr", config.TargetAddr,
		"log_level", config.LogLevel,
	)

	if config.TLSInsecure {
		logger.Warn("TLS verification disabled (insecure)")
	}

	logger.Info("reconnect settings",
		"interval", config.ReconnectInterval,
		"max_attempts", config.MaxReconnectAttempts,
	)

	// Create client
	client := NewClient(config, logger)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run client with reconnection logic in a goroutine
	clientErr := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				clientErr <- fmt.Errorf("panic in client.Run(): %v", r)
			}
		}()
		clientErr <- client.Run()
	}()

	// Wait for either shutdown signal or fatal error
	select {
	case sig := <-sigChan:
		logger.Info("received signal, initiating graceful shutdown", "signal", sig)
		if err := client.Shutdown(); err != nil {
			logger.Error("shutdown error", "error", err)
		}
	case err := <-clientErr:
		if err != nil {
			logger.Error("client error", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("shutdown complete")
}
