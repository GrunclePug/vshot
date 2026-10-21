// Package x11 owns the X connection state used by the overlay: the window,
// its graphics context, input grabs, keyboard mapping and pixmap uploads.
package x11

import (
	"fmt"
	"image"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// xcCrosshair is the crosshair glyph in the standard X "cursor" font.
const xcCrosshair = 34

type Session struct {
	X      *xgb.Conn
	Win    xproto.Window
	GC     xproto.Gcontext
	Width  int
	Height int

	depth   byte
	maxReq  int // maximum request size in bytes
	cursor  xproto.Cursor
	keyMin  xproto.Keycode
	keyPer  int
	keySyms []xproto.Keysym
}

// NewSession creates an override-redirect window covering bounds (in root
// coordinates). The window is not mapped until Map is called, so the screen
// can be captured first.
func NewSession(X *xgb.Conn, bounds image.Rectangle) (*Session, error) {
	setup := xproto.Setup(X)
	screen := setup.DefaultScreen(X)
	s := &Session{
		X:      X,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		depth:  screen.RootDepth,
		maxReq: int(setup.MaximumRequestLength) * 4,
	}

	var err error
	if s.Win, err = xproto.NewWindowId(X); err != nil {
		return nil, err
	}
	if s.GC, err = xproto.NewGcontextId(X); err != nil {
		return nil, err
	}
	if s.cursor, err = xproto.NewCursorId(X); err != nil {
		return nil, err
	}

	err = xproto.CreateWindowChecked(X, s.depth, s.Win, screen.Root,
		int16(bounds.Min.X), int16(bounds.Min.Y), uint16(s.Width), uint16(s.Height), 0,
		xproto.WindowClassInputOutput, screen.RootVisual,
		xproto.CwOverrideRedirect|xproto.CwEventMask,
		[]uint32{1, xproto.EventMaskExposure | xproto.EventMaskKeyPress}).Check()
	if err != nil {
		return nil, fmt.Errorf("create window: %w", err)
	}
	// Without this every CopyArea in a redraw answers with a NoExposure event.
	xproto.CreateGC(X, s.GC, xproto.Drawable(s.Win), xproto.GcGraphicsExposures, []uint32{0})

	if font, err := xproto.NewFontId(X); err == nil {
		xproto.OpenFont(X, font, uint16(len("cursor")), "cursor")
		xproto.CreateGlyphCursor(X, s.cursor, font, font, xcCrosshair, xcCrosshair+1,
			0, 0, 0, 0xffff, 0xffff, 0xffff)
		xproto.CloseFont(X, font)
	}

	if err := s.loadKeymap(setup); err != nil {
		return nil, fmt.Errorf("keyboard mapping: %w", err)
	}
	return s, nil
}

// Map shows the window and takes exclusive control of pointer and keyboard.
// Grabs are retried briefly because they fail until the window is viewable
// or while another client (e.g. a hotkey daemon) still holds them.
func (s *Session) Map() error {
	xproto.MapWindow(s.X, s.Win)

	grab := func(try func() (byte, error)) bool {
		for range 500 {
			if status, err := try(); err == nil && status == xproto.GrabStatusSuccess {
				return true
			}
			time.Sleep(2 * time.Millisecond)
		}
		return false
	}

	ok := grab(func() (byte, error) {
		r, err := xproto.GrabPointer(s.X, false, s.Win,
			xproto.EventMaskButtonPress|xproto.EventMaskButtonRelease|xproto.EventMaskButton1Motion,
			xproto.GrabModeAsync, xproto.GrabModeAsync, s.Win, s.cursor, xproto.TimeCurrentTime).Reply()
		if err != nil {
			return 0, err
		}
		return r.Status, nil
	})
	if !ok {
		return fmt.Errorf("could not grab pointer")
	}

	ok = grab(func() (byte, error) {
		r, err := xproto.GrabKeyboard(s.X, false, s.Win, xproto.TimeCurrentTime,
			xproto.GrabModeAsync, xproto.GrabModeAsync).Reply()
		if err != nil {
			return 0, err
		}
		return r.Status, nil
	})
	if !ok {
		return fmt.Errorf("could not grab keyboard")
	}
	return nil
}

// NewPixmap allocates a window-sized pixmap without contents.
func (s *Session) NewPixmap() (xproto.Pixmap, error) {
	p, err := xproto.NewPixmapId(s.X)
	if err != nil {
		return 0, err
	}
	xproto.CreatePixmap(s.X, s.depth, p, xproto.Drawable(s.Win), uint16(s.Width), uint16(s.Height))
	return p, nil
}

// UploadImage copies img into a new window-sized pixmap. Data is sent in
// row bands that fit within the server's maximum request size.
func (s *Session) UploadImage(img *image.RGBA) (xproto.Pixmap, error) {
	p, err := s.NewPixmap()
	if err != nil {
		return 0, err
	}

	const putImageHeader = 24
	stride := s.Width * 4
	rows := (s.maxReq - putImageHeader) / stride
	if rows < 1 {
		return 0, fmt.Errorf("screen too wide for a single X request")
	}
	buf := make([]byte, stride*min(rows, s.Height))

	for y0 := 0; y0 < s.Height; y0 += rows {
		n := min(rows, s.Height-y0)
		for y := range n {
			src := img.Pix[img.PixOffset(img.Rect.Min.X, img.Rect.Min.Y+y0+y):]
			dst := buf[y*stride : (y+1)*stride]
			for i := 0; i < stride; i += 4 {
				dst[i], dst[i+1], dst[i+2], dst[i+3] = src[i+2], src[i+1], src[i], 0xff
			}
		}
		xproto.PutImage(s.X, xproto.ImageFormatZPixmap, xproto.Drawable(p), s.GC,
			uint16(s.Width), uint16(n), 0, int16(y0), 0, s.depth, buf[:n*stride])
	}
	return p, nil
}

func (s *Session) loadKeymap(setup *xproto.SetupInfo) error {
	count := byte(setup.MaxKeycode - setup.MinKeycode + 1)
	r, err := xproto.GetKeyboardMapping(s.X, setup.MinKeycode, count).Reply()
	if err != nil {
		return err
	}
	s.keyMin, s.keyPer, s.keySyms = setup.MinKeycode, int(r.KeysymsPerKeycode), r.Keysyms
	return nil
}

// Keysym translates a key event's keycode and modifier state into a keysym
// using the server's current keyboard layout.
func (s *Session) Keysym(code xproto.Keycode, state uint16) xproto.Keysym {
	if code < s.keyMin || s.keyPer < 1 {
		return 0
	}
	i := int(code-s.keyMin) * s.keyPer
	if i+s.keyPer > len(s.keySyms) {
		return 0
	}
	lower := s.keySyms[i]
	upper := lower
	if s.keyPer > 1 && s.keySyms[i+1] != 0 {
		upper = s.keySyms[i+1]
	} else if lower >= 'a' && lower <= 'z' {
		upper = lower - 'a' + 'A'
	}

	shift := state&xproto.ModMaskShift != 0
	if state&xproto.ModMaskLock != 0 && lower >= 'a' && lower <= 'z' {
		shift = !shift
	}
	if shift {
		return upper
	}
	return lower
}
