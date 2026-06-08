// Package source fetches origin images for the processor. Remote fetching is
// an SSRF-sensitive operation, so the Fetcher enforces a scheme allow-list,
// blocks requests that resolve to private/loopback addresses, caps the
// response size, and applies a timeout.
package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// ErrBlockedAddress is returned when a URL resolves to a non-public address.
var ErrBlockedAddress = errors.New("destination address is not allowed")

// ErrInvalidURL is returned for a malformed URL or unsupported scheme — these
// are caller mistakes rather than upstream failures.
var ErrInvalidURL = errors.New("invalid source url")

// Fetcher retrieves remote source images under safety constraints.
type Fetcher struct {
	client   *http.Client
	maxBytes int64
}

// Config tunes a Fetcher.
type Config struct {
	Timeout      time.Duration // per-request deadline; defaults to 10s
	MaxBytes     int64         // response body cap; defaults to 25 MiB
	AllowPrivate bool          // permit private/loopback IPs (tests only)
}

// New builds a Fetcher. The zero Config is safe for production use.
func New(cfg Config) *Fetcher {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 25 << 20 // 25 MiB
	}

	dialer := &net.Dialer{Timeout: cfg.Timeout}
	if !cfg.AllowPrivate {
		// Control runs after DNS resolution with the concrete IP about to be
		// dialed, so it rejects private targets even across HTTP redirects and
		// DNS-rebinding tricks.
		dialer.Control = func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || isBlocked(ip) {
				return ErrBlockedAddress
			}
			return nil
		}
	}

	return &Fetcher{
		client: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: &http.Transport{DialContext: dialer.DialContext},
		},
		maxBytes: cfg.MaxBytes,
	}
}

// Fetch downloads the image at rawURL and returns its bytes. It rejects
// non-http(s) schemes and bodies larger than the configured cap.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w: unsupported scheme %q", ErrInvalidURL, u.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", u.Redacted(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("origin returned status %d", resp.StatusCode)
	}

	// Read at most maxBytes+1 so we can detect an over-cap body precisely.
	data, err := io.ReadAll(io.LimitReader(resp.Body, f.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if int64(len(data)) > f.maxBytes {
		return nil, fmt.Errorf("source exceeds %d byte limit", f.maxBytes)
	}
	return data, nil
}

// isBlocked reports whether ip is in a range we refuse to contact.
func isBlocked(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast()
}
