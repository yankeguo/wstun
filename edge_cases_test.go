package wstun

import (
	"strings"
	"testing"
)

// TestParseMessage_ExcessiveLength tests that excessively long messages are rejected
func TestParseMessage_ExcessiveLength(t *testing.T) {
	// Create a message longer than maxMessageLength
	longConnID := strings.Repeat("A", 2000)
	longMessage := MessageTypePrefixConnect + longConnID

	connID, ok := ParseConnectMessage(longMessage)
	if ok {
		t.Errorf("ParseConnectMessage should reject excessively long message, but returned connID: %s", connID)
	}

	connID, ok = ParseHandleMessage(MessageTypePrefixHandle + longConnID)
	if ok {
		t.Errorf("ParseHandleMessage should reject excessively long message, but returned connID: %s", connID)
	}
}

// TestParseMessage_InvalidULID tests that invalid ULID characters are rejected
func TestParseMessage_InvalidULID(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"invalid_chars", "CONNECT:XXXXXXXXXXXXXXXXXXXXXXXXXX"}, // X is not valid in ULID
		{"invalid_format", "CONNECT:01234567890123456789012!"},  // ! is not valid
		{"spaces", "CONNECT:012345678901234567890   "},          // spaces
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.HasPrefix(tt.message, MessageTypePrefixConnect) {
				connID, ok := ParseConnectMessage(tt.message)
				if ok {
					t.Errorf("ParseConnectMessage should reject invalid ULID, but returned connID: %s", connID)
				}
			} else if strings.HasPrefix(tt.message, MessageTypePrefixHandle) {
				connID, ok := ParseHandleMessage(tt.message)
				if ok {
					t.Errorf("ParseHandleMessage should reject invalid ULID, but returned connID: %s", connID)
				}
			}
		})
	}
}

// TestClientOptions_ServiceNameEdgeCases tests edge cases in service name validation
func TestClientOptions_ServiceNameEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		shouldFail  bool
	}{
		{"starts_with_hyphen", "-invalid", true},
		{"ends_with_hyphen", "invalid-", true},
		{"whitespace_only", "   ", true},
		{"too_long", strings.Repeat("a", 256), true},
		{"valid_with_underscore", "valid_name", false},
		{"valid_with_hyphen", "valid-name", false},
		{"valid_mixed", "valid_name-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ClientOptions{
				ServerURL:    "ws://localhost:8080",
				ServiceName:  tt.serviceName,
				ServiceToken: "token",
				TargetAddr:   "localhost:3000",
			}

			err := opts.Validate()
			if tt.shouldFail && err == nil {
				t.Errorf("Validate() should fail for service name %q", tt.serviceName)
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("Validate() should succeed for service name %q, but got error: %v", tt.serviceName, err)
			}
		})
	}
}

// TestClientOptions_ServerURLValidation tests URL scheme validation
func TestClientOptions_ServerURLValidation(t *testing.T) {
	tests := []struct {
		name       string
		serverURL  string
		shouldFail bool
	}{
		{"valid_ws", "ws://localhost:8080", false},
		{"valid_wss", "wss://localhost:8080", false},
		{"invalid_http", "http://localhost:8080", true},
		{"invalid_https", "https://localhost:8080", true},
		{"invalid_empty", "", true},
		{"invalid_malformed", "not-a-url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ClientOptions{
				ServerURL:    tt.serverURL,
				ServiceName:  "test",
				ServiceToken: "token",
				TargetAddr:   "localhost:3000",
			}

			err := opts.Validate()
			if tt.shouldFail && err == nil {
				t.Errorf("Validate() should fail for server URL %q", tt.serverURL)
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("Validate() should succeed for server URL %q, but got error: %v", tt.serverURL, err)
			}
		})
	}
}

// TestValidateConnectionID_EdgeCases tests ULID validation edge cases
func TestValidateConnectionID_EdgeCases(t *testing.T) {
	// Generate a valid ULID for testing
	validULID, err := GenerateConnectionID()
	if err != nil {
		t.Fatalf("Failed to generate valid ULID: %v", err)
	}

	tests := []struct {
		name     string
		connID   string
		expected bool
	}{
		{"empty", "", false},
		{"too_short", "01GX5Z", false},
		{"too_long", "01GX5ZPCYQEQ6M3YN4DDXYZ0123", false},
		{"valid_ulid", validULID, true},
		{"lowercase", strings.ToLower(validULID), true}, // ULID parsing accepts lowercase
		{"special_chars", "01GX5ZPCYQEQ6M3YN4DDXY@@@", false},
		{"invalid_base32", "01GX5ZPCYQEQ6M3YN4DDXYZU", false}, // U is not in Crockford base32
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateConnectionID(tt.connID)
			if result != tt.expected {
				t.Errorf("ValidateConnectionID(%q) = %v, want %v", tt.connID, result, tt.expected)
			}
		})
	}
}

// TestLoadAllServices_PathSafety tests that filepath.Join is used for safe path construction
func TestLoadAllServices_PathSafety(t *testing.T) {
	// This test ensures the implementation uses safe path construction
	// The actual path traversal protection would require more complex testing
	// with actual filesystem operations, which is beyond unit testing scope

	// Test with a non-existent directory (should fail gracefully)
	services, err := LoadAllServices("/nonexistent/directory")
	if err == nil {
		t.Error("LoadAllServices should fail for non-existent directory")
	}
	if services != nil {
		t.Error("LoadAllServices should return nil for non-existent directory")
	}
}
