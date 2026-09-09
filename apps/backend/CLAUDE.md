# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests — no Docker, no database, nothing to start first
make test
# or: go test ./... | grep -v 'no test files'

# Run a single test
go test ./internal/clicks/domain/click_handler_service/... -run TestName

# Run the concurrency-sensitive tests under the race detector
go test ./... -race

# Run API server locally
go run ./cmd/api -config cmd/api/example.yaml

# Generate protobuf code (requires buf CLI)
make proto

# Build Docker image
make dBuild
```

## Architecture

This is a Go backend for a collaborative map-clicking game. It follows **hexagonal architecture (ports & adapters)**.

### One application

**`cmd/api`** — HTTP/WebSocket server handling clicks, tile ownership queries, and real-time updates. Follows `New()` → `Configure()` → `Run()`.

It runs as a **single self-contained container with no dependencies**: the tile map lives in process and is persisted to a local snapshot file. There is no database, no cache, and no second process.

The X/Twitter reporting job used to be a separate `cmd/bookkeeper` process. It now runs inside `cmd/api` as an optional goroutine behind `bookkeeper.enabled` (default off) — the recent-updates window only exists inside the API process, so a separate process would have nothing to read.

### Domain Layer (`internal/clicks/domain/`)

Core interfaces (ports) defined in `gateways.go`:
- `TilesChecker` — validates tile IDs (0..maxIndex)
- `TileStorage` — reads/writes tile→country ownership
- `TileReporter` — notifies downstream of tile updates
- `CountryChecker` — validates ISO country codes

`ClickHandlerService` wires these interfaces together and contains all game logic. The Prometheus-instrumented version (`prom_click_handler_service/`) wraps it via decorator pattern.

`runner/` is the scheduled job the bookkeeper uses; it reads through the storage's `PastUpdates`.

### Adapters

**Primary (input):**
- `adapters/primary/http/planetv1controller/` — the API. `ClickService` implements the generated `planetv1connect.ClickServiceHandler` and nothing else; it never sees an `http.ResponseWriter`, which is the point of serving the contract with Connect rather than by hand. Two interceptors wrap it: `NewRateLimitInterceptor` (see [Rate limiting](#rate-limiting)) and `NewErrorInterceptor`, which maps domain errors onto Connect codes, logs the unexpected ones and keeps their cause off the wire, so **handlers return their errors bare** — `domain.ErrInvalidArgument` becomes `CodeInvalidArgument`, a code a handler picked itself is left alone, and anything else is logged once and answered as `internal error`.
- `adapters/primary/http/websocket_publisher/` — subscribes to the tile update stream, broadcasts to WebSocket clients

**There is one server, one mux, and no version prefix.** Connect names its own path, `/planet.v1.ClickService/`, so nothing is mounted under a prefix of ours. The RPCs, the websocket upgrade and `/metrics` are the only three things on the router. Nothing here needs a connection-level demultiplexer such as `cmux`; that is for running a real gRPC server, which owns its own HTTP/2 handler, beside a REST one.

`Configure` sets `http.Protocols` with both HTTP/1.1 and unencrypted HTTP/2, because the generated handler also speaks gRPC and gRPC-Web and those need HTTP/2. Browsers reach the same routes over HTTP/1.1. Verified: HTTP/1.1 and h2c both answer on the same port.

### The map load

`GetMap` is an ordinary RPC, but marked `idempotency_level = NO_SIDE_EFFECTS` in the proto, so Connect sends it as an **HTTP GET** and the handler sets `Cache-Control: public, max-age=5` on the response. A burst of visitors can therefore share one origin response; the websocket carries everything that happens after a chunk was built, so a client starting from a slightly old map converges anyway. `MapDensity` is marked the same way.

The response never repeats a tile id. `GetMapResponse` carries `start_tile_id`, the interned `codes` table, and `tiles` — a `bytes` field holding two bytes per tile, little endian, indexing into `codes`. Tile ids are implicit in the position, which is what makes it far smaller than the deprecated `map<uint32, string>`: **516 KB against 3.6 MB** for a full 257,948-tile map.

`memory_tile_storage.StateBatchDense` builds it. The interned ids are copied out exactly as stored and the table travels with them, so nothing is translated on the way out and the client needs no shared country list. Protobuf does all the framing — there is no hand-rolled magic or length-prefixing on either side, and therefore no encoder and decoder that have to be edited together.

**Secondary (output):**
- `adapters/secondary/memory_tile_storage/` — the tile map. A preallocated `[]uint16` indexed by tile id, with country codes interned into a side table (2 bytes per tile — ~2 MB for a 1M-tile map). Fans updates out in process, serves `PastUpdates` from a bounded ring buffer, and persists to a local snapshot file.
- `adapters/secondary/in_memory_tile_checker/` — validates tile IDs
- `adapters/secondary/in_memory_country_checker/` — validates country codes (hardcoded)
- `adapters/secondary/x_publisher/` — posts to X/Twitter

Beyond the `domain.TileStorage` port, `memory_tile_storage` also exposes `Subscribe(ctx) (<-chan domain.TileUpdate, error)` for the websocket publisher and `PastUpdates(ctx, duration, now)` for the bookkeeper.

### Key Flow

```
POST /planet.v1.ClickService/Click
  → ClickService
  → ClickHandlerService (validates tile ID + country)
  → MemoryTileStorage.Set() [writes the tile, fans the update out in process]
  → WebsocketPublisher (fans out to WS clients)
```

`Set` is a no-op when the tile already holds that value — no write, no update published.

### Rate limiting

`NewRateLimitInterceptor` throttles **`Click` only**, per source IP, from a `kernel/ratelimit` token bucket — 1 click/s with a burst of 10 by default (`rateLimiter.*`). `MapDensity` and `GetMap` are cacheable reads a proxy in front absorbs; limiting them would punish a page load rather than a bot. A refused click answers `CodeResourceExhausted`, i.e. HTTP 429, and never reaches the domain.

The bucket key is whatever `IPReaderMiddleware` put on the context: `X-Real-IP` if present, otherwise the peer address. **The reverse proxy must set that header itself** — `deploy/vps/Caddyfile` does, with `header_up X-Real-IP {client_ip}` on every backend route. Merely forwarding it would let a client send its own and buy a fresh bucket per request. The fallback is the peer address rather than a constant precisely so a missing header degrades to per-connection buckets instead of rate limiting the whole game as one player.

### Durability

The whole map is snapshotted to `tilesStorage.snapshotPath`:
- a compact binary encoding, not JSON: magic + version + CRC32, then the interned country code table, then two bytes per tile
- written atomically — temp file, fsync, `os.Rename`, fsync of the directory — so a crash mid-write leaves the previous snapshot intact
- flushed every `snapshotInterval` when the state changed, and once more on graceful shutdown (`cmd/api` handles SIGINT/SIGTERM for exactly this)
- restored at boot; a missing, truncated, or corrupt snapshot logs and starts from an empty map, it never prevents a start
- a snapshot taken at a different `gameMap.maxIndex` restores the overlap

**What this costs:** anything written since the last snapshot is lost on a hard kill (`SIGKILL`, OOM, power loss), bounded by `snapshotInterval`. And because the state is per-process, **this is single-instance only** — two API replicas would each hold their own divergent map. Both are deliberate: the game state is a few MB and the WebSocket fanout was already per-instance, so a database was buying durability alone.

The snapshot file is the only thing worth backing up.

### Kernel (`internal/kernel/`)

Shared infrastructure: `cfgutil` (YAML + env config via koanf), `httpserver` (middleware, formats), `logging`, `prom` (Prometheus), `xtime`, `ctxutil`, `basicutil`, `ratelimit`.

`ratelimit` is a keyed token bucket held in this process, like the tile map it protects — with one API instance, a shared counter would buy nothing. Its `Run` loop periodically forgets the buckets that have refilled to capacity, which is free: such a bucket holds exactly what a freshly created one would, and without it the map would keep an entry per address that ever clicked.

### Configuration

Config is loaded from a YAML file (`-config` flag), with environment variables overriding it — `cfgutil` uses `.` as the nesting delimiter, so `tilesStorage.snapshotPath=/data/tiles` in the environment overrides the file. See `cmd/api/example.yaml` for the full schema.

- `httpServer.bindAddress` — the encoding is negotiated per request, so there is no format setting.
- `gameMap.maxIndex` — total number of tiles
- `tilesStorage.snapshotPath` — where the state is persisted; **empty disables durability**
- `tilesStorage.snapshotInterval` — how often a changed state is flushed
- `tilesStorage.subscriberBuffer` — per-WebSocket-subscriber channel capacity; updates for a subscriber that cannot keep up are dropped, not blocked on
- `tilesStorage.pastUpdatesBuffer`, `tilesStorage.pastUpdatesRetention` — size and age bounds on the recent-updates ring buffer the bookkeeper reads
- `bookkeeper.enabled`, `bookkeeper.runner.interval`
- `rateLimiter.perSecond`, `rateLimiter.burst`, `rateLimiter.sweepInterval` — the per-IP click throttle (defaults 1/s, burst 10, swept every minute)

### Protobuf

API contracts live in the monorepo-shared [`/proto/planet/v1/planet.proto`](../../proto/planet/v1/planet.proto) (also used by the frontend). Generated code goes to `generated/proto/`. Use `make proto` to regenerate after editing `.proto` files (requires the `buf` CLI, plus `protoc-gen-go` and `protoc-gen-connect-go` on `PATH`).

The proto package is `planet.v1`, and it is the **only** version number: Connect derives its route from it, and `planetv1controller` and `planetv1connect` follow. There is no gRPC here — Connect serves the service definition over ordinary HTTP/1.1 POSTs (and h2c, for clients that want it).

### Testing

Unit tests only, using `testify`. There are no integration tests and no Docker dependency — `go test ./...` runs everything from a clean checkout.
