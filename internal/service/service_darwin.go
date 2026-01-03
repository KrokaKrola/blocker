//go:build darwin

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// Windows stubs for darwin build
func (s *Service) installWindows() error {
	return fmt.Errorf("Windows not supported on this platform")
}

func (s *Service) uninstallWindows() error {
	return fmt.Errorf("Windows not supported on this platform")
}

func (s *Service) startWindows() error {
	return fmt.Errorf("Windows not supported on this platform")
}

func (s *Service) stopWindows() error {
	return fmt.Errorf("Windows not supported on this platform")
}

func (s *Service) statusWindows() (string, error) {
	return "", fmt.Errorf("Windows not supported on this platform")
}

func (s *Service) isInstalledWindows() bool {
	return false
}

const launchAgentTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Name}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.Executable}}</string>
        <string>run</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}/blocker.log</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}/blocker.err</string>
    <key>WorkingDirectory</key>
    <string>{{.WorkingDir}}</string>
</dict>
</plist>
`

type plistData struct {
	Name       string
	Executable string
	LogPath    string
	WorkingDir string
}

// getPlistPath returns the path to the LaunchAgent plist file
func (s *Service) getPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", s.Name+".plist")
}

// getLogPath returns the path for log files
func (s *Service) getLogPath() string {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".blocker", "logs")
	os.MkdirAll(logPath, 0755)
	return logPath
}

// installDarwin installs the LaunchAgent on macOS
func (s *Service) installDarwin() error {
	plistPath := s.getPlistPath()

	// Create LaunchAgents directory if it doesn't exist
	launchAgentsDir := filepath.Dir(plistPath)
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Get working directory (where executable is located)
	workingDir := filepath.Dir(s.Executable)

	// Create plist data
	data := plistData{
		Name:       s.Name,
		Executable: s.Executable,
		LogPath:    s.getLogPath(),
		WorkingDir: workingDir,
	}

	// Parse and execute template
	tmpl, err := template.New("plist").Parse(launchAgentTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse plist template: %w", err)
	}

	file, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("failed to create plist file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	// Load the LaunchAgent
	cmd := exec.Command("launchctl", "load", plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to load LaunchAgent: %w, output: %s", err, string(output))
	}

	return nil
}

// uninstallDarwin removes the LaunchAgent on macOS
func (s *Service) uninstallDarwin() error {
	plistPath := s.getPlistPath()

	// Check if plist exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return nil // Already uninstalled
	}

	// Unload the LaunchAgent
	cmd := exec.Command("launchctl", "unload", plistPath)
	cmd.Run() // Ignore errors, service might not be loaded

	// Remove the plist file
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	return nil
}

// startDarwin starts the LaunchAgent on macOS
func (s *Service) startDarwin() error {
	cmd := exec.Command("launchctl", "start", s.Name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start service: %w, output: %s", err, string(output))
	}
	return nil
}

// stopDarwin stops the LaunchAgent on macOS
func (s *Service) stopDarwin() error {
	cmd := exec.Command("launchctl", "stop", s.Name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop service: %w, output: %s", err, string(output))
	}
	return nil
}

// statusDarwin returns the status of the LaunchAgent on macOS
func (s *Service) statusDarwin() (string, error) {
	cmd := exec.Command("launchctl", "list", s.Name)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		if strings.Contains(string(output), "Could not find service") {
			return "not installed", nil
		}
		return "unknown", nil
	}

	// Parse output to determine if running
	outputStr := string(output)
	if strings.Contains(outputStr, "PID") || strings.Contains(outputStr, "\t0\t") == false {
		// Check if there's a PID (first column)
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, s.Name) {
				parts := strings.Fields(line)
				if len(parts) >= 1 && parts[0] != "-" {
					return fmt.Sprintf("running (PID: %s)", parts[0]), nil
				}
			}
		}
	}

	return "installed (not running)", nil
}

// isInstalledDarwin checks if the LaunchAgent is installed on macOS
func (s *Service) isInstalledDarwin() bool {
	plistPath := s.getPlistPath()
	_, err := os.Stat(plistPath)
	return err == nil
}
