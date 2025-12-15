# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Marmithon is a simple IRC bot written in Go that connects to IRC networks and provides various utility commands. It's built using the hellabot IRC library and includes features like URL title extraction, CVE lookups, unit conversions, airport information, user activity tracking, native identd server, Prometheus metrics, and automatic reconnection.

## Architecture

The codebase follows a modular structure:

- **main.go**: Entry point that initializes the bot, loads configuration, sets up commands, starts services (identd, metrics), and manages IRC connection with reconnection logic
- **config/**: Configuration management using TOML files
- **command/**: Command system with individual handlers for different bot features
  - **command.go**: Core command framework with trigger processing, URL detection, and metrics tracking
  - **various.go**: Utility commands (version, CVE lookup, unit conversion)
  - **atc.go**: Aviation-related commands (airport search, distance calculation)
  - **seen.go**: User activity tracking with SQLite database
  - **title.go**: URL title extraction functionality
  - **units.go**: Unit conversion system
- **identd/**: RFC 1413 compliant identd server for IRC authentication
- **metrics/**: Prometheus metrics collection and HTTP server

## Key Components

### Command System
Commands are registered in `setupCommands()` in main.go and processed through the `CommandTrigger` that listens for PRIVMSG events. Each command implements the `command.Func` signature and receives the IRC message and parsed arguments.

### Configuration
Uses TOML configuration files (default: `marmithon.toml`, dev: `dev.toml`) with settings for:
- IRC connection (server, nickname, channels, SSL, password)
- Identd server (enabled, port, username)
- Metrics server (enabled, port)
- Reconnection behavior (enabled, delay, max attempts)
- API URLs for external services

The bot validates configuration on startup and provides sensible defaults for optional settings.

### Database
Uses SQLite for the "seen" functionality to track when users were last active in channels. Database is initialized at startup and stored in `/data/seen.db`.

### Identd Server
Native RFC 1413 identd server implementation that responds to identification requests from IRC servers. Runs on port 113 (configurable) and returns the configured username.

### Metrics & Monitoring
Prometheus-compatible metrics server exposing:
- Uptime, connection status, message counters
- Command execution counts, reconnection attempts
- Channel counts
- Health check endpoint at `/health`

### Automatic Reconnection
When enabled, the bot automatically reconnects to IRC if the connection is lost. Configurable delay between attempts and optional maximum attempt limit.

### External APIs
Integrates with external services:
- CVE information from cve.circl.lu
- Airport data from configurable API (default: ask.fly.dev)
- Automatic URL title extraction for links posted in channels

## Development Commands

### Building
- **Local build**: `make build-local` - Builds with git commit and build time info
- **Docker build**: `make build` - Cross-platform ARM64 Docker build
- **Direct Go build**: `go build`

### Testing
No specific test commands found in the project. Use standard Go testing:
- `go test ./...` - Run all tests
- `go test ./command` - Test specific package

### Deployment
- **Deploy to registry**: `make deploy` - Tags and pushes Docker image to forge.internal registry
- **Fly.io deployment**: Uses `fly.toml` configuration for deployment to Fly.io platform

### Docker
The project uses multi-stage Docker builds with Alpine base images. The Dockerfile exposes ports 113 (identd) and 9090 (metrics). See `Dockerfile` for build process.

## Configuration Files

- **marmithon.toml**: Production IRC configuration (server, nick, channels)
- **dev.toml**: Development configuration
- **fly.toml**: Fly.io deployment configuration
- **go.mod**: Go module dependencies

## CI/CD

Uses Forgejo Actions (`.forgejo/workflows/build.yaml`) for automated builds on main branch pushes, building ARM64 Docker images and pushing to internal registry.