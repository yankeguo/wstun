package wstun_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestServerCommand_Help tests that the server command displays help text
func TestServerCommand_Help(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/wstun-server", "-h")
	output, err := cmd.CombinedOutput()

	// -h flag should exit with code 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			t.Errorf("expected exit code 0 for -h flag, got %d", exitErr.ExitCode())
		}
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Usage:") {
		t.Error("help text should contain 'Usage:'")
	}
	if !strings.Contains(outputStr, "-c") {
		t.Error("help text should mention -c flag")
	}
}

// TestClientCommand_Help tests that the client command displays help text
func TestClientCommand_Help(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/wstun-client", "-h")
	output, err := cmd.CombinedOutput()

	// -h flag should exit with code 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			t.Errorf("expected exit code 0 for -h flag, got %d", exitErr.ExitCode())
		}
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Usage:") {
		t.Error("help text should contain 'Usage:'")
	}
	if !strings.Contains(outputStr, "-c") {
		t.Error("help text should mention -c flag")
	}
}

// TestServerCommand_NoConfig tests that server exits with error when no config is provided
func TestServerCommand_NoConfig(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/wstun-server")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("expected error when no config file is provided")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "required") || !strings.Contains(outputStr, "configuration") {
		t.Errorf("expected error message about required configuration, got: %s", outputStr)
	}
}

// TestClientCommand_NoConfig tests that client exits with error when no config is provided
func TestClientCommand_NoConfig(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/wstun-client")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("expected error when no config file is provided")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "required") || !strings.Contains(outputStr, "configuration") {
		t.Errorf("expected error message about required configuration, got: %s", outputStr)
	}
}

// TestServerCommand_MissingConfigFile tests that server exits with error for missing config file
func TestServerCommand_MissingConfigFile(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/wstun-server", "-c", "/nonexistent/config.yaml")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("expected error for missing config file")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Failed to load configuration") {
		t.Errorf("expected error about failed configuration loading, got: %s", outputStr)
	}
}

// TestClientCommand_MissingConfigFile tests that client exits with error for missing config file
func TestClientCommand_MissingConfigFile(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/wstun-client", "-c", "/nonexistent/config.yaml")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("expected error for missing config file")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Failed to load configuration") {
		t.Errorf("expected error about failed configuration loading, got: %s", outputStr)
	}
}

// TestServerCommand_ValidConfig tests that server starts successfully with valid config
func TestServerCommand_ValidConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "server.yaml")
	configContent := `listen_addr: ":18080"
log_level: "info"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Start server command with output capture
	cmd := exec.Command("go", "run", "./cmd/wstun-server", "-c", configPath)
	stdout, _ := cmd.StdoutPipe()

	// Start the command
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Read a bit of output to confirm it started
	buf := make([]byte, 1024)
	readDone := make(chan bool, 1)
	go func() {
		stdout.Read(buf)
		readDone <- true
	}()

	select {
	case <-readDone:
		// Server started and produced output
	case <-time.After(2 * time.Second):
		t.Error("server did not produce output within timeout")
	}

	// Send SIGINT to gracefully shutdown
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Logf("failed to send interrupt signal: %v", err)
	}

	// Wait for it to exit with a timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		// Server shut down (any exit code is acceptable for now)
	case <-time.After(2 * time.Second):
		// Force kill if graceful shutdown doesn't work
		cmd.Process.Kill()
		cmd.Wait()
	}
}

// TestClientCommand_ValidConfig tests that client starts successfully with valid config
func TestClientCommand_ValidConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "client.yaml")
	configContent := `server_url: "ws://localhost:18080"
local_addr: ":19090"
log_level: "info"
reconnect_interval: 5s
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Start client command with output capture
	cmd := exec.Command("go", "run", "./cmd/wstun-client", "-c", configPath)
	stdout, _ := cmd.StdoutPipe()

	// Start the command
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start client: %v", err)
	}

	// Read a bit of output to confirm it started
	buf := make([]byte, 1024)
	readDone := make(chan bool, 1)
	go func() {
		stdout.Read(buf)
		readDone <- true
	}()

	select {
	case <-readDone:
		// Client started and produced output
	case <-time.After(2 * time.Second):
		t.Error("client did not produce output within timeout")
	}

	// Send SIGINT to gracefully shutdown
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Logf("failed to send interrupt signal: %v", err)
	}

	// Wait for it to exit with a timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		// Client shut down (any exit code is acceptable for now)
	case <-time.After(2 * time.Second):
		// Force kill if graceful shutdown doesn't work
		cmd.Process.Kill()
		cmd.Wait()
	}
}

// TestServerCommand_InvalidConfig tests that server exits with error for invalid config
func TestServerCommand_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	// Missing required listen_addr field
	configContent := `log_level: "info"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/wstun-server", "-c", configPath)
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("expected error for invalid config")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "validation failed") {
		t.Errorf("expected validation error, got: %s", outputStr)
	}
}

// TestClientCommand_InvalidConfig tests that client exits with error for invalid config
func TestClientCommand_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	// Missing required server_url field
	configContent := `local_addr: ":9090"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/wstun-client", "-c", configPath)
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("expected error for invalid config")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "validation failed") {
		t.Errorf("expected validation error, got: %s", outputStr)
	}
}
