//go:build windows

package engine

// UnmountDevice handles unmounting or volume release on Windows.
func UnmountDevice(devicePath string) error {
	return nil
}
