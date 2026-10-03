package ui

import (
	"fmt"
	"image"

	"github.com/grunclepug/vshot/internal/config"
)

type handle int

const (
	handleNone handle = iota
	handleTopLeft
	handleTop
	handleTopRight
	handleRight
	handleBottomRight
	handleBottom
	handleBottomLeft
	handleLeft
	handleBody
	handleNew // dragging out a fresh selection
)

// State is the interactive selection and command-line state. All points are
// in image (and window) coordinates.
type State struct {
	Bounds     image.Rectangle
	Selection  image.Rectangle
	Command    bool
	CommandBuf string
	Message    string

	active handle
	anchor image.Point     // pointer position at press
	start  image.Rectangle // selection at press
}

func NewState(bounds image.Rectangle) *State {
	return &State{Bounds: bounds}
}

func (s *State) HasSelection() bool { return !s.Selection.Empty() }

func (s *State) ClearSelection() {
	s.Selection = image.Rectangle{}
	s.active = handleNone
}

// Press starts a drag at p: resizing or moving the selection when p is on
// it, otherwise starting a new selection.
func (s *State) Press(p image.Point) {
	s.Message = ""
	s.anchor = p
	s.start = s.Selection
	s.active = s.handleAt(p)
	if s.active == handleNone {
		s.active = handleNew
		s.Selection = image.Rectangle{}
	}
}

// Drag updates the selection for pointer position p and reports whether it
// changed.
func (s *State) Drag(p image.Point) bool {
	d := p.Sub(s.anchor)
	r := s.start

	switch s.active {
	case handleNone:
		return false
	case handleNew:
		// The selection spans both pointer pixels, inclusive.
		r = image.Rectangle{s.anchor, p}.Canon()
		r.Max = r.Max.Add(image.Pt(1, 1))
	case handleBody:
		r = r.Add(d)
		r = r.Add(image.Pt(shiftInto(r.Min.X, r.Max.X, s.Bounds.Min.X, s.Bounds.Max.X),
			shiftInto(r.Min.Y, r.Max.Y, s.Bounds.Min.Y, s.Bounds.Max.Y)))
	default:
		switch s.active {
		case handleTopLeft, handleLeft, handleBottomLeft:
			r.Min.X += d.X
		case handleTopRight, handleRight, handleBottomRight:
			r.Max.X += d.X
		}
		switch s.active {
		case handleTopLeft, handleTop, handleTopRight:
			r.Min.Y += d.Y
		case handleBottomLeft, handleBottom, handleBottomRight:
			r.Max.Y += d.Y
		}
	}

	r = r.Canon().Intersect(s.Bounds)
	if r == s.Selection {
		return false
	}
	s.Selection = r
	return true
}

func (s *State) Release() { s.active = handleNone }

// StatusText is the line shown in the bar.
func (s *State) StatusText() string {
	switch {
	case s.Command:
		return ":" + s.CommandBuf
	case s.Message != "":
		return s.Message
	case s.HasSelection():
		return fmt.Sprintf("-- VISUAL --  %dx%d", s.Selection.Dx(), s.Selection.Dy())
	default:
		return "-- NORMAL --"
	}
}

func (s *State) handleAt(p image.Point) handle {
	if !s.HasSelection() {
		return handleNone
	}
	r := config.DotSize
	for i, c := range HandlePoints(s.Selection) {
		if p.In(image.Rect(c.X-r, c.Y-r, c.X+r+1, c.Y+r+1)) {
			return handleTopLeft + handle(i)
		}
	}
	if p.In(s.Selection) {
		return handleBody
	}
	return handleNone
}

// HandlePoints returns the grab handle centers of r, in handle order
// starting at handleTopLeft and going clockwise.
func HandlePoints(r image.Rectangle) [8]image.Point {
	x0, y0, x1, y1 := r.Min.X, r.Min.Y, r.Max.X-1, r.Max.Y-1
	mx, my := (x0+x1)/2, (y0+y1)/2
	return [8]image.Point{
		{x0, y0}, {mx, y0}, {x1, y0}, {x1, my},
		{x1, y1}, {mx, y1}, {x0, y1}, {x0, my},
	}
}

// shiftInto returns the offset that moves span [lo, hi) inside [min, max).
func shiftInto(lo, hi, min, max int) int {
	switch {
	case lo < min:
		return min - lo
	case hi > max:
		return max - hi
	}
	return 0
}
