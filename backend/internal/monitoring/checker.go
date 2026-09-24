// Package monitoring executes scheduled HTTP probes without depending on browser activity.
package monitoring

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/uptime-app/backend/internal/model"
)

var errBlockedAddress = errors.New("blocked destination")

// Special-use destinations must not turn the public checker into an internal-network proxy.
var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"), netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("168.63.129.16/32"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("3fff::/20"),
}

func publicAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || address.Zone() != "" || !address.IsGlobalUnicast() {
		return false
	}
	if address.Is6() && !netip.MustParsePrefix("2000::/3").Contains(address) {
		return false
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

type Checker struct{ client *http.Client }

func NewChecker() *Checker {
	transport := &http.Transport{
		Proxy:                  nil,
		DisableKeepAlives:      true,
		DialContext:            publicDial,
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12},
		ResponseHeaderTimeout:  10 * time.Second,
		MaxResponseHeaderBytes: 64 << 10,
	}
	return &Checker{client: &http.Client{
		Transport:     transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func publicDial(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := net.Dialer{}
	return dialPublic(ctx, network, address, dialDependencies{
		lookup: net.DefaultResolver.LookupNetIP,
		dial:   dialer.DialContext,
	})
}

type dialDependencies struct {
	lookup func(context.Context, string, string) ([]netip.Addr, error)
	dial   func(context.Context, string, string) (net.Conn, error)
}

func dialPublic(ctx context.Context, network, address string, dependencies dialDependencies) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addresses, err := dependencies.lookup(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if len(addresses) == 0 {
		return nil, &net.DNSError{Err: "no addresses", Name: host, IsNotFound: true}
	}
	for _, ip := range addresses {
		if !publicAddress(ip) {
			return nil, errBlockedAddress
		}
	}
	// Dial the validated literal, never the hostname, to prevent a second DNS lookup/rebinding.
	for i, ip := range addresses {
		attemptCtx := ctx
		cancel := func() {}
		if deadline, ok := ctx.Deadline(); ok {
			attemptCtx, cancel = context.WithTimeout(ctx, time.Until(deadline)/time.Duration(len(addresses)-i))
		}
		var connection net.Conn
		connection, err = dependencies.dial(attemptCtx, network, net.JoinHostPort(ip.String(), port))
		cancel()
		if err == nil {
			return connection, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, err
}

func (c *Checker) Check(ctx context.Context, monitor model.Monitor) model.CheckResult {
	started := time.Now()
	ctx, cancel := context.WithTimeout(ctx, model.CheckTimeout(monitor.IntervalSeconds))
	defer cancel()
	result := model.CheckResult{}
	parsed, err := url.Parse(monitor.URL)
	if err == nil && (parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https")) {
		err = errBlockedAddress
	}
	if err == nil {
		var request *http.Request
		request, err = http.NewRequestWithContext(ctx, http.MethodGet, monitor.URL, nil)
		if err == nil {
			request.Header.Set("User-Agent", "Uptime-Monitor/1.0")
			var response *http.Response
			response, err = c.client.Do(request)
			if response != nil {
				_ = response.Body.Close()
				result.StatusCode = &response.StatusCode
			}
		}
	}
	if err != nil {
		result.Error = checkError(err)
	}
	result.CheckedAt = time.Now().UTC()
	result.DurationMS = time.Since(started).Milliseconds()
	return result
}

func checkError(err error) string {
	var dns *net.DNSError
	var network net.Error
	var certificate *tls.CertificateVerificationError
	var unknownAuthority x509.UnknownAuthorityError
	var record tls.RecordHeaderError
	switch {
	case errors.Is(err, errBlockedAddress):
		return "blocked_address"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.As(err, &network) && network.Timeout():
		return "timeout"
	case errors.As(err, &dns):
		return "dns"
	case errors.As(err, &certificate), errors.As(err, &unknownAuthority), errors.As(err, &record):
		return "tls"
	case strings.Contains(err.Error(), "tls:"):
		return "tls"
	default:
		return "connection"
	}
}
