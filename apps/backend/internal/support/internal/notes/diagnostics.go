package notes

import (
	"math"
	"strings"
	"unicode"
)

const (
	maxBuildLength     = 64
	maxUserAgentLength = 256
	maxGPULength       = 256
	maxWebGLLength     = 128
	maxViewportSide    = 100_000
	maxTextureSide     = 1 << 20
	maxPixelRatio      = 16
	maxCores           = 1024
	maxMemoryGB        = 1024
)

// What a note says of the page, the browser and the device it came from. Plain data, a part of every note's
// snapshot: the page wrote it, so a note keeps it cleaned and trusts none of it.
type Diagnostics struct {
	Build          string
	ViewportWidth  uint32
	ViewportHeight uint32
	PixelRatio     float64
	ConnectionDown bool
	UserAgent      string
	GPUVendor      string
	GPURenderer    string
	WebGLVersion   string
	MaxTextureSize uint32
	ContextLost    bool
	Cores          uint32
	MemoryGB       float64
}

func (d Diagnostics) cleaned() Diagnostics {
	return Diagnostics{
		Build:          printable(d.Build, maxBuildLength),
		ViewportWidth:  min(d.ViewportWidth, maxViewportSide),
		ViewportHeight: min(d.ViewportHeight, maxViewportSide),
		PixelRatio:     measured(d.PixelRatio, maxPixelRatio),
		ConnectionDown: d.ConnectionDown,
		UserAgent:      printable(d.UserAgent, maxUserAgentLength),
		GPUVendor:      printable(d.GPUVendor, maxGPULength),
		GPURenderer:    printable(d.GPURenderer, maxGPULength),
		WebGLVersion:   printable(d.WebGLVersion, maxWebGLLength),
		MaxTextureSize: min(d.MaxTextureSize, maxTextureSide),
		ContextLost:    d.ContextLost,
		Cores:          min(d.Cores, maxCores),
		MemoryGB:       measured(d.MemoryGB, maxMemoryGB),
	}
}

func printable(value string, maxRunes int) string {
	cleaned := []rune(strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.ToValidUTF8(value, ""))))

	if len(cleaned) > maxRunes {
		cleaned = cleaned[:maxRunes]
	}
	return strings.TrimSpace(string(cleaned))
}

// Two decimals, from 0 to the most: what is not a number, or is below 0, is 0.
func measured(value float64, most float64) float64 {
	if math.IsNaN(value) || value <= 0 {
		return 0
	}
	return math.Round(min(value, most)*100) / 100
}
