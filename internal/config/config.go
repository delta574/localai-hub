package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/delta574/localai-hub/internal/auth"
)

type Config struct {
	mu sync.RWMutex

	Port            int               `json:"port"`
	Theme           string            `json:"theme"`
	ActiveModel     string            `json:"activeModel"`
	SystemPrompt    string            `json:"systemPrompt"`
	Temperature     float64           `json:"temperature"`
	MaxTokens       int               `json:"maxTokens"`
	ContextSize     int               `json:"contextSize"`
	LlamaServerPath string            `json:"llamaServerPath"`
	ApiKeys         []auth.ApiKeyEntry `json:"apiKeys"`
	DataDir         string            `json:"-"`
}

func Default(dataDir string) *Config {
	return &Config{
		Port:         8080,
		Theme:        "dark",
		SystemPrompt: "You are a helpful assistant.",
		Temperature:  0.7,
		MaxTokens:    2048,
		ContextSize:  4096,
		DataDir:      dataDir,
	}
}

func Load(dataDir string) (*Config, error) {
	path := filepath.Join(dataDir, "config.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &Config{DataDir: dataDir}
	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.saveLocked()
}

func (c *Config) saveLocked() error {
	path := filepath.Join(c.DataDir, "config.json")
	tmpPath := path + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}

// --- typed accessors ---

func (c *Config) GetActiveModel() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ActiveModel
}

func (c *Config) SetActiveModel(m string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ActiveModel = m
}

func (c *Config) GetSystemPrompt() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SystemPrompt
}

func (c *Config) SetSystemPrompt(p string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SystemPrompt = p
}

func (c *Config) GetTemperature() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Temperature
}

func (c *Config) SetTemperature(t float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Temperature = t
}

func (c *Config) Update(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn()
	c.saveLocked()
}

func (c *Config) View(fn func()) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	fn()
}

func (c *Config) ViewActiveModel() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ActiveModel
}

// --- API key methods ---

func (c *Config) HasApiKeys() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.ApiKeys) > 0
}

func (c *Config) VerifyApiKey(rawKey string) *auth.ApiKeyEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.ApiKeys {
		if !c.ApiKeys[i].Enabled {
			continue
		}
		if auth.Verify(rawKey, c.ApiKeys[i].Hash) {
			c.ApiKeys[i].LastUsedAt = time.Now()
			c.saveLocked()
			return &c.ApiKeys[i]
		}
	}
	return nil
}

func (c *Config) AddApiKey(name string) (string, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	rawKey, hash, err := auth.GenerateKey()
	if err != nil {
		return "", "", err
	}

	entry := auth.ApiKeyEntry{
		ID:        fmt.Sprintf("key_%d", time.Now().UnixNano()),
		Name:      name,
		Hash:      hash,
		CreatedAt: time.Now(),
		Enabled:   true,
	}

	c.ApiKeys = append(c.ApiKeys, entry)
	if err := c.saveLocked(); err != nil {
		return "", "", err
	}

	return entry.ID, rawKey, nil
}

func (c *Config) DeleteApiKey(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.ApiKeys {
		if c.ApiKeys[i].ID == id {
			c.ApiKeys = append(c.ApiKeys[:i], c.ApiKeys[i+1:]...)
			return c.saveLocked()
		}
	}
	return fmt.Errorf("api key not found: %s", id)
}

func (c *Config) ToggleApiKey(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.ApiKeys {
		if c.ApiKeys[i].ID == id {
			c.ApiKeys[i].Enabled = !c.ApiKeys[i].Enabled
			return c.saveLocked()
		}
	}
	return fmt.Errorf("api key not found: %s", id)
}

func (c *Config) GetApiKeys() []auth.ApiKeyEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]auth.ApiKeyEntry, len(c.ApiKeys))
	copy(result, c.ApiKeys)
	return result
}

func (c *Config) GetPort() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Port
}
