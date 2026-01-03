//go:build windows

package service

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// Darwin stubs for windows build
func (p *ProxyConfig) enableDarwin() error {
	return fmt.Errorf("macOS not supported on this platform")
}

func (p *ProxyConfig) disableDarwin() error {
	return fmt.Errorf("macOS not supported on this platform")
}

func (p *ProxyConfig) isEnabledDarwin() (bool, error) {
	return false, fmt.Errorf("macOS not supported on this platform")
}

const internetSettingsKey = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

// enableWindows enables the system proxy on Windows
func (p *ProxyConfig) enableWindows() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	// Set proxy server address
	proxyServer := fmt.Sprintf("%s:%d", p.Host, p.Port)
	if err := key.SetStringValue("ProxyServer", proxyServer); err != nil {
		return fmt.Errorf("failed to set ProxyServer: %w", err)
	}

	// Enable proxy
	if err := key.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("failed to enable proxy: %w", err)
	}

	// Set bypass list (localhost and local addresses)
	bypassList := "localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*;<local>"
	if err := key.SetStringValue("ProxyOverride", bypassList); err != nil {
		return fmt.Errorf("failed to set ProxyOverride: %w", err)
	}

	// Notify the system that proxy settings have changed
	notifyProxyChange()

	return nil
}

// disableWindows disables the system proxy on Windows
func (p *ProxyConfig) disableWindows() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	// Disable proxy
	if err := key.SetDWordValue("ProxyEnable", 0); err != nil {
		return fmt.Errorf("failed to disable proxy: %w", err)
	}

	// Notify the system that proxy settings have changed
	notifyProxyChange()

	return nil
}

// isEnabledWindows checks if the system proxy is enabled on Windows
func (p *ProxyConfig) isEnabledWindows() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	// Check if proxy is enabled
	proxyEnable, _, err := key.GetIntegerValue("ProxyEnable")
	if err != nil {
		return false, nil
	}

	if proxyEnable != 1 {
		return false, nil
	}

	// Check if our proxy is configured
	proxyServer, _, err := key.GetStringValue("ProxyServer")
	if err != nil {
		return false, nil
	}

	expectedProxy := fmt.Sprintf("%s:%d", p.Host, p.Port)
	return proxyServer == expectedProxy, nil
}

// notifyProxyChange notifies Windows that proxy settings have changed
func notifyProxyChange() {
	// This would ideally call InternetSetOption with INTERNET_OPTION_SETTINGS_CHANGED
	// and INTERNET_OPTION_REFRESH, but for simplicity we'll skip it
	// Most applications will pick up the change on next connection
}
