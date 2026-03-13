package notify

import "context"

// Notifier sends system notifications for detected QR codes.
type Notifier interface {
	// Notify sends a system notification with the decoded QR text.
	// If isURL is true, an additional "Open URL" action is included.
	Notify(ctx context.Context, text string, isURL bool) error
	// Close cleans up resources (D-Bus connections, etc.).
	Close() error
}
