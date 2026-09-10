# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests — no Docker, no database, nothing to start first
make test
# or: go test -tags testing ./... | grep -v 'no test files'

# Run a single test
go test -tags testing ./internal/clicks/domain/click_handler_service/... -run TestName

# Run the concurrency-sensitive tests under the race detector
go test -tags testing ./... -race

# Fail on any unreachable function, production or test helper
make deadcode

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

### Three bounded contexts, one process

- **`internal/clicks/`** — the tile game: clicks, ownership, the map, the update stream.
- **`internal/chat/`** — the live chat: messages, identity, retention.
- **`internal/session/`** — the mint: what a caller has to prove before it may click.

They share the process, the transport and the country list, and **nothing else**. None imports another; each owns its own domain types, its own proto package, its own storage adapter (where it has one) and its own edge. The one place that knows about all three is **`internal/app/`**, the composition root — which is why it sits beside them rather than inside any of them.

Clicks and sessions meet only through `kernel/session.Signer`: the session context mints, the clicks context verifies a signature. Neither imports the other, and **clicks knows nothing about Turnstile** — swapping the attester changes one line in `internal/app`.

Adding a fourth context means a `proto/<name>/v1`, an `internal/<name>/`, and one `configure<Name>` in `internal/app`. Connect derives the route from the proto package, so there is no prefix to allocate and no router to edit. `internal/session/` is the worked example: it cost one proto file, one domain rule, two adapters and one wiring file, and no existing route changed.

**`cmd/api`** is the only binary: an HTTP/WebSocket server serving all three contexts. Follows `New()` → `Configure()` → `Run()`.

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
POST /session.v1.SessionService/CreateSession
  → SessionService
  → session_service (attests, then mints)
  → turnstile_attester → Cloudflare siteverify
  → kernel/session.Signer.Mint [HMAC over expiry+id+IP; nothing stored]

POST /planet.v1.ClickService/Click   [X-Session-Token: <the minted token>]
  → VPNBlockInterceptor, SessionInterceptor, RateLimitInterceptor
  → ClickService
  → ClickHandlerService (validates tile ID + country)
  → MemoryTileStorage.Set() [writes the tile, fans the update out in process]
  → wspublisher (fans out to /ws/listen clients)
```

`Set` is a no-op when the tile already holds that value — no write, no update published.

```
POST /chat.v1.ChatService/SendMessage
  → BlocklistInterceptor, then RateLimitInterceptor (both kernel/connectutil)
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

**Abuse controls live at the edge**, in two interceptors — ahead of decoding the message and well ahead of validating it, so a flood of malformed messages costs a sender exactly what a flood of well-formed ones does. `chat.blockedIPs` cuts an address off from every chat RPC; `chat.rateLimiter` throttles `SendMessage` alone (`GetHistory` is one read on join, and limiting it would punish a page load). **Both are `kernel/connectutil`'s**, the same ones the click chain uses — see [Shared interceptors](#shared-interceptors). The domain then bounds the message in **runes** (280) and the name (24), validates UTF-8, and **strips control characters** — a newline would otherwise let a sender forge a line in the JSONL log.

Refusal reasons are logged, never returned: a sender learns *that* they were refused, not which check tripped. **The stored text is raw — the frontend must escape it.**

The **vendored VPN lists** do not cover chat: `NewVPNBlockInterceptor` wraps `Click` alone. Chat's blocklist is the same `*ipblock.Blocklist` type, built by `ipblock.NewDenyList` from config prefixes instead of vendored data — so entries are CIDRs and a `/24` is one line rather than 256. Extending the vendored lists to chat is therefore a wiring change (hand `configureVPNBlocklist`'s result to `NewBlocklistInterceptor`), not a second list to write.

**The log is an append-only JSONL file**, not the tile snapshot's whole-state codec: different shape, different write pattern. One line per message with `at`, `id`, `name`, `tag`, `authorId`, `country`, `ip`, `userAgent`, `text`. It is fsynced every `flushInterval` rather than per message (a hard kill loses at most that window — the same bargain the snapshot makes), pruned hourly past `retention`, and its tail repopulates the in-memory history at boot so a restart does not blank the chat. Corrupt lines are skipped and reported, never fatal.

That log holds **personal data** — IPs next to user-authored text — so the retention window is a policy decision rather than a cache size. It lives on the `tile_state` volume, which the droplet's weekly disk backup already covers.

**Two streams, two routes.** Chat broadcasts on `/ws/chat`, not on `/ws/listen`. Frames carry a bare protobuf message with no type tag, so a second payload on the tile stream would be indistinguishable from a `TileUpdate` to every already-deployed client. `kernel/wspublisher` is generic over its payload and takes a route, instantiated once per stream.

**Sending is an RPC, not a read on the socket**: both publishers lean on `CloseRead` for instant disconnect detection, and the RPC path already has the middleware stack and the interceptors.

`GetHistory` is marked `NO_SIDE_EFFECTS`, so Connect sends it as a GET — but it answers `Cache-Control: no-store`, the opposite of `GetMap`. A client fetches it once on join to seed what the websocket then keeps up to date, so a cached answer would show a joiner a chat missing the last few minutes.

### Rate limiting

`NewRateLimitInterceptor` throttles **`Click` only**, per source IP, from a `kernel/ratelimit` token bucket — 1 click/s with a burst of 10 by default (`rateLimiter.*`). `MapDensity` and `GetMap` are cacheable reads a proxy in front absorbs; limiting them would punish a page load rather than a bot. A refused click answers `CodeResourceExhausted`, i.e. HTTP 429, and never reaches the domain.

The bucket key is whatever `IPReaderMiddleware` put on the context: `X-Real-IP` if present, otherwise the peer address. **The reverse proxy must set that header itself** — `deploy/vps/Caddyfile` does, with `header_up X-Real-IP {client_ip}` on every backend route. Merely forwarding it would let a client send its own and buy a fresh bucket per request. The fallback is the peer address rather than a constant precisely so a missing header degrades to per-connection buckets instead of rate limiting the whole game as one player.

Chat and sessions each have **their own limiter instance** with their own budget, because what each call costs has nothing to do with what a click costs:

- `chat.rateLimiter` — one message every 3s, five in hand. A message fans out to every connected client and lands in a log everyone will read.
- `session.rateLimiter` — one mint every 30s, ten in hand. A mint costs a siteverify round trip to a third party, so an unthrottled `CreateSession` is a free way to spend this server's Turnstile quota.

### VPN blocklist

`NewVPNBlockInterceptor` refuses **`Click` only**, with `CodePermissionDenied` (HTTP 403), when the source address falls in a vendored VPN range. It sits **outside the rate limiter** in the interceptor chain, deliberately: a refused address must not also spend a token, or the next click would come back 429 and the web app would show the throttle dialog instead of the VPN one.

**It exists because of the rate limiter, not instead of it.** The bucket is keyed on an address, and a commercial VPN is the cheapest way to get a fresh one; refusing those addresses is what makes the bucket hold. It raises the floor rather than closing the door — residential proxies appear in no public list, and nothing here stops one. **That gap is what [Sessions](#sessions-internalsession) closes**, by requiring something an address cannot buy; chasing list completeness instead is a treadmill.

Reads and the websocket are untouched. A VPN user still loads the planet and follows it live; they cannot paint. That is also what keeps a false positive readable: the page works and says why, instead of failing to load.

**The ranges are vendored and embedded**, from [X4BNet/lists_vpn](https://github.com/X4BNet/lists_vpn) (MIT, rebuilt daily from ASN ownership), in `internal/kernel/ipblock/data`. Not fetched at boot: `cmd/api` is a self-contained container with no startup dependencies, and a boot that can fail because GitHub is down is a worse trade than a list that ages between deploys — the Cloudflare ranges in `deploy/vps/Caddyfile` are maintained the same way. Refresh with `make vpn-lists` and commit; the tests assert the lists still parse and are not truncated.

`ipblock` holds them as sorted, merged `[lo, hi]` ranges of 16-byte addresses and binary-searches them — ~63k prefixes fold to far fewer ranges, about 2 MB resident and well under 100 ns per lookup. **IPv4 and IPv6 live in separate slices.** They cannot share one: an IPv4 address in its v4-mapped form sits inside `::ffff:0:0/96`, so a single ordering would let a v6 prefix as short as `::/16` silently swallow every IPv4 address on the internet.

`vpnBlocklist.includeDatacenters` adds the much broader hosting list, which catches a self-hosted VPN on a VPS. It is off by default because it also refuses Apple iCloud Private Relay and Cloudflare WARP — both egress from datacenter ranges, both on by default for a lot of ordinary mobile traffic. `blocked_clicks{list}` is labelled per list precisely so you can see what turning it on would cost before turning it on. `vpnBlocklist.allow` is the escape hatch and beats both lists.

**The `X-Real-IP` caveat applies here too, and matters more.** Anything that reaches `backend:8080` directly bypasses the blocklist by sending its own header, exactly as it bypasses the throttle. Caddy replaces the header on every backend route, so this is only reachable if the backend port is exposed — but the bypass is now security-relevant rather than merely an abuse nuisance.

### Sessions (`internal/session/`)

The answer to the one thing an address-based defence cannot do. The rate limiter and the VPN blocklist both key on an address, so the whole defence reduces to "can the attacker get addresses" — and against residential proxy pools, which appear in no public list, it can. **A session is what makes clicking cost something to start.**

`Click` requires a token this server minted, in the `X-Session-Token` header. The only way to get one is `session.v1.SessionService/CreateSession`, which verifies a **Cloudflare Turnstile** token against siteverify before minting. A script that reads the proto and POSTs `Click` no longer has a complete client: it has to solve Turnstile first.

**The token is stateless.** `kernel/session` mints `base64url(expiry ‖ random id ‖ HMAC-SHA256(expiry ‖ id ‖ ip))` — 48 bytes, 64 characters. Nothing is stored, swept or replicated; verification is one HMAC. That is what keeps this compatible with a process that holds the whole game in memory and has no database to put a session table in.

**It is bound to the address that minted it**, so a token lifted off the wire is worth nothing anywhere else. The MAC covers the address without carrying it, so the token leaks nothing. A player whose address changes mid-session — a phone moving from wifi to cellular — fails verification, and the client mints again and retries: self-healing, and invisible.

The signature is checked **before** the expiry, in constant time, so a forger learns nothing about whether their token would otherwise have been in date.

**`session.enforce` is the rollout switch.** False — the shipping default — makes the interceptor decide nothing: every click passes and its verdict is counted. `click_session_checks{verdict}` then says exactly what enforcing would refuse (`missing` and `invalid`) before it refuses it, which is what lets the backend deploy ahead of the frontend. True answers `CodeUnauthenticated` (HTTP 401), and the client is expected to mint and retry rather than show the player anything.

**Where it sits in the chain:** error mapping, VPN blocklist, **session**, throttle. Outside the limiter for the same reason the blocklist is — a click refused for its session must not also spend a token, or the retry that follows the mint would come back 429 and the web app would show the throttle dialog instead. `TestSessionCheckRunsBeforeTheThrottle` pins it.

**Reads and the websocket are untouched.** A visitor loads the planet, watches it live and reads the chat without ever minting anything; a session is only ever needed to paint. `GetMap` is a cacheable GET and a per-session header on it would defeat that cache.

**Minting has its own throttle** (`session.rateLimiter`, one every 30s with 10 in hand). A mint costs a siteverify round trip to a third party, so it cannot share the click budget: unthrottled, the endpoint is a free way to spend this server's siteverify quota.

**`kernel/turnstile` fails closed on everything.** A network error, a non-2xx, a body that is not JSON, a token for another action or another hostname are all refused exactly as a forged one is. Failing open would make the check decorative — an attacker who can reach the backend can also make siteverify unreachable from it. It validates `action` and `hostname` as well as `success`, because **the sitekey is public**: without those two checks a token minted by the same widget embedded on any other page would be accepted here.

**`session.turnstile.enabled: false` mints for anyone who asks** (`open_attester`). That is how a local backend runs without a widget and a secret, and it still exercises the whole click path — the token is bound and expires. It is never the production choice, and the server warns at boot when it is on.

**Two secrets, neither in git.** `session.secret` signs the tokens; anyone holding it can mint one the API accepts. `session.turnstile.secret` is the widget's secret half. Both come from the environment via `deploy/vps/docker-compose.yaml`, as `chat.service.tagSalt` does. An empty `session.secret` generates one at boot and warns — which invalidates every session in flight on each restart, costing every player one extra round trip.

**What it does not stop:** a person who solves Turnstile in a real browser and then runs a userscript. They hold a genuine session, and nothing here distinguishes them from a player. This raises the floor from "twenty lines of Python" to "drive a real browser"; the signal that survives that is behavioural — timing regularity and tile-id structure over a session — which is what the session id on the context exists to be keyed on. Nothing in production reads it yet, so `ctxutil.GetSessionID` sits behind the `testing` tag (see [Testing](#testing)) until something does.


### Shared interceptors

`kernel/connectutil` holds the two interceptors both contexts need, because the policy is the same whatever the procedure is — only the procedure names and the wording of the refusal differ, and those are arguments:

- `NewRateLimitInterceptor(limiter, refusal, procedures...)` — a `kernel/ratelimit` bucket keyed on the context IP, answering `CodeResourceExhausted` (429)
- `NewIPBlockInterceptor(blocklist, refusal, onBlocked, procedures...)` — an `ipblock.Blocklist` lookup answering `CodePermissionDenied` (403), with an optional hook the click counter hangs on
- `NewSessionInterceptor(verifier, clock, refusal, enforce, onVerdict, procedures...)` — a `kernel/session` signature check answering `CodeUnauthenticated` (401), which puts the session id on the context and, with `enforce` false, counts without refusing

Each context keeps a thin named constructor over these — `planetv1controller.NewRateLimitInterceptor` and `NewVPNBlockInterceptor`, `chatv1controller.NewRateLimitInterceptor` and `NewBlocklistInterceptor` — which is where the procedure list, the refusal wording and the metric live. **A context names its own policy; neither reimplements the mechanism.**

Both chains order them the same way: error mapping outermost, then the blocklist, then the limiter. **The blocklist has to sit outside the limiter** — a refused address must not also spend a token, or its next call would come back 429 and the client would report the wrong reason. `TestVPNBlockRunsBeforeTheThrottle` pins that for clicks.

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

Shared infrastructure: `cfgutil` (YAML + env config via koanf), `httpserver` (middleware, formats), `logging`, `prom` (Prometheus), `xtime`, `ctxutil`, `ratelimit`, `ipblock`, `atomicfile`, `wspublisher`.

Two of these are here because both bounded contexts need them and neither should depend on the other:

- `wspublisher` — the WebSocket fanout, generic over its payload and its encoder. It knows nothing about what it carries, so the payload type and the wire encoding stay with the context that owns them.
- `atomicfile` — temp file, fsync, rename, fsync of the directory. Written for the tile snapshot; the chat log's retention rewrites need the same guarantee, and duplicating 80 lines of carefully-written fsync/rename code is how the two drift apart. Covered by the existing snapshot tests.

`session` mints and verifies the click token — see [Sessions](#sessions-internalsession). `turnstile` is the siteverify client it is fed by; both are in the kernel because the session context mints with them and the clicks context verifies with them, and neither context may depend on the other.

`ipblock` is the VPN prefix set — see [VPN blocklist](#vpn-blocklist). `ratelimit` is a keyed token bucket held in this process, like the tile map it protects — with one API instance, a shared counter would buy nothing. Its `Run` loop periodically forgets the buckets that have refilled to capacity, which is free: such a bucket holds exactly what a freshly created one would, and without it the map would keep an entry per address that ever clicked.

`ipscope` decides what a bucket is keyed on, and every throttle goes through it. Over IPv4 that is the address; over IPv6 it is the surrounding **/64**, because the smallest allocation a subscriber receives is a /64 and most receive far more — a bucket per v6 address is one the same line walks out of by picking its next address, turning one home connection into thousands of callers with a throttle each. The session token binds to the same unit, so the address a token is valid for and the address that spends a budget cannot diverge. Blocking deliberately does **not** use it: the VPN and datacenter lists are precise prefixes already, and widening a hit to the surrounding /64 would refuse neighbours who are not on them.

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
- `session.enabled` — off registers nothing, so `session.v1.SessionService/` 404s and clicks are judged on address alone
- `session.enforce` — off counts what enforcing would refuse without refusing it; the mode to deploy in
- `session.secret` — signs the tokens; **empty generates one at boot**, invalidating every session in flight on each restart
- `session.ttl` — how long a minted token is accepted (default 1h)
- `session.rateLimiter.*` — the per-IP `CreateSession` throttle, same shape as `rateLimiter`
- `session.turnstile.enabled` — off mints for anyone who asks, which is how a local backend runs without a widget
- `session.turnstile.secret` — the widget's secret half, from the environment
- `session.turnstile.hostnames` — the frontend origins siteverify must report; **empty refuses every token** rather than accepting any, and a production value must not include `localhost`
- `session.turnstile.action` — must match the widget's `data-action` (default `session`)
- `chat.enabled` — the kill switch; off means the routes are never registered
- `chat.storage.logPath` — the JSONL message log; **empty keeps chat entirely in memory**
- `chat.storage.historySize`, `chat.storage.retention`, `chat.storage.flushInterval`, `chat.storage.pruneInterval`, `chat.storage.subscriberBuffer`
- `chat.service.tagSalt` — salts the per-sender tag; **empty regenerates one at boot**, changing every tag on restart
- `chat.service.maxTextLength`, `chat.service.maxNameLength` — bounds in runes (280, 24)
- `chat.rateLimiter.*` — the per-IP `SendMessage` throttle, same shape as `rateLimiter`
- `chat.blockedIPs` — prefixes refused every chat RPC, parsed by `kernel/ipblock` exactly as `vpnBlocklist.allow` is

### Protobuf

API contracts live in the monorepo-shared [`/proto`](../../proto) (also used by the frontend), one package per bounded context: [`planet/v1/planet.proto`](../../proto/planet/v1/planet.proto) and [`chat/v1/chat.proto`](../../proto/chat/v1/chat.proto). Generated code goes to `generated/proto/`. Use `make proto` to regenerate after editing `.proto` files (requires the `buf` CLI, plus `protoc-gen-go` and `protoc-gen-connect-go` on `PATH`).

The proto package is the **only** version number: Connect derives each route from it, and the controller and connect package names follow — `planet.v1` gives `planetv1controller` and `planetv1connect`, `chat.v1` gives `chatv1controller` and `chatv1connect`. There is no gRPC here — Connect serves the service definitions over ordinary HTTP/1.1 POSTs (and h2c, for clients that want it).

### Testing

Unit tests only, using `testify`. There are no integration tests and no Docker dependency — `make test` runs everything from a clean checkout.

**Tests build with `-tags testing`, so use `make test` rather than a bare `go test ./...`.** Anything else that loads test files needs the tag too: `go vet -tags testing ./...`, and an editor's language server (`gopls` `buildFlags: ["-tags=testing"]`, or `go.buildTags` in VS Code), which otherwise reports the helpers as undefined. A helper that more than one package needs cannot live in a `_test.go` file, so it lives in an ordinary `.go` file carrying `//go:build testing`. The tag, not a filename convention, is what keeps such a helper out of the production binary — and what lets `make deadcode` tell a helper apart from production code. `ctxutil.GetSessionID` is the one that exists today.

**`make deadcode` fails on any unreachable function**, in two passes, because "is this reachable?" has two different right answers depending on whether test code counts as a caller. The first pass excludes tests and tagged files, so **production code whose only caller is a test is reported as dead** — the case a plain `deadcode -test` forgives. The second pass includes both but keeps only findings inside tagged files, so an unused shared helper is reported too. `deadcode` is fetched at a pinned version by the target, so there is nothing to install.
