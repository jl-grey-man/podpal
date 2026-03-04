# Podpal — Rockbox Boot Logo Patcher

Web app that lets users patch custom boot logos into Rockbox firmware for iPod.

**GitHub:** https://github.com/jl-grey-man/podpal

## Quick Start

```bash
go run main.go          # starts on :8080
go test ./...           # run all tests
```

## Tech Stack

- **Go** with `chi/v5` router
- **HTMX** (CDN) + `html/template`
- Reference BMPs embedded via `//go:embed`
- Rockbox zips downloaded on-demand, cached to `cache/`

## Project Structure

```
main.go                          # Server entry point
internal/
  models/ipod.go                 # iPod model definitions (10 models)
  patcher/
    format.go                    # RGB565 LE/BE + greyscale pixel conversion
    patcher.go                   # Checksum calc, logo search, patch pipeline
  downloader/downloader.go       # Download + cache Rockbox zips from rockbox.org
  imgconv/imgconv.go             # Load, center-crop, resize user images
web/handler.go                   # HTTP handlers (GET /, POST /patch, GET /download/{id})
assets/
  embed.go                       # //go:embed for reference BMPs
  bitmaps/                       # 5 reference Rockbox logo BMPs
templates/index.html             # Single-page HTMX UI
static/style.css                 # Minimal styling
tools/patch_rockbox_logo.py      # Original Python reference tool
cache/                           # (gitignored) Downloaded Rockbox zip cache
```

## Dependencies

- `github.com/go-chi/chi/v5` — HTTP router
- `golang.org/x/image` — BMP decoder

## How Patching Works

1. Download fresh `rockbox.ipod` from rockbox.org (cached)
2. Generate reference logo from embedded source BMP → native pixel format
3. Search firmware payload for reference logo byte pattern
4. Convert user's uploaded image to same native format (crop + resize + convert)
5. Replace logo bytes in-place
6. Recalculate Rockbox checksum (model_num + sum of all payload bytes)
7. Return ZIP with patched file + original backup + restore instructions

## Rockbox Binary Format

- Bytes 0-3: checksum (big-endian uint32) = model_number + sum(payload_bytes)
- Bytes 4-7: model ID string (e.g. "ipvd" for iPod Video)
- Bytes 8+: firmware payload

## Native Pixel Formats

- **Format 4**: Little-endian RGB565 (iPod Video, Classic, Nano 2G)
- **Format 5**: Big-endian RGB565 (iPod Color, Nano 1G)
- **Format 6**: Greyscale 4-grey, 2bpp column-packed (iPod 1G-4G, Mini)

## Fragile Areas

- **Greyscale format 6**: Bitwise NOT on brightness differs between Python (`~x` = arbitrary precision negative) and Go (`^uint8(x)` = proper uint8 NOT). Must use `^uint8(brightness)` in Go.
- **Logo search**: Depends on reference BMPs matching the Rockbox version exactly. If Rockbox updates their logos, the embedded BMPs must be updated too.
- **Checksum**: Must be recalculated after ANY byte change in the payload. Off-by-one in the payload slice will produce a wrong checksum.

## Environment Variables

None required. The `.env` file exists but is not used by the Go app.

## Commands

- `go run main.go` — start dev server on :8080
- `go test ./...` — run all tests
- `go build -o podpal .` — build binary
