package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default("/tmp/data")
	if cfg.Port != 8080 {
		t.Errorf("expected 8080, got %d", cfg.Port)
	}
	if cfg.Theme != "dark" {
		t.Errorf("expected dark, got %q", cfg.Theme)
	}
	if cfg.DataDir != "/tmp/data" {
		t.Errorf("expected /tmp/data, got %q", cfg.DataDir)
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	cfg := Default(dir)
	cfg.ActiveModel = "test-model"
	cfg.SystemPrompt = "custom prompt"
	cfg.Temperature = 0.5
	cfg.MaxTokens = 100
	cfg.ContextSize = 2048

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if got.ActiveModel != "test-model" {
		t.Errorf("expected test-model, got %q", got.ActiveModel)
	}
	if got.SystemPrompt != "custom prompt" {
		t.Errorf("expected custom prompt, got %q", got.SystemPrompt)
	}
	if got.Temperature != 0.5 {
		t.Errorf("expected 0.5, got %f", got.Temperature)
	}
	if got.MaxTokens != 100 {
		t.Errorf("expected 100, got %d", got.MaxTokens)
	}
	if got.ContextSize != 2048 {
		t.Errorf("expected 2048, got %d", got.ContextSize)
	}
}

func TestLoadMissing(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "nonexistent-dir-for-test")
	os.RemoveAll(dir) // ensure it doesn't exist

	_, err := Load(dir)
	if err == nil {
		t.Error("expected error for missing config")
	}
}

func TestActiveModelAccessors(t *testing.T) {
	cfg := Default("/tmp")
	cfg.SetActiveModel("model-a")
	if got := cfg.GetActiveModel(); got != "model-a" {
		t.Errorf("expected model-a, got %q", got)
	}
}

func TestSystemPromptAccessors(t *testing.T) {
	cfg := Default("/tmp")
	cfg.SetSystemPrompt("hello")
	if got := cfg.GetSystemPrompt(); got != "hello" {
		t.Errorf("expected hello, got %q", got)
	}
}

func TestTemperatureAccessors(t *testing.T) {
	cfg := Default("/tmp")
	cfg.SetTemperature(0.9)
	if got := cfg.GetTemperature(); got != 0.9 {
		t.Errorf("expected 0.9, got %f", got)
	}
}

func TestAddGetDeleteApiKey(t *testing.T) {
	dir := t.TempDir()
	cfg := Default(dir)
	if cfg.HasApiKeys() {
		t.Error("expected no keys initially")
	}

	id, raw, err := cfg.AddApiKey("test-key")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.HasApiKeys() {
		t.Error("expected keys after add")
	}

	keys := cfg.GetApiKeys()
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].ID != id {
		t.Errorf("expected id %q, got %q", id, keys[0].ID)
	}
	if keys[0].Name != "test-key" {
		t.Errorf("expected name test-key, got %q", keys[0].Name)
	}
	if !keys[0].Enabled {
		t.Error("expected key to be enabled")
	}

	if entry := cfg.VerifyApiKey(raw); entry == nil || entry.ID != id {
		t.Error("VerifyApiKey should return the entry for valid key")
	}

	if entry := cfg.VerifyApiKey(raw + "bad"); entry != nil {
		t.Error("VerifyApiKey should return nil for invalid key")
	}

	if err := cfg.DeleteApiKey(id); err != nil {
		t.Fatal(err)
	}
	if cfg.HasApiKeys() {
		t.Error("expected no keys after delete")
	}
}

func TestToggleApiKey(t *testing.T) {
	dir := t.TempDir()
	cfg := Default(dir)
	id, raw, err := cfg.AddApiKey("toggle-test")
	if err != nil {
		t.Fatal(err)
	}

	if err := cfg.ToggleApiKey(id); err != nil {
		t.Fatal(err)
	}
	keys := cfg.GetApiKeys()
	if len(keys) != 1 || keys[0].Enabled {
		t.Error("expected key to be disabled after toggle")
	}
	if entry := cfg.VerifyApiKey(raw); entry != nil {
		t.Error("VerifyApiKey should return nil for disabled key")
	}

	if err := cfg.ToggleApiKey(id); err != nil {
		t.Fatal(err)
	}
	keys = cfg.GetApiKeys()
	if len(keys) != 1 || !keys[0].Enabled {
		t.Error("expected key to be enabled after second toggle")
	}
	if entry := cfg.VerifyApiKey(raw); entry == nil {
		t.Error("VerifyApiKey should return entry for re-enabled key")
	}
}

func TestUpdate(t *testing.T) {
	dir := t.TempDir()
	cfg := Default(dir)
	cfg.Update(func() {
		cfg.ActiveModel = "updated-model"
		cfg.Temperature = 0.1
	})
	if cfg.GetActiveModel() != "updated-model" {
		t.Errorf("expected updated-model, got %q", cfg.GetActiveModel())
	}
	if cfg.GetTemperature() != 0.1 {
		t.Errorf("expected 0.1, got %f", cfg.GetTemperature())
	}
}
