// Package config holds compile-time settings. Edit and rebuild to change them.
package config

const (
	BorderColor     uint32  = 0xFF00FF // selection border and handles, 0xRRGGBB
	BarColor        uint32  = 0x222222 // status bar background
	TextColor       uint32  = 0xbbbbbb // status bar text
	MaskAlpha       float64 = 0.5      // darkening outside the selection, 0..1
	DotSize         int     = 4        // grab handle radius in pixels
	DefaultDir              = "~/Pictures/Screenshots"
	TimestampFormat         = "Screenshot_2006-01-02_150405.png" // Go time layout for :w with no name
)
