package handlers

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1 KiB"},
		{1500, "1.46 KiB"},
		{1048576, "1 MiB"},
		{1073741824, "1 GiB"},
		{1099511627776, "1 TiB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytes(%d): expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0.0, "0"},
		{1.0, "1"},
		{1.5, "1.50"},
		{1.2345, "1.23"},
		{10.0, "10"},
		{10.05, "10.05"},
	}

	for _, tt := range tests {
		result := formatFloat(tt.input)
		if result != tt.expected {
			t.Errorf("formatFloat(%f): expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}

func TestIntToString(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{9223372036854775807, "9223372036854775807"},
	}

	for _, tt := range tests {
		result := intToString(tt.input)
		if result != tt.expected {
			t.Errorf("intToString(%d): expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}

func BenchmarkFormatBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		formatBytes(1500)
		formatBytes(1048576)
		formatBytes(123456789)
	}
}

func BenchmarkFormatFloat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		formatFloat(123.456)
	}
}

func BenchmarkIntToString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		intToString(123456789)
	}
}
