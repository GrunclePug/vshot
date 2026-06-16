package x11

import (
	"fmt"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type Session struct {
	X         *xgb.Conn
	Win       xproto.Window
	GC        xproto.Gcontext
	Pixmap    xproto.Pixmap
	Width     int
	Height    int
	Depth     uint8
	TargetBpp int
}

func NewSession() (*Session, error) {
	X, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}

	setup := xproto.Setup(X)
	screen := setup.DefaultScreen(X)
	w, h := int(screen.WidthInPixels), int(screen.HeightInPixels)

	targetBpp := 32
	for _, f := range setup.PixmapFormats {
		if f.Depth == screen.RootDepth {
			targetBpp = int(f.BitsPerPixel)
			break
		}
	}

	win, _ := xproto.NewWindowId(X)
	mask := uint32(xproto.CwOverrideRedirect | xproto.CwEventMask)
	values := []uint32{
		1, // CwOverrideRedirect: true
		xproto.EventMaskExposure | xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease | xproto.EventMaskButton1Motion | xproto.EventMaskKeyPress,
	}

	xproto.CreateWindow(X, screen.RootDepth, win, screen.Root, 0, 0, uint16(w), uint16(h), 0,
		xproto.WindowClassInputOutput, screen.RootVisual, mask, values)
	xproto.MapWindow(X, win)
	xproto.SetInputFocus(X, xproto.InputFocusPointerRoot, win, xproto.TimeCurrentTime)

	gc, _ := xproto.NewGcontextId(X)
	xproto.CreateGC(X, gc, xproto.Drawable(win), 0, nil)
	pixmap, _ := xproto.NewPixmapId(X)
	xproto.CreatePixmap(X, screen.RootDepth, pixmap, xproto.Drawable(win), uint16(w), uint16(h))

	return &Session{X: X, Win: win, GC: gc, Pixmap: pixmap, Width: w, Height: h, Depth: screen.RootDepth, TargetBpp: targetBpp}, nil
}

// GrabInput ensures the application takes exclusive control of the pointer and keyboard.
func (s *Session) GrabInput() error {
	for range 1000 {
		pGrab, err := xproto.GrabPointer(s.X, false, s.Win,
			xproto.EventMaskButtonPress|xproto.EventMaskButtonRelease|xproto.EventMaskButton1Motion,
			xproto.GrabModeAsync, xproto.GrabModeAsync, 0, 0, xproto.TimeCurrentTime).Reply()

		kGrab, err2 := xproto.GrabKeyboard(s.X, false, s.Win,
			xproto.TimeCurrentTime, xproto.GrabModeAsync, xproto.GrabModeAsync).Reply()

		if err == nil && err2 == nil && pGrab.Status == xproto.GrabStatusSuccess && kGrab.Status == xproto.GrabStatusSuccess {
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}
	return fmt.Errorf("failed to grab input context")
}
