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

func (c *Config) Lock()    { c.mu.Lock() }
func (c *Config) Unlock()  { c.mu.Unlock() }
func (c *Config) RLock()   { c.mu.RLock() }
func (c *Config) RUnlock() { c.mu.RUnlock() }

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

func (c *Config) HasApiKeys() bool {
	return len(c.ApiKeys) > 0
}

func (c *Config) VerifyApiKey(rawKey string) *auth.ApiKeyEntry {
	for i := range c.ApiKeys {
		if !c.ApiKeys[i].Enabled {
			continue
		}
		if auth.Verify(rawKey, c.ApiKeys[i].Hash) {
			c.ApiKeys[i].LastUsedAt = time.Now()
			c.Save()
			return &c.ApiKeys[i]
		}
	}
	return nil
}

func (c *Config) AddApiKey(name string) (string, string, error) {
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
	if err := c.Save(); err != nil {
		return "", "", err
	}

	return entry.ID, rawKey, nil
}

func (c *Config) DeleteApiKey(id string) error {
	for i := range c.ApiKeys {
		if c.ApiKeys[i].ID == id {
			c.ApiKeys = append(c.ApiKeys[:i], c.ApiKeys[i+1:]...)
			return c.Save()
		}
	}
	return fmt.Errorf("api key not found: %s", id)
}

func (c *Config) ToggleApiKey(id string) error {
	for i := range c.ApiKeys {
		if c.ApiKeys[i].ID == id {
			c.ApiKeys[i].Enabled = !c.ApiKeys[i].Enabled
			return c.Save()
		}
	}
	return fmt.Errorf("api key not found: %s", id)
}
