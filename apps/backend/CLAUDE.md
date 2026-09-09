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

# Refresh the vendored VPN/datacenter ranges from upstream, then commit them
make vpn-lists

# Build Docker image
make dBuild
```

## Architecture

This is a Go backend for a collaborative map-clicking game. It follows **hexagonal architecture (ports & adapters)**.

### Two bounded contexts, one process

- **`internal/clicks/`** — the tile game: clicks, ownership, the map, the update stream.
- **`internal/chat/`** — the live chat: messages, identity, retention.

They share the process, the transport and the country list, and **nothing else**. Neither imports the other; each owns its own domain types, its own proto package, its own storage adapter and its own edge. The one place that knows about both is **`internal/app/`**, the composition root — which is why it sits beside them rather than inside either.

Adding a third context means a `proto/<name>/v1`, an `internal/<name>/`, and one `configure<Name>` in `internal/app`. Connect derives the route from the proto package, so there is no prefix to allocate and no router to edit.

**`cmd/api`** is the only binary: an HTTP/WebSocket server serving both contexts. Follows `New()` → `Configure()` → `Run()`.

It runs as a **single self-contained container with no dependencies**: the tile map lives in process and is persisted to a local snapshot file. There is no database, no cache, and no second process.

The X/Twitter reporting job used to be a separate `cmd/bookkeeper` process. It now runs inside `cmd/api` as an optional goroutine behind `bookkeeper.enabled` (default off) — the recent-updates window only exists inside the API process, so a separate process would have nothing to read.

### Clicks domain (`internal/clicks/domain/`)

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
- the tile stream, broadcast by a `kernel/wspublisher` instance the composition root builds from `planetv1controller.TileUpdateRoute` and `EncodeTileUpdate`.

**There is one server, one mux, and no version prefix.** Connect names each service's path from its proto package — `/planet.v1.ClickService/` and `/chat.v1.ChatService/` — so nothing is mounted under a prefix of ours. The two services, the websocket upgrade and `/metrics` are the only things on the router. Nothing here needs a connection-level demultiplexer such as `cmux`; that is for running a real gRPC server, which owns its own HTTP/2 handler, beside a REST one.

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
  → wspublisher (fans out to /ws/listen clients)
```

`Set` is a no-op when the tile already holds that value — no write, no update published.

```
POST /chat.v1.ChatService/SendMessage
  → GuardInterceptor (block list, then the per-IP throttle)
  → ChatService
  → chat_service (sanitizes, stamps id/time/tag)
  → MemoryChatStorage.Append() [appends to the JSONL log, then fans out]
  → wspublisher (fans out to /ws/chat clients)
```

A failed log write fails the whole post: the log is the audit trail, so a message nobody can account for later is not one that gets broadcast.

### Chat (`internal/chat/`)

Chat is a separate bounded context, not a feature of the tile game: it shares the process, the transport and the country list, and has its own proto package, domain, storage and edge. Nothing under `internal/chat/` imports `internal/clicks/`, and the reverse holds too.

**Off by default.** With `chat.enabled` false nothing is registered, so `/chat.v1.ChatService/` and `/ws/chat` both answer 404 — the unauthenticated public write endpoint does not exist at all rather than existing and erroring.

**Identity without accounts.** A client picks its own display name and sends a UUID it persists locally. **Neither is trusted for anything** — anyone can post with any name. What a sender cannot forge is `author_tag`: a salted hash of their IP, 6 hex characters, so two people using the same name still look different and a mute has a key that means something. The salt is `chat.service.tagSalt`; left empty it is regenerated at boot, which changes everyone's tag on restart, and the server warns about it.

**Abuse controls live at the edge**, in `NewGuardInterceptor` — ahead of decoding the message and well ahead of validating it, so a flood of malformed messages costs a sender exactly what a flood of well-formed ones does. `chat.blockedIPs` cuts an address off from every chat RPC; `chat.rateLimiter` throttles `SendMessage` alone (`GetHistory` is one read on join, and limiting it would punish a page load). The domain then bounds the message in **runes** (280) and the name (24), validates UTF-8, and **strips control characters** — a newline would otherwise let a sender forge a line in the JSONL log.

Refusal reasons are logged, never returned: a sender learns *that* they were refused, not which check tripped. **The stored text is raw — the frontend must escape it.**

The [VPN blocklist](#vpn-blocklist) does **not** cover chat: it wraps `Click` alone. If chat turns out to need it, it is the same `ipblock.Blocklist` and a second interceptor, not a second list.

**The log is an append-only JSONL file**, not the tile snapshot's whole-state codec: different shape, different write pattern. One line per message with `at`, `id`, `name`, `tag`, `authorId`, `country`, `ip`, `userAgent`, `text`. It is fsynced every `flushInterval` rather than per message (a hard kill loses at most that window — the same bargain the snapshot makes), pruned hourly past `retention`, and its tail repopulates the in-memory history at boot so a restart does not blank the chat. Corrupt lines are skipped and reported, never fatal.

That log holds **personal data** — IPs next to user-authored text — so the retention window is a policy decision rather than a cache size. It lives on the `tile_state` volume, which the droplet's weekly disk backup already covers.

**Two streams, two routes.** Chat broadcasts on `/ws/chat`, not on `/ws/listen`. Frames carry a bare protobuf message with no type tag, so a second payload on the tile stream would be indistinguishable from a `TileUpdate` to every already-deployed client. `kernel/wspublisher` is generic over its payload and takes a route, instantiated once per stream.

**Sending is an RPC, not a read on the socket**: both publishers lean on `CloseRead` for instant disconnect detection, and the RPC path already has the middleware stack and the interceptors.

`GetHistory` is marked `NO_SIDE_EFFECTS`, so Connect sends it as a GET — but it answers `Cache-Control: no-store`, the opposite of `GetMap`. A client fetches it once on join to seed what the websocket then keeps up to date, so a cached answer would show a joiner a chat missing the last few minutes.

### Rate limiting

`NewRateLimitInterceptor` throttles **`Click` only**, per source IP, from a `kernel/ratelimit` token bucket — 1 click/s with a burst of 10 by default (`rateLimiter.*`). `MapDensity` and `GetMap` are cacheable reads a proxy in front absorbs; limiting them would punish a page load rather than a bot. A refused click answers `CodeResourceExhausted`, i.e. HTTP 429, and never reaches the domain.

The bucket key is whatever `IPReaderMiddleware` put on the context: `X-Real-IP` if present, otherwise the peer address. **The reverse proxy must set that header itself** — `deploy/vps/Caddyfile` does, with `header_up X-Real-IP {client_ip}` on every backend route. Merely forwarding it would let a client send its own and buy a fresh bucket per request. The fallback is the peer address rather than a constant precisely so a missing header degrades to per-connection buckets instead of rate limiting the whole game as one player.

Chat has **its own limiter instance** with its own budget (`chat.rateLimiter`, one message every 3s with 5 in hand by default). A message fans out to every connected client and lands in a log everyone will read, so it costs far more than a click and the two budgets have nothing to do with each other.

### VPN blocklist

`NewVPNBlockInterceptor` refuses **`Click` only**, with `CodePermissionDenied` (HTTP 403), when the source address falls in a vendored VPN range. It sits **outside the rate limiter** in the interceptor chain, deliberately: a refused address must not also spend a token, or the next click would come back 429 and the web app would show the throttle dialog instead of the VPN one.

**It exists because of the rate limiter, not instead of it.** The bucket is keyed on an address, and a commercial VPN is the cheapest way to get a fresh one; refusing those addresses is what makes the bucket hold. It raises the floor rather than closing the door — residential proxies appear in no public list, and nothing here stops one.

Reads and the websocket are untouched. A VPN user still loads the planet and follows it live; they cannot paint. That is also what keeps a false positive readable: the page works and says why, instead of failing to load.

**The ranges are vendored and embedded**, from [X4BNet/lists_vpn](https://github.com/X4BNet/lists_vpn) (MIT, rebuilt daily from ASN ownership), in `internal/kernel/ipblock/data`. Not fetched at boot: `cmd/api` is a self-contained container with no startup dependencies, and a boot that can fail because GitHub is down is a worse trade than a list that ages between deploys — the Cloudflare ranges in `deploy/vps/Caddyfile` are maintained the same way. Refresh with `make vpn-lists` and commit; the tests assert the lists still parse and are not truncated.

`ipblock` holds them as sorted, merged `[lo, hi]` ranges of 16-byte addresses and binary-searches them — ~63k prefixes fold to far fewer ranges, about 2 MB resident and well under 100 ns per lookup. **IPv4 and IPv6 live in separate slices.** They cannot share one: an IPv4 address in its v4-mapped form sits inside `::ffff:0:0/96`, so a single ordering would let a v6 prefix as short as `::/16` silently swallow every IPv4 address on the internet.

`vpnBlocklist.includeDatacenters` adds the much broader hosting list, which catches a self-hosted VPN on a VPS. It is off by default because it also refuses Apple iCloud Private Relay and Cloudflare WARP — both egress from datacenter ranges, both on by default for a lot of ordinary mobile traffic. `blocked_clicks{list}` is labelled per list precisely so you can see what turning it on would cost before turning it on. `vpnBlocklist.allow` is the escape hatch and beats both lists.

**The `X-Real-IP` caveat applies here too, and matters more.** Anything that reaches `backend:8080` directly bypasses the blocklist by sending its own header, exactly as it bypasses the throttle. Caddy replaces the header on every backend route, so this is only reachable if the backend port is exposed — but the bypass is now security-relevant rather than merely an abuse nuisance.

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

Shared infrastructure: `cfgutil` (YAML + env config via koanf), `httpserver` (middleware, formats), `logging`, `prom` (Prometheus), `xtime`, `ctxutil`, `basicutil`, `ratelimit`, `ipblock`, `atomicfile`, `wspublisher`.

Two of these are here because both bounded contexts need them and neither should depend on the other:

- `wspublisher` — the WebSocket fanout, generic over its payload and its encoder. It knows nothing about what it carries, so the payload type and the wire encoding stay with the context that owns them.
- `atomicfile` — temp file, fsync, rename, fsync of the directory. Written for the tile snapshot; the chat log's retention rewrites need the same guarantee, and duplicating 80 lines of carefully-written fsync/rename code is how the two drift apart. Covered by the existing snapshot tests.

`ipblock` is the VPN prefix set — see [VPN blocklist](#vpn-blocklist). `ratelimit` is a keyed token bucket held in this process, like the tile map it protects — with one API instance, a shared counter would buy nothing. Its `Run` loop periodically forgets the buckets that have refilled to capacity, which is free: such a bucket holds exactly what a freshly created one would, and without it the map would keep an entry per address that ever clicked.

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
- `vpnBlocklist.enabled`, `vpnBlocklist.includeDatacenters`, `vpnBlocklist.allow` — the VPN refusal (see [VPN blocklist](#vpn-blocklist)); disabled parses nothing and allocates nothing
- `chat.enabled` — the kill switch; off means the routes are never registered
- `chat.storage.logPath` — the JSONL message log; **empty keeps chat entirely in memory**
- `chat.storage.historySize`, `chat.storage.retention`, `chat.storage.flushInterval`, `chat.storage.pruneInterval`, `chat.storage.subscriberBuffer`
- `chat.service.tagSalt` — salts the per-sender tag; **empty regenerates one at boot**, changing every tag on restart
- `chat.service.maxTextLength`, `chat.service.maxNameLength` — bounds in runes (280, 24)
- `chat.rateLimiter.*` — the per-IP `SendMessage` throttle, same shape as `rateLimiter`
- `chat.blockedIPs` — addresses refused every chat RPC

### Protobuf

API contracts live in the monorepo-shared [`/proto`](../../proto) (also used by the frontend), one package per bounded context: [`planet/v1/planet.proto`](../../proto/planet/v1/planet.proto) and [`chat/v1/chat.proto`](../../proto/chat/v1/chat.proto). Generated code goes to `generated/proto/`. Use `make proto` to regenerate after editing `.proto` files (requires the `buf` CLI, plus `protoc-gen-go` and `protoc-gen-connect-go` on `PATH`).

The proto package is the **only** version number: Connect derives each route from it, and the controller and connect package names follow — `planet.v1` gives `planetv1controller` and `planetv1connect`, `chat.v1` gives `chatv1controller` and `chatv1connect`. There is no gRPC here — Connect serves the service definitions over ordinary HTTP/1.1 POSTs (and h2c, for clients that want it).

### Testing

Unit tests only, using `testify`. There are no integration tests and no Docker dependency — `go test ./...` runs everything from a clean checkout.
