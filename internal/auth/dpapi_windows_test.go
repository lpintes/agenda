package auth

import (
	"bytes"
	"testing"
)

func TestProtectRoundTrip(t *testing.T) {
	data := []byte(`{"access_token":"a","refresh_token":"r"}`)
	enc, err := protect(data)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(enc, []byte("refresh_token")) {
		t.Fatal("encrypted data contains plain text")
	}
	dec, err := unprotect(enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec, data) {
		t.Fatalf("got %q, want %q", dec, data)
	}
}

func TestUnprotectGarbage(t *testing.T) {
	if _, err := unprotect([]byte("not a DPAPI blob")); err == nil {
		t.Fatal("expected an error")
	}
}
