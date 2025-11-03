package wstun

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ServiceDefinition represents a single service configuration on the server.
type ServiceDefinition struct {
	// ServiceToken is the authentication token for this service
	ServiceToken string `yaml:"service_token"`

	// ListenAddr is the address where this service will be exposed on the server
	ListenAddr string `yaml:"listen_addr"`
}

// Validate checks if the ServiceDefinition is valid.
func (s *ServiceDefinition) Validate() error {
	if s.ServiceToken == "" {
		return fmt.Errorf("service_token is required")
	}

	if s.ListenAddr == "" {
		return fmt.Errorf("listen_addr is required")
	}

	return nil
}

// ServerOptions represents the configuration for the WebSocket tunnel server.
type ServerOptions struct {
	// ListenAddr is the address to bind the server (e.g., ":8080" or "0.0.0.0:8080")
	ListenAddr string `yaml:"listen_addr"`

	// ServicesDir is the directory containing service definition files
	ServicesDir string `yaml:"services_dir"`

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

	if o.ServicesDir == "" {
		return fmt.Errorf("services_dir is required")
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

	// ServiceName is the name of the service to expose (required)
	ServiceName string `yaml:"service_name"`

	// ServiceToken is the authentication token for the service (required)
	ServiceToken string `yaml:"service_token"`

	// TargetAddr is the local service address to tunnel (required)
	TargetAddr string `yaml:"target_addr"`

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

	// Validate server URL format and scheme
	parsedURL, err := url.Parse(o.ServerURL)
	if err != nil {
		return fmt.Errorf("server_url is invalid: %w", err)
	}
	if parsedURL.Scheme != "ws" && parsedURL.Scheme != "wss" {
		return fmt.Errorf("server_url must use ws:// or wss:// scheme, got: %s://", parsedURL.Scheme)
	}

	if o.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}

	// Trim whitespace and validate service name is not empty after trimming
	trimmedName := strings.TrimSpace(o.ServiceName)
	if trimmedName == "" {
		return fmt.Errorf("service_name cannot be empty or whitespace-only")
	}

	// Validate service name length
	if len(o.ServiceName) > 255 {
		return fmt.Errorf("service_name too long (max 255 characters, got %d)", len(o.ServiceName))
	}

	// Validate service name format (alphanumeric, hyphens, underscores only)
	// Cannot start or end with hyphen
	if strings.HasPrefix(o.ServiceName, "-") || strings.HasSuffix(o.ServiceName, "-") {
		return fmt.Errorf("service_name cannot start or end with hyphen")
	}
	for _, r := range o.ServiceName {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("service_name must contain only alphanumeric characters, hyphens, and underscores")
		}
	}

	if o.ServiceToken == "" {
		return fmt.Errorf("service_token is required")
	}

	if o.TargetAddr == "" {
		return fmt.Errorf("target_addr is required")
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

// LoadServiceDefinition loads and validates a ServiceDefinition from a YAML file.
func LoadServiceDefinition(path string) (*ServiceDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read service definition file: %w", err)
	}

	var service ServiceDefinition
	if err := yaml.Unmarshal(data, &service); err != nil {
		return nil, fmt.Errorf("failed to parse service YAML: %w", err)
	}

	if err := service.Validate(); err != nil {
		return nil, fmt.Errorf("service definition validation failed: %w", err)
	}

	return &service, nil
}

// LoadAllServices loads all service definitions from a directory.
// Returns a map of service name (derived from filename) to ServiceDefinition.
// Files must have .yaml or .yml extension.
func LoadAllServices(servicesDir string) (map[string]*ServiceDefinition, error) {
	// Check if directory exists
	dirInfo, err := os.Stat(servicesDir)
	if err != nil {
		return nil, fmt.Errorf("services directory not found: %w", err)
	}
	if !dirInfo.IsDir() {
		return nil, fmt.Errorf("services_dir is not a directory: %s", servicesDir)
	}

	// Read directory entries
	entries, err := os.ReadDir(servicesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read services directory: %w", err)
	}

	services := make(map[string]*ServiceDefinition)
	var loadErrors []string

	for _, entry := range entries {
		// Skip directories and non-YAML files
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Check for .yaml or .yml extension
		if !(len(name) > 5 && name[len(name)-5:] == ".yaml") &&
			!(len(name) > 4 && name[len(name)-4:] == ".yml") {
			continue
		}

		// Derive service name from filename
		var serviceName string
		if len(name) > 5 && name[len(name)-5:] == ".yaml" {
			serviceName = name[:len(name)-5]
		} else {
			serviceName = name[:len(name)-4]
		}

		// Load service definition using safe path construction
		filePath := filepath.Join(servicesDir, name)
		service, err := LoadServiceDefinition(filePath)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %v", name, err))
			continue
		}

		services[serviceName] = service
	}

	// Report any load errors
	if len(loadErrors) > 0 {
		return services, fmt.Errorf("some service definitions failed to load: %v", loadErrors)
	}

	return services, nil
}
