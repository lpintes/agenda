package auth

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

func TestExpired(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"other", errors.New("boom"), false},
		{"invalid_grant", &oauth2.RetrieveError{ErrorCode: "invalid_grant"}, true},
		{"other code", &oauth2.RetrieveError{ErrorCode: "invalid_client"}, false},
		// As returned by the HTTP client of the Calendar API.
		{"wrapped", fmt.Errorf("calendar Work: %w", &url.Error{
			Op: "Get", URL: "https://www.googleapis.com/calendar/v3/x",
			Err: &oauth2.RetrieveError{ErrorCode: "invalid_grant"},
		}), true},
	}
	for _, tt := range tests {
		if got := Expired(tt.err); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}
