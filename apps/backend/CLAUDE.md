# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests
make test
# or: go test ./... | grep -v 'no test files'

# Run a single test
go test ./internal/clicks/domain/click_handler_service/... -run TestName

# Run the concurrency-sensitive tests under the race detector
go test ./... -race

# Run API server locally (no external services needed)
go run ./cmd/api -config cmd/api/example.yaml

# Start Redis dev environment — ONLY needed for the redis driver.
# The dockertest integration tests spin up their own container.
make dbUp
make dbDown

# Copy tile state out of a running Redis into a snapshot file (one-off migration)
go run ./cmd/redis-to-snapshot -config cmd/api/example.yaml

# Generate protobuf code (requires buf CLI)
make proto

# Build Docker image
make dBuild
```

## Architecture

This is a Go backend for a collaborative map-clicking game. It follows **hexagonal architecture (ports & adapters)**.

### One application

- **`cmd/api`** — HTTP/WebSocket server handling clicks, tile ownership queries, and real-time updates. Follows `New()` → `Configure()` → `Run()`.
- **`cmd/redis-to-snapshot`** — one-off migration aid, not part of the running system.

`cmd/api` runs as a **single self-contained container** by default: the tile map lives in process and is persisted to a local snapshot file. There is no second service to run.

The X/Twitter reporting job used to be a separate `cmd/bookkeeper` process. It now runs inside `cmd/api` as an optional goroutine behind `bookkeeper.enabled` (default off) — with the memory driver the recent-updates window only exists inside the API process, so a separate process would have nothing to read.

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
- `adapters/secondary/memory_tile_storage/` — **the default.** Holds the map in a preallocated `[]uint16` indexed by tile id, with country codes interned into a side table (2 bytes per tile — ~2 MB for a 1M-tile map). Fans updates out in process, serves `PastUpdates` from a bounded ring buffer, and persists to a local snapshot file.
- `adapters/secondary/redis_tile_storage/` — the original Redis-backed storage, kept so rollback is a config change. Persists tile ownership in Redis using a Lua script (`static/setAndPublishOnStream.lua`) for atomic SET + XADD to the `tileUpdates` stream.
- `adapters/secondary/in_memory_tile_checker/` — validates tile IDs
- `adapters/secondary/in_memory_country_checker/` — validates country codes (hardcoded)
- `adapters/secondary/x_publisher/` — posts to X/Twitter

Both storage adapters expose the same three things — the `domain.TileStorage` port, `Subscribe(ctx) (<-chan domain.TileUpdate, error)` and `PastUpdates(ctx, duration, now)` — so they are interchangeable. That contract is the unexported `tilesStorage` interface in `internal/clicks/app/v2.go`; keep them in step when changing either.

### Key Flow

```
POST /v2/rpc/click
  → ClicksController
  → ClickHandlerService (validates tile ID + country)
  → MemoryTileStorage.Set() [writes the tile, fans the update out in process]
  → WebsocketPublisher (fans out to WS clients)
```

`Set` is a no-op when the tile already holds that value — no write, no update published. `memory_tile_storage` mirrors that behaviour from `static/setAndPublishOnStream.lua`; keep the two in step.

### Durability (memory driver)

The whole map is snapshotted to `tilesStorage.memory.snapshotPath`:
- a compact binary encoding, not JSON: magic + version + CRC32, then the interned country code table, then two bytes per tile
- written atomically — temp file, fsync, `os.Rename`, fsync of the directory — so a crash mid-write leaves the previous snapshot intact
- flushed every `snapshotInterval` when the state changed, and once more on graceful shutdown (`cmd/api` handles SIGINT/SIGTERM for exactly this)
- restored at boot; a missing, truncated, or corrupt snapshot logs and starts from an empty map, it never prevents a start
- a snapshot taken at a different `gameMap.maxIndex` restores the overlap

**What this trades away:** anything written since the last snapshot is lost on a hard kill (`SIGKILL`, OOM, power loss), bounded by `snapshotInterval`. And because the state is per-process, running more than one API instance would give each its own divergent map — the memory driver is single-instance only.

### Kernel (`internal/kernel/`)

Shared infrastructure: `cfgutil` (YAML + env config via koanf), `httpserver` (middleware, formats), `logging`, `prom` (Prometheus), `xredis`, `xtime`, `ctxutil`, `xenvs`.

### Configuration

Config is loaded from a YAML file (`-config` flag), with environment variables overriding it — `cfgutil` uses `.` as the nesting delimiter, so `tilesStorage.driver=redis` in the environment overrides the file. See `cmd/api/example.yaml` for the full schema.

- `httpServer.bindAddress`, `httpServer.format` (`json` or `binary` for protobuf)
- `gameMap.maxIndex` — total number of tiles
- `tilesStorage.driver` — `memory` (default) or `redis`
- `tilesStorage.memory.*` — `snapshotPath` (empty disables durability), `snapshotInterval`, `subscriberBuffer`, `pastUpdatesBuffer`, `pastUpdatesRetention`
- `tilesStorage.redis.setAndPublishOnStreamSha1` — SHA1 of the Lua script (must match `static/setAndPublishOnStream.sha1`). **Only read by the redis driver** — the default path needs no Lua script and no sha1 to keep in sync.
- `redis.*` — connection settings. **Only dialled when `driver: redis`.**
- `bookkeeper.enabled`, `bookkeeper.runner.interval`

### Protobuf

API contracts live in the monorepo-shared [`/proto/clicks/v1/clicks.proto`](../../proto/clicks/v1/clicks.proto) (also used by the frontend). Generated code goes to `generated/proto/`. Use `make proto` to regenerate after editing `.proto` files (requires `buf` CLI).

### Testing

- Unit tests use `testify`
- `memory_tile_storage` is covered by unit tests only — no Docker required
- The `redis_tile_storage` integration tests use `dockertest` to spin up real Redis containers
- `make dbUp` loads the Lua script into the dev Redis instance; integration tests handle their own Redis via dockertest
