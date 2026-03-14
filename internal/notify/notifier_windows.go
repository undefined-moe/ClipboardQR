//go:build windows

package notify

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

const (
	maxBodyLen = 200
	appID      = "ClipboardQR"
)

type windowsNotifier struct {
	mu      sync.Mutex
	lastCmd *exec.Cmd
}

// NewNotifier creates a Windows notifier using PowerShell WinRT Toast API.
// Copies decoded text to clipboard only when user clicks the notification.
func NewNotifier() (Notifier, error) {
	return &windowsNotifier{}, nil
}

func (n *windowsNotifier) Notify(_ context.Context, text string, isURL bool) error {
	// Kill previous notification process if still running.
	n.mu.Lock()
	if n.lastCmd != nil && n.lastCmd.Process != nil {
		_ = n.lastCmd.Process.Kill()
	}
	n.mu.Unlock()

	// Truncate display text.
	displayText := text
	runes := []rune(displayText)
	if len(runes) > maxBodyLen {
		displayText = string(runes[:maxBodyLen]) + "…"
	}
	displayText = strings.ReplaceAll(displayText, "\n", " ")

	xmlBody := escapeXML(displayText + " (点击复制到剪贴板)")

	// "复制内容" uses foreground activation → fires Activated event → copies.
	// "打开链接" uses protocol activation → Windows opens URL directly.
	actionsXML := `<action content="复制内容" arguments="copy" activationType="foreground" />`
	if isURL {
		actionsXML += fmt.Sprintf(
			`<action content="打开链接" arguments="%s" activationType="protocol" />`,
			escapeXML(text))
	}

	// Escape for PowerShell single-quoted string.
	psText := strings.ReplaceAll(text, "'", "''")

	// PowerShell script using WinRT Toast API with click-to-copy event handler.
	// The process stays alive until the toast is dismissed or timeout expires,
	// so the Activated handler has a chance to fire and copy to clipboard.
	script := fmt.Sprintf(`
[void][Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime]
[void][Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime]

$xmlStr = @'
<toast activationType="foreground">
  <visual>
    <binding template="ToastGeneric">
      <text>QR Code Detected</text>
      <text>%s</text>
    </binding>
  </visual>
  <actions>
    %s
  </actions>
</toast>
'@

$xml = [Windows.Data.Xml.Dom.XmlDocument]::new()
$xml.LoadXml($xmlStr)

$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)

Register-ObjectEvent -InputObject $toast -EventName Activated -Action {
    Set-Clipboard '%s'
} | Out-Null

Register-ObjectEvent -InputObject $toast -EventName Dismissed -SourceIdentifier 'ToastDismissed' | Out-Null

[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('%s').Show($toast)

$null = Wait-Event -SourceIdentifier 'ToastDismissed' -Timeout 60
Start-Sleep -Seconds 1
`, xmlBody, actionsXML, psText, appID)

	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden",
		"-ExecutionPolicy", "Bypass", "-Command", script)

	go func() {
		n.mu.Lock()
		n.lastCmd = cmd
		n.mu.Unlock()
		_ = cmd.Run()
	}()

	return nil
}

func escapeXML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return r.Replace(s)
}

func (n *windowsNotifier) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.lastCmd != nil && n.lastCmd.Process != nil {
		_ = n.lastCmd.Process.Kill()
	}
	return nil
}
