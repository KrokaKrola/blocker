package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Proxy   ProxyConfig   `yaml:"proxy"`
	Blacklist []string    `yaml:"blacklist"`
	Logging LoggingConfig `yaml:"logging"`
}

// ProxyConfig represents proxy server settings
type ProxyConfig struct {
	Port int    `yaml:"port"`
	Bind string `yaml:"bind"`
}

// LoggingConfig represents logging settings
type LoggingConfig struct {
	Level      string `yaml:"level"`
	LogBlocked bool   `yaml:"log_blocked"`
	LogAllowed bool   `yaml:"log_allowed"`
}

// Manager handles configuration loading and access
type Manager struct {
	config     *Config
	configPath string
	mu         sync.RWMutex
}

// NewManager creates a new configuration manager
func NewManager(configPath string) *Manager {
	return &Manager{
		configPath: configPath,
	}
}

// Load reads and parses the configuration file
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if cfg.Proxy.Port == 0 {
		cfg.Proxy.Port = 8080
	}
	if cfg.Proxy.Bind == "" {
		cfg.Proxy.Bind = "127.0.0.1"
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}

	m.config = &cfg
	return nil
}

// Get returns the current configuration (thread-safe)
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetBlacklist returns the current blacklist (thread-safe)
func (m *Manager) GetBlacklist() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.config == nil {
		return nil
	}
	return m.config.Blacklist
}

// AddToBlacklist adds a domain to the blacklist and saves
func (m *Manager) AddToBlacklist(domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already exists
	for _, d := range m.config.Blacklist {
		if d == domain {
			return fmt.Errorf("domain %s already in blacklist", domain)
		}
	}

	m.config.Blacklist = append(m.config.Blacklist, domain)
	return m.save()
}

// RemoveFromBlacklist removes a domain from the blacklist and saves
func (m *Manager) RemoveFromBlacklist(domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	found := false
	newList := make([]string, 0, len(m.config.Blacklist))
	for _, d := range m.config.Blacklist {
		if d == domain {
			found = true
			continue
		}
		newList = append(newList, d)
	}

	if !found {
		return fmt.Errorf("domain %s not found in blacklist", domain)
	}

	m.config.Blacklist = newList
	return m.save()
}

// save writes the current configuration to file
func (m *Manager) save() error {
	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(m.configPath, data, 0644)
}

// GetConfigPath returns the default config path
func GetConfigPath() string {
	// First check for config in current directory
	if _, err := os.Stat("configs/config.yaml"); err == nil {
		return "configs/config.yaml"
	}

	// Then check executable directory
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		configPath := filepath.Join(exeDir, "configs", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Check home directory
	home, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(home, ".blocker", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Default to current directory
	return "configs/config.yaml"
}

// EnsureConfigExists creates default config if it doesn't exist
func EnsureConfigExists(configPath string) error {
	if _, err := os.Stat(configPath); err == nil {
		return nil // Already exists
	}

	// Create directory if needed
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config
	defaultConfig := Config{
		Proxy: ProxyConfig{
			Port: 8080,
			Bind: "127.0.0.1",
		},
		Blacklist: []string{
			"facebook.com",
			"twitter.com",
			"instagram.com",
		},
		Logging: LoggingConfig{
			Level:      "info",
			LogBlocked: true,
			LogAllowed: false,
		},
	}

	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}
