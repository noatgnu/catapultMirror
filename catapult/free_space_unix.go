// catapult/free_space_unix.go
//go:build !windows
// +build !windows

package catapult

import (
	"golang.org/x/sys/unix"
)

// GetFreeSpace returns the available free space on Unix-based systems for the given path.
// It uses the Unix `statfs` system call to retrieve the file system statistics.
//
// Parameters:
// - path: The directory path to check the free space for.
//
// Returns:
// - int64: The available free space in bytes.
// - error: An error object if there was an issue retrieving the free space.
func GetFreeSpace(path string) (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}
