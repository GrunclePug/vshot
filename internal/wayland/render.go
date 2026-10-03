package wayland

import (
	"errors"
	"image"
	"math"

	"github.com/grunclepug/vshot/internal/config"
	"github.com/grunclepug/vshot/internal/ui"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	barTextX    = 8
	barBaseline = 16
)

// view is the overlay on one output. It draws in software into shared
// memory buffers the size of the output's capture, which the viewport
// scales to the output's logical size.
type view struct {
	s                        *session
	out                      *output
	surface, layer, viewport uint32
	w, h                     int // logical size
	px, py                   float64
	app                      *ui.App
	rect                     image.Rectangle // area of app.Image shown here
	bright, dim              []byte          // rect in buffer byte order (B, G, R, X)
	scale                    int             // integer UI scale for decorations
	bufs                     [2]*shmBuffer
	dirty, waiting           bool
}

func (v *view) handleLayer(op uint16, m *msg) {
	switch op {
	case 0: // configure
		serial, w, h := m.uint(), m.uint(), m.uint()
		v.s.c.send(v.layer, 6, serial) // ack_configure
		v.w, v.h = int(w), int(h)
		v.s.c.send(v.viewport, 2, int32(w), int32(h)) // set_destination
		v.dirty = true
	case 1: // closed
		v.s.err = errors.New("overlay closed by the compositor")
	}
}

func (v *view) setApp(app *ui.App, rect image.Rectangle) {
	v.app, v.rect = app, rect
	v.scale = max(1, int(math.Round(float64(rect.Dx())/float64(v.w))))
	app.State.Scale = v.scale
	v.bright = toBGRX(app.Image, rect)
	v.dim = toBGRX(ui.Dimmed(app.Image.SubImage(rect).(*image.RGBA)), rect)
	v.dirty = true
}

// appPoint maps the last pointer position to app image coordinates.
func (v *view) appPoint() image.Point {
	x := int(v.px * float64(v.rect.Dx()) / float64(v.w))
	y := int(v.py * float64(v.rect.Dy()) / float64(v.h))
	return v.rect.Min.Add(image.Pt(x, y))
}

func (v *view) onBar() bool {
	return v.appPoint().Y-v.rect.Min.Y >= v.rect.Dy()-ui.BarHeight*v.scale
}

// draw renders and commits a frame if one is needed and the compositor is
// ready for it.
func (v *view) draw() error {
	if !v.dirty || v.waiting || v.app == nil {
		return nil
	}
	var buf *shmBuffer
	for i, b := range v.bufs {
		if b == nil {
			nb, err := newShmBuffer(v.s.c, v.s.shm, v.rect.Dx(), v.rect.Dy(), v.rect.Dx()*4, formatXRGB8888)
			if err != nil {
				return err
			}
			v.bufs[i], b = nb, nb
		}
		if !b.busy {
			buf = b
			break
		}
	}
	if buf == nil {
		return nil // wait for a release
	}

	v.paint(buf)

	c := v.s.c
	buf.busy = true
	v.dirty, v.waiting = false, true
	cb := c.newID(func(uint16, *msg) { v.waiting = false })
	c.send(v.surface, 1, buf.id, int32(0), int32(0))                              // attach
	c.send(v.surface, 9, int32(0), int32(0), int32(buf.width), int32(buf.height)) // damage_buffer
	c.send(v.surface, 3, cb)                                                      // frame
	return c.send(v.surface, 6)                                                   // commit
}

func (v *view) paint(buf *shmBuffer) {
	c := canvas{buf.data, buf.stride, buf.width, buf.height}
	copy(c.pix, v.dim)

	st := v.app.State
	if !st.Selection.Empty() {
		sel := st.Selection.Sub(v.rect.Min) // may extend past this output
		if vis := sel.Intersect(c.bounds()); !vis.Empty() {
			for y := vis.Min.Y; y < vis.Max.Y; y++ {
				i, j := y*c.stride+vis.Min.X*4, y*c.stride+vis.Max.X*4
				copy(c.pix[i:j], v.bright[i:j])
			}
		}

		k := v.scale
		c.fill(image.Rect(sel.Min.X, sel.Min.Y, sel.Max.X, sel.Min.Y+k), config.BorderColor)
		c.fill(image.Rect(sel.Min.X, sel.Max.Y-k, sel.Max.X, sel.Max.Y), config.BorderColor)
		c.fill(image.Rect(sel.Min.X, sel.Min.Y, sel.Min.X+k, sel.Max.Y), config.BorderColor)
		c.fill(image.Rect(sel.Max.X-k, sel.Min.Y, sel.Max.X, sel.Max.Y), config.BorderColor)
		for _, p := range ui.HandlePoints(sel) {
			c.circle(p, config.DotSize*k, config.BorderColor)
		}
	}

	k := v.scale
	barY := c.h - ui.BarHeight*k
	c.fill(image.Rect(0, barY, c.w, c.h), config.BarColor)
	c.text(st.StatusText(), barTextX*k, barY, k)
}

// canvas draws into a B, G, R, X pixel buffer.
type canvas struct {
	pix    []byte
	stride int
	w, h   int
}

func (c canvas) bounds() image.Rectangle { return image.Rect(0, 0, c.w, c.h) }

func (c canvas) set(x, y int, rgb uint32) {
	i := y*c.stride + x*4
	c.pix[i], c.pix[i+1], c.pix[i+2], c.pix[i+3] = byte(rgb), byte(rgb>>8), byte(rgb>>16), 0xff
}

func (c canvas) fill(r image.Rectangle, rgb uint32) {
	r = r.Intersect(c.bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c.set(x, y, rgb)
		}
	}
}

func (c canvas) circle(center image.Point, radius int, rgb uint32) {
	r2 := radius * radius
	b := image.Rect(center.X-radius, center.Y-radius, center.X+radius+1, center.Y+radius+1).Intersect(c.bounds())
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if dx, dy := x-center.X, y-center.Y; dx*dx+dy*dy <= r2 {
				c.set(x, y, rgb)
			}
		}
	}
}

// text draws s with a bitmap font scaled by k, in a bar whose top is barY.
func (c canvas) text(s string, x, barY, k int) {
	mask := image.NewAlpha(image.Rect(0, 0, (c.w-x)/k, ui.BarHeight))
	d := font.Drawer{Dst: mask, Src: image.Opaque, Face: basicfont.Face7x13, Dot: fixed.P(0, barBaseline)}
	d.DrawString(s)
	for my := range mask.Rect.Dy() {
		for mx := range mask.Rect.Dx() {
			if mask.AlphaAt(mx, my).A >= 0x80 {
				c.fill(image.Rect(x+mx*k, barY+my*k, x+(mx+1)*k, barY+(my+1)*k), config.TextColor)
			}
		}
	}
}

// toBGRX converts the rect area of img to B, G, R, X bytes, rect.Dx()*4 per row.
func toBGRX(img *image.RGBA, rect image.Rectangle) []byte {
	stride := rect.Dx() * 4
	out := make([]byte, stride*rect.Dy())
	for y := range rect.Dy() {
		src := img.Pix[img.PixOffset(rect.Min.X, rect.Min.Y+y):]
		dst := out[y*stride : (y+1)*stride]
		for i := 0; i < stride; i += 4 {
			dst[i], dst[i+1], dst[i+2], dst[i+3] = src[i+2], src[i+1], src[i], 0xff
		}
	}
	return out
}
