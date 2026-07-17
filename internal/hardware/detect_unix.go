//go:build !windows

package hardware

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func detectRAM(info *Info) {
	switch runtime.GOOS {
	case "linux":
		detectLinuxRAM(info)
	case "darwin":
		detectMacRAM(info)
	}
}

func detectLinuxRAM(info *Info) {
	total, free := readMemInfo()
	if total == 0 {
		var si unix.Sysinfo_t
		if err := unix.Sysinfo(&si); err == nil {
			total = int(si.TotalRam * uint64(si.Unit) / (1024 * 1024 * 1024))
			free = int(si.Freeram * uint64(si.Unit) / (1024 * 1024 * 1024))
		}
	}
	info.RAMTotalGB = total
	info.RAMFreeGB = free
}

func readMemInfo() (int, int) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	var total, free int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			total = parseKB(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			free = parseKB(line)
		}
	}
	return total, free
}

func parseKB(line string) int {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	kb, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0
	}
	return kb / (1024 * 1024)
}

func detectMacRAM(info *Info) {
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

func diskFreeGB(path string) int {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0
	}
	return int(stat.Bfree * uint64(stat.Bsize) / (1024 * 1024 * 1024))
}
