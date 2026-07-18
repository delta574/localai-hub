package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/delta574/localai-hub/internal/download"
	"github.com/delta574/localai-hub/internal/httputil"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListModels(w http.ResponseWriter, r *http.Request) {
	installed, err := h.downloader.InstalledModels()
	if err != nil {
		slog.Error("list installed models", "error", err)
		installed = []string{}
	}
	installedSet := make(map[string]bool)
	for _, name := range installed {
		installedSet[name] = true
	}

	models := make([]map[string]any, 0, len(download.CuratedModels))
	for _, m := range download.CuratedModels {
		isInstalled := installedSet[m.HFFile]
		models = append(models, map[string]any{
			"id":        m.ID,
			"name":      m.Name,
			"sizeGB":    m.SizeGB,
			"minRamGB":  m.MinRAMGB,
			"quality":   m.Quality,
			"tagline":   m.Tagline,
			"installed": isInstalled,
			"file":      m.HFFile,
		})
	}
	httputil.WriteJSON(w, http.StatusOK, models)
}

func (h *Handler) PullModel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelID string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}

	var model *download.Model
	for _, m := range download.CuratedModels {
		if m.ID == req.ModelID {
			model = &m
			break
		}
	}
	if model == nil {
		httputil.WriteError(w, http.StatusNotFound, "unknown model")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httputil.WriteError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	writeSSEHeaders(w)

	events := make(chan download.ProgressEvent)
	if err := h.downloader.StartPull(r.Context(), model, events); err != nil {
		slog.Error("pull failed", "error", err)
		httputil.WriteError(w, http.StatusConflict, err.Error())
		return
	}

	for evt := range events {
		data, _ := json.Marshal(evt)
		io.WriteString(w, "data: "+string(data)+"\n\n")
		flusher.Flush()
	}
}

func (h *Handler) DeleteModel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.downloader.DeleteModel(id); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "model not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) OpenAIModels(w http.ResponseWriter, r *http.Request) {
	installed, err := h.downloader.InstalledModels()
	if err != nil {
		slog.Error("list installed models", "error", err)
		installed = []string{}
	}
	data := make([]map[string]any, 0, len(installed))
	for _, name := range installed {
		data = append(data, map[string]any{
			"id":      name,
			"object":  "model",
			"created": time.Now().Unix(),
			"owned_by": "local",
		})
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data":   data,
	})
}
