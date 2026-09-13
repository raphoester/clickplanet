# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests — no Docker, no database, nothing to start first
make test
# or: go test -tags testing ./... | grep -v 'no test files'

# Run a single test
go test -tags testing ./internal/planet/internal/clicks/usecases/click/... -run TestName

# Run the concurrency-sensitive tests under the race detector
go test -tags testing ./... -race

# Fail on any unreachable function, production or test helper
make deadcode

# Lint. Needs golangci-lint on PATH at the pinned version: make setup-tools
make lint

# gofumpt every Go file and run go mod tidy — rewrites in place
make tidy

# Fail if `make tidy` would change anything, go.mod included
make check-format

# Run API server locally
go run ./cmd/api -config cmd/api/example.yaml

# Generate protobuf code (requires buf CLI)
make proto

# Refresh the embedded tile coordinates blob from the shared /map, then commit it
make map

# Refresh the vendored VPN/datacenter ranges from upstream, then commit them
make vpn-lists

# Build Docker image
make dBuild
```

## Architecture

This is a Go backend for a collaborative map-clicking game. It follows **hexagonal architecture (ports & adapters)**.

### Three bounded contexts, one process

- **`internal/planet/`** — the tile game: clicks, ownership, the map, the update stream.
- **`internal/chat/`** — the live chat: messages, identity, retention.
- **`internal/session/`** — the mint: what a caller has to prove before it may click.

They share the process, the transport and the country list, and **nothing else**. None imports another; each owns its own domain types, its own proto package, its own storage adapter (where it has one) and its own edge.

Bonus boxes live inside `internal/planet/` rather than beside it: what they grant
is click allowance, and what carries them is the planet stream. A context of
their own would have to import both.

**`internal/antibot/` is a domain library, not a fourth context.** It has no proto
package, no adapters and no `module.go`, and it cannot be wired without a caller
composing it — the clicks edge does, the way it gates on `session`. It is not a
shared package because "is this caller a bot" is the business this game is in,
while `shared` is for things that would read the same in any other program.
`planet` → `antibot` is the only module-to-module import in the backend.

#### A module publishes its root package and hides the rest

Every module's interior lives behind **its own `internal/`** — `internal/planet/internal/clicks`,
`internal/chat/internal/adapters/…`, `internal/antibot/internal/jury`. Go's own
rule does the enforcing: such a package is importable only from the tree rooted
at the parent of that `internal`, so `chat` importing `planet/internal/clicks`
**does not compile**. There is no linter to run, no allowlist to maintain and
nothing to keep in sync.

So the whole of what one module may use of another is **what sits in the other's
root package**:

| module | its public API |
|---|---|
| `planet` | `Config`, `NewModule` |
| `chat` | `Config`, `NewModule` |
| `session` | `Config`, `NewModule` |
| `antibot` | `Config`, `Observer`, `Guard`, `New`, `Description`, `Click`, `Report` |

That holds for `cmd/api` too: the composition root lists modules and cannot
reach a domain type, a storage adapter or a controller even if it wanted to. A
module's `Config` may carry a field whose *type* is internal (`planet.Config.TilesStorage`
is `memory_tile_storage.Config`) — koanf fills it by reflection and a caller can
still set its fields, it just cannot name the type. That is the right amount of
access: the settings are published because they are in the file, and the code
that reads them is not.

**Adding a package inside a module is therefore free, and taking one out of
`internal/` is a deliberate act** that shows up in review as exactly one moved
directory.

### The composite layer

Each context wires **itself**, in a `module.go` at its root (`internal/planet/module.go`, `internal/chat/module.go`, `internal/session/module.go`). That file is the context's manifest: its `Config`, whether it is on, and its DI sequence. **A module takes its config and nothing else, and builds every object it needs itself** — there is no `Deps` struct and nothing is handed down from `main`. A module is a `cpbootstrap.Module` — a name, an `Enabled` flag and a DI sequence — and the sequence is handed a `cpbootstrap.Props` carrying registrars and nothing else:

- `props.RPC.Mount(build, interceptors...)` — the module hands over what *builds* the handler, plus the interceptors it wants. `cpbootstrap` builds it, so it can put its own interceptor outside every module's — see [The error net](#the-error-net)
- `props.Runners.Add(name, run)` — a goroutine, given the process-lifetime context
- `props.Closers.Add(name, close)` — a cleanup, run in reverse registration order
- `props.Logger`, `props.Metrics`
- `props.Server` — the bind address and the stream heartbeat, **the only config a module reads that is not its own**. It is the transport every module answers over, so it belongs to the layer that owns the server rather than to any context.

A module never sees the router, the signal handler or another module's objects. **There is no mutable app object and nothing to leave a half-built dependency on**: `internal/shared/cpbootstrap` builds every module in order, then serves.

**`cmd/api/main.go` is the composition root, and it is the only one** — there is no `internal/app`, because a package whose whole job is to be called once by `main` was a level of indirection and nothing else. It does two things: load the config, and list the modules.

**The aggregation is the whole of it — a flat slice, no branches, no dependencies threaded through:**

```go
return []bootstrap.Module{
	session.NewModule(config.Session),
	planet.NewModule(config.Planet),
	chat.NewModule(config.Chat),
}
```

**Every module is always listed; each reads its own switch.** `NewModule` sets `Enabled` from the module's own config and `cpbootstrap` skips the ones that are off, so turning chat off is a config change and never an edit here. A disabled module is never built, so its routes are **absent** rather than present and refusing — `/chat.v1.ChatService/` 404s, which is the contract chat and sessions already had.

#### What two contexts need, without either handing it to the other

`main` builds no objects at all, so a thing two contexts need is **a config block they both declare**, and each builds its own instance from it.

- **`shared/cpsession.Config`** is the `session:` block, and it is shared because two contexts read it: `session` mints with it, `planet` verifies with it. Each calls `cpsession.NewSigner(config)` itself. The same secret and TTL produce the same MAC, so the two signers agree by construction and there is no object to pass — `TestBothContextsReadTheSameSessionBlock` pins that they read one block, and `TestTwoSignersOverOneConfigAgree` pins that one block means one key. Neither module imports the other, and **the planet context knows nothing about Turnstile** — the siteverify client lives at `session/internal/turnstile`, so it *cannot* reach it, and swapping the attester changes one line in `internal/session/module.go`.
- **`shared/cpcountries`** is the ISO list. It is stateless and hardcoded, so each module just calls `cpcountries.New()`, the way it calls `cptime.SystemClock{}`. It sits in `shared` and not under `planet/internal/adapters/` for exactly the reason that layer exists: neither context may depend on the other — and now could not, since that directory is unreachable from chat.

**This is why `session.secret` is now required** rather than invented at boot — see [Sessions](#sessions-internalsession).

Adding a context that callers talk to means a `proto/<name>/v1`, an `internal/<name>/` with a `module.go`, and one line in the slice. Connect derives the route from the proto package, so there is no prefix to allocate and no router to edit.

**`cmd/api`** is the only binary: `main.go` is the composition root, about 85 lines of config and a module list, and `cpbootstrap` is the rest.

It runs as a **single self-contained container with no dependencies**: the tile map lives in process and is persisted to a local snapshot file. There is no database, no cache, and no second process.

### Inside the planet module: parts, not layers

**There is no `domain` package, deliberately.** "Domain" names a layer, and a
layer is the one thing every part of a module has in common — so a package
called that collects whatever does not fit elsewhere and grows until nothing in
it has a reason to sit beside anything else. It had shrunk to a sentinel and two
structs, which is what a shell looks like.

What sits under `internal/planet/internal/` is **one package per part of the
game**, each named for what it is:

- **`clicks/`** — the tile game: what a click is worth, what the map looks like,
  and what changes when somebody takes a tile.
- **`bonuses/`** — next, and the reason this is worth doing now. A part named
  for itself is a directory to add; under a `domain` package it would have been
  a subdirectory of a word that describes neither.

A part's **root holds its vocabulary** — the sentinels and the types that cross
between a use case and an adapter, so belong to neither. `clicks` holds
its caller-error sentinels, `TileUpdate` and `DenseBatch`, and nothing else: no ports,
no service, no logic. **Its rules live one level down, one package per use case.**

The adapters stay where they are, under `internal/adapters/`, because an adapter
is answerable to the transport or the store and not to one part — `planetv1controller`
already serves both parts over one Connect service.

#### Use cases (`internal/planet/internal/clicks/usecases/`)

**One package per procedure, and each declares its own ports.** The service the
edge serves has five procedures, so there are five packages, each exporting
`New` and a `UseCase` with one `Execute`:

| package | what it does | what it needs |
|---|---|---|
| `click` | validates the country and the tile, then writes | `TilesChecker`, `TileStorage`, `CountryChecker` |
| `get_map` | a range of the map as one dense batch | `MaxIndexReader`, `DenseMapReader` |
| `map_density` | how many tiles there are | `MaxIndexReader` |
| `get_budget` | a caller's allowance, unspent | `ClickBudgetReader` |
| `listen_for_events` | one client's live feed, heartbeat included | `UpdatesSubscriber` |

**The interfaces in that last column are declared by the package that calls
them**, not gathered in a `gateways.go` every use case imports. That is the
whole point of the split: a shared port file makes every dependency everyone's,
so `Click` ends up compiling against the map reader it never calls and a change
to one procedure's needs is a change to the file all five read. Here, adding a
dependency to `get_map` is invisible to the other four. The adapters are
unchanged — `memory_tile_storage` happens to satisfy three of these ports at
once, which is why `module.go` hands it over three times.

**`click` is the only one that writes**, and the only one with an interface of
its own (`IUseCase`), because `click/prom_click` decorates it — the counting is
a wrapper rather than a line inside the rule, so a process that does not want it
leaves it out and the rule does not change.

The ports are written in the vocabulary the `clicks` root holds, and nothing
travels between a use case and an adapter that is not declared in one of the
two.

`Geography` is the other thing the `clicks` root holds, beside the sentinels and `TileUpdate`: the shape of the map, in `geography.go`. It is a model rather than a port — `geodesic_map` builds one and hands it over. See [Map geography](#map-geography).

### Adapters

**Primary (input):**
- `internal/adapters/primary/http/planetv1controller/` — the API. `ClickService` implements the generated `planetv1connect.ClickServiceHandler` and nothing else; it never sees an `http.ResponseWriter`, which is the point of serving the contract with Connect rather than by hand. **Each procedure is its own package** — `click_handler`, `get_map_handler`, `map_density_handler`, `get_budget_handler`, `listen_for_events_handler` — holding the one use case it calls and declaring the one port it needs. Each owns the mapping both ways, and each is tested on that mapping alone.

`ClickService` is those five embedded, and **nothing else**: no fields of its own, no methods of its own, and **no constructor** — it is a bag of handlers, so the DI sequence that already builds them writes the literal. It has no test either. An aggregation's only claim is that it carries all five procedures, and `var _ planetv1connect.ClickServiceHandler = ClickService{}` is that claim, checked at compile time. A test that served it and called a procedure would be re-testing the handler package that procedure lives in.

**A caller error becomes a Connect code in the handler, not centrally.** `click_handler` turns `clicks.ErrUnknownCountry` and `clicks.ErrTileOutOfRange` into `CodeInvalidArgument` and `clicks.ErrThrottled` into `CodeResourceExhausted`; `get_map_handler` turns `clicks.ErrInvalidTileRange` into `CodeInvalidArgument`. The sentinel these replaced was `ErrInvalidArgument`, which was a status code wearing a domain hat: it told a reader nothing a use case could act on, and it made every caller error in the game the same one. There is **no error interceptor in this package** — see [The error net](#the-error-net).
- the tile stream, as `ClickService.ListenForEvents` — a Connect server-streaming RPC like any other procedure on the service. See [The live streams](#the-live-streams).

**There is one server, one mux, and no version prefix.** Connect names each service's path from its proto package — `/planet.v1.ClickService/` and `/chat.v1.ChatService/` — so nothing is mounted under a prefix of ours. The three services and `/metrics` are the only things on the router. Nothing here needs a connection-level demultiplexer such as `cmux`; that is for running a real gRPC server, which owns its own HTTP/2 handler, beside a REST one.

`cpbootstrap` sets `http.Protocols` with both HTTP/1.1 and unencrypted HTTP/2, because the generated handler also speaks gRPC and gRPC-Web and those need HTTP/2. Browsers reach the same routes over HTTP/1.1. Verified: HTTP/1.1 and h2c both answer on the same port.

### The live streams

Both live feeds are served **two ways at once**, and that is a transition, not a design:

- `ClickService.ListenForEvents` → `stream PlanetEvent`, and `ChatService.ListenForEvents` → `stream ChatEvent`. Ordinary Connect server-streaming RPCs, on the same routes and the same port as everything else.

**One stream per API, and an envelope rather than a bare payload.** A `PlanetEvent` is a `oneof` of `tile_update` and `heartbeat`; `ChatEvent` is a `oneof` of `message` and `heartbeat`. **A new kind of live event is a new case in that `oneof`, never a second stream** — one connection per client, one route to configure, and a client that does not know a case reads an unset `oneof` and skips it instead of breaking. That is what the bare `TileUpdate` frame could not do, on the websocket or off it.

**`heartbeat` is not decoration.** Cloudflare cuts a silent response at **~125s with a 524** — measured against production three times, exactly 125.1s. The websocket never hit this because Cloudflare keeps those open; a chunked HTTP response is not so lucky. A quiet chat is the normal case, and a quiet planet happens, so both streams send a heartbeat every `httpServer.streamHeartbeat` (30s by default, and it **must** stay well under 125s). Without it a silent stream dies and reconnects forever, losing whatever was published in each gap.
These replaced a pair of websockets on `/ws/listen` and `/ws/chat`, broadcast by a `wspublisher` fanout. **Nothing here speaks websocket any more** — no upgrade route, no second mux, no `coder/websocket` dependency.

Each handler calls the storage's `Subscribe(ctx)` **per call**, and the request context is what unsubscribes — it is cancelled however the stream ends, so a disconnect needs no `CloseRead` equivalent. Both storages already handed every subscriber its own buffered channel and dropped rather than blocked for a slow one, so one subscription per connected client is what they were built for; `subscriberBuffer` now bounds a client rather than the single fanout.

**The streaming RPCs are not wrapped by any interceptor except error mapping**, because every other one is a `connect.UnaryInterceptorFunc` and streams skip those by construction. Reads and the live feeds are therefore untouched by the throttle, the VPN blocklist and the session check, exactly as they were when they were websockets. A policy that ever has to reach a stream must be written as a full `connect.Interceptor`.

### The map load

`GetMap` is an ordinary RPC, but marked `idempotency_level = NO_SIDE_EFFECTS` in the proto, so Connect sends it as an **HTTP GET** and the handler sets `Cache-Control: public, max-age=5` on the response. A burst of visitors can therefore share one origin response; `ListenForEvents` carries everything that happens after a chunk was built, so a client starting from a slightly old map converges anyway. `MapDensity` is marked the same way.

The response never repeats a tile id. `GetMapResponse` carries `start_tile_id`, the interned `codes` table, and `tiles` — a `bytes` field holding two bytes per tile, little endian, indexing into `codes`. Tile ids are implicit in the position, which is what makes it far smaller than the deprecated `map<uint32, string>`: **516 KB against 3.6 MB** for a full 257,948-tile map.

`memory_tile_storage.StateBatchDense` builds it. The interned ids are copied out exactly as stored and the table travels with them, so nothing is translated on the way out and the client needs no shared country list. Protobuf does all the framing — there is no hand-rolled magic or length-prefixing on either side, and therefore no encoder and decoder that have to be edited together.

**Secondary (output):**
- `internal/adapters/secondary/memory_tile_storage/` — the tile map. A preallocated `[]uint16` indexed by tile id, with country codes interned into a side table (2 bytes per tile — ~2 MB for a 1M-tile map). Fans updates out in process and persists to a local snapshot file.
- `internal/adapters/secondary/in_memory_tile_checker/` — validates tile IDs
- country codes are validated by `shared/cpcountries`, which chat shares — see [The composite layer](#the-composite-layer)

Beyond the `click.TileStorage` port, `memory_tile_storage` also exposes `Subscribe(ctx) (<-chan clicks.Change, error)`, one call per open stream. A `Change` is a tile update or a bomb blast, on one channel so the two keep their order — see [What a bomb does](#what-a-bomb-does).

### Key Flow

```
POST /session.v1.SessionService/CreateSession
  → SessionService
  → session_service (attests, then mints)
  → turnstile_attester → Cloudflare siteverify
  → shared/cpsession.Signer.Mint [HMAC over expiry+id+IP; nothing stored]

POST /planet.v1.ClickService/Click   [X-Session-Token: <the minted token>]
  → [cpbootstrap: error net], CacheInterceptor, VPNBlockInterceptor, SessionInterceptor
  → ClickService → click_handler
  → throttle_click  (spends a token, or refuses)
  → antibot_click   (judges; a flagged caller is answered OK and dropped)
  → prom_click      (counts)
  → clicks/usecases/click (validates tile ID + country)
  → MemoryTileStorage.Set() [writes the tile, fans the update out in process]
  → every subscriber: one per open ListenForEvents stream
```

`Set` is a no-op when the tile already holds that value — no write, no update published.

```
POST /chat.v1.ChatService/SendMessage
  → [cpbootstrap: error net], BlocklistInterceptor, then RateLimitInterceptor (both shared/cpconnect)
  → ChatService
  → chat_service (sanitizes, stamps id/time/tag)
  → MemoryChatStorage.Append() [appends to the JSONL log, then fans out]
  → every subscriber: one per open ListenForEvents stream
```

A failed log write fails the whole post: the log is the audit trail, so a message nobody can account for later is not one that gets broadcast.

### Chat (`internal/chat/`)

Chat is a separate bounded context, not a feature of the tile game: it shares the process, the transport and the country list, and has its own proto package, domain, storage and edge. Nothing under `internal/chat/` imports `internal/planet/`, and the reverse holds too — and since each module's interior sits behind its own `internal/`, neither now can.

**Off by default.** With `chat.enabled` false nothing is registered, so `/chat.v1.ChatService/` answers 404 — the unauthenticated public write endpoint does not exist at all rather than existing and erroring.

**Identity without accounts.** A client picks its own display name and sends a UUID it persists locally. **Neither is trusted for anything** — anyone can post with any name. What a sender cannot forge is `author_tag`: a salted hash of their IP, 6 hex characters, so two people using the same name still look different and a mute has a key that means something. The salt is `chat.service.tagSalt`; left empty it is regenerated at boot, which changes everyone's tag on restart, and the server warns about it.

**Abuse controls live at the edge**, in two interceptors — ahead of decoding the message and well ahead of validating it, so a flood of malformed messages costs a sender exactly what a flood of well-formed ones does. `chat.blockedIPs` cuts an address off from every chat RPC; `chat.rateLimiter` throttles `SendMessage` alone (`GetHistory` is one read on join, and limiting it would punish a page load). **Both are `shared/cpconnect`'s**, the same ones the click chain uses — see [Shared interceptors](#shared-interceptors). The domain then bounds the message in **runes** (280) and the name (24), validates UTF-8, and **strips control characters** — a newline would otherwise let a sender forge a line in the JSONL log.

Refusal reasons are logged, never returned: a sender learns *that* they were refused, not which check tripped. **The stored text is raw — the frontend must escape it.**

The **vendored VPN lists** do not cover chat: `NewVPNBlockInterceptor` wraps `Click` alone. Chat's blocklist is the same `*cpipblock.Blocklist` type, built by `cpipblock.NewDenyList` from config prefixes instead of vendored data — so entries are CIDRs and a `/24` is one line rather than 256. Extending the vendored lists to chat is therefore a wiring change (build the list in `describeModules` and hand it to both modules, the way `shared/cpcountries` already is), not a second list to write.

**The log is an append-only JSONL file**, not the tile snapshot's whole-state codec: different shape, different write pattern. One line per message with `at`, `id`, `name`, `tag`, `authorId`, `country`, `ip`, `userAgent`, `text`. It is fsynced every `flushInterval` rather than per message (a hard kill loses at most that window — the same bargain the snapshot makes), pruned hourly past `retention`, and its tail repopulates the in-memory history at boot so a restart does not blank the chat. Corrupt lines are skipped and reported, never fatal.

That log holds **personal data** — IPs next to user-authored text — so the retention window is a policy decision rather than a cache size. It lives on the `tile_state` volume, which the droplet's weekly disk backup already covers.

**Chat has its own stream**, `ChatService.ListenForEvents` — see [The live streams](#the-live-streams). It replaced a `/ws/chat` websocket that had to be kept apart from the tile one because frames carried a bare protobuf message with no type tag: a second payload on either socket would have been indistinguishable from the first. The `oneof` envelope is exactly what removes that constraint.

**Sending is an RPC, not a read on the socket**: both publishers lean on `CloseRead` for instant disconnect detection, and the RPC path already has the middleware stack and the interceptors.

`GetHistory` is marked `NO_SIDE_EFFECTS`, so Connect sends it as a GET — but it answers `Cache-Control: no-store`, the opposite of `GetMap`. A client fetches it once on join to seed what the stream then keeps up to date, so a cached answer would show a joiner a chat missing the last few minutes.

### Rate limiting

`throttle_click` throttles clicks, per source IP, from a `shared/cpratelimit` token bucket — 1 click/s with a burst of 10 by default (`rateLimiter.*`). `MapDensity` and `GetMap` are cacheable reads a proxy in front absorbs; limiting them would punish a page load rather than a bot. A refused click answers `CodeResourceExhausted`, i.e. HTTP 429, and never reaches the map.

**It is a decorator over the click use case, not an interceptor over the procedure.** Two things fall out of that. The allowance comes back as a return value (`click.Out`) instead of being left on the context for a handler to find, which is what `cpctx.AddRateBudgetToContext` existed for and why it is gone. And "a click refused for its address or its session must not also spend a token" stops being a rule about the order of a list and becomes a property of the shape: every interceptor is outside the whole click chain by construction. `MapDensity` and `GetMap` are untouched for free, being other procedures entirely — under an interceptor that took a procedure list.

#### Saying what is left

The web app shows the player how many clicks they have in hand, and **the server is the only thing that knows**. A client running its own copy of the bucket would drift within seconds — it cannot see the clicks the same address makes from another tab, and its idea of when a click was spent is a round trip out of date.

Polling for it would be worse, so nothing polls. `Limiter.Take` returns the bucket's state alongside its verdict, and the state carries the **policy** (`Capacity`, `PerSecond`) as well as the reading: given both, a client replays the same refill arithmetic between two answers and is exact without asking. The pip count and the fill rate on screen are therefore the server's burst and refill rate — **changing `rateLimiter.*` changes the display with no frontend release.**

That reading travels two ways, because a refused call has no response message to put it in:

- an allowed call carries it on `click.Out`, and `click_handler` puts it in `ClickResponse.budget`.
- a refused one carries it on the same `click.Out`, beside `clicks.ErrThrottled`, and `click_handler` attaches it as a **connect error detail** — a refusal has no response message to put it in.

Either way the decorator decides the policy and the handler decides how to say it. `clickbudget.Encode` is the one place that shape is agreed, because two procedures answer with a `ClickBudget`: the click that just spent a token, and `GetBudget`.

`ClickService.GetBudget` covers the cold start — a client that has just loaded and has no click to learn from. It reads through `Limiter.Peek`, which spends nothing and, for an address that never clicked, **creates no bucket**: reading an allowance must not be a way to make the limiter remember a caller. It is deliberately not `NO_SIDE_EFFECTS`, so it is a POST no cache will serve a stale answer to; every click re-anchors the client afterwards, so it is asked once per page load.

**This tells a scripted clicker exactly when to fire**, which is a real cost against [Anti-bot](#anti-bot-internalantibot). It is a small one — a script can already infer the same schedule by counting its own 429s — and it is paid to stop honest players being refused with no warning.

The bucket key is whatever `IPReaderMiddleware` put on the context: `X-Real-IP` if present, otherwise the peer address. **The reverse proxy must set that header itself** — `deploy/vps/Caddyfile` does, with `header_up X-Real-IP {client_ip}` on every backend route. Merely forwarding it would let a client send its own and buy a fresh bucket per request. The fallback is the peer address rather than a constant precisely so a missing header degrades to per-connection buckets instead of rate limiting the whole game as one player.

#### A big country pays more per click (`clicks/toll`)

A click costs more tokens the more of the map its country holds. `toll.steps` is
a table of `{share, cost}`: from `share` of **every tile on the map**, a click for
that country costs `cost` tokens. No steps prices every click at one.
A cost may be a fraction of a token (production runs x1.25 from 25%, x1.5 from
50%, x2 from 70%), which is why `ClickBudget.cost` is a double. It moved to new
field numbers rather than changing type in place: a client built against the old
`uint32` reads a cost of zero and simply says nothing about price.

- **The price is taken at the click, from the country clicked for.** A slower
  refill for a big country would have been read off whatever country the caller
  played last, so a player could bank tokens on a small one and spend them on a big one.
- **The share is of the whole map, not of owned tiles**, so early in a game nobody pays more.
- **`memory_tile_storage` keeps a tile count per country**, moved by `set` and
  `Clear` and rebuilt from the snapshot, so `Share` is one read and no scan.
- **The budget goes out already divided by the cost** (`toll.Of`): ten tokens at a
  cost of 2 are five clicks refilling at 0.5/s. The meter narrows off the server's
  numbers the way a bonus widens it, and `ClickBudget` also carries `cost`,
  `share` and the next step so the client can say why.
- **Bonuses compose with it.** A triple bonus multiplies the bucket and the price
  divides it, so it is still worth three times the clicks. A spread is one click at
  the country's price. A bomb is not throttled, and lowers the share of whoever it hits.
- **A cost above `rateLimiter.burst` refuses the boot**: no bucket could ever pay it.

`GetBudget` takes the country, because the price depends on it. Known risk, not
handled yet: a country sitting on a step can cross it back and forth click to click.

Chat and sessions each have **their own limiter instance** with their own budget, because what each call costs has nothing to do with what a click costs:

- `chat.rateLimiter` — one message every 3s, five in hand. A message fans out to every connected client and lands in a log everyone will read.
- `session.rateLimiter` — one mint every 30s, ten in hand. A mint costs a siteverify round trip to a third party, so an unthrottled `CreateSession` is a free way to spend this server's Turnstile quota.

### VPN blocklist

`NewVPNBlockInterceptor` refuses **`Click` only**, with `CodePermissionDenied` (HTTP 403), when the source address falls in a vendored VPN range. It sits **outside the rate limiter** in the interceptor chain, deliberately: a refused address must not also spend a token, or the next click would come back 429 and the web app would show the throttle dialog instead of the VPN one.

**It exists because of the rate limiter, not instead of it.** The bucket is keyed on an address, and a commercial VPN is the cheapest way to get a fresh one; refusing those addresses is what makes the bucket hold. It raises the floor rather than closing the door — residential proxies appear in no public list, and nothing here stops one. **That gap is what [Sessions](#sessions-internalsession) closes**, by requiring something an address cannot buy; chasing list completeness instead is a treadmill.

Reads and the streams are untouched. A VPN user still loads the planet and follows it live; they cannot paint. That is also what keeps a false positive readable: the page works and says why, instead of failing to load.

**The ranges are vendored and embedded**, from [X4BNet/lists_vpn](https://github.com/X4BNet/lists_vpn) (MIT, rebuilt daily from ASN ownership), in `internal/shared/cpipblock/cpdata`. Not fetched at boot: `cmd/api` is a self-contained container with no startup dependencies, and a boot that can fail because GitHub is down is a worse trade than a list that ages between deploys — the Cloudflare ranges in `deploy/vps/Caddyfile` are maintained the same way. Refresh with `make vpn-lists` and commit; the tests assert the lists still parse and are not truncated.

`cpipblock` holds them as sorted, merged `[lo, hi]` ranges of 16-byte addresses and binary-searches them — ~63k prefixes fold to far fewer ranges, about 2 MB resident and well under 100 ns per lookup. **IPv4 and IPv6 live in separate slices.** They cannot share one: an IPv4 address in its v4-mapped form sits inside `::ffff:0:0/96`, so a single ordering would let a v6 prefix as short as `::/16` silently swallow every IPv4 address on the internet.

`vpnBlocklist.includeDatacenters` adds the much broader hosting list, which catches a self-hosted VPN on a VPS. It is off by default because it also refuses Apple iCloud Private Relay and Cloudflare WARP — both egress from datacenter ranges, both on by default for a lot of ordinary mobile traffic. `blocked_clicks{list}` is labelled per list precisely so you can see what turning it on would cost before turning it on. `vpnBlocklist.allow` is the escape hatch and beats both lists.

**The `X-Real-IP` caveat applies here too, and matters more.** Anything that reaches `backend:8080` directly bypasses the blocklist by sending its own header, exactly as it bypasses the throttle. Caddy replaces the header on every backend route, so this is only reachable if the backend port is exposed — but the bypass is now security-relevant rather than merely an abuse nuisance.

### Sessions (`internal/session/`)

The answer to the one thing an address-based defence cannot do. The rate limiter and the VPN blocklist both key on an address, so the whole defence reduces to "can the attacker get addresses" — and against residential proxy pools, which appear in no public list, it can. **A session is what makes clicking cost something to start.**

`Click` requires a token this server minted, in the `X-Session-Token` header. The only way to get one is `session.v1.SessionService/CreateSession`, which verifies a **Cloudflare Turnstile** token against siteverify before minting. A script that reads the proto and POSTs `Click` no longer has a complete client: it has to solve Turnstile first.

**The token is stateless.** `shared/cpsession` mints `base64url(expiry ‖ random id ‖ HMAC-SHA256(expiry ‖ id ‖ ip))` — 48 bytes, 64 characters. Nothing is stored, swept or replicated; verification is one HMAC. That is what keeps this compatible with a process that holds the whole game in memory and has no database to put a session table in.

**It is bound to the address that minted it**, so a token lifted off the wire is worth nothing anywhere else. The MAC covers the address without carrying it, so the token leaks nothing. A player whose address changes mid-session — a phone moving from wifi to cellular — fails verification, and the client mints again and retries: self-healing, and invisible.

The signature is checked **before** the expiry, in constant time, so a forger learns nothing about whether their token would otherwise have been in date.

**`session.enforce` is the rollout switch.** False — the shipping default — makes the interceptor decide nothing: every click passes and its verdict is counted. `click_session_checks{verdict}` then says exactly what enforcing would refuse (`missing` and `invalid`) before it refuses it, which is what lets the backend deploy ahead of the frontend. True answers `CodeUnauthenticated` (HTTP 401), and the client is expected to mint and retry rather than show the player anything.

**Where it sits in the chain:** error mapping, VPN blocklist, **session**, throttle. Outside the limiter for the same reason the blocklist is — a click refused for its session must not also spend a token, or the retry that follows the mint would come back 429 and the web app would show the throttle dialog instead. `TestSessionCheckRunsBeforeTheThrottle` pins it.

**Reads and the streams are untouched.** A visitor loads the planet, watches it live and reads the chat without ever minting anything; a session is only ever needed to paint. `GetMap` is a cacheable GET and a per-session header on it would defeat that cache.

**Minting has its own throttle** (`session.rateLimiter`, one every 30s with 10 in hand). A mint costs a siteverify round trip to a third party, so it cannot share the click budget: unthrottled, the endpoint is a free way to spend this server's siteverify quota.

**`session/internal/turnstile` fails closed on everything.** A network error, a non-2xx, a body that is not JSON, a token for another action or another hostname are all refused exactly as a forged one is. Failing open would make the check decorative — an attacker who can reach the backend can also make siteverify unreachable from it. It validates `action` and `hostname` as well as `success`, because **the sitekey is public**: without those two checks a token minted by the same widget embedded on any other page would be accepted here.

**`session.turnstile.enabled: false` mints for anyone who asks** (`open_attester`). That is how a local backend runs without a widget and a secret, and it still exercises the whole click path — the token is bound and expires. It is never the production choice, and the server warns at boot when it is on.

**Two secrets, neither in git.** `session.secret` signs the tokens; anyone holding it can mint one the API accepts. `session.turnstile.secret` is the widget's secret half. Both come from the environment via `deploy/vps/docker-compose.yaml`, as `chat.service.tagSalt` does. **An empty `session.secret` with `session.enabled` true refuses the boot**, naming the variable to set. It used to generate one and warn; that stopped being possible when the two contexts started deriving their own signer from the block instead of sharing one object — a server that invented a secret would invent a different one per context and could not verify what it had just minted. Failing at boot is also the better trade on its own: the generated key invalidated every session in flight on each restart.

**What it does not stop:** a person who solves Turnstile in a real browser and then runs a userscript. They hold a genuine session, and nothing here distinguishes them from a player. This raises the floor from "twenty lines of Python" to "drive a real browser"; the signal that survives that is behavioural — timing regularity and tile-id structure over a session — which is what the session id on the context exists to be keyed on. Nothing in production reads it yet, so `cpctx.GetSessionID` sits behind the `testing` tag (see [Testing](#testing)) until something does.


### Bonus boxes (`internal/planet/internal/clicks/bonus/`)

A question-mark box flies past the planet every so often; whoever catches it
gets one of four bonuses. Each box draws its kind from `bonus.kinds`, a weight
per kind — a kind's chance is its weight over the sum of the weights, so the
strong ones can be made rare (production runs 5 : 2 : 1 : 2):

- **`triple_clicks`** — the allowance is multiplied by `bonus.multiplier` for
  `bonus.duration`. See [What a bonus does to the bucket](#what-a-bonus-does-to-the-bucket).
- **`spread_clicks`** — every click also takes the tiles touching the one
  clicked, for `bonus.spreadDuration` instead (10s by default — it is strong). See [What a spread does to a click](#what-a-spread-does-to-a-click).
- **`bomb`** — one bomb, to be dropped within `bonus.bombDuration` (30s). It
  clears a circle of `bonus.bombRings` tile spacings around where it lands,
  whoever holds the tiles. See [What a bomb does](#what-a-bomb-does).
- **`enclose_clicks`** — a click that closes a shape of the caller's own tiles
  also takes the tiles inside it: `bonus.encloseShapes` shapes (3), each of at
  most `bonus.encloseMaxTiles` tiles (15), within `bonus.encloseDuration` (30s).
  See [What an enclose does to a click](#what-an-enclose-does-to-a-click).

Off by
default — `bonus.enabled` false offers nothing and answers `ClaimBonus` with
`CodeUnimplemented`, so the capability is absent rather than present and
refusing, the same shape chat has.

**The server picks who gets one, and that is the whole design.** A box broadcast
to everyone is caught by whichever client reacts fastest, and that is a script
every time: it reads the event off the stream and answers in twenty milliseconds
while a person is still moving the mouse. Broadcasting would make this a machine
for handing extra clicks to exactly the callers [Anti-bot](#anti-bot-internalantibot)
exists to stop.

So an offer is **addressed**: it is sent down one caller's stream, and nobody
else sees it or can claim it. Reflexes buy nothing, and what is left to do —
notice the box and click it — is the part that was meant to be the game.

**Every caller is on a schedule of their own.** A single server-wide ticker
drawing one winner made the rate each player saw `1/(interval × players)`, so
the feature got rarer the busier the game was — a box every couple of minutes at
ten players, one an hour at three hundred. What a player experiences must not
depend on how many other people are online.

That schedule is one `nextOfferAt` per caller and **one sweep for everybody**,
not a timer each: `bonus.sweepInterval` walks the map the way the antibot sweeps
already do.

**The wait is drawn uniformly from `[minInterval, maxInterval]`.** The spread is
for feel and not for defence — a script does not predict the schedule, it
watches the stream, so there is nothing here to hide from one.

**The pace answers what the player did with the last box:**

- **Missed** — the token lapsed unclaimed, which the sweep sees without the
  client saying anything — the next one comes at `missRetry`. That applies to
  **one** miss; a second in a row waits the ordinary window, or a tab that never
  catches anything would collect a box every `missRetry` forever.
- **Caught** — the next is due a window after the **bonus ends**, not after the
  catch. Timed from the catch, a second box lands on a running bonus and either
  stacks or is wasted.

**Only callers who have clicked inside `activeWithin` are offered anything.** A
tab left open overnight is not playing, and it is also what keeps the miss rule
from needing a back-off of its own. A caller whose turn comes up while they are
away **loses the slot rather than banking it** — otherwise they are handed a box
the instant they come back.

**A schedule outlives its stream by `forgetAfter`.** Without that, closing the
tab and opening it again draws a fresh wait, and a player could reload until
they got a short one.

**`maxBoostPerHour` bounds what a caller can be granted.** Nothing here is a race
any more, but catch rate is where an advantage is left: a script catches every
box it is offered where a person catches some. This makes the worst case a
number you choose rather than a function of reflexes.

**A caller is a scope, not a connection.** `Attend` is keyed on `cpipscope.Of`,
the same unit the throttle and the session token use, and holds every stream
sharing it — so twenty tabs are one entrant on one schedule, and all of them are
sent the box. The handler's `defer` is what removes it; there is no
context goroutine per connected client, because the fanout deliberately does not
pay that cost.

**The offer is the state, so there is no crypto here.** The registry already has
to remember who it offered what, so the token is 16 random bytes and the map is
the check: unknown, spent, lapsed, or offered to somebody else all fail the same
way. Unlike a session token there is nothing to verify statelessly — and nothing
to sweep either, since both paths that take the lock forget what has lapsed on
the way past.

**A claim answers `CodeNotFound` and says nothing about why.** The difference
between "no such token" and "not yours" is exactly what a script guessing tokens
would measure.

**`ClaimBonus` is session-gated**, appended to `NewSessionInterceptor`'s
procedure list: a bonus is only ever spent as clicks, and clicks need a session,
so the box that grants them should not be the one way to widen an allowance
without proving anything. It is deliberately **not** throttled — a claim is
already gated on holding a token the server addressed to you, and spending a
click token to collect a bonus is backwards.

**Two new cases on `PlanetEvent`, not a second stream** — `bonus_offered`, which
reaches one caller, and `bonus_taken`, which reaches everyone. The private reward
with a public outcome is what keeps the spectacle without the scramble. A client
too old to know either case reads an unset `oneof` and skips it, which is the
whole reason the envelope exists.

The catch is published **after** the boost lands, so a catch announced to the
planet that then failed to apply is the one lie this cannot tell.

#### What a spread does to a click

**The server picks the tiles, off its own map.** A client that named the tiles
a click spreads to could name any tiles it liked — that is why the spread waited
for [Map geography](#map-geography). The client paints the tile it clicked, as it
always has, and the neighbours reach it over the stream like anyone else's.

`claim_bonus` starts it with `bonus.Spreads.Grant(scope, until)` instead of a
boost, and answers the allowance unchanged. `bonus.Spreads` is a map of scope to
end time; each grant forgets the spreads that ran out, so it needs no sweep.

`click/spread_click` is the decorator that reads it, and **it sits right against
the rule**, inside the count, the shadow ban and the throttle:

- a click is **one click** to the throttle and to `prom_click`, however many tiles it took
- a click the shadow ban drops never reaches the rule, so it spreads nothing
- a click the rule refuses (unknown country, tile out of range) spreads nothing
- the neighbours are not reported to the antibot jury — only the clicked tile is
  a click the caller made

Each neighbour is an ordinary `Set`, so it publishes its own `TileUpdate` and a
tile already held is a no-op. One click is at most 7 updates. **A lone island
takes itself and nothing else**: `Neighbours` is empty there, and the bonus does
not pretend otherwise.

**Then it tells the planet, with `tiles_spread`**, as an enclose does with
`tiles_enclosed`: the tile clicked and the neighbours it took, after they are set,
so every client can animate why seven tiles flipped. `Registry.PublishSpread`
sends it to every caller.

A spread is one event per click, which is why a caller's bonus feed buffers 32
events rather than a handful.

**A triple clicks bonus is not an event of its own: it is `TileUpdate.boosted`.**
The limiter's `State` says whether a boost runs, `throttle_click` copies that
onto `click.In.Boosted`, and the rule writes with `SetBoosted` instead of `Set`,
so the update it publishes carries the flag. The flag rides with the change it
describes — same message, same order, no second frame per click — and a click on
a tile already held publishes nothing, so it shows nothing either. A spread
could not be done this way: its animation needs the clicked tile and its
neighbours together, which one flag per tile cannot say.

#### What a bomb does

`claim_bonus` hands the bomb over with `bonus.Bombs.Grant(scope, until)` — the
spread's counterpart, a map of scope to deadline — and answers the blast radius
on `ClaimBonusResponse.blast_radius`, so the client draws its aiming ring at the
width of what it will clear. `DropBomb` spends it through `drop_bomb`.

**The client names a point, never a tile.** The sea has no tiles, and whether an
aim is on land is the server's call: `Geography.Nearest` finds the closest tile,
and an aim further than one tile spacing from it is **in the sea** — the bomb is
spent, nothing is cleared, and the blast is still broadcast with tile 0 so every
screen draws a splash. That was a product decision: a bad aim costs the bomb.

On land the tiles are `Geography.Within(centre, radius)`: every tile within
`radius` of arc of the tile hit — a true circle, ~230 tiles inland, found by a
straight scan (~0.5ms, once per bomb). The radius is `bombRings × Geography.Spacing()`,
the mean arc between touching tiles measured at boot (0.0040 rad on the
257,948-tile map, so 0.032), rather than a number in the config that could drift
from the map; the same radius goes to clients, so the ring they draw is the clear.

**It is not `Disc`, deliberately.** Rings of neighbours on a honeycomb make a
hexagon, which showed in production as a hexagonal crater inside a round ring —
and a walk over neighbours stops at water, so an island just offshore survived a
bomb that visibly covered it. A circle has neither problem.

`drop_bomb` checks the country and the target **before** taking the bomb, so a
malformed request does not cost one. `Registry.Dropped` then brings the next box
to a window from the drop, not from when the bomb would have lapsed. Held time
still counts in full towards `maxBoostPerHour`, like any bonus.

**The blast is one event, and it rides the tile feed.** `memory_tile_storage.Clear`
empties the tiles under one lock and publishes a single `clicks.Change{Blast}`
on the same channel as the `clicks.Change{Update}` every `Set` sends. Two
reasons it is not one `TileUpdate` per tile:

- **Order.** A bonus event travels on the registry's channel, and two channels
  merged by a `select` have no order between them. A tile retaken a moment after
  the blast could then reach a client *before* the blast and be blanked by it,
  with nothing to repair it until a reload. On one channel it cannot.
- **Timing.** The client holds the clear back until its drawing of the bomb hits
  the ground, which it can only do with the whole clear in one message. 217
  updates in a burst would also overflow a slow subscriber's buffer.

`DropBomb` is session-gated like `Click` and `ClaimBonus` — it writes the map.
It is not throttled: holding a bomb the server granted is the gate.
`prom_drop_bomb` counts `bonus_bombs_dropped_total{outcome=land|sea|refused}`
and `bonus_bomb_tiles_cleared_total`.

#### What an enclose does to a click

**It looks for a small inside, never for the outline.** On a sphere every loop
cuts the planet in two, and both halves are inside it. So after an accepted
click, `click/enclose_click` floods out from each neighbour of the clicked tile
that is not the caller's, over tiles that are not the caller's. A flood that
runs out before passing `encloseMaxTiles` found a pocket; one that passes it is
open ground or too big, and takes nothing. The limit is therefore also what
tells closed from open — there is no second rule.

- **A pocket is walled by the caller's tiles alone.** A flood that reaches a tile
  with fewer than six neighbours has reached the edge of the land — a coast, a
  lake — and is open. Cutting off the tip of a peninsula is not a closed shape.
  The twelve lattice corners have five neighbours and read as an edge too.
- **A shape too big takes nothing.** Taking part of it would mean choosing which
  part, and there is no good answer.
- **A triangle costs nothing**: three tiles that touch each other have no inside,
  so the flood finds open ground and no shape is spent.
- **Only a click that takes a tile closes a shape.** A click on a tile the caller
  already held changes nothing, so it closes nothing: a shape finished before the
  bonus stays as it is. The owner is read before the rule writes, since afterwards
  the map no longer says whether the click took the tile.
- **Each pocket costs one shape**, spent through `bonus.Enclosure.Spend`, which
  settles two clicks racing for the last one. A click that closes two shapes with
  one left takes the first. A bonus with no shape left is over before its time.

**The use case only wires three objects together.** `Terrain` is the map as
the search sees it — who holds a tile, what touches it — and finds the pockets a
click closed. `bonus.Enclosure` is one caller's running bonus: its size limit and
its shapes left. `Annexer` spends a shape per pocket, takes the tiles and
announces them. `Execute` asks for the running bonus, lets the rule write, and
hands the pockets to the annexer.

It sits beside `spread_click`, against the rule and inside everything else, so
it is one click to the throttle and to `prom_click`, a shadow-banned click never
reaches it, and the tiles it takes are not reported to the antibot jury. Each
tile taken is an ordinary `Set`. The search holds no lock across the map, so a
tile can change under it; the worst that does is fill a pocket that opened a
moment ago.

**Then it tells the planet, with `tiles_enclosed`.** The tiles already travel as
tile updates, but a patch flipping at once says nothing about why, so every
client is sent the shape — closing tile, wall, and filled tiles nearest the
closing tile first — to animate. `Registry.PublishEnclosed` sends it after the
tiles are set, to every caller. The caller who closed it gets a copy of their own
with `yours` and `enclosures_left`, which is how the meter counts down; nobody
else learns how many shapes somebody has left.

`prom_enclose` wraps that publisher, so it counts exactly the shapes that were
closed: `bonus_enclosures_total` and `bonus_enclosed_tiles_total`.

#### What a bonus does to the bucket

`cpratelimit.Limiter.Boost(key, multiplier, until)` multiplies both the ceiling and
the refill rate until it lapses. It is **opt-in and additive**: a bucket nobody
boosts holds `multiplier: 1` and behaves exactly as it did before boosting
existed, which matters because the same limiter type throttles chat and session
mints and neither has any business being boosted.

Three things in there are easy to get wrong, and each has a test:

- **The refill interval is split at the moment the boost lapses.** An interval
  that straddles the end would otherwise be paid entirely at one rate or the
  other, over-granting a caller that went quiet across it.
- **The tokens are clamped back to the plain burst when it ends.** The ceiling
  came down with it, and a bucket left holding thirty under a burst of ten would
  spend the difference long after the minute was up.
- **The sweep skips a bucket still boosted.** It forgets buckets that have
  refilled to capacity, on the grounds that such a bucket holds what a fresh one
  would — which stops being true under a boost, and forgetting it would end the
  boost early.

The reward needs **no frontend release to be visible**: `State` already carries
the policy as well as the reading, so a boosted bucket reports a capacity of 30
and a rate of 3/s, and the meter widens off the server's own numbers. See
[Saying what is left](#saying-what-is-left).

### Anti-bot (`internal/antibot/`)

What is left after sessions. A player who solves Turnstile in a real browser and
then runs a userscript holds a genuine session, and no address- or token-based
check can tell them from a player. The signal that survives is **behavioural**.

**The whole of its API is seven names**, and `internal/antibot/antibot.go` is all
of it: `Config`, `Observer`, `Guard`, `New` and `Description` to wire it, plus
`Click` and `Report` — the two types a caller writes down, because it builds one
and is handed the other. A caller hands over the block and the two hooks it wants
findings reported through, and gets back a `Guard` — nil when the block is off —
that answers `Inspect`, `Committed`, `Flagged`, `Run` and `Describe`. It is
**one** `Run` whatever the file turned on: how many sweepers there are is this
package's business, which is why `planet` registers one runner rather than six.

**A caller is never taught this package's vocabulary.** The edge does two things
with a watchdog's opinion — count it if it argued for the ban, and put it in the
log line — so an `Opinion` answers `Fired()` and renders itself with `String()`,
and `Verdict`, `Evidence`, `Field` and the `clear`/`suspect`/`certain` ladder stay
inside. The alternative shipped briefly and is what this rule is written against:
the edge held a `formatOpinion` that compared against `antibot.Clear`, reached
through `Evidence.Rule` and `Evidence.Fields`, and decided their ordering —
sixteen lines of antibot's business in the clicks package, and four exported
names to support it. **The edge owns the message and the attribute names; how one
reading words itself is this package's.**

The config blocks are the same bargain the rest of the backend makes — the
settings are published because they are in the file, and koanf fills them by
reflection, so a caller sets `config.Retaker.Detector.MaxSpread` without ever
naming a type.

Everything else is under `antibot/internal/`, so the click edge
could not assemble a jury out of watchdogs even if it wanted to. `internal/planet/antibot.go`
is the whole of the clicks side, and what is left in it is genuinely the edge's:
the metric names, the message and attribute names of the ban line, and where
in the chain it sits.

**Detection and consequence are separate, and the consequence is the boring
half.** `antibot/internal/shadowban` takes a scope and a clock and runs a ban. It knows
nothing about tiles, reactions or what earned it, which is why the same sentence
serves three different findings and would serve a fourth.

**A flagged caller's clicks are answered `OK` and dropped.** That is the whole
point — a refusal names the check that tripped, and the author fixes it in an
afternoon; a silent no-op names nothing. It is not permanent (the caller reads
the map back over the same stream and will notice), but it moves the cost of
the next round onto them.

#### Three watchdogs, one jury

A `Watchdog` measures one behaviour over one caller and returns a `Verdict`:

- **`retaker`** — takes a tile back moments after losing it, over and over.
- **`sequencer`** — walks the tile ids rather than the map: 1, 2, 3, 4, on and on.
- **`metronome`** — never varies and never stops.

**Every watchdog has two levels, and that is the design.** `Certain` is a reading
no hand produces and bans on its own. `Suspect` is a reading that would ban real
players if it were trusted alone, and counts only alongside another watchdog
measuring something else. A watchdog with one level would have to sit at the
strict end and miss every bot that jitters, or at the loose end and ban humans —
which is exactly the corner the first version of this was painted into.

The `Jury` crosses them: guilty on one `Certain`, or on `jury.minSuspects`
different watchdogs at `Suspect` inside `jury.suspicionWindow`.

**Crossing is worth less than it looks, and the config says so.** Weak signals
only multiply confidence when they are independent, and these only half are: a
machine sweep is sequential *and* regular *and* never rests, so two suspicions
can be one behaviour counted twice. So can the most obsessed player. It still
earns its place because `sequencer` reads ids and `metronome` reads time — two
genuinely different measurements — but `minSuspects: 2` is the loosest this
should be, and 3 in practice means only `retaker`'s certain rule ever fires.

**Crossing is also what makes it fast.** The overnight sweep from the
screenshots trips no watchdog's `Certain` for a long time — `sequencer` wants two
hundred steps, `metronome` wants half an hour — but both read `Suspect` within
two minutes, and two `Suspect`s is a ban. `TestTheOvernightSweepIsCaught` pins
that it is stopped inside 180 clicks.

**And crossing is what the counter-move costs.** Shuffle the ids and `sequencer`
goes quiet; with nothing left to corroborate it, `metronome` has to reach
`Certain` alone, which means `certainFor` — thirty minutes of free sweeping,
bought for one line of the bot's code. `TestSweepingInARandomOrderStillGetsCaught`
pins that too. The answer to that is a fourth watchdog, not a looser bound on the
third: loosening `metronome` to catch it sooner is how the obsessed player gets
banned.

**Every watchdog sees every click, including the ones a ban is already
dropping.** A watchdog cut off the moment another one banned the caller would be
judging a caller that appears to have stopped, and the ban would lapse on silence
the caller never produced. It also means a ban sustains itself: while it runs,
the caller takes no tiles so `retaker` starves, but ids and timing still flow, so
`sequencer` and `metronome` keep re-flagging through `reflagInterval`.

#### What each one actually measures

**`retaker`: speed is not the signal, regularity is.** Two humans fighting over a
tile are fast too, and the player clicking back at a bot is the fastest of all.
`maxSpread` — the p90-p10 of the reactions — is the bound that does the work, and
on its own it is `Suspect`. `maxMedian` on top of it is `Certain`.

The first defaults here were wrong, and the split exists because of it. They
shipped at 250ms/120ms, sized for a bot answering off the update stream; the one
actually seen in production answers at ~1s, sailed past `maxMedian` and never
flagged — while holding a spread of 138ms across twenty reactions. Under one
boolean rule that bot was invisible. It is `Suspect` now.

**`sequencer`: the step, not the size of it.** Tile ids come from the
icosahedron's vertex order and not from a grid of latitudes, so filling in a
shape by hand does not hold a constant step from one click to the next. The rule
is the share of recent steps sitting at the modal step; the modal step's *size*
is not bounded, because a constant stride of 7 is no more human than a constant
stride of 1. A modal step of zero is excluded — that is somebody leaning on one
tile, which is the throttle's problem.

This is the only watchdog whose bounds need no measuring pass. Forty clicks at a
constant step is already past anything a hand produces; two hundred is not
arguable.

**`metronome`: the median is deliberately not bounded.** The claim is never that
the caller is fast. A caller pushing *past* the throttle gets its surviving
clicks handed back at exactly the refill rate, so tuning to the ceiling works
against it. What is measured is `maxSpread` over an unbroken run, where a pause
longer than `maxGap` ends the run and the evidence starts again from nothing.

**That run is the answer to a jittered delay.** A spread test is beatable by
construction — randomise and the band widens to look human. What is not cheap to
fake is *stopping*: a person's session has breaks in it. `activeFor` and
`longestGap` still feed no rule, because deciding on them alone would ban the
genuinely obsessed; they go in the log, beside a rule that did fire.

#### The parts that are easy to get wrong

**Three things are deliberately not reactions**, and each is a way to get an
honest player flagged:

- a click onto a tile the caller's own country already holds — it is a no-op,
  `Set` publishes nothing, so it neither reacts nor becomes something to react to
- a click the handler refused (invalid country, tile out of range) — it changed
  no tile
- a click that was itself dropped by the ban — same reason

The interceptor reads the tile's owner **before** the handler runs and puts it on
`antibot.Click` as `Held`/`NoOp`, because afterwards the map no longer remembers
it. `Committed` is then called only for a click the handler accepted. Without
that split, a griefer spams a tile with deliberately invalid clicks and the next
honest player to click it looks like it is reacting to something.

Note that `sequencer` and `metronome` ignore all of it: a bot sweeping ids walks
over tiles it already owns and over ids the handler refuses, and both are part of
the walk.

**It is a decorator over the click use case** (`antibot_click`), not an interceptor.
None of what it does is about HTTP: it reads who held the tile before the write,
reports the take afterwards, and answers a flagged caller OK with nothing
written — and the middle one only works adjacent to the write, because
afterwards the map no longer remembers who held the tile.

**It sits innermost, inside the throttle** — the opposite of the blocklist and the
session check. A shadow-banned caller has to keep hitting the same 429s everyone
else does; a caller that is never throttled again has been told. `TestAntiBotRunsAfterTheThrottle`
pins it.

**`antiBot.shadowBan.enforce` is the rollout switch,** the same shape as
`session.enforce`: false judges, logs and counts without dropping anything. The
two surfaces are built for a box with no dashboard — `click_reaction_seconds` is
a histogram whose raw bucket counts show the bot band by eye, and each flag
writes one `antibot ban` log line carrying the scope, every watchdog's verdict
and numbers (**including the ones that said `clear`** — what did not fire is half
of reading a line that did), the tiles, and the country the caller painted with
most. **The address is never a metric label** — unbounded cardinality, and
personal data in every scrape. `topCountry` is context for a human reading the
log and never an input to a rule: the client declares it, so it is changed by
editing one string, and real players paint the same flags a bot does.

**A flag repeats, and that is most of its value.** `reflagInterval` is how soon a
caller already serving a ban can be judged again; at or above a watchdog's
`trackWindow`, each flag rests on evidence the previous one never saw, so
`flags=6` on a line is six independent judgements agreeing rather than one
verdict repeated.

Keyed on `cpipscope.Of`, the same unit as the throttle, so a v6 caller cannot serve
a ban on one address and click from the next in its own /64. It only bites a bot
with a stable address — against a residential proxy pool it evaporates for
exactly the reason the rate limiter does.

`memory_tile_storage.Owner` exists for this: one indexed read under the existing
lock, declared as a local port in the controller the way each use case declares
its own.

### The error net

**No handler's raw error reaches the wire, and no module has to remember that.**
`cpbootstrap` wraps `cpconnect.NewErrorInterceptor` around every service it
mounts, outside whatever interceptors the module named: an unrecognised error is
logged once, with its procedure, and answered as `internal error` with the cause
stripped. `TestEveryMountedServiceGetsTheErrorNet` mounts a module that asks for
nothing and pins that it still gets it.

**It lives at the mount and not in a controller because it is a property of the
process**, not of a context. It used to be three near-identical constructors,
one per module, each of which could have been left out of a chain and none of
which any test would have missed.

**That is also why `Mount` takes a builder rather than a handler.** A Connect
interceptor is baked in at `New<Service>Handler`, so there is nothing to wrap
afterwards, and an HTTP middleware is far too late — by then the error is a
serialized response body. Building inside `Mount` is the only place the server
can put anything around every procedure in the process. The module writes a
one-line closure, because the generated constructor takes the service
*interface* while the module holds the concrete type, and no type parameter can
infer that conversion.

**What each module still owns is its own mapping.** A caller error is named by
the part that found it and turned into a code by the handler that knows which
procedure was asked — `click_handler` and `get_map_handler` in planet,
`chatv1controller.toConnect` for `ErrInvalidMessage`, `sessionv1controller.toConnect`
for `ErrAttestationFailed`. The net never sees those, because a `*connect.Error`
is passed through untouched.

The session one keeps a log line of its own: a refused mint is logged at **Info**
there, because the net logs at Error and a refusal is the check doing its job
rather than a fault of this server — on a public endpoint it is the common case.

### Shared interceptors

`shared/cpconnect` holds the two interceptors both contexts need, because the policy is the same whatever the procedure is — only the procedure names and the wording of the refusal differ, and those are arguments:

- `NewRateLimitInterceptor(limiter, refusal, procedures...)` — a `shared/cpratelimit` bucket keyed on `cpctx.RateLimitKey`, answering `CodeResourceExhausted` (429). It reports nothing about what is left: a context that shows a player their allowance throttles inside its own use case instead, where the reading is a return value. This is for the procedures where a refusal is the whole story — chat and sessions
- `NewIPBlockInterceptor(blocklist, refusal, onBlocked, procedures...)` — a `cpipblock.Blocklist` lookup answering `CodePermissionDenied` (403), with an optional hook the click counter hangs on
- `NewSessionInterceptor(verifier, clock, refusal, enforce, onVerdict, procedures...)` — a `shared/cpsession` signature check answering `CodeUnauthenticated` (401), which puts the session id on the context and, with `enforce` false, counts without refusing
- `NewErrorInterceptor(logger, mapper)` — the net, applied by `cpbootstrap` rather than by any module (see [The error net](#the-error-net)). It is a full `connect.Interceptor` rather than a `UnaryInterceptorFunc`, so it covers the streaming handlers too; without that, a stream would be the one procedure whose raw error the caller sees. The `Mapper` is optional and nothing passes one any more.

Each context keeps a thin named constructor over these — `planetv1controller.NewVPNBlockInterceptor`, `chatv1controller.NewRateLimitInterceptor` and `NewBlocklistInterceptor` — which is where the procedure list, the refusal wording and the metric live. **A context names its own policy; neither reimplements the mechanism.**

Both chains order them the same way: error mapping outermost, then the blocklist, then the limiter. **The blocklist has to sit outside the limiter** — a refused address must not also spend a token, or its next call would come back 429 and the client would report the wrong reason. `TestVPNBlockRunsBeforeTheThrottle` pins that for clicks.

### Durability

The whole map is snapshotted to `tilesStorage.snapshotPath`:
- a compact binary encoding, not JSON: magic + version + CRC32, then the interned country code table, then two bytes per tile
- written atomically — temp file, fsync, `os.Rename`, fsync of the directory — so a crash mid-write leaves the previous snapshot intact
- flushed every `snapshotInterval` when the state changed, and once more on graceful shutdown (`cmd/api` handles SIGINT/SIGTERM for exactly this)
- restored at boot; a missing, truncated, or corrupt snapshot logs and starts from an empty map, it never prevents a start
- a snapshot taken at a different `gameMap.maxIndex` restores the overlap

**What this costs:** anything written since the last snapshot is lost on a hard kill (`SIGKILL`, OOM, power loss), bounded by `snapshotInterval`. And because the state is per-process, **this is single-instance only** — two API replicas would each hold their own divergent map. Both are deliberate: the game state is a few MB and the update fanout was already per-instance, so a database was buying durability alone.

The snapshot file is the only thing worth backing up.

### Shared (`internal/shared/`)

Shared infrastructure: `cpbootstrap` (the composite layer), `cpcountries`, `cpconfigs` (YAML + env config via koanf), `cphttpserver` (middleware, formats), `cpprom` (Prometheus), `cptime`, `cpctx`, `cpconnect`, `cpratelimit`, `cpipblock`, `cpipscope`, `cpsession`, `cpatomicfile`, `cpsecrets`.

**Every package here is prefixed `cp`, and a new one must be.** A call site reads
`cptime.SystemClock{}` or `cpctx.GetSourceIP(ctx)`, so the prefix says the
dependency is this layer's without the reader going to the import block — and an
unprefixed name in a module is a module's own package by construction. It also
settles the collisions a shared layer attracts: `cptime` beside stdlib `time`,
`cpsession` beside the session *module*, `cpconnect` beside `connectrpc.com/connect`.
`cpsession` is what let `internal/session/module.go` drop the `sharedsession`
import alias it needed while the two were both called `session`.

**There is no logging package here, deliberately.** Every constructor that logs
takes a `*slog.Logger` from the standard library. The wrapper that used to sit
here was an interface of four methods over `log/slog` plus a `cplf` field type
that stringified every value on the way in — so a structured logger was being
flattened to strings by the layer whose job was to keep them structured. A
handler is the supported way to change where logs go, and `slog.New(slog.DiscardHandler)`
is the nop. The one thing lost is the name `Warning`, which is `Warn` in slog.

The `util` suffix is dropped rather than prefixed — `cpctx`, not `cpctxutil`.
A package named for what it *is* stays that; one named "utilities for X" was
never saying anything the prefix does not.

`cpbootstrap` runs the modules — see [The composite layer](#the-composite-layer). It knows nothing about this game: it takes a list of modules, builds each one under a startup deadline, mounts what they claimed on one router, and serves until the process is signalled.

Two of these are here because both bounded contexts need them and neither should depend on the other:

- `cpsecrets` — the random hex a config may leave it to the server to invent. Chat's tag salt and the session signing key are the two, and both pay the same price for an empty setting: what the old one covered stops being recognised on restart.
- `cpatomicfile` — temp file, fsync, rename, fsync of the directory. Written for the tile snapshot; the chat log's retention rewrites need the same guarantee, and duplicating 80 lines of carefully-written fsync/rename code is how the two drift apart. Covered by the existing snapshot tests.

`cpsession` mints and verifies the click token — see [Sessions](#sessions-internalsession). It is here because **both** contexts read it: the session context mints with it, the planet context verifies with it, and neither may depend on the other.

The siteverify client it is fed by is **not** here. `turnstile` sat here on the same "both contexts need it" rule, but only one ever did, so it now lives at `session/internal/turnstile` where the compiler keeps it. **The bar is not that a package is shareable, it is that it would read the same in any other program and that two modules actually import it** — "shared" names the symptom, and a directory admitted on the weaker reading becomes a dumping ground. `cpsecrets` passes narrowly — chat is its only caller today, but it is twenty lines of `crypto/rand` with no domain in it at all.

`cpcountries` is the ISO country list both the tile game and the chat validate against. `cpipblock` is the VPN prefix set — see [VPN blocklist](#vpn-blocklist). `cpratelimit` is a keyed token bucket held in this process, like the tile map it protects — with one API instance, a shared counter would buy nothing. Its `Run` loop periodically forgets the buckets that have refilled to capacity, which is free: such a bucket holds exactly what a freshly created one would, and without it the map would keep an entry per address that ever clicked.

`cpipscope` decides what a bucket is keyed on, and every throttle goes through it. Over IPv4 that is the address; over IPv6 it is the surrounding **/64**, because the smallest allocation a subscriber receives is a /64 and most receive far more — a bucket per v6 address is one the same line walks out of by picking its next address, turning one home connection into thousands of callers with a throttle each. The session token binds to the same unit, so the address a token is valid for and the address that spends a budget cannot diverge. Blocking deliberately does **not** use it: the VPN and datacenter lists are precise prefixes already, and widening a hit to the surrounding /64 would refuse neighbours who are not on them.

### Map geography

**Which tiles touch which.** It exists so that bonuses depending on the shape of the map can be
decided by the server: "a click spreads to the 6 adjacent tiles" cannot be computed on the client
without the client naming the tiles it gets, which is the whole of the cheat.

**It is `clicks`, not kernel**: the kernel is for what would read the same in any other program,
and this is the tile game's own map. It is then split across the two layers, on the line of *what
survives a change of input format*:

- **`clicks.Geography`** (`internal/clicks/geography.go`) — the model. `NewGeography(positions, edges)`
  builds it, and `Neighbours`, `Disc` and `Position` read it. It knows nothing about icosahedra,
  blobs or file formats; it takes an edge list in any order, with repeats, and enforces what has
  to be true of a map: tiles in range, nothing touching itself, no tile above `MaxDegree`, and
  **no asymmetric edge** — a tile that spreads onto a neighbour which would not spread back is a
  one-way street on the map.
- **`internal/adapters/secondary/geodesic_map`** — the recovery. Everything in it exists because the
  adjacency is *not* shipped: the positions are, and the edges have to be worked back out of them.
  The blob decode, the icosahedron lattice and the position index are all knowledge of the shipped
  artifact, not of the game. **Ship a precomputed edge list one day and this package goes while
  `clicks.Geography` does not change at all.**

The blob itself is **`generated/map`**, beside `generated/proto` and for the same reason: both are
this app's committed copy of something the root owns, written by a `make` target and never edited
by hand. The vendored VPN ranges sit next to their reader instead (`shared/cpipblock/cpdata`) because
they come from a third party, not from `/`. The directory is `map` to mirror its source; the
package is `mapdata`, since `map` is a keyword.

That line also splits the checks. The domain enforces what is true of any map; the adapter enforces
what is true of *this* blob — the tile count against `gameMap.maxIndex`, and that every tile lands
on a detail-300 lattice vertex.

And it splits the tests. `domain` is tested against a hand-built patch of honeycomb — fast, no 5 MB
asset, and it says what `Geography` does rather than what the shipped blob happens to contain.
`geodesic_map` is tested against the real blob, and holds every number below.

**The tile grid is a regular honeycomb, not an arbitrary numbering.** Tile ids look like noise but
they are the land vertices of `THREE.IcosahedronGeometry(1, 300)`, deduplicated by position — a
geodesic sphere, where **every interior vertex has exactly 6 neighbours** and the 12 icosahedron
corners have 5. The parameters and the evidence for them are in [`/map/README.md`](../../map/README.md).

#### How adjacency is computed

Each icosahedron face is a triangular integer lattice: `(p, q, r)` with `p + q + r = 301`, all
non-negative, sitting at `normalize(p*A + q*B + r*C)` for the face's **un-normalised** corners —
THREE lerps across the flat triangle and normalises afterwards, so normalising the corners first
moves every vertex. The six neighbours are the ±1 exchanges between any two coordinates. Walking
all 20 faces, resolving each lattice vertex back to a tile by position, and taking the union gives
the adjacency exactly.

**Cross-face edges and the 12 corners need no special case.** A vertex on a shared edge is reached
from both faces, contributing 4 neighbours each with 2 in common; a corner is reached from all five
faces that meet there, 2 each with each shared once. 6 and 5, which is what the lattice says.

**Do not replace this with a radius-based nearest-neighbour search.** It is the obvious approach and
it fails quietly: because THREE subdivides the flat triangle, spacing varies ~25% between face
middles and corners (nearest-neighbour distances run 0.00309 to 0.00440), so no single radius works
anywhere. Measured, a tuned radius gave **2,137 tiles degrees of 7, 8 and even 10**. The lattice
walk has no threshold in it at all — the only tolerance is `matchEpsilon`, and that is f32 rounding
(~1e-7 against a 3.09e-3 gap between the closest two tiles), not a search radius.

#### The off-by-one

**Wire tile id = blob array index + 1.** The blob is 0-indexed by position; ids on the wire are
1-based, which is what `in_memory_tile_checker`, `memory_tile_storage`'s unused slot 0 and the
frontend's `integerToColor(i + 1)` all agree on. Getting it wrong shifts every neighbourhood by one
tile, **symmetrically, with a degree histogram that still looks right** —
`TestTileIDsAreOneBasedOverTheBlob` is what catches it.

#### Checked at boot, not just in the tests

`geodesic_map.Load` is the first thing the planet module builds and **fails the boot** on a blob whose
tile count is not `gameMap.maxIndex`, on any tile that does not sit on the detail-300 lattice, on a
degree above 6, or on an asymmetric edge. The tests cannot see the blob a container was actually
built with, and that is the thing that drifts; regenerating the coordinates renumbers every tile, so
a process quietly disagreeing with the frontend about what tile 42 is has no repair after the fact.
In practice the failure is unreachable in production — the blob is embedded, so a bad one fails
`make test` long before a deploy — which is what makes it cheap.

It costs ~0.3s of boot and about 12 MB resident. The boot log carries the whole result in one line:

```
map geography loaded asset=coordinates-26a9aeab.bin tiles=257948 edges=752820
  degrees="[186 530 1440 4087 6216 7829 237660]" took=263ms
```

Those numbers are pinned by the tests. 752,820 undirected edges, average degree 5.837, and the
degree histogram reads: 186 tiles with no neighbours, then 530, 1440, 4087, 6216, 7829, and 237,660
inland tiles with the full 6. Walking adjacency alone finds **443 landmasses**, the largest four
being 142,827 (Afro-Eurasia), 67,957 (the Americas), 20,037 (Antarctica) and 12,335 (Australia).

#### The API, and what is not in it yet

All three are on `clicks.Geography`, so a rule that reads them needs no port and no adapter import.

`Neighbours(id)` hands back a window into the map's own CSR table — read it, never write it, which
is what keeps it allocation-free on a click path. **It returns nothing for the 186 single-tile
islands**, so a consumer has to have an answer for an empty neighbour set; whether the game paints
nothing or refunds the bonus there is a rule, not a fact about the map.

`Disc(id, radius)` is a breadth-first walk over that, returning `1 + 3r(r+1)` tiles inland and less
wherever the land runs out. **Discs are computed, never stored**: a radius-3 table would be ~38 MB
to save ~50µs, behind a 1 click/sec/IP throttle. The search scratch is a generation-stamped array
from a pool, one per concurrent caller, so nothing is cleared per call and two clicks cannot stamp
the same array.

`Neighbours` is what the spread bonus reads — see [What a spread does to a click](#what-a-spread-does-to-a-click).
`Within`, `Position`, `Nearest` and `Spacing` are what the bomb reads — see [What a bomb does](#what-a-bomb-does).
`Within`, `Nearest` and `Spacing` are straight scans (a few ms over 257,948 tiles): `Spacing` runs once at
boot, `Within` and `Nearest` once per bomb. `Disc` is back behind the `testing` tag — see [Testing](#testing).

#### Known faults, inherited and documented

From the blob, not from this package: the antimeridian row carries ~¼ the tiles it should, so
neighbourhoods near the dateline are lopsided, and 2,523 tiles fall outside every country. Both
leave tiles with fewer than 6 neighbours, which is also what a coastline does — there is no way to
tell them apart from the geometry, and fixing them means regenerating and renumbering.

### Configuration

**`shared/cpconfigs` owns loading**, the way `cpbootstrap` owns running: the binary asks for the config it wants and never for the flag, the parser, or the precedence between file and environment.

```go
var config Config
err := configs.Load(&config, configs.FromFlag())
```

Where the file comes from is an option — `FromFlag()` reads `-config`, which is how the container runs it; `FromFile(path)` names one outright and lives behind the `testing` tag, because two test packages need it and no production caller does (see [Testing](#testing)). **An empty path is not an error**: every field keeps its zero value and the environment alone can carry a whole config.

Config is loaded from a YAML file, with environment variables overriding it — `.` is the nesting delimiter, so `tilesStorage.snapshotPath=/data/tiles` in the environment overrides the file. See `cmd/api/example.yaml` for the full schema.

**A config that implements `Validate() error` is asked to check itself**, and the load fails with its sentence wrapped in `cpconfigs.ErrValidation`. That is where a bad setting is refused out loud rather than becoming a zero value nothing reports.

**Every block validates its own, and `cmd/api` only fans out:**

```go
func (c Config) Validate() error {
	return errors.Join(
		c.HTTPServer.Validate(),
		c.Clicks.Validate(),
		c.Session.Validate(),
		c.Chat.Validate(),
	)
}
```

The binary never reads inside a block to check it, so a new bound is added in the module that owns it and nothing here changes. `errors.Join` also means a broken file reports **everything** wrong at once rather than one line per restart.

- `cpbootstrap.ServerConfig` — `bindAddress` empty listens on port 80
- `planet.Config` — `gameMap.maxIndex` zero is a map that refuses every click
- `shared/cpsession.Config` — `secret` empty while `enabled`, and a negative `ttl`. It sits with the block rather than with either context, because both read it and it must be checked exactly once
- `chat.Config` — nothing: every chat setting has a usable default, so an unset one is a default and not a mistake. It implements the hook anyway, so a check added later lands in chat

There is no struct-tag validation and therefore no validator dependency — a hook the config implements covers this app's needs.

**Each module owns its own config struct** — `planet.Config`, `chat.Config`, `session.Config` — and `app.Config` is the three of them plus `httpServer`. The planet keys stayed at the top level of the file rather than moving under a `planet:` section: `app.Config` squashes that struct (`koanf:",squash"`), so the file and every `deploy/` environment variable are unchanged.

- `httpServer.bindAddress` — the encoding is negotiated per request, so there is no format setting.
- `httpServer.streamHeartbeat` — how often a silent live stream sends a heartbeat (default 30s). **Must stay well under the proxy's idle cut**: Cloudflare answers 524 at ~125s, and a stream that never speaks is one it kills.
- `gameMap.maxIndex` — total number of tiles
- `tilesStorage.snapshotPath` — where the state is persisted; **empty disables durability**
- `tilesStorage.snapshotInterval` — how often a changed state is flushed
- `tilesStorage.subscriberBuffer` — per-subscriber channel capacity, which is now per connected client rather than per fanout; updates for a subscriber that cannot keep up are dropped, not blocked on
- `rateLimiter.perSecond`, `rateLimiter.burst`, `rateLimiter.sweepInterval` — the per-IP click throttle (defaults 1/s, burst 10, swept every minute)
- `vpnBlocklist.enabled`, `vpnBlocklist.includeDatacenters`, `vpnBlocklist.allow` — the VPN refusal (see [VPN blocklist](#vpn-blocklist)); disabled parses nothing and allocates nothing
- `bonus.enabled` — off offers nothing and answers `ClaimBonus` Unimplemented
- `bonus.interval` — how often a box is put in front of somebody; a ceiling, since nothing is offered while nobody is watching
- `bonus.offerTTL` — how long the token stays good; **must outlast the flight the client draws**, or a box caught on its last frame is refused
- `bonus.kinds` — a weight per kind (`triple_clicks`, `spread_clicks`); a kind's chance is its weight over the sum. Left out or 0 is never offered, empty offers every kind equally, and an unknown kind, a negative weight or all zeros refuse the boot
- `bonus.spreadDuration` — how long a caught `spread_clicks` runs (default 10s). It is much shorter than `bonus.duration` because a click that takes seven tiles is worth far more than three clicks; `maxBoostPerHour` counts the time each bonus really ran
- `bonus.duration`, `bonus.multiplier` — how long a caught `triple_clicks` runs and what it multiplies the allowance by; the client reads both off the answer, so changing them changes the meter with no frontend release
- `antiBot.enabled` — off registers nothing and measures nothing
- `antiBot.shadowBan.enforce` — off judges, logs and counts without dropping; the mode to deploy in
- `antiBot.shadowBan.banDuration`, `reflagInterval`, `sweepInterval` — how long one flag silences a caller, how soon it can be judged again, and how often a ban nothing would still print is forgotten
- `antiBot.jury.minSuspects` — how many watchdogs at `suspect` make a ban; one at `certain` bans alone
- `antiBot.jury.suspicionWindow`, `trackWindow`, `sweepInterval` — how long a verdict stands while another watchdog catches up, and how long a silent caller is remembered
- `antiBot.retaker.enabled`, `detector.reactionWindow`, `minReactions`, `maxSpread`, `maxMedian` — what counts as a reaction, how many are needed, and the band that reads `suspect` then `certain`
- `antiBot.sequencer.enabled`, `detector.minSteps`, `minShare`, `certainSteps`, `certainShare` — how long a run of constant-stride clicks must be, and how much of it must sit at that stride
- `antiBot.metronome.enabled`, `detector.maxGap`, `maxSpread`, `minClicks`, `certainFor`, `certainClicks` — what ends a run, how tight its gaps must be, and how long it must hold
- every watchdog also takes `detector.trackWindow` and `detector.sweepInterval` — how far back its evidence counts, and how often what can no longer matter is forgotten
- `session.enabled` — off registers nothing, so `session.v1.SessionService/` 404s and clicks are judged on address alone
- `session.enforce` — off counts what enforcing would refuse without refusing it; the mode to deploy in
- `session.secret` — signs the tokens; **required once `session.enabled` is true**, and an empty one refuses the boot rather than being invented
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
- `chat.blockedIPs` — prefixes refused every chat RPC, parsed by `shared/cpipblock` exactly as `vpnBlocklist.allow` is

### Protobuf

API contracts live in the monorepo-shared [`/proto`](../../proto) (also used by the frontend), one package per bounded context: [`planet/v1/planet.proto`](../../proto/planet/v1/planet.proto) and [`chat/v1/chat.proto`](../../proto/chat/v1/chat.proto). Generated code goes to `generated/proto/`. Use `make proto` to regenerate after editing `.proto` files (requires the `buf` CLI, plus `protoc-gen-go` and `protoc-gen-connect-go` on `PATH`).

`generated/` is for everything the root owns and this app carries a committed copy of, because the Docker build context is this directory: `generated/proto` from [`/proto`](../../proto) via `make proto`, and `generated/map` from [`/map`](../../map) via `make map` — see [Map geography](#map-geography). Nothing in there is edited by hand; run the target.

The proto package is the **only** version number: Connect derives each route from it, and the controller and connect package names follow — `planet.v1` gives `planetv1controller` and `planetv1connect`, `chat.v1` gives `chatv1controller` and `chatv1connect`. There is no gRPC here — Connect serves the service definitions over ordinary HTTP/1.1 POSTs (and h2c, for clients that want it).

### Testing

Unit tests only, using `testify`. There are no integration tests and no Docker dependency — `make test` runs everything from a clean checkout.

**Tests build with `-tags testing`, so use `make test` rather than a bare `go test ./...`.** Anything else that loads test files needs the tag too: `go vet -tags testing ./...`, and an editor's language server (`gopls` `buildFlags: ["-tags=testing"]`, or `go.buildTags` in VS Code), which otherwise reports the helpers as undefined. A helper that more than one package needs cannot live in a `_test.go` file, so it lives in an ordinary `.go` file carrying `//go:build testing`. The tag, not a filename convention, is what keeps such a helper out of the production binary — and what lets `make deadcode` tell a helper apart from production code. `cpctx.GetSessionID`, `cpconfigs.FromFile` and `cptime.FixedClock` — the stand-still clock a dozen test packages drive time with — are the three that exist today.

### Linting

`make lint` runs **golangci-lint**, configured in [`.golangci.yaml`](.golangci.yaml).
The version is pinned in the `Makefile` and matched by the CI job, because an
unpinned linter turns a green branch red on somebody else's release schedule.
`make setup-tools` installs that version.

**Formatting is gofumpt**, enabled in the `formatters` block of the same file
and run by `make tidy`. It goes through golangci-lint rather than a `gofumpt`
binary of its own: one pinned version to install instead of two that can
disagree, and it already knows which files are generated. gofumpt is a strict
superset of gofmt, so anything it accepts `gofmt` accepts.

`make check-format` is the non-rewriting half, and it is what CI runs. Note that
`golangci-lint fmt --diff` **prints a diff but exits 0 either way**, so the
target tests its output rather than its status — a `check-format` written the
obvious way passes on an unformatted tree. It also runs `go mod tidy` and fails
on a resulting diff, so an untidy `go.mod` is caught in the same place.

**`run.build-tags` is `testing`**, for the same reason `go vet` needs it: without
the tag the linters load a tree where the shared helpers are undefined and report
that instead of anything real.

**Test files are linted like production code.** The only per-path exclusion is
`generated/`, which `make proto` owns. Where a linter is wrong about a specific
line, the line carries a `//nolint` naming the linter and the reason — which
keeps the rule live everywhere else:

- `nilnil` — six constructors return `(nil, nil)` for **"this feature is off"**,
  and the caller checks for nil and mounts nothing. A sentinel would make every
  caller unwrap one. Annotated per site so an *accidental* `nil, nil` is still caught.
- `gosec` G304 — file paths that come from config, never from a request.
- `gosec` G404 — `math/rand` in the antibot tests is deterministic on purpose:
  a fixed seed replays the exact click stream a watchdog is asserted against.
- `bodyclose` — it cannot see a body closed by a helper's `t.Cleanup`.

**Two linters are deliberately off**, and both were switched off on evidence
rather than left out:

- **`exhaustruct`** wants every field named at every struct literal. That suits
  DTOs; it does not suit this codebase, where most structs are stateful objects
  whose zero values are correct and meaningful. It produced 161 findings, of
  which the representative ones are `Jury is missing field mu` (a `sync.Mutex`
  you cannot meaningfully write in a literal), `caller is missing fields clicks,
  tiles` (counters that start at zero), and nine `Evidence is missing fields
  Rule, Fields` — where `Evidence{}` *is* the "nothing to report" value.
- **`forbidigo`** has nothing to forbid here yet. It is worth turning on the day
  a retired pattern needs to stay retired.

`wrapcheck` is on, with `extra-ignore-sigs` for the signatures this codebase
returns bare **on purpose**: use cases called by a handler, because the handler maps the
domain sentinels centrally and a wrap would put a second sentence in front of a
message it already chose; and pure delegations, where the callee already named
what failed.

### Git hooks

`./.githooks/install` points git at [`.githooks/`](../../.githooks). It is a
script rather than a root `Makefile` target because this repo keeps build tooling
inside each app, and hooks are the one genuinely repo-wide thing.

- **pre-commit** — `make tidy` (re-staging only what was already staged, plus
  `go.mod`/`go.sum` if tidy moved them) and `make lint-ci`, plus the frontend's
  eslint, each only when that app has staged changes.
- **commit-msg** — conventional commits, which the history already uses, and a
  72-character subject so `git log --oneline` stays readable. Merge, revert,
  fixup and squash subjects are git's to format and are left alone.
- **pre-push** — `make test`, `make deadcode`, `make lint-ci` and
  `make check-format`, run concurrently; output is only printed for a step that
  fails.

All three take `--no-verify`. The hooks re-point `core.hooksPath` at a *relative*
`.githooks` on every run, so a worktree runs its own branch's hooks rather than
the main checkout's.

The tag has a second use, same mechanism and a different reason: **production code that is written
and tested but has no caller yet**. `clicks.Geography`'s `Disc` is there: the bomb used it for a
release, then moved to a circle. The tag is what keeps `make deadcode` a wall rather than a thing people learn to
ignore, and removing the line is the whole of promoting such a function.

**`make deadcode` fails on any unreachable function**, in two passes, because "is this reachable?" has two different right answers depending on whether test code counts as a caller. The first pass excludes tests and tagged files, so **production code whose only caller is a test is reported as dead** — the case a plain `deadcode -test` forgives. The second pass includes both but keeps only findings inside tagged files, so an unused shared helper is reported too. `deadcode` is fetched at a pinned version by the target, so there is nothing to install.
