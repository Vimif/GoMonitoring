package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckOrigin(t *testing.T) {
	// Test case 1: Localhost (Current whitelist, should pass)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Origin", "http://localhost:8080")
	if !upgrader.CheckOrigin(req) {
		t.Error("Blocked localhost origin (should be allowed)")
	}

	// Test case 2: 127.0.0.1 (Current whitelist, should pass)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "127.0.0.1:8080"
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	if !upgrader.CheckOrigin(req) {
		t.Error("Blocked 127.0.0.1 origin (should be allowed)")
	}

	// Test case 3: Custom hostname (e.g. valid deployment, should pass)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "monitoring.company.internal"
	req.Header.Set("Origin", "http://monitoring.company.internal")
	if !upgrader.CheckOrigin(req) {
		t.Error("Blocked custom hostname origin (FIXME: currently fails due to whitelist)")
	}

	// Test case 4: Evil origin (should fail)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Origin", "http://evil.com")
	if upgrader.CheckOrigin(req) {
		t.Error("Allowed evil origin (Security Risk!)")
	}

	// Test case 5: No origin (should pass)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Del("Origin")
	if !upgrader.CheckOrigin(req) {
		t.Error("Blocked request with no origin")
	}
}
