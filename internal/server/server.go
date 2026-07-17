package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/delta574/localai-hub/internal/api"
	"github.com/delta574/localai-hub/internal/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	cfg      *config.Config
	router   *chi.Mux
	api      *api.Handler
	staticFS http.FileSystem
}

func New(cfg *config.Config, staticFS http.FileSystem) *Server {
	s := &Server{
		cfg:      cfg,
		router:   chi.NewRouter(),
		staticFS: staticFS,
	}

	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(securityHeaders)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	}))

	return s
}

func (s *Server) RegisterAPI(h *api.Handler) {
	s.api = h

	s.router.Route("/api", func(r chi.Router) {
		r.Use(maxBodySize)
		r.Get("/system/info", h.SystemInfo)
		r.Get("/models", h.ListModels)
		r.Post("/models/pull", h.PullModel)
		r.Delete("/models/{id}", h.DeleteModel)
		r.Get("/conversations", h.ListConversations)
		r.Post("/conversations", h.CreateConversation)
		r.Get("/conversations/{id}", h.GetConversation)
		r.Put("/conversations/{id}", h.UpdateConversation)
		r.Delete("/conversations/{id}", h.DeleteConversation)
		r.Put("/config", h.UpdateConfig)
		r.Get("/keys", h.ListApiKeys)
		r.Post("/keys", h.CreateApiKey)
		r.Delete("/keys/{id}", h.DeleteApiKey)
		r.Put("/keys/{id}/toggle", h.ToggleApiKey)
	})

	s.router.Route("/v1", func(r chi.Router) {
		r.Use(maxBodySize)
		r.Use(s.apiKeyMiddleware)
		r.Post("/chat/completions", h.ChatCompletions)
		r.Get("/models", h.OpenAIModels)
	})

	if s.staticFS != nil {
		s.router.Get("/*", s.serveStatic)
	}
}

func (s *Server) apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.cfg.RLock()
		hasKeys := s.cfg.HasApiKeys()
		s.cfg.RUnlock()

		if !hasKeys {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing API key"})
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")

		s.cfg.Lock()
		entry := s.cfg.VerifyApiKey(token)
		s.cfg.Unlock()

		if entry == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid API key"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) Router() *chi.Mux {
	return s.router
}

func maxBodySize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; img-src 'self' data:; style-src 'self' 'unsafe-inline'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	reqPath := strings.TrimPrefix(r.URL.Path, "/")
	if reqPath == "" {
		reqPath = "index.html"
	}

	if f, err := s.staticFS.Open(reqPath); err == nil {
		f.Close()
		http.FileServer(s.staticFS).ServeHTTP(w, r)
		return
	}

	if content, err := s.staticFS.Open("index.html"); err == nil {
		content.Close()
		origPath := r.URL.Path
		r.URL.Path = "/"
		http.FileServer(s.staticFS).ServeHTTP(w, r)
		r.URL.Path = origPath
		return
	}

	http.NotFound(w, r)
}
