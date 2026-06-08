package transform

import (
	"fmt"

	"github.com/davidbyttow/govips/v2/vips"
)

// Result holds the encoded output of a transform.
type Result struct {
	Data        []byte
	ContentType string
}

// Apply runs the requested transform pipeline (resize → encode) on the source
// image bytes using libvips and returns the encoded result.
//
// The caller is responsible for vips.Startup/Shutdown around the process
// lifetime; Apply only manages the per-request image handle.
func Apply(src []byte, opts Options) (Result, error) {
	img, err := vips.NewImageFromBuffer(src)
	if err != nil {
		return Result{}, fmt.Errorf("decode source: %w", err)
	}
	defer img.Close() // release native memory for this request

	// Resize step: scale to the requested width, preserving aspect ratio.
	if opts.Width > 0 {
		cur := img.Width()
		if cur <= 0 {
			return Result{}, fmt.Errorf("source has no width")
		}
		scale := float64(opts.Width) / float64(cur)
		if err := img.Resize(scale, vips.KernelLanczos3); err != nil {
			return Result{}, fmt.Errorf("resize to width %d: %w", opts.Width, err)
		}
	}

	// Encode step: pick the output format, applying quality where it applies.
	return encode(img, opts)
}

// encode serializes img into the requested format. When opts.Format is
// FormatOriginal the source's own format is kept.
func encode(img *vips.ImageRef, opts Options) (Result, error) {
	format := opts.Format
	if format == FormatOriginal {
		format = nativeFormat(img.Format())
	}

	var (
		data []byte
		err  error
	)
	switch format {
	case FormatWebP:
		p := vips.NewWebpExportParams()
		if opts.Quality > 0 {
			p.Quality = opts.Quality
		}
		data, _, err = img.ExportWebp(p)
	case FormatJPEG:
		p := vips.NewJpegExportParams()
		if opts.Quality > 0 {
			p.Quality = opts.Quality
		}
		data, _, err = img.ExportJpeg(p)
	case FormatAVIF:
		p := vips.NewAvifExportParams()
		if opts.Quality > 0 {
			p.Quality = opts.Quality
		}
		data, _, err = img.ExportAvif(p)
	case FormatPNG:
		p := vips.NewPngExportParams()
		if opts.Quality > 0 {
			p.Quality = opts.Quality
		}
		data, _, err = img.ExportPng(p)
	default:
		// Unknown native format (e.g. GIF/TIFF source with no override):
		// fall back to PNG, a safe lossless container.
		data, _, err = img.ExportPng(vips.NewPngExportParams())
		format = FormatPNG
	}
	if err != nil {
		return Result{}, fmt.Errorf("encode as %s: %w", format, err)
	}

	return Result{Data: data, ContentType: contentType(format)}, nil
}

// nativeFormat maps a libvips source type to our Format, or FormatOriginal
// when we have no first-class encoder for it.
func nativeFormat(t vips.ImageType) Format {
	switch t {
	case vips.ImageTypeJPEG:
		return FormatJPEG
	case vips.ImageTypePNG:
		return FormatPNG
	case vips.ImageTypeWEBP:
		return FormatWebP
	case vips.ImageTypeAVIF:
		return FormatAVIF
	default:
		return FormatOriginal
	}
}

// contentType returns the MIME type for an output Format.
func contentType(f Format) string {
	switch f {
	case FormatJPEG:
		return "image/jpeg"
	case FormatPNG:
		return "image/png"
	case FormatWebP:
		return "image/webp"
	case FormatAVIF:
		return "image/avif"
	default:
		return "application/octet-stream"
	}
}
