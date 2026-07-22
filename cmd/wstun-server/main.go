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
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Configure structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: config.SlogLevel(),
	}))

	logger.Info("starting server",
		"config", *configPath,
		"listen_addr", config.ListenAddr,
		"log_level", config.LogLevel,
	)

	if config.TLSCert != "" && config.TLSKey != "" {
		logger.Info("TLS enabled", "cert", config.TLSCert, "key", config.TLSKey)
	}

	if config.MaxConnections > 0 {
		logger.Info("max connections configured", "max", config.MaxConnections)
	}

	if config.RetryAttempts > 0 {
		logger.Info("retry configured",
			"attempts", config.RetryAttempts,
			"interval", config.RetryInterval,
			"max_interval", config.RetryMaxInterval,
		)
	}

	// Create and start server
	server, err := NewServer(config, logger)
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	if err := server.Start(); err != nil {
		logger.Error("failed to start server", "error", err)
		os.Exit(1)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Info("received signal, initiating graceful shutdown", "signal", sig)

	// Graceful shutdown
	if err := server.Shutdown(); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("shutdown complete")
}
