package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/grunclepug/vshot/internal/x11"
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
	return x11.Main(all)
}
