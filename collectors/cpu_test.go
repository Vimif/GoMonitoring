package collectors

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCPUUsage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "idle based calculation",
			input:    "Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.7 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
			expected: 7.3, // 100 - 92.7
		},
		{
			name:     "high usage",
			input:    "Cpu(s): 45.3 us, 12.7 sy,  0.0 ni, 42.0 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
			expected: 58.0, // 100 - 42.0
		},
		{
			name:     "no idle value - sum us+sy+ni",
			input:    "Cpu(s): 10.5 us,  5.2 sy,  1.3 ni",
			expected: 17.0, // 10.5 + 5.2 + 1.3
		},
		{
			name:     "zero usage",
			input:    "Cpu(s):  0.0 us,  0.0 sy,  0.0 ni, 100.0 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
			expected: 0.0,
		},
		{
			name:     "full usage",
			input:    "Cpu(s): 100.0 us,  0.0 sy,  0.0 ni,  0.0 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
			expected: 100.0,
		},
		{
			name:     "decimal precision",
			input:    "Cpu(s):  3.14 us,  2.71 sy,  0.0 ni, 94.15 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
			expected: 5.85, // 100 - 94.15
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0.0,
		},
		{
			name:     "invalid format",
			input:    "invalid cpu output",
			expected: 0.0,
		},
		{
			name:     "no decimal idle",
			input:    "Cpu(s):  5 us,  2 sy,  0 ni, 93 id,  0 wa,  0 hi,  0 si,  0 st",
			expected: 7.0, // 100 - 93
		},
		{
			name:     "french locale format",
			input:    "Cpu(s) :  5,2%us,  2,1%sy,  0,0%ni, 92,7%id,  0,0%wa,  0,0%hi,  0,0%si,  0,0%st",
			expected: 7.3,
		},
		{
			name:     "with wait and interrupt",
			input:    "Cpu(s): 10.0 us,  5.0 sy,  1.0 ni, 80.0 id,  2.0 wa,  1.0 hi,  1.0 si,  0.0 st",
			expected: 20.0, // 100 - 80
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCPUUsage(tt.input)
			assert.InDelta(t, tt.expected, result, 0.1, "CPU usage should match expected value")
		})
	}
}

func TestParseCPUUsage_EdgeCases(t *testing.T) {
	t.Run("very long string", func(t *testing.T) {
		input := "Cpu(s):  1.0 us,  1.0 sy,  0.0 ni, 98.0 id" + strings.Repeat(" ", 1000)
		result := parseCPUUsage(input)
		assert.InDelta(t, 2.0, result, 0.1)
	})

	t.Run("negative values should be parsed", func(t *testing.T) {
		// En thÃ©orie impossible mais testons la robustesse
		input := "Cpu(s): -5.0 us,  2.0 sy,  0.0 ni, 103.0 id"
		result := parseCPUUsage(input)
		// Le rÃ©sultat dÃ©pend de la regex - elle devrait matcher les nombres positifs
		assert.GreaterOrEqual(t, result, 0.0, "CPU usage should not be negative")
	})

	t.Run("special characters in string", func(t *testing.T) {
		input := "Cpu(s): 5.0 us, <script>alert('xss')</script> 2.0 sy, 0.0 ni, 93.0 id"
		result := parseCPUUsage(input)
		// La regex devrait extraire les nombres valides
		assert.GreaterOrEqual(t, result, 0.0)
	})
}

// Benchmark tests
func BenchmarkParseCPUUsage(b *testing.B) {
	input := "Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.7 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseCPUUsage(input)
	}
}

func BenchmarkParseCPUUsage_Fallback(b *testing.B) {
	// Test du fallback (pas de valeur idle)
	input := "Cpu(s): 10.5 us,  5.2 sy,  1.3 ni"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseCPUUsage(input)
	}
}

// Tests pour les fonctions d'extraction de donnÃ©es
// Note: Les tests de CollectCPUInfo nÃ©cessitent un mock SSH client
// qui sera crÃ©Ã© dans ssh/mock_client.go (Sprint 2.3)

func TestParseCPUUsage_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
		desc     string
	}{
		{
			name:     "Ubuntu 20.04 top output",
			input:    "%Cpu(s):  3.8 us,  1.2 sy,  0.0 ni, 94.9 id,  0.1 wa,  0.0 hi,  0.0 si,  0.0 st",
			expected: 5.1,
			desc:     "Standard Ubuntu format with % prefix",
		},
		{
			name:     "CentOS 7 top output",
			input:    "Cpu(s):  7.1%us,  3.2%sy,  0.0%ni, 89.5%id,  0.2%wa,  0.0%hi,  0.0%si,  0.0%st",
			expected: 10.5,
			desc:     "CentOS format with % suffix",
		},
		{
			name:     "Debian top output",
			input:    "Cpu(s):  2.3 us,  1.1 sy,  0.0 ni, 96.6 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
			expected: 3.4,
			desc:     "Standard Debian format",
		},
		{
			name:     "Heavy load server",
			input:    "Cpu(s): 78.5 us, 15.2 sy,  0.0 ni,  5.8 id,  0.5 wa,  0.0 hi,  0.0 si,  0.0 st",
			expected: 94.2,
			desc:     "Server under heavy load",
		},
		{
			name:     "Idle server",
			input:    "Cpu(s):  0.3 us,  0.1 sy,  0.0 ni, 99.6 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
			expected: 0.4,
			desc:     "Nearly idle server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCPUUsage(tt.input)
			assert.InDelta(t, tt.expected, result, 0.1, tt.desc)
		})
	}
}

// Tests de validation des valeurs retournÃ©es
func TestParseCPUUsage_ValidRange(t *testing.T) {
	inputs := []string{
		"Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.7 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
		"Cpu(s): 45.3 us, 12.7 sy,  0.0 ni, 42.0 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
		"Cpu(s):  0.0 us,  0.0 sy,  0.0 ni, 100.0 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
		"Cpu(s): 100.0 us,  0.0 sy,  0.0 ni,  0.0 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			result := parseCPUUsage(input)
			assert.GreaterOrEqual(t, result, 0.0, "CPU usage should be >= 0")
			assert.LessOrEqual(t, result, 100.0, "CPU usage should be <= 100")
		})
	}
}
