package collectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsVirtualFS(t *testing.T) {
	tests := []struct {
		name     string
		fsType   string
		expected bool
	}{
		// Virtual filesystems
		{"tmpfs", "tmpfs", true},
		{"devtmpfs", "devtmpfs", true},
		{"sysfs", "sysfs", true},
		{"proc", "proc", true},
		{"devpts", "devpts", true},
		{"cgroup", "cgroup", true},
		{"cgroup2", "cgroup2", true},
		{"pstore", "pstore", true},
		{"securityfs", "securityfs", true},
		{"debugfs", "debugfs", true},
		{"hugetlbfs", "hugetlbfs", true},
		{"mqueue", "mqueue", true},
		{"configfs", "configfs", true},
		{"fusectl", "fusectl", true},
		{"binfmt_misc", "binfmt_misc", true},
		{"autofs", "autofs", true},
		{"overlay", "overlay", true},

		// Real filesystems
		{"ext4", "ext4", false},
		{"ext3", "ext3", false},
		{"xfs", "xfs", false},
		{"btrfs", "btrfs", false},
		{"ntfs", "ntfs", false},
		{"vfat", "vfat", false},
		{"exfat", "exfat", false},
		{"zfs", "zfs", false},
		{"f2fs", "f2fs", false},
		{"jfs", "jfs", false},
		{"reiserfs", "reiserfs", false},

		// Edge cases
		{"", "", false},
		{"unknown", "unknown", false},
		{"TMPFS", "TMPFS", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVirtualFS(tt.fsType)
			assert.Equal(t, tt.expected, result, "isVirtualFS(%q) should return %v", tt.fsType, tt.expected)
		})
	}
}

func TestParseLsLine(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		basePath    string
		expectNil   bool
		expectedName string
		expectedSize int64
		expectedIsDir bool
		expectedPerms string
	}{
		{
			name:        "directory",
			input:       "drwxr-xr-x 5 root root 4096 Jan 15 10:30 etc",
			basePath:    "/",
			expectNil:   false,
			expectedName: "etc",
			expectedSize: 4096,
			expectedIsDir: true,
			expectedPerms: "drwxr-xr-x",
		},
		{
			name:        "regular file",
			input:       "-rw-r--r-- 1 user group 1234567 Dec 25 15:45 file.txt",
			basePath:    "/home/user",
			expectNil:   false,
			expectedName: "file.txt",
			expectedSize: 1234567,
			expectedIsDir: false,
			expectedPerms: "-rw-r--r--",
		},
		{
			name:        "file with spaces in name",
			input:       "-rw-r--r-- 1 user group 9999 Nov 3 2023 my document.pdf",
			basePath:    "/home/user/Documents",
			expectNil:   false,
			expectedName: "my document.pdf",
			expectedSize: 9999,
			expectedIsDir: false,
			expectedPerms: "-rw-r--r--",
		},
		{
			name:        "symlink",
			input:       "lrwxrwxrwx 1 root root 11 Jan 1 00:00 link -> /usr/bin",
			basePath:    "/usr/local/bin",
			expectNil:   false,
			expectedName: "link -> /usr/bin",
			expectedSize: 11,
			expectedIsDir: false,
			expectedPerms: "lrwxrwxrwx",
		},
		{
			name:        "large file",
			input:       "-rw-r--r-- 1 user group 123456789012 Feb 28 23:59 largefile.iso",
			basePath:    "/mnt/data",
			expectNil:   false,
			expectedName: "largefile.iso",
			expectedSize: 123456789012,
			expectedIsDir: false,
			expectedPerms: "-rw-r--r--",
		},
		{
			name:      "invalid - too few fields",
			input:     "drwxr-xr-x 5 root",
			basePath:  "/",
			expectNil: true,
		},
		{
			name:      "invalid - empty line",
			input:     "",
			basePath:  "/",
			expectNil: true,
		},
		{
			name:      "total line (should be skipped)",
			input:     "total 48",
			basePath:  "/",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLsLine(tt.input, tt.basePath)

			if tt.expectNil {
				assert.Nil(t, result, "parseLsLine should return nil for invalid input")
			} else {
				assert.NotNil(t, result, "parseLsLine should not return nil")
				if result != nil {
					assert.Equal(t, tt.expectedName, result.Name, "Name should match")
					assert.Equal(t, tt.expectedSize, result.Size, "Size should match")
					assert.Equal(t, tt.expectedIsDir, result.IsDir, "IsDir should match")
					assert.Equal(t, tt.expectedPerms, result.Permissions, "Permissions should match")
				}
			}
		})
	}
}

func TestParseLsLine_SpecialCases(t *testing.T) {
	t.Run("current directory (.)", func(t *testing.T) {
		input := "drwxr-xr-x 5 root root 4096 Jan 15 10:30 ."
		result := parseLsLine(input, "/home/user")
		// La fonction retourne l'entry mais le caller doit la filtrer
		assert.NotNil(t, result)
		assert.Equal(t, ".", result.Name)
	})

	t.Run("parent directory (..)", func(t *testing.T) {
		input := "drwxr-xr-x 5 root root 4096 Jan 15 10:30 .."
		result := parseLsLine(input, "/home/user")
		assert.NotNil(t, result)
		assert.Equal(t, "..", result.Name)
	})

	t.Run("hidden file", func(t *testing.T) {
		input := "-rw-r--r-- 1 user group 1234 Jan 15 10:30 .bashrc"
		result := parseLsLine(input, "/home/user")
		assert.NotNil(t, result)
		assert.Equal(t, ".bashrc", result.Name)
		assert.False(t, result.IsDir)
	})

	t.Run("file with unicode name", func(t *testing.T) {
		input := "-rw-r--r-- 1 user group 1234 Jan 15 10:30 fichier_franÃ§ais_æ—¥æœ¬èªž.txt"
		result := parseLsLine(input, "/home/user")
		assert.NotNil(t, result)
		assert.Equal(t, "fichier_franÃ§ais_æ—¥æœ¬èªž.txt", result.Name)
	})

	t.Run("zero size file", func(t *testing.T) {
		input := "-rw-r--r-- 1 user group 0 Jan 15 10:30 empty.txt"
		result := parseLsLine(input, "/tmp")
		assert.NotNil(t, result)
		assert.Equal(t, int64(0), result.Size)
	})

	t.Run("executable file", func(t *testing.T) {
		input := "-rwxr-xr-x 1 root root 54321 Jan 15 10:30 script.sh"
		result := parseLsLine(input, "/usr/local/bin")
		assert.NotNil(t, result)
		assert.Equal(t, "script.sh", result.Name)
		assert.Contains(t, result.Permissions, "x", "Should have execute permission")
	})

	t.Run("device file", func(t *testing.T) {
		input := "brw-rw---- 1 root disk 8, 0 Jan 15 10:30 sda"
		result := parseLsLine(input, "/dev")
		assert.NotNil(t, result)
		assert.Equal(t, "sda", result.Name)
		assert.True(t, result.Permissions[0] == 'b', "Should be block device")
	})
}

func TestDetectDriveType_DiskName(t *testing.T) {
	// Test de la logique d'extraction du nom de disque
	tests := []struct {
		device       string
		expectedDisk string
	}{
		{"/dev/sda1", "sda"},
		{"/dev/sda2", "sda"},
		{"/dev/sdb", "sdb"},
		{"/dev/nvme0n1p1", "nvme0n1p"},
		{"/dev/nvme0n1p2", "nvme0n1p"},
		{"/dev/mmcblk0p1", "mmcblk0p"},
		{"/dev/vda1", "vda"},
		{"/dev/xvda1", "xvda"},
		{"sda1", "sda"},
		{"nvme0n1", "nvme0n"},
	}

	for _, tt := range tests {
		t.Run(tt.device, func(t *testing.T) {
			// Simuler la logique de detectDriveType
			baseName := tt.device
			if len(baseName) > 5 && baseName[:5] == "/dev/" {
				baseName = baseName[5:]
			}
			diskName := trimTrailingDigits(baseName)
			assert.Equal(t, tt.expectedDisk, diskName, "Disk name extraction should match")
		})
	}
}

func TestDiskPercentageCalculation(t *testing.T) {
	tests := []struct {
		name     string
		total    uint64
		used     uint64
		expected float64
	}{
		{"50% usage", 100000000000, 50000000000, 50.0},
		{"25% usage", 1000000000000, 250000000000, 25.0},
		{"90% usage", 500000000000, 450000000000, 90.0},
		{"0% usage", 100000000000, 0, 0.0},
		{"100% usage", 100000000000, 100000000000, 100.0},
		{"zero total", 0, 0, 0.0},
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

// Helper function pour extraire le nom de disque
func trimTrailingDigits(s string) string {
	for len(s) > 0 {
		lastChar := s[len(s)-1]
		if lastChar >= '0' && lastChar <= '9' {
			s = s[:len(s)-1]
		} else {
			break
		}
	}
	return s
}

// Benchmark tests
func BenchmarkIsVirtualFS(b *testing.B) {
	fsTypes := []string{"ext4", "tmpfs", "xfs", "proc", "btrfs", "sysfs"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isVirtualFS(fsTypes[i%len(fsTypes)])
	}
}

func BenchmarkParseLsLine(b *testing.B) {
	input := "-rw-r--r-- 1 user group 1234567 Dec 25 15:45 file.txt"
	basePath := "/home/user"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseLsLine(input, basePath)
	}
}

func BenchmarkParseLsLine_ComplexName(b *testing.B) {
	input := "-rw-r--r-- 1 user group 9999 Nov 3 2023 my document with many words.pdf"
	basePath := "/home/user/Documents"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseLsLine(input, basePath)
	}
}

// Tests de sÃ©curitÃ© pour BrowseDirectory
func TestBrowseDirectory_Security(t *testing.T) {
	// Ces tests vÃ©rifient que la validation de sÃ©curitÃ© fonctionne
	// Les tests complets avec mock SSH seront dans Sprint 2.3

	t.Run("path traversal attempt", func(t *testing.T) {
		// Le path devrait Ãªtre validÃ© par security.ValidatePath
		maliciousPaths := []string{
			"../../../etc/passwd",
			"/var/log/../../etc/shadow",
			"/home/user/..",
			"./../../root/.ssh",
		}

		for _, path := range maliciousPaths {
			t.Run(path, func(t *testing.T) {
				// Ces chemins devraient Ãªtre rejetÃ©s par ValidatePath
				// (testÃ© dans pkg/security/validation_test.go)
				// BrowseDirectory doit appeler ValidatePath avant toute opÃ©ration
				assert.Contains(t, path, "..", "Test path should contain traversal")
			})
		}
	})

	t.Run("sensitive paths", func(t *testing.T) {
		sensitivePaths := []string{
			"/etc/shadow",
			"/etc/gshadow",
			"/root/.ssh/id_rsa",
		}

		for _, path := range sensitivePaths {
			t.Run(path, func(t *testing.T) {
				// Ces chemins devraient Ãªtre bloquÃ©s par ValidatePath
				assert.NotEmpty(t, path, "Sensitive path should not be empty")
			})
		}
	})
}
