# Marmithon

A simple IRC bot that I maintain for friends.

Features:
+ Config file based setup with TOML
+ Command system with various utilities (CVE lookup, unit conversion, airport info)
+ Based on the great IRC package <https://github.com/whyrusleeping/hellabot>
+ Automatic URL title extraction for links posted in channels
+ User activity tracking with `!seen` command
+ Native RFC 1413 identd server
+ Prometheus metrics endpoint for monitoring
+ Automatic reconnection on connection loss

## Configuration

Configuration is done via TOML files (`marmithon.toml` for production, `dev.toml` for development).

Example configuration:

```toml
server = "irc.ircnet.com:6667"
nick = "Marmithon"
ssl = false
channels = ["#channel1", "#channel2"]

# Identd server configuration (RFC 1413)
identdEnabled = true
identdPort = "113"
identdUsername = "marmithon"

# Prometheus metrics server
metricsEnabled = true
metricsPort = "9090"

# Automatic reconnection
reconnectEnabled = true
reconnectDelaySeconds = 30
reconnectMaxAttempts = 0  # 0 = unlimited
```

## Monitoring

When metrics are enabled, Marmithon exposes Prometheus-compatible metrics at `http://localhost:9090/metrics`:

- `marmithon_uptime_seconds` - Bot uptime
- `marmithon_connected` - Connection status
- `marmithon_messages_received_total` - Messages received
- `marmithon_messages_sent_total` - Messages sent
- `marmithon_commands_executed_total` - Commands executed
- `marmithon_reconnects_total` - Reconnection attempts
- `marmithon_channels` - Number of joined channels

Health check available at `http://localhost:9090/health`

## Building

```bash
make build-local  # Local build
make build        # Docker build
go build          # Direct Go build
```

## Running

```bash
./marmithon -config marmithon.toml
```
