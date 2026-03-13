package tray

import (
	"github.com/getlantern/systray"
)

var quitCh = make(chan struct{})

// Run starts the system tray icon. It blocks until the tray is quit.
// onReady is called when the tray is ready; onQuit is called on exit.
// IMPORTANT: systray.Run blocks — all app logic must run in goroutines from onReady.
func Run(onReady func(), onQuit func()) {
	systray.Run(func() {
		// Set up tray icon and menu
		systray.SetIcon(iconBytes)
		systray.SetTitle("ClipboardQR")
		systray.SetTooltip("ClipboardQR — watching clipboard for QR codes")

		// Title item (disabled — display only)
		title := systray.AddMenuItem("ClipboardQR", "")
		title.Disable()
		systray.AddSeparator()

		// Platform-specific menu items (auto-start on Windows, no-op on others)
		SetupPlatformMenuItems()

		// Quit item
		quit := systray.AddMenuItem("退出", "退出 ClipboardQR")

		// Handle quit click in goroutine
		go func() {
			<-quit.ClickedCh
			close(quitCh)
			systray.Quit()
		}()

		// Call the user's onReady callback
		if onReady != nil {
			go onReady()
		}
	}, func() {
		if onQuit != nil {
			onQuit()
		}
	})
}

// QuitCh returns a channel that is closed when the user clicks "Quit" in the tray menu.
func QuitCh() <-chan struct{} {
	return quitCh
}
