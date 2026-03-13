//go:build linux

package clipboard

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"golang.design/x/clipboard"
)

const waylandPollInterval = 1 * time.Second

// linuxWatcher selects X11 or Wayland mode at startup.
type linuxWatcher struct{}

// NewWatcher creates a new ClipboardWatcher for the current Linux session.
// It automatically detects whether to use X11 (golang.design/x/clipboard) or
// Wayland (wl-paste polling fallback).
func NewWatcher() (ClipboardWatcher, error) {
	return &linuxWatcher{}, nil
}

func (w *linuxWatcher) Watch(ctx context.Context) (<-chan []byte, error) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		log.Println("Clipboard: using Wayland (wl-paste polling)")
		return w.watchWayland(ctx)
	}
	log.Println("Clipboard: using X11")
	return w.watchX11(ctx)
}

// watchX11 uses golang.design/x/clipboard.Watch for event-driven X11 monitoring.
func (w *linuxWatcher) watchX11(ctx context.Context) (<-chan []byte, error) {
	if err := clipboard.Init(); err != nil {
		return nil, fmt.Errorf("clipboard init failed: %w", err)
	}
	ch := clipboard.Watch(ctx, clipboard.FmtImage)
	out := make(chan []byte, 1)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case imgBytes, ok := <-ch:
				if !ok {
					return
				}
				if len(imgBytes) > 0 {
					select {
					case out <- imgBytes:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return out, nil
}

// watchWayland polls wl-paste every second for PNG clipboard content.
func (w *linuxWatcher) watchWayland(ctx context.Context) (<-chan []byte, error) {
	// Verify wl-paste is available
	if _, err := exec.LookPath("wl-paste"); err != nil {
		return nil, fmt.Errorf("wl-paste not found: install wl-clipboard package (e.g., apt install wl-clipboard)")
	}

	out := make(chan []byte, 1)
	go func() {
		defer close(out)
		var lastHash [sha256.Size]byte
		ticker := time.NewTicker(waylandPollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cmd := exec.CommandContext(ctx, "wl-paste", "--type", "image/png", "--no-newline")
				data, err := cmd.Output()
				if err != nil || len(data) == 0 {
					// No image in clipboard or error — ignore silently
					continue
				}
				h := sha256.Sum256(data)
				if h == lastHash {
					continue
				}
				lastHash = h
				select {
				case out <- data:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}
