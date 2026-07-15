package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

func LatestLlamaServerVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/ggml-org/llama.cpp/releases/latest")
	if err != nil {
		return "", fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	var rel GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("decode release: %w", err)
	}
	return rel.TagName, nil
}

func LlamaServerURL(tag string) string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var platform, variant, ext string
	switch goos {
	case "windows":
		platform = "win"
		ext = ".zip"
	case "darwin":
		platform = "macos"
		ext = ".tar.gz"
	default:
		platform = "ubuntu"
		ext = ".tar.gz"
	}

	switch goarch {
	case "arm64":
		if goos == "windows" {
			variant = "cpu-arm64"
		} else {
			variant = "arm64"
		}
	default:
		if goos == "windows" {
			variant = "cpu-x64"
		} else {
			variant = "x64"
		}
	}

	filename := fmt.Sprintf("llama-%s-bin-%s-%s%s", tag, platform, variant, ext)
	return fmt.Sprintf("https://github.com/ggml-org/llama.cpp/releases/download/%s/%s", tag, filename)
}

type LlamaServerDownloader struct {
	binDir string
	client *http.Client
}

func NewLlamaServerDownloader(binDir string) *LlamaServerDownloader {
	return &LlamaServerDownloader{
		binDir: binDir,
		client: &http.Client{},
	}
}

func (d *LlamaServerDownloader) IsDownloaded() bool {
	name := "llama-server"
	if runtime.GOOS == "windows" {
		name = "llama-server.exe"
	}
	_, err := os.Stat(filepath.Join(d.binDir, name))
	return err == nil
}

func (d *LlamaServerDownloader) LlamaServerPath() string {
	name := "llama-server"
	if runtime.GOOS == "windows" {
		name = "llama-server.exe"
	}
	return filepath.Join(d.binDir, name)
}

func (d *LlamaServerDownloader) Download() error {
	if d.IsDownloaded() {
		return nil
	}

	tag, err := LatestLlamaServerVersion()
	if err != nil {
		return fmt.Errorf("get latest version: %w", err)
	}

	url := LlamaServerURL(tag)
	fmt.Printf("Downloading llama-server %s for %s/%s...\n", tag, runtime.GOOS, runtime.GOARCH)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download llama-server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	isZip := strings.HasSuffix(url, ".zip")

	tmpDir, err := os.MkdirTemp("", "llama-dl-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "archive"+func() string {
		if isZip {
			return ".zip"
		}
		return ".tar.gz"
	}())

	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return fmt.Errorf("save archive: %w", err)
	}
	f.Close()

	if isZip {
		return d.extractZip(tmpFile)
	}
	return d.extractTarGz(tmpFile)
}

func (d *LlamaServerDownloader) extractZip(path string) error {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	target := "llama-server.exe"
	for _, f := range zr.File {
		if filepath.Base(f.Name) == target {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("open %s in zip: %w", f.Name, err)
			}
			defer rc.Close()

			out, err := os.Create(filepath.Join(d.binDir, target))
			if err != nil {
				return fmt.Errorf("create %s: %w", target, err)
			}
			defer out.Close()

			if _, err := io.Copy(out, rc); err != nil {
				return fmt.Errorf("extract %s: %w", target, err)
			}
			return nil
		}
	}
	return fmt.Errorf("llama-server.exe not found in archive")
}

func (d *LlamaServerDownloader) extractTarGz(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open tar.gz: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	target := "llama-server"
	if runtime.GOOS == "windows" {
		target = "llama-server.exe"
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}
		if filepath.Base(header.Name) == target {
			out, err := os.Create(filepath.Join(d.binDir, target))
			if err != nil {
				return fmt.Errorf("create %s: %w", target, err)
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil {
				return fmt.Errorf("extract %s: %w", target, err)
			}
			if err := os.Chmod(out.Name(), 0755); err != nil {
				return fmt.Errorf("chmod %s: %w", out.Name(), err)
			}
			return nil
		}
	}
	return fmt.Errorf("%s not found in archive", target)
}
