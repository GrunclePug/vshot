package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/grunclepug/vshot/internal/wayland"
	"github.com/grunclepug/vshot/internal/x11"
)

var Version = "dev"

func main() {
	all := flag.Bool("all", false, "capture all monitors instead of the one under the pointer")
	backend := flag.String("backend", "auto", "display backend: auto, x11 or wayland")
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

	if err := run(*backend, *all); err != nil {
		fmt.Fprintln(os.Stderr, "vshot:", err)
		os.Exit(1)
	}
}

func run(backend string, all bool) error {
	if backend == "auto" {
		// Check Wayland first: XWayland also sets DISPLAY, but can't see the real screen.
		backend = "x11"
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			backend = "wayland"
		}
	}
	switch backend {
	case "wayland":
		return wayland.Main(all)
	case "x11":
		return x11.Main(all)
	}
	return fmt.Errorf("unknown backend %q", backend)
}
