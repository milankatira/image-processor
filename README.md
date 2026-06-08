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
| `f_*`  | output format      | `jpeg`/`jpg`, `png`, `webp`, `avif` | yes (keeps source format) |
| `q_*`  | encode quality     | integer `1`–`100`             | yes (encoder default) |
| `r_*`  | resize width (px)  | positive integer              | yes (no resize) |

Tokens may appear in any order and are all optional — `/123.png` returns the
original. The final path segment is always the source filename, resolved inside
`./images` (directory traversal is stripped).

## Run

```bash
go build -o imgproc .
./imgproc                 # listens on :8080, serves from ./images
```

```bash
curl -o out.webp http://localhost:8080/f_webp/q_80/r_320/123.png
```

## Responses

- `200` — transformed image (`Content-Type` set to the output format, `Cache-Control: public, max-age=86400`)
- `400` — malformed/unsupported token (e.g. `f_bmp`, `q_0`)
- `404` — source image not found
- `422` — source could not be decoded/encoded

## Layout

```
main.go              HTTP server + request handler
transform/params.go  URL token → Options parser
transform/process.go libvips pipeline (resize → encode)
transform/*_test.go  unit tests for parsing and the real vips pipeline
images/              source images
```

## Test

```bash
go test ./...
```
