package request

import (
	"io"
	"net"
	"net/url"
	"testing"
	"time"
)

func TestCategorizeError_Timeout(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantType string
	}{
		{
			name:     "timeout error",
			err:      &timeoutError{},
			wantType: "timeout",
		},
		{
			name:     "deadline exceeded",
			err:      &deadlineExceededError{},
			wantType: "timeout",
		},
		{
			name:     "i/o timeout",
			err:      &ioTimeoutError{},
			wantType: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, _ := CategorizeError(tt.err)
			if gotType != tt.wantType {
				t.Errorf("CategorizeError() errorType = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

func TestCategorizeError_DNSError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantType string
	}{
		{
			name:     "no such host",
			err:      &dnsError{msg: "no such host"},
			wantType: "dns_error",
		},
		{
			name:     "DNS error",
			err:      &dnsError{msg: "dns lookup failed"},
			wantType: "dns_error",
		},
		{
			name:     "name resolution error",
			err:      &dnsError{msg: "name resolution failed"},
			wantType: "dns_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, _ := CategorizeError(tt.err)
			if gotType != tt.wantType {
				t.Errorf("CategorizeError() errorType = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

func TestCategorizeError_ConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantType string
	}{
		{
			name:     "EOF error",
			err:      io.EOF,
			wantType: "connection_error",
		},
		{
			name:     "connection refused",
			err:      &connectionError{msg: "connection refused"},
			wantType: "connection_error",
		},
		{
			name:     "connection reset",
			err:      &connectionError{msg: "connection reset"},
			wantType: "connection_error",
		},
		{
			name:     "broken pipe",
			err:      &connectionError{msg: "broken pipe"},
			wantType: "connection_error",
		},
		{
			name:     "network unreachable",
			err:      &connectionError{msg: "network is unreachable"},
			wantType: "connection_error",
		},
		{
			name:     "generic net.Error",
			err:      &genericNetError{},
			wantType: "connection_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, _ := CategorizeError(tt.err)
			if gotType != tt.wantType {
				t.Errorf("CategorizeError() errorType = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

func TestCategorizeError_URLError(t *testing.T) {
	// URL error with underlying timeout
	underlyingErr := &timeoutError{}
	urlErr := &url.Error{
		Op:  "Get",
		URL: "https://example.com",
		Err: underlyingErr,
	}

	gotType, _ := CategorizeError(urlErr)
	if gotType != "timeout" {
		t.Errorf("CategorizeError() errorType = %v, want timeout", gotType)
	}
}

func TestCategorizeError_UnknownError(t *testing.T) {
	err := &unknownError{msg: "some unknown error"}

	gotType, _ := CategorizeError(err)
	if gotType != "unknown_error" {
		t.Errorf("CategorizeError() errorType = %v, want unknown_error", gotType)
	}
}

func TestCategorizeError_NilError(t *testing.T) {
	gotType, gotCode := CategorizeError(nil)
	if gotType != "" || gotCode != "" {
		t.Errorf("CategorizeError(nil) = (%v, %v), want (\"\", \"\")", gotType, gotCode)
	}
}

// Mock error types for testing
type timeoutError struct{}

func (e *timeoutError) Error() string { return "timeout" }

type deadlineExceededError struct{}

func (e *deadlineExceededError) Error() string { return "deadline exceeded" }

type ioTimeoutError struct{}

func (e *ioTimeoutError) Error() string { return "i/o timeout" }

type dnsError struct {
	msg string
}

func (e *dnsError) Error() string { return e.msg }

type connectionError struct {
	msg string
}

func (e *connectionError) Error() string { return e.msg }

type genericNetError struct{}

func (e *genericNetError) Error() string   { return "network error" }
func (e *genericNetError) Timeout() bool   { return false }
func (e *genericNetError) Temporary() bool { return false }

type unknownError struct {
	msg string
}

func (e *unknownError) Error() string { return e.msg }

// Test with real net.Error types
func TestCategorizeError_RealNetError(t *testing.T) {
	// Create a real net.Error using net.DialTimeout with invalid address
	conn, err := net.DialTimeout("tcp", "192.0.2.1:80", 10*time.Millisecond)
	if conn != nil {
		conn.Close()
	}
	if err != nil {
		gotType, _ := CategorizeError(err)
		// Should be connection_error or timeout depending on which happens first
		if gotType != "connection_error" && gotType != "timeout" {
			t.Errorf("CategorizeError() errorType = %v, want connection_error or timeout", gotType)
		}
	}
}
