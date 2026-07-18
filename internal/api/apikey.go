package api

import (
	"encoding/json"
	"net/http"

	"github.com/delta574/localai-hub/internal/auth"
	"github.com/delta574/localai-hub/internal/httputil"

	"github.com/go-chi/chi/v5"
)

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
