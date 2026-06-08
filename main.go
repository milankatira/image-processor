package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/milankatira/image-rocessor/transform"
)

// imageRoot is the directory holding the original source images.
const imageRoot = "images"

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
	log.Printf("image-processor listening on %s (serving from %q)", addr, imageRoot)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

// handleImage parses the transform tokens from the path, loads the source
// image, applies the libvips pipeline, and streams the encoded result.
//
// Example: GET /f_webp/q_80/r_320/123.png
func handleImage(w http.ResponseWriter, r *http.Request) {
	opts, name, err := transform.ParsePath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// filepath.Base strips any directory traversal, so the lookup stays
	// inside imageRoot regardless of the requested name.
	srcPath := filepath.Join(imageRoot, filepath.Base(name))
	src, err := os.ReadFile(srcPath)
	if err != nil {
		log.Printf("read %q: %v", srcPath, err)
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	res, err := transform.Apply(src, opts)
	if err != nil {
		log.Printf("transform %q with %+v: %v", srcPath, opts, err)
		http.Error(w, "failed to process image", http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", res.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if _, err := w.Write(res.Data); err != nil {
		log.Printf("write response: %v", err)
	}
}
