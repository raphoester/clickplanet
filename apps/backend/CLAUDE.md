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
- `adapters/primary/http/clicks_v3_controller/` — the v3 API, as two separate types:
  - `ClickService` implements the generated `planetv1connect.ClickServiceHandler` and nothing else. It never sees an `http.ResponseWriter` — that is the point of serving the contract with Connect rather than by hand. Served at `POST /v3/planet.v1.ClickService/<Method>`.
  - `MapHandler` is a plain `http.Handler` for `GET /v3/map`, which is deliberately not an RPC (see below).
  - `NewErrorInterceptor` maps domain errors onto Connect codes, logs the unexpected ones and keeps their cause off the wire. **Handlers return their errors bare** — `domain.ErrInvalidArgument` becomes `CodeInvalidArgument`, a code a handler picked itself is left alone, and anything else is logged once and answered as `internal error`. No handler carries a logger or repeats that block.

  Neither declares its own routes: `app/wiring.go` mounts them, the way connect-go's own getting-started does.
- `adapters/primary/http/clicks_controller/` — **deprecated** v2 endpoints (`POST /v2/rpc/click`, `GET /v2/rpc/map-density`, `POST /v2/rpc/ownerships-by-batch`). They wrap binary protobuf in a base64 JSON envelope and pick that encoding from `httpServer.format` rather than from the request. Frozen; mounted until the deployed frontends move.
- `adapters/primary/http/websocket_publisher/` — subscribes to the tile update stream, broadcasts to WebSocket clients

Both versions are wired to the same domain instances in `app/wiring.go`, and both serve the same websocket (`/v2/ws/listen` and `/v3/ws/listen`) — the tile map lives in this process, so two sets of adapters over two storages would be two different games.

**There is one server and one mux.** Connect handlers are ordinary `http.Handler`s, so the v3 RPCs, the v2 REST endpoints, the map GET, the websocket upgrade and `/metrics` all mount on the same `http.ServeMux` on one port. Nothing here needs a connection-level demultiplexer such as `cmux`; that is for running a real gRPC server, which owns its own HTTP/2 handler, beside a REST one.

`Configure` sets `http.Protocols` with both HTTP/1.1 and unencrypted HTTP/2, because the generated handler also speaks gRPC and gRPC-Web and those need HTTP/2. Browsers reach the same routes over HTTP/1.1. Verified: HTTP/1.1 and h2c both answer on the same port.

### The v3 map encoding

`GET /v3/map?start=&end=` (both optional, defaulting to the whole map) answers `application/octet-stream` in the layout documented in `memory_tile_storage/wire.go`: a magic, the interned country code table, then two bytes per tile. Tile ids are implicit in the position, which is what makes it about seven times smaller than the v2 `map<uint32, string>` — 516 KB against 3.6 MB for a full 257,948-tile map. The interned ids are written as stored and the table travels with them, so nothing translates on the way out and the client needs no shared country list.

It is a GET and says `Cache-Control: public, max-age=5`, so a burst of visitors can share one origin response. The websocket carries everything that happens after a chunk was built, so a client starting from a slightly old map converges anyway.

**Secondary (output):**
- `adapters/secondary/memory_tile_storage/` — the tile map. A preallocated `[]uint16` indexed by tile id, with country codes interned into a side table (2 bytes per tile — ~2 MB for a 1M-tile map). Fans updates out in process, serves `PastUpdates` from a bounded ring buffer, and persists to a local snapshot file.
- `adapters/secondary/in_memory_tile_checker/` — validates tile IDs
- `adapters/secondary/in_memory_country_checker/` — validates country codes (hardcoded)
- `adapters/secondary/x_publisher/` — posts to X/Twitter

Beyond the `domain.TileStorage` port, `memory_tile_storage` also exposes `Subscribe(ctx) (<-chan domain.TileUpdate, error)` for the websocket publisher and `PastUpdates(ctx, duration, now)` for the bookkeeper.

### Key Flow

```
POST /v2/rpc/click
  → ClicksController
  → ClickHandlerService (validates tile ID + country)
  → MemoryTileStorage.Set() [writes the tile, fans the update out in process]
  → WebsocketPublisher (fans out to WS clients)
```

`Set` is a no-op when the tile already holds that value — no write, no update published.

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

Shared infrastructure: `cfgutil` (YAML + env config via koanf), `httpserver` (middleware, formats), `logging`, `prom` (Prometheus), `xtime`, `ctxutil`, `basicutil`.

### Configuration

Config is loaded from a YAML file (`-config` flag), with environment variables overriding it — `cfgutil` uses `.` as the nesting delimiter, so `tilesStorage.snapshotPath=/data/tiles` in the environment overrides the file. See `cmd/api/example.yaml` for the full schema.

- `httpServer.bindAddress`, `httpServer.format` (`json` or `binary` for protobuf) — v2 only. v3 negotiates the encoding per request.
- `gameMap.maxIndex` — total number of tiles
- `tilesStorage.snapshotPath` — where the state is persisted; **empty disables durability**
- `tilesStorage.snapshotInterval` — how often a changed state is flushed
- `tilesStorage.subscriberBuffer` — per-WebSocket-subscriber channel capacity; updates for a subscriber that cannot keep up are dropped, not blocked on
- `tilesStorage.pastUpdatesBuffer`, `tilesStorage.pastUpdatesRetention` — size and age bounds on the recent-updates ring buffer the bookkeeper reads
- `bookkeeper.enabled`, `bookkeeper.runner.interval`

### Protobuf

API contracts live in the monorepo-shared [`/proto/planet/v1/planet.proto`](../../proto/planet/v1/planet.proto) (also used by the frontend). Generated code goes to `generated/proto/`. Use `make proto` to regenerate after editing `.proto` files (requires the `buf` CLI, plus `protoc-gen-go` and `protoc-gen-connect-go` on `PATH`).

The proto package is `planet.v1`, and stays at `v1`: "v3" is the HTTP API version and lives in the URL prefix, not in the package name. There is no gRPC here — Connect serves the same service definition over ordinary HTTP/1.1 POSTs.

### Testing

Unit tests only, using `testify`. There are no integration tests and no Docker dependency — `go test ./...` runs everything from a clean checkout.
