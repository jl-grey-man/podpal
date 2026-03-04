package downloader

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFirmware(t *testing.T) {
	// Create a test zip with a rockbox.ipod file
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	f, err := w.Create(".rockbox/rockbox.ipod")
	if err != nil {
		t.Fatal(err)
	}
	firmware := []byte("test firmware data")
	if _, err := f.Write(firmware); err != nil {
		t.Fatal(err)
	}

	// Add a decoy file
	f2, err := w.Create(".rockbox/config.cfg")
	if err != nil {
		t.Fatal(err)
	}
	f2.Write([]byte("some config"))

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	d := New(t.TempDir())
	result, err := d.extractFirmware(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(result, firmware) {
		t.Errorf("expected %q, got %q", firmware, result)
	}
}

func TestExtractFromCache(t *testing.T) {
	// Create a cached zip
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create(".rockbox/rockbox.ipod")
	f.Write([]byte("cached firmware"))
	w.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.zip")
	os.WriteFile(path, buf.Bytes(), 0644)

	d := New(dir)
	result, err := d.extractFromCache(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != "cached firmware" {
		t.Errorf("expected 'cached firmware', got %q", result)
	}
}

func TestExtractFirmware_NotFound(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("other_file.txt")
	f.Write([]byte("not firmware"))
	w.Close()

	d := New(t.TempDir())
	_, err := d.extractFirmware(buf.Bytes())
	if err == nil {
		t.Error("expected error when rockbox.ipod not in zip")
	}
}
