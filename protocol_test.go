package wstun

import (
	"strings"
	"testing"
)

// Test protocol constants
func TestProtocolConstants(t *testing.T) {
	if MessageTypeWaiting != "WAITING" {
		t.Errorf("MessageTypeWaiting = %s, want WAITING", MessageTypeWaiting)
	}
	if MessageTypePrefixConnect != "CONNECT:" {
		t.Errorf("MessageTypePrefixConnect = %s, want CONNECT:", MessageTypePrefixConnect)
	}
	if MessageTypePrefixHandle != "HANDLE:" {
		t.Errorf("MessageTypePrefixHandle = %s, want HANDLE:", MessageTypePrefixHandle)
	}
}

// Test ULID generation
func TestGenerateConnectionID(t *testing.T) {
	id1, err := GenerateConnectionID()
	if err != nil {
		t.Fatalf("GenerateConnectionID() error = %v", err)
	}
	if len(id1) != 26 {
		t.Errorf("GenerateConnectionID() length = %d, want 26", len(id1))
	}

	id2, err := GenerateConnectionID()
	if err != nil {
		t.Fatalf("GenerateConnectionID() error = %v", err)
	}
	if id1 == id2 {
		t.Errorf("GenerateConnectionID() generated duplicate IDs")
	}
}

// Test message formatting
func TestFormatMessages(t *testing.T) {
	connID := "01234567890123456789012345"

	connect := FormatConnectMessage(connID)
	expected := "CONNECT:01234567890123456789012345"
	if connect != expected {
		t.Errorf("FormatConnectMessage() = %s, want %s", connect, expected)
	}

	handle := FormatHandleMessage(connID)
	expected = "HANDLE:01234567890123456789012345"
	if handle != expected {
		t.Errorf("FormatHandleMessage() = %s, want %s", handle, expected)
	}
}

// Test message parsing
func TestParseMessages(t *testing.T) {
	tests := []struct {
		name    string
		message string
		parser  func(string) (string, bool)
		wantID  string
		wantOK  bool
	}{
		{
			name:    "valid CONNECT",
			message: "CONNECT:01234567890123456789012345",
			parser:  ParseConnectMessage,
			wantID:  "01234567890123456789012345",
			wantOK:  true,
		},
		{
			name:    "valid HANDLE",
			message: "HANDLE:01234567890123456789012345",
			parser:  ParseHandleMessage,
			wantID:  "01234567890123456789012345",
			wantOK:  true,
		},
		{
			name:    "invalid prefix",
			message: "INVALID:01234567890123456789012345",
			parser:  ParseConnectMessage,
			wantID:  "",
			wantOK:  false,
		},
		{
			name:    "short ID",
			message: "CONNECT:123",
			parser:  ParseConnectMessage,
			wantID:  "",
			wantOK:  false,
		},
		{
			name:    "long ID",
			message: "CONNECT:012345678901234567890123456",
			parser:  ParseConnectMessage,
			wantID:  "",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := tt.parser(tt.message)
			if gotID != tt.wantID || gotOK != tt.wantOK {
				t.Errorf("parser(%s) = (%s, %v), want (%s, %v)",
					tt.message, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

// Test connection ID validation
func TestValidateConnectionID(t *testing.T) {
	validID, err := GenerateConnectionID()
	if err != nil {
		t.Fatalf("GenerateConnectionID() error = %v", err)
	}

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "valid ULID",
			id:   validID,
			want: true,
		},
		{
			name: "invalid length",
			id:   "123",
			want: false,
		},
		{
			name: "invalid format",
			id:   "!!!!!!!!!!!!!!!!!!!!!!!!",
			want: false,
		},
		{
			name: "empty string",
			id:   "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateConnectionID(tt.id)
			if got != tt.want {
				t.Errorf("ValidateConnectionID(%s) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

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
