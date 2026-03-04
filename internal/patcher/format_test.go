package patcher

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"

	"podpal/internal/models"
)

func solidImage(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestRGB565LE_Red(t *testing.T) {
	img := solidImage(1, 1, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	data, err := ImageToNative(img, 1, 1, models.FormatRGB565LE)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 bytes, got %d", len(data))
	}
	val := binary.LittleEndian.Uint16(data)
	// Pure red in RGB565: 11111_000000_00000 = 0xF800
	if val != 0xF800 {
		t.Errorf("expected 0xF800, got 0x%04X", val)
	}
}

func TestRGB565BE_Green(t *testing.T) {
	img := solidImage(1, 1, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	data, err := ImageToNative(img, 1, 1, models.FormatRGB565BE)
	if err != nil {
		t.Fatal(err)
	}
	val := binary.BigEndian.Uint16(data)
	// Pure green in RGB565: 00000_111111_00000 = 0x07E0
	if val != 0x07E0 {
		t.Errorf("expected 0x07E0, got 0x%04X", val)
	}
}

func TestRGB565LE_White(t *testing.T) {
	img := solidImage(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	data, err := ImageToNative(img, 1, 1, models.FormatRGB565LE)
	if err != nil {
		t.Fatal(err)
	}
	val := binary.LittleEndian.Uint16(data)
	if val != 0xFFFF {
		t.Errorf("expected 0xFFFF, got 0x%04X", val)
	}
}

func TestNativeLogoSize(t *testing.T) {
	// iPod Video: 320x98 RGB565 = 62720 bytes
	size := NativeLogoSize(320, 98, models.FormatRGB565LE)
	if size != 62720 {
		t.Errorf("expected 62720, got %d", size)
	}

	// iPod Mini: 138x46 greyscale = ((138+3)/4) * 46 * 2 = 35*46*2 = 3220
	size = NativeLogoSize(138, 46, models.FormatGrey4)
	expected := ((138 + 3) / 4) * 46 * 2
	if size != expected {
		t.Errorf("expected %d, got %d", expected, size)
	}
}

func TestGrey4_Black(t *testing.T) {
	// Black pixel: brightness=0, ~0 & 0xC0 = 0xFF & 0xC0 = 0xC0
	// For x=0: shift by 0, grey2bit = 0xC0
	img := solidImage(1, 1, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	data, err := ImageToNative(img, 1, 1, models.FormatGrey4)
	if err != nil {
		t.Fatal(err)
	}
	// dstW = (1+3)/4 = 1, output = 1*1*2 = 2 bytes
	if len(data) != 2 {
		t.Fatalf("expected 2 bytes, got %d", len(data))
	}
	val := binary.LittleEndian.Uint16(data)
	if val != 0x00C0 {
		t.Errorf("expected 0x00C0, got 0x%04X", val)
	}
}

func TestGrey4_White(t *testing.T) {
	// White pixel: brightness=255, ~255 & 0xC0 = 0x00 & 0xC0 = 0x00
	img := solidImage(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	data, err := ImageToNative(img, 1, 1, models.FormatGrey4)
	if err != nil {
		t.Fatal(err)
	}
	val := binary.LittleEndian.Uint16(data)
	if val != 0x0000 {
		t.Errorf("expected 0x0000, got 0x%04X", val)
	}
}
