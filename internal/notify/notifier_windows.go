//go:build windows

package notify

import (
	"context"
	"fmt"

	"github.com/go-toast/toast"
	"golang.design/x/clipboard"
)

const (
	maxBodyLen = 200
	appID      = "ClipboardQR"
)

type windowsNotifier struct{}

// NewNotifier creates a Windows Toast notifier.
// Auto-copies decoded text to clipboard since go-toast only supports
// protocol-type actions (cannot implement Copy button callback).
func NewNotifier() (Notifier, error) {
	return &windowsNotifier{}, nil
}

func (n *windowsNotifier) Notify(_ context.Context, text string, isURL bool) error {
	// Auto-copy decoded text to clipboard
	if err := clipboard.Init(); err == nil {
		clipboard.Write(clipboard.FmtText, []byte(text))
	}

	// Truncate display text
	displayText := text
	runes := []rune(displayText)
	if len(runes) > maxBodyLen {
		displayText = string(runes[:maxBodyLen]) + "..."
	}

	// Build notification body with hint
	var body string
	if isURL {
		body = fmt.Sprintf("%s\n(已复制到剪贴板，点击\"打开链接\"在浏览器中打开)", displayText)
	} else {
		body = fmt.Sprintf("%s\n(已复制到剪贴板)", displayText)
	}

	notification := toast.Notification{
		AppID:   appID,
		Title:   "QR Code Detected",
		Message: body,
	}

	// Add "Open URL" button for URLs (protocol action - only type go-toast supports)
	if isURL {
		notification.Actions = []toast.Action{
			{
				Type:      "protocol",
				Label:     "打开链接",
				Arguments: text,
			},
		}
	}

	return notification.Push()
}

func (n *windowsNotifier) Close() error {
	return nil
}
