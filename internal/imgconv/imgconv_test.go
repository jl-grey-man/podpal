package imgconv

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestCropAndResize_Wide(t *testing.T) {
	// 400x100 source → 320x98 target (wider source, should crop sides)
	src := image.NewRGBA(image.Rect(0, 0, 400, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 400; x++ {
			src.Set(x, y, color.RGBA{R: 128, G: 64, B: 32, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}

	result, err := CropAndResize(&buf, 320, 98)
	if err != nil {
		t.Fatal(err)
	}

	bounds := result.Bounds()
	if bounds.Dx() != 320 || bounds.Dy() != 98 {
		t.Errorf("expected 320x98, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestCropAndResize_Tall(t *testing.T) {
	// 100x400 source → 320x98 target (taller source, should crop top/bottom)
	src := image.NewRGBA(image.Rect(0, 0, 100, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 100; x++ {
			src.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}

	result, err := CropAndResize(&buf, 320, 98)
	if err != nil {
		t.Fatal(err)
	}

	bounds := result.Bounds()
	if bounds.Dx() != 320 || bounds.Dy() != 98 {
		t.Errorf("expected 320x98, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestCropAndResize_ExactRatio(t *testing.T) {
	// Same aspect ratio — no cropping needed
	src := image.NewRGBA(image.Rect(0, 0, 640, 196))
	result := CropAndResizeImage(src, 320, 98)
	bounds := result.Bounds()
	if bounds.Dx() != 320 || bounds.Dy() != 98 {
		t.Errorf("expected 320x98, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}
