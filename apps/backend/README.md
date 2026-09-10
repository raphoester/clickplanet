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
POST /planet.v1.ClickService/Click
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
| Language         | Go 1.27                                                  |
| Real-time        | WebSockets (`coder/websocket`), in-process fan-out       |
| Storage          | In-memory, with binary snapshots to a local file         |
| API contracts    | Protocol Buffers over Connect (no gRPC)                  |
| Metrics          | Prometheus (decorator pattern over the core service)     |
| Config           | YAML + environment variable overrides (`koanf`)          |
| Scheduling       | `gocron` (bookkeeper interval jobs)                      |
| Testing          | `testify` (unit tests only — no Docker needed)           |
| Containerization | Docker (multi-stage build, non-root runtime)             |

## API

| Method | Path                                     | Description                                |
|--------|------------------------------------------|--------------------------------------------|
| `POST` | `/planet.v1.ClickService/Click`          | Claim a tile for a country                 |
| `GET`  | `/planet.v1.ClickService/MapDensity`     | Total number of tiles on the map           |
| `GET`  | `/planet.v1.ClickService/GetMap`         | Bulk fetch tile ownership for a tile range |
| `POST` | `/planet.v1.ClickService/ListenForUpdates` | Server stream of real-time tile updates  |
| `GET`  | `/ws/listen`                             | The same stream, as a WebSocket             |
| `GET`  | `/metrics`                               | Prometheus metrics                         |

The RPCs are served with [Connect](https://connectrpc.com), which is plain HTTP — no gRPC. The encoding is negotiated per request (`application/proto` or `application/json`), and the two reads are marked side-effect free, so they arrive as cacheable GETs.

`ListenForUpdates` is a server-streaming RPC carrying typed `TileUpdate` messages. `/ws/listen` carries the same updates as raw binary frames and is **kept only until the deployed frontend has moved over**.

`Click` is rate limited per source IP — 1 click/s with a burst of 10 by default, configurable under `rateLimiter`. Over that, it answers `429`. The reads and the streams are not limited.

## Running locally

```bash
# Run the API server — nothing to start first
go run ./cmd/api -config cmd/api/example.yaml

# Run tests
make test
```

See `cmd/api/example.yaml` for the full configuration schema.
