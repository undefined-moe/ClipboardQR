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
	textCh := clipboard.Watch(ctx, clipboard.FmtText)
	go func() {
		textCount := 0
		for {
			select {
			case <-ctx.Done():
				return
			case textBytes, ok := <-textCh:
				if !ok {
					return
				}
				textCount++
				log.Printf("Clipboard: [diag] text event #%d, size=%d bytes, content=%q",
					textCount, len(textBytes), truncate(textBytes, 100))

				imgProbe := clipboard.Read(clipboard.FmtImage)
				log.Printf("Clipboard: [diag] image probe at text event #%d: image size=%d bytes",
					textCount, len(imgProbe))
				if len(imgProbe) > 0 {
					log.Printf("Clipboard: [diag] image IS available but Watch missed it! first 16 bytes: %x", head(imgProbe, 16))
				}
			}
		}
	}()

	log.Println("Clipboard: starting Watch for FmtImage...")
	ch := clipboard.Watch(ctx, clipboard.FmtImage)
	log.Println("Clipboard: Watch channel created, listening for clipboard image events")
	log.Println("Clipboard: [diag] also watching FmtText for diagnostics")
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

func truncate(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}

func head(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[:n]
}
