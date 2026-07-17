package server

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/delta574/localai-hub/internal/api"
	"github.com/delta574/localai-hub/internal/config"
	"github.com/delta574/localai-hub/internal/httputil"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type rateLimiter struct {
	mu     sync.Mutex
	visits map[string][]time.Time
	limit  int
	window time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visits: make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(rl.window)
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.window)
		for ip, times := range rl.visits {
			keep := 0
			for _, t := range times {
				if t.After(cutoff) {
					times[keep] = t
					keep++
				}
			}
			if keep == 0 {
				delete(rl.visits, ip)
			} else {
				rl.visits[ip] = times[:keep]
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) check(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-rl.window)
	times := rl.visits[ip]
	keep := 0
	for _, t := range times {
		if t.After(cutoff) {
			times[keep] = t
			keep++
		}
	}
	times = times[:keep]
	if len(times) >= rl.limit {
		rl.visits[ip] = times
		return false
	}
	rl.visits[ip] = append(times, now)
	return true
}

func rateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	rl := newRateLimiter(limit, window)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if host, _, err := net.SplitHostPort(ip); err == nil {
				ip = host
			}
			if !rl.check(ip) {
				httputil.WriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

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
		AllowedOrigins:   []string{"http://localhost:8080", "http://127.0.0.1:8080", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
	}))
	s.router.Use(maxBodySize)

	return s
}

func (s *Server) RegisterAPI(h *api.Handler) {
	s.api = h

	s.router.Route("/api", func(r chi.Router) {
		r.Use(rateLimit(100, time.Minute))
		r.Get("/system/info", h.SystemInfo)
		r.Get("/models", h.ListModels)
		r.With(rateLimit(20, time.Minute)).Post("/models/pull", h.PullModel)
		r.With(rateLimit(30, time.Minute)).Delete("/models/{id}", h.DeleteModel)
		r.Get("/conversations", h.ListConversations)
		r.With(rateLimit(60, time.Minute)).Post("/conversations", h.CreateConversation)
		r.Get("/conversations/{id}", h.GetConversation)
		r.With(rateLimit(60, time.Minute)).Put("/conversations/{id}", h.UpdateConversation)
		r.With(rateLimit(30, time.Minute)).Delete("/conversations/{id}", h.DeleteConversation)
		r.With(rateLimit(30, time.Minute)).Put("/config", h.UpdateConfig)
		r.Get("/keys", h.ListApiKeys)
		r.With(rateLimit(60, time.Minute)).Post("/keys", h.CreateApiKey)
		r.With(rateLimit(30, time.Minute)).Delete("/keys/{id}", h.DeleteApiKey)
		r.With(rateLimit(30, time.Minute)).Put("/keys/{id}/toggle", h.ToggleApiKey)
	})

	s.router.Route("/v1", func(r chi.Router) {
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
		hasKeys := s.cfg.HasApiKeys()

		if !hasKeys {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			httputil.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing API key"})
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")

		entry := s.cfg.VerifyApiKey(token)

		if entry == nil {
			httputil.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid API key"})
			return
		}

		next.ServeHTTP(w, r)
	})
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
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'")
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
