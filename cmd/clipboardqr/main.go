package main

import (
	"flag"
	"log"
	"os"
)

var verbose = flag.Bool("v", false, "enable verbose logging")

func main() {
	flag.Parse()
	if !*verbose {
		log.SetOutput(os.Stderr)
		// In non-verbose mode, suppress routine logs
		// (keep for now; platform wiring in Task 8 will refine)
	}
	// TODO: wiring implemented in Task 8
	// systray.Run(onReady, onQuit) will be the main loop
	log.Println("ClipboardQR: stub — full implementation in Task 8")
}
