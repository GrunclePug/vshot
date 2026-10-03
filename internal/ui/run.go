// Package ui implements the interactive selection overlay.
package ui

import (
	"errors"
	"fmt"
	"image"
	"os"

	"github.com/grunclepug/vshot/internal/x11"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type app struct {
	s    *x11.Session
	r    *renderer
	st   *State
	img  *image.RGBA
	quit bool
}

// Run shows img in the session window and handles input until the user
// quits.
func Run(s *x11.Session, img *image.RGBA) error {
	r, err := newRenderer(s, img)
	if err != nil {
		return err
	}
	a := &app{s: s, r: r, st: NewState(img.Bounds()), img: img}

	r.draw(a.st)
	if err := s.Map(); err != nil {
		return err
	}

	for !a.quit {
		ev, err := s.X.WaitForEvent()
		if ev == nil && err == nil {
			return errors.New("X connection closed")
		}

		// Handle everything queued before redrawing once, so fast pointer
		// motion doesn't back up behind redraws.
		dirty := false
		for !a.quit && (ev != nil || err != nil) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "vshot: X error: %v\n", err)
			} else if a.handle(ev) {
				dirty = true
			}
			ev, err = s.X.PollForEvent()
		}
		if dirty && !a.quit {
			r.draw(a.st)
		}
	}
	return nil
}

// handle processes one event and reports whether a redraw is needed.
func (a *app) handle(ev xgb.Event) bool {
	switch e := ev.(type) {
	case xproto.ExposeEvent:
		if e.Count == 0 {
			a.r.present()
		}
	case xproto.ButtonPressEvent:
		if e.Detail == xproto.ButtonIndex1 && int(e.EventY) < a.r.barY {
			a.st.Press(image.Pt(int(e.EventX), int(e.EventY)))
			return true
		}
	case xproto.MotionNotifyEvent:
		return a.st.Drag(image.Pt(int(e.EventX), int(e.EventY)))
	case xproto.ButtonReleaseEvent:
		if e.Detail == xproto.ButtonIndex1 {
			a.st.Release()
		}
	case xproto.KeyPressEvent:
		a.key(a.s.Keysym(e.Detail, e.State))
		return true
	}
	return false
}
