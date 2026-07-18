package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/delta574/localai-hub/internal/httputil"

	"github.com/go-chi/chi/v5"
)

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
