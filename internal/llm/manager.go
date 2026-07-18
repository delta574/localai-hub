package llm

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	dataDir   string
	cmd       *exec.Cmd
	port      int
	mu        sync.Mutex
	running   bool
	modelPath string
}

func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir: dataDir,
		port:    0,
	}
}

func (m *Manager) llamaServerPath() string {
	binDir := filepath.Join(m.dataDir, "bin")
	if runtime.GOOS == "windows" {
		return filepath.Join(binDir, "llama-server.exe")
	}
	return filepath.Join(binDir, "llama-server")
}

func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func (m *Manager) Port() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.port
}

func (m *Manager) BaseURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", m.Port())
}

func (m *Manager) Start(modelPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running && m.modelPath == modelPath {
		return nil
	}

	if m.running {
		m.stopLocked()
	}

	llamaPath := m.llamaServerPath()
	if _, err := os.Stat(llamaPath); os.IsNotExist(err) {
		return fmt.Errorf("llama-server not found at %s; download it first", llamaPath)
	}

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found at %s", modelPath)
	}

	port := m.findFreePort()
	threads := runtime.NumCPU()
	if threads > 1 {
		threads--
	}

	args := []string{
		"-m", modelPath,
		"-c", "4096",
		"--port", fmt.Sprintf("%d", port),
		"--host", "127.0.0.1",
		"-t", fmt.Sprintf("%d", threads),
		"-ngl", "0",
	}

	slog.Info("starting llama-server", "args", strings.Join(args, " "))

	cmd := exec.Command(llamaPath, args...)
	cmd.Stdout = os.Stdout
	var stderrBuf strings.Builder
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start llama-server: %w", err)
	}

	m.cmd = cmd
	m.running = true
	m.port = port
	m.modelPath = modelPath

	go func() {
		err := cmd.Wait()
		if err != nil {
			slog.Warn("llama-server exited", "error", err, "stderr", stderrBuf.String())
		}
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	err := m.waitForHealth(60 * time.Second)
	if err != nil {
		slog.Error("llama-server not healthy, stopping", "stderr", stderrBuf.String())
		m.cmd.Process.Kill()
		m.running = false
	}
	return err
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
}

func (m *Manager) stopLocked() {
	if m.cmd != nil && m.cmd.Process != nil {
		slog.Info("stopping llama-server")
		m.cmd.Process.Kill()
	}
	m.running = false
}

func (m *Manager) findFreePort() int {
	for port := 8081; port < 9000; port++ {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return port
		}
		conn.Close()
	}
	return 8081
}

func (m *Manager) waitForHealth(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)

		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", m.port))
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			slog.Info("llama-server is ready", "port", m.port)
			return nil
		}
	}
	return fmt.Errorf("llama-server did not become healthy within %v", timeout)
}

func (m *Manager) ChatCompletionsRaw(ctx context.Context, w io.Writer, body []byte) error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return fmt.Errorf("llama-server not running")
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", m.port)
	m.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create proxy request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("proxy request: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(w, resp.Body)
	return err
}
