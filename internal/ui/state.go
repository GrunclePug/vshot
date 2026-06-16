package ui

import (
	"image"

	"github.com/grunclepug/vshot/internal/display"
)

type Mode int

const (
	ModeNormal Mode = iota
	ModeVisual
	ModeCommand
)

type GrabHandle int

const (
	HandleNone GrabHandle = iota
	HandleTopLeft
	HandleTop
	HandleTopRight
	HandleRight
	HandleBottomRight
	HandleBottom
	HandleBottomLeft
	HandleLeft
	HandleBody
)

type State struct {
	Mode         Mode
	Screenshot   *display.Screenshot
	Selection    image.Rectangle
	HasSelection bool

	ActiveHandle GrabHandle
	DragStart    image.Point
	InitialRect  image.Rectangle

	CommandBuffer string
	ErrorMessage  string
}

func NewState(shot *display.Screenshot) *State {
	return &State{
		Mode:       ModeNormal,
		Screenshot: shot,
	}
}

func (s *State) ClearSelection() {
	s.Selection = image.Rectangle{}
	s.HasSelection = false
	s.ActiveHandle = HandleNone
	s.Mode = ModeNormal
}

func (s *State) UpdateSelectionFromDrag(currentPos image.Point) {
	dx := currentPos.X - s.DragStart.X
	dy := currentPos.Y - s.DragStart.Y

	if s.ActiveHandle == HandleBody {
		rect := s.InitialRect.Add(image.Pt(dx, dy))
		bounds := s.Screenshot.Image.Bounds()

		if rect.Min.X < bounds.Min.X {
			rect = rect.Add(image.Pt(bounds.Min.X-rect.Min.X, 0))
		}
		if rect.Max.X > bounds.Max.X {
			rect = rect.Add(image.Pt(bounds.Max.X-rect.Max.X, 0))
		}
		if rect.Min.Y < bounds.Min.Y {
			rect = rect.Add(image.Pt(0, bounds.Min.Y-rect.Min.Y))
		}
		if rect.Max.Y > bounds.Max.Y {
			rect = rect.Add(image.Pt(0, bounds.Max.Y-rect.Max.Y))
		}

		s.Selection = rect
		return
	}

	rect := s.InitialRect
	switch s.ActiveHandle {
	case HandleTopLeft:
		rect.Min.X += dx
		rect.Min.Y += dy
	case HandleTop:
		rect.Min.Y += dy
	case HandleTopRight:
		rect.Max.X += dx
		rect.Min.Y += dy
	case HandleRight:
		rect.Max.X += dx
	case HandleBottomRight:
		rect.Max.X += dx
		rect.Max.Y += dy
	case HandleBottom:
		rect.Max.Y += dy
	case HandleBottomLeft:
		rect.Min.X += dx
		rect.Max.Y += dy
	case HandleLeft:
		rect.Min.X += dx
	}

	s.Selection = canonicalize(rect)
	s.Mode = ModeVisual
}

func (s *State) GetHandleAt(pt image.Point, dotSize int) GrabHandle {
	if !s.HasSelection {
		return HandleNone
	}

	r := s.Selection
	midX, midY := r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2

	in := func(c image.Point) bool {
		return pt.X >= c.X-dotSize && pt.X <= c.X+dotSize &&
			pt.Y >= c.Y-dotSize && pt.Y <= c.Y+dotSize
	}

	if in(image.Pt(r.Min.X, r.Min.Y)) {
		return HandleTopLeft
	}
	if in(image.Pt(midX, r.Min.Y)) {
		return HandleTop
	}
	if in(image.Pt(r.Max.X, r.Min.Y)) {
		return HandleTopRight
	}
	if in(image.Pt(r.Max.X, midY)) {
		return HandleRight
	}
	if in(image.Pt(r.Max.X, r.Max.Y)) {
		return HandleBottomRight
	}
	if in(image.Pt(midX, r.Max.Y)) {
		return HandleBottom
	}
	if in(image.Pt(r.Min.X, r.Max.Y)) {
		return HandleBottomLeft
	}
	if in(image.Pt(r.Min.X, midY)) {
		return HandleLeft
	}
	if pt.In(r) {
		return HandleBody
	}

	return HandleNone
}

func canonicalize(r image.Rectangle) image.Rectangle {
	if r.Min.X > r.Max.X {
		r.Min.X, r.Max.X = r.Max.X, r.Min.X
	}
	if r.Min.Y > r.Max.Y {
		r.Min.Y, r.Max.Y = r.Max.Y, r.Min.Y
	}
	return r
}
