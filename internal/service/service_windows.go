//go:build windows

package service

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Darwin stubs for windows build
func (s *Service) installDarwin() error {
	return fmt.Errorf("macOS not supported on this platform")
}

func (s *Service) uninstallDarwin() error {
	return fmt.Errorf("macOS not supported on this platform")
}

func (s *Service) startDarwin() error {
	return fmt.Errorf("macOS not supported on this platform")
}

func (s *Service) stopDarwin() error {
	return fmt.Errorf("macOS not supported on this platform")
}

func (s *Service) statusDarwin() (string, error) {
	return "", fmt.Errorf("macOS not supported on this platform")
}

func (s *Service) isInstalledDarwin() bool {
	return false
}

const serviceName = "BlockerService"

// installWindows installs the Windows service
func (s *Service) installWindows() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Check if service already exists
	existingSvc, err := m.OpenService(serviceName)
	if err == nil {
		existingSvc.Close()
		return fmt.Errorf("service %s already exists", serviceName)
	}

	// Create the service
	svcConfig := mgr.Config{
		DisplayName:  s.DisplayName,
		Description:  s.Description,
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
	}

	newSvc, err := m.CreateService(serviceName, s.Executable, svcConfig, "run")
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer newSvc.Close()

	// Set recovery options (restart on failure)
	recoveryActions := []mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 10 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
	}

	err = newSvc.SetRecoveryActions(recoveryActions, 86400) // Reset after 24 hours
	if err != nil {
		// Non-fatal, just log it
		fmt.Printf("Warning: failed to set recovery actions: %v\n", err)
	}

	// Start the service
	err = newSvc.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// uninstallWindows removes the Windows service
func (s *Service) uninstallWindows() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	winSvc, err := m.OpenService(serviceName)
	if err != nil {
		return nil // Service doesn't exist, already uninstalled
	}
	defer winSvc.Close()

	// Stop the service first
	_, err = winSvc.Control(svc.Stop)
	if err != nil {
		// Ignore stop errors, service might not be running
	}

	// Wait a bit for the service to stop
	time.Sleep(2 * time.Second)

	// Delete the service
	err = winSvc.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}

// startWindows starts the Windows service
func (s *Service) startWindows() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	winSvc, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer winSvc.Close()

	err = winSvc.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// stopWindows stops the Windows service
func (s *Service) stopWindows() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	winSvc, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer winSvc.Close()

	_, err = winSvc.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return nil
}

// statusWindows returns the status of the Windows service
func (s *Service) statusWindows() (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	winSvc, err := m.OpenService(serviceName)
	if err != nil {
		return "not installed", nil
	}
	defer winSvc.Close()

	status, err := winSvc.Query()
	if err != nil {
		return "unknown", fmt.Errorf("failed to query service: %w", err)
	}

	switch status.State {
	case svc.Running:
		return fmt.Sprintf("running (PID: %d)", status.ProcessId), nil
	case svc.StartPending:
		return "starting", nil
	case svc.StopPending:
		return "stopping", nil
	case svc.Stopped:
		return "stopped", nil
	case svc.Paused:
		return "paused", nil
	default:
		return "unknown", nil
	}
}

// isInstalledWindows checks if the Windows service is installed
func (s *Service) isInstalledWindows() bool {
	m, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer m.Disconnect()

	winSvc, err := m.OpenService(serviceName)
	if err != nil {
		return false
	}
	winSvc.Close()
	return true
}

// RunAsService runs the application as a Windows service
// This should be called from main when running as a service
func RunAsService(runFunc func() error, stopFunc func() error) error {
	return svc.Run(serviceName, &serviceHandler{
		runFunc:  runFunc,
		stopFunc: stopFunc,
	})
}

// IsRunningAsService checks if the process is running as a Windows service
func IsRunningAsService() bool {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return false
	}
	return isService
}

// serviceHandler implements svc.Handler
type serviceHandler struct {
	runFunc  func() error
	stopFunc func() error
}

func (h *serviceHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	// Start the service in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- h.runFunc()
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for {
		select {
		case err := <-errChan:
			if err != nil {
				return true, 1
			}
			return false, 0
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				h.stopFunc()
				return false, 0
			case svc.Interrogate:
				changes <- c.CurrentStatus
			}
		}
	}
}
