package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Service represents a system service
type Service struct {
	Name        string
	DisplayName string
	Description string
	Executable  string
	Port        int
}

// New creates a new Service instance
func New(port int) (*Service, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks to get actual path
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	return &Service{
		Name:        "com.blocker",
		DisplayName: "Network Blocker",
		Description: "Blocks access to blacklisted websites",
		Executable:  exe,
		Port:        port,
	}, nil
}

// Install installs the service
func (s *Service) Install() error {
	switch runtime.GOOS {
	case "darwin":
		return s.installDarwin()
	case "windows":
		return s.installWindows()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Uninstall removes the service
func (s *Service) Uninstall() error {
	switch runtime.GOOS {
	case "darwin":
		return s.uninstallDarwin()
	case "windows":
		return s.uninstallWindows()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Start starts the service
func (s *Service) Start() error {
	switch runtime.GOOS {
	case "darwin":
		return s.startDarwin()
	case "windows":
		return s.startWindows()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Stop stops the service
func (s *Service) Stop() error {
	switch runtime.GOOS {
	case "darwin":
		return s.stopDarwin()
	case "windows":
		return s.stopWindows()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Status returns the service status
func (s *Service) Status() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return s.statusDarwin()
	case "windows":
		return s.statusWindows()
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// IsInstalled checks if the service is installed
func (s *Service) IsInstalled() bool {
	switch runtime.GOOS {
	case "darwin":
		return s.isInstalledDarwin()
	case "windows":
		return s.isInstalledWindows()
	default:
		return false
	}
}
