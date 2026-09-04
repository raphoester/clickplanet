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
- `adapters/primary/http/clicks_controller/` — REST endpoints (`POST /v2/rpc/click`, `GET /v2/rpc/map-density`, `POST /v2/rpc/ownerships-by-batch`)
- `adapters/primary/http/websocket_publisher/` — subscribes to the tile update stream, broadcasts to WebSocket clients

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

- `httpServer.bindAddress`, `httpServer.format` (`json` or `binary` for protobuf)
- `gameMap.maxIndex` — total number of tiles
- `tilesStorage.snapshotPath` — where the state is persisted; **empty disables durability**
- `tilesStorage.snapshotInterval` — how often a changed state is flushed
- `tilesStorage.subscriberBuffer` — per-WebSocket-subscriber channel capacity; updates for a subscriber that cannot keep up are dropped, not blocked on
- `tilesStorage.pastUpdatesBuffer`, `tilesStorage.pastUpdatesRetention` — size and age bounds on the recent-updates ring buffer the bookkeeper reads
- `bookkeeper.enabled`, `bookkeeper.runner.interval`

### Protobuf

API contracts live in the monorepo-shared [`/proto/clicks/v1/clicks.proto`](../../proto/clicks/v1/clicks.proto) (also used by the frontend). Generated code goes to `generated/proto/`. Use `make proto` to regenerate after editing `.proto` files (requires `buf` CLI).

### Testing

Unit tests only, using `testify`. There are no integration tests and no Docker dependency — `go test ./...` runs everything from a clean checkout.
