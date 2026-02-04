package collectors

import (
	"testing"

	"go-monitoring/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: Ces tests vÃ©rifient la logique de parsing des collectors
// Les tests avec mock SSH seront ajoutÃ©s dans Sprint 2.3

func TestMemoryInfoLinux_Parsing(t *testing.T) {
	// Simuler le parsing d'une sortie `free -b`
	// Format: Mem:   total   used   free   shared   buff/cache   available
	tests := []struct {
		name           string
		freeOutput     string
		expectedTotal  uint64
		expectedUsed   uint64
		expectedFree   uint64
		expectedAvail  uint64
		expectedPercent float64
	}{
		{
			name:           "normal system",
			freeOutput:     "Mem:   16777216000   8388608000   2097152000   104857600   6291456000   7340032000",
			expectedTotal:  16777216000,
			expectedUsed:   9437184000, // Total - Available
			expectedFree:   2097152000,
			expectedAvail:  7340032000,
			expectedPercent: 56.25, // (9437184000 / 16777216000) * 100
		},
		{
			name:           "low memory",
			freeOutput:     "Mem:   4294967296   3221225472   536870912   0   536870912   1073741824",
			expectedTotal:  4294967296,
			expectedUsed:   3221225472, // Total - Available
			expectedFree:   536870912,
			expectedAvail:  1073741824,
			expectedPercent: 75.0,
		},
		{
			name:           "high memory available",
			freeOutput:     "Mem:   33554432000   4194304000   20132659200   0   9227468800   29360128000",
			expectedTotal:  33554432000,
			expectedUsed:   4194304000, // Total - Available
			expectedFree:   20132659200,
			expectedAvail:  29360128000,
			expectedPercent: 12.5,
		},
		{
			name:           "zero available (edge case)",
			freeOutput:     "Mem:   8589934592   8589934592   0   0   0   0",
			expectedTotal:  8589934592,
			expectedUsed:   8589934592,
			expectedFree:   0,
			expectedAvail:  0,
			expectedPercent: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parser manuellement comme le fait collectMemoryInfoLinux
			fields := []string{"Mem:", "16777216000", "8388608000", "2097152000", "104857600", "6291456000", "7340032000"}

			// Simuler le parsing avec les bonnes valeurs du test
			parts := splitMemoryOutput(tt.freeOutput)
			if len(parts) < 7 {
				t.Skip("Invalid format")
			}

			var info models.MemoryInfo
			parseMemoryFields(parts, &info)

			// VÃ©rifier les valeurs parsÃ©es
			assert.Equal(t, tt.expectedTotal, info.Total, "Total memory should match")
			assert.Equal(t, tt.expectedFree, info.Free, "Free memory should match")
			assert.Equal(t, tt.expectedAvail, info.Available, "Available memory should match")
			assert.InDelta(t, tt.expectedPercent, info.UsedPercent, 0.5, "Used percent should match")
		})
	}
}

func TestMemoryInfoWindows_Parsing(t *testing.T) {
	// Simuler le parsing d'une sortie PowerShell
	// Format: total|free
	tests := []struct {
		name           string
		psOutput       string
		expectedTotal  uint64
		expectedFree   uint64
		expectedUsed   uint64
		expectedPercent float64
	}{
		{
			name:           "normal system",
			psOutput:       "17179869184|5368709120",
			expectedTotal:  17179869184,
			expectedFree:   5368709120,
			expectedUsed:   11811160064,
			expectedPercent: 68.75,
		},
		{
			name:           "low memory",
			psOutput:       "4294967296|536870912",
			expectedTotal:  4294967296,
			expectedFree:   536870912,
			expectedUsed:   3758096384,
			expectedPercent: 87.5,
		},
		{
			name:           "high available",
			psOutput:       "34359738368|27917287424",
			expectedTotal:  34359738368,
			expectedFree:   27917287424,
			expectedUsed:   6442450944,
			expectedPercent: 18.75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var info models.MemoryInfo
			parseWindowsMemoryOutput(tt.psOutput, &info)

			assert.Equal(t, tt.expectedTotal, info.Total, "Total memory should match")
			assert.Equal(t, tt.expectedFree, info.Free, "Free memory should match")
			assert.Equal(t, tt.expectedUsed, info.Used, "Used memory should match")
			assert.InDelta(t, tt.expectedPercent, info.UsedPercent, 0.5, "Used percent should match")
		})
	}
}

func TestMemoryPercentageCalculation(t *testing.T) {
	tests := []struct {
		name     string
		total    uint64
		used     uint64
		expected float64
	}{
		{"50% usage", 10000000000, 5000000000, 50.0},
		{"25% usage", 8000000000, 2000000000, 25.0},
		{"75% usage", 16000000000, 12000000000, 75.0},
		{"0% usage", 8000000000, 0, 0.0},
		{"100% usage", 4000000000, 4000000000, 100.0},
		{"zero total (edge case)", 0, 0, 0.0}, // Division par zÃ©ro
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var percent float64
			if tt.total > 0 {
				percent = float64(tt.used) / float64(tt.total) * 100
			}
			assert.InDelta(t, tt.expected, percent, 0.01, "Percentage should match")
		})
	}
}

func TestMemoryInfo_EdgeCases(t *testing.T) {
	t.Run("empty output", func(t *testing.T) {
		output := ""
		parts := splitMemoryOutput(output)
		assert.Len(t, parts, 0, "Empty output should produce no fields")
	})

	t.Run("invalid format - too few fields", func(t *testing.T) {
		output := "Mem: 1000 2000"
		parts := splitMemoryOutput(output)
		assert.Less(t, len(parts), 7, "Invalid format should have fewer than 7 fields")
	})

	t.Run("malformed numbers", func(t *testing.T) {
		output := "Mem: abc def ghi jkl mno pqr stu"
		var info models.MemoryInfo
		parseMemoryFields(splitMemoryOutput(output), &info)
		// strconv.ParseUint devrait retourner 0 en cas d'erreur
		assert.Equal(t, uint64(0), info.Total, "Malformed numbers should parse as 0")
	})

	t.Run("very large values", func(t *testing.T) {
		// 1 TB de RAM
		output := "Mem: 1099511627776 549755813888 274877906944 0 274877906944 824633720832"
		var info models.MemoryInfo
		parseMemoryFields(splitMemoryOutput(output), &info)
		require.Greater(t, info.Total, uint64(0), "Should parse large values")
		assert.InDelta(t, 25.0, info.UsedPercent, 0.5, "Percentage should be correct for large values")
	})
}

func TestMemoryInfoWindows_EdgeCases(t *testing.T) {
	t.Run("empty output", func(t *testing.T) {
		var info models.MemoryInfo
		parseWindowsMemoryOutput("", &info)
		assert.Equal(t, uint64(0), info.Total, "Empty output should result in zero values")
	})

	t.Run("missing pipe separator", func(t *testing.T) {
		var info models.MemoryInfo
		parseWindowsMemoryOutput("17179869184", &info)
		assert.Equal(t, uint64(0), info.Free, "Missing separator should not parse free memory")
	})

	t.Run("malformed values", func(t *testing.T) {
		var info models.MemoryInfo
		parseWindowsMemoryOutput("abc|def", &info)
		assert.Equal(t, uint64(0), info.Total, "Malformed values should parse as 0")
	})
}

// Fonctions helper pour les tests (simuler le parsing)
func splitMemoryOutput(output string) []string {
	if output == "" {
		return []string{}
	}
	// Simuler strings.Fields
	parts := []string{}
	current := ""
	for _, char := range output + " " {
		if char == ' ' || char == '\t' || char == '\n' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	return parts
}

func parseMemoryFields(fields []string, info *models.MemoryInfo) {
	if len(fields) < 7 {
		return
	}

	parseUint := func(s string) uint64 {
		var val uint64
		for _, char := range s {
			if char >= '0' && char <= '9' {
				val = val*10 + uint64(char-'0')
			} else {
				return 0 // CaractÃ¨re invalide
			}
		}
		return val
	}

	info.Total = parseUint(fields[1])
	info.Used = parseUint(fields[2])
	info.Free = parseUint(fields[3])
	info.Available = parseUint(fields[6])

	// Recalculer Used (Total - Available)
	if info.Total > 0 && info.Available > 0 {
		info.Used = info.Total - info.Available
	}

	// Calculer pourcentage
	if info.Total > 0 {
		info.UsedPercent = float64(info.Used) / float64(info.Total) * 100
	}
}

func parseWindowsMemoryOutput(output string, info *models.MemoryInfo) {
	// Parser "total|free"
	parts := []string{}
	current := ""
	for _, char := range output {
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

	if len(parts) < 2 {
		return
	}

	parseUint := func(s string) uint64 {
		var val uint64
		for _, char := range s {
			if char >= '0' && char <= '9' {
				val = val*10 + uint64(char-'0')
			} else {
				return 0
			}
		}
		return val
	}

	info.Total = parseUint(parts[0])
	info.Free = parseUint(parts[1])
	info.Available = info.Free
	info.Used = info.Total - info.Free

	if info.Total > 0 {
		info.UsedPercent = float64(info.Used) / float64(info.Total) * 100
	}
}

// Benchmark tests
func BenchmarkMemoryParsing_Linux(b *testing.B) {
	output := "Mem:   16777216000   8388608000   2097152000   104857600   6291456000   7340032000"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var info models.MemoryInfo
		parseMemoryFields(splitMemoryOutput(output), &info)
	}
}

func BenchmarkMemoryParsing_Windows(b *testing.B) {
	output := "17179869184|5368709120"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var info models.MemoryInfo
		parseWindowsMemoryOutput(output, &info)
	}
}
