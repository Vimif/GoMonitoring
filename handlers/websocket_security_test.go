package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestWebSocketOriginSecurity(t *testing.T) {
	// Save original check function to restore later
	originalCheck := upgrader.CheckOrigin
	defer func() { upgrader.CheckOrigin = originalCheck }()

	// Test cases
	tests := []struct {
		name           string
		requestHost    string
		requestOrigin  string
		expectedStatus int // 0 means upgrade success (hijacked), otherwise HTTP status
		expectAllowed  bool
	}{
		{
			name:           "Valid Localhost Origin",
			requestHost:    "localhost:8080",
			requestOrigin:  "http://localhost:8080",
			expectAllowed:  true,
		},
		{
			name:           "Valid IP Origin",
			requestHost:    "127.0.0.1:8080",
			requestOrigin:  "http://127.0.0.1:8080",
			expectAllowed:  true,
		},
		{
			name:           "Valid Domain Origin (Production Scenario)",
			requestHost:    "my-app.example.com",
			requestOrigin:  "https://my-app.example.com",
			expectAllowed:  true, // Currently FAILS with hardcoded list
		},
		{
			name:           "Invalid Origin (CSWSH Attack)",
			requestHost:    "localhost:8080",
			requestOrigin:  "http://evil.com",
			expectAllowed:  false,
		},
		{
			name:           "Direct Request (No Origin)",
			requestHost:    "localhost:8080",
			requestOrigin:  "",
			expectAllowed:  true, // Non-browser clients allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			req.Host = tt.requestHost
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}

			allowed := upgrader.CheckOrigin(req)

			if tt.expectAllowed && !allowed {
				t.Errorf("Expected allowed, got denied. Host: %s, Origin: %s", tt.requestHost, tt.requestOrigin)
			}
			if !tt.expectAllowed && allowed {
				t.Errorf("Expected denied, got allowed. Host: %s, Origin: %s", tt.requestHost, tt.requestOrigin)
			}
		})
	}
}
