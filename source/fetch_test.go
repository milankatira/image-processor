package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetch(t *testing.T) {
	const body = "fake-image-bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Write([]byte(body))
		case "/big":
			w.Write([]byte(strings.Repeat("x", 100)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// httptest binds to loopback, so allow private addresses for the happy path.
	f := New(Config{AllowPrivate: true, MaxBytes: 50})

	t.Run("fetches body", func(t *testing.T) {
		got, err := f.Fetch(context.Background(), srv.URL+"/ok")
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		if string(got) != body {
			t.Errorf("body = %q, want %q", got, body)
		}
	})

	t.Run("rejects non-200", func(t *testing.T) {
		if _, err := f.Fetch(context.Background(), srv.URL+"/missing"); err == nil {
			t.Fatal("expected error on 404")
		}
	})

	t.Run("enforces size cap", func(t *testing.T) {
		if _, err := f.Fetch(context.Background(), srv.URL+"/big"); err == nil {
			t.Fatal("expected error when body exceeds MaxBytes")
		}
	})

	t.Run("rejects non-http scheme", func(t *testing.T) {
		if _, err := f.Fetch(context.Background(), "file:///etc/passwd"); err == nil {
			t.Fatal("expected error for file:// scheme")
		}
	})
}

func TestFetchBlocksPrivateAddress(t *testing.T) {
	// Default config (AllowPrivate=false) must refuse loopback targets — the
	// core SSRF guard.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("secret"))
	}))
	defer srv.Close()

	f := New(Config{})
	if _, err := f.Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("expected private-address fetch to be blocked")
	}
}
