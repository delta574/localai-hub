package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/delta574/localai-hub/internal/config"
	"github.com/delta574/localai-hub/internal/httputil"
)

func TestSecurityHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected X-Content-Type-Options: nosniff")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("expected X-Frame-Options: DENY")
	}
	if w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Error("expected Referrer-Policy: no-referrer")
	}
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected Content-Security-Policy")
	}
	// script-src is omitted — SvelteKit embeds per-page SHA-256 hashes via <meta> tags
	if strings.Contains(csp, "script-src") {
		t.Error("expected no script-src in server CSP (handled by SvelteKit meta tags)")
	}
}

func TestMaxBodySize(t *testing.T) {
	// Verify middleware wraps body reader — body under limit reads fine
	handler := maxBodySize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, 100)
		n, _ := r.Body.Read(body)
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"size": fmt.Sprintf("%d", n)})
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", strings.NewReader("small body"))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMaxBodySize_SmallOK(t *testing.T) {
	w := httptest.NewRecorder()
	handler := maxBodySize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"ok": "true"})
	}))

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"ok":true}`))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestNewServer(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default(dir)

	s := New(cfg, nil)
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.Router() == nil {
		t.Error("expected non-nil router")
	}
}

func TestRateLimiter_Check(t *testing.T) {
	rl := newRateLimiter(3, time.Minute)
	ip := "192.168.1.1"

	for i := 0; i < 3; i++ {
		if !rl.check(ip) {
			t.Errorf("expected allow on request %d", i+1)
		}
	}
	if rl.check(ip) {
		t.Error("expected block after limit reached")
	}
}

func TestRateLimiter_Expiry(t *testing.T) {
	rl := newRateLimiter(1, 50*time.Millisecond)
	ip := "192.168.1.1"

	if !rl.check(ip) {
		t.Error("expected allow on first request")
	}
	// Should be blocked immediately
	if rl.check(ip) {
		t.Error("expected block on second request")
	}
	// Wait for window to expire
	time.Sleep(100 * time.Millisecond)
	if !rl.check(ip) {
		t.Error("expected allow after window expiry")
	}
}

func TestServerShutdown(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default(dir)
	s := New(cfg, nil)

	httpServer := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: s.Router(),
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go httpServer.Serve(ln)
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}
