package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	mu           sync.RWMutex `json:"-"`
	filePath     string       `json:"-"`
	RokuIP       string       `json:"roku_ip"`
	RokuPassword string       `json:"roku_password"`
	ServerPort   int          `json:"server_port"`
	AutoAdvance  bool         `json:"auto_advance"`
	LastSubtitle string       `json:"last_subtitle"`
}

var defaultConfig = Config{
	ServerPort:  8080,
	AutoAdvance: true,
}

func Load(customPath ...string) (*Config, error) {
	path := "config.json"
	if len(customPath) > 0 && customPath[0] != "" {
		path = customPath[0]
	}

	cfg := defaultConfig
	cfg.filePath = path

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			_ = cfg.Save()
			return &cfg, nil
		}
		return &cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return &cfg, err
	}
	cfg.filePath = path
	return &cfg, nil
}

func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(c.filePath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	return os.WriteFile(c.filePath, data, 0600)
}

func (c *Config) Update(ip, password string, port int, autoAdvance bool) error {
	c.mu.Lock()
	if ip != "" {
		c.RokuIP = ip
	}
	if password != "" {
		c.RokuPassword = password
	}
	if port > 0 {
		c.ServerPort = port
	}
	c.AutoAdvance = autoAdvance
	c.mu.Unlock()

	return c.Save()
}

func (c *Config) Get() (ip, password string, port int, autoAdvance bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.RokuIP, c.RokuPassword, c.ServerPort, c.AutoAdvance
}
