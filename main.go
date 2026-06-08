package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/milankatira/image-rocessor/source"
	"github.com/milankatira/image-rocessor/transform"
)

// imageRoot is the directory holding the original (local) source images.
const imageRoot = "images"

// fetcher retrieves remote source images named via the ?url= query parameter.
var fetcher = source.New(source.Config{})

func main() {
	// Quiet libvips' verbose info logging — only surface real errors.
	vips.LoggingSettings(nil, vips.LogLevelError)

	// Boot libvips once for the lifetime of the process. Shutdown on exit
	// releases the native memory and operation cache.
	vips.Startup(nil)
	defer vips.Shutdown()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleImage)

	addr := ":8080"
	log.Printf("image-processor listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

// handleImage parses the transform tokens from the path, obtains the source
// image — remotely when ?url= is present, otherwise from the local imageRoot —
// applies the libvips pipeline, and streams the encoded result.
//
//	Remote: GET /f_webp/q_80/r_320/?url=https://example.com/photo.jpg
//	Local:  GET /f_webp/q_80/r_320/123.png
func handleImage(w http.ResponseWriter, r *http.Request) {
	remoteURL := r.URL.Query().Get("url")

	var (
		opts transform.Options
		src  []byte
		err  error
	)

	if remoteURL != "" {
		// Remote mode: every path segment is a transform token; the source
		// image comes from the ?url= parameter.
		opts, err = transform.ParseTokens(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		src, err = fetcher.Fetch(r.Context(), remoteURL)
		if err != nil {
			log.Printf("fetch %q: %v", remoteURL, err)
			http.Error(w, "could not fetch source image", fetchStatus(err))
			return
		}
	} else {
		// Local mode: final path segment is a filename under imageRoot.
		var name string
		opts, name, err = transform.ParsePath(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// filepath.Base strips traversal, keeping the lookup inside imageRoot.
		srcPath := filepath.Join(imageRoot, filepath.Base(name))
		src, err = os.ReadFile(srcPath)
		if err != nil {
			log.Printf("read %q: %v", srcPath, err)
			http.Error(w, "image not found", http.StatusNotFound)
			return
		}
	}

	res, err := transform.Apply(src, opts)
	if err != nil {
		log.Printf("transform with %+v: %v", opts, err)
		http.Error(w, "failed to process image", http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", res.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if _, err := w.Write(res.Data); err != nil {
		log.Printf("write response: %v", err)
	}
}

// fetchStatus maps a fetch failure to an HTTP status: a blocked address is a
// client error (400); anything else is an upstream/gateway failure (502).
func fetchStatus(err error) int {
	if errors.Is(err, source.ErrBlockedAddress) || errors.Is(err, source.ErrInvalidURL) {
		return http.StatusBadRequest
	}
	return http.StatusBadGateway
}
