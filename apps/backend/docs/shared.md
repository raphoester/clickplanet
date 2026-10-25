# Shared (`internal/shared/`)

Shared infrastructure: `cpbootstrap` (the composite layer), `cpredis` (the event bus over Redis Streams, which each module connects to on its own — see [Telling other modules what happened](composite-layer.md#telling-other-modules-what-happened)), `cpoutbox` (the outbox a store writes in its transaction, and its relay — see [What may not be lost](composite-layer.md#what-may-not-be-lost-the-outbox)), `cpcountries`, `cpconfigs` (YAML + env config via koanf), `cphttpserver` (middleware, formats), `cpprom` (Prometheus), `cptime`, `cpctx`, `cpconnect`, `cpratelimit`, `cpipblock`, `cpipscope`, `cpsession`, `cpsessionverifier`, `cpsecrets`, `cppg` (postgres — see [Durability](durability.md)), `cpcolls` (collections), `cpzernio` (the client of the posting service), `cpanthropic` (the client of Anthropic's API).

**Every package here is prefixed `cp`, and a new one must be.** A call site reads
`cptime.SystemClock{}` or `cpctx.GetSourceIP(ctx)`, so the prefix says the
dependency is this layer's without the reader going to the import block — and an
unprefixed name in a module is a module's own package by construction. It also
settles the collisions a shared layer attracts: `cptime` beside stdlib `time`,
`cpsession` beside the session *module*, `cpconnect` beside `connectrpc.com/connect`.
`cpsession` is what let the old `internal/session/module.go` drop the `sharedsession`
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

`cpbootstrap` runs the modules — see [The composite layer](composite-layer.md). It knows nothing about this game: it takes a list of modules, builds each one under a startup deadline, mounts what they claimed on one router, and serves until the process is signalled.

Two of these are here because both bounded contexts need them and neither should depend on the other:

- `cpsecrets` — the random hex a config may leave it to the server to invent. The player module's tag salt is the only one left, and it costs nothing: the tag never leaves memory. The click token's key is not invented, because a key the server made up could not verify what another restart had minted — see [Sessions](sessions.md).

`cpsession` mints and verifies the click token — see [Sessions](sessions.md). It is here because **both** contexts read it: `auth` mints with its `Signer`, `planet` verifies with its `Verifier`, and neither may depend on the other. The two halves are separate types over separate config, so what each context can do with it is decided at compile time.

The siteverify client it is fed by is **not** here. `turnstile` sat here on the same "both contexts need it" rule, but only one ever did, so it now lives at `session/internal/turnstile` where the compiler keeps it. **The bar is not that a package is shareable, it is that it would read the same in any other program and that two modules actually import it** — "shared" names the symptom, and a directory admitted on the weaker reading becomes a dumping ground. `cpsecrets` passes narrowly — chat is its only caller today, but it is twenty lines of `crypto/rand` with no domain in it at all.

`cpsessionverifier` is the exception to that bar: it knows `auth.v1.InternalService`, so it would not read the same in another program. It is the `cpconnect.SessionVerifier` that `planet`, `player`, `chat` and `seasons` build, which takes the key from `auth` once — see [Sessions](sessions.md). It is here because the alternative was one copy in each module, and a fix to one copy missed the others.

`cpcountries` is the ISO country list both the tile game and the chat validate against. `cpipblock` is the VPN prefix set — see [VPN blocklist](vpn-blocklist.md). `cpratelimit` is a keyed token bucket held in this process, like the tile map it protects — with one API instance, a shared counter would buy nothing. Its `Run` loop periodically forgets the buckets that have refilled to capacity, which is free: such a bucket holds exactly what a freshly created one would, and without it the map would keep an entry per address that ever clicked.

`cpipscope` decides what a scope's bucket is keyed on, and every throttle goes through it. Over IPv4 that is the address; over IPv6 it is the surrounding **/64**, because the smallest allocation a subscriber receives is a /64 and most receive far more — a bucket per v6 address is one the same line walks out of by picking its next address, turning one home connection into thousands of callers with a throttle each. The session token binds to the same unit, so the address a token is valid for and the address that spends a budget cannot diverge. Blocking deliberately does **not** use it: the VPN and datacenter lists are precise prefixes already, and widening a hit to the surrounding /64 would refuse neighbours who are not on them.
`cpzernio` is Zernio's HTTP API as Go: a generated client, the vendored OpenAPI document it is generated from, and the two files that configure the generation, all in the one directory, as `cpipblock` keeps its lists in `cpdata`. **Nothing in it is written by hand but the two config files**: `make zernio-spec` refreshes `openapi.json`, `make zernio` writes `client.gen.go`. It knows nothing of clips or of this game, which is why it is here and not in the module: the clips job's `posts/zernio` adapter is its one caller today, and holds everything that is ours (see [The clips job](clips-job.md)). The linters, gofumpt and `make deadcode` leave `client.gen.go` alone by its generated header, not by its path.
`cpanthropic` builds the client of Anthropic's API (`NewClient`, from a key and, for a test, a base URL; no key is
`ErrNoKey`) and says what a failed call went wrong with, the request left out (`Redacted`): the SDK's error dumps
the request it failed on, key and all. **It knows no model, no prompt and no timeout**: those are each caller's, set
per request, so one client serves calls of any length. The clips job's editor is its one caller today.
`cpcolls` holds the collections the standard library does not — today `Set[T]`. **A set is a `*cpcolls.Set`, never a map.** A `map[T]struct{}` or a `map[T]bool` written `m[k] = true` fails `make lint`: a ruleguard rule in `tools/ruleguard/rules.go`, run by gocritic, reports both everywhere but in `cpcolls` itself. A nil `*Set` reads as empty, like a nil map, so a lookup in a map of sets needs no `ok` check.
