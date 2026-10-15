package ui

import (
	"errors"
	"net"
)

// describe returns the reason of a failed request for the user. Network
// failures get a plain explanation instead of the long text of the HTTP
// client, which repeats the whole request URL.
func describe(err error) string {
	var opErr *net.OpError
	var dnsErr *net.DNSError
	var netErr net.Error
	switch {
	case errors.As(err, &dnsErr), errors.As(err, &opErr):
		return "cannot reach Google, check the internet connection"
	case errors.As(err, &netErr) && netErr.Timeout():
		return "Google did not answer in time"
	}
	return err.Error()
}
