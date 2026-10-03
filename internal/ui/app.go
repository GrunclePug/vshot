// Package ui implements vshot's display-independent behavior: selection,
// keys, commands and output. Backends own the window, event loop and
// drawing, and feed input into an App.
package ui

import (
	"image"

	"github.com/grunclepug/vshot/internal/config"
)

// BarHeight is the height of the status bar along the bottom edge.
const BarHeight = 24

// Keysym is an X keysym. Wayland's xkbcommon uses the same values.
type Keysym uint32

const (
	KeyBackSpace Keysym = 0xff08
	KeyReturn    Keysym = 0xff0d
	KeyEscape    Keysym = 0xff1b
	KeyKPEnter   Keysym = 0xff8d
)

type App struct {
	State *State
	Image *image.RGBA

	clipboard []string
	quit      bool
}

// NewApp creates an App for a captured image. clipboard is the command
// that reads PNG data on stdin and places it on the clipboard.
func NewApp(img *image.RGBA, clipboard []string) *App {
	return &App{State: NewState(img.Bounds()), Image: img, clipboard: clipboard}
}

// Done reports whether the user has asked to quit.
func (a *App) Done() bool { return a.quit }

// Press handles a primary button press at p and reports whether a redraw is
// needed. Presses on the status bar are ignored.
func (a *App) Press(p image.Point) bool {
	if p.Y >= a.Image.Rect.Max.Y-BarHeight {
		return false
	}
	a.State.Press(p)
	return true
}

// Drag handles pointer motion and reports whether a redraw is needed.
func (a *App) Drag(p image.Point) bool { return a.State.Drag(p) }

// Release handles a primary button release.
func (a *App) Release() { a.State.Release() }

// Dimmed returns a copy of img darkened by config.MaskAlpha, for drawing
// the area outside the selection.
func Dimmed(img *image.RGBA) *image.RGBA {
	f := uint32((1 - min(max(config.MaskAlpha, 0), 1)) * 256)
	out := image.NewRGBA(img.Rect)
	for i, v := range img.Pix {
		out.Pix[i] = uint8(uint32(v) * f >> 8)
	}
	return out
}
