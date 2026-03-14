//go:build darwin

package notify

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

const maxBodyLen = 200

type darwinNotifier struct {
	mu      sync.Mutex
	lastCmd *exec.Cmd
}

// NewNotifier creates a macOS notifier.
// Uses terminal-notifier for click-to-copy when available, falling back
// to an osascript alert dialog with a Copy button.
func NewNotifier() (Notifier, error) {
	return &darwinNotifier{}, nil
}

func (n *darwinNotifier) Notify(_ context.Context, text string, isURL bool) error {
	displayText := text
	if len([]rune(displayText)) > maxBodyLen {
		displayText = string([]rune(displayText)[:maxBodyLen]) + "…"
	}

	// Kill previous notification process if still running.
	n.mu.Lock()
	if n.lastCmd != nil && n.lastCmd.Process != nil {
		_ = n.lastCmd.Process.Kill()
	}
	n.mu.Unlock()

	// Prefer terminal-notifier (supports click callbacks).
	if tnPath, err := exec.LookPath("terminal-notifier"); err == nil {
		return n.notifyTerminalNotifier(tnPath, text, displayText, isURL)
	}

	// Fallback: osascript alert dialog with Copy button.
	return n.notifyOsascript(text, displayText, isURL)
}

// notifyTerminalNotifier shows a macOS notification via terminal-notifier.
// Clicking the notification body copies text to clipboard.
// For URLs an additional "打开链接" action button opens the link.
func (n *darwinNotifier) notifyTerminalNotifier(tnPath, text, displayText string, isURL bool) error {
	body := fmt.Sprintf("%s\n(点击通知复制到剪贴板)", displayText)

	args := []string{
		"-title", "QR Code Detected",
		"-message", body,
		"-timeout", "30",
	}
	if isURL {
		args = append(args, "-actions", "打开链接")
	}

	cmd := exec.Command(tnPath, args...)

	// terminal-notifier blocks until the notification is dismissed or clicked.
	// Run in a goroutine so the watch loop keeps processing images.
	go func() {
		n.mu.Lock()
		n.lastCmd = cmd
		n.mu.Unlock()

		output, err := cmd.Output()
		if err != nil {
			return
		}

		result := strings.TrimSpace(string(output))

		// Only copy on active user interaction, not on timeout/close.
		if result == "@CLOSED" || result == "@TIMEOUT" {
			return
		}

		// Copy to clipboard via pbcopy (won't re-trigger image watcher).
		pbcopy := exec.Command("pbcopy")
		pbcopy.Stdin = strings.NewReader(text)
		_ = pbcopy.Run()

		// Open URL if that specific action was clicked.
		if result == "打开链接" && isURL {
			_ = exec.Command("open", text).Start()
		}
	}()

	return nil
}

// notifyOsascript shows an AppleScript alert dialog with Copy/Cancel buttons.
// Used as fallback when terminal-notifier is not installed.
func (n *darwinNotifier) notifyOsascript(text, displayText string, isURL bool) error {
	escape := func(s string) string {
		s = strings.ReplaceAll(s, `\`, `\\`)
		return strings.ReplaceAll(s, `"`, `\"`)
	}
	safeText := escape(text)
	safeDisplay := escape(displayText)

	var buttons, handler string
	if isURL {
		buttons = `{"取消", "打开链接", "复制内容"}`
		handler = fmt.Sprintf(
			"set btn to button returned of theResult\n"+
				"if btn is \"复制内容\" then\n"+
				"\tset the clipboard to \"%s\"\n"+
				"else if btn is \"打开链接\" then\n"+
				"\tset the clipboard to \"%s\"\n"+
				"\topen location \"%s\"\n"+
				"end if", safeText, safeText, safeText)
	} else {
		buttons = `{"取消", "复制内容"}`
		handler = fmt.Sprintf(
			"if button returned of theResult is \"复制内容\" then\n"+
				"\tset the clipboard to \"%s\"\n"+
				"end if", safeText)
	}

	script := fmt.Sprintf(
		"set theResult to display alert \"QR Code Detected\" message \"%s\" buttons %s default button \"复制内容\" giving up after 30\n"+
			"if gave up of theResult then return\n"+
			"%s", safeDisplay, buttons, handler)

	cmd := exec.Command("osascript", "-e", script)

	go func() {
		n.mu.Lock()
		n.lastCmd = cmd
		n.mu.Unlock()
		_ = cmd.Run()
	}()

	return nil
}

func (n *darwinNotifier) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.lastCmd != nil && n.lastCmd.Process != nil {
		_ = n.lastCmd.Process.Kill()
	}
	return nil
}
