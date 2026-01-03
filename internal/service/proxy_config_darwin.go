//go:build darwin

package service

import (
	"fmt"
	"os/exec"
	"strings"
)

// Windows stubs for darwin build
func (p *ProxyConfig) enableWindows() error {
	return fmt.Errorf("Windows not supported on this platform")
}

func (p *ProxyConfig) disableWindows() error {
	return fmt.Errorf("Windows not supported on this platform")
}

func (p *ProxyConfig) isEnabledWindows() (bool, error) {
	return false, fmt.Errorf("Windows not supported on this platform")
}

// getNetworkServices returns a list of network services
func getNetworkServices() ([]string, error) {
	cmd := exec.Command("networksetup", "-listallnetworkservices")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list network services: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	services := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and the header line
		if line == "" || strings.HasPrefix(line, "An asterisk") {
			continue
		}
		// Skip disabled services (marked with *)
		if strings.HasPrefix(line, "*") {
			continue
		}
		services = append(services, line)
	}

	return services, nil
}

// enableDarwin enables the system proxy on macOS
func (p *ProxyConfig) enableDarwin() error {
	services, err := getNetworkServices()
	if err != nil {
		return err
	}

	proxyAddr := p.Host
	proxyPort := fmt.Sprintf("%d", p.Port)
	successCount := 0

	for _, service := range services {
		// Set HTTP proxy
		cmd := exec.Command("networksetup", "-setwebproxy", service, proxyAddr, proxyPort)
		if output, err := cmd.CombinedOutput(); err != nil {
			// Skip services that don't support proxies (like Bluetooth)
			if !strings.Contains(string(output), "not supported") {
				fmt.Printf("Warning: failed to set HTTP proxy for %s: %v\n", service, err)
			}
			continue
		}

		// Set HTTPS proxy
		cmd = exec.Command("networksetup", "-setsecurewebproxy", service, proxyAddr, proxyPort)
		if output, err := cmd.CombinedOutput(); err != nil {
			if !strings.Contains(string(output), "not supported") {
				fmt.Printf("Warning: failed to set HTTPS proxy for %s: %v\n", service, err)
			}
			continue
		}

		// Enable HTTP proxy
		cmd = exec.Command("networksetup", "-setwebproxystate", service, "on")
		cmd.Run()

		// Enable HTTPS proxy
		cmd = exec.Command("networksetup", "-setsecurewebproxystate", service, "on")
		cmd.Run()

		fmt.Printf("Configured proxy for: %s\n", service)
		successCount++
	}

	if successCount == 0 {
		return fmt.Errorf("failed to configure proxy for any network service")
	}

	return nil
}

// disableDarwin disables the system proxy on macOS
func (p *ProxyConfig) disableDarwin() error {
	services, err := getNetworkServices()
	if err != nil {
		return err
	}

	for _, service := range services {
		// Disable HTTP proxy
		cmd := exec.Command("networksetup", "-setwebproxystate", service, "off")
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: failed to disable HTTP proxy for %s: %v\n", service, err)
		}

		// Disable HTTPS proxy
		cmd = exec.Command("networksetup", "-setsecurewebproxystate", service, "off")
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: failed to disable HTTPS proxy for %s: %v\n", service, err)
		}
	}

	return nil
}

// isEnabledDarwin checks if the system proxy is enabled on macOS
func (p *ProxyConfig) isEnabledDarwin() (bool, error) {
	services, err := getNetworkServices()
	if err != nil {
		return false, err
	}

	expectedAddr := fmt.Sprintf("%s:%d", p.Host, p.Port)

	for _, service := range services {
		// Check HTTP proxy
		cmd := exec.Command("networksetup", "-getwebproxy", service)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		outputStr := string(output)
		if strings.Contains(outputStr, "Enabled: Yes") &&
			strings.Contains(outputStr, fmt.Sprintf("Server: %s", p.Host)) &&
			strings.Contains(outputStr, fmt.Sprintf("Port: %d", p.Port)) {
			return true, nil
		}

		// Check HTTPS proxy
		cmd = exec.Command("networksetup", "-getsecurewebproxy", service)
		output, err = cmd.Output()
		if err != nil {
			continue
		}

		outputStr = string(output)
		if strings.Contains(outputStr, "Enabled: Yes") &&
			strings.Contains(outputStr, fmt.Sprintf("Server: %s", p.Host)) &&
			strings.Contains(outputStr, fmt.Sprintf("Port: %d", p.Port)) {
			return true, nil
		}
	}

	// Suppress unused variable warning
	_ = expectedAddr

	return false, nil
}
