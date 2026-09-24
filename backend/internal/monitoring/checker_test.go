package monitoring

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"github.com/uptime-app/backend/internal/model"
)

func TestPublicAddress(t *testing.T) {
	for _, address := range []string{
		"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.100.100.200", "0.0.0.0", "224.0.0.1",
		"192.168.1.1", "172.16.0.1", "192.0.0.1", "198.18.0.1", "203.0.113.1", "240.1.1.1",
		"::1", "::", "fe80::1", "fc00::1", "::ffff:127.0.0.1", "64:ff9b::a00:1", "2002:7f00:1::",
		"2001:db8::1", "2001::1", "ff02::1", "3fff::1",
	} {
		if publicAddress(netip.MustParseAddr(address)) {
			t.Errorf("allowed %s", address)
		}
	}
	for _, address := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111", "::ffff:8.8.8.8"} {
		if !publicAddress(netip.MustParseAddr(address)) {
			t.Errorf("blocked %s", address)
		}
	}
}

func TestCheckerStatusAndNoRedirect(t *testing.T) {
	var redirected atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(200)
		case "/fail":
			w.WriteHeader(503)
		case "/redirect":
			w.Header().Set("Location", "/target")
			w.WriteHeader(302)
		case "/target":
			redirected.Add(1)
		}
	}))
	defer server.Close()
	checker := NewChecker()
	// Only this test replaces the production dial policy to reach its loopback fixture.
	checker.client.Transport = http.DefaultTransport.(*http.Transport).Clone()
	for path, want := range map[string]int{"/ok": 200, "/fail": 503, "/redirect": 302} {
		result := checker.Check(context.Background(), model.Monitor{URL: server.URL + path, IntervalSeconds: 5})
		if result.Error != "" || result.StatusCode == nil || *result.StatusCode != want {
			t.Fatalf("%s result = %+v", path, result)
		}
	}
	if redirected.Load() != 0 {
		t.Fatal("followed redirect")
	}
	blocked := NewChecker().Check(context.Background(), model.Monitor{URL: server.URL, IntervalSeconds: 5})
	if blocked.Error != "blocked_address" {
		t.Fatalf("loopback result: %+v", blocked)
	}
}

func TestCheckerTLSAndTimeout(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	defer server.Close()
	checker := NewChecker()
	checker.client.Transport = &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	result := checker.Check(context.Background(), model.Monitor{URL: server.URL, IntervalSeconds: 5})
	if result.Error != "tls" {
		t.Fatalf("TLS result: %+v", result)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	result = checker.Check(ctx, model.Monitor{URL: server.URL, IntervalSeconds: 5})
	if result.Error != "timeout" {
		t.Fatalf("timeout result: %+v", result)
	}
	if checkError(&net.DNSError{Err: "missing", IsNotFound: true}) != "dns" {
		t.Fatal("DNS classification")
	}
	if checkError(errors.New("refused")) != "connection" {
		t.Fatal("connection classification")
	}
}

func TestPublicDialResolvesAndBlocksLocalhost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := publicDial(ctx, "tcp", "localhost:80")
	if !errors.Is(err, errBlockedAddress) {
		t.Fatalf("resolved localhost: %v", err)
	}
}

func TestDialPinsDNSAndRejectsMixedAnswers(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		lookups, dials := 0, 0
		dependencies := dialDependencies{
			lookup: func(_ context.Context, _, _ string) ([]netip.Addr, error) {
				lookups++
				addresses := []netip.Addr{netip.MustParseAddr("8.8.8.8")}
				if mixed || lookups > 1 {
					addresses = append(addresses, netip.MustParseAddr("127.0.0.1"))
				}
				return addresses, nil
			},
			dial: func(_ context.Context, _, address string) (net.Conn, error) {
				dials++
				if address != "8.8.8.8:443" {
					t.Fatalf("dialed unpinned hostname: %s", address)
				}
				return nil, errors.New("test connection refused")
			},
		}
		_, err := dialPublic(context.Background(), "tcp", "rebinding.example:443", dependencies)
		if lookups != 1 {
			t.Fatal("DNS resolved twice")
		}
		if mixed && (dials != 0 || !errors.Is(err, errBlockedAddress)) {
			t.Fatal("mixed public/private answer allowed")
		}
		if !mixed && dials != 1 {
			t.Fatal("validated literal not dialed")
		}
	}
}
