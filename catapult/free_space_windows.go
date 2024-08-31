// catapult/free_space_windows.go
//go:build windows
// +build windows

package catapult

import (
	"golang.org/x/sys/windows"
)

// GetFreeSpace returns the available free space on Windows systems for the given path.
// It uses the Windows API to retrieve the disk space information.
//
// Parameters:
// - path: The directory path to check the free space for.
//
// Returns:
// - int64: The available free space in bytes.
// - error: An error object if there was an issue retrieving the free space.
func GetFreeSpace(path string) (int64, error) {
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	if err := windows.GetDiskFreeSpaceEx(pathPtr, (*uint64)(&freeBytesAvailable), (*uint64)(&totalNumberOfBytes), (*uint64)(&totalNumberOfFreeBytes)); err != nil {
		return 0, err
	}
	return int64(freeBytesAvailable), nil
}
