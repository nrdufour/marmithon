# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Marmithon is a simple IRC bot written in Go that connects to IRC networks and provides various utility commands. It's built using the hellabot IRC library and includes features like URL title extraction, CVE lookups, unit conversions, airport information, and user activity tracking.

## Architecture

The codebase follows a modular structure:

- **main.go**: Entry point that initializes the bot, loads configuration, sets up commands, and starts the IRC connection
- **config/**: Configuration management using TOML files
- **command/**: Command system with individual handlers for different bot features
  - **command.go**: Core command framework with trigger processing and URL detection
  - **various.go**: Utility commands (version, CVE lookup, unit conversion)
  - **atc.go**: Aviation-related commands (airport search, distance calculation)
  - **seen.go**: User activity tracking with SQLite database
  - **title.go**: URL title extraction functionality
  - **units.go**: Unit conversion system

## Key Components

### Command System
Commands are registered in `setupCommands()` in main.go and processed through the `CommandTrigger` that listens for PRIVMSG events. Each command implements the `command.Func` signature and receives the IRC message and parsed arguments.

### Configuration
Uses TOML configuration files (default: `production.toml`, dev: `dev.toml`) with settings for server, nickname, channels, SSL, and API URLs. The bot validates configuration on startup.

### Database
Uses SQLite for the "seen" functionality to track when users were last active in channels. Database is initialized at startup and stored in `/data/seen.db`.

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
The project uses multi-stage Docker builds with distroless base images for security. See `Dockerfile` for build process.

## Configuration Files

- **marmithon.toml**: Production IRC configuration (server, nick, channels)
- **dev.toml**: Development configuration
- **fly.toml**: Fly.io deployment configuration
- **go.mod**: Go module dependencies

## CI/CD

Uses Forgejo Actions (`.forgejo/workflows/build.yaml`) for automated builds on main branch pushes, building ARM64 Docker images and pushing to internal registry.