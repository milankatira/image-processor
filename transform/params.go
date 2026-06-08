// Package transform parses image-transform URL tokens and applies them
// to source images via libvips.
package transform

import (
	"fmt"
	"strconv"
	"strings"
)

// Format is the desired output encoding.
type Format string

const (
	FormatOriginal Format = "" // keep the source's format
	FormatJPEG     Format = "jpeg"
	FormatPNG      Format = "png"
	FormatWebP     Format = "webp"
	FormatAVIF     Format = "avif"
)

// Options is a parsed, validated transform spec. The zero value is a no-op
// (original format, server-default quality, no resize).
type Options struct {
	Format  Format
	Quality int // 1-100; 0 means "use encoder default"
	Width   int // target width in px; 0 means "do not resize"
}

// ParsePath turns a request path such as "/f_webp/q_80/r_320/123.png" into a
// set of Options plus the source filename ("123.png"). Tokens may appear in
// any order and are all optional, so "/123.png" is valid and yields zero Options.
func ParsePath(path string) (Options, string, error) {
	segments := splitNonEmpty(path, "/")
	if len(segments) == 0 {
		return Options{}, "", fmt.Errorf("empty path")
	}

	// The final segment is always the source filename; the rest are tokens.
	filename := segments[len(segments)-1]
	tokens := segments[:len(segments)-1]

	opts, err := parseTokens(tokens)
	if err != nil {
		return Options{}, "", err
	}
	return opts, filename, nil
}

// ParseTokens parses a path made up entirely of transform tokens, with no
// trailing source filename. Used when the source image is supplied out-of-band
// (e.g. a ?url= query parameter): "/f_webp/q_80/r_320/" → Options{...}.
func ParseTokens(path string) (Options, error) {
	return parseTokens(splitNonEmpty(path, "/"))
}

func parseTokens(tokens []string) (Options, error) {
	opts := Options{}
	for _, tok := range tokens {
		if err := applyToken(&opts, tok); err != nil {
			return Options{}, err
		}
	}
	return opts, nil
}

// applyToken parses a single "<key>_<value>" token into opts.
func applyToken(opts *Options, tok string) error {
	key, value, ok := strings.Cut(tok, "_")
	if !ok {
		return fmt.Errorf("malformed token %q: expected <key>_<value>", tok)
	}

	switch key {
	case "f":
		f, err := parseFormat(value)
		if err != nil {
			return err
		}
		opts.Format = f
	case "q":
		q, err := strconv.Atoi(value)
		if err != nil || q < 1 || q > 100 {
			return fmt.Errorf("invalid quality %q: want integer 1-100", value)
		}
		opts.Quality = q
	case "r":
		wpx, err := strconv.Atoi(value)
		if err != nil || wpx < 1 {
			return fmt.Errorf("invalid resize width %q: want positive integer", value)
		}
		opts.Width = wpx
	default:
		return fmt.Errorf("unknown token key %q", key)
	}
	return nil
}

// parseFormat normalizes a format string to a supported Format.
func parseFormat(v string) (Format, error) {
	switch strings.ToLower(v) {
	case "jpeg", "jpg":
		return FormatJPEG, nil
	case "png":
		return FormatPNG, nil
	case "webp":
		return FormatWebP, nil
	case "avif":
		return FormatAVIF, nil
	default:
		return FormatOriginal, fmt.Errorf("unsupported format %q", v)
	}
}

func splitNonEmpty(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
