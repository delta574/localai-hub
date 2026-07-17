package hardware

import (
	"os"
	"path/filepath"
	"runtime"
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

func Detect(dataDir string) *Info {
	info := &Info{
		CPUCores: runtime.NumCPU(),
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
	}

	detectRAM(info)
	info.DiskFreeGB = diskFreeGB(".")

	if info.RAMTotalGB == 0 {
		info.RAMTotalGB = 4
		info.RAMFreeGB = 2
	}
	if info.RAMFreeGB < 1 {
		info.RAMFreeGB = info.RAMTotalGB - 1
	}

	if _, err := os.Stat(filepath.Join(dataDir, "config.json")); os.IsNotExist(err) {
		info.IsFirstLaunch = true
	}

	return info
}
