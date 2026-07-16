package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"localai-hub/internal/config"
	"localai-hub/internal/download"
	"localai-hub/internal/hardware"
	"localai-hub/internal/llm"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	cfg        *config.Config
	hw         *hardware.Info
	downloader *download.Downloader
	llm        *llm.Manager
	dataDir    string
}

func New(cfg *config.Config, hw *hardware.Info, d *download.Downloader, l *llm.Manager, dataDir string) *Handler {
	return &Handler{
		cfg:        cfg,
		hw:         hw,
		downloader: d,
		llm:        l,
		dataDir:    dataDir,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *Handler) SystemInfo(w http.ResponseWriter, r *http.Request) {
	installed, err := h.downloader.InstalledModels()
	if err != nil {
		installed = []string{}
	}

	h.cfg.RLock()
	info := map[string]any{
		"ram": map[string]int{
			"total": h.hw.RAMTotalGB,
			"free":  h.hw.RAMFreeGB,
		},
		"diskFreeGB": h.hw.DiskFreeGB,
		"cpu":       h.hw.CPUCores,
		"os":        h.hw.OS,
		"arch":      h.hw.Arch,
		"isFirstLaunch": h.hw.IsFirstLaunch,
		"recommendedModel": download.Recommend(h.hw.RAMFreeGB),
		"installedModels":  installed,
		"llamaServerRunning": h.llm.IsRunning(),
		"activeModel":       h.cfg.ActiveModel,
		"systemPrompt":     h.cfg.SystemPrompt,
		"temperature":      h.cfg.Temperature,
	}
	h.cfg.RUnlock()
	writeJSON(w, http.StatusOK, info)
}

func (h *Handler) ListModels(w http.ResponseWriter, r *http.Request) {
	installed, _ := h.downloader.InstalledModels()
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
	writeJSON(w, http.StatusOK, models)
}

func (h *Handler) PullModel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelID string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
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
		writeError(w, http.StatusNotFound, "unknown model")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	events := make(chan download.ProgressEvent)
	if err := h.downloader.StartPull(r.Context(), model, events); err != nil {
		slog.Error("pull failed", "error", err)
		writeError(w, http.StatusConflict, err.Error())
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
		writeError(w, http.StatusNotFound, "model not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) OpenAIModels(w http.ResponseWriter, r *http.Request) {
	installed, _ := h.downloader.InstalledModels()
	data := make([]map[string]any, 0, len(installed))
	for _, name := range installed {
		data = append(data, map[string]any{
			"id":      name,
			"object":  "model",
			"created": time.Now().Unix(),
			"owned_by": "local",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data":   data,
	})
}

func (h *Handler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot read body")
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
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	h.cfg.RLock()
	targetModel := chatReq.Model
	if targetModel == "" {
		targetModel = h.cfg.ActiveModel
	}
	h.cfg.RUnlock()

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
		installed, _ := h.downloader.InstalledModels()
		for _, name := range installed {
			modelPath = filepath.Join(h.downloader.ModelsDir(), name)
			break
		}
	}
	if modelPath == "" {
		writeError(w, http.StatusNotFound, "no model installed")
		return
	}

	if err := h.llm.Start(modelPath); err != nil {
		slog.Error("failed to start llama-server", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to start inference engine: "+err.Error())
		return
	}

	if chatReq.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
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
			writeError(w, http.StatusInternalServerError, err.Error())
		}
	}
}

func sanitizeConvID(id string) bool {
	if id == "" || strings.Contains(id, "..") || strings.Contains(id, "/") || strings.Contains(id, "\\") {
		return false
	}
	return len(id) <= 64 && len(id) > 0
}

func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	dir := filepath.Join(h.dataDir, "conversations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	convs := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			id := e.Name()[:len(e.Name())-5]
			title := id
			if data, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
				var c struct {
					Title string `json:"title"`
				}
				json.Unmarshal(data, &c)
				if c.Title != "" {
					title = c.Title
				}
			}
			convs = append(convs, map[string]any{
				"id":    id,
				"title": title,
			})
		}
	}
	writeJSON(w, http.StatusOK, convs)
}

func (h *Handler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	id := time.Now().Format("20060102150405.000")
	path := filepath.Join(h.dataDir, "conversations", id+".json")
	f, err := os.Create(path)
	if err != nil {
		slog.Error("create conversation file", "error", err)
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(map[string]any{
		"id":       id,
		"messages": []any{},
		"created":  time.Now(),
	}); err != nil {
		slog.Error("write conversation file", "error", err)
		writeError(w, http.StatusInternalServerError, "write failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !sanitizeConvID(id) {
		writeError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	path := filepath.Join(h.dataDir, "conversations", id+".json")
	f, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	defer f.Close()

	var data map[string]any
	json.NewDecoder(f).Decode(&data)
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) UpdateConversation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !sanitizeConvID(id) {
		writeError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	path := filepath.Join(h.dataDir, "conversations", id+".json")

	var body struct {
		Messages []map[string]string `json:"messages"`
		Title    string              `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	data := map[string]any{
		"id":       id,
		"messages": body.Messages,
		"title":    body.Title,
		"updated":  time.Now(),
	}
	f, err := os.Create(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(data)
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !sanitizeConvID(id) {
		writeError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	path := filepath.Join(h.dataDir, "conversations", id+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "conversation not found")
		} else {
			slog.Error("delete conversation", "error", err)
			writeError(w, http.StatusInternalServerError, "delete failed")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	h.cfg.Lock()
	defer h.cfg.Unlock()

	if v, ok := updates["port"]; ok {
		if p, ok := v.(float64); ok {
			h.cfg.Port = int(p)
		}
	}
	if v, ok := updates["theme"]; ok {
		if s, ok := v.(string); ok {
			h.cfg.Theme = s
		}
	}
	if v, ok := updates["activeModel"]; ok {
		if s, ok := v.(string); ok {
			h.cfg.ActiveModel = s
		}
	}
	if v, ok := updates["systemPrompt"]; ok {
		if s, ok := v.(string); ok {
			h.cfg.SystemPrompt = s
		}
	}
	if v, ok := updates["temperature"]; ok {
		if t, ok := v.(float64); ok {
			h.cfg.Temperature = t
		}
	}
	if v, ok := updates["maxTokens"]; ok {
		if t, ok := v.(float64); ok {
			h.cfg.MaxTokens = int(t)
		}
	}
	if v, ok := updates["contextSize"]; ok {
		if c, ok := v.(float64); ok {
			h.cfg.ContextSize = int(c)
		}
	}

	if err := h.cfg.Save(); err != nil {
		slog.Error("failed to save config", "error", err)
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
