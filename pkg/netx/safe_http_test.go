package netx

import (
	"net"
	"testing"
	"time"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true},
		{"100.64.0.1", true},
		{"192.0.2.1", true},
		{"198.51.100.1", true},
		{"203.0.113.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}

	for _, tc := range tests {
		ip := net.ParseIP(tc.ip)
		if got := IsPrivateIP(ip); got != tc.expected {
			t.Errorf("IsPrivateIP(%s) = %v, expected %v", tc.ip, got, tc.expected)
		}
	}

	if !IsPrivateIP(nil) {
		t.Errorf("IsPrivateIP(nil) should be true")
	}
}

func TestNewSafeHTTPClient(t *testing.T) {
	client := NewSafeHTTPClient(5 * time.Second)
	if client == nil {
		t.Fatalf("expected non-nil safe http client")
	}
	if client.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", client.Timeout)
	}
}
