//go:build linux

package hardware

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func detectRAM(info *Info) {
	total, free := readMemInfo()
	if total == 0 {
		var si unix.Sysinfo_t
		if err := unix.Sysinfo(&si); err == nil {
			total = int(si.Totalram * uint64(si.Unit) / (1024 * 1024 * 1024))
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
