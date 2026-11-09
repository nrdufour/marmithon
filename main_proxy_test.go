package main

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// recordingDialer stands in for the SOCKS5 dialer and records what it was
// asked to dial.
type recordingDialer struct{ dialed []string }

func (d *recordingDialer) Dial(network, addr string) (net.Conn, error) {
	d.dialed = append(d.dialed, addr)
	return net.Dial(network, addr)
}

// TLS connections must go through the given dialer (regression: hellabot's
// DialTLS defaulted to a direct tls.Dial, bypassing the proxy).
func TestDialTLSViaUsesTheDialer(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "https://")

	d := &recordingDialer{}
	pool := srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	conn, err := dialTLSVia(d, "tcp", addr, &tls.Config{RootCAs: pool, ServerName: "example.com"})
	if err != nil {
		t.Fatalf("dialTLSVia: %v", err)
	}
	conn.Close()
	if len(d.dialed) != 1 || d.dialed[0] != addr {
		t.Fatalf("dialer not used for the TLS connection: %v", d.dialed)
	}
}

// Without an explicit ServerName, the host from addr is used, like tls.Dial.
// Go's TLS client refuses to handshake with neither a ServerName nor
// InsecureSkipVerify, so a successful, verified handshake against the test
// certificate (valid for 127.0.0.1) proves the default was applied.
func TestDialTLSViaDefaultsServerNameFromAddr(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "https://")
	pool := srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	conn, err := dialTLSVia(&recordingDialer{}, "tcp", addr, &tls.Config{RootCAs: pool})
	if err != nil {
		t.Fatalf("dialTLSVia: %v", err)
	}
	conn.Close()
}
