package service

import (
	"fmt"
	"runtime"
)

// ProxyConfig handles system proxy configuration
type ProxyConfig struct {
	Host string
	Port int
}

// NewProxyConfig creates a new ProxyConfig instance
func NewProxyConfig(host string, port int) *ProxyConfig {
	return &ProxyConfig{
		Host: host,
		Port: port,
	}
}

// Enable sets the system proxy to use our blocker
func (p *ProxyConfig) Enable() error {
	switch runtime.GOOS {
	case "darwin":
		return p.enableDarwin()
	case "windows":
		return p.enableWindows()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Disable removes the system proxy configuration
func (p *ProxyConfig) Disable() error {
	switch runtime.GOOS {
	case "darwin":
		return p.disableDarwin()
	case "windows":
		return p.disableWindows()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// IsEnabled checks if the system proxy is currently enabled
func (p *ProxyConfig) IsEnabled() (bool, error) {
	switch runtime.GOOS {
	case "darwin":
		return p.isEnabledDarwin()
	case "windows":
		return p.isEnabledWindows()
	default:
		return false, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
