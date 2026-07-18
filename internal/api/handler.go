package api

import (
	"context"
	"io"
	"net/http"

	"github.com/delta574/localai-hub/internal/config"
	"github.com/delta574/localai-hub/internal/download"
	"github.com/delta574/localai-hub/internal/hardware"
	"github.com/delta574/localai-hub/internal/httputil"
)

type LLMBackend interface {
	IsRunning() bool
	Start(modelPath string) error
	ChatCompletionsRaw(ctx context.Context, w io.Writer, body []byte) error
}

type Handler struct {
	cfg        *config.Config
	hw         *hardware.Info
	downloader *download.Downloader
	llm        LLMBackend
	dataDir    string
}

func New(cfg *config.Config, hw *hardware.Info, d *download.Downloader, l LLMBackend, dataDir string) *Handler {
	return &Handler{
		cfg:        cfg,
		hw:         hw,
		downloader: d,
		llm:        l,
		dataDir:    dataDir,
	}
}

func (h *Handler) SystemInfo(w http.ResponseWriter, r *http.Request) {
	installed, err := h.downloader.InstalledModels()
	if err != nil {
		installed = []string{}
	}

	info := map[string]any{
		"ram": map[string]int{
			"total": h.hw.RAMTotalGB,
			"free":  h.hw.RAMFreeGB,
		},
		"diskFreeGB":      h.hw.DiskFreeGB,
		"cpu":             h.hw.CPUCores,
		"os":              h.hw.OS,
		"arch":            h.hw.Arch,
		"isFirstLaunch":   h.hw.IsFirstLaunch,
		"recommendedModel": download.Recommend(h.hw.RAMFreeGB),
		"installedModels":  installed,
		"llamaServerRunning": h.llm.IsRunning(),
		"activeModel":     h.cfg.ViewActiveModel(),
		"systemPrompt":    h.cfg.GetSystemPrompt(),
		"temperature":     h.cfg.GetTemperature(),
	}
	httputil.WriteJSON(w, http.StatusOK, info)
}
