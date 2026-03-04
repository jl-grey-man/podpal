package models

import "fmt"

// BmpFormat represents the Rockbox native bitmap format.
type BmpFormat int

const (
	FormatRGB565LE BmpFormat = 4 // Little-endian RGB565 (iPod Video, Classic, Nano 2G)
	FormatRGB565BE BmpFormat = 5 // Big-endian RGB565 (iPod Color, Nano 1G)
	FormatGrey4    BmpFormat = 6 // Greyscale 4-grey, 2bpp column-packed (iPod 1G-4G, Mini)
)

// IPod holds per-model configuration derived from Rockbox source.
type IPod struct {
	Key         string    // lookup key (e.g. "ipodvideo")
	ModelID     string    // 4-byte string in .ipod header
	ModelNum    uint32    // checksum seed
	LCDWidth    int       // screen width
	LCDHeight   int       // screen height
	LogoWidth   int       // boot logo width
	LogoHeight  int       // boot logo height
	BmpFormat   BmpFormat // native pixel format
	LCDDepth    int       // bits per pixel (16 or 2)
	Description string    // human-readable name
}

// All returns all supported iPod models in display order.
func All() []IPod {
	return []IPod{
		{
			Key: "ipodvideo", ModelID: "ipvd", ModelNum: 5,
			LCDWidth: 320, LCDHeight: 240, LogoWidth: 320, LogoHeight: 98,
			BmpFormat: FormatRGB565LE, LCDDepth: 16,
			Description: "iPod Video (5th/5.5th Gen)",
		},
		{
			Key: "ipod6g", ModelID: "ip6g", ModelNum: 71,
			LCDWidth: 320, LCDHeight: 240, LogoWidth: 320, LogoHeight: 98,
			BmpFormat: FormatRGB565LE, LCDDepth: 16,
			Description: "iPod Classic (6th/6.5th/7th Gen)",
		},
		{
			Key: "ipodnano2g", ModelID: "nn2g", ModelNum: 62,
			LCDWidth: 176, LCDHeight: 132, LogoWidth: 176, LogoHeight: 54,
			BmpFormat: FormatRGB565LE, LCDDepth: 16,
			Description: "iPod Nano 2nd Gen",
		},
		{
			Key: "ipodcolor", ModelID: "ipcl", ModelNum: 8,
			LCDWidth: 220, LCDHeight: 176, LogoWidth: 220, LogoHeight: 68,
			BmpFormat: FormatRGB565BE, LCDDepth: 16,
			Description: "iPod Color/Photo",
		},
		{
			Key: "ipodnano1g", ModelID: "nano", ModelNum: 4,
			LCDWidth: 176, LCDHeight: 132, LogoWidth: 176, LogoHeight: 54,
			BmpFormat: FormatRGB565BE, LCDDepth: 16,
			Description: "iPod Nano 1st Gen",
		},
		{
			Key: "ipod3g", ModelID: "ip3g", ModelNum: 7,
			LCDWidth: 160, LCDHeight: 128, LogoWidth: 160, LogoHeight: 53,
			BmpFormat: FormatGrey4, LCDDepth: 2,
			Description: "iPod 3rd Gen",
		},
		{
			Key: "ipod4g", ModelID: "ip4g", ModelNum: 9,
			LCDWidth: 160, LCDHeight: 128, LogoWidth: 160, LogoHeight: 53,
			BmpFormat: FormatGrey4, LCDDepth: 2,
			Description: "iPod 4th Gen (greyscale)",
		},
		{
			Key: "ipodmini1g", ModelID: "mini", ModelNum: 3,
			LCDWidth: 138, LCDHeight: 110, LogoWidth: 138, LogoHeight: 46,
			BmpFormat: FormatGrey4, LCDDepth: 2,
			Description: "iPod Mini 1st Gen",
		},
		{
			Key: "ipodmini2g", ModelID: "mn2g", ModelNum: 11,
			LCDWidth: 138, LCDHeight: 110, LogoWidth: 138, LogoHeight: 46,
			BmpFormat: FormatGrey4, LCDDepth: 2,
			Description: "iPod Mini 2nd Gen",
		},
		{
			Key: "ipod1g2g", ModelID: "1g2g", ModelNum: 19,
			LCDWidth: 160, LCDHeight: 128, LogoWidth: 160, LogoHeight: 53,
			BmpFormat: FormatGrey4, LCDDepth: 2,
			Description: "iPod 1st/2nd Gen",
		},
	}
}

// ByKey returns the model with the given key, or nil if not found.
func ByKey(key string) *IPod {
	for _, m := range All() {
		if m.Key == key {
			return &m
		}
	}
	return nil
}

// BmpFilename returns the expected reference BMP filename for this model.
func (m *IPod) BmpFilename() string {
	return fmt.Sprintf("rockboxlogo.%dx%dx%d.bmp", m.LogoWidth, m.LogoHeight, m.LCDDepth)
}
