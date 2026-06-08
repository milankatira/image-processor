package transform

import "testing"

func TestParsePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantOpts Options
		wantFile string
		wantErr  bool
	}{
		{
			name:     "full spec in canonical order",
			path:     "/f_webp/q_80/r_320/123.png",
			wantOpts: Options{Format: FormatWebP, Quality: 80, Width: 320},
			wantFile: "123.png",
		},
		{
			name:     "tokens in any order",
			path:     "/r_640/f_jpeg/q_50/photo.png",
			wantOpts: Options{Format: FormatJPEG, Quality: 50, Width: 640},
			wantFile: "photo.png",
		},
		{
			name:     "no tokens means zero options",
			path:     "/123.png",
			wantOpts: Options{},
			wantFile: "123.png",
		},
		{
			name:     "jpg alias normalizes to jpeg",
			path:     "/f_jpg/123.png",
			wantOpts: Options{Format: FormatJPEG},
			wantFile: "123.png",
		},
		{
			name:     "gif format",
			path:     "/f_gif/123.png",
			wantOpts: Options{Format: FormatGIF},
			wantFile: "123.png",
		},
		{
			name:     "tif alias normalizes to tiff",
			path:     "/f_tif/123.png",
			wantOpts: Options{Format: FormatTIFF},
			wantFile: "123.png",
		},
		{name: "quality below range", path: "/q_0/123.png", wantErr: true},
		{name: "quality above range", path: "/q_101/123.png", wantErr: true},
		{name: "negative width", path: "/r_-5/123.png", wantErr: true},
		{name: "unsupported format", path: "/f_bmp/123.png", wantErr: true},
		{name: "unknown token key", path: "/z_9/123.png", wantErr: true},
		{name: "malformed token", path: "/webp/123.png", wantErr: true},
		{name: "empty path", path: "/", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, file, err := ParsePath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParsePath(%q) = nil error, want error", tt.path)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePath(%q) unexpected error: %v", tt.path, err)
			}
			if opts != tt.wantOpts {
				t.Errorf("opts = %+v, want %+v", opts, tt.wantOpts)
			}
			if file != tt.wantFile {
				t.Errorf("file = %q, want %q", file, tt.wantFile)
			}
		})
	}
}
