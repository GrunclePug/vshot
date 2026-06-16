package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/grunclepug/vshot/internal/display"
	"github.com/grunclepug/vshot/internal/ui"
	"github.com/grunclepug/vshot/internal/x11"

	"github.com/jezek/xgb/xproto"
)

var Version = "dev"

func main() {
	allDisplays := flag.Bool("all", false, "Capture all active displays")
	showVersion := flag.Bool("v", false, "Print version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("vshot version %s\n", Version)
		os.Exit(0)
	}

	session, err := x11.NewSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "X11 connection failed: %v\n", err)
		os.Exit(1)
	}
	defer session.X.Close()

	shot, err := display.CaptureScreen(session.X, *allDisplays)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Capture error: %v\n", err)
		os.Exit(1)
	}

	if err := session.GrabInput(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer xproto.UngrabPointer(session.X, xproto.TimeCurrentTime)
	defer xproto.UngrabKeyboard(session.X, xproto.TimeCurrentTime)

	state := ui.NewState(shot)
	canvas := ui.NewCanvas(session.Width, session.Height)
	bar := ui.NewBarRenderer()
	renderer := ui.NewRenderer(session, bar)

	renderer.Redraw(state, canvas)
	ui.RunEventLoop(session, state, canvas, renderer)
}
