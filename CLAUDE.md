# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests
make test
# or: go test ./... | grep -v 'no test files'

# Run a single test
go test ./internal/clicks/domain/click_handler_service/... -run TestName

# Start Redis dev environment (required for integration tests and local dev)
make dbUp

# Stop Redis dev environment
make dbDown

# Generate protobuf code (requires buf CLI)
make proto

# Build Docker image
make dBuild

# Run API server locally
go run ./cmd/api -config cmd/api/example.yaml

# Run bookkeeper locally
go run ./cmd/bookkeeper -config cmd/bookkeeper/example.yaml
```

## Architecture

This is a Go backend for a collaborative map-clicking game. It follows **hexagonal architecture (ports & adapters)**.

### Two Applications

- **`cmd/api`** — HTTP/WebSocket server handling clicks, tile ownership queries, and real-time updates
- **`cmd/bookkeeper`** — Background job that reads the Redis stream and publishes updates to X/Twitter

Both follow the same init pattern: `New()` → `Configure()` → `Run()`.

### Domain Layer (`internal/clicks/domain/`)

Core interfaces (ports) defined in `gateways.go`:
- `TilesChecker` — validates tile IDs (0..maxIndex)
- `TileStorage` — reads/writes tile→country ownership
- `TileReporter` — notifies downstream of tile updates
- `CountryChecker` — validates ISO country codes

`ClickHandlerService` wires these interfaces together and contains all game logic. The Prometheus-instrumented version (`prom_click_handler_service/`) wraps it via decorator pattern.

### Adapters

**Primary (input):**
- `adapters/primary/http/clicks_controller/` — REST endpoints (`POST /v2/rpc/click`, `GET /v2/rpc/map-density`, `POST /v2/rpc/ownerships-by-batch`)
- `adapters/primary/http/websocket_publisher/` — subscribes to Redis stream, broadcasts to WebSocket clients

**Secondary (output):**
- `adapters/secondary/redis_tile_storage/` — persists tile ownership in Redis using a Lua script (`static/setAndPublishOnStream.lua`) for atomic SET + XADD
- `adapters/secondary/in_memory_tile_checker/` — validates tile IDs
- `adapters/secondary/in_memory_country_checker/` — validates country codes (hardcoded)
- `adapters/secondary/x_publisher/` — posts to X/Twitter

### Key Flow

```
POST /v2/rpc/click
  → ClicksController
  → ClickHandlerService (validates tile ID + country)
  → RedisTileStorage.Set() [Lua script: atomic SET + XADD to "tileUpdates" stream]
  → WebsocketPublisher (reads stream, fans out to WS clients)
```

### Kernel (`internal/kernel/`)

Shared infrastructure: `cfgutil` (YAML + env config via koanf), `httpserver` (middleware, formats), `logging`, `prom` (Prometheus), `xredis`, `xtime`, `ctxutil`, `xenvs`.

### Configuration

Config is loaded from a YAML file (`-config` flag) with environment variable override support. See `cmd/api/example.yaml` for the full schema. Key fields:
- `httpServer.bindAddress`, `httpServer.format` (`json` or `binary` for protobuf)
- `gameMap.maxIndex` — total number of tiles
- `redis.*` — Redis connection settings
- `tilesStorage.setAndPublishOnStreamSha1` — SHA1 of the Lua script (must match `static/setAndPublishOnStream.sha1`)

### Protobuf

API contracts live in `api/proto/clicks/v1/clicks.proto`. Generated code goes to `generated/proto/`. Use `make proto` to regenerate after editing `.proto` files (requires `buf` CLI).

### Testing

- Unit tests use `testify`
- Integration tests use `dockertest` to spin up real Redis containers
- `make dbUp` loads the Lua script into the dev Redis instance; integration tests handle their own Redis via dockertest
