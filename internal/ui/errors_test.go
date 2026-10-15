package ui

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"
)

type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestDescribe(t *testing.T) {
	const u = "https://www.googleapis.com/calendar/v3/calendars/x/events"
	offline := "cannot reach Google, check the internet connection"
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"other", errors.New("boom"), "boom"},
		{"dns", &url.Error{Op: "Get", URL: u, Err: &net.DNSError{Err: "no such host", Name: "www.googleapis.com"}}, offline},
		{"dial", fmt.Errorf("calendar Work: %w", &url.Error{Op: "Get", URL: u, Err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("refused")}}), offline},
		{"timeout", &url.Error{Op: "Get", URL: u, Err: timeoutError{}}, "Google did not answer in time"},
		{"not network", &url.Error{Op: "Get", URL: u, Err: context.Canceled}, (&url.Error{Op: "Get", URL: u, Err: context.Canceled}).Error()},
	}
	for _, tt := range tests {
		if got := describe(tt.err); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}
