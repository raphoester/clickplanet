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

### Four bounded contexts, one process

- **`internal/clicks/`** — the tile game: clicks, ownership, the map, the update stream.
- **`internal/chat/`** — the live chat: messages, identity, retention.
- **`internal/session/`** — the mint: what a caller has to prove before it may click.
- **`internal/antibot/`** — who is a machine, and what happens to them.

`antibot` is the one with no proto package and no adapters, because nothing
calls it: the clicks edge gates on it the way it gates on `session`. It is a
context and not a kernel package because "is this caller a bot" is the business
this game is in, while the kernel is for things that would read the same in any
other program.

They share the process, the transport and the country list, and **nothing else**. None imports another; each owns its own domain types, its own proto package, its own storage adapter (where it has one) and its own edge.

### The composite layer

Each context wires **itself**, in a `module.go` at its root (`internal/clicks/module.go`, `internal/chat/module.go`, `internal/session/module.go`). A module is a `bootstrap.Module` — a name and a DI sequence — and the sequence is handed a `bootstrap.Props` carrying registrars and nothing else:

- `props.RPC.Mount(path, handler)` — both return values of a generated `New<Service>Handler` go straight into it
- `props.Runners.Add(name, run)` — a goroutine, given the process-lifetime context
- `props.Closers.Add(name, close)` — a cleanup, run in reverse registration order
- `props.Logger`, `props.Metrics`

A module never sees the router, the server, the signal handler or another module's dependencies. **There is no mutable app object and nothing to leave a half-built dependency on**: `internal/kernel/bootstrap` builds every module in order, then serves.

**`internal/app/` is the composition root and does three things**: load the config, build what more than one context needs, and name the modules. That list is the whole DI sequence, and it is readable top to bottom in `describeModules`.

**What two contexts share is a variable there that both are handed**, which is the only way they can share anything:

- `kernel/session.Signer` — the session context mints with it, the clicks context verifies with it. Nil when sessions are off, which leaves the click chain as it was before they existed. Neither imports the other, and **clicks knows nothing about Turnstile** — swapping the attester changes one line in `internal/session/module.go`.
- the country list — both the tile game and the chat validate against it.

Adding a context that callers talk to means a `proto/<name>/v1`, an `internal/<name>/` with a `module.go`, and one line in `describeModules`. Connect derives the route from the proto package, so there is no prefix to allocate and no router to edit.

**`cmd/api`** is the only binary and is now four lines: `app.Run(ctx)`.

It runs as a **single self-contained container with no dependencies**: the tile map lives in process and is persisted to a local snapshot file. There is no database, no cache, and no second process.

### Clicks domain (`internal/clicks/domain/`)

Core interfaces (ports) defined in `gateways.go`:
- `TilesChecker` — validates tile IDs (0..maxIndex)
- `TileStorage` — reads/writes tile→country ownership
- `TileReporter` — notifies downstream of tile updates
- `CountryChecker` — validates ISO country codes

`ClickHandlerService` wires these interfaces together and contains all game logic. The Prometheus-instrumented version (`prom_click_handler_service/`) wraps it via decorator pattern.

### Adapters

**Primary (input):**
- `adapters/primary/http/planetv1controller/` — the API. `ClickService` implements the generated `planetv1connect.ClickServiceHandler` and nothing else; it never sees an `http.ResponseWriter`, which is the point of serving the contract with Connect rather than by hand. Two interceptors wrap it: `NewRateLimitInterceptor` (see [Rate limiting](#rate-limiting)) and `NewErrorInterceptor`, which maps domain errors onto Connect codes, logs the unexpected ones and keeps their cause off the wire, so **handlers return their errors bare** — `domain.ErrInvalidArgument` becomes `CodeInvalidArgument`, a code a handler picked itself is left alone, and anything else is logged once and answered as `internal error`.
- the tile stream, as `ClickService.ListenForEvents` — a Connect server-streaming RPC like any other procedure on the service. See [The live streams](#the-live-streams).

**There is one server, one mux, and no version prefix.** Connect names each service's path from its proto package — `/planet.v1.ClickService/` and `/chat.v1.ChatService/` — so nothing is mounted under a prefix of ours. The three services and `/metrics` are the only things on the router. Nothing here needs a connection-level demultiplexer such as `cmux`; that is for running a real gRPC server, which owns its own HTTP/2 handler, beside a REST one.

`bootstrap` sets `http.Protocols` with both HTTP/1.1 and unencrypted HTTP/2, because the generated handler also speaks gRPC and gRPC-Web and those need HTTP/2. Browsers reach the same routes over HTTP/1.1. Verified: HTTP/1.1 and h2c both answer on the same port.

### The live streams

Both live feeds are served **two ways at once**, and that is a transition, not a design:

- `ClickService.ListenForEvents` → `stream PlanetEvent`, and `ChatService.ListenForEvents` → `stream ChatEvent`. Ordinary Connect server-streaming RPCs, on the same routes and the same port as everything else.

**One stream per API, and an envelope rather than a bare payload.** A `PlanetEvent` is a `oneof` of `tile_update` and `heartbeat`; `ChatEvent` is a `oneof` of `message` and `heartbeat`. **A new kind of live event is a new case in that `oneof`, never a second stream** — one connection per client, one route to configure, and a client that does not know a case reads an unset `oneof` and skips it instead of breaking. That is what the bare `TileUpdate` frame could not do, on the websocket or off it.

**`heartbeat` is not decoration.** Cloudflare cuts a silent response at **~125s with a 524** — measured against production three times, exactly 125.1s. The websocket never hit this because Cloudflare keeps those open; a chunked HTTP response is not so lucky. A quiet chat is the normal case, and a quiet planet happens, so both streams send a heartbeat every `httpServer.streamHeartbeat` (30s by default, and it **must** stay well under 125s). Without it a silent stream dies and reconnects forever, losing whatever was published in each gap.
These replaced a pair of websockets on `/ws/listen` and `/ws/chat`, broadcast by a `kernel/wspublisher` fanout. **Nothing here speaks websocket any more** — no upgrade route, no second mux, no `coder/websocket` dependency.

Each handler calls the storage's `Subscribe(ctx)` **per call**, and the request context is what unsubscribes — it is cancelled however the stream ends, so a disconnect needs no `CloseRead` equivalent. Both storages already handed every subscriber its own buffered channel and dropped rather than blocked for a slow one, so one subscription per connected client is what they were built for; `subscriberBuffer` now bounds a client rather than the single fanout.

**The streaming RPCs are not wrapped by any interceptor except error mapping**, because every other one is a `connect.UnaryInterceptorFunc` and streams skip those by construction. Reads and the live feeds are therefore untouched by the throttle, the VPN blocklist and the session check, exactly as they were when they were websockets. A policy that ever has to reach a stream must be written as a full `connect.Interceptor`.

### The map load

`GetMap` is an ordinary RPC, but marked `idempotency_level = NO_SIDE_EFFECTS` in the proto, so Connect sends it as an **HTTP GET** and the handler sets `Cache-Control: public, max-age=5` on the response. A burst of visitors can therefore share one origin response; `ListenForEvents` carries everything that happens after a chunk was built, so a client starting from a slightly old map converges anyway. `MapDensity` is marked the same way.

The response never repeats a tile id. `GetMapResponse` carries `start_tile_id`, the interned `codes` table, and `tiles` — a `bytes` field holding two bytes per tile, little endian, indexing into `codes`. Tile ids are implicit in the position, which is what makes it far smaller than the deprecated `map<uint32, string>`: **516 KB against 3.6 MB** for a full 257,948-tile map.

`memory_tile_storage.StateBatchDense` builds it. The interned ids are copied out exactly as stored and the table travels with them, so nothing is translated on the way out and the client needs no shared country list. Protobuf does all the framing — there is no hand-rolled magic or length-prefixing on either side, and therefore no encoder and decoder that have to be edited together.

**Secondary (output):**
- `adapters/secondary/memory_tile_storage/` — the tile map. A preallocated `[]uint16` indexed by tile id, with country codes interned into a side table (2 bytes per tile — ~2 MB for a 1M-tile map). Fans updates out in process and persists to a local snapshot file.
- `adapters/secondary/in_memory_tile_checker/` — validates tile IDs
- `adapters/secondary/in_memory_country_checker/` — validates country codes (hardcoded)

Beyond the `domain.TileStorage` port, `memory_tile_storage` also exposes `Subscribe(ctx) (<-chan domain.TileUpdate, error)`, one call per open stream.

### Key Flow

```
POST /session.v1.SessionService/CreateSession
  → SessionService
  → session_service (attests, then mints)
  → turnstile_attester → Cloudflare siteverify
  → kernel/session.Signer.Mint [HMAC over expiry+id+IP; nothing stored]

POST /planet.v1.ClickService/Click   [X-Session-Token: <the minted token>]
  → VPNBlockInterceptor, SessionInterceptor, RateLimitInterceptor, AntiBotInterceptor
  → ClickService
  → ClickHandlerService (validates tile ID + country)
  → MemoryTileStorage.Set() [writes the tile, fans the update out in process]
  → every subscriber: one per open ListenForEvents stream
```

`Set` is a no-op when the tile already holds that value — no write, no update published.

```
POST /chat.v1.ChatService/SendMessage
  → BlocklistInterceptor, then RateLimitInterceptor (both kernel/connectutil)
  → ChatService
  → chat_service (sanitizes, stamps id/time/tag)
  → MemoryChatStorage.Append() [appends to the JSONL log, then fans out]
  → every subscriber: one per open ListenForEvents stream
```

A failed log write fails the whole post: the log is the audit trail, so a message nobody can account for later is not one that gets broadcast.

### Chat (`internal/chat/`)

Chat is a separate bounded context, not a feature of the tile game: it shares the process, the transport and the country list, and has its own proto package, domain, storage and edge. Nothing under `internal/chat/` imports `internal/clicks/`, and the reverse holds too.

**Off by default.** With `chat.enabled` false nothing is registered, so `/chat.v1.ChatService/` answers 404 — the unauthenticated public write endpoint does not exist at all rather than existing and erroring.

**Identity without accounts.** A client picks its own display name and sends a UUID it persists locally. **Neither is trusted for anything** — anyone can post with any name. What a sender cannot forge is `author_tag`: a salted hash of their IP, 6 hex characters, so two people using the same name still look different and a mute has a key that means something. The salt is `chat.service.tagSalt`; left empty it is regenerated at boot, which changes everyone's tag on restart, and the server warns about it.

**Abuse controls live at the edge**, in two interceptors — ahead of decoding the message and well ahead of validating it, so a flood of malformed messages costs a sender exactly what a flood of well-formed ones does. `chat.blockedIPs` cuts an address off from every chat RPC; `chat.rateLimiter` throttles `SendMessage` alone (`GetHistory` is one read on join, and limiting it would punish a page load). **Both are `kernel/connectutil`'s**, the same ones the click chain uses — see [Shared interceptors](#shared-interceptors). The domain then bounds the message in **runes** (280) and the name (24), validates UTF-8, and **strips control characters** — a newline would otherwise let a sender forge a line in the JSONL log.

Refusal reasons are logged, never returned: a sender learns *that* they were refused, not which check tripped. **The stored text is raw — the frontend must escape it.**

The **vendored VPN lists** do not cover chat: `NewVPNBlockInterceptor` wraps `Click` alone. Chat's blocklist is the same `*ipblock.Blocklist` type, built by `ipblock.NewDenyList` from config prefixes instead of vendored data — so entries are CIDRs and a `/24` is one line rather than 256. Extending the vendored lists to chat is therefore a wiring change (build the list in `describeModules` and hand it to both modules, the way the country list already is), not a second list to write.

**The log is an append-only JSONL file**, not the tile snapshot's whole-state codec: different shape, different write pattern. One line per message with `at`, `id`, `name`, `tag`, `authorId`, `country`, `ip`, `userAgent`, `text`. It is fsynced every `flushInterval` rather than per message (a hard kill loses at most that window — the same bargain the snapshot makes), pruned hourly past `retention`, and its tail repopulates the in-memory history at boot so a restart does not blank the chat. Corrupt lines are skipped and reported, never fatal.

That log holds **personal data** — IPs next to user-authored text — so the retention window is a policy decision rather than a cache size. It lives on the `tile_state` volume, which the droplet's weekly disk backup already covers.

**Chat has its own stream**, `ChatService.ListenForEvents` — see [The live streams](#the-live-streams). It replaced a `/ws/chat` websocket that had to be kept apart from the tile one because frames carried a bare protobuf message with no type tag: a second payload on either socket would have been indistinguishable from the first. The `oneof` envelope is exactly what removes that constraint.

**Sending is an RPC, not a read on the socket**: both publishers lean on `CloseRead` for instant disconnect detection, and the RPC path already has the middleware stack and the interceptors.

`GetHistory` is marked `NO_SIDE_EFFECTS`, so Connect sends it as a GET — but it answers `Cache-Control: no-store`, the opposite of `GetMap`. A client fetches it once on join to seed what the stream then keeps up to date, so a cached answer would show a joiner a chat missing the last few minutes.

### Rate limiting

`NewRateLimitInterceptor` throttles **`Click` only**, per source IP, from a `kernel/ratelimit` token bucket — 1 click/s with a burst of 10 by default (`rateLimiter.*`). `MapDensity` and `GetMap` are cacheable reads a proxy in front absorbs; limiting them would punish a page load rather than a bot. A refused click answers `CodeResourceExhausted`, i.e. HTTP 429, and never reaches the domain.

#### Saying what is left

The web app shows the player how many clicks they have in hand, and **the server is the only thing that knows**. A client running its own copy of the bucket would drift within seconds — it cannot see the clicks the same address makes from another tab, and its idea of when a click was spent is a round trip out of date.

Polling for it would be worse, so nothing polls. `Limiter.Take` returns the bucket's state alongside its verdict, and the state carries the **policy** (`Capacity`, `PerSecond`) as well as the reading: given both, a client replays the same refill arithmetic between two answers and is exact without asking. The pip count and the fill rate on screen are therefore the server's burst and refill rate — **changing `rateLimiter.*` changes the display with no frontend release.**

That reading travels two ways, because a refused call has no response message to put it in:

- an allowed call gets it on the context (`ctxutil.AddClickBudgetToContext`), and `ClickService.Click` puts it in `ClickResponse.budget`. The interceptor decides the policy; the handler decides how to say it — the same split `NewSessionInterceptor` already uses for the session id.
- a refused one gets it as a **connect error detail**, built by the `describe` function each context passes. `planetv1controller` passes one; chat and sessions pass nil, because nothing displays those allowances.

`ClickService.GetBudget` covers the cold start — a client that has just loaded and has no click to learn from. It reads through `Limiter.Peek`, which spends nothing and, for an address that never clicked, **creates no bucket**: reading an allowance must not be a way to make the limiter remember a caller. It is deliberately not `NO_SIDE_EFFECTS`, so it is a POST no cache will serve a stale answer to; every click re-anchors the client afterwards, so it is asked once per page load.

**This tells a scripted clicker exactly when to fire**, which is a real cost against [Anti-bot](#anti-bot-internalantibot). It is a small one — a script can already infer the same schedule by counting its own 429s — and it is paid to stop honest players being refused with no warning.

The bucket key is whatever `IPReaderMiddleware` put on the context: `X-Real-IP` if present, otherwise the peer address. **The reverse proxy must set that header itself** — `deploy/vps/Caddyfile` does, with `header_up X-Real-IP {client_ip}` on every backend route. Merely forwarding it would let a client send its own and buy a fresh bucket per request. The fallback is the peer address rather than a constant precisely so a missing header degrades to per-connection buckets instead of rate limiting the whole game as one player.

Chat and sessions each have **their own limiter instance** with their own budget, because what each call costs has nothing to do with what a click costs:

- `chat.rateLimiter` — one message every 3s, five in hand. A message fans out to every connected client and lands in a log everyone will read.
- `session.rateLimiter` — one mint every 30s, ten in hand. A mint costs a siteverify round trip to a third party, so an unthrottled `CreateSession` is a free way to spend this server's Turnstile quota.

### VPN blocklist

`NewVPNBlockInterceptor` refuses **`Click` only**, with `CodePermissionDenied` (HTTP 403), when the source address falls in a vendored VPN range. It sits **outside the rate limiter** in the interceptor chain, deliberately: a refused address must not also spend a token, or the next click would come back 429 and the web app would show the throttle dialog instead of the VPN one.

**It exists because of the rate limiter, not instead of it.** The bucket is keyed on an address, and a commercial VPN is the cheapest way to get a fresh one; refusing those addresses is what makes the bucket hold. It raises the floor rather than closing the door — residential proxies appear in no public list, and nothing here stops one. **That gap is what [Sessions](#sessions-internalsession) closes**, by requiring something an address cannot buy; chasing list completeness instead is a treadmill.

Reads and the streams are untouched. A VPN user still loads the planet and follows it live; they cannot paint. That is also what keeps a false positive readable: the page works and says why, instead of failing to load.

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

**Reads and the streams are untouched.** A visitor loads the planet, watches it live and reads the chat without ever minting anything; a session is only ever needed to paint. `GetMap` is a cacheable GET and a per-session header on it would defeat that cache.

**Minting has its own throttle** (`session.rateLimiter`, one every 30s with 10 in hand). A mint costs a siteverify round trip to a third party, so it cannot share the click budget: unthrottled, the endpoint is a free way to spend this server's siteverify quota.

**`kernel/turnstile` fails closed on everything.** A network error, a non-2xx, a body that is not JSON, a token for another action or another hostname are all refused exactly as a forged one is. Failing open would make the check decorative — an attacker who can reach the backend can also make siteverify unreachable from it. It validates `action` and `hostname` as well as `success`, because **the sitekey is public**: without those two checks a token minted by the same widget embedded on any other page would be accepted here.

**`session.turnstile.enabled: false` mints for anyone who asks** (`open_attester`). That is how a local backend runs without a widget and a secret, and it still exercises the whole click path — the token is bound and expires. It is never the production choice, and the server warns at boot when it is on.

**Two secrets, neither in git.** `session.secret` signs the tokens; anyone holding it can mint one the API accepts. `session.turnstile.secret` is the widget's secret half. Both come from the environment via `deploy/vps/docker-compose.yaml`, as `chat.service.tagSalt` does. An empty `session.secret` generates one at boot and warns — which invalidates every session in flight on each restart, costing every player one extra round trip.

**What it does not stop:** a person who solves Turnstile in a real browser and then runs a userscript. They hold a genuine session, and nothing here distinguishes them from a player. This raises the floor from "twenty lines of Python" to "drive a real browser"; the signal that survives that is behavioural — timing regularity and tile-id structure over a session — which is what the session id on the context exists to be keyed on. Nothing in production reads it yet, so `ctxutil.GetSessionID` sits behind the `testing` tag (see [Testing](#testing)) until something does.


### Anti-bot (`internal/antibot/`)

What is left after sessions. A player who solves Turnstile in a real browser and
then runs a userscript holds a genuine session, and no address- or token-based
check can tell them from a player. The signal that survives is **behavioural**.

**Detection and consequence are separate, and the consequence is the boring
half.** `antibot/shadowban` takes a scope and a clock and runs a ban. It knows
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

**It sits innermost, after the throttle** — the opposite of the blocklist and the
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

Keyed on `ipscope.Of`, the same unit as the throttle, so a v6 caller cannot serve
a ban on one address and click from the next in its own /64. It only bites a bot
with a stable address — against a residential proxy pool it evaporates for
exactly the reason the rate limiter does.

`memory_tile_storage.Owner` exists for this: one indexed read under the existing
lock, declared as a local port in the controller the way `DenseMapReader` is.

### Shared interceptors

`kernel/connectutil` holds the two interceptors both contexts need, because the policy is the same whatever the procedure is — only the procedure names and the wording of the refusal differ, and those are arguments:

- `NewRateLimitInterceptor(limiter, refusal, describe, procedures...)` — a `kernel/ratelimit` bucket keyed on the context IP, answering `CodeResourceExhausted` (429). `describe` is optional and is what makes the allowance visible — see [Saying what is left](#saying-what-is-left); chat and sessions pass nil
- `NewIPBlockInterceptor(blocklist, refusal, onBlocked, procedures...)` — an `ipblock.Blocklist` lookup answering `CodePermissionDenied` (403), with an optional hook the click counter hangs on
- `NewSessionInterceptor(verifier, clock, refusal, enforce, onVerdict, procedures...)` — a `kernel/session` signature check answering `CodeUnauthenticated` (401), which puts the session id on the context and, with `enforce` false, counts without refusing
- `NewErrorInterceptor(logger, mapper)` — the one that keeps an unexpected error's cause off the wire. It is a full `connect.Interceptor` rather than a `UnaryInterceptorFunc`, so it covers the streaming handlers too; without that, a stream would be the one procedure whose raw error the caller sees. Each context passes the `Mapper` naming the domain errors it wants translated, and returns nil from it for anything it does not recognise.

Each context keeps a thin named constructor over these — `planetv1controller.NewRateLimitInterceptor`, `NewVPNBlockInterceptor` and `NewErrorInterceptor`, `chatv1controller.NewRateLimitInterceptor`, `NewBlocklistInterceptor` and `NewErrorInterceptor` — which is where the procedure list, the refusal wording, the domain errors and the metric live. **A context names its own policy; neither reimplements the mechanism.**

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

### Kernel (`internal/kernel/`)

Shared infrastructure: `bootstrap` (the composite layer), `cfgutil` (YAML + env config via koanf), `httpserver` (middleware, formats), `logging`, `prom` (Prometheus), `xtime`, `ctxutil`, `ratelimit`, `ipblock`, `atomicfile`, `secrets`.

`bootstrap` runs the modules — see [The composite layer](#the-composite-layer). It knows nothing about this game: it takes a list of modules, builds each one under a startup deadline, mounts what they claimed on one router, and serves until the process is signalled.

Two of these are here because both bounded contexts need them and neither should depend on the other:

- `secrets` — the random hex a config may leave it to the server to invent. Chat's tag salt and the session signing key are the two, and both pay the same price for an empty setting: what the old one covered stops being recognised on restart.
- `atomicfile` — temp file, fsync, rename, fsync of the directory. Written for the tile snapshot; the chat log's retention rewrites need the same guarantee, and duplicating 80 lines of carefully-written fsync/rename code is how the two drift apart. Covered by the existing snapshot tests.

`session` mints and verifies the click token — see [Sessions](#sessions-internalsession). `turnstile` is the siteverify client it is fed by; both are in the kernel because the session context mints with them and the clicks context verifies with them, and neither context may depend on the other.

`ipblock` is the VPN prefix set — see [VPN blocklist](#vpn-blocklist). `ratelimit` is a keyed token bucket held in this process, like the tile map it protects — with one API instance, a shared counter would buy nothing. Its `Run` loop periodically forgets the buckets that have refilled to capacity, which is free: such a bucket holds exactly what a freshly created one would, and without it the map would keep an entry per address that ever clicked.

`ipscope` decides what a bucket is keyed on, and every throttle goes through it. Over IPv4 that is the address; over IPv6 it is the surrounding **/64**, because the smallest allocation a subscriber receives is a /64 and most receive far more — a bucket per v6 address is one the same line walks out of by picking its next address, turning one home connection into thousands of callers with a throttle each. The session token binds to the same unit, so the address a token is valid for and the address that spends a budget cannot diverge. Blocking deliberately does **not** use it: the VPN and datacenter lists are precise prefixes already, and widening a hit to the surrounding /64 would refuse neighbours who are not on them.

### Configuration

Config is loaded from a YAML file (`-config` flag), with environment variables overriding it — `cfgutil` uses `.` as the nesting delimiter, so `tilesStorage.snapshotPath=/data/tiles` in the environment overrides the file. See `cmd/api/example.yaml` for the full schema.

**Each module owns its own config struct** — `clicks.Config`, `chat.Config`, `session.Config` — and `app.Config` is the three of them plus `httpServer`. The clicks keys stayed at the top level of the file rather than moving under a `clicks:` section: `app.Config` squashes that struct (`koanf:",squash"`), so the file and every `deploy/` environment variable are unchanged.

- `httpServer.bindAddress` — the encoding is negotiated per request, so there is no format setting.
- `httpServer.streamHeartbeat` — how often a silent live stream sends a heartbeat (default 30s). **Must stay well under the proxy's idle cut**: Cloudflare answers 524 at ~125s, and a stream that never speaks is one it kills.
- `gameMap.maxIndex` — total number of tiles
- `tilesStorage.snapshotPath` — where the state is persisted; **empty disables durability**
- `tilesStorage.snapshotInterval` — how often a changed state is flushed
- `tilesStorage.subscriberBuffer` — per-subscriber channel capacity, which is now per connected client rather than per fanout; updates for a subscriber that cannot keep up are dropped, not blocked on
- `rateLimiter.perSecond`, `rateLimiter.burst`, `rateLimiter.sweepInterval` — the per-IP click throttle (defaults 1/s, burst 10, swept every minute)
- `vpnBlocklist.enabled`, `vpnBlocklist.includeDatacenters`, `vpnBlocklist.allow` — the VPN refusal (see [VPN blocklist](#vpn-blocklist)); disabled parses nothing and allocates nothing
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
