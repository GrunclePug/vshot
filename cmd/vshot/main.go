package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/grunclepug/vshot/internal/ui"
	"github.com/grunclepug/vshot/internal/x11"
	"github.com/jezek/xgb"
)

var Version = "dev"

func main() {
	all := flag.Bool("all", false, "capture all monitors instead of the one under the pointer")
	version := flag.Bool("v", false, "print version and exit")
	flag.Parse()
	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(2)
	}

	if *version {
		fmt.Println("vshot", Version)
		return
	}

	if err := run(*all); err != nil {
		fmt.Fprintln(os.Stderr, "vshot:", err)
		os.Exit(1)
	}
}

func run(all bool) error {
	X, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("connect to X server: %w", err)
	}
	defer X.Close()

	// Capture before any window of ours is mapped, so it can't appear in the shot.
	img, bounds, err := x11.Capture(X, all)
	if err != nil {
		return fmt.Errorf("capture: %w", err)
	}

	s, err := x11.NewSession(X, bounds)
	if err != nil {
		return err
	}
	return ui.Run(s, img)
}
