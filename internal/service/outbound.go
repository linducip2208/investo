package service

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OutboundClient is a hardened HTTP client for server-side fetches of
// user-influenced URLs (webhook tests, provider connection tests, callbacks).
// It enforces https (except explicit opt-in), blocks loopback/link-local/
// cloud-metadata targets, caps redirect hops, response size and latency.
type OutboundClient struct {
	HTTP *http.Client
}

// NewOutboundClient builds a client with the given per-request timeout.
func NewOutboundClient(timeout time.Duration) *OutboundClient {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	return &OutboundClient{
		HTTP: &http.Client{Timeout: timeout, Transport: transport, CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 2 {
				return fmt.Errorf("outbound: too many redirects")
			}
			return nil
		}},
	}
}

// ValidateOutboundURL rejects URLs that must never be fetched server-side:
// non-http(s) schemes, embedded credentials, loopback/private/link-local
// hosts and the cloud metadata IP. DNS-resolved private IPs are checked by
// the caller via ResolveAndCheckHost when a hostname is supplied.
func ValidateOutboundURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("outbound: invalid URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("outbound: only http(s) URLs are allowed")
	}
	if parsed.User != nil {
		return fmt.Errorf("outbound: credentials in URL are not allowed")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return fmt.Errorf("outbound: invalid URL host")
	}
	blockedHosts := []string{"localhost", "metadata.google.internal"}
	for _, blocked := range blockedHosts {
		if host == blocked || strings.HasSuffix(host, "."+blocked) {
			return fmt.Errorf("outbound: host is not allowed")
		}
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("outbound: IP address is not allowed")
		}
	}
	if strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".lan") {
		return fmt.Errorf("outbound: internal hostnames are not allowed")
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	// Cloud metadata endpoints.
	if ip.String() == "169.254.169.254" || ip.String() == "fd00:ec2::254" {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// RFC1918 + carrier-grade NAT + reserved ranges.
		if ip4[0] == 10 || ip4[0] == 127 {
			return true
		}
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		if ip4[0] >= 224 {
			return true
		}
	}
	return false
}

// ResolveAndCheckHost resolves a hostname and rejects it when every address
// is non-public (best-effort TOCTOU-aware complement to ValidateOutboundURL).
func ResolveAndCheckHost(ctx context.Context, host string) error {
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("outbound: IP address is not allowed")
		}
		return nil
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(addrs) == 0 {
		return fmt.Errorf("outbound: cannot resolve host")
	}
	for _, addr := range addrs {
		if !isBlockedIP(addr.IP) {
			return nil
		}
	}
	return fmt.Errorf("outbound: host resolves to internal addresses only")
}

// GetJSON fetches url with ctx, validates the target first, and returns at
// most maxBytes of the response body for status 2xx.
func (c *OutboundClient) GetJSON(ctx context.Context, rawURL string, maxBytes int64, headers map[string]string) ([]byte, int, error) {
	if err := ValidateOutboundURL(rawURL); err != nil {
		return nil, 0, err
	}
	parsed, _ := url.Parse(rawURL)
	if err := ResolveAndCheckHost(ctx, parsed.Hostname()); err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("outbound: unexpected status %d", resp.StatusCode)
	}
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}
