//go:build windows

package tray

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/getlantern/systray"
)

func startupLinkPath() string {
	appData := os.Getenv("APPDATA")
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "ClipboardQR.lnk")
}

func isAutoStartEnabled() bool {
	_, err := os.Stat(startupLinkPath())
	return err == nil
}

func setAutoStart(enable bool) error {
	linkPath := startupLinkPath()
	if enable {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		// Create .lnk shortcut via PowerShell
		script := `$s=(New-Object -COM WScript.Shell).CreateShortcut('` + linkPath + `');$s.TargetPath='` + exePath + `';$s.Save()`
		return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Run()
	}
	return os.Remove(linkPath)
}

// SetupPlatformMenuItems adds the auto-start toggle menu item on Windows.
func SetupPlatformMenuItems() {
	enabled := isAutoStartEnabled()
	autoStart := systray.AddMenuItemCheckbox("开机自启", "设置 ClipboardQR 开机自动启动", enabled)

	go func() {
		for range autoStart.ClickedCh {
			if autoStart.Checked() {
				// Currently checked → disable
				if err := setAutoStart(false); err == nil {
					autoStart.Uncheck()
				}
			} else {
				// Currently unchecked → enable
				if err := setAutoStart(true); err == nil {
					autoStart.Check()
				}
			}
		}
	}()
}
