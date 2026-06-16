package ui

import (
	"github.com/grunclepug/vshot/internal/config"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type BarRenderer struct {
	Height int
	FontID xproto.Font
}

func NewBarRenderer() *BarRenderer {
	return &BarRenderer{Height: 24}
}

func (b *BarRenderer) DrawNative(X *xgb.Conn, win xproto.Window, gc xproto.Gcontext, state *State, barY int, width int) {
	text := "-- NORMAL --"
	if state.ErrorMessage != "" {
		text = state.ErrorMessage
	} else if state.Mode == ModeCommand {
		text = ":" + state.CommandBuffer
	} else if state.HasSelection {
		text = "-- VISUAL AREA --"
	}

	xproto.ChangeGC(X, gc, xproto.GcForeground|xproto.GcBackground, []uint32{config.BarColor, config.BarColor})
	xproto.PolyFillRectangle(X, xproto.Drawable(win), gc, []xproto.Rectangle{
		{X: 0, Y: int16(barY), Width: uint16(width), Height: uint16(b.Height)},
	})

	xproto.ChangeGC(X, gc, xproto.GcForeground, []uint32{config.TextColor})
	xproto.ImageText8(X, uint8(len(text)), xproto.Drawable(win), gc, 8, int16(barY+16), text)
}
