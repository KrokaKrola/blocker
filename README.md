# Network Blocker

A cross-platform (macOS/Windows) network request interceptor that blocks access to blacklisted websites by running as a local HTTP proxy.

## Features

- **Block websites** - Blocks access to blacklisted domains
- **Flexible wildcards** - Support for prefix (`*.example.com`), suffix (`google.*`), and double (`*.google.*`) wildcards
- **Auto-subdomain blocking** - `facebook.com` automatically blocks `www.facebook.com`, `m.facebook.com`, etc.
- **Auto-restart** - Runs as a system service that restarts automatically if killed or on system boot
- **File logging** - All blocked requests are logged to `~/.blocker/logs/blocker.log`
- **Cross-platform** - Works on macOS and Windows
- **Easy management** - Simple CLI commands to manage blacklist

## Installation

### Prerequisites

- Go 1.21 or later

### Build from source

```bash
# Clone or download the project
cd blocker

# Copy example config
cp configs/config.example.yaml configs/config.yaml

# Build
go build -o blocker ./cmd/blocker
```

### Cross-compile

```bash
# For macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o blocker-darwin ./cmd/blocker

# For macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o blocker-darwin-arm64 ./cmd/blocker

# For Windows
GOOS=windows GOARCH=amd64 go build -o blocker.exe ./cmd/blocker
```

## Quick Start

```bash
# 1. Install as service and enable system proxy
./blocker install --proxy

# 2. Check status
./blocker status

# That's it! Blocked sites will now show connection errors.
```

## Usage

### Service Management

```bash
# Install service + configure system proxy
./blocker install --proxy

# Check status
./blocker status

# Restart service (apply config changes)
./blocker restart

# Uninstall service + disable proxy
./blocker uninstall

# Run in foreground (for testing/debugging)
./blocker run
```

### Blacklist Management

```bash
# List blocked domains
./blocker list

# Add a domain
./blocker add youtube.com

# Add with wildcard (block all TLDs)
./blocker add "google.*"

# Remove a domain
./blocker remove youtube.com

# Apply changes
./blocker restart
```

### Viewing Logs

```bash
# View last 50 log lines
./blocker logs

# View last 100 lines
./blocker logs -n 100

# Follow logs in real-time
./blocker logs -f
```

## Configuration

Configuration file: `configs/config.yaml`

Copy from example: `cp configs/config.example.yaml configs/config.yaml`

```yaml
proxy:
  port: 8888
  bind: 127.0.0.1

blacklist:
  - facebook.com
  - twitter.com
  - instagram.com
  - "*.tiktok.com"
  - "google.*"

logging:
  level: info
  log_blocked: true
  log_allowed: false
```

### Blacklist Patterns

| Pattern | Description | Matches | Does NOT Match |
|---------|-------------|---------|----------------|
| `facebook.com` | Domain + all subdomains | `facebook.com`, `www.facebook.com`, `m.facebook.com` | `facebook.de` |
| `*.tiktok.com` | Subdomains only | `www.tiktok.com`, `vm.tiktok.com` | `tiktok.com` |
| `google.*` | All TLDs + subdomains | `google.com`, `google.de`, `www.google.es` | - |
| `*.google.*` | Subdomains + all TLDs | `www.google.com`, `mail.google.de` | `google.com` |

## How It Works

1. **Proxy Server** - Runs a local HTTP/HTTPS proxy on the configured port
2. **System Proxy** - System is configured to route traffic through the proxy
3. **Request Interception** - All browser/app traffic goes through the proxy
4. **Blacklist Check** - Each request is checked against the blacklist patterns
5. **Block or Forward** - Blocked requests get refused, allowed requests pass through

### HTTPS Handling

- HTTPS sites are blocked at the connection level (CONNECT method refused)
- When a blacklisted HTTPS site is accessed, the connection is refused
- The browser will show a "connection failed" or similar error

## Platform Details

### macOS

- **Service**: LaunchAgent (`~/Library/LaunchAgents/com.blocker.plist`)
- **Logs**: `~/.blocker/logs/blocker.log`
- **Proxy config**: Uses `networksetup` command

### Windows

- **Service**: Windows Service (`BlockerService`)
- **Logs**: `%USERPROFILE%\.blocker\logs\blocker.log`
- **Proxy config**: Uses registry settings
- **Auto-restart**: Configured via service recovery options

## CLI Reference

```
blocker - Network Blocker CLI

Commands:
  run         Run the proxy server in foreground
  install     Install blocker as a system service
              Flags: -p, --proxy  Also configure system proxy
  uninstall   Uninstall service and disable proxy
  restart     Restart service to apply config changes
  status      Show service and proxy status
  add         Add a domain to the blacklist
  remove      Remove a domain from the blacklist
  list        List all blacklisted domains
  logs        View blocker logs
              Flags: -f, --follow  Follow in real-time
                     -n, --lines   Number of lines (default: 50)

Global Flags:
  -c, --config string   Config file path
  -h, --help            Help for any command
```

## Troubleshooting

### Proxy not working

1. Check if the service is running: `./blocker status`
2. Check system proxy: System Preferences → Network → Proxies (macOS)
3. Try running in foreground: `./blocker run`
4. Check logs: `./blocker logs`

### Can't access any websites

The proxy is enabled but the blocker isn't running. Fix with:

```bash
./blocker uninstall   # Disables proxy and removes service
```

Or manually disable proxy in System Preferences → Network → Proxies.

### Changes to blacklist not taking effect

```bash
./blocker restart
```

### Port already in use

Edit `configs/config.yaml` and change the port, then restart:

```bash
./blocker restart
```

## License

MIT
