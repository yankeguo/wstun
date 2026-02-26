package wstun

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestServerOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    ServerOptions
		wantErr bool
	}{
		{
			name: "valid minimal config",
			opts: ServerOptions{
				ListenAddr:  ":8080",
				ServicesDir: "./services",
				LogLevel:    "info",
			},
			wantErr: false,
		},
		{
			name: "missing listen_addr",
			opts: ServerOptions{
				ServicesDir: "./services",
				LogLevel:    "info",
			},
			wantErr: true,
		},
		{
			name: "missing services_dir",
			opts: ServerOptions{
				ListenAddr: ":8080",
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			opts: ServerOptions{
				ListenAddr:  ":8080",
				ServicesDir: "./services",
				LogLevel:    "invalid",
			},
			wantErr: true,
		},
		{
			name: "negative max connections",
			opts: ServerOptions{
				ListenAddr:     ":8080",
				ServicesDir:    "./services",
				MaxConnections: -1,
			},
			wantErr: true,
		},
		{
			name: "negative retry_attempts",
			opts: ServerOptions{
				ListenAddr:    ":8080",
				ServicesDir:   "./services",
				RetryAttempts: -1,
			},
			wantErr: true,
		},
		{
			name: "negative retry_interval",
			opts: ServerOptions{
				ListenAddr:    ":8080",
				ServicesDir:   "./services",
				RetryInterval: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative retry_max_interval",
			opts: ServerOptions{
				ListenAddr:       ":8080",
				ServicesDir:      "./services",
				RetryMaxInterval: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "tls_cert without tls_key",
			opts: ServerOptions{
				ListenAddr:  ":8080",
				ServicesDir: "./services",
				TLSCert:     "cert.pem",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerOptions_SetDefaults(t *testing.T) {
	opts := ServerOptions{
		ListenAddr:  ":8080",
		ServicesDir: "./services",
	}
	opts.SetDefaults()

	if opts.LogLevel != "info" {
		t.Errorf("expected default log_level to be 'info', got '%s'", opts.LogLevel)
	}
	if opts.RetryInterval != 5*time.Second {
		t.Errorf("expected default retry_interval to be 5s, got %v", opts.RetryInterval)
	}
	if opts.RetryMaxInterval != 60*time.Second {
		t.Errorf("expected default retry_max_interval to be 60s, got %v", opts.RetryMaxInterval)
	}
}

func TestClientOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    ClientOptions
		wantErr bool
	}{
		{
			name: "valid minimal config",
			opts: ClientOptions{
				ServerURL:    "ws://localhost:8080",
				ServiceName:  "my-service",
				ServiceToken: "secret-token",
				TargetAddr:   "127.0.0.1:3000",
				LogLevel:     "info",
			},
			wantErr: false,
		},
		{
			name: "missing server_url",
			opts: ClientOptions{
				ServiceName:  "my-service",
				ServiceToken: "secret-token",
				TargetAddr:   "127.0.0.1:3000",
			},
			wantErr: true,
		},
		{
			name: "missing service_name",
			opts: ClientOptions{
				ServerURL:    "ws://localhost:8080",
				ServiceToken: "secret-token",
				TargetAddr:   "127.0.0.1:3000",
			},
			wantErr: true,
		},
		{
			name: "missing service_token",
			opts: ClientOptions{
				ServerURL:   "ws://localhost:8080",
				ServiceName: "my-service",
				TargetAddr:  "127.0.0.1:3000",
			},
			wantErr: true,
		},
		{
			name: "missing target_addr",
			opts: ClientOptions{
				ServerURL:    "ws://localhost:8080",
				ServiceName:  "my-service",
				ServiceToken: "secret-token",
			},
			wantErr: true,
		},
		{
			name: "invalid service_name format",
			opts: ClientOptions{
				ServerURL:    "ws://localhost:8080",
				ServiceName:  "my service!", // spaces and special chars not allowed
				ServiceToken: "secret-token",
				TargetAddr:   "127.0.0.1:3000",
			},
			wantErr: true,
		},
		{
			name: "valid service_name with hyphens and underscores",
			opts: ClientOptions{
				ServerURL:    "ws://localhost:8080",
				ServiceName:  "my-service_123",
				ServiceToken: "secret-token",
				TargetAddr:   "127.0.0.1:3000",
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			opts: ClientOptions{
				ServerURL:    "ws://localhost:8080",
				ServiceName:  "my-service",
				ServiceToken: "secret-token",
				TargetAddr:   "127.0.0.1:3000",
				LogLevel:     "invalid",
			},
			wantErr: true,
		},
		{
			name: "negative reconnect interval",
			opts: ClientOptions{
				ServerURL:         "ws://localhost:8080",
				ServiceName:       "my-service",
				ServiceToken:      "secret-token",
				TargetAddr:        "127.0.0.1:3000",
				ReconnectInterval: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative max reconnect attempts",
			opts: ClientOptions{
				ServerURL:            "ws://localhost:8080",
				ServiceName:          "my-service",
				ServiceToken:         "secret-token",
				TargetAddr:           "127.0.0.1:3000",
				MaxReconnectAttempts: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientOptions_SetDefaults(t *testing.T) {
	opts := ClientOptions{
		ServerURL:    "ws://localhost:8080",
		ServiceName:  "my-service",
		ServiceToken: "secret-token",
		TargetAddr:   "127.0.0.1:3000",
	}
	opts.SetDefaults()

	if opts.LogLevel != "info" {
		t.Errorf("expected default log_level to be 'info', got '%s'", opts.LogLevel)
	}
	if opts.ReconnectInterval != 5*time.Second {
		t.Errorf("expected default reconnect_interval to be 5s, got %v", opts.ReconnectInterval)
	}
}

func TestLoadServerConfig(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid config",
			content: `listen_addr: ":8080"
services_dir: "./services"
log_level: "info"
max_connections: 100
`,
			wantErr: false,
		},
		{
			name: "invalid yaml",
			content: `listen_addr: ":8080"
invalid yaml content
`,
			wantErr: true,
		},
		{
			name: "missing listen_addr",
			content: `services_dir: "./services"
log_level: "info"
`,
			wantErr: true,
		},
		{
			name: "missing services_dir",
			content: `listen_addr: ":8080"
log_level: "info"
`,
			wantErr: true,
		},
		{
			name: "valid config with retry settings",
			content: `listen_addr: ":8080"
services_dir: "./services"
log_level: "info"
retry_attempts: 3
retry_interval: 5s
retry_max_interval: 60s
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tt.name+".yaml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			_, err := LoadServerConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadServerConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Test missing file
	t.Run("missing file", func(t *testing.T) {
		_, err := LoadServerConfig("/nonexistent/config.yaml")
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})
}

func TestLoadClientConfig(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid config",
			content: `server_url: "ws://localhost:8080"
service_name: "my-service"
service_token: "secret-token"
target_addr: "127.0.0.1:3000"
log_level: "info"
reconnect_interval: 10s
max_reconnect_attempts: 5
`,
			wantErr: false,
		},
		{
			name: "invalid yaml",
			content: `server_url: "ws://localhost:8080"
invalid yaml content
`,
			wantErr: true,
		},
		{
			name: "missing service_name",
			content: `server_url: "ws://localhost:8080"
service_token: "secret-token"
target_addr: "127.0.0.1:3000"
`,
			wantErr: true,
		},
		{
			name: "missing service_token",
			content: `server_url: "ws://localhost:8080"
service_name: "my-service"
target_addr: "127.0.0.1:3000"
`,
			wantErr: true,
		},
		{
			name: "missing target_addr",
			content: `server_url: "ws://localhost:8080"
service_name: "my-service"
service_token: "secret-token"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tt.name+".yaml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			_, err := LoadClientConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadClientConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Test missing file
	t.Run("missing file", func(t *testing.T) {
		_, err := LoadClientConfig("/nonexistent/config.yaml")
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})
}

func TestServiceDefinition_Validate(t *testing.T) {
	tests := []struct {
		name    string
		service ServiceDefinition
		wantErr bool
	}{
		{
			name: "valid service definition",
			service: ServiceDefinition{
				ServiceToken: "secret-token",
				ListenAddr:   ":9090",
			},
			wantErr: false,
		},
		{
			name: "missing service_token",
			service: ServiceDefinition{
				ListenAddr: ":9090",
			},
			wantErr: true,
		},
		{
			name: "missing listen_addr",
			service: ServiceDefinition{
				ServiceToken: "secret-token",
			},
			wantErr: true,
		},
		{
			name:    "empty service definition",
			service: ServiceDefinition{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.service.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadServiceDefinition(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid service definition",
			content: `service_token: "secret-token"
listen_addr: ":9090"
`,
			wantErr: false,
		},
		{
			name: "invalid yaml",
			content: `service_token: "secret-token"
invalid yaml content
`,
			wantErr: true,
		},
		{
			name: "missing service_token",
			content: `listen_addr: ":9090"
`,
			wantErr: true,
		},
		{
			name: "missing listen_addr",
			content: `service_token: "secret-token"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			servicePath := filepath.Join(tmpDir, tt.name+".yaml")
			if err := os.WriteFile(servicePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test service: %v", err)
			}

			_, err := LoadServiceDefinition(servicePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadServiceDefinition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Test missing file
	t.Run("missing file", func(t *testing.T) {
		_, err := LoadServiceDefinition("/nonexistent/service.yaml")
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})
}

func TestLoadAllServices(t *testing.T) {
	tmpDir := t.TempDir()

	// Create services directory
	servicesDir := filepath.Join(tmpDir, "services")
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		t.Fatalf("failed to create services directory: %v", err)
	}

	// Create valid service files
	validServices := map[string]string{
		"service1.yaml": `service_token: "token1"
listen_addr: ":9091"
`,
		"service2.yaml": `service_token: "token2"
listen_addr: ":9092"
`,
		"service-with-dash.yaml": `service_token: "token3"
listen_addr: ":9093"
`,
	}

	for name, content := range validServices {
		path := filepath.Join(servicesDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write service file %s: %v", name, err)
		}
	}

	// Create an invalid service file
	invalidPath := filepath.Join(servicesDir, "invalid.yaml")
	if err := os.WriteFile(invalidPath, []byte("invalid yaml content"), 0644); err != nil {
		t.Fatalf("failed to write invalid service file: %v", err)
	}

	// Create a non-YAML file (should be ignored)
	txtPath := filepath.Join(servicesDir, "readme.txt")
	if err := os.WriteFile(txtPath, []byte("This is a readme"), 0644); err != nil {
		t.Fatalf("failed to write txt file: %v", err)
	}

	// Test loading all services
	t.Run("load valid services", func(t *testing.T) {
		services, err := LoadAllServices(servicesDir)
		if err == nil {
			t.Error("expected error due to invalid service file, got nil")
		}
		// Should still load valid services
		if len(services) != 3 {
			t.Errorf("expected 3 valid services, got %d", len(services))
		}

		// Verify service names derived from filenames
		expectedNames := []string{"service1", "service2", "service-with-dash"}
		for _, name := range expectedNames {
			if _, ok := services[name]; !ok {
				t.Errorf("expected service %s to be loaded", name)
			}
		}

		// Verify service content
		if svc, ok := services["service1"]; ok {
			if svc.ServiceToken != "token1" {
				t.Errorf("expected service1 token to be 'token1', got '%s'", svc.ServiceToken)
			}
			if svc.ListenAddr != ":9091" {
				t.Errorf("expected service1 listen_addr to be ':9091', got '%s'", svc.ListenAddr)
			}
		}
	})

	// Test empty directory
	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		if err := os.MkdirAll(emptyDir, 0755); err != nil {
			t.Fatalf("failed to create empty directory: %v", err)
		}

		services, err := LoadAllServices(emptyDir)
		if err != nil {
			t.Errorf("expected no error for empty directory, got %v", err)
		}
		if len(services) != 0 {
			t.Errorf("expected 0 services from empty directory, got %d", len(services))
		}
	})

	// Test non-existent directory
	t.Run("non-existent directory", func(t *testing.T) {
		_, err := LoadAllServices("/nonexistent/directory")
		if err == nil {
			t.Error("expected error for non-existent directory, got nil")
		}
	})

	// Test file instead of directory
	t.Run("file instead of directory", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "notadir.txt")
		if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := LoadAllServices(filePath)
		if err == nil {
			t.Error("expected error when path is a file not directory, got nil")
		}
	})
}
