# image-processor

On-the-fly image transformation HTTP service backed by [libvips](https://www.libvips.org/) (via [govips](https://github.com/davidbyttow/govips)).

Transform parameters are encoded as path-prefix tokens, so a single URL fully
describes the desired output:

```
/f_webp/q_80/r_320/123.png
   │      │     │     └── source file in ./images
   │      │     └──────── resize to 320px wide (aspect preserved)
   │      └────────────── quality 80
   └───────────────────── output format webp
```

## Tokens

| Token  | Meaning            | Values                        | Optional |
|--------|--------------------|-------------------------------|----------|
| `f_*`  | output format      | `jpeg`/`jpg`, `png`, `webp`, `avif`, `gif`, `tiff`/`tif` | yes (keeps source format) |
| `q_*`  | encode quality     | integer `1`–`100`             | yes (encoder default) |
| `r_*`  | resize width (px)  | positive integer              | yes (no resize) |

Tokens may appear in any order and are all optional.

**Formats** — output (`f_*`) is one of `jpeg`, `png`, `webp`, `avif`, `gif`, `tiff`.
Input can be any of those plus anything else libvips decodes (heic, svg, pdf, …);
when `f_*` is omitted the source format is preserved. AVIF encodes at a reduced
effort level so responses stay sub-second on large images.

## Source image: remote or local

The source can come from either place:

**Remote (`?url=`)** — the image is fetched from any public http(s) URL:

```
/f_webp/q_80/r_320/?url=https://example.com/photo.jpg
```

Remote fetching is SSRF-guarded: only `http`/`https` schemes are allowed,
URLs resolving to loopback/private/link-local addresses are refused (the check
runs at dial time, so it survives redirects and DNS rebinding), the response
body is capped (25 MiB), and a 10s timeout applies.

**Local** — the final path segment is a filename under `./images`
(directory traversal is stripped):

```
/f_webp/q_80/r_320/123.png      →  ./images/123.png
```

## Run

```bash
go build -o imgproc .
./imgproc                 # listens on :8080, serves from ./images
```

```bash
# remote source
curl -o out.webp "http://localhost:8080/f_webp/q_80/r_320/?url=https://picsum.photos/1200/800.jpg"
# local source
curl -o out.webp http://localhost:8080/f_webp/q_80/r_320/123.png
```

## Responses

- `200` — transformed image (`Content-Type` set to the output format, `Cache-Control: public, max-age=86400`)
- `400` — malformed/unsupported token (`f_bmp`, `q_0`), bad `?url=` scheme, or a blocked (private) address
- `404` — local source image not found
- `422` — source could not be decoded/encoded
- `502` — remote origin unreachable or returned a non-200

## Layout

```
main.go              HTTP server + request handler (remote vs local routing)
transform/params.go  URL token → Options parser
transform/process.go libvips pipeline (resize → encode)
source/fetch.go      SSRF-guarded remote image fetcher
*/*_test.go          unit tests for parsing, fetching, and the real vips pipeline
images/              local source images
```

## Test

```bash
go test ./...
```
