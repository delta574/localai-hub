package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const LlamaServerVersion = "b10034"

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

func (d *LlamaServerDownloader) LatestVersion() (string, error) {
	return LlamaServerVersion, nil
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
		client: &http.Client{Timeout: 5 * time.Minute},
	}
}

func (d *LlamaServerDownloader) IsDownloaded() bool {
	target := "libllama-server-impl.so"
	if runtime.GOOS == "windows" {
		target = "llama-server-impl.dll"
	}
	path := filepath.Join(d.binDir, target)
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.Size() > 1024*1024
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

	tag, err := d.LatestVersion()
	if err != nil {
		return fmt.Errorf("get latest version: %w", err)
	}

	url := LlamaServerURL(tag)
	fmt.Printf("Downloading llama-server %s for %s/%s...\n", tag, runtime.GOOS, runtime.GOARCH)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			fmt.Printf("Retrying download (attempt %d/3)...\n", attempt+1)
			time.Sleep(2 * time.Second)
		}
		if err := d.downloadOnce(url); err != nil {
			lastErr = err
			fmt.Printf("Download failed: %v\n", err)
			continue
		}
		return nil
	}
	return fmt.Errorf("download failed after 3 attempts: %w", lastErr)
}

func (d *LlamaServerDownloader) downloadOnce(url string) error {
	resp, err := d.client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP GET: %w", err)
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

	for _, f := range zr.File {
		name := filepath.Clean(f.Name)
		if strings.Contains(name, "..") {
			return fmt.Errorf("invalid archive entry: %s (path traversal)", f.Name)
		}
		outPath := filepath.Join(d.binDir, name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(outPath, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(outPath), 0755)

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open %s: %w", f.Name, err)
		}

		out, err := os.Create(outPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create %s: %w", name, err)
		}

		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return fmt.Errorf("extract %s: %w", name, err)
		}
	}
	return nil
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
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		name := header.Name
		if idx := strings.IndexByte(name, '/'); idx >= 0 {
			name = name[idx+1:]
		}
		if name == "" {
			continue
		}
		if strings.Contains(name, "..") {
			return fmt.Errorf("invalid archive entry: %s (path traversal)", header.Name)
		}

		outPath := filepath.Join(d.binDir, name)

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(outPath, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(outPath), 0755)
			out, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("create %s: %w", name, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("extract %s: %w", name, err)
			}
			out.Close()
			if err := os.Chmod(outPath, header.FileInfo().Mode()); err != nil {
				return fmt.Errorf("chmod %s: %w", outPath, err)
			}
		}
	}
	return nil
}
