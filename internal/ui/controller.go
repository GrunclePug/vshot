package ui

import (
	"fmt"
	"image"
	"os"

	"github.com/grunclepug/vshot/internal/config"
	"github.com/grunclepug/vshot/internal/x11"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func RunEventLoop(session *x11.Session, state *State, canvas *Canvas, renderer *Renderer) {
	for {
		ev, err := session.X.WaitForEvent()
		if err != nil {
			fmt.Fprintf(os.Stderr, "X11 Connection Error: %v\n", err)
			break
		}
		if ev == nil {
			break
		}

		if xerr, ok := ev.(xgb.Error); ok {
			fmt.Fprintf(os.Stderr, "X11 Protocol Error: %v\n", xerr)
			continue
		}

		// Local redraw closure to capture required dependencies
		redraw := func() { renderer.Redraw(state, canvas) }

		switch e := ev.(type) {
		case xproto.ExposeEvent:
			if e.Count == 0 {
				redraw()
			}
		case xproto.ButtonPressEvent:
			mousePt := image.Pt(int(e.EventX), int(e.EventY))
			if mousePt.Y < renderer.barY {
				handle := state.GetHandleAt(mousePt, config.DotSize)
				state.ActiveHandle = handle
				state.DragStart = mousePt
				if handle == HandleNone {
					state.Selection = image.Rect(mousePt.X, mousePt.Y, mousePt.X, mousePt.Y)
					state.InitialRect = state.Selection
					state.ActiveHandle = HandleBottomRight
				} else {
					state.InitialRect = state.Selection
				}
				state.HasSelection = true
				redraw()
			}
		case xproto.ButtonReleaseEvent:
			state.ActiveHandle = HandleNone
		case xproto.MotionNotifyEvent:
			if state.ActiveHandle != HandleNone {
				state.UpdateSelectionFromDrag(image.Pt(int(e.EventX), int(e.EventY)))
				// Poll for more motion events to keep it smooth
				ev, err := session.X.PollForEvent()
				if ev != nil || err != nil {
					continue
				}
				redraw()
			}
		case xproto.KeyPressEvent:
			HandleKeyPress(session.X, e, state, redraw)
		}
	}
}
