//go:build !windows

package tray

// SetupPlatformMenuItems adds no platform-specific items on Linux/macOS.
// Auto-start management is not implemented on these platforms.
func SetupPlatformMenuItems() {}
