package patcher

import (
	"encoding/binary"
	"fmt"
	"image"

	"podpal/internal/models"
)

// ImageToNative converts an image to Rockbox native bitmap format.
// The image must already be cropped and resized to logoW x logoH.
func ImageToNative(img image.Image, logoW, logoH int, format models.BmpFormat) ([]byte, error) {
	switch format {
	case models.FormatRGB565LE:
		return encodeRGB565(img, logoW, logoH, binary.LittleEndian), nil
	case models.FormatRGB565BE:
		return encodeRGB565(img, logoW, logoH, binary.BigEndian), nil
	case models.FormatGrey4:
		return encodeGrey4(img, logoW, logoH), nil
	default:
		return nil, fmt.Errorf("unsupported bmp format: %d", format)
	}
}

// NativeLogoSize returns the expected byte size of a native logo.
func NativeLogoSize(logoW, logoH int, format models.BmpFormat) int {
	switch format {
	case models.FormatRGB565LE, models.FormatRGB565BE:
		return logoW * logoH * 2
	case models.FormatGrey4:
		dstW := (logoW + 3) / 4
		return dstW * logoH * 2
	default:
		return 0
	}
}

func encodeRGB565(img image.Image, w, h int, order binary.ByteOrder) []byte {
	buf := make([]byte, w*h*2)
	bounds := img.Bounds()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// RGBA() returns 16-bit values, shift to 8-bit
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			rgb565 := uint16(r8>>3)<<11 | uint16(g8>>2)<<5 | uint16(b8>>3)
			offset := (y*w + x) * 2
			order.PutUint16(buf[offset:], rgb565)
		}
	}
	return buf
}

func encodeGrey4(img image.Image, w, h int) []byte {
	dstW := (w + 3) / 4
	dst := make([]uint16, dstW*h)
	bounds := img.Bounds()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			brightness := (3*uint16(r8) + 6*uint16(g8) + uint16(b8)) / 10
			// Bitwise NOT on uint8, then mask top 2 bits, shift by pixel position
			grey2bit := uint16((^uint8(brightness) & 0xC0) >> (2 * (x & 3)))
			dst[y*dstW+x/4] |= grey2bit
		}
	}
	buf := make([]byte, dstW*h*2)
	for i, v := range dst {
		binary.LittleEndian.PutUint16(buf[i*2:], v)
	}
	return buf
}
