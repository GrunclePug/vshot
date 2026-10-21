package x11

import (
	"errors"
	"fmt"
	"image"
	"os"

	"github.com/grunclepug/vshot/internal/ui"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// Clipboard is the command the X11 backend uses to copy PNG data.
var Clipboard = []string{"xclip", "-selection", "clipboard", "-t", "image/png"}

// Main runs vshot on the X server named by $DISPLAY.
func Main(all bool) error {
	X, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("connect to X server: %w", err)
	}
	defer X.Close()

	// Capture before any window of ours is mapped, so it can't appear in the shot.
	img, bounds, err := Capture(X, all)
	if err != nil {
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			return fmt.Errorf("capture: %w (XWayland can't read the screen; use -backend wayland)", err)
		}
		return fmt.Errorf("capture: %w", err)
	}

	s, err := NewSession(X, bounds)
	if err != nil {
		return err
	}
	return run(s, ui.NewApp(img, Clipboard))
}

func run(s *Session, a *ui.App) error {
	r, err := newRenderer(s, a.Image)
	if err != nil {
		return err
	}

	r.draw(a.State)
	if err := s.Map(); err != nil {
		return err
	}

	for !a.Done() {
		ev, err := s.X.WaitForEvent()
		if ev == nil && err == nil {
			return errors.New("connection to the X server closed")
		}

		// Handle everything queued before redrawing once, so fast pointer
		// motion doesn't back up behind redraws.
		dirty := false
		for !a.Done() && (ev != nil || err != nil) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "vshot: X error: %v\n", err)
			} else if handle(s, r, a, ev) {
				dirty = true
			}
			ev, err = s.X.PollForEvent()
		}
		if dirty && !a.Done() {
			r.draw(a.State)
		}
	}
	return nil
}

// handle processes one event and reports whether a redraw is needed.
func handle(s *Session, r *renderer, a *ui.App, ev xgb.Event) bool {
	switch e := ev.(type) {
	case xproto.ExposeEvent:
		if e.Count == 0 {
			r.present()
		}
	case xproto.ButtonPressEvent:
		if e.Detail == xproto.ButtonIndex1 && int(e.EventY) < r.barY {
			return a.Press(image.Pt(int(e.EventX), int(e.EventY)))
		}
	case xproto.MotionNotifyEvent:
		return a.Drag(image.Pt(int(e.EventX), int(e.EventY)))
	case xproto.ButtonReleaseEvent:
		if e.Detail == xproto.ButtonIndex1 {
			a.Release()
		}
	case xproto.KeyPressEvent:
		a.Key(ui.Keysym(s.Keysym(e.Detail, e.State)))
		return true
	}
	return false
}
