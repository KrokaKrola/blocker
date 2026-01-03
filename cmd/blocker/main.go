package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/user/blocker/internal/blocker"
	"github.com/user/blocker/internal/config"
	"github.com/user/blocker/internal/logger"
	"github.com/user/blocker/internal/proxy"
	"github.com/user/blocker/internal/service"
)

var (
	configPath string
	cfgManager *config.Manager
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "blocker",
		Short: "Network Blocker - Block access to blacklisted websites",
		Long: `Network Blocker is a local proxy that intercepts HTTP/HTTPS requests
and blocks access to websites in your blacklist.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "config file path")

	// Add commands
	rootCmd.AddCommand(runCmd())
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(uninstallCmd())
	rootCmd.AddCommand(restartCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(addCmd())
	rootCmd.AddCommand(removeCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(logsCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runCmd creates the run command
func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the proxy server in foreground",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxy()
		},
	}
}

// runProxy starts the proxy server
func runProxy() error {
	// Load configuration
	if configPath == "" {
		configPath = config.GetConfigPath()
	}

	if err := config.EnsureConfigExists(configPath); err != nil {
		return fmt.Errorf("failed to ensure config exists: %w", err)
	}

	cfgManager = config.NewManager(configPath)
	if err := cfgManager.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := cfgManager.Get()

	// Initialize file logger
	logCfg := logger.DefaultConfig()
	fileLogger, err := logger.New(logCfg)
	if err != nil {
		log.Printf("Warning: failed to initialize file logger: %v", err)
	} else {
		defer fileLogger.Close()
		log.Printf("Logging to: %s", logger.GetLogPath())
	}

	// Create blocker
	b := blocker.New()
	b.SetLogging(cfg.Logging.LogBlocked, cfg.Logging.LogAllowed)
	b.UpdateBlacklist(cfg.Blacklist)

	// Create and start proxy server
	srv := proxy.New(cfg.Proxy.Bind, cfg.Proxy.Port, b)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		srv.Stop()
	}()

	log.Printf("Network Blocker started on %s", srv.Addr())
	log.Printf("Configure your system/browser to use this proxy")
	log.Printf("Press Ctrl+C to stop")

	return srv.Start()
}

// installCmd creates the install command
func installCmd() *cobra.Command {
	var enableProxy bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install blocker as a system service",
		Long: `Install blocker as a system service that:
- Starts automatically on system boot
- Restarts automatically if it crashes
- Optionally configures system proxy settings`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config to get port
			if configPath == "" {
				configPath = config.GetConfigPath()
			}

			if err := config.EnsureConfigExists(configPath); err != nil {
				return fmt.Errorf("failed to ensure config exists: %w", err)
			}

			cfgManager = config.NewManager(configPath)
			if err := cfgManager.Load(); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cfg := cfgManager.Get()

			// Create and install service
			svc, err := service.New(cfg.Proxy.Port)
			if err != nil {
				return fmt.Errorf("failed to create service: %w", err)
			}

			if svc.IsInstalled() {
				return fmt.Errorf("service is already installed")
			}

			fmt.Println("Installing blocker service...")
			if err := svc.Install(); err != nil {
				return fmt.Errorf("failed to install service: %w", err)
			}
			fmt.Println("Service installed successfully!")

			// Configure system proxy if requested
			if enableProxy {
				fmt.Println("Configuring system proxy...")
				proxyConfig := service.NewProxyConfig(cfg.Proxy.Bind, cfg.Proxy.Port)
				if err := proxyConfig.Enable(); err != nil {
					fmt.Printf("Warning: failed to configure system proxy: %v\n", err)
					fmt.Println("You may need to configure your system proxy manually.")
				} else {
					fmt.Println("System proxy configured!")
				}
			} else {
				fmt.Printf("\nTo use the blocker, configure your system proxy to: %s:%d\n", cfg.Proxy.Bind, cfg.Proxy.Port)
				fmt.Println("Or run 'blocker install --proxy' to configure it automatically.")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&enableProxy, "proxy", "p", false, "also configure system proxy settings")

	return cmd
}

// uninstallCmd creates the uninstall command
func uninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall blocker system service",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config to get port
			if configPath == "" {
				configPath = config.GetConfigPath()
			}

			cfgManager = config.NewManager(configPath)
			cfgManager.Load() // Ignore errors, we just need the port

			cfg := cfgManager.Get()
			port := 8080
			bind := "127.0.0.1"
			if cfg != nil {
				port = cfg.Proxy.Port
				bind = cfg.Proxy.Bind
			}

			// Disable system proxy first
			fmt.Println("Disabling system proxy...")
			proxyConfig := service.NewProxyConfig(bind, port)
			if err := proxyConfig.Disable(); err != nil {
				fmt.Printf("Warning: failed to disable system proxy: %v\n", err)
			}

			// Create and uninstall service
			svc, err := service.New(port)
			if err != nil {
				return fmt.Errorf("failed to create service: %w", err)
			}

			fmt.Println("Uninstalling blocker service...")
			if err := svc.Uninstall(); err != nil {
				return fmt.Errorf("failed to uninstall service: %w", err)
			}

			fmt.Println("Service uninstalled successfully!")
			return nil
		},
	}
}

// restartCmd creates the restart command to re-apply config changes
func restartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart the blocker service to apply config changes",
		Long: `Restart the blocker service to apply any configuration changes.
Use this after modifying the blacklist or config file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config to get port
			if configPath == "" {
				configPath = config.GetConfigPath()
			}

			cfgManager = config.NewManager(configPath)
			cfgManager.Load()

			cfg := cfgManager.Get()
			port := 8888
			bind := "127.0.0.1"
			if cfg != nil {
				port = cfg.Proxy.Port
				bind = cfg.Proxy.Bind
			}

			// Check if service is installed
			svc, err := service.New(port)
			if err != nil {
				return fmt.Errorf("failed to create service: %w", err)
			}

			if !svc.IsInstalled() {
				return fmt.Errorf("service is not installed. Run 'blocker install --proxy' first")
			}

			fmt.Println("Restarting blocker service...")

			// Stop the service
			if err := svc.Stop(); err != nil {
				// Ignore stop errors, service might not be running
				fmt.Printf("Note: %v\n", err)
			}

			// Unload and reload the LaunchAgent to pick up any binary changes
			if err := svc.Uninstall(); err != nil {
				return fmt.Errorf("failed to uninstall service: %w", err)
			}

			if err := svc.Install(); err != nil {
				return fmt.Errorf("failed to reinstall service: %w", err)
			}

			// Re-enable proxy
			proxyConfig := service.NewProxyConfig(bind, port)
			if err := proxyConfig.Enable(); err != nil {
				fmt.Printf("Warning: failed to configure system proxy: %v\n", err)
			}

			fmt.Println("Service restarted successfully!")
			fmt.Println("Config changes are now active.")
			return nil
		},
	}
}

// statusCmd creates the status command
func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show blocker service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			if configPath == "" {
				configPath = config.GetConfigPath()
			}

			cfgManager = config.NewManager(configPath)
			cfgManager.Load()

			cfg := cfgManager.Get()
			port := 8080
			bind := "127.0.0.1"
			if cfg != nil {
				port = cfg.Proxy.Port
				bind = cfg.Proxy.Bind
			}

			// Check service status
			svc, err := service.New(port)
			if err != nil {
				return fmt.Errorf("failed to create service: %w", err)
			}

			status, err := svc.Status()
			if err != nil {
				status = "unknown"
			}

			fmt.Printf("Service Status: %s\n", status)

			// Check proxy status
			proxyConfig := service.NewProxyConfig(bind, port)
			proxyEnabled, _ := proxyConfig.IsEnabled()
			if proxyEnabled {
				fmt.Printf("System Proxy: enabled (%s:%d)\n", bind, port)
			} else {
				fmt.Println("System Proxy: disabled")
			}

			// Show config path
			fmt.Printf("Config File: %s\n", configPath)

			// Show log file path
			fmt.Printf("Log File: %s\n", logger.GetLogPath())

			// Show blacklist count
			if cfg != nil {
				fmt.Printf("Blacklisted Domains: %d\n", len(cfg.Blacklist))
			}

			return nil
		},
	}
}

// addCmd creates the add command
func addCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [domain]",
		Short: "Add a domain to the blacklist",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			if configPath == "" {
				configPath = config.GetConfigPath()
			}

			cfgManager = config.NewManager(configPath)
			if err := cfgManager.Load(); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if err := cfgManager.AddToBlacklist(domain); err != nil {
				return err
			}

			fmt.Printf("Added '%s' to blacklist\n", domain)
			fmt.Println("Run 'blocker restart' to apply changes")
			return nil
		},
	}
}

// removeCmd creates the remove command
func removeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [domain]",
		Short: "Remove a domain from the blacklist",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			if configPath == "" {
				configPath = config.GetConfigPath()
			}

			cfgManager = config.NewManager(configPath)
			if err := cfgManager.Load(); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if err := cfgManager.RemoveFromBlacklist(domain); err != nil {
				return err
			}

			fmt.Printf("Removed '%s' from blacklist\n", domain)
			fmt.Println("Run 'blocker restart' to apply changes")
			return nil
		},
	}
}

// listCmd creates the list command
func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all blacklisted domains",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				configPath = config.GetConfigPath()
			}

			cfgManager = config.NewManager(configPath)
			if err := cfgManager.Load(); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			blacklist := cfgManager.GetBlacklist()

			if len(blacklist) == 0 {
				fmt.Println("Blacklist is empty")
				return nil
			}

			fmt.Printf("Blacklisted domains (%d):\n", len(blacklist))
			for i, domain := range blacklist {
				fmt.Printf("  %d. %s\n", i+1, domain)
			}

			return nil
		},
	}
}

// logsCmd creates the logs command
func logsCmd() *cobra.Command {
	var follow bool
	var lines int

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View blocker logs",
		Long:  "View the blocker log file. Use -f to follow (tail) the log in real-time.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logPath := logger.GetLogPath()

			// Check if log file exists
			if _, err := os.Stat(logPath); os.IsNotExist(err) {
				fmt.Printf("Log file not found: %s\n", logPath)
				fmt.Println("The log file will be created when the blocker runs.")
				return nil
			}

			fmt.Printf("Log file: %s\n\n", logPath)

			if follow {
				// Use tail -f for real-time following
				fmt.Println("Following logs (Ctrl+C to stop)...")
				fmt.Println("---")
				tailCmd := fmt.Sprintf("tail -f %s", logPath)
				return syscall.Exec("/bin/sh", []string{"sh", "-c", tailCmd}, os.Environ())
			}

			// Read last N lines
			tailCmd := fmt.Sprintf("tail -n %d %s", lines, logPath)
			output, err := execCommand("sh", "-c", tailCmd)
			if err != nil {
				return fmt.Errorf("failed to read logs: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output in real-time")
	cmd.Flags().IntVarP(&lines, "lines", "n", 50, "number of lines to show")

	return cmd
}

// execCommand executes a shell command and returns its output
func execCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	return string(output), err
}
