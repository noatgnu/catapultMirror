// free_space_unix.go
//go:build !windows
// +build !windows

package main

import (
	"golang.org/x/sys/unix"
)

// Get the available free space on Unix-based systems
func getFreeSpace(path string) (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}
