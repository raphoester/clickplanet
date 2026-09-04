# ClickPlanet — Backend

Backend for [clickplanet.lol](https://clickplanet.lol), a real-time multiplayer game where countries compete to own tiles on a world map. Written in Go.

## How it works

Every tile on the map has an owner (a country code). Players click tiles to claim them for their country. All connected clients see ownership changes in real time.

It is a **single binary with no dependencies** — `api`. The whole game state is ~1M tiles × a two-byte country code, so it lives in memory and is snapshotted to a local file rather than in a database.

An optional background job reports recent activity to X/Twitter. It used to be a second binary; it now runs inside `api` as a goroutine behind `bookkeeper.enabled` (off by default), because the recent-updates window only exists inside the API process.

## Architecture

The project follows **hexagonal architecture** (ports & adapters), keeping the domain model isolated from infrastructure concerns.

```
internal/clicks/
├── domain/          # Core interfaces and business logic
├── adapters/
│   ├── primary/     # Inbound: HTTP REST, WebSocket
│   └── secondary/   # Outbound: tile storage, X publisher
└── app/             # Wires everything together
```

**Click flow:**

```
POST /v2/rpc/click
  → validate country + tile ID
  → write the tile in memory, fan the update out in process
  → WebSocket publisher fans updates out to all connected clients
```

Claiming a tile for the country that already owns it writes nothing and publishes nothing, so redundant fan-out is avoided.

**Storage:** a preallocated `[]uint16` indexed by tile id, with country codes interned into a side table — two bytes per tile, ~2 MB for a million of them. It is snapshotted to disk periodically and on graceful shutdown, in a compact binary format written atomically (temp file + fsync + rename). A missing or corrupt snapshot starts an empty map rather than blocking a start.

The tradeoffs are deliberate: writes since the last snapshot are lost on a hard kill, and the server is single-instance, since two replicas would each hold their own map.

## Stack

| Concern          | Technology                                               |
|------------------|----------------------------------------------------------|
| Language         | Go 1.23                                                  |
| Real-time        | WebSockets (`coder/websocket`), in-process fan-out       |
| Storage          | In-memory, with binary snapshots to a local file         |
| API contracts    | Protocol Buffers (supports JSON and binary wire formats) |
| Metrics          | Prometheus (decorator pattern over the core service)     |
| Config           | YAML + environment variable overrides (`koanf`)          |
| Scheduling       | `gocron` (bookkeeper interval jobs)                      |
| Testing          | `testify` (unit tests only — no Docker needed)           |
| Containerization | Docker (multi-stage build, non-root runtime)             |

## API

| Method | Path                          | Description                                |
|--------|-------------------------------|--------------------------------------------|
| `POST` | `/v2/rpc/click`               | Claim a tile for a country                 |
| `GET`  | `/v2/rpc/map-density`         | Total number of clicks across the map      |
| `POST` | `/v2/rpc/ownerships-by-batch` | Bulk fetch tile ownership for a tile range |
| `GET`  | `/v2/ws/listen`               | WebSocket stream of real-time tile updates |
| `GET`  | `/metrics`                    | Prometheus metrics                         |

Request/response bodies use Protocol Buffers. The server supports both binary and JSON wire formats, configured via `httpServer.format`.

## Running locally

```bash
# Run the API server — nothing to start first
go run ./cmd/api -config cmd/api/example.yaml

# Run tests
make test
```

See `cmd/api/example.yaml` for the full configuration schema.
