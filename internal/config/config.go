package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DefaultBaseURL     = "https://api.gdpdf.com/api/"
	DefaultToolURL     = "https://api.gdpdf.com/api/"
	DefaultDownloadURL = "https://res.gdpdf.com/"
)

type Config struct {
	Token       string `json:"token,omitempty"`
	DeviceID    string `json:"device_id,omitempty"`
	BaseURL     string `json:"base_url,omitempty"`
	ToolURL     string `json:"tool_url,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	Format      string `json:"format,omitempty"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pdf-cli")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func Load() *Config {
	cfg := &Config{
		BaseURL:     DefaultBaseURL,
		ToolURL:     DefaultToolURL,
		DownloadURL: DefaultDownloadURL,
		Format:      "pretty",
	}
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, cfg)
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.ToolURL == "" {
		cfg.ToolURL = DefaultToolURL
	}
	if cfg.DownloadURL == "" {
		cfg.DownloadURL = DefaultDownloadURL
	}
	return cfg
}

func (c *Config) Save() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0600)
}

func (c *Config) GetBaseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}

func (c *Config) GetToolURL() string {
	if c.ToolURL != "" {
		return c.ToolURL
	}
	return DefaultToolURL
}

func (c *Config) GetDownloadURL() string {
	if c.DownloadURL != "" {
		return c.DownloadURL
	}
	return DefaultDownloadURL
}
