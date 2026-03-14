//go:build windows

package main

import (
	"log"
	"os"
	"syscall"
)

const _ATTACH_PARENT_PROCESS = ^uint32(0) // 0xFFFFFFFF

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	pAttachConsole = kernel32.NewProc("AttachConsole")
)

func init() {
	// Built with -H windowsgui so no console is allocated on startup.
	// If launched from cmd/powershell, attach to the parent's console
	// so stdout/stderr work normally. On double-click from Explorer
	// there is no parent console — the call fails and the process
	// stays in pure GUI mode with no black window.
	r, _, _ := pAttachConsole.Call(uintptr(_ATTACH_PARENT_PROCESS))
	if r == 0 {
		return
	}

	conout, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	os.Stdout = conout
	os.Stderr = conout
	// Re-point the default logger to the attached console.
	// main() may later override this with io.Discard if -v is not set.
	log.SetOutput(conout)
}
