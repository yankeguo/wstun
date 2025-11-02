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
				ListenAddr: ":8080",
				LogLevel:   "info",
			},
			wantErr: false,
		},
		{
			name: "missing listen_addr",
			opts: ServerOptions{
				LogLevel: "info",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			opts: ServerOptions{
				ListenAddr: ":8080",
				LogLevel:   "invalid",
			},
			wantErr: true,
		},
		{
			name: "negative max connections",
			opts: ServerOptions{
				ListenAddr:     ":8080",
				MaxConnections: -1,
			},
			wantErr: true,
		},
		{
			name: "tls_cert without tls_key",
			opts: ServerOptions{
				ListenAddr: ":8080",
				TLSCert:    "cert.pem",
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
		ListenAddr: ":8080",
	}
	opts.SetDefaults()

	if opts.LogLevel != "info" {
		t.Errorf("expected default log_level to be 'info', got '%s'", opts.LogLevel)
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
				ServerURL: "ws://localhost:8080",
				LocalAddr: ":9090",
				LogLevel:  "info",
			},
			wantErr: false,
		},
		{
			name: "missing server_url",
			opts: ClientOptions{
				LocalAddr: ":9090",
			},
			wantErr: true,
		},
		{
			name: "missing local_addr",
			opts: ClientOptions{
				ServerURL: "ws://localhost:8080",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			opts: ClientOptions{
				ServerURL: "ws://localhost:8080",
				LocalAddr: ":9090",
				LogLevel:  "invalid",
			},
			wantErr: true,
		},
		{
			name: "negative reconnect interval",
			opts: ClientOptions{
				ServerURL:         "ws://localhost:8080",
				LocalAddr:         ":9090",
				ReconnectInterval: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative max reconnect attempts",
			opts: ClientOptions{
				ServerURL:            "ws://localhost:8080",
				LocalAddr:            ":9090",
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
		ServerURL: "ws://localhost:8080",
		LocalAddr: ":9090",
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
			name: "missing required field",
			content: `log_level: "info"
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
local_addr: ":9090"
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
			name: "missing required field",
			content: `server_url: "ws://localhost:8080"
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

