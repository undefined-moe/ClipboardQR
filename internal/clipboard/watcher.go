package clipboard

import "context"

// ClipboardWatcher monitors the system clipboard for image changes.
type ClipboardWatcher interface {
	// Watch starts monitoring the clipboard and returns a channel that
	// receives PNG/TIFF encoded image bytes on each new clipboard image.
	// The channel is closed when ctx is cancelled.
	Watch(ctx context.Context) (<-chan []byte, error)
}
