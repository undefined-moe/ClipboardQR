//go:build darwin

package notify

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gen2brain/beeep"
)

const maxBodyLen = 200

type darwinNotifier struct{}

// NewNotifier creates a macOS notifier using beeep with auto-copy fallback.
// No action buttons are available without a signed app bundle.
func NewNotifier() (Notifier, error) {
	return &darwinNotifier{}, nil
}

func (n *darwinNotifier) Notify(_ context.Context, text string, isURL bool) error {
	// Auto-copy decoded text to clipboard via pbcopy.
	// Using pbcopy (not clipboard lib) to write TEXT — won't re-trigger image watcher.
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	_ = cmd.Run() // best-effort, don't fail if pbcopy not available

	// Prepare notification body with usage hint
	displayText := text
	if len([]rune(displayText)) > maxBodyLen {
		displayText = string([]rune(displayText)[:maxBodyLen]) + "…"
	}

	var body string
	if isURL {
		body = fmt.Sprintf("%s\n(已复制到剪贴板，⌘V 粘贴)", displayText)
	} else {
		body = fmt.Sprintf("%s\n(已复制到剪贴板)", displayText)
	}

	return beeep.Notify("QR Code Detected", body, "")
}

func (n *darwinNotifier) Close() error {
	return nil
}
