//go:build linux

package notify

import (
	"context"
	"log"
	"os/exec"
	"sync"

	dbus "github.com/godbus/dbus/v5"
	"golang.design/x/clipboard"

	libnotify "github.com/esiqveland/notify"
)

const maxBodyLen = 200

// linuxNotifier uses D-Bus freedesktop notifications with action buttons.
type linuxNotifier struct {
	notifier libnotify.Notifier
	conn     *dbus.Conn
	// fallback is true when D-Bus is unavailable (e.g., SSH session).
	fallback bool
	mu       sync.Mutex
	lastText string
	lastURL  bool
}

// NewNotifier creates a Linux D-Bus notifier with action button support.
// Falls back to log output if D-Bus is unavailable.
func NewNotifier() (Notifier, error) {
	n := &linuxNotifier{}

	conn, err := dbus.SessionBusPrivate()
	if err != nil {
		log.Printf("D-Bus unavailable (%v), falling back to log output", err)
		n.fallback = true
		return n, nil
	}
	if err := conn.Auth(nil); err != nil {
		conn.Close()
		log.Printf("D-Bus auth failed (%v), falling back to log output", err)
		n.fallback = true
		return n, nil
	}
	if err := conn.Hello(); err != nil {
		conn.Close()
		log.Printf("D-Bus hello failed (%v), falling back to log output", err)
		n.fallback = true
		return n, nil
	}
	n.conn = conn

	notifierInst, err := libnotify.New(conn,
		libnotify.WithOnAction(func(s *libnotify.ActionInvokedSignal) {
			n.mu.Lock()
			text := n.lastText
			isURL := n.lastURL
			n.mu.Unlock()

			switch s.ActionKey {
			case "copy":
				if initErr := clipboard.Init(); initErr == nil {
					clipboard.Write(clipboard.FmtText, []byte(text))
				}
			case "open":
				if isURL {
					_ = exec.Command("xdg-open", text).Start()
				}
			}
		}),
	)
	if err != nil {
		conn.Close()
		log.Printf("notify.New failed (%v), falling back to log output", err)
		n.fallback = true
		return n, nil
	}
	n.notifier = notifierInst
	return n, nil
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "\u2026"
}

// Notify sends a desktop notification with action buttons.
// Always includes a "Copy" button; adds "Open URL" when isURL is true.
func (n *linuxNotifier) Notify(_ context.Context, text string, isURL bool) error {
	if n.fallback {
		log.Printf("[QR] %s", text)
		return nil
	}

	n.mu.Lock()
	n.lastText = text
	n.lastURL = isURL
	n.mu.Unlock()

	body := truncate(text, maxBodyLen)

	notification := libnotify.Notification{
		AppName:       "ClipboardQR",
		Summary:       "QR Code Detected",
		Body:          body,
		ExpireTimeout: libnotify.ExpireTimeoutSetByNotificationServer,
		Actions: []libnotify.Action{
			{Key: "copy", Label: "\u590d\u5236\u5185\u5bb9"},
		},
	}
	if isURL {
		notification.Actions = append(notification.Actions, libnotify.Action{
			Key:   "open",
			Label: "\u6253\u5f00\u94fe\u63a5",
		})
	}

	_, err := n.notifier.SendNotification(notification)
	return err
}

// Close releases D-Bus resources.
func (n *linuxNotifier) Close() error {
	if n.notifier != nil {
		n.notifier.Close()
	}
	if n.conn != nil {
		return n.conn.Close()
	}
	return nil
}
