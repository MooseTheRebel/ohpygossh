package gohpygossh

import "testing"

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		in, host, port string
	}{
		{"example.com", "example.com", "22"},
		{"example.com:2222", "example.com", "2222"},
		{"10.0.0.1", "10.0.0.1", "22"},
		{"10.0.0.1:2200", "10.0.0.1", "2200"},
		{"::1", "::1", "22"},
		{"fe80::1", "fe80::1", "22"},
		{"[::1]:2222", "::1", "2222"},
		{"[2001:db8::1]", "2001:db8::1", "22"},
		{"Build01.Example.com", "build01.example.com", "22"},
		{"Build01.Example.com:2222", "build01.example.com", "2222"},
	}
	for _, tt := range tests {
		host, port := splitHostPort(tt.in)
		if host != tt.host || port != tt.port {
			t.Errorf("splitHostPort(%q) = (%q, %q), want (%q, %q)", tt.in, host, port, tt.host, tt.port)
		}
	}
}
