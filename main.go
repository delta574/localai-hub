package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"localai-hub/internal/api"
	"localai-hub/internal/config"
	"localai-hub/internal/download"
	"localai-hub/internal/hardware"
	"localai-hub/internal/llm"
	"localai-hub/internal/server"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	dataDir := flag.String("data", ".", "Data directory (models, config, conversations)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	absDir, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatalf("failed to resolve data dir: %v", err)
	}

	for _, d := range []string{"models", "conversations", "bin"} {
		if err := os.MkdirAll(filepath.Join(absDir, d), 0755); err != nil {
			log.Fatalf("failed to create %s dir: %v", d, err)
		}
	}

	cfg, err := config.Load(absDir)
	if err != nil {
		slog.Warn("no existing config, using defaults", "error", err)
		cfg = config.Default(absDir)
	}
	cfg.Port = *port

	hw := hardware.Detect()
	modelDownloader := download.New(absDir)

	llamaDL := download.NewLlamaServerDownloader(filepath.Join(absDir, "bin"))
	if !llamaDL.IsDownloaded() {
		slog.Info("llama-server not found, downloading...")
		if err := llamaDL.Download(); err != nil {
			slog.Warn("failed to download llama-server", "error", err)
			slog.Warn("place llama-server.exe manually in bin/ directory")
		} else {
			slog.Info("llama-server downloaded successfully")
		}
	}

	llmManager := llm.NewManager(absDir)
	srv := server.New(cfg, staticFS())

	apiHandler := api.New(cfg, hw, modelDownloader, llmManager, absDir)
	srv.RegisterAPI(apiHandler)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: srv.Router(),
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port, "data", absDir)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	url := fmt.Sprintf("http://localhost:%d", cfg.Port)
	slog.Info("opening browser", "url", url)
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	slog.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	llmManager.Stop()
	httpServer.Shutdown(ctx)
}
