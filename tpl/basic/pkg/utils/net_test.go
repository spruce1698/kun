package utils

import (
	"testing"
)

func TestIPv42Int64_Valid(t *testing.T) {
	tests := []struct {
		ip   string
		want int64
	}{
		{"127.0.0.1", 2130706433},
		{"0.0.0.0", 0},
		{"255.255.255.255", 4294967295},
		{"192.168.1.1", 3232235777},
	}

	for _, tt := range tests {
		got := IPv42Int64(tt.ip)
		if got != tt.want {
			t.Errorf("IPv42Int64(%q) = %d, want %d", tt.ip, got, tt.want)
		}
	}
}

func TestIPv42Int64_InvalidNilSafe(t *testing.T) {
	invalids := []string{
		"",
		"invalid",
		"999.999.999.999",
		"::1", // IPv6
		"2001:db8::1",
	}

	for _, ip := range invalids {
		got := IPv42Int64(ip)
		if got != 0 {
			t.Errorf("IPv42Int64(%q) = %d, want 0", ip, got)
		}
	}
}

func TestIsIPv4_And_IsIPv6(t *testing.T) {
	if !IsIPv4("192.168.1.1") {
		t.Error("expected 192.168.1.1 to be IPv4")
	}
	if IsIPv4("::1") {
		t.Error("expected ::1 not to be IPv4")
	}
	if !IsIPv6("::1") {
		t.Error("expected ::1 to be IPv6")
	}
	if IsIPv6("192.168.1.1") {
		t.Error("expected 192.168.1.1 not to be IPv6")
	}
	if IsIPv4("invalid") || IsIPv6("invalid") {
		t.Error("invalid IP string should be false for both")
	}
}
