//go:build windows

package clipboard

import (
	"context"
	"log"
	"time"

	"golang.design/x/clipboard"
)

const windowsRaceDelay = 150 * time.Millisecond

type windowsWatcher struct{}

// NewWatcher creates a Windows clipboard watcher.
// Uses golang.design/x/clipboard which works correctly on Windows.
func NewWatcher() (ClipboardWatcher, error) {
	return &windowsWatcher{}, nil
}

func (w *windowsWatcher) Watch(ctx context.Context) (<-chan []byte, error) {
	log.Println("Clipboard: initializing Windows clipboard...")
	if err := clipboard.Init(); err != nil {
		log.Printf("Clipboard: Init failed: %v", err)
		return nil, err
	}
	log.Println("Clipboard: Init succeeded, using Windows (WM_CLIPBOARDUPDATE)")

	// golang.design/x/clipboard uses AddClipboardFormatListener internally
	log.Println("Clipboard: starting Watch for FmtImage...")
	ch := clipboard.Watch(ctx, clipboard.FmtImage)
	log.Println("Clipboard: Watch channel created, listening for clipboard image events")
	out := make(chan []byte, 1)

	go func() {
		defer close(out)
		eventCount := 0
		for {
			select {
			case <-ctx.Done():
				log.Printf("Clipboard: context cancelled, total events received: %d", eventCount)
				return
			case imgBytes, ok := <-ch:
				eventCount++
				if !ok {
					log.Printf("Clipboard: watch channel closed, total events received: %d", eventCount)
					return
				}
				log.Printf("Clipboard: event #%d received, raw byte length: %d", eventCount, len(imgBytes))
				if len(imgBytes) == 0 {
					log.Printf("Clipboard: event #%d skipped (empty image bytes)", eventCount)
					continue
				}
				// Windows race condition: WM_CLIPBOARDUPDATE fires before clipboard
				// is fully written. Wait 150ms to ensure data is available.
				log.Printf("Clipboard: event #%d waiting %v for race delay...", eventCount, windowsRaceDelay)
				select {
				case <-time.After(windowsRaceDelay):
					log.Printf("Clipboard: event #%d race delay done, forwarding %d bytes", eventCount, len(imgBytes))
				case <-ctx.Done():
					return
				}
				select {
				case out <- imgBytes:
					log.Printf("Clipboard: event #%d forwarded to processing pipeline", eventCount)
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}
