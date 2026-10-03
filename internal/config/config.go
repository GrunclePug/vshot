// Package config holds compile-time settings. Edit and rebuild to change them.
package config

const (
	DotSize         int = 4 // grab handle radius in pixels
	DefaultDir          = "~/Pictures/Screenshots"
	TimestampFormat     = "Screenshot_2006-01-02_150405.png" // Go time layout for :w with no name
)

// Theme is the overlay's look. Colors are 0xRRGGBB.
type Theme struct {
	Border      uint32  // selection border and handles
	BorderWidth int     // selection border width in pixels
	Bar         uint32  // status bar background
	Text        uint32  // status bar text
	Mask        uint32  // color blended over everything outside the selection
	MaskAlpha   float64 // strength of that blend, 0..1
}

// Each backend has its own theme, so vshot can match the desktop it runs in.
var (
	// dwm's default scheme.
	X11Theme = Theme{
		Border: 0xFF00FF, BorderWidth: 1,
		Bar: 0x222222, Text: 0xbbbbbb,
		Mask: 0x000000, MaskAlpha: 0.5,
	}

	// river profile palette: teal accent, base background, 2px borders.
	WaylandTheme = Theme{
		Border: 0x5fd4c4, BorderWidth: 2,
		Bar: 0x0e141a, Text: 0xe6edf0,
		Mask: 0x0e141a, MaskAlpha: 0.5,
	}
)
