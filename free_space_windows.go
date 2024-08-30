// free_space_windows.go
//go:build windows
// +build windows

package main

import (
	"golang.org/x/sys/windows"
)

// Get the available free space on Windows systems
func getFreeSpace(path string) (int64, error) {
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
