package hardware

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	info := Detect(dir)

	if info.CPUCores == 0 {
		t.Error("expected CPU cores > 0")
	}
	if info.OS == "" {
		t.Error("expected non-empty OS")
	}
	if info.Arch == "" {
		t.Error("expected non-empty Arch")
	}
	if !info.IsFirstLaunch {
		t.Error("expected first launch in empty dir")
	}
	if info.RAMTotalGB == 0 {
		t.Error("expected non-zero RAM total")
	}
	if info.RAMFreeGB == 0 {
		t.Error("expected non-zero RAM free")
	}
	if info.RAMFreeGB >= info.RAMTotalGB {
		t.Error("expected RAM free < total")
	}
}

func TestDetect_NotFirstLaunch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0644)
	info := Detect(dir)
	if info.IsFirstLaunch {
		t.Error("expected not first launch when config exists")
	}
}

func TestDetect_RAMFallback(t *testing.T) {
	// Detect creates info even with no RAM detection
	dir := t.TempDir()
	info := Detect(dir)
	if info.RAMTotalGB < 1 {
		t.Error("expected at least fallback RAM total")
	}
}
