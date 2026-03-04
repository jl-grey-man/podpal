package patcher

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"

	"podpal/assets"
	"podpal/internal/models"

	_ "golang.org/x/image/bmp"
)

// CalcChecksum computes the Rockbox firmware checksum.
// checksum = model_num + sum(all payload bytes), as uint32.
func CalcChecksum(payload []byte, modelNum uint32) uint32 {
	sum := modelNum
	for _, b := range payload {
		sum += uint32(b)
	}
	return sum
}

// GenerateReferenceLogo generates the reference logo from the embedded BMP.
func GenerateReferenceLogo(model *models.IPod) ([]byte, error) {
	bmpName := model.BmpFilename()
	f, err := assets.Bitmaps.Open("bitmaps/" + bmpName)
	if err != nil {
		return nil, fmt.Errorf("open embedded BMP %s: %w", bmpName, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode BMP %s: %w", bmpName, err)
	}

	return ImageToNative(img, model.LogoWidth, model.LogoHeight, model.BmpFormat)
}

// FindLogoInPayload searches for the reference logo in the firmware payload.
// Returns the offset into payload where the logo starts, or -1 if not found.
func FindLogoInPayload(payload, refLogo []byte) int {
	logoLen := len(refLogo)
	if logoLen < 64 {
		return -1
	}

	// Find a 64-byte all-nonzero chunk near the middle as search needle
	mid := logoLen / 2
	var needle []byte
	var needleOffset int
	found := false

	for start := mid; start < min(mid+2000, logoLen-64); start += 2 {
		chunk := refLogo[start : start+64]
		allNonZero := true
		for _, b := range chunk {
			if b == 0 {
				allNonZero = false
				break
			}
		}
		if allNonZero {
			needle = chunk
			needleOffset = start
			found = true
			break
		}
	}

	// Fallback: first 64 bytes with at least some non-zero
	if !found {
		for start := 0; start < logoLen-64; start += 2 {
			chunk := refLogo[start : start+64]
			hasNonZero := false
			for _, b := range chunk {
				if b != 0 {
					hasNonZero = true
					break
				}
			}
			if hasNonZero {
				needle = chunk
				needleOffset = start
				found = true
				break
			}
		}
	}

	if !found {
		return -1
	}

	pos := bytes.Index(payload, needle)
	if pos == -1 {
		return -1
	}

	logoStart := pos - needleOffset
	if logoStart < 0 || logoStart+logoLen > len(payload) {
		return -1
	}

	// Verify full match
	candidate := payload[logoStart : logoStart+logoLen]
	if bytes.Equal(candidate, refLogo) {
		return logoStart
	}

	// Partial match — accept if >95% matches
	matchCount := 0
	for i := range candidate {
		if candidate[i] == refLogo[i] {
			matchCount++
		}
	}
	if float64(matchCount)/float64(logoLen) > 0.95 {
		return logoStart
	}

	return -1
}

// PatchResult holds the result of a patching operation.
type PatchResult struct {
	Patched  []byte // the patched firmware binary
	Original []byte // the original firmware binary (unmodified copy)
}

// Patch takes a raw rockbox.ipod binary and a user image, and returns a patched binary.
func Patch(firmware []byte, userImg image.Image, model *models.IPod) (*PatchResult, error) {
	if len(firmware) < 8 {
		return nil, fmt.Errorf("firmware too small: %d bytes", len(firmware))
	}

	// Keep original copy
	original := make([]byte, len(firmware))
	copy(original, firmware)

	// Parse header
	storedChecksum := binary.BigEndian.Uint32(firmware[0:4])
	payload := firmware[8:]

	// Verify checksum
	calcSum := CalcChecksum(payload, model.ModelNum)
	if calcSum != storedChecksum {
		return nil, fmt.Errorf("checksum mismatch: stored=0x%08X calculated=0x%08X — firmware may be corrupted", storedChecksum, calcSum)
	}

	// Generate reference logo
	refLogo, err := GenerateReferenceLogo(model)
	if err != nil {
		return nil, fmt.Errorf("generate reference logo: %w", err)
	}

	// Find logo in payload
	logoOffset := FindLogoInPayload(payload, refLogo)
	if logoOffset < 0 {
		return nil, fmt.Errorf("could not find logo data in firmware — the Rockbox version may not match the reference BMPs")
	}

	// Convert user image to native format
	newLogo, err := ImageToNative(userImg, model.LogoWidth, model.LogoHeight, model.BmpFormat)
	if err != nil {
		return nil, fmt.Errorf("convert image: %w", err)
	}

	if len(newLogo) != len(refLogo) {
		return nil, fmt.Errorf("logo size mismatch: new=%d reference=%d", len(newLogo), len(refLogo))
	}

	// Patch the payload
	patched := make([]byte, len(firmware))
	copy(patched, firmware)
	copy(patched[8+logoOffset:], newLogo)

	// Recalculate checksum
	newChecksum := CalcChecksum(patched[8:], model.ModelNum)
	binary.BigEndian.PutUint32(patched[0:4], newChecksum)

	// Verify patched size hasn't changed
	if len(patched) != len(firmware) {
		return nil, fmt.Errorf("patched size %d != original %d", len(patched), len(firmware))
	}

	return &PatchResult{
		Patched:  patched,
		Original: original,
	}, nil
}
