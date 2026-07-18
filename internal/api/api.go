package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/delta574/localai-hub/internal/auth"
	"github.com/delta574/localai-hub/internal/config"
	"github.com/delta574/localai-hub/internal/download"
	"github.com/delta574/localai-hub/internal/hardware"
	"github.com/delta574/localai-hub/internal/httputil"

	"github.com/go-chi/chi/v5"
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

func writeSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
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
		"diskFreeGB": h.hw.DiskFreeGB,
		"cpu":       h.hw.CPUCores,
		"os":        h.hw.OS,
		"arch":      h.hw.Arch,
		"isFirstLaunch": h.hw.IsFirstLaunch,
		"recommendedModel": download.Recommend(h.hw.RAMFreeGB),
		"installedModels":  installed,
		"llamaServerRunning": h.llm.IsRunning(),
		"activeModel":   h.cfg.ViewActiveModel(),
		"systemPrompt": h.cfg.GetSystemPrompt(),
		"temperature":  h.cfg.GetTemperature(),
	}
	httputil.WriteJSON(w, http.StatusOK, info)
}

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
		targetModel = h.cfg.ViewActiveModel()
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
		httputil.WriteJSON(w, http.StatusOK, []any{})
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
	httputil.WriteJSON(w, http.StatusOK, convs)
}

func (h *Handler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	id := time.Now().Format("20060102150405.000")
	path := filepath.Join(h.dataDir, "conversations", id+".json")
	f, err := os.Create(path)
	if err != nil {
		slog.Error("create conversation file", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "create failed")
		return
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(map[string]any{
		"id":       id,
		"messages": []any{},
		"created":  time.Now(),
	}); err != nil {
		slog.Error("write conversation file", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "write failed")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !sanitizeConvID(id) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	path := filepath.Join(h.dataDir, "conversations", id+".json")
	f, err := os.Open(path)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "conversation not found")
		return
	}
	defer f.Close()

	var data map[string]any
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		slog.Error("decode conversation", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "read failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, data)
}

func (h *Handler) UpdateConversation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !sanitizeConvID(id) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	path := filepath.Join(h.dataDir, "conversations", id+".json")

	var body struct {
		Messages []map[string]string `json:"messages"`
		Title    string              `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON")
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
		httputil.WriteError(w, http.StatusInternalServerError, "save failed")
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(data)
	httputil.WriteJSON(w, http.StatusOK, data)
}

func (h *Handler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !sanitizeConvID(id) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}
	path := filepath.Join(h.dataDir, "conversations", id+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			httputil.WriteError(w, http.StatusNotFound, "conversation not found")
		} else {
			slog.Error("delete conversation", "error", err)
			httputil.WriteError(w, http.StatusInternalServerError, "delete failed")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	h.cfg.Update(func() {
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
	})

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func toKeyResp(entry auth.ApiKeyEntry) map[string]any {
	return map[string]any{
		"id":         entry.ID,
		"name":       entry.Name,
		"prefix":     auth.KeyPrefixDisplay("lah_" + entry.Hash[:4]),
		"createdAt":  entry.CreatedAt,
		"lastUsedAt": entry.LastUsedAt,
		"enabled":    entry.Enabled,
	}
}

func (h *Handler) ListApiKeys(w http.ResponseWriter, r *http.Request) {
	entries := h.cfg.GetApiKeys()
	keys := make([]map[string]any, 0, len(entries))
	for _, k := range entries {
		keys = append(keys, toKeyResp(k))
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

func (h *Handler) CreateApiKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	id, rawKey, err := h.cfg.AddApiKey(req.Name)

	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create key")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]string{"id": id, "key": rawKey})
}

func (h *Handler) DeleteApiKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.cfg.DeleteApiKey(id)

	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "key not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ToggleApiKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.cfg.ToggleApiKey(id)

	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "key not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
