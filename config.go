package wstun

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ServerOptions represents the configuration for the WebSocket tunnel server.
type ServerOptions struct {
	// ListenAddr is the address to bind the server (e.g., ":8080" or "0.0.0.0:8080")
	ListenAddr string `yaml:"listen_addr"`

	// TLSCert is the path to the TLS certificate file (optional)
	TLSCert string `yaml:"tls_cert"`

	// TLSKey is the path to the TLS private key file (optional)
	TLSKey string `yaml:"tls_key"`

	// LogLevel specifies logging verbosity (optional, default: "info")
	LogLevel string `yaml:"log_level"`

	// MaxConnections is the maximum number of concurrent connections (optional, 0 = unlimited)
	MaxConnections int `yaml:"max_connections"`
}

// Validate checks if the ServerOptions configuration is valid.
func (o *ServerOptions) Validate() error {
	if o.ListenAddr == "" {
		return fmt.Errorf("listen_addr is required")
	}

	// If one TLS option is set, both must be set
	if (o.TLSCert != "" && o.TLSKey == "") || (o.TLSCert == "" && o.TLSKey != "") {
		return fmt.Errorf("both tls_cert and tls_key must be set together")
	}

	// Validate TLS files exist if specified
	if o.TLSCert != "" {
		if _, err := os.Stat(o.TLSCert); err != nil {
			return fmt.Errorf("tls_cert file not found: %w", err)
		}
	}
	if o.TLSKey != "" {
		if _, err := os.Stat(o.TLSKey); err != nil {
			return fmt.Errorf("tls_key file not found: %w", err)
		}
	}

	// Validate log level
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if o.LogLevel != "" && !validLogLevels[o.LogLevel] {
		return fmt.Errorf("log_level must be one of: debug, info, warn, error")
	}

	if o.MaxConnections < 0 {
		return fmt.Errorf("max_connections must be >= 0")
	}

	return nil
}

// SetDefaults applies default values to optional fields.
func (o *ServerOptions) SetDefaults() {
	if o.LogLevel == "" {
		o.LogLevel = "info"
	}
}

// ClientOptions represents the configuration for the WebSocket tunnel client.
type ClientOptions struct {
	// ServerURL is the WebSocket server URL to connect to (required)
	ServerURL string `yaml:"server_url"`

	// LocalAddr is the local address to listen on for incoming connections (required)
	LocalAddr string `yaml:"local_addr"`

	// TLSInsecure skips TLS certificate verification (optional, default: false)
	TLSInsecure bool `yaml:"tls_insecure"`

	// ReconnectInterval is the time between reconnection attempts (optional, default: 5s)
	ReconnectInterval time.Duration `yaml:"reconnect_interval"`

	// MaxReconnectAttempts is the maximum number of reconnection attempts (optional, 0 = unlimited)
	MaxReconnectAttempts int `yaml:"max_reconnect_attempts"`

	// LogLevel specifies logging verbosity (optional, default: "info")
	LogLevel string `yaml:"log_level"`
}

// Validate checks if the ClientOptions configuration is valid.
func (o *ClientOptions) Validate() error {
	if o.ServerURL == "" {
		return fmt.Errorf("server_url is required")
	}

	if o.LocalAddr == "" {
		return fmt.Errorf("local_addr is required")
	}

	// Validate log level
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if o.LogLevel != "" && !validLogLevels[o.LogLevel] {
		return fmt.Errorf("log_level must be one of: debug, info, warn, error")
	}

	if o.ReconnectInterval < 0 {
		return fmt.Errorf("reconnect_interval must be >= 0")
	}

	if o.MaxReconnectAttempts < 0 {
		return fmt.Errorf("max_reconnect_attempts must be >= 0")
	}

	return nil
}

// SetDefaults applies default values to optional fields.
func (o *ClientOptions) SetDefaults() {
	if o.LogLevel == "" {
		o.LogLevel = "info"
	}
	if o.ReconnectInterval == 0 {
		o.ReconnectInterval = 5 * time.Second
	}
}

// LoadServerConfig loads and validates ServerOptions from a YAML file.
func LoadServerConfig(path string) (*ServerOptions, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var opts ServerOptions
	if err := yaml.Unmarshal(data, &opts); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	opts.SetDefaults()

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &opts, nil
}

// LoadClientConfig loads and validates ClientOptions from a YAML file.
func LoadClientConfig(path string) (*ClientOptions, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var opts ClientOptions
	if err := yaml.Unmarshal(data, &opts); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	opts.SetDefaults()

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &opts, nil
}
