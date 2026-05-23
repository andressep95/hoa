package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFile = "config.json"

// Config is the top-level persisted configuration.
type Config struct {
	ActiveProvider string                    `json:"activeProvider"`
	Providers      map[string]ProviderConfig `json:"providers"`
	Memory         MemoryConfig              `json:"memory,omitempty"`
	Database       DatabaseConfig            `json:"database,omitempty"`
	Harness        HarnessConfig             `json:"harness"`
}

// ProviderConfig holds credentials and model selection for one provider.
type ProviderConfig struct {
	APIKey  string       `json:"apiKey,omitempty"`
	BaseURL string       `json:"baseUrl,omitempty"`
	Models  ModelsConfig `json:"models"`
}

// ModelsConfig holds the dual-model selection.
type ModelsConfig struct {
	Base     string `json:"base"`
	Planning string `json:"planning"`
}

// MemoryConfig holds the optional memory provider settings.
type MemoryConfig struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider,omitempty"` // "oracle" (future: "postgres", "chroma")
	DSN      string `json:"dsn,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	APIKey   string `json:"apiKey,omitempty"` // project API key for memory server
}

// DatabaseConfig holds vector store connection info (deprecated, use MemoryConfig).
type DatabaseConfig struct {
	URL      string `json:"url,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

// HarnessConfig holds harness behavior flags.
type HarnessConfig struct {
	VerifyAfterWrite bool    `json:"verifyAfterWrite"`
	SDDEnforced      bool    `json:"sddEnforced"`
	MaxRetries       int     `json:"maxRetries"`
	CompactThreshold float64 `json:"compactThreshold"`
}

// Dir returns the HOA config directory (~/.hoa).
func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".hoa")
}

// Load reads config from ~/.hoa/config.json. Returns os.ErrNotExist if missing.
func Load() (*Config, error) {
	path := filepath.Join(Dir(), configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	// Decrypt API keys in memory
	key, err := LoadOrCreateKey(Dir())
	if err != nil {
		return nil, fmt.Errorf("loading keyring: %w", err)
	}
	for name, p := range cfg.Providers {
		if IsEncrypted(p.APIKey) {
			plain, err := Decrypt(key, p.APIKey)
			if err != nil {
				return nil, fmt.Errorf("decrypting %s api key: %w", name, err)
			}
			p.APIKey = plain
			cfg.Providers[name] = p
		}
	}
	return &cfg, nil
}

// Save writes config to ~/.hoa/config.json, encrypting API keys on disk.
func Save(cfg *Config) error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	key, err := LoadOrCreateKey(dir)
	if err != nil {
		return fmt.Errorf("loading keyring: %w", err)
	}
	// Copy and encrypt API keys for persistence
	toSave := *cfg
	toSave.Providers = make(map[string]ProviderConfig, len(cfg.Providers))
	for name, p := range cfg.Providers {
		if p.APIKey != "" && !IsEncrypted(p.APIKey) {
			enc, err := Encrypt(key, p.APIKey)
			if err != nil {
				return fmt.Errorf("encrypting %s api key: %w", name, err)
			}
			p.APIKey = enc
		}
		toSave.Providers[name] = p
	}
	data, err := json.MarshalIndent(toSave, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, configFile), data, 0600)
}

// ActiveProviderConfig returns the config for the currently active provider.
func (c *Config) ActiveProviderConfig() (ProviderConfig, bool) {
	p, ok := c.Providers[c.ActiveProvider]
	return p, ok
}

// APIKey returns the API key for the active provider, falling back to env vars.
func (c *Config) APIKey() string {
	if p, ok := c.ActiveProviderConfig(); ok && p.APIKey != "" {
		return p.APIKey
	}
	// Env var fallback
	envVars := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"openai":    "OPENAI_API_KEY",
		"google":    "GOOGLE_API_KEY",
	}
	if envName, ok := envVars[c.ActiveProvider]; ok {
		return os.Getenv(envName)
	}
	return ""
}
