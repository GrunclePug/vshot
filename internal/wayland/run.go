// Package wayland is vshot's backend for wlroots-style Wayland compositors.
// It needs wlr-screencopy, wlr-layer-shell and wp-viewporter, and uses
// cursor-shape when available.
package wayland

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"math"
	"strings"

	"github.com/grunclepug/vshot/internal/ui"
)

// Clipboard is the command the Wayland backend uses to copy PNG data.
var Clipboard = []string{"wl-copy", "--type", "image/png"}

const (
	layerOverlay      = 3
	anchorAll         = 1 | 2 | 4 | 8
	keyboardExclusive = 1
	shapeCrosshair    = 8
	btnLeft           = 0x110 // evdev BTN_LEFT
	keymapXKBv1       = 1
)

type output struct {
	id        uint32
	name      string
	x, y      int32 // position in the compositor layout (logical pixels), from xdg-output
	transform int32
}

func (o *output) label() string {
	if o.name != "" {
		return o.name
	}
	return fmt.Sprint(o.id)
}

type session struct {
	c *conn

	compositor, shm, layerShell, viewporter, cursorShape, seat uint32
	xdgOutputs                                                 uint32
	screencopy, screencopyVersion, seatVersion                 uint32
	outputs                                                    []*output

	views                []*view
	pointer, cursorDev   uint32
	keyboard             uint32
	keys                 *keymap
	pointerView, kbdView *view
	pressApp             *ui.App
	err                  error
}

// Main runs vshot on the compositor named by $WAYLAND_DISPLAY.
func Main(all bool) error {
	c, err := dial()
	if err != nil {
		return fmt.Errorf("connect to Wayland compositor: %w", err)
	}
	defer c.close()

	// The keymap must exist before the seat is bound: the compositor sends
	// keyboard events as soon as a keyboard is available.
	keys, err := newKeymap()
	if err != nil {
		return err
	}
	defer keys.free()

	s := &session{c: c, keys: keys}
	if err := s.bindGlobals(); err != nil {
		return err
	}

	// Capture everything before any overlay exists, so none appears in the shot.
	shots := make([]*image.RGBA, len(s.outputs))
	for i, o := range s.outputs {
		if shots[i], err = s.captureOutput(o); err != nil {
			return err
		}
	}

	if err := s.createViews(); err != nil {
		return err
	}
	if err := s.assignApps(shots, all); err != nil {
		return err
	}
	return s.loop()
}

func (s *session) bindGlobals() error {
	c := s.c
	registry := c.newID(nil)
	bind := func(name uint32, iface string, version uint32, h handler) uint32 {
		id := c.newID(h)
		c.send(registry, 0, name, iface, version, id)
		return id
	}

	c.handlers[registry] = func(op uint16, m *msg) {
		if op != 0 { // global
			return
		}
		name, iface, version := m.uint(), m.string(), m.uint()
		switch iface {
		case "wl_compositor":
			s.compositor = bind(name, iface, min(version, 4), nil)
		case "wl_shm":
			s.shm = bind(name, iface, 1, nil)
		case "wl_seat":
			if s.seat == 0 {
				s.seatVersion = min(version, 5)
				s.seat = bind(name, iface, s.seatVersion, s.handleSeat)
			}
		case "wl_output":
			o := &output{}
			o.id = bind(name, iface, min(version, 4), func(op uint16, m *msg) {
				switch op {
				case 0: // geometry
					m.int() // x, y: wlroots reports 0,0; xdg-output has the layout position
					m.int()
					m.int() // physical width
					m.int() // physical height
					m.int() // subpixel
					m.string()
					m.string()
					o.transform = m.int()
				case 4: // name
					o.name = m.string()
				}
			})
			s.outputs = append(s.outputs, o)
		case "zwlr_layer_shell_v1":
			s.layerShell = bind(name, iface, min(version, 4), nil)
		case "zwlr_screencopy_manager_v1":
			s.screencopyVersion = min(version, 3)
			s.screencopy = bind(name, iface, s.screencopyVersion, nil)
		case "wp_viewporter":
			s.viewporter = bind(name, iface, 1, nil)
		case "wp_cursor_shape_manager_v1":
			s.cursorShape = bind(name, iface, 1, nil)
		case "zxdg_output_manager_v1":
			s.xdgOutputs = bind(name, iface, min(version, 2), nil)
		}
	}
	if err := c.send(displayID, 1, registry); err != nil { // get_registry
		return err
	}
	// The first roundtrip delivers the globals, the second the initial
	// events of the objects bound in response (output geometry, seat).
	if err := c.roundtrip(); err != nil {
		return err
	}
	if s.xdgOutputs != 0 {
		for _, o := range s.outputs {
			c.send(s.xdgOutputs, 1, c.newID(func(op uint16, m *msg) {
				if op == 0 { // logical_position
					o.x, o.y = m.int(), m.int()
				}
			}), o.id) // get_xdg_output
		}
	}
	if err := c.roundtrip(); err != nil {
		return err
	}

	var missing []string
	for _, g := range []struct {
		id   uint32
		name string
	}{
		{s.compositor, "wl_compositor"}, {s.shm, "wl_shm"}, {s.seat, "wl_seat"},
		{s.layerShell, "wlr-layer-shell"}, {s.screencopy, "wlr-screencopy"}, {s.viewporter, "wp-viewporter"},
	} {
		if g.id == 0 {
			missing = append(missing, g.name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("compositor does not support %s", strings.Join(missing, ", "))
	}
	if len(s.outputs) == 0 {
		return errors.New("no outputs")
	}
	return nil
}

// createViews puts a fullscreen overlay surface on every output and waits
// until each has been configured with its size.
func (s *session) createViews() error {
	c := s.c
	for _, o := range s.outputs {
		v := &view{s: s, out: o}
		v.surface = c.newID(nil)
		v.layer = c.newID(v.handleLayer)
		v.viewport = c.newID(nil)
		c.send(s.compositor, 0, v.surface)                                               // create_surface
		c.send(s.layerShell, 0, v.layer, v.surface, o.id, uint32(layerOverlay), "vshot") // get_layer_surface
		c.send(v.layer, 1, uint32(anchorAll))                                            // set_anchor
		c.send(v.layer, 2, int32(-1))                                                    // set_exclusive_zone: ignore panels
		c.send(v.layer, 4, uint32(keyboardExclusive))                                    // set_keyboard_interactivity
		c.send(s.viewporter, 1, v.viewport, v.surface)                                   // get_viewport
		if err := c.send(v.surface, 6); err != nil {                                     // commit
			return err
		}
		s.views = append(s.views, v)
	}

	for _, v := range s.views {
		for v.w == 0 {
			if err := c.dispatch(); err != nil {
				return err
			}
			if s.err != nil {
				return s.err
			}
		}
	}
	return nil
}

// assignApps connects views to apps: one per output, or with all set a
// single app over every output arranged as in the compositor layout.
func (s *session) assignApps(shots []*image.RGBA, all bool) error {
	if !all {
		for i, v := range s.views {
			v.setApp(ui.NewApp(shots[i], Clipboard), shots[i].Bounds())
		}
		return nil
	}

	if s.xdgOutputs == 0 {
		return errors.New("-all needs xdg-output support from the compositor")
	}
	// Output positions are logical; captures are in buffer pixels.
	ratio := float64(shots[0].Rect.Dx()) / float64(s.views[0].w)
	rects := make([]image.Rectangle, len(shots))
	var union image.Rectangle
	for i, v := range s.views {
		if r := float64(shots[i].Rect.Dx()) / float64(v.w); math.Abs(r-ratio) > 0.01 {
			return errors.New("-all needs every output at the same scale")
		}
		p := image.Pt(int(math.Round(float64(v.out.x)*ratio)), int(math.Round(float64(v.out.y)*ratio)))
		rects[i] = shots[i].Rect.Add(p)
		union = union.Union(rects[i])
	}

	img := image.NewRGBA(image.Rect(0, 0, union.Dx(), union.Dy()))
	for i := range rects {
		rects[i] = rects[i].Sub(union.Min)
		draw.Draw(img, rects[i], shots[i], image.Point{}, draw.Src)
	}
	app := ui.NewApp(img, Clipboard)
	for i, v := range s.views {
		v.setApp(app, rects[i])
	}
	return nil
}

func (s *session) loop() error {
	for {
		for _, v := range s.views {
			if err := v.draw(); err != nil {
				return err
			}
		}
		if err := s.c.dispatch(); err != nil {
			return err
		}
		if s.err != nil {
			return s.err
		}
		for _, v := range s.views {
			if v.app.Done() {
				return nil
			}
		}
	}
}

// redraw marks every view showing app as needing a new frame.
func (s *session) redraw(app *ui.App) {
	for _, v := range s.views {
		if v.app == app {
			v.dirty = true
		}
	}
}

func (s *session) viewFor(surface uint32) *view {
	for _, v := range s.views {
		if v.surface == surface {
			return v
		}
	}
	return nil
}

func (s *session) handleSeat(op uint16, m *msg) {
	if op != 0 { // capabilities
		return
	}
	caps := m.uint()
	c := s.c
	// Devices can come and go; drop objects for lost capabilities so a new
	// one is requested when the capability returns.
	if caps&1 == 0 && s.pointer != 0 {
		if s.cursorDev != 0 {
			c.send(s.cursorDev, 0) // destroy
		}
		if s.seatVersion >= 3 {
			c.send(s.pointer, 1) // release
		}
		s.pointer, s.cursorDev, s.pointerView = 0, 0, nil
	}
	if caps&2 == 0 && s.keyboard != 0 {
		if s.seatVersion >= 3 {
			c.send(s.keyboard, 0) // release
		}
		s.keyboard = 0
	}
	if caps&1 != 0 && s.pointer == 0 {
		s.pointer = c.newID(s.handlePointer)
		c.send(s.seat, 0, s.pointer) // get_pointer
		if s.cursorShape != 0 {
			s.cursorDev = c.newID(nil)
			c.send(s.cursorShape, 1, s.cursorDev, s.pointer) // get_pointer
		}
	}
	if caps&2 != 0 && s.keyboard == 0 {
		s.keyboard = c.newID(s.handleKeyboard)
		c.send(s.seat, 1, s.keyboard) // get_keyboard
	}
}

func (s *session) handlePointer(op uint16, m *msg) {
	switch op {
	case 0: // enter
		serial, surface := m.uint(), m.uint()
		s.pointerView = s.viewFor(surface)
		if s.cursorDev != 0 {
			s.c.send(s.cursorDev, 1, serial, uint32(shapeCrosshair)) // set_shape
		}
		s.motion(m.fixed(), m.fixed())
	case 1: // leave
		s.pointerView = nil
	case 2: // motion
		m.uint() // time
		s.motion(m.fixed(), m.fixed())
	case 3: // button
		m.uint() // serial
		m.uint() // time
		button, state := m.uint(), m.uint()
		v := s.pointerView
		if button != btnLeft {
			return
		}
		if state == 1 && v != nil && v.app != nil && !v.onBar() {
			v.app.Press(v.appPoint())
			s.pressApp = v.app
			s.redraw(v.app)
		} else if state == 0 && s.pressApp != nil {
			s.pressApp.Release()
			s.pressApp = nil
		}
	}
}

func (s *session) motion(x, y float64) {
	v := s.pointerView
	if v == nil {
		return
	}
	v.px, v.py = x, y
	if v.app != nil && v.app == s.pressApp && v.app.Drag(v.appPoint()) {
		s.redraw(v.app)
	}
}

func (s *session) handleKeyboard(op uint16, m *msg) {
	switch op {
	case 0: // keymap
		format, f, size := m.uint(), m.fd(), m.uint()
		if format != keymapXKBv1 {
			return
		}
		if err := s.keys.load(f, size); err != nil {
			s.err = fmt.Errorf("keymap: %w", err)
		}
	case 1: // enter
		m.uint() // serial
		s.kbdView = s.viewFor(m.uint())
	case 3: // key
		m.uint() // serial
		m.uint() // time
		key, state := m.uint(), m.uint()
		if state != 1 || len(s.views) == 0 { // ignore presses while still capturing
			return
		}
		v := s.pointerView
		if v == nil {
			v = s.kbdView
		}
		if v == nil {
			v = s.views[0]
		}
		if v.app != nil {
			v.app.Key(ui.Keysym(s.keys.keysym(key)))
			s.redraw(v.app)
		}
	case 4: // modifiers
		m.uint() // serial
		s.keys.setModifiers(m.uint(), m.uint(), m.uint(), m.uint())
	}
}
