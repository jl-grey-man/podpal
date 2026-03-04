package patcher

import (
	"encoding/binary"
	"image/color"
	"testing"

	"podpal/internal/models"
)

func TestCalcChecksum(t *testing.T) {
	// Simple test: payload of [1, 2, 3], model_num = 5
	// Expected: 5 + 1 + 2 + 3 = 11
	payload := []byte{1, 2, 3}
	sum := CalcChecksum(payload, 5)
	if sum != 11 {
		t.Errorf("expected 11, got %d", sum)
	}
}

func TestCalcChecksum_Overflow(t *testing.T) {
	// Verify uint32 overflow wraps correctly
	payload := make([]byte, 256)
	for i := range payload {
		payload[i] = 0xFF
	}
	sum := CalcChecksum(payload, 0)
	// 256 * 255 = 65280
	if sum != 65280 {
		t.Errorf("expected 65280, got %d", sum)
	}
}

func TestGenerateReferenceLogo(t *testing.T) {
	model := models.ByKey("ipodvideo")
	if model == nil {
		t.Fatal("model not found")
	}

	logo, err := GenerateReferenceLogo(model)
	if err != nil {
		t.Fatal(err)
	}

	expected := NativeLogoSize(model.LogoWidth, model.LogoHeight, model.BmpFormat)
	if len(logo) != expected {
		t.Errorf("expected %d bytes, got %d", expected, len(logo))
	}
}

func TestFindLogoInPayload(t *testing.T) {
	model := models.ByKey("ipodvideo")
	if model == nil {
		t.Fatal("model not found")
	}

	logo, err := GenerateReferenceLogo(model)
	if err != nil {
		t.Fatal(err)
	}

	// Build a fake payload with the logo embedded at offset 1000
	payload := make([]byte, 100000)
	for i := range payload {
		payload[i] = byte(i % 251) // some non-trivial pattern
	}
	copy(payload[1000:], logo)

	offset := FindLogoInPayload(payload, logo)
	if offset != 1000 {
		t.Errorf("expected offset 1000, got %d", offset)
	}
}

func TestPatch_RoundTrip(t *testing.T) {
	model := models.ByKey("ipodvideo")
	if model == nil {
		t.Fatal("model not found")
	}

	logo, err := GenerateReferenceLogo(model)
	if err != nil {
		t.Fatal(err)
	}

	// Build a fake firmware: 4 bytes checksum + 4 bytes model ID + payload
	payload := make([]byte, 100000)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	copy(payload[1000:], logo)

	firmware := make([]byte, 8+len(payload))
	copy(firmware[4:8], []byte(model.ModelID))
	copy(firmware[8:], payload)

	// Calculate correct checksum
	checksum := CalcChecksum(payload, model.ModelNum)
	binary.BigEndian.PutUint32(firmware[0:4], checksum)

	// Patch with a solid red image
	userImg := solidImage(320, 98, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	result, err := Patch(firmware, userImg, model)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Patched) != len(firmware) {
		t.Errorf("patched size %d != original %d", len(result.Patched), len(firmware))
	}

	// Verify original is unchanged
	if !bytesEqual(result.Original, firmware) {
		t.Error("original was modified")
	}

	// Verify patched checksum is valid
	patchedChecksum := binary.BigEndian.Uint32(result.Patched[0:4])
	patchedCalc := CalcChecksum(result.Patched[8:], model.ModelNum)
	if patchedChecksum != patchedCalc {
		t.Errorf("patched checksum mismatch: stored=0x%08X calc=0x%08X", patchedChecksum, patchedCalc)
	}

	// Verify the logo area changed
	if bytesEqual(result.Patched[8+1000:8+1000+len(logo)], logo) {
		t.Error("logo area was not changed")
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
