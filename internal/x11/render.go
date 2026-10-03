package x11

import (
	"image"

	"github.com/grunclepug/vshot/internal/config"
	"github.com/grunclepug/vshot/internal/ui"
	"github.com/jezek/xgb/xproto"
)

const (
	barTextX    = 8
	barBaseline = 16
)

// renderer composes frames server-side: the screenshot is uploaded once in
// a bright and a dimmed copy, and each frame is assembled from those with
// CopyArea into a back buffer, so redraws send no pixel data.
type renderer struct {
	s                 *Session
	bright, dim, back xproto.Pixmap
	barY              int
}

func newRenderer(s *Session, img *image.RGBA) (*renderer, error) {
	bright, err := s.UploadImage(img)
	if err != nil {
		return nil, err
	}
	dim, err := s.UploadImage(ui.Dimmed(img, config.X11Theme))
	if err != nil {
		return nil, err
	}
	back, err := s.NewPixmap()
	if err != nil {
		return nil, err
	}
	return &renderer{s: s, bright: bright, dim: dim, back: back, barY: s.Height - ui.BarHeight}, nil
}

func (r *renderer) draw(st *ui.State) {
	X, gc, back := r.s.X, r.s.GC, xproto.Drawable(r.back)
	theme := config.X11Theme

	xproto.CopyArea(X, xproto.Drawable(r.dim), back, gc, 0, 0, 0, 0, uint16(r.s.Width), uint16(r.s.Height))

	if sel := st.Selection; !sel.Empty() {
		x, y, w, h := int16(sel.Min.X), int16(sel.Min.Y), uint16(sel.Dx()), uint16(sel.Dy())
		xproto.CopyArea(X, xproto.Drawable(r.bright), back, gc, x, y, x, y, w, h)

		xproto.ChangeGC(X, gc, xproto.GcForeground, []uint32{theme.Border})
		var edges []xproto.Rectangle
		for _, e := range ui.BorderRects(sel, theme.BorderWidth) {
			if e = e.Intersect(sel); !e.Empty() {
				edges = append(edges, xproto.Rectangle{X: int16(e.Min.X), Y: int16(e.Min.Y), Width: uint16(e.Dx()), Height: uint16(e.Dy())})
			}
		}
		xproto.PolyFillRectangle(X, back, gc, edges)

		rad := config.DotSize
		arcs := make([]xproto.Arc, 0, 8)
		for _, c := range ui.HandlePoints(sel) {
			arcs = append(arcs, xproto.Arc{
				X: int16(c.X - rad), Y: int16(c.Y - rad),
				Width: uint16(2 * rad), Height: uint16(2 * rad),
				Angle2: 360 * 64,
			})
		}
		xproto.PolyFillArc(X, back, gc, arcs)
	}

	text := st.StatusText()
	if len(text) > 255 {
		text = text[:255]
	}
	xproto.ChangeGC(X, gc, xproto.GcForeground|xproto.GcBackground, []uint32{theme.Bar, theme.Bar})
	xproto.PolyFillRectangle(X, back, gc, []xproto.Rectangle{
		{X: 0, Y: int16(r.barY), Width: uint16(r.s.Width), Height: ui.BarHeight},
	})
	xproto.ChangeGC(X, gc, xproto.GcForeground, []uint32{theme.Text})
	xproto.ImageText8(X, byte(len(text)), back, gc, barTextX, int16(r.barY+barBaseline), text)

	r.present()
}

// present copies the back buffer to the window.
func (r *renderer) present() {
	xproto.CopyArea(r.s.X, xproto.Drawable(r.back), xproto.Drawable(r.s.Win), r.s.GC,
		0, 0, 0, 0, uint16(r.s.Width), uint16(r.s.Height))
}
