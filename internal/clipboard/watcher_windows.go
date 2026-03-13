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
	if err := clipboard.Init(); err != nil {
		return nil, err
	}
	log.Println("Clipboard: using Windows (WM_CLIPBOARDUPDATE)")

	// golang.design/x/clipboard uses AddClipboardFormatListener internally
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
				if len(imgBytes) == 0 {
					continue
				}
				// Windows race condition: WM_CLIPBOARDUPDATE fires before clipboard
				// is fully written. Wait 150ms to ensure data is available.
				select {
				case <-time.After(windowsRaceDelay):
				case <-ctx.Done():
					return
				}
				select {
				case out <- imgBytes:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}
