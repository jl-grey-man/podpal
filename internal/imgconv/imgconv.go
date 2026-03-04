package imgconv

import (
	"fmt"
	"image"
	"io"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"golang.org/x/image/draw"
)

// CropAndResize decodes an image from r, center-crops it to the target aspect
// ratio, and resizes to targetW x targetH using CatmullRom interpolation.
func CropAndResize(r io.Reader, targetW, targetH int) (image.Image, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	return CropAndResizeImage(src, targetW, targetH), nil
}

// CropAndResizeImage center-crops an image to the target aspect ratio,
// then resizes to targetW x targetH using CatmullRom interpolation.
func CropAndResizeImage(src image.Image, targetW, targetH int) image.Image {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	// Center-crop to target aspect ratio
	targetRatio := float64(targetW) / float64(targetH)
	srcRatio := float64(srcW) / float64(srcH)

	var cropRect image.Rectangle
	if srcRatio > targetRatio {
		// Source is wider — crop sides
		newW := int(float64(srcH) * targetRatio)
		left := (srcW - newW) / 2
		cropRect = image.Rect(bounds.Min.X+left, bounds.Min.Y, bounds.Min.X+left+newW, bounds.Max.Y)
	} else {
		// Source is taller — crop top/bottom
		newH := int(float64(srcW) / targetRatio)
		top := (srcH - newH) / 2
		cropRect = image.Rect(bounds.Min.X, bounds.Min.Y+top, bounds.Max.X, bounds.Min.Y+top+newH)
	}

	// Resize using CatmullRom
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, cropRect, draw.Over, nil)

	return dst
}
