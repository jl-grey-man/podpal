package downloader

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Downloader downloads and caches Rockbox firmware zips.
type Downloader struct {
	cacheDir string
	mu       sync.Map // per-model mutex to prevent duplicate downloads
}

// New creates a Downloader that caches zips in the given directory.
func New(cacheDir string) *Downloader {
	return &Downloader{cacheDir: cacheDir}
}

// rockboxURL returns the download URL for a given model key.
// Example: https://download.rockbox.org/release/3.15/rockbox-ipodvideo-3.15.zip
func rockboxURL(modelKey string) string {
	return fmt.Sprintf("https://download.rockbox.org/release/3.15/rockbox-%s-3.15.zip", modelKey)
}

// GetFirmware returns the rockbox.ipod bytes for a given model.
// Downloads and caches the zip if not already cached.
func (d *Downloader) GetFirmware(modelKey string) ([]byte, error) {
	// Per-model mutex
	muIface, _ := d.mu.LoadOrStore(modelKey, &sync.Mutex{})
	mu := muIface.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	// Check cache
	cachedPath := filepath.Join(d.cacheDir, modelKey+".zip")
	if data, err := d.extractFromCache(cachedPath); err == nil {
		return data, nil
	}

	// Download
	url := rockboxURL(modelKey)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", url, err)
	}

	// Save to cache
	if err := os.MkdirAll(d.cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	if err := os.WriteFile(cachedPath, zipData, 0644); err != nil {
		return nil, fmt.Errorf("write cache file: %w", err)
	}

	return d.extractFirmware(zipData)
}

func (d *Downloader) extractFromCache(path string) ([]byte, error) {
	zipData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return d.extractFirmware(zipData)
}

func (d *Downloader) extractFirmware(zipData []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	for _, f := range r.File {
		// Look for rockbox.ipod anywhere in the zip
		name := filepath.Base(f.Name)
		if strings.EqualFold(name, "rockbox.ipod") {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open %s in zip: %w", f.Name, err)
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}

	return nil, fmt.Errorf("rockbox.ipod not found in zip")
}
