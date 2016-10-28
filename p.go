package main

import (
	"fmt"
	"os"

	"github.com/google/pprof/driver"

	"github.com/hirochachacha/p/ui"
)

func main() {
	ui := ui.New()
	defer ui.Close()

	if err := driver.PProf(&driver.Options{
		UI: ui,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "pprof: %v\n", err)
		os.Exit(2)
	}
}
