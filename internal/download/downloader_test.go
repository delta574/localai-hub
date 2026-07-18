package download

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestModelsDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	d := New(dir)
	expected := filepath.Join(dir, "models")
	if d.ModelsDir() != expected {
		t.Errorf("expected %q, got %q", expected, d.ModelsDir())
	}
}

func TestModelPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	d := New(dir)
	m := CuratedModels[0]
	expected := filepath.Join(dir, "models", m.HFFile)
	if d.ModelPath(&m) != expected {
		t.Errorf("expected %q, got %q", expected, d.ModelPath(&m))
	}
}

func TestRecommend(t *testing.T) {
	t.Parallel()
	tests := []struct {
		freeRAM int
		minName string
	}{
		{1, "Gemma 3 1B"},
		{2, "Qwen3 1.5B"},
		{4, "Phi-4-mini 3.8B"},
		{16, "Phi-4-mini 3.8B"},
		{0, "Qwen3 1.5B"},
	}
	for _, tc := range tests {
		m := Recommend(tc.freeRAM)
		if m == nil {
			t.Fatalf("Recommend(%d) returned nil", tc.freeRAM)
		}
	}
}

func TestRecommendHandlesNegativeRAM(t *testing.T) {
	m := Recommend(-1)
	if m == nil {
		t.Fatal("expected non-nil model for negative RAM")
	}
}

func TestSetClient(t *testing.T) {
	dir := t.TempDir()
	d := New(dir)
	custom := &http.Client{Timeout: 5 * time.Second}
	d.SetClient(custom)
}

func TestStartPullConcurrentDuplicate(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "models"), 0755)
	d := New(dir)

	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	defer ts.Close()
	d.SetClient(ts.Client())

	m := CuratedModels[0]
	events := make(chan ProgressEvent, 10)

	if err := d.StartPull(context.Background(), &m, events); err != nil {
		t.Fatal(err)
	}

	err := d.StartPull(context.Background(), &m, events)
	if err == nil || !strings.Contains(err.Error(), "already in progress") {
		t.Errorf("expected 'already in progress' error, got %v", err)
	}
	go func() {
		for range events {
		}
	}()
}
