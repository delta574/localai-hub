package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	mu sync.RWMutex

	Port            int     `json:"port"`
	Theme           string  `json:"theme"`
	ActiveModel     string  `json:"activeModel"`
	SystemPrompt    string  `json:"systemPrompt"`
	Temperature     float64 `json:"temperature"`
	MaxTokens       int     `json:"maxTokens"`
	ContextSize     int     `json:"contextSize"`
	LlamaServerPath string  `json:"llamaServerPath"`
	DataDir         string  `json:"-"`
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
