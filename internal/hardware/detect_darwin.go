//go:build darwin

package hardware

import "golang.org/x/sys/unix"

func detectRAM(info *Info) {
	total, err := unix.SysctlUint64("hw.memsize")
	if err == nil {
		info.RAMTotalGB = int(total / (1024 * 1024 * 1024))
	}

	freePages, err := unix.SysctlUint64("vm.page_free_count")
	if err == nil {
		pageSize := unix.Getpagesize()
		info.RAMFreeGB = int(freePages * uint64(pageSize) / (1024 * 1024 * 1024))
	}
}
