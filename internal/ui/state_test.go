package ui

import (
	"image"
	"testing"
)

var bounds = image.Rect(0, 0, 100, 100)

func TestNewSelectionIsInclusive(t *testing.T) {
	s := NewState(bounds)
	s.Press(image.Pt(30, 40))
	s.Drag(image.Pt(10, 20))
	s.Release()
	if want := image.Rect(10, 20, 31, 41); s.Selection != want {
		t.Fatalf("selection = %v, want %v", s.Selection, want)
	}
}

func TestSelectionReachesScreenEdge(t *testing.T) {
	s := NewState(bounds)
	s.Press(image.Pt(50, 50))
	s.Drag(image.Pt(99, 99))
	if s.Selection.Max != bounds.Max {
		t.Fatalf("selection = %v, want max %v", s.Selection, bounds.Max)
	}
}

func TestClickWithoutDragClearsSelection(t *testing.T) {
	s := NewState(bounds)
	s.Selection = image.Rect(10, 10, 20, 20)
	s.Press(image.Pt(80, 80))
	s.Release()
	if s.HasSelection() {
		t.Fatalf("selection = %v, want none", s.Selection)
	}
}

func TestResizeFromHandle(t *testing.T) {
	s := NewState(bounds)
	s.Selection = image.Rect(10, 10, 50, 50)
	s.Press(image.Pt(49, 49)) // bottom-right handle center
	s.Drag(image.Pt(59, 69))
	if want := image.Rect(10, 10, 60, 70); s.Selection != want {
		t.Fatalf("selection = %v, want %v", s.Selection, want)
	}
}

func TestResizePastOppositeEdgeFlips(t *testing.T) {
	s := NewState(bounds)
	s.Selection = image.Rect(40, 40, 60, 60)
	s.Press(image.Pt(59, 49)) // right handle
	s.Drag(image.Pt(29, 49))
	if want := image.Rect(30, 40, 40, 60); s.Selection != want {
		t.Fatalf("selection = %v, want %v", s.Selection, want)
	}
}

func TestMoveIsClampedToBounds(t *testing.T) {
	s := NewState(bounds)
	s.Selection = image.Rect(20, 20, 40, 40)
	s.Press(image.Pt(30, 25))
	s.Drag(image.Pt(500, -500))
	if want := image.Rect(80, 0, 100, 20); s.Selection != want {
		t.Fatalf("selection = %v, want %v", s.Selection, want)
	}
}

func TestMotionWithoutPressIsIgnored(t *testing.T) {
	s := NewState(bounds)
	if s.Drag(image.Pt(5, 5)) || s.HasSelection() {
		t.Fatal("drag without press changed state")
	}
}
