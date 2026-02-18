# Marmithon Rust Rewrite Plan

## Overview

Rewrite the Marmithon IRC bot from Go (~2,200 lines across 11 files) to Rust, preserving all existing functionality, French-language responses, and deployment pipeline.

---

## Dependency Mapping

| Go Dependency | Rust Crate | Notes |
|---|---|---|
| `whyrusleeping/hellabot` | `irc` (tokio-based) | Mature async IRC client with SSL, channel mgmt |
| `BurntSushi/toml` | `toml` + `serde` | Identical TOML config parsing |
| `PuerkitoBio/goquery` | `scraper` | CSS selector-based HTML parsing |
| `modernc.org/sqlite` | `rusqlite` + `r2d2-sqlite` | Native SQLite bindings, connection pooling |
| `bcicen/go-units` | Manual implementation | Only ~20 conversions used; simpler to own it |
| `log15` | `tracing` + `tracing-subscriber` | Structured async-friendly logging |
| `net/http` (stdlib) | `reqwest` | Async HTTP client with cookie jar, timeouts |
| `regexp` (stdlib) | `regex` | Near-identical API |
| `golang.org/x/net/proxy` | `tokio-socks` | SOCKS5 proxy support for IRC connection |

## Project Structure

```
marmithon-rs/
├── Cargo.toml
├── src/
│   ├── main.rs           # Entry point, signal handling, reconnection loop
│   ├── config.rs         # TOML config loading + validation
│   ├── command/
│   │   ├── mod.rs        # Command registry, dispatch, URL detection
│   │   ├── seen.rs       # !seen - SQLite user tracking
│   │   ├── title.rs      # URL title extraction + platform extractors + cache
│   │   ├── various.rs    # !cve, !version
│   │   ├── atc.rs        # !icao, !distance, !time
│   │   └── units.rs      # !convert with custom nautical miles
│   ├── identd.rs         # RFC 1413 identd server
│   └── metrics.rs        # Prometheus text format + /health endpoint
├── Dockerfile
└── Makefile
```

## Implementation Phases

### Phase 1: Skeleton + Config + IRC Connection

**Files:** `Cargo.toml`, `src/main.rs`, `src/config.rs`

1. Initialize Cargo project with dependencies:
   ```toml
   [dependencies]
   irc = "1"
   tokio = { version = "1", features = ["full"] }
   serde = { version = "1", features = ["derive"] }
   toml = "0.8"
   tracing = "0.1"
   tracing-subscriber = "0.3"
   reqwest = { version = "0.12", features = ["cookies"] }
   rusqlite = { version = "0.32", features = ["bundled"] }
   r2d2_sqlite = "0.25"
   r2d2 = "0.8"
   scraper = "0.22"
   regex = "1"
   chrono = "0.4"
   rand = "0.9"
   anyhow = "1"
   tokio-socks = "0.5"
   ```

2. Config struct with `serde::Deserialize`, matching current TOML field names:
   - `server`, `nick`, `server_password`, `channels`, `ssl`
   - `airport_api_url`, `identd_enabled/port/username`, `metrics_enabled/port`
   - `proxy_address`
   - `reconnect_delay_seconds`, `reconnect_max_attempts`
   - Apply same defaults as Go version
   - Same validation (server format, nick, channels non-empty)

3. Basic IRC connection using `irc` crate:
   - SSL support, server password, channel auto-join
   - SOCKS5 proxy via `tokio-socks` when `proxy_address` is set
   - Signal handling with `tokio::signal` (SIGINT, SIGTERM)
   - Graceful shutdown: farewell messages ("Ah ! Je meurs !"), QUIT, 5s timeout

4. Reconnection loop:
   - Same logic: attempt counter, configurable delay, optional max attempts
   - French log messages matching current output

5. Keepalive PING:
   - Wait for RPL_ENDOFNAMES (366) before starting
   - `tokio::time::interval(30s)` sending `PING :keepalive`
   - Cancel via `tokio::sync::watch` or `CancellationToken` on disconnect/shutdown

### Phase 2: Command Framework

**Files:** `src/command/mod.rs`

1. Define traits/structs:
   ```rust
   type CommandFn = Box<dyn Fn(&IrcClient, &Message, &[&str]) + Send + Sync>;

   struct Command {
       name: String,
       description: String,
       usage: String,
       run: CommandFn,
   }

   struct CommandList {
       prefix: String,
       commands: HashMap<String, Command>,
   }
   ```

2. `process()` method:
   - Track user activity (spawn task for `update_user_seen`)
   - Route to `handle_command()` if message starts with prefix
   - Otherwise route to `handle_url_detection()`

3. Help system: `!help` lists commands, `!help <cmd>` shows details
   - Same French messages: "Voici ce que je peux faire:", etc.

4. URL detection: same regex pattern `https?://...`
   - Extract first URL, spawn task for title retrieval

### Phase 3: SQLite + !seen Command

**Files:** `src/command/seen.rs`

1. Database setup:
   - `rusqlite` with `r2d2` connection pool (10 max, 5 idle)
   - WAL mode, busy_timeout=5000
   - Same schema: `user_seen (nickname TEXT PRIMARY KEY, channel TEXT, last_seen_at DATETIME, last_message TEXT)`

2. `update_user_seen()`: INSERT OR REPLACE with `datetime('now')`
3. `get_user_seen()`: case-insensitive lookup with COLLATE NOCASE
   - Parse timestamps with `chrono` using same 4 format fallbacks
4. `search_users_seen()`: wildcard `*`→`%`, `?`→`_` conversion, LIMIT 10
5. `format_time_difference()`: French time formatting (jours/heures/minutes/secondes, "à l'instant")
6. Random responses: same 8 present + 7 not-seen French phrases
7. `!seen` handler: self-check, wildcard detection, 5-minute presence heuristic
   - Data directory fallback: `/data` → `/tmp`

### Phase 4: URL Title Extraction

**Files:** `src/command/title.rs`

1. Title cache:
   - `RwLock<HashMap<String, CacheEntry>>` with 1-hour expiration
   - Cleanup every 100 entries

2. Platform-specific extractors (same regex patterns):
   - YouTube: 5 regex patterns for title, append " - YouTube"
   - Vimeo: og:title or `<title>` tag, append " - Vimeo"
   - Dailymotion: og:title or `<title>` tag, append " - Dailymotion"
   - Twitch: og:title or `<title>` tag, append " - Twitch"
   - Yahoo: `"Yahoo, c'est de la merrddeuuhhh"`

3. Generic fallback: `scraper` crate for `<title>`, `og:title`, `meta[name="title"]`

4. HTTP client:
   - `reqwest` with cookie jar, 15s timeout, 1MB body limit
   - Same User-Agent: `Marmithon-TitleBot/1.0 (IRC Title Extractor)`
   - Same headers (Accept, Accept-Language, DNT, Connection)
   - Check Content-Type is `text/html`

5. `clean_title()`: HTML entity decoding (6 entities), whitespace normalization, 300-char limit

6. IRC formatting: bold `\x02` for title, `\x0F\x0314[cache]` for cached

### Phase 5: Other Commands

**Files:** `src/command/various.rs`, `src/command/atc.rs`, `src/command/units.rs`

#### !cve (`various.rs`)
- Regex validation: `^CVE-\d{4}-\d{4,}$`
- HTTP GET to `http://cve.circl.lu/api/cve/{CVE}` with 15s timeout
- JSON deserialization with `serde` (same nested struct)
- Handle 404 vs other errors, display summary + first reference

#### !version (`various.rs`)
- Build-time constants via `env!()` or build script for git commit + build time

#### !icao, !distance, !time (`atc.rs`)
- Airport search: query `{api}/api/airport/search?name=...&country=...`
- Distance: query `{api}/api/airport/distance?departure=...&destination=...`
- Time: query `{api}/api/airport/time?icao=...`
- Same ICAO 4-char validation, JSON parsing, French error messages
- Filter airports with valid 4-char ICAO codes

#### !convert (`units.rs`)
- Custom nautical mile conversions (same map: km↔nmi, m↔nmi)
- Intermediate conversions through meters
- No external units library — implement the common conversions directly:
  - Distance: m, km, ft, mi, in, cm, mm, nmi, yd
  - Weight: kg, g, lb, oz, ton
  - Temperature: C, F, K (non-linear)
  - Volume: l, ml, gal, qt, pt
  - Data: B, KB, MB, GB, TB
  - Energy: J, kJ, cal, kcal, Wh, kWh
- `!convert search <term>` for unit discovery
- `determine_precision()`: same logic (≥1000→0dp, ≥10→1dp, ≥1→2dp, else→4dp)
- Suggestion system for unknown units

### Phase 6: Identd Server

**File:** `src/identd.rs`

- `TcpListener` on configurable port (default 113)
- Per-connection `tokio::spawn` handler
- 10-second read timeout
- Parse `server_port, client_port` format
- Respond with RFC 1413: `<port>, <port> : USERID : UNIX : <username>\r\n`
- Graceful shutdown via `CancellationToken`

### Phase 7: Prometheus Metrics

**File:** `src/metrics.rs`

- Atomic counters: `AtomicU64` for messages_received/sent, commands_executed, reconnects
- `AtomicBool` for connected status
- `RwLock<HashSet<String>>` for channel tracking
- HTTP server via `axum` or plain `hyper`:
  - `GET /metrics`: Prometheus text exposition format (same metric names: `marmithon_*`)
  - `GET /health`: "OK" if connected, 503 if not
- Start time for uptime calculation

### Phase 8: Build + Docker + CI

1. **Makefile:**
   ```makefile
   build-local:
       cargo build --release

   build:
       docker buildx build --platform linux/arm64 --tag marmithon .

   deploy: build
       docker tag marmithon forge.internal/nemo/marmithon:test
       docker push forge.internal/nemo/marmithon:test
   ```

2. **Dockerfile** (multi-stage):
   ```dockerfile
   FROM rust:1.84-alpine AS build
   RUN apk add --no-cache musl-dev
   WORKDIR /marmithon
   COPY . .
   RUN cargo build --release

   FROM alpine:3.23
   RUN apk add --no-cache ca-certificates curl bind-tools netcat-openbsd
   RUN addgroup -g 65532 nonroot && adduser -D -u 65532 -G nonroot nonroot
   COPY --from=build /marmithon/target/release/marmithon /app/marmithon
   COPY --from=build /marmithon/marmithon.toml /app/
   EXPOSE 113 9090
   USER nonroot:nonroot
   ENTRYPOINT ["/app/marmithon"]
   CMD ["-config", "/app/marmithon.toml"]
   ```

3. **Build script** (`build.rs`): Inject git commit and build time as compile-time constants

4. **CI** (`.forgejo/workflows/build.yaml`): Adapt for Rust build

5. **rusqlite bundled feature**: Embeds SQLite, no system dependency — eliminates the CGO-equivalent problem entirely

---

## Migration Strategy

1. Build the Rust version alongside the Go version (in a `marmithon-rs/` subdirectory or a new branch)
2. Test against the same IRC server with a different nick (e.g., `marmithon-dev`)
3. Verify feature parity:
   - [ ] Connect, join channels, respond to commands
   - [ ] !help, !version
   - [ ] !cve lookup
   - [ ] !seen tracking + wildcard search
   - [ ] !convert with standard + nautical mile units
   - [ ] !icao, !distance, !time
   - [ ] URL title extraction (generic + YouTube/Vimeo/Dailymotion/Twitch/Yahoo)
   - [ ] Title caching with [cache] indicator
   - [ ] Identd server responses
   - [ ] Prometheus metrics endpoint
   - [ ] Health check endpoint
   - [ ] Reconnection after disconnect
   - [ ] Keepalive PING every 30s
   - [ ] SOCKS5 proxy support
   - [ ] Graceful shutdown with farewell messages
4. Swap the Docker image once validated
5. Keep the existing SQLite database — schema is identical, Rust reads it directly

## Estimated Size

~3,000-3,500 lines of Rust (vs ~2,200 Go) due to:
- More explicit error handling (`Result`/`?` chains)
- Struct definitions with serde derive attributes
- Async boilerplate (`async fn`, `.await`)
- Conversion table data in `units.rs`

The binary will be ~4-6MB (static musl) vs ~16MB Go.
