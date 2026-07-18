package api

import (
	"encoding/json"
	"net/http"

	"github.com/delta574/localai-hub/internal/httputil"
)

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
