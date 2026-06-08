package transform

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
)

// TestMain boots libvips once for the whole package test run.
func TestMain(m *testing.M) {
	vips.LoggingSettings(nil, vips.LogLevelError)
	vips.Startup(nil)
	defer vips.Shutdown()
	m.Run()
}

// makePNG returns an encoded w×h PNG to feed the pipeline.
func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img, err := vips.Black(w, h)
	if err != nil {
		t.Fatalf("create black image: %v", err)
	}
	defer img.Close()
	data, _, err := img.ExportPng(vips.NewPngExportParams())
	if err != nil {
		t.Fatalf("encode seed png: %v", err)
	}
	return data
}

func TestApply(t *testing.T) {
	src := makePNG(t, 640, 480)

	t.Run("resize preserves aspect and changes format", func(t *testing.T) {
		res, err := Apply(src, Options{Format: FormatWebP, Quality: 80, Width: 320})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
		if res.ContentType != "image/webp" {
			t.Errorf("ContentType = %q, want image/webp", res.ContentType)
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(res.Data))
		if err != nil {
			// Go's stdlib can't decode webp by default; fall back to dimension
			// check via libvips below. Only assert format when decodable.
			assertVipsWidth(t, res.Data, 320, 240)
			return
		}
		if cfg.Width != 320 || cfg.Height != 240 {
			t.Errorf("dims = %dx%d (%s), want 320x240", cfg.Width, cfg.Height, format)
		}
	})

	t.Run("format only keeps original dimensions", func(t *testing.T) {
		res, err := Apply(src, Options{Format: FormatJPEG})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
		if res.ContentType != "image/jpeg" {
			t.Errorf("ContentType = %q, want image/jpeg", res.ContentType)
		}
		assertVipsWidth(t, res.Data, 640, 480)
	})

	t.Run("no options keeps source format", func(t *testing.T) {
		res, err := Apply(src, Options{})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
		if res.ContentType != "image/png" {
			t.Errorf("ContentType = %q, want image/png", res.ContentType)
		}
	})

	t.Run("lower quality yields smaller jpeg", func(t *testing.T) {
		hi, err := Apply(src, Options{Format: FormatJPEG, Quality: 95})
		if err != nil {
			t.Fatalf("Apply hi: %v", err)
		}
		lo, err := Apply(src, Options{Format: FormatJPEG, Quality: 20})
		if err != nil {
			t.Fatalf("Apply lo: %v", err)
		}
		if len(lo.Data) >= len(hi.Data) {
			t.Errorf("q20 (%d bytes) should be smaller than q95 (%d bytes)", len(lo.Data), len(hi.Data))
		}
	})

	t.Run("invalid source errors", func(t *testing.T) {
		if _, err := Apply([]byte("not an image"), Options{}); err == nil {
			t.Fatal("expected error decoding garbage input")
		}
	})
}

// assertVipsWidth re-decodes data with libvips and checks its dimensions —
// works for formats Go's stdlib can't read (webp/avif).
func assertVipsWidth(t *testing.T, data []byte, wantW, wantH int) {
	t.Helper()
	img, err := vips.NewImageFromBuffer(data)
	if err != nil {
		t.Fatalf("re-decode result: %v", err)
	}
	defer img.Close()
	if img.Width() != wantW || img.Height() != wantH {
		t.Errorf("dims = %dx%d, want %dx%d", img.Width(), img.Height(), wantW, wantH)
	}
}
