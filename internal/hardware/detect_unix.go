//go:build !windows

package hardware

import (
	"golang.org/x/sys/unix"
)

func diskFreeGB(path string) int {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0
	}
	return int(stat.Bfree * uint64(stat.Bsize) / (1024 * 1024 * 1024))
}
