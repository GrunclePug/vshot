package ui

import (
	"github.com/grunclepug/vshot/internal/x11"
	"github.com/jezek/xgb/xproto"
)

type Renderer struct {
	session *x11.Session
	bar     *BarRenderer
	barY    int
}

func NewRenderer(s *x11.Session, bar *BarRenderer) *Renderer {
	return &Renderer{
		session: s,
		bar:     bar,
		barY:    s.Height - bar.Height,
	}
}

func (r *Renderer) Redraw(state *State, canvas *Canvas) {
	bgBytes := canvas.RenderFrame(state, r.session.TargetBpp)
	bytesPerPixel := r.session.TargetBpp / 8
	rowStride := ((r.session.Width * bytesPerPixel) + 3) & ^3

	for y := 0; y < r.session.Height; y += 32 {
		chunkH := 32
		if y+chunkH > r.session.Height {
			chunkH = r.session.Height - y
		}
		xproto.PutImage(r.session.X, xproto.ImageFormatZPixmap, xproto.Drawable(r.session.Pixmap), r.session.GC, uint16(r.session.Width), uint16(chunkH), 0, int16(y), 0, r.session.Depth, bgBytes[y*rowStride:(y+chunkH)*rowStride])
	}

	r.bar.DrawNative(r.session.X, xproto.Window(r.session.Pixmap), r.session.GC, state, r.barY, r.session.Width)
	xproto.CopyArea(r.session.X, xproto.Drawable(r.session.Pixmap), xproto.Drawable(r.session.Win), r.session.GC, 0, 0, 0, 0, uint16(r.session.Width), uint16(r.session.Height))
}
