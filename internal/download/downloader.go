package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ProgressEvent struct {
	Type             string  `json:"type"`
	ModelID          string  `json:"modelId,omitempty"`
	BytesDownloaded  int64   `json:"bytesDownloaded,omitempty"`
	TotalBytes       int64   `json:"totalBytes,omitempty"`
	Percent          float64 `json:"percent,omitempty"`
	Speed            int64   `json:"speed,omitempty"`
	Message          string  `json:"message,omitempty"`
}

type Downloader struct {
	dataDir string
	client  *http.Client
	mu      sync.Mutex
	running map[string]context.CancelFunc
}

func New(dataDir string) *Downloader {
	return &Downloader{
		dataDir: dataDir,
		client: &http.Client{
			Timeout: 30 * time.Minute,
		},
		running: make(map[string]context.CancelFunc),
	}
}

func (d *Downloader) ModelsDir() string {
	return filepath.Join(d.dataDir, "models")
}

func (d *Downloader) InstalledModels() ([]string, error) {
	dir := d.ModelsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var models []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".gguf") {
			models = append(models, e.Name())
		}
	}
	return models, nil
}

func (d *Downloader) IsInstalled(model *Model) bool {
	path := filepath.Join(d.ModelsDir(), model.HFFile)
	_, err := os.Stat(path)
	return err == nil
}

func (d *Downloader) ModelPath(model *Model) string {
	return filepath.Join(d.ModelsDir(), model.HFFile)
}

func (d *Downloader) StartPull(ctx context.Context, model *Model, events chan<- ProgressEvent) error {
	d.mu.Lock()
	if _, ok := d.running[model.ID]; ok {
		d.mu.Unlock()
		return fmt.Errorf("download already in progress for %s", model.ID)
	}
	ctx, cancel := context.WithCancel(ctx)
	d.running[model.ID] = cancel
	d.mu.Unlock()

	go func() {
		defer close(events)
		defer func() {
			d.mu.Lock()
			delete(d.running, model.ID)
			d.mu.Unlock()
		}()

		url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", model.HFRepo, model.HFFile)
		tmpPath := filepath.Join(d.ModelsDir(), model.HFFile+".partial")
		finalPath := filepath.Join(d.ModelsDir(), model.HFFile)

		if err := d.downloadFile(ctx, url, tmpPath, finalPath, model, events); err != nil {
			events <- ProgressEvent{Type: "error", ModelID: model.ID, Message: err.Error()}
			return
		}
		events <- ProgressEvent{Type: "done", ModelID: model.ID, Message: "installed"}
	}()

	return nil
}

func (d *Downloader) downloadFile(ctx context.Context, url, tmpPath, finalPath string, model *Model, events chan<- ProgressEvent) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	partialSize := int64(0)
	if stat, err := os.Stat(tmpPath); err == nil {
		partialSize = stat.Size()
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", partialSize))
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	totalBytes := resp.ContentLength + partialSize
	if resp.StatusCode == http.StatusPartialContent {
		if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
			if parts := strings.Split(contentRange, "/"); len(parts) == 2 {
				if total, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					totalBytes = total
				}
			}
		}
	}

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	var written int64 = partialSize
	buf := make([]byte, 32*1024)
	lastUpdate := time.Now()
	lastWritten := written

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return fmt.Errorf("write file: %w", werr)
			}
			written += int64(n)

			if time.Since(lastUpdate) > 200*time.Millisecond {
				elapsed := time.Since(lastUpdate).Seconds()
				speed := int64(float64(written-lastWritten) / elapsed)
				pct := float64(0)
				if totalBytes > 0 {
					pct = float64(written) / float64(totalBytes) * 100
				}

				events <- ProgressEvent{
					Type:            "progress",
					ModelID:         model.ID,
					BytesDownloaded: written,
					TotalBytes:      totalBytes,
					Percent:         pct,
					Speed:           speed,
				}

				lastUpdate = time.Now()
				lastWritten = written
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read response: %w", err)
		}
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	var renameErr error
	for i := 0; i < 5; i++ {
		if err := os.Rename(tmpPath, finalPath); err == nil {
			renameErr = nil
			break
		}
		renameErr = err
		time.Sleep(100 * time.Millisecond)
	}
	if renameErr != nil {
		return fmt.Errorf("finalize file: %w", renameErr)
	}

	return nil
}

func (d *Downloader) DeleteModel(modelID string) error {
	for _, m := range CuratedModels {
		if m.ID == modelID {
			path := filepath.Join(d.ModelsDir(), m.HFFile)
			if !strings.HasPrefix(path, d.ModelsDir()) {
				return fmt.Errorf("invalid model path")
			}
			return os.Remove(path)
		}
	}
	return fmt.Errorf("model not found: %s", modelID)
}
