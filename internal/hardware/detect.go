package hardware

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

type Info struct {
	RAMTotalGB    int    `json:"ramTotalGB"`
	RAMFreeGB     int    `json:"ramFreeGB"`
	DiskFreeGB    int    `json:"diskFreeGB"`
	CPUCores      int    `json:"cpuCores"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	IsFirstLaunch bool   `json:"isFirstLaunch"`
}

var globalMemoryStatus = syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")
var getDiskFreeSpaceEx = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

type memStatus struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func DiskFreeGB(path string) int {
	if runtime.GOOS != "windows" {
		return 0
	}
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}
	var user, total, free uint64
	ret, _, _ := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&user)),
		uintptr(unsafe.Pointer(&total)),
		uintptr(unsafe.Pointer(&free)),
	)
	if ret == 0 {
		return 0
	}
	return int(total / (1024 * 1024 * 1024))
}

func Detect(dataDir string) *Info {
	info := &Info{
		CPUCores: runtime.NumCPU(),
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
	}

	if runtime.GOOS == "windows" {
		var m memStatus
		m.Length = uint32(unsafe.Sizeof(m))
		ret, _, _ := globalMemoryStatus.Call(uintptr(unsafe.Pointer(&m)))
		if ret != 0 {
			info.RAMTotalGB = int(m.TotalPhys / (1024 * 1024 * 1024))
			info.RAMFreeGB = int(m.AvailPhys / (1024 * 1024 * 1024))
		}
	}
	if info.RAMTotalGB == 0 {
		info.RAMTotalGB = 4
		info.RAMFreeGB = 2
	}
	if info.RAMFreeGB < 1 {
		info.RAMFreeGB = info.RAMTotalGB - 1
	}
	info.DiskFreeGB = DiskFreeGB(".")

	if _, err := os.Stat(filepath.Join(dataDir, "config.json")); os.IsNotExist(err) {
		info.IsFirstLaunch = true
	}

	return info
}
