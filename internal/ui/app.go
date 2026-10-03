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
// needed. Backends ignore presses on their status bar before calling it.
func (a *App) Press(p image.Point) bool {
	a.State.Press(p)
	return true
}

// Drag handles pointer motion and reports whether a redraw is needed.
func (a *App) Drag(p image.Point) bool { return a.State.Drag(p) }

// Release handles a primary button release.
func (a *App) Release() { a.State.Release() }

// Dimmed returns a copy of img with the theme's mask color blended over it,
// for drawing the area outside the selection.
func Dimmed(img *image.RGBA, t config.Theme) *image.RGBA {
	a := uint32(min(max(t.MaskAlpha, 0), 1) * 256)
	mask := [4]uint32{(t.Mask >> 16 & 0xff) * a, (t.Mask >> 8 & 0xff) * a, (t.Mask & 0xff) * a, 0xff * a}
	out := image.NewRGBA(img.Rect)
	n := img.Rect.Dx() * 4
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		src := img.Pix[img.PixOffset(img.Rect.Min.X, y):][:n]
		dst := out.Pix[out.PixOffset(img.Rect.Min.X, y):][:n]
		for i, v := range src {
			dst[i] = uint8((uint32(v)*(256-a) + mask[i%4]) >> 8)
		}
	}
	return out
}

// BorderRects returns the four edges of a border of width w inside r.
func BorderRects(r image.Rectangle, w int) [4]image.Rectangle {
	return [4]image.Rectangle{
		{r.Min, image.Pt(r.Max.X, r.Min.Y+w)},
		{image.Pt(r.Min.X, r.Max.Y-w), r.Max},
		{r.Min, image.Pt(r.Min.X+w, r.Max.Y)},
		{image.Pt(r.Max.X-w, r.Min.Y), r.Max},
	}
}
