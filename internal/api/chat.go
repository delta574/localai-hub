package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/delta574/localai-hub/internal/download"
	"github.com/delta574/localai-hub/internal/httputil"
)

func (h *Handler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "cannot read body")
		return
	}

	var chatReq struct {
		Model    string `json:"model"`
		Stream   bool   `json:"stream"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &chatReq); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	targetModel := chatReq.Model
	if targetModel == "" {
		targetModel = h.cfg.GetActiveModel()
	}

	modelPath := ""
	if targetModel != "" {
		for _, m := range download.CuratedModels {
			if m.ID == targetModel {
				modelPath = h.downloader.ModelPath(&m)
				break
			}
		}
	}
	if modelPath == "" {
		installed, err := h.downloader.InstalledModels()
		if err != nil {
			slog.Error("list installed models", "error", err)
			installed = []string{}
		}
		for _, name := range installed {
			modelPath = filepath.Join(h.downloader.ModelsDir(), name)
			break
		}
	}
	if modelPath == "" {
		httputil.WriteError(w, http.StatusNotFound, "no model installed")
		return
	}

	if err := h.llm.Start(modelPath); err != nil {
		slog.Error("failed to start llama-server", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to start inference engine")
		return
	}

	if chatReq.Stream {
		writeSSEHeaders(w)
	} else {
		w.Header().Set("Content-Type", "application/json")
	}

	if err := h.llm.ChatCompletionsRaw(r.Context(), w, body); err != nil {
		slog.Error("chat completion error", "error", err)
		if chatReq.Stream {
			errData, _ := json.Marshal(map[string]string{"error": err.Error()})
			io.WriteString(w, "data: "+string(errData)+"\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		} else {
			httputil.WriteError(w, http.StatusInternalServerError, "chat completion failed")
		}
	}
}
