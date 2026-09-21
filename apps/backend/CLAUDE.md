# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests. The postgres store tests start a postgres:16-alpine container, so Docker must be running
make test
# or: go test -tags testing ./... | grep -v 'no test files'

# Run a single test
go test -tags testing ./internal/planet/internal/clicks/usecases/click_usecase/... -run TestName

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

### Four bounded contexts, one process

- **`internal/planet/`** — the tile game: clicks, ownership, the map, the update stream.
- **`internal/chat/`** — the live chat: messages, identity, retention.
- **`internal/auth/`** — who a caller is and what it has to prove before it may click: Turnstile, accounts, the click token. See [Auth](#auth-internalauth).
- **`internal/player/`** — what the game keeps about one account: the name it chose, the tiles it took, its daily streak. See [Player](#player-internalplayer).

They share the process, the transport and the country list, and **nothing else**. None imports another; each owns its own domain types, its own proto package, its own storage adapter (where it has one) and its own edge.

**A module never calls another module's code or reads its data in its own stack trace.** When one needs an answer from another it asks over Connect, on a loopback listener — see [Calling another module](#calling-another-module). Three do: `planet`, `player` and `chat` take the click token's verifying key from `auth`, `player` asks `auth` whether an account is linked, and `chat` asks `player` who posts: the username, or the guest code. When one only has to say that something happened, it publishes an event in process and never learns who listens — see [Telling other modules what happened](#telling-other-modules-what-happened). `planet` publishes `TileTaken` and `auth` publishes `AccountDeleted`, `SignedIn` and `SignedOut`; `player` hears all four.

Bonus boxes live inside `internal/planet/` rather than beside it: what they grant
is click allowance, and what carries them is the planet stream. A context of
their own would have to import both.

**`internal/antibot/` is a domain library, not a fifth context.** It has no proto
package, no adapters and no `module.go`, and it cannot be wired without a caller
composing it — the clicks edge does, the way it gates on `auth`. It is not a
shared package because "is this caller a bot" is the business this game is in,
while `shared` is for things that would read the same in any other program.
`planet` → `antibot` is the only module-to-module import in the backend.

#### A module publishes its root package and hides the rest

Every module's interior lives behind **its own `internal/`** — `internal/planet/internal/clicks`,
`internal/chat/internal/messages`, `internal/antibot/internal/jury`. Go's own
rule does the enforcing: such a package is importable only from the tree rooted
at the parent of that `internal`, so `chat` importing `planet/internal/clicks`
**does not compile**. There is no linter to run, no allowlist to maintain and
nothing to keep in sync.

So the whole of what one module may use of another is **what sits in the other's
root package**:

| module | its public API |
|---|---|
| `planet` | `Config`, `NewModule` |
| `chat` | `Config`, `StorageConfig`, `NewModule` |
| `auth` | `Config`, `NewModule`; behind the `testing` tag, `NewModuleWithFakeProviders` and the fake's types |
| `player` | `Config`, `NewModule` |
| `antibot` | `Config`, `Observer`, `Guard`, `New`, `Description`, `Click`, `Report`, `Sentence`, `Examination`, `Reading` |

That holds for `cmd/api` too: the composition root lists modules and cannot
reach a domain type, a storage adapter or a controller even if it wanted to. A
module's `Config` may carry a field whose *type* is internal (`planet.Config.TilesStorage`
is `inmemory_tile_storage.Config`) — koanf fills it by reflection and a caller can
still set its fields, it just cannot name the type. That is the right amount of
access: the settings are published because they are in the file, and the code
that reads them is not.

**Adding a package inside a module is therefore free, and taking one out of
`internal/` is a deliberate act** that shows up in review as exactly one moved
directory.

### Every id is a type of its own

**An id is never a bare `uuid.UUID`, `string` or `[]byte` in a signature.** Each kind of id is its own named type, declared beside the entity it names: `accounts.AccountID` (`type AccountID uuid.UUID`) and `accounts.TokenHash` (`type TokenHash []byte`, which names a session). A function that asks for an account then says so, and passing a session's hash or any other uuid where an account is asked for **does not compile**. With bare types, `DeleteSessions(ctx, account)` and `DeleteSession(ctx, tokenHash)` differ by one letter and nothing checks which one a caller meant.

- **The only conversions are at the edge of the process.** A storage adapter converts to the driver's type on the way out and back on the way in (`uuid.UUID(account)` in `postgres_account_store`; lib/pq cannot take a named array), a handler to the wire's type (`account.ID.String()`), and the token codec to bytes. The id type has no `Scan` or `Value` of its own.
- **An id is never unwrapped to hand it on.** `uuid.UUID(account)` in a use case, to call a port or a shared package, throws away what the type was for. The id that crosses modules is declared where it crosses: `cpsession.AccountID` sits with the click token, which `auth` mints and `planet` reads, and `accounts.AccountID` is an alias of it, so an account reaches `Mint` and comes back in `Claims.Account` as the same type. `cpsession.NoAccount` is a token minted for nobody.
- **A new id starts as a type.** Adding one later means touching every signature it already flows through.

### Manipulators are verbs, builders are nouns

**A name says whether calling it changes anything.** From Elegant Objects:

```
class Document {
    OutputPipe output();          // builder: a noun, reads, changes nothing
}

class OutputPipe {
    void write(InputStream stream); // manipulator: a verb, does the work
    int bytes();
    long time();
}
```

- **A manipulator performs an action and changes state**: it writes a row, sends a request, spends a token, mutates its receiver or an argument. It is named from a verb — `SaveSignIn`, `DeleteSessions`, `Session.Extend`, `Grant`, `Execute`.
- **A builder only reads or computes, and changes nothing** — not its receiver, not an argument, not the world. It is named from a noun, with `With`, `As` or `Of` where that reads better — `accounts.Cookie`, `accounts.ExpiredSessionCookie`, `store.Session(ctx, hash)`, `accounts.OutcomeOf`, `Flow.Challenge`, `Config.WithDefaults`, `oauth_http.Resource[T]`.
- **A verb on a pure function is a bug in the name.** `ClearCookie()` that only returns a `Set-Cookie` string reads as if it cleared something; it is `ExpiredSessionCookie()`, and the caller is what sends it. Likewise a read port is `Session(ctx, hash)`, not `FindSession`.
- **A builder that returns a boolean is an adjective** — `empty`, `readable`, `negative`: `Account.Linked()`, `Providers.Off()`, `Client.Configured()`, `Account.linkedTo(provider)`. Not `IsLinked`, `HasProvider` or `holds`. And the comma-ok second result of a builder (`CookieValue(…) (string, bool)`) is Go's idiom, not a name.
- **A manipulator returns no answer about what it did.** Ask first with an adjective, then act: `if session.Extendable(now, lifetime)`, then save `session.Extended(now, lifetime)`. Not `ExtendIfDue(...) bool`, which both acts and answers.
- **Prefer immutable values: a builder returns a changed copy, it does not change its receiver.** `session.Extended(now, lifetime)` is a new `*Session` with the new expiry, and `session` is left as it was; the only manipulator left is the store's `SaveSession`, which writes the copy. A method that sets a field on a domain value is the last resort, not the default.
- **A builder that returns an error is still a noun**: `Session.ExpiryError(now)`, `Flow.CallbackError(state, now)`, `Config.pruneError()`.
- **Two exceptions, both imposed from outside:** `Validate() error` is the hook `cpconfigs` calls by name, and a generated Connect handler method carries its RPC's name (`GetMe`).
- **Go's `New…` constructors stay** — building an object is what they say.

The auth module follows this. Code outside it has not been checked against the rule yet: fix a name when you touch the code, not in a sweep.

### The composite layer

Each context wires **itself**, in a `module.go` at its root (`internal/planet/module.go`, `internal/chat/module.go`, `internal/auth/module.go`, `internal/player/module.go`). That file is the context's manifest: its `Config`, whether it is on, and its DI sequence. **A module takes its config and nothing else, and builds every object it needs itself** — there is no `Deps` struct and nothing is handed down from `main`. A module is a `cpbootstrap.Module` — a name, an `Enabled` flag and a DI sequence — and the sequence is handed a `cpbootstrap.Props` carrying registrars and nothing else:

- `props.RPC.Mount(build, interceptors...)` — the module hands over what *builds* the handler, plus the interceptors it wants. `cpbootstrap` builds it, so it can put its own interceptor outside every module's — see [The error net](#the-error-net)
- `props.AdminRPC.Mount(build, interceptors...)` — the same, for an operator service: served only on the loopback admin listener — see [Operator tools](#operator-tools-adminservice)
- `props.InternalRPC.Mount(build, interceptors...)` — the same again, for what other modules call: served only on the loopback internal listener
- `props.Internal.Dial()` — the HTTP client and base URL a generated `New<Service>Client` takes, to call another module — see [Calling another module](#calling-another-module)
- `props.Events` — the in-process event bus: `props.Events.Publish(message)`, and `cpbootstrap.Subscribe(props.Events, name, buffer, handler)` for the runner that delivers one type — see [Telling other modules what happened](#telling-other-modules-what-happened)
- `props.Runners.Add(runner)` — a `Runner` (`Name()` and `Run(ctx)`), run as a goroutine with the process-lifetime context
- `props.Closers.Add(name, close)` — a cleanup, run in reverse registration order
- `props.Logger`, `props.Metrics`
- `props.Server` — the bind address and the stream heartbeat, **the only config a module reads that is not its own**. It is the transport every module answers over, so it belongs to the layer that owns the server rather than to any context.

**A constructor builds and a `Load…` method reads.** Nothing that reads a file or parses a blob happens inside `New`: the DI sequence calls `New`, then the load, one line each — `embedded_geodesic_map.New` then `LoadGeography`/`LoadBorders`, `inmemory_tile_storage.New` then `LoadSnapshot`, `cpipblock.New` then `Load`, `antibot.New` then `LoadState` (which connects to its schema too), `inmemory_ledger_storage.New` then `Load`.

A module never sees the router, the signal handler or another module's objects. **There is no mutable app object and nothing to leave a half-built dependency on**: `internal/shared/cpbootstrap` builds every module in order, then serves.

**Shutdown goes in this order**: end every open stream, `http.Server.Shutdown` (the public server, then the admin and internal ones — a public call in flight may still be waiting on an internal one), the closers in reverse order, then cancel and wait on the runners. The first step is the drain interceptor — see [Ending the streams on shutdown](#ending-the-streams-on-shutdown). There are no storage closers left: the tile map, the ledger, bans and evidence all flush from their runners, after the closers, once the server has stopped taking writes.

**`cmd/api/main.go` is the composition root, and it is the only one** — there is no `internal/app`, because a package whose whole job is to be called once by `main` was a level of indirection and nothing else. It does two things: load the config, and list the modules. It builds no objects, derives nothing, and reads inside no block.

**The aggregation is the whole of it — a flat slice, no branches, no dependencies threaded through:**

```go
return []bootstrap.Module{
	auth.NewModule(config.Auth),
	planet.NewModule(config.Planet),
	chat.NewModule(config.Chat),
	player.NewModule(config.Player),
}
```

**Every module is always listed; a module with a switch reads its own.** `NewModule` sets `Enabled` and `cpbootstrap` skips the ones that are off, so turning auth off is a config change and never an edit here. `planet`, `chat` and `player` have no switch and are always on; `auth` has one. A disabled module is never built, so its routes are **absent** rather than present and refusing — `/auth.v1.AuthService/` 404s.

#### What two contexts need, without either handing it to the other

`main` builds no objects at all, so a thing two contexts need is **a config block they both declare**, and each builds its own instance from it.

- **`shared/cpsession`** is the click token. The `auth:` block is read by two contexts, but they share no key and no type: `cpsession.SignerConfig` carries the Ed25519 seed and only `auth.Config` declares it, while `planet.Config` declares `cpsession.VerifierConfig`, which is two switches and **no key at all**. `auth` builds a `*Signer`, `planet` a `*Verifier`, **which has no `Mint` on it**. Under the old shared `Config` both built a `*Signer` off one HMAC secret, so `planet` held a live minting object and nothing but discipline stopped it using it.

  **`planet` asks `auth` for the key** over the internal listener, rather than reading one — see [Calling another module](#calling-another-module) and [Sessions](#sessions-internalauth). So there is one key in the whole file, `auth.secret`, nothing derived for a human to paste, and nothing to keep in step with it.

  Neither module imports the other's packages, and **the planet context knows nothing about Turnstile** — the siteverify client lives at `auth/internal/attestation/turnstile`, so it *cannot* reach it.
- **`shared/cpcountries`** is the ISO list. It is stateless and hardcoded, so each module just calls `cpcountries.New()`, the way it calls `cptime.SystemClock{}`. It sits in `shared` and not under `planet/internal/clicks/` for exactly the reason that layer exists: neither context may depend on the other — and now could not, since that directory is unreachable from chat.

**This is why `auth.secret` is now required** rather than invented at boot — see [Sessions](#sessions-internalauth).

#### Calling another module

**Over Connect, on a loopback listener, never in the caller's stack trace.** A module that other modules call mounts an internal service with `props.InternalRPC.Mount`; `cpbootstrap` serves it on `httpServer.internalBindAddress` alone, which must be loopback. The caller builds a generated client over `props.Internal.Dial()`.

| caller | asks | for | through |
|---|---|---|---|
| `planet`, `player`, `chat` | `auth.v1.InternalService/GetVerifyingKey` | the public half of the click token key, once per boot | each its own `rpc_session_verifier` |
| `player` | `auth.v1.InternalService/GetAccount` | whether an account is linked, on each `SetName`; when it was made, on each `GetPlayer` | `players/rpc_account_reader` |
| `chat` | `player.v1.InternalService/GetAuthor` | the name a sender is shown under, on each `SendMessage` | `messages/rpc_player_authors` |
| `chat` | `player.v1.InternalService/GetAuthors` | who everyone in the window is, once per `GetHistory` | `messages/rpc_player_authors` |

A module cannot import another's interior, so `player`'s and `chat`'s `rpc_session_verifier` are copies of `planet`'s.

- **Each module's data stays in one place.** The caller holds an address, never the other module's config block, pool, objects or root package. The seed is read by `auth` and nothing else.
- **`Dial` is the one place that knows the transport is loopback HTTP.** Moving to unix sockets changes the listener and `internalDialer`, and no module.
- **`Dial` fails with no internal listener**, and the caller reports it. An internal service with no listener is not served, and logged, like an admin one.
- **The error net and the drain wrap internal services too.**
- **What travels is a public key**, which is the point: a holder can check a token and cannot mint one, so the hop carries nothing worth stealing. Under the HMAC this replaced there was no such thing to send.
- `TestAModuleCallsAnotherOverTheInternalListener` pins the path, `TestAnInternalServiceIsNotOnThePublicRouter` pins that Caddy cannot reach it.

The proto sits beside the public one in the module's package (`proto/auth/v1/internal.proto`), as `admin.proto` does, and the frontend generates it too without using it.

**The loopback rule is for synchronous calls, and it exists to stop one module loading another's config.** A caller that needs an answer has to reach the module that holds the data. Done in process, that is how `planet` once built a signer off `auth`'s secret and how a key ends up duplicated across blocks in the file. The listener makes the caller hold an address and nothing else.

#### Telling other modules what happened

**Over Go channels, in process.** There is no broker, and a modular monolith still needs messaging. An emitter publishes an event and never knows who listens, so it has no reason to load a listener's config, and the problem the loopback rule solves does not arise. A loopback stream would buy nothing here: it adds encoding, reconnects and an address the subscriber must hold, and it loses events on a crash exactly as a channel does.

- **The payload is a proto-generated message**, declared in the emitter's proto package (`proto/<name>/v1/events.proto`). A subscriber imports `generated/proto`, never the emitter's root package, which would also hand it the emitter's `Config`. The proto is also the schema, if AsyncAPI or a real broker ever comes.
- **Each subscriber gets its own copy.** A proto message is a pointer: the emitter builds a fresh one per event and never touches it again, and nothing shared is mutated.
- **Each subscriber reads its own buffered channel, in its own goroutine.** No subscriber code runs in the emitter's stack trace, so a click never waits on a listener.
- **A full buffer drops the event and counts it.** Delivery is at most once and not durable: a crash or a restart loses what was in flight. A flow that cannot lose an event (money, rewards) needs a durable path of its own, not this.
- **Subscribers register while modules are built**, before any runner starts, so no event is published to nobody at boot.

**`cpbootstrap` holds the bus** (`events.go`), and a module reaches it as `props.Events`:

- **Publish** takes any proto message and never blocks: a non-blocking send to each subscriber of that message's type. The first subscriber gets the message, each other one a `proto.Clone`.
- **`cpbootstrap.Subscribe[T](props.Events, name, buffer, handler)`** registers a channel of `buffer` events for type `T` and answers the `Runner` that reads it and calls `handler.Handle(ctx, event)`. The module adds that runner. A second subscriber of one type with the same name refuses the boot, and so does a zero buffer.
- **Registering after the runners start is refused.** `Run` seals the bus once every module is built. An event published while modules are built (or before the runner starts) waits in the buffer, so the order of modules in `cmd/api` does not matter.
- **`events_dropped_total{event, subscriber}`** counts what a full buffer dropped, and **`events_failed_total{event, subscriber}`** what a handler answered with an error. The bus logs nothing: a subscriber that wants a log line wraps its handler in a decorator (`player`'s `log_subscriber`).
- **A subscriber's runner drains its buffer when it stops**, so what was published before the server stopped is still handled. A store it writes to must outlive it: `cppg.CloseAfter` closes the pool once the runners returned.
- **Tests:** `events_test.go` pins order per subscriber, a copy each, drop and count, no subscriber code in the publisher's stack trace, register-before-run, and the drain at shutdown. `cpbootstrap.RecordedEvents` (behind the tag) stands in for the bus where a use case only publishes.

The events today:

| event | published by | when | heard by |
|---|---|---|---|
| `planet.v1.TileTaken{account_id, tile_id, country, taken_at}` | `planet`, `ledger/publishing_ledger_storage` | each take the ledger records with an account: a click, a spread or an enclose, one event per tile | `player`, for the stats |
| `planet.v1.BombLanded{country, tile_id, ground, cleared, landed_at}` | `planet`, `drop_bomb_usecase/publishing_drop_bomb` | each bomb that went off, on land or in the sea; a refused drop and a dud publish nothing | `chat`, which announces it |
| `auth.v1.AccountDeleted{account_id}` | `auth`, `delete_account_usecase` and `prune_guests_usecase` | after the account row is deleted; a pruned guest is a deleted account | `player`, which forgets the profile, the stats and the visit |
| `auth.v1.SignedIn{previous_account_id, account_id}` | `auth`, `complete_sign_in_usecase` | after a sign-in or a link is saved; `previous_account_id` is empty for a browser that had no live session | `player`, which moves the browser's visit to the account |
| `auth.v1.SignedOut{account_id}` | `auth`, `sign_out_usecase` and `sign_out_everywhere_usecase` | after the session, or every session, is deleted; a cookie with no session publishes nothing | `player`, which takes the account off the roster |

Adding a context that callers talk to means a `proto/<name>/v1`, an `internal/<name>/` with a `module.go`, and one line in the slice. Connect derives the route from the proto package, so there is no prefix to allocate and no router to edit.

**`cmd/api`** is the only binary: `main.go` is the composition root, about 85 lines of config and a module list, and `cpbootstrap` is the rest.

It runs as **one container beside a postgres**: the tile map lives in process, and postgres is where it is kept between boots — see [Durability](#durability). There is no cache and no second API process.

### Inside the planet module: concepts, not layers

**There is no `domain` package and no `adapters` package, deliberately.** Both
name a layer, and a layer is the one thing every part of a module has in common
— so a package called that collects whatever does not fit elsewhere. What sits
under `internal/planet/internal/` is **one directory per concept of the game**,
and each concept carries its own layers inside it:

```
internal/planet/internal/
  clicks/                         the board: tiles, the map, what a click costs
    usecases/<name>_usecase/      one package per procedure
    inmemory_tile_storage/        an adapter: <tech>_<thing>_<role>
    embedded_geodesic_map/
  ledger/                         every take of every tile, and the operator tools that read it
    inmemory_ledger_storage/
    postgres_ledger_store/
    usecases/
  bonuses/                        the boxes, and what each one grants
    inmemory_charge_storage/
    postgres_charge_store/
    usecases/
  planetv1controller/             the edge: maps the wire to the use cases, nothing else
```

- **`clicks/`** — what a click is worth, what the map looks like, and what
  changes when somebody takes a tile. Its root holds the rules that need no
  port: `Board` (which tile ids exist), `Toll` (what a click costs), `Pacing`
  (how an operator's bulk change is spread out), `Geography` and `Borders`.
- **`ledger/`** — every take of every tile, oldest first. Its root holds `Taking`, `Player`, `Tally`, `Runs`,
  the `Storage` port, `Recording` (the tile writer that records) and `Retention`. `FindPlayers`, `TopPlayers`, `BanPlayer` and
  `RevertPlayer` live here: they are one moderation workflow — find, ban, undo.
- **`bonuses/`** — the boxes, their schedule, and the running bonuses they grant.
  Its root also holds the rules a bonus plays by: `Terrain` and `Pocket` (what an
  enclose closes) and `BombRules` (where a bomb lands and what it clears).

**A concept's root is its domain.** The use cases under `usecases/` load, call
the root, and persist; a rule that could be unit-tested without a port belongs
in the root.

**An adapter lives under the concept whose port it implements**, named
`<tech>_<thing>_<role>` — `inmemory_tile_storage`, `embedded_geodesic_map`. A
second implementation of the same port is a sibling directory, not a new layer.

**A port with a contract suite pins what every adapter must do.** `clicks.TileStorage`
is the whole tile map as it is kept, and `clicks.TileStorageContractSuite`
(`tile_storage_contract_testing.go`, behind the `testing` tag) is its behaviour:
a no-op publishes nothing, a blast is one event, a restore is a compare-and-set,
and so on. An adapter's test suite embeds it and sets `NewStorage`, then adds only
what is its own — `inmemory_tile_storage` adds the snapshot and the slow-subscriber
tests. A second tile storage runs the same suite by embedding it the same way.
A port with one adapter and no second one coming (`ledger.Storage`) has no suite.

**The controller is the one exception**, at `internal/planet/internal/planetv1controller/`,
because it serves every concept over one Connect service. It only maps.

#### Use cases (`<concept>/usecases/`)

**One package per procedure, and each declares its own ports.** Each exports
`New` and a `UseCase` with one `Execute`:

| package | what it does | what it needs |
|---|---|---|
| `clicks/usecases/click_usecase` | validates the country and the tile, then writes | `TilesChecker`, `TileStorage`, `CountryChecker` |
| `clicks/usecases/get_map_usecase` | a range of the map as one dense batch | `MaxIndexReader`, `DenseMapReader` |
| `clicks/usecases/map_density_usecase` | how many tiles there are | `MaxIndexReader` |
| `clicks/usecases/get_budget_usecase` | a caller's allowance, unspent | `ClickBudgetReader` |
| `clicks/usecases/listen_for_events_usecase` | one client's live feed, heartbeat included | `UpdatesSubscriber` |
| `clicks/usecases/reassign_country_usecase` | gives one country's tiles to another | `Map`, `CountryChecker` |
| `clicks/usecases/paint_random_tiles_usecase` | paints random tiles with a flag, starting on one country's ground or anywhere | `Borders`, `Neighbours`, `Map`, `CountryChecker` |
| `ledger/usecases/find_players_usecase` | who is painting a flag, and where | `Ledger`, `Owners`, `Borders`, `Bans` |
| `ledger/usecases/top_players_usecase` | who takes the most tiles, every flag | `Ledger`, `Owners`, `Bans` |
| `ledger/usecases/ban_player_usecase` | the operator's shadow ban | `Banner` |
| `ledger/usecases/inspect_player_usecase` | what the antibot holds on one caller | `Examiner` |
| `ledger/usecases/revert_player_usecase` | gives back what one caller still holds | `Ledger`, `Map` |
| `bonuses/usecases/claim_bonus_usecase` | redeems a box | `Registry`, `Charger` |
| `bonuses/usecases/drop_bomb_usecase` | spends a bomb where it was aimed | `Bombs`, `Map`, `Clearer` |
| `bonuses/usecases/get_charges_usecase` | what the caller holds | `Charges` |
| `bonuses/usecases/use_refill_usecase` | fills the caller's bank with its refill | `Refills`, `Bank`, `Pricer` |

**The interfaces in that last column are declared by the package that calls
them**, not gathered in a `gateways.go` every use case imports. A shared port
file makes every dependency everyone's, so `Click` ends up compiling against the
map reader it never calls. Here, adding a dependency to `get_map_usecase` is
invisible to the others — `inmemory_tile_storage` happens to satisfy several of
these ports at once, which is why `module.go` hands it over several times.

**`click_usecase` is the only one with an interface of its own (`IUseCase`)**,
because the click chain decorates it — `prom_click`, `throttle_click`,
`antibot_click`, `antibot_attempt_click`, and the bonus decorators `spread_click`, `enclose_click` and
`bonus_click`. The counting is a wrapper rather than a line inside the rule, so
a process that does not want it leaves it out and the rule does not change.

`get_map_usecase` and `listen_for_events_usecase` are decorated too, by
`antibot_get_map` and `antibot_listen_for_events`, through the port their
handler declares: they tell the guard what a caller reads, for the `scraper`.

`Geography` is in the `clicks` root, beside the sentinels and `TileUpdate`: the shape of the map, in `geography.go`. It is a model rather than a port — `embedded_geodesic_map` builds one and hands it over. See [Map geography](#map-geography).

### Inside the chat module: the same shape

Chat follows the same rules as planet: no `domain`, no `adapters`, one directory per concept. It has four:
`messages`, `reactions` (which imports `messages`), `announcements`, and `feed`, the live stream, which carries all
three. `subscribers/` is its edge for events, as `chatv1controller/` is its edge for the wire.

```
internal/chat/internal/
  messages/                             Message, MessageID, Record, Limits, AccountID, Author, ErrNoAccount, Window,
                                        Named and AccountsOf, the Storage port and its StorageContractSuite
    postgres_message_store/             Storage, over chat.messages
    inmemory_message_storage/           Storage in a slice — behind the testing tag, tests only
    rpc_player_authors/                 who an account is, from player.v1.InternalService/GetAuthor(s): one on
                                        each post, the whole window on each history
    log_authors/                        logs a caller, or a page of them, it could not name
    usecases/send_message_usecase/      names, cleans, appends, publishes — Appender, Publisher, CountryChecker, Authors
    usecases/get_history_usecase/       the window, each message named and with its reactions, the caller's marked
                                        and the announcements in the same window
                                                  — MessageReader, ReactionReader, AnnouncementReader, Authors
    usecases/prune_usecase/             deletes past retention, from each table; Runner — Pruner
      log_prune/                        logs what a prune deleted
  reactions/                            Reaction, Reactor, Reactions, Count, Tally, Change, the Storage port and its suite
    postgres_reaction_store/            Storage, over chat.reactions
    inmemory_reaction_storage/          Storage in a slice — behind the testing tag, tests only
    usecases/react_usecase/             puts a reaction on or off, publishes the tally — Messages, Board, Publisher
  announcements/                        Announcement, AnnouncementID, Kind, Bomb (a payload), the Storage port and its suite
    postgres_announcement_store/        Storage, over chat.announcements
    inmemory_announcement_storage/      Storage in a slice — behind the testing tag, tests only
    usecases/announce_usecase/          keeps an announcement, then publishes it — Appender, Publisher
  feed/                                 Update: a message sent, a message's new reactions, or an announcement
    inprocess_feed/                     the fanout to every open stream, in this process
    usecases/listen_for_events_usecase/ one client's feed, heartbeat   — UpdatesSubscriber
  chatv1controller/                     ChatService (a bag), the interceptors
    send_message_handler/  get_history_handler/  listen_for_events_handler/  react_handler/
    chatmessage/                        Encode, EncodeCounts and Reaction (the wire's enum, checked), shared by the handlers
    chatannouncement/                   Encode, shared by the history and the stream
    rpc_session_verifier/               the key from auth.v1.InternalService, asked once (planet's, copied)
  subscribers/                          Timeout
    bomb_landed_subscriber/             planet.v1.BombLanded → announce_usecase, as a Bomb payload
    log_subscriber/                     logs an event a subscriber refused (player's, copied)
  migrations/                           the chat schema
```

- **The rules that need no port are in the `messages` root**: `Limits` (runes, UTF-8, control characters), `AccountIDOf` (the context's account, or `cpsession.NoAccount`) and `NewRecord` (truncates the author id and user agent). They are tested there, not through the use case.
- **`send_message_usecase.Config` stays under `chat.Config.Service`**, so the `chat.service.*` keys do not change.
- **`chatv1controller`'s root tests are about the chain** (error net, blocklist, throttle). Each handler package tests its own mapping.

### Adapters

**Primary (input):**
- `internal/planet/internal/planetv1controller/` — the API. `ClickService` implements the generated `planetv1connect.ClickServiceHandler` and nothing else; it never sees an `http.ResponseWriter`, which is the point of serving the contract with Connect rather than by hand. **Each procedure is its own package** — `click_handler`, `get_map_handler`, `map_density_handler`, `get_budget_handler`, `listen_for_events_handler` — holding the one use case it calls and declaring the one port it needs. Each owns the mapping both ways, and each is tested on that mapping alone.

`ClickService` is those five embedded, and **nothing else**: no fields of its own, no methods of its own, and **no constructor** — it is a bag of handlers, so the DI sequence that already builds them writes the literal. It has no test either. An aggregation's only claim is that it carries all five procedures, and `var _ planetv1connect.ClickServiceHandler = ClickService{}` is that claim, checked at compile time. A test that served it and called a procedure would be re-testing the handler package that procedure lives in.

**A caller error becomes a Connect code in the handler, not centrally.** `click_handler` turns `clicks.ErrUnknownCountry` and `clicks.ErrTileOutOfRange` into `CodeInvalidArgument` and `clicks.ErrThrottled` into `CodeResourceExhausted`; `get_map_handler` turns `clicks.ErrInvalidTileRange` into `CodeInvalidArgument`. The sentinel these replaced was `ErrInvalidArgument`, which was a status code wearing a domain hat: it told a reader nothing a use case could act on, and it made every caller error in the game the same one. There is **no error interceptor in this package** — see [The error net](#the-error-net).
- the tile stream, as `ClickService.ListenForEvents` — a Connect server-streaming RPC like any other procedure on the service. See [The live streams](#the-live-streams).

**There is one server, one mux, and no version prefix.** Connect names each service's path from its proto package — `/planet.v1.ClickService/` and `/chat.v1.ChatService/` — so nothing is mounted under a prefix of ours. The three services and `/metrics` are the only things on the router; operator services have a loopback router of their own (see [Operator tools](#operator-tools-adminservice)). Nothing here needs a connection-level demultiplexer such as `cmux`; that is for running a real gRPC server, which owns its own HTTP/2 handler, beside a REST one.

`cpbootstrap` sets `http.Protocols` with both HTTP/1.1 and unencrypted HTTP/2, because the generated handler also speaks gRPC and gRPC-Web and those need HTTP/2. Browsers reach the same routes over HTTP/1.1. Verified: HTTP/1.1 and h2c both answer on the same port.

### The live streams

Both live feeds are served **two ways at once**, and that is a transition, not a design:

- `ClickService.ListenForEvents` → `stream PlanetEvent`, `ChatService.ListenForEvents` → `stream ChatEvent`, and `PlayerService.ListenForEvents` → `stream PlayerEvent`. Ordinary Connect server-streaming RPCs, on the same routes and the same port as everything else.

**One stream per API, and an envelope rather than a bare payload.** A `PlanetEvent` is a `oneof` of `tile_update` and `heartbeat`; `ChatEvent` is a `oneof` of `message`, `heartbeat`, `reactions` and `announcement`. **A new kind of live event is a new case in that `oneof`, never a second stream** — one connection per client, one route to configure, and a client that does not know a case reads an unset `oneof` and skips it instead of breaking. That is what the bare `TileUpdate` frame could not do, on the websocket or off it.

**`heartbeat` is not decoration.** Cloudflare cuts a silent response at **~125s with a 524** — measured against production three times, exactly 125.1s. The websocket never hit this because Cloudflare keeps those open; a chunked HTTP response is not so lucky. A quiet chat is the normal case, and a quiet planet happens, so both streams send a heartbeat every `httpServer.streamHeartbeat` (30s by default, and it **must** stay well under 125s). Without it a silent stream dies and reconnects forever, losing whatever was published in each gap.
These replaced a pair of websockets on `/ws/listen` and `/ws/chat`, broadcast by a `wspublisher` fanout. **Nothing here speaks websocket any more** — no upgrade route, no second mux, no `coder/websocket` dependency.

Each handler calls the storage's `Subscribe(ctx)` **per call**, and the request context is what unsubscribes — it is cancelled however the stream ends, so a disconnect needs no `CloseRead` equivalent. Both storages already handed every subscriber its own buffered channel and dropped rather than blocked for a slow one, so one subscription per connected client is what they were built for; `subscriberBuffer` now bounds a client rather than the single fanout.

#### Ending the streams on shutdown

**`http.Server.Shutdown` waits for every connection to go idle, and a stream never does.** Until 2026-09-15 every deploy waited out the whole `ShutdownTimeout`, logged `failed to shut down the http server error="context deadline exceeded"` (23 times on 2026-09-14), and then cut the streams hard — Caddy answered 502 on both `ListenForEvents` routes.

So `cpbootstrap` owns a **draining context**, cancelled just before `Shutdown`, and wraps every service it mounts in `drainInterceptor` (`cpbootstrap/drain.go`), between the error net and the module's own interceptors. For a streaming handler it gives the handler a context that is also cancelled when draining starts. Both handlers already return `nil` on `ctx.Done()`, so the stream ends **cleanly**, with an end-of-stream message, and the connection goes idle. No module changes anything to get it: a new stream handler only has to return when its context is done, which it must do anyway to unsubscribe.

- **Unary calls are untouched.** `WrapUnary` passes through, so a click in flight finishes and `Shutdown` waits for it as before. This is why it is not `http.Server.BaseContext`: that context is the parent of every request, and cancelling it would cancel the unary calls too.
- **A stream opened after draining started ends at once**, since `context.AfterFunc` runs straight away on a done context.
- **The frontend reopens a stream that ends cleanly** exactly as one that fails — `openStream` schedules the same reconnect either way, at 500ms after a stream that received anything — but without logging `stream failed`.
- `TestAnOpenStreamEndsCleanlyAndDoesNotHoldTheShutdown` opens a stream, shuts down, and asserts `Run` returns in under a second with no error logged, the client sees a clean end, and the closers ran after the stream ended. Without the drain it takes the full 5s and logs the production line. `TestAUnaryCallInFlightFinishesDuringTheShutdown` pins the other half.

A stream blocked inside a `Send` to a client that reads nothing is not woken by its context; `ShutdownTimeout` is still the backstop for that one.

**The streaming RPCs are wrapped by error mapping, the drain and the session reader, and nothing else**, because every other interceptor is a `connect.UnaryInterceptorFunc` and streams skip those by construction. Reads and the live feeds are therefore untouched by the throttle, the VPN blocklist and the session check, exactly as they were when they were websockets. A policy that ever has to reach a stream must be written as a full `connect.Interceptor`, as `cpconnect.NewSessionReaderInterceptor` is.

**The planet stream reads a token when the client sends one.** `planetv1controller.NewSessionReaderInterceptor` covers `ListenForEvents`: the token is verified once, from the headers that open the stream, and the account stays on the context for as long as the stream is open. It refuses nothing, so a stream with no token or a bad one opens as before. The web client sends the token it holds and never mints for it, and reopens the stream when a click goes out under a new token (see the frontend's CLAUDE.md). Nothing on the stream reads the account yet: offers and the `yours` flag are keyed by scope. `TestAStreamOpenedWithATokenKnowsItsAccount` pins it over HTTP.

### The map load

`GetMap` is an ordinary RPC, but marked `idempotency_level = NO_SIDE_EFFECTS` in the proto, so Connect sends it as an **HTTP GET** and the handler sets `Cache-Control: public, max-age=5` on the response. A burst of visitors can therefore share one origin response; `ListenForEvents` carries everything that happens after a chunk was built, so a client starting from a slightly old map converges anyway. `MapDensity` is marked the same way.

The response never repeats a tile id. `GetMapResponse` carries `start_tile_id`, the interned `codes` table, and `tiles` — a `bytes` field holding two bytes per tile, little endian, indexing into `codes`. Tile ids are implicit in the position, which is what makes it far smaller than the deprecated `map<uint32, string>`: **516 KB against 3.6 MB** for a full 257,948-tile map.

`inmemory_tile_storage.StateBatchDense` builds it. The interned ids are copied out exactly as stored and the table travels with them, so nothing is translated on the way out and the client needs no shared country list. Protobuf does all the framing — there is no hand-rolled magic or length-prefixing on either side, and therefore no encoder and decoder that have to be edited together.

**Secondary (output):**
- `clicks/inmemory_tile_storage/` — the tile map. A preallocated `[]uint16` indexed by tile id, with country codes interned into a side table (2 bytes per tile — ~2 MB for a 1M-tile map). Fans updates out in process, and flushes the tiles that changed through its `Persistence` port.
- `clicks/postgres_tile_store/` — that port, over the `planet.tiles` table. See [Durability](#durability).
- `ledger/inmemory_ledger_storage/` — the ledger, in memory, flushed through its own `Persistence` port.
- `ledger/postgres_ledger_store/` — that port, over `planet.ledger_takes`, `ledger_head` and `ledger_forgotten`.
- `bonuses/inmemory_charge_storage/` — the charges each account holds, in memory, flushed through its own `Persistence` port.
- `bonuses/postgres_charge_store/` — that port, over `planet.charges`.
- `clicks.Board` (not an adapter) — validates tile IDs
- country codes are validated by `shared/cpcountries`, which chat shares — see [The composite layer](#the-composite-layer)

`inmemory_tile_storage` implements `clicks.TileStorage`, including `Subscribe(ctx) (<-chan clicks.Change, error)`, one call per open stream. A `Change` is a tile update or a bomb blast, on one channel so the two keep their order — see [What a bomb does](#what-a-bomb-does).

### Key Flow

```
POST /auth.v1.AuthService/CreateSession   [Cookie: cp_sid=…, sent by the web client with credentials]
  → [cpbootstrap: error net], RateLimitInterceptor (the mint budget)
  → AuthService → create_session_handler
  → create_session_usecase: attest (turnstile_attester → Cloudflare siteverify)
  → accounts: the cookie's live Session (extended and saved when due), or a GuestSession, stored
  → shared/cpsession.Signer.Mint [Ed25519 over version+expiry+id+account+scope; nothing stored]
  ← token, and Set-Cookie when the session is new or renewed

POST /auth.v1.AuthService/StartSignIn   [Cookie: cp_sid, read for a link only]
  → start_sign_in_usecase: a link needs the cookie's account; seal the flow with the intent
  ← authorization URL, Set-Cookie: cp_oauth (sealed flow)
  … the provider … → https://clickplanet.lol/auth/callback?code&state
POST /auth.v1.AuthService/CompleteSignIn   [Cookie: cp_oauth, cp_sid]
  → complete_sign_in_usecase: open and check the flow, a link's account, provider.Exchange, accounts.OutcomeOf, SaveSignIn
  ← Set-Cookie: cp_sid (new session), cp_oauth cleared; the client mints again

POST /session.v1.SessionService/CreateSession   [deprecated]
  → create_anonymous_session_usecase: attest, then mint with no account

POST /planet.v1.ClickService/Click   [X-Session-Token: <the minted token>]
  → [cpbootstrap: error net], CacheInterceptor, VPNBlockInterceptor, SessionInterceptor
      [rpc_session_verifier: the key from auth.v1.InternalService, asked once per boot
       then cpsession.Verifier: one signature check — this context holds no seed]
  → ClickService → click_handler
  → antibot_attempt_click (times every try for the metronome; drops nothing)
  → throttle_click  (spends from the account's bucket and its scope's together, or refuses)
  → antibot_click   (judges; a flagged caller is answered OK and dropped)
  → prom_click      (counts)
  → clicks/usecases/click_usecase (validates tile ID + country)
  → MemoryTileStorage.Set() [writes the tile, fans the update out in process]
  → every subscriber: one per open ListenForEvents stream
```

`Set` is a no-op when the tile already holds that value — no write, no update published.

```
POST /chat.v1.ChatService/SendMessage   [X-Session-Token: required, naming an account]
  → [cpbootstrap: error net], BlocklistInterceptor, RateLimitInterceptor, then SessionInterceptor (a reader: refuses nothing)
      [rpc_session_verifier: the key from auth.v1.InternalService, asked once per boot]
  → ChatService → send_message_handler (the account off the context, or none)
  → messages/usecases/send_message_usecase: no account is ErrNoAccount (Unauthenticated), then
      who posts (log_authors → rpc_player_authors → player.v1.InternalService/GetAuthor): the account's
      username, or "guest_" and its guest code; stamps id/time
  → postgres_message_store.Append() [inserts into chat.messages: the account, never a copy of its name]
  → inprocess_feed.Publish() → every subscriber: one per open ListenForEvents stream
```

A failed insert fails the whole post: the table is the audit trail, so a message nobody can account for later is not one that gets broadcast.

```
POST /chat.v1.ChatService/React   [X-Session-Token: required, naming an account]
  → [cpbootstrap: error net], BlocklistInterceptor, ReactionRateLimitInterceptor, SessionInterceptor (a reader)
  → react_handler (refuses a Reaction the proto does not name)
  → reactions/usecases/react_usecase: who reacts is the account (reactions.ReactorOf); no account is ErrNoAccount
      is the message shown (postgres_message_store.Shown), what it carries (postgres_reaction_store.Reactions)
  → postgres_reaction_store.Save() [inserts or deletes in chat.reactions and bumps chat.reaction_versions, one statement]
      then reads the tally and its version back, one statement
  → inprocess_feed.Publish() the whole tally, versioned
```

### Chat (`internal/chat/`)

Chat is a separate bounded context, not a feature of the tile game: it shares the process, the transport and the country list, and has its own proto package, domain, storage and edge. Nothing under `internal/chat/` imports `internal/planet/`, and the reverse holds too — and since each module's interior sits behind its own `internal/`, neither now can.

**Always on.** There is no `chat.enabled`: the module is built on every boot, and its database block is required.

**Identity: a username, or a guest code, and always an account.** `SendMessage` reads the click token (`chatv1controller.NewSessionInterceptor`, over `cpconnect.NewSessionReaderInterceptor`, on the key `auth` hands over the internal listener). **A sender with no token, a bad one or a token with no account is refused** with `Unauthenticated` (`messages.ErrNoAccount`) before anything is asked. Every browser that passed Turnstile has an account, a guest one until it signs in, so the web client mints for a post as it does for a click. **The name is the player module's**, asked of `player.v1.InternalService/GetAuthor` on every post: the account's username, or `guest_` and the account's guest code (see [Player](#player-internalplayer)). Nobody types a guest's name, so no guest can pass for a player or for another guest. **A post the player module could not answer for is refused** with `Unavailable` (`messages.ErrAuthorUnavailable`): no answer within a second, or an error. `log_authors`, a decorator around the adapter, logs it at Error. Messages sent before usernames existed got the prefix from migration `20260917200000_guest_prefix_backfill`, and messages sent before guest codes keep the name a guest typed then, until the retention drops them.

**A message is kept as an account, not as a copy of a name.** `chat.messages.account_id` is who sent it; the name and the crown are read from the player module when the message is *shown*, never stored beside it. That is what makes a rename show on everything its player ever said, and a deleted account stop being named at all — a frozen copy could do neither, and the copy outliving the account it named was a small privacy hole of its own.

- **The history names a whole window in one ask.** `get_history_usecase` collects the distinct accounts (`messages.AccountsOf`) and asks `player.v1.InternalService/GetAuthors` once, however many messages each of them sent; `messages.Named` stitches the answer back on. An account that says fifty things costs one lookup, not fifty. **A history nobody could be named in is refused** rather than shown anonymous.
- **`SendMessage` still asks `GetAuthor` for one**, because what it publishes has to go out named: everyone already watching is shown who is talking without a second read. That call is also the one that gives a guest its code, which is why the read path uses `GetAuthors` instead — it draws no code, so showing the chat never writes.
- **An account that can no longer be named was deleted**, and reads as `messages.DeletedName` (`[deleted]`). No username can look like it: `SetName` refuses punctuation. What the account said stays, so a thread keeps its shape.
- **A message with no `account_id` is from before this**, and keeps the `name` and `author_admin` its row carries. Those two columns exist for that alone. **TODO** (see `postgres_message_store.row`): once `chat.storage.retention` has passed since this shipped, every remaining row has an account, and the columns and `messages.Named`'s fallback can go.

The client also sends a UUID it persists locally, kept in the log and **trusted for nothing**. **The sender's address is never public.** It is kept in `chat.messages.ip` for moderation. The salted hash of it that used to go beside every name (`author_tag`) is gone from the wire and the table (migration `20260918210000_drop_tag`): it changed whenever a player changed network, and it told anybody which names shared one.

**Abuse controls live at the edge**, in two interceptors — ahead of decoding the message and well ahead of validating it, so a flood of malformed messages costs a sender exactly what a flood of well-formed ones does. `chat.blockedIPs` cuts an address off from every chat RPC; `chat.rateLimiter` throttles `SendMessage` alone (`GetHistory` is one read on join, and limiting it would punish a page load). **Both are `shared/cpconnect`'s**, the same ones the click chain uses — see [Shared interceptors](#shared-interceptors). The domain then bounds the message in **runes** (280) and the name (24), validates UTF-8, and **strips control characters** — a newline once let a sender forge a line in the old log file, and a NUL is not valid in a postgres `text`.

Refusal reasons are logged, never returned: a sender learns *that* they were refused, not which check tripped. **The stored text is raw — the frontend must escape it.**

The **vendored VPN lists** do not cover chat: `NewVPNBlockInterceptor` wraps `Click` alone. Chat's blocklist is the same `*cpipblock.Blocklist` type, built by `cpipblock.NewDenyList` from config prefixes instead of vendored data — so entries are CIDRs and a `/24` is one line rather than 256. Extending the vendored lists to chat is therefore a wiring change (build the list in `describeModules` and hand it to both modules, the way `shared/cpcountries` already is), not a second list to write.

**Every message is a row in `chat.messages`**, inserted before it is broadcast: `seq` (the order it was accepted in), `id`, `sent_at`, `account_id` (who sent it), `country`, `ip`, `user_agent`, `text` — plus `name`, `author_admin` and `author_id`, of which the first two are only ever read, for rows written before `account_id` existed, and the third is a UUID the client made up and nothing trusts. **Postgres is the chat's only copy.** There is no cache in front of it, nothing loaded at boot and nothing flushed: chat is low volume (one message per 3s per address), so every write is one statement and every read one query. `send_message_usecase` inserts (5s timeout) and only then publishes to `inprocess_feed`; `GetHistory` reads the newest `chat.storage.historySize` rows within `retention` (`messages.Window`), straight from the table. A stream can therefore see two messages sent at the same instant in the other order than `seq`; the client sorts by time.

**The fanout is not storage.** `feed/inprocess_feed` keeps nothing: it hands each update to every open stream, drops for one too slow to keep up (`chat.storage.subscriberBuffer`, a key kept from before), and a client that was not listening reads the history instead.

**Each table has its own `Storage` port and a contract suite** (`messages.StorageContractSuite`, `reactions.StorageContractSuite`, behind the `testing` tag). The postgres stores run it against a real postgres; `inmemory_message_storage` and `inmemory_reaction_storage` run it too, and exist **only for tests** — both files carry the `testing` tag, have no persistence port and are never built into the binary. Use case tests use them instead of hand-written fakes.

**The prune** is `prune_usecase` on a `Runner`, as auth's guest prune is: it deletes messages, then reactions, then announcements, older than `retention`, once at boot and every `pruneInterval`, and `log_prune` logs it. The runner sits inside `cppg.CloseAfter`, so the pool closes after it stops.

The table holds **personal data** — IPs next to user-authored text — so the retention window is a policy decision rather than a cache size.

#### Announcements

**The chat also says things on its own**: a line between the messages with no sender, which the client draws
without a bubble. Today there is one kind, `bomb`: every bomb that went off, on land or in the sea.

- **A separate type and a separate table, not a message with no author.** An announcement has no name, tag, IP,
  text or reactions, and a message has no kind or payload; sharing a base would make every column of one
  optional in the other. So `announcements.Announcement` is `{ID, Kind, At, Payload}`, in `chat.announcements`
  (`id uuid` primary key, `kind`, `payload jsonb`, `announced_at`), with its own store, its own contract suite and
  its own case on the wire (`chat.v1.Announcement`: `kind` and `payload` as a JSON string).
- **The id is a real key**, `announcements.AnnouncementID` (`type AnnouncementID uuid.UUID`), made by the server.
  A message's id is `text` with a `seq` beside it because it is not trusted to be unique; nothing here has that
  history. A read orders by `announced_at`, then `id`.
- **The payload is the template's values, not the sentence.** The client writes the line, so a new wording, or
  a translation, needs no migration. Each kind owns its payload's shape: `announcements.Bomb` is
  `{country, ground, tile, cleared}`, `ground` and `tile` absent in the sea. **A client shows nothing for a kind
  it does not know**, the way it skips an unknown `oneof` case, so a new kind ships server first.
- **How a bomb gets here**: `planet`'s `publishing_drop_bomb` decorator publishes `planet.v1.BombLanded` once the
  blast is cleared, with the ground read off `clicks.Borders`; `bomb_landed_subscriber` turns it into a `Bomb`
  payload and `announce_usecase` inserts it, then publishes it on `inprocess_feed`. The announcement's time is
  the event's `landed_at`. **Delivery is at most once**, like every event: a full buffer (256) or a restart loses
  the line, never the bomb.
- **`GetHistory` returns them beside the messages**, in `announcements`, the newest `historySize` within
  `retention`, bounded apart from the messages so a burst of bombs never pushes one out. The client puts the two
  lists in one by time.
- **Not personal data**, but the prune deletes them past `retention` with the messages they sit between.

#### Reactions

**A message carries reactions from a fixed set**, `chat.v1.Reaction`. The proto enum is the whole list: the frontend draws each one from its own images (see its CLAUDE.md), and `chatmessage.Reaction` refuses a number the proto does not name with `InvalidArgument`. **The number is what is stored**, so a value is never renumbered or reused.

- **Who reacts** (`reactions.ReactorOf`): the account, `account:<uuid>`, whether it chose a username or not. No account is refused with `Unauthenticated` (`messages.ErrNoAccount`), and nothing is asked of the player module. So two accounts behind one address are two reactors. A reactor gives each reaction at most once per message. Rows from before, `guest:<tag>`, match no caller and age out with the retention.
- **`React` is on or off, not a toggle.** Asking for what is already there changes nothing, is not written and is not published, so a retry cannot flip it twice. It answers the message's counts with `mine` set for the caller. It asks the player module nothing.
- **Only a message in the window can be reacted to** (`ErrUnknownMessage` → `NotFound`): nobody is shown any other. `postgres_message_store.Shown` asks the same question as the history, for one id.
- **Saved, then read back, then published, with no lock.** Two reactions at once can publish their tallies in either order, so **each tally carries a version**: `chat.reaction_versions` holds one counter per message, bumped in the same statement as the change (a data-modifying CTE: no row changed, no bump), and read in the same statement as the reactions, so a tally and its version are one snapshot. The client keeps the highest version it has seen per message and drops a lower one. This holds across processes, which an in-memory lock would not. `reactions.Reactions` is a value: `With`/`Without` answer a copy.
- **The stream sends all of a message's counts, not the difference** (`ReactionsChanged`), with their version, so a client that missed a frame is right on the next, and one that gets two out of order keeps the newer. It cannot know who reads it, so `mine` is always false there; the client keeps its own between calls.
- **`GetHistory` marks the caller's own.** It reads the optional token (the session reader covers it): the account's reactions are marked, and a caller with no token has none. It asks the player module nothing.
- **Stored in `chat.reactions`**, their own table and their own store, one row per `(message_id, reaction, reactor)`, with `reacted_at`. A read replays the rows oldest first, so each reaction keeps the place it first appeared in. The prune deletes rows older than `chat.storage.retention`, like messages: the reactor is an account, so it is personal data. It deletes a version whose last change is that old too, which is only ever one whose message's reactions are all gone.
- **Its own rate bucket**, `chat.reactionLimiter` (defaults: 1 a second, 10 in hand), and the blocklist covers `React` too.

**Chat has its own stream**, `ChatService.ListenForEvents` — see [The live streams](#the-live-streams). It replaced a `/ws/chat` websocket that had to be kept apart from the tile one because frames carried a bare protobuf message with no type tag: a second payload on either socket would have been indistinguishable from the first. The `oneof` envelope is exactly what removes that constraint.

**Sending is an RPC, not a read on the socket**: both publishers lean on `CloseRead` for instant disconnect detection, and the RPC path already has the middleware stack and the interceptors.

`GetHistory` is marked `NO_SIDE_EFFECTS`, so Connect sends it as a GET — but it answers `Cache-Control: no-store`, the opposite of `GetMap`. A client fetches it once on join to seed what the stream then keeps up to date, so a cached answer would show a joiner a chat missing the last few minutes.

### Rate limiting

`throttle_click` throttles clicks from `shared/cpratelimit` token buckets. `MapDensity` and `GetMap` are cacheable reads a proxy in front absorbs; limiting them would punish a page load rather than a bot. A refused click answers `CodeResourceExhausted`, i.e. HTTP 429, and never reaches the map.

#### Two buckets per click

**A click spends from two buckets at once: its account's and its scope's.** The account is the one the click token names; the scope is the address, or its /64 over IPv6 (`cpipscope`). The account's bucket is `rateLimiter.*`: production runs one click every 5s (`perSecond: 0.2`) with a bank of 60, which fills in five minutes; unset, it is 1/s with a burst of 10. The scope's is `rateLimiter.scopeMultiplier` (10) times that, because many players can share one address — a campus, a school, a carrier NAT.

- **A linked account refills faster, into the same bank.** An account that signed in with Google or Discord refills `rateLimiter.linkedMultiplier` (2) times as fast, to make signing in worth it; its bank is the same size. It is the same `account:<id>` bucket either way — the rate is its pace, set by each click (see below) — so signing in neither tops the bank up nor empties it. The scope's bucket still bounds it. **Planet learns it from the token**, which carries a linked byte (see [Sessions](#sessions-internalauth)), so a click costs no call to `auth`. A player who links mid-session refills at 1× until the client mints again. `TestALinkedAccountClicksTwiceAsFastAsAGuest` and `TestSigningInKeepsTheBankAndSpeedsUpItsRefill` pin it.
- **A click is refused when either bucket is empty, and a refusal spends from neither.** `Limiter.TakeAll` checks every bucket and spends from all or none under one lock, so a player refused for a busy scope keeps its own tokens.
- **A token with no account spends one bucket, the scope's at 1×** — exactly the throttle from before accounts. The deprecated `session.v1` mint, an invalid token while `auth.enforce` is off, and `auth.enabled` false all land here. It is a separate bucket from the scope's shared one, because a key never changes its scale.
- **One limiter holds both.** A bucket's `Scale` is set when it is made and multiplies its burst and rate for good; `clicks.Buckets` names the keys (`account:<id>` at 1, `scope:<scope>` at the scope multiplier, or the bare scope at 1 with no account). **A key's `Pace` multiplies the payer's own bucket's rate from that take on, and leaves its burst alone**: the linked multiplier for a signed-in account, divided by the country's slowdown (see [A big country refills slower](#a-big-country-refills-slower-clickstoll)). The scope's bucket takes no pace. `clicks.PayerOf(ctx)` reads the scope and the account off the context, so the throttle, `GetBudget` and a bonus claim cannot disagree on who pays.
- **What this buys.** Before accounts, one address was one allowance, so a campus played as one player and a bot farm with many cookies on one address was no worse off than one tab. Now each player behind an address has its own allowance, bounded together by the scope's, and a bot moving across addresses keeps spending one account's.
- `TestManyAccountsOnOneScopeShareTheScopesBucket` and `TestOneAccountOnManyScopesSpendsOneAllowance` pin both halves over HTTP.

**It is a decorator over the click use case, not an interceptor over the procedure.** Two things fall out of that. The allowance comes back as a return value (`click_usecase.Out`) instead of being left on the context for a handler to find, which is what `cpctx.AddRateBudgetToContext` existed for and why it is gone. And "a click refused for its address or its session must not also spend a token" stops being a rule about the order of a list and becomes a property of the shape: every interceptor is outside the whole click chain by construction. `MapDensity` and `GetMap` are untouched for free, being other procedures entirely — under an interceptor that took a procedure list.

#### Saying what is left

The web app shows the player how many clicks they have in hand, and **the server is the only thing that knows**. A client running its own copy of the bucket would drift within seconds — it cannot see the clicks the same address makes from another tab, and its idea of when a click was spent is a round trip out of date.

Polling for it would be worse, so nothing polls. `Limiter.TakeAll` returns each bucket's state alongside its verdict, and the state carries the **policy** (`Capacity`, `PerSecond`) as well as the reading: given both, a client replays the same refill arithmetic between two answers and is exact without asking. The pip count and the fill rate on screen are therefore the server's burst and refill rate — **changing `rateLimiter.*` changes the display with no frontend release.**

That reading travels two ways, because a refused call has no response message to put it in:

- an allowed call carries it on `click_usecase.Out`, and `click_handler` puts it in `ClickResponse.budget`.
- a refused one carries it on the same `click_usecase.Out`, beside `clicks.ErrThrottled`, and `click_handler` attaches it as a **connect error detail** — a refusal has no response message to put it in.

Either way the decorator decides the policy and the handler decides how to say it. `clickbudget.Encode` is the one place that shape is agreed, because two procedures answer with a `ClickBudget`: the click that just spent a token, and `GetBudget`.

**Every reading also carries `linked_multiplier`**, what signing in multiplies the refill by (`Buckets.BudgetOf`). It is the same for every caller, so the client can tell a guest what signing in is worth with no number of its own.

**The reading is the tighter bucket** (`clicks.Tightest`): the one with fewer tokens, and on a tie the smaller one. A player behind a busy campus sees the scope's limit rather than a full meter that refuses. The reading is still one bucket's `Capacity`, `PerSecond` and `Tokens`, so the client arithmetic does not change. `TestTheBudgetIsTheTighterBucket` pins it.

`ClickService.GetBudget` covers the cold start — a client that has just loaded and has no click to learn from. It reads through `Limiter.Peek`, which spends nothing and, for an address that never clicked, **creates no bucket**: reading an allowance must not be a way to make the limiter remember a caller. `NewSessionReaderInterceptor` reads a token on it when the client sends one, so the reading is the account's, and **refuses nothing**: with no token or a bad one it reads the scope's bucket from before accounts. The web client sends the token it already holds, never a fresh one; before its first click it holds none, and neither bucket has been spent from. Without the token the meter showed that other bucket, always full, until a click contradicted it. It is deliberately not `NO_SIDE_EFFECTS`, so it is a POST no cache will serve a stale answer to; every click re-anchors the client afterwards, so it is asked once per page load.

**This tells a scripted clicker exactly when to fire**, which is a real cost against [Anti-bot](#anti-bot-internalantibot). It is a small one — a script can already infer the same schedule by counting its own 429s — and it is paid to stop honest players being refused with no warning.

The scope is whatever `IPReaderMiddleware` put on the context: `X-Real-IP` if present, otherwise the peer address. **The reverse proxy must set that header itself** — `deploy/vps/caddy/Caddyfile` does, with `header_up X-Real-IP {client_ip}` on every backend route. Merely forwarding it would let a client send its own and buy a fresh bucket per request. The fallback is the peer address rather than a constant precisely so a missing header degrades to per-connection buckets instead of rate limiting the whole game as one player.

#### A big country refills slower (`clicks.Toll`)

**Every click costs one token.** What the map share changes is how fast the
tokens come back. `toll.steps` is a table of `{share, slowdown}`: from `share` of
**every tile on the map**, a player of that country refills `slowdown` times
slower (production runs 1.5× from 25%, 2× from 50%, 3× from 70%). No steps
refills every country at the plain rate. `ClickBudget.slowdown` took the field
number of the old `cost`, whose meaning it replaces.

- **The bank never changes size**: not with the country, a bonus or signing in.
  Only its rate moves, so the number a player sees changes when it clicks, when
  time passes, or when a bonus is caught — never on a switch of flag.
- **The pace is set by each click, from the country clicked for, and applies from
  then on.** The time already past was refilled at the pace in force over it, so
  switching flags moves nothing until the next click. A refused click sets it too.
  A player can refill on a small country and spend the bank on a big one; the
  bank bounds what that buys.
- **The share is of the whole map, not of owned tiles**, so early in a game nobody is slowed.
- **`inmemory_tile_storage` keeps a tile count per country**, moved by `set` and
  `Clear` and rebuilt at boot from postgres, so `Share` is one read and no scan.
- **Only the payer's own bucket is slowed.** The scope's is a ceiling shared by
  players of every flag, so it refills at its plain rate.
- **The budget goes out as the bucket holds it**, in clicks, with the slowdown,
  the share and the next step of the country asked about so the client can say why.
- **Bonuses compose with it.** A refill fills the bank to its size and leaves the
  pace alone. A spread is one click. A bomb is not throttled, and lowers the
  share of whoever it hits.

`GetBudget` takes the country, for the slowdown it answers. Known risk, not
handled yet: a country sitting on a step can cross it back and forth click to click.

Chat and sessions each have **their own limiter instance** with their own budget, because what each call costs has nothing to do with what a click costs:

- `chat.rateLimiter` — one message every 3s, five in hand. A message fans out to every connected client and lands in a log everyone will read.
- `chat.reactionLimiter` — one reaction a second, ten in hand. A reaction is a short row and a small frame.
- `auth.rateLimiter` — one mint every 30s, ten in hand, across both `CreateSession` paths. A mint costs a siteverify round trip to a third party, so an unthrottled `CreateSession` is a free way to spend this server's Turnstile quota.

### VPN blocklist

`NewVPNBlockInterceptor` refuses **`Click` only**, with `CodePermissionDenied` (HTTP 403), when the source address falls in a vendored VPN range. It sits **outside the rate limiter** in the interceptor chain, deliberately: a refused address must not also spend a token, or the next click would come back 429 and the web app would show the throttle dialog instead of the VPN one.

**It exists because of the rate limiter, not instead of it.** The scope's bucket is keyed on an address, and a commercial VPN is the cheapest way to get a fresh one; refusing those addresses is what makes the bucket hold. It raises the floor rather than closing the door — residential proxies appear in no public list, and nothing here stops one. **That gap is what [Sessions](#sessions-internalauth) closes**, by requiring something an address cannot buy; chasing list completeness instead is a treadmill.

Reads and the streams are untouched. A VPN user still loads the planet and follows it live; they cannot paint. That is also what keeps a false positive readable: the page works and says why, instead of failing to load.

**The ranges are vendored and embedded**, from [X4BNet/lists_vpn](https://github.com/X4BNet/lists_vpn) (MIT, rebuilt daily from ASN ownership), in `internal/shared/cpipblock/cpdata`. Not fetched at boot: `cmd/api` is a self-contained container with no startup dependencies, and a boot that can fail because GitHub is down is a worse trade than a list that ages between deploys — the Cloudflare ranges in `deploy/vps/caddy/Caddyfile` are maintained the same way. Refresh with `make vpn-lists` and commit; the tests assert the lists still parse and are not truncated.

**X4BNet works from ASN ownership, so the `vpn` list folds in four more sources**, all fetched by the same target (it needs `jq`):

- `vpn_providers.txt` — [Joe12387/open-source-vpn-ip-lists](https://github.com/Joe12387/open-source-vpn-ip-lists) (CC0): each provider's own server list, which catches servers rented inside networks X4BNet does not attribute to a VPN.
- `vpn_az0.txt` — [az0/vpn_ip](https://github.com/az0/vpn_ip) (GPL-3; IP addresses are facts): hostnames pulled from VPN apps, APIs and browser extensions (Hola, CyberGhost, …) and resolved.
- `tor_exits.txt` — the Tor Project's exit list.
- `vpn_netnames.txt` — every range registered under a name in `VPN_NETNAMES`, from the RIPE and ARIN RDAP servers. **This is how Firefox's built-in VPN is caught**: its egress (`MOZILLA-FIREFOX-VPN`) is Mozilla's address space announced by Fastly, so no ASN list can find it, and it hands each browser session its own /64 — one client walks the range and gets a fresh throttle bucket and ban scope each time.

Measured on 2026-09-14 against the X4BNet vpn and datacenter lists together: the provider lists add ~1,250 addresses, az0 ~1,800 beyond those, Tor ~750. iCloud Private Relay's published egress list is left out on purpose, for the reason the datacenter list is off by default.

`cpipblock` holds them as sorted, merged `[lo, hi]` ranges of 16-byte addresses and binary-searches them — ~63k prefixes fold to far fewer ranges, about 2 MB resident and well under 100 ns per lookup. **IPv4 and IPv6 live in separate slices.** They cannot share one: an IPv4 address in its v4-mapped form sits inside `::ffff:0:0/96`, so a single ordering would let a v6 prefix as short as `::/16` silently swallow every IPv4 address on the internet.

`vpnBlocklist.includeDatacenters` adds the much broader hosting list, which catches a self-hosted VPN on a VPS. It is off by default because it also refuses Apple iCloud Private Relay and Cloudflare WARP — both egress from datacenter ranges, both on by default for a lot of ordinary mobile traffic. `blocked_clicks{list}` is labelled per list precisely so you can see what turning it on would cost before turning it on. `vpnBlocklist.allow` is the escape hatch and beats both lists.

**The `X-Real-IP` caveat applies here too, and matters more.** Anything that reaches `backend:8080` directly bypasses the blocklist by sending its own header, exactly as it bypasses the throttle. Caddy replaces the header on every backend route, so this is only reachable if the backend port is exposed — but the bypass is now security-relevant rather than merely an abuse nuisance.

### Sessions (`internal/auth/`)

The answer to the one thing an address-based defence cannot do. The rate limiter and the VPN blocklist both key on an address, so the whole defence reduces to "can the attacker get addresses" — and against residential proxy pools, which appear in no public list, it can. **A session is what makes clicking cost something to start.**

`Click` requires a token this server minted, in the `X-Session-Token` header. The only way to get one is `auth.v1.AuthService/CreateSession` (or the deprecated `session.v1.SessionService/CreateSession`), which verifies a **Cloudflare Turnstile** token against siteverify before minting. A script that reads the proto and POSTs `Click` no longer has a complete client: it has to solve Turnstile first.

**The token is stateless.** `shared/cpsession` mints `base64url(version ‖ expiry ‖ random id ‖ account ‖ linked ‖ Ed25519(version ‖ expiry ‖ id ‖ account ‖ linked ‖ scope))` — 98 bytes, 131 characters. The account is 16 bytes, all zero (`uuid.Nil`) for a caller with none; `linked` is one byte, 1 when the account signed in with a provider. `Mint` takes a `cpsession.Holder` (`cpsession.Nobody` for no account), and `Verify` answers both in `Claims`. The linked byte is signed, so a guest cannot claim the linked allowance. Nothing is stored, swept or replicated; verification is one signature check. That is what keeps this compatible with a process that holds the whole game in memory and has no database to put a session table in.

**It is signed, not MACed, and that is the point.** An HMAC key verifies and mints with the same bytes, so every context that could check a click could also issue one. Ed25519 splits that: the seed is `auth.secret` and only the auth module is handed it. Verification costs tens of microseconds against a MAC's one, which is nothing at this traffic — production is thousands of clicks per five minutes — and buys a boundary the compiler holds.

**`planet` asks for the key, it does not hold one.** `planetv1controller/rpc_session_verifier` calls `auth.v1.InternalService/GetVerifyingKey` over the internal listener **once**, on the first click after a boot, and keeps the answer: the key does not change while the process runs, so every click after that is one signature check and no I/O, exactly as it was when this module read a key of its own.

It cannot ask at boot instead — `cpbootstrap` builds every module before it listens, so the internal listener is not up while `planet` is being built. By the time a click arrives the server is serving, so in practice it is never late. A failed fetch is not remembered, so the next click tries again; with `auth.enforce` false the click passes and is counted either way. The cost of the laziness is that a misconfigured `auth` shows up on the first click rather than at boot.

**The first byte is a version.** Before it, the length *was* the discriminator, so every format change made every token in flight malformed at once. Now `planet` can accept two versions across a rollout instead, and key rotation is the same move: mint under the new version while both verify. Version 2 added the linked byte; a version 1 token (97 bytes) still verifies, as not linked, and its support can go once the last one has expired (1h after the deploy).

**It is bound to the address that minted it**, so a token lifted off the wire is worth nothing anywhere else. The scope is signed over but never travels — the verifier rebuilds the message with the scope it observes — so the token leaks nothing. A player whose address changes mid-session — a phone moving from wifi to cellular — fails verification, and the client mints again and retries: self-healing, and invisible.

The signature is checked **before** the expiry, so a forger learns nothing about whether their token would otherwise have been in date.

**`auth.enforce` is the rollout switch.** False — the shipping default — makes the interceptor decide nothing: every click passes and its verdict is counted. `click_session_checks{verdict}` then says exactly what enforcing would refuse (`missing` and `invalid`) before it refuses it, which is what lets the backend deploy ahead of the frontend. True answers `CodeUnauthenticated` (HTTP 401), and the client is expected to mint and retry rather than show the player anything.

**Where it sits in the chain:** error mapping, VPN blocklist, **session**, throttle. Outside the limiter for the same reason the blocklist is — a click refused for its session must not also spend a token, or the retry that follows the mint would come back 429 and the web app would show the throttle dialog instead. `TestSessionCheckRunsBeforeTheThrottle` pins it.

**Reads and the streams are untouched.** A visitor loads the planet, watches it live and reads the chat without ever minting anything; a session is only ever needed to paint. `GetMap` is a cacheable GET and a per-session header on it would defeat that cache.

**Minting has its own throttle** (`auth.rateLimiter`, one every 30s with 10 in hand, shared by both `CreateSession` paths). A mint costs a siteverify round trip to a third party, so it cannot share the click budget: unthrottled, the endpoint is a free way to spend this server's siteverify quota.

**`auth/internal/attestation/turnstile` fails closed on everything.** A network error, a non-2xx, a body that is not JSON, a token for another action or another hostname are all refused exactly as a forged one is. Failing open would make the check decorative — an attacker who can reach the backend can also make siteverify unreachable from it. It validates `action` and `hostname` as well as `success`, because **the sitekey is public**: without those two checks a token minted by the same widget embedded on any other page would be accepted here.

**`auth.turnstile.enabled: false` mints for anyone who asks** (`open_attester`). That is how a local backend runs without a widget and a secret, and it still exercises the whole click path — the token is bound and expires. It is never the production choice, and the server warns at boot when it is on.

**Two secrets, neither in git.** `auth.secret` is the Ed25519 **seed**, 32 bytes as 64 hex characters — what `openssl rand -hex 32` already produced for the key it replaces. Anyone holding it can mint a token the API accepts. `auth.turnstile.secret` is the widget's secret half. Both come from the environment via `deploy/vps/docker-compose.yaml`, as `player.tagSalt` does. **An empty or malformed `auth.secret` with `auth.enabled` true refuses the boot**, naming the variable to set. It used to generate one and warn; a server that invented a key would invent a different one per restart and could not verify what it had just minted.

**There is still only one key to set.** The public half is derived at boot, so nothing has to be pasted into a second setting and nothing can drift out of step with the seed.

**What it does not stop:** a person who solves Turnstile in a real browser and then runs a userscript. They hold a genuine session, and nothing here distinguishes them from a player. This raises the floor from "twenty lines of Python" to "drive a real browser"; the signal that survives that is behavioural — timing regularity and tile-id structure over a session — which is what the session id on the context exists to be keyed on. Nothing in production reads it yet, so `cpctx.GetSessionID` sits behind the `testing` tag (see [Testing](#testing)) until something does.


### Auth (`internal/auth/`)

**One module admits a caller: Turnstile, then the account its cookie holds, then the click token.** Every clicking browser gets an account, a guest one until it signs in; signing in with Google or Discord links to the same row, so a guest's history needs no merge. Off by default (`auth.enabled`); off, `auth.v1` and `session.v1` 404.

It was two modules, `session` and `auth`, for one PR. The mint was auth's only caller, and "not a bot" and "who" are decided on the same request for the same purpose, so the line between them cut through one piece of work.

```
internal/auth/internal/
  accounts/                                AccountID, TokenHash, Account, Identity, Session, Token, Lifetime, OutcomeOf, the cookie; the Store, IDProvider and TokenGenerator ports
    postgres_account_store/                accounts, identities and sessions in the auth schema
    inmemory_account_store/                the same port in maps, behind the testing tag
    uuid_id_provider/  random_token_generator/
    usecases/create_session_usecase/       attest, resume or start a guest, mint
    usecases/create_anonymous_session_usecase/   deprecated: attest, mint with no account
    usecases/get_me_usecase/               reads only
    usecases/get_account_usecase/          reads one account, for another module
    usecases/sign_out_usecase/  sign_out_everywhere_usecase/  delete_account_usecase/
    usecases/prune_guests_usecase/         deletes idle guests: Executor, Runner, and log_prune_guests
  signin/                                  Flow (state, PKCE verifier, nonce, intent), Provider, Providers, Sealer, the cp_oauth cookie
    google_identity_provider/  discord_identity_provider/  oauth_http/
    aes_flow_sealer/  random_secret_generator/
    usecases/start_sign_in_usecase/  complete_sign_in_usecase/
  attestation/                             Attester, ErrAttestationFailed
    turnstile/  turnstile_attester/  open_attester/
  authv1controller/                        AuthService and InternalService: one handler package per procedure, authprovider for the enum
  sessionv1controller/                     deprecated SessionService
  migrations/
```

- **`session.v1.SessionService/CreateSession` is deprecated**, in the proto and in the code. It mints a token with no account for clients that predate accounts, and goes once none calls it. Both paths spend one mint budget, so the old one is not a second allowance.
- **The cookie is `cp_sid`**: a 32-byte token from `random_token_generator`, `HttpOnly; Secure; SameSite=Lax; Path=/`, host-only on the API's domain. The API and the frontend are the same site, so it is not a third-party cookie. **Only its SHA-256 is stored** (`sessions.token_hash`), so a copy of the table signs nobody in. Caddy redacts `Cookie` and `Set-Cookie` in its logs.
- **The rules are on `accounts.Session`**: `GuestSession` builds one with a full `guestTTL` (90 days) and `LinkedSession` with a full `linkedTTL` (30 days), `ExpiryError` is `ErrSessionExpired` past it, `Extendable(now, lifetime)` is whether `extendEvery` (24h) has passed since the last extension, `Extended(now, lifetime)` is a copy whose expiry moved by the TTL of its kind, and `Cookie` is its `Set-Cookie`. `Session.Linked` is read by the store (does the account have an identity), never written, so every session of an account extends as a linked one once any browser links it. `create_session_usecase` only orders them: attest, read the cookie, find the session, check it, extend and save it when due — or, when there is no cookie, no such session or an expired one, start and store a guest — then mint.
- **Only a caller that passed attestation gets an account**, so bots that fail Turnstile make no rows. The mint throttle bounds how many guests one address makes.
- **A failing database fails the mint.** No fallback to a token with no account: the error net answers `internal`.
- **Absence is a sentinel, never `nil, nil`**: `ErrNoSessionCookie`, `ErrSessionNotFound` (the port's, for an unknown token hash), `ErrSessionExpired`, and `ErrNoAccount`, which `GetMe` answers as `Unauthenticated`. `GetMe` creates, extends and saves nothing, and answers `no-store`.
- **Ids and tokens are injected** (`IDProvider`, `TokenGenerator`), like the clock. Tests use `accounts.SequentialIDs` and `SequentialTokens` (behind the tag), so they assert exact ids.
- **`accounts.StoreContractSuite` is the port's behaviour**, like `clicks.TileStorageContractSuite`. Both stores embed it: postgres adds only what the port cannot show (the token is never stored, `last_seen_at`, an unverified email is `NULL`), and the use cases are tested over the in-memory one, which can `FailWith` an error.
- **No cache.** Mints are one per 30s per address, so one indexed read each is cheap, and a sign-out has nothing to invalidate.
- **`InternalService/GetAccount(account_id)`** answers `linked`: whether the account signed in with a provider (`Account.Linked`). An unknown account, or an id that is not one (`accounts.AccountIDOf`), is `linked` false and not an error; a store failure is. `player` asks it before it gives an account a username.
- **`create_session_usecase` mints whether the account is linked** (`Session.Linked`, read by the store), so the token says it and `planet` gives a linked account its faster bucket.
- **`planet` reads the account off the token**: the session interceptor puts it on the context (`cpctx.GetAccount`, and `cpctx.GetLinked`), and the throttle, the bans and the ledger key on it beside the scope — see [Two buckets per click](#two-buckets-per-click), [Anti-bot](#anti-bot-internalantibot) and [Manual bans](#manual-bans-findplayers-topplayers-banplayer-revertplayer-inspectplayer).

#### Signing in (`internal/auth/internal/signin/`)

**OAuth 2.0 authorization code with PKCE, run by this server, over two RPCs.** No raw HTTP route and no Caddy change: the provider sends the browser to the frontend's callback page, and that page calls the API.

- **`StartSignIn(provider, intent)`** draws a `signin.Flow` — state, PKCE verifier and nonce, 32 random bytes each, and the intent — seals it into the `cp_oauth` cookie (10 minutes, same attributes as `cp_sid`) and answers the provider's authorization URL. Nothing is stored on the server.
- **The intent is sign in or link** (`accounts.Intent`). Unset on the wire is sign in, and so is a flow sealed before intents existed: `IntentSignIn` is the zero value. The menu's "Sign in with" buttons send sign in, and "Link" sends link.
- **A link needs an account when it starts.** With no live `cp_sid`, `StartSignIn` answers `Unauthenticated` and sets no cookie. The flow keeps the account id, and `CompleteSignIn` refuses a link whose browser is on another account or none by then (`Flow.AccountError`, `ErrFlowInvalid`), before the provider is asked.
- **`CompleteSignIn(code, state)`** opens the cookie, checks the state (constant time) and the expiry, trades the code with the verifier, and clears `cp_oauth` on every answer it maps, success or refusal. A browser that did not start the sign-in has no cookie that matches, which is the whole CSRF defence: a code carried to a victim's browser is refused before the provider is asked.
- **The cookie is AES-256-GCM** (`aes_flow_sealer`), under a key derived with HKDF from `auth.secret` and a label of its own, so there is no second secret to set and the browser can neither read the verifier nor change the flow.
- **Google** (`openid email`) reads the user from the ID token in the token endpoint's answer. Its signature is not checked: it comes straight from Google over TLS, which OpenID Connect Core 3.1.3.7 accepts instead; issuer, audience, expiry and nonce are. **Discord** (`identify email`) reads `/users/@me`. Both go through `oauth_http`, which answers `ErrProviderRefused` for a 4xx or an answer that does not decode, and a plain error (the error net's `internal`) when the provider could not be asked.
- **`accounts.OutcomeOf` decides, and emails are never compared.** To sign in: an identity already linked signs in to its account; a new one links to the account the browser is on; no account, or one that already holds a user of that provider, gets a new account. The guest a browser leaves for a known identity is left as it was — nothing is merged, and the prune deletes it later.
- **A link never moves the browser.** A new identity links to the account; one already on this account is `SignedIn` and changes nothing but the session token; one on **another** account is `ErrIdentityLinkedElsewhere`; an account that already has a user of that provider is `ErrProviderAlreadyLinked`. A refusal writes nothing and keeps `cp_sid`: only `cp_oauth` is cleared. The handler answers `AlreadyExists` with a `LinkRefusal` detail, which is what the client matches. Before this, a link to an identity another account used moved the browser to that account, and the two accounts could never be joined. To move an identity, the player signs in with it, deletes that account, then links it — there is no merge.
- **`accounts.SignIn` is written in one transaction**: the new account if any, the identity if new, the new session, and the deletion of the browser's previous session. The session token changes on every sign-in. Two browsers linking the same new identity at once: the second insert finds it taken (`ErrIdentityTaken`), and the use case runs once more with the identity known — a sign-in moves, a link is refused.
- **An email is kept only when the provider says it is verified** (`accounts.NewIdentity`), and it is `NULL` otherwise. It is for contact, never for finding an account.
- **Off by default** (`auth.signIn.enabled`). Off, `signin.Providers` is empty and both RPCs answer `Unimplemented`, which Connect sends as HTTP 404. A provider is offered once its `clientId` is set. `StartSignIn` and `CompleteSignIn` spend the mint budget: each can cost a round trip to a third party or make an account.
- **`GetSignInOptions` is how the client knows which buttons to show**: every provider offered, and an empty list while sign-in is off — never `Unimplemented`, since the question has an answer either way. **It is not throttled and sets no cookie.** A client asks on every page load, and probing with `StartSignIn` instead would spend the mint budget a real sign-in needs and start a flow for nothing. `TestAskingIsNotThrottled` pins it. A server with the whole `auth` module off still 404s it, which the client reads as no provider.
- **The client mints again after `CompleteSignIn`**, so its click token carries the account. The old token names the old account until it expires (1h).
- **`CompleteSignIn` publishes `auth.v1.SignedIn`** once the sign-in is saved: the account the browser was on, empty for none, and the one it is on now. A refusal publishes nothing. `player` moves the roster line, since the client holds no token to announce with until its next click.
- `auth.NewModuleWithFakeProviders` (behind the tag) boots the module with `signin.FakeProvider` for every provider: `Grant(code, claim)`, and the fake checks the verifier against the challenge it was shown. `e2e/sign_in_test.go` drives it.

#### Signing out, deleting, pruning

- **`SignOut`** deletes this browser's session and clears the cookie; a browser with no session succeeds too. **`SignOutEverywhere`** deletes every session of the account and answers `Unauthenticated` with none. Neither touches the account. Both publish **`auth.v1.SignedOut`** once a session was deleted; `SignOut` reads the session first, expired or not, to name its account.
- **`DeleteAccount`** deletes the account row, and its identities and sessions go with it by cascade, then publishes **`auth.v1.AccountDeleted`**. `player` hears it and deletes the profile and the stats. Planet and chat keep nothing else keyed on the account; the ledger and the chat log age out.
- **No unlink RPC.** When one comes, a linked account must keep its last provider: without one it is a guest holding an email.
- **`prune_guests_usecase`** is an `Executor` and a `Runner` that calls it; `log_prune_guests` is the decorator that logs, so the loop holds no log line. Every `auth.prune.interval` (1h) it deletes, 1000 rows a statement, the accounts with no identity whose `last_seen_at` is older than `auth.prune.idleFor`, and publishes `auth.v1.AccountDeleted` for each (`PruneGuests` answers the ids it deleted). That defaults to `guestTTL`, and less refuses the boot: `last_seen_at` moves when a session is extended, so a guest idle that long has no live cookie left.

### Player (`internal/player/`)

**What the game keeps about one account: the name it chose or its guest code, the tiles it took and its daily streak; and who is playing now.** It makes no account and mints nothing. Always on: the chat asks it who posts.

```
internal/player/internal/
  players/                          AccountID, Name (NameOf), GuestCode (GuestCodes, DisplayNameOf), Author (AuthorOf), Tag (TagOf), Profile,
                                    Stats, Day; the Store port and its contract suite
    postgres_player_store/          the Store over player.profiles, player.guest_codes and player.stats
    random_code_generator/          draws guest codes from crypto/rand
    inmemory_player_store/          the same port in maps, behind the testing tag
    rpc_account_reader/             whether an account is linked, from auth.v1.InternalService/GetAccount
    usecases/get_profile_usecase/  set_name_usecase/  get_stats_usecase/  get_author_usecase/  get_player_usecase/
    usecases/get_authors_usecase/  — who many accounts are, and a pure read: it draws no guest code
      set_name_usecase/renaming_set_name/   shows a kept name on the roster at once
    usecases/record_take_usecase/  forget_account_usecase/
  presence/                         Visit, Entry, RosterOf, TTL: who is playing
    inmemory_visit_storage/         the last visit of each account, capped, keyed, pruned every 5s (a Runner), and each change to its subscribers
    usecases/announce_usecase/  get_roster_usecase/  listen_for_events_usecase/  move_visit_usecase/  forget_visit_usecase/
  playerv1controller/               PlayerService and InternalService (bags), the session interceptor
    get_profile_handler/  set_name_handler/  get_stats_handler/  get_author_handler/  get_player_handler/
    get_authors_handler/
    announce_handler/  leave_handler/  get_roster_handler/  listen_for_events_handler/
    caller/                         the account on the context, or Unauthenticated
    playermessage/                  Profile, Stats and Player as player.v1 messages
    rpc_session_verifier/           the key from auth.v1.InternalService, asked once (planet's, copied)
  subscribers/                      the edge for events, as the controller is for the wire
    tile_taken_subscriber/  account_deleted_subscriber/  signed_in_subscriber/  signed_out_subscriber/  log_subscriber/
  migrations/
```

- **The caller is the account in the click token.** Every call but `GetRoster`, `GetPlayer` and `ListenForEvents` sits behind `cpconnect.NewSessionInterceptor`, always enforcing, on the key `auth` hands over the internal listener, as `planet` does. No token, a bad one, or a token with no account (the deprecated mint) is `Unauthenticated`. `player_session_checks{verdict}` counts the verdicts.
- **`GetProfile`** answers the account id and its name, empty when none was chosen. **`GetStats`** answers `tiles_taken`, `streak_current`, `streak_best` and `streak_last_day` (YYYY-MM-DD).
- **`SetName` chooses a username.** `players.NameOf` is the rule. It puts the name in **NFC** and cuts the spaces (U+0020) at its ends — the same text, as it shows — and changes nothing else. Then:
  - **3 to 15 characters, counted in code points after NFC**, as postgres' `char_length` counts, so both agree. A letter with a combining mark NFC cannot compose counts two.
  - **Each is a letter of any script (`\p{L}`), a combining mark (`Mn`, `Mc`) right after a letter or a mark, at most 3 in a row, a decimal digit (`Nd`), `_` or a space.** Never two spaces in a row. So emojis, punctuation, symbols, controls, enclosing marks, and every other space (NBSP, ideographic) are refused. **Invisible characters are refused by name**, since some are letters or marks: Hangul fillers (U+3164, U+115F…, `Other_Default_Ignorable_Code_Point`) and variation selectors; zero-width, bidi and soft hyphen are format characters, refused as not letters.
  - **One script** (`oneScript`): the letters' scripts, Common and Inherited left out, are one, or a mix UTS #39 calls *highly restrictive* — Latin with Han, Hiragana and Katakana; with Han and Bopomofo; or with Han and Hangul. So `Adа` (Cyrillic `а`) and `guеst_x` are refused.
  - **Not starting with `guest_` once folded** (`Name.Folded`), which the chat puts before every guest's name. Folded, so `ＧＵＥＳＴ_x` and `gueſt_x` are refused too.

  A name that breaks a rule is `InvalidArgument` (`ErrInvalidName`). The name keeps the case it was typed in. `GetPlayer` runs the same rule on the name it is asked for, so `" Ada "` finds `Ada`.
- **Known risk: whole-script confusables.** The one-script rule stops a name that swaps one letter for a lookalike of another script, not a name written wholly in a script that looks like another: `Аԁа`, all Cyrillic, is not `Ada`, and both can be held. Closing it needs the UTS #39 skeletons (`confusables.txt`, vendored by a make target) folded into the unique key. Nothing on screen tells two such names apart today: the chat shows no tag beside a name any more.
- **Usernames are unique ignoring case** (`Name.Folded`): **NFKC_Casefold**, the NFKC of the Unicode full case fold of the NFKD, so `Straße` is `STRASSE`, `Émile` is `éMILE`, and a full-width `Ａｄａ` is `ada`. A name another account holds is `AlreadyExists` (`ErrNameTaken`); an account setting its own name again, in any case, is not refused, and a rename or a deleted account frees the old one. **The game computes the fold and the store keeps it** in `profiles.name_folded`: postgres' `lower()` depends on the database's locale (a C locale folds ASCII only) and folds neither `ß` nor full-width letters, so an index on it would disagree with Go on non-ASCII names. Postgres holds uniqueness, not a read before the write: a unique index on `name_folded`, whose violation (`23505` on `profiles_name_key`) `postgres_player_store.SaveProfile` answers as `ErrNameTaken`, so two players asking for one name at once cannot both get it. `ProfileNamed` reads through it. The in-memory store compares `Folded` under its lock, and `StoreContractSuite` pins both.
- **The table's `CHECK` holds what it can say without the locale**: 3 to 15 `char_length`, `IS NFC NORMALIZED` (a UTF8 database), no space at either end or two in a row, no ASCII but letters, digits, `_` and space, no C0 or C1 control, and none of the zero-width, bidi and other invisible format characters (a range list in the migration); and `name_folded` not empty and not starting with `guest_`. Letter categories and scripts are Go's alone: postgres' regex classes depend on the locale too.
- **Only a linked account may hold one.** `set_name_usecase` checks the name first, so a name no account may hold costs no call, then asks its `Accounts` port whether the caller signed in with a provider: `rpc_account_reader` calls `auth.v1.InternalService/GetAccount` over the internal listener on every `SetName` (2s timeout), since an account links at any time and a name is chosen rarely. A guest is `PermissionDenied` (`ErrNotLinked`). Auth answers `linked` false for an unknown account or an id that is not one; **a failure to ask is a real error**, the error net's `internal`, never a guest. With `auth` off that call 404s, so no name can be set.
- **Migration `20260918190000_unicode_usernames`** moves to this rule. Every name kept before is ASCII of 3 to 20, so only length breaks the new rule: **a name over 15 is cut to its first 15 characters**, the player keeping most of it. A cut name that another would then hold, ignoring case — a name of 15 or fewer, or an older cut name (`updated_at`, then `account_id`) — is **deleted** instead, and its player chooses again; an untouched name is never the one to go. Then it fills `name_folded` with an ASCII-only `translate()` (exact for these names, in any locale), replaces the `CHECK`, and moves `profiles_name_key` to `name_folded`. The down migration deletes the names the old rule refuses, drops the column and puts the old `CHECK` and the `lower(name)` index back; cut names stay cut. `migrations_test.go` migrates to the version before, writes names, and migrates the rest.
- **Migration `20260917180000_usernames`** deletes the profiles whose name breaks the new rule, then, of names that differ only in case, every one but the oldest (`updated_at`, then `account_id`), replaces the name `CHECK` with the new pattern and adds the unique index. No client called `SetName` before usernames existed, so nothing a player chose is lost. The down migration drops the index and puts the old `CHECK` back; the deleted rows stay deleted.
- **A guest code names an account that chose no username**: 6 lowercase hex characters (`players.GuestCode`), shown as `guest_` and the code (`players.DisplayNameOf`). It is drawn by `random_code_generator` the first time the account is shown (`players.GuestCodes.Assign`), kept in `player.guest_codes`, and deleted with the account, which frees it. **It is unique**: the table's `guest_codes_code_key` refuses a code another account holds, the store answers `ErrGuestCodeTaken`, and `Assign` draws again, up to 10 times (`ErrNoFreeGuestCode`). An account keeps its first code: a second save for it changes nothing, so two first announces at once leave one code. **It does not change with the network**, unlike the address, and says nothing about the account or the address. A linked account with no username keeps the code it had as a guest until it picks a name.
- **`InternalService/GetAuthor(account_id)`** is for the chat: the name the game shows for the account (`players.AuthorOf`: the username, or `guest_` and the code) and `admin`. **`get_author_usecase` gives a guest its code the first time it is asked** (`Assign`, then read again), and the roster's announce and move use the same use case, so the chat and the roster always show one name for one account. An empty id, or one that is not an account, is `InvalidArgument`: the chat refuses a sender with no account before asking.
- **`InternalService/GetAuthors(account_ids)`** is the same question about a page of accounts, for a caller showing many people at once — the chat naming a whole history. **It is a pure read**: unlike `GetAuthor` it draws nobody a guest code, so a read path never writes. That is the whole reason the two exist side by side; `GetAuthor` belongs where somebody is about to be *made* visible, `GetAuthors` where somebody is merely being shown. One query (`Store.Authors`, an `unnest` of the ids left-joined onto `profiles` and `guest_codes`) answers the page, a repeated id is answered once, and **an account it cannot name — never shown, or deleted — is left out** rather than failing the call, so the caller decides what stands in its place. An id that is not an account is still `InvalidArgument`: a caller holding one has a bug.
- **The tag is private.** `players.TagOf` (SHA-256 of `player.tagSalt`, a NUL and the address, cut to 6 hex characters) is kept on a roster visit so the storage can cap the visits of one address, and never leaves the server.
- **Who is playing is `presence`, in memory only.** A client calls **`Announce(country_id)`** with its click token when it gets one, when its flag or name changes, and every 30s. A visit counts for `presence.TTL` (90s), so a hidden tab whose timers fire once a minute stays on; the prune runs every 5s. A restart empties it, and clients fill it again within one interval. **`Leave`** takes the caller off at once: a client sends it with `keepalive` when its page closes. **`GetRoster`** needs no token (the session interceptor does not list it), is `NO_SIDE_EFFECTS`, and answers `public, max-age=5`; it stays for clients from before the stream.
  - **`ListenForEvents` is the roster, live.** It needs no token. The first event is the whole roster (`roster`, in `GetRoster`'s order); each one after is a line that joined or changed (`entry`), or the key of one that left (`left`); `heartbeat` every `httpServer.streamHeartbeat`. **A line's `key` is the storage's**: a counter handed to an account's first visit, kept by every later announce, `Move` and `Rename`, so a sign-in renames one line rather than adding one. It says nothing about the account. `Storage.Subscribe` answers the roster and registers the subscriber under one lock, so nothing is lost between the two, and every change is published under that lock, in order. An announce that changes nothing on the line (name, flag, guest, admin) publishes nothing, unless the visit had gone stale. **A subscriber `SubscriberBuffer` (256) changes behind is closed, not skipped**: a skipped change would leave its roster wrong for good, and the stream ends, reconnects and starts from a whole roster.
  - **One entry per account**, so the tabs of one browser and the devices of one signed-in player are one line. A visitor who never got a click token is not listed: announcing must not cost a Turnstile mint.
  - **Named as the chat names a sender**, by the same use case (`get_author_usecase`): the username, or `guest_` and the guest code. The name is read on every announce. **No address, and no hash of one, is on a line.**
  - **Sorted**: players with a username first, then guests; each group by name ignoring case, then by key.
  - **Caps, against a script minting accounts** (`inmemory_visit_storage`): at most 10 accounts per tag, where a new account pushes out the tag's oldest visit, and 10,000 in all, where a new account is not recorded. The mint throttle already bounds how fast one address makes accounts.
  - An unknown country is `InvalidArgument`; a failed profile read is the error net's `internal`, and nothing is recorded.
  - **A sign-in, a new name, a sign-out and a deletion change the roster at once, with no announce.** The client drops its click token on each of these, and a new one waits for a click, so waiting for its next announce left a guest line on the roster, or two lines, for up to 90s. So: `auth.v1.SignedIn` moves the browser's visit to the account it is on now, under that account's name (its username, or its own guest code), over any visit the account held (`move_visit_usecase`, `Storage.Move`; one account before and after changes nothing, and reads nothing). `SetName` renames the caller's visit once the name is kept (`renaming_set_name`, `Storage.Rename`). `auth.v1.SignedOut` and `auth.v1.AccountDeleted` take the account off (`forget_visit_usecase`, `Storage.Forget`); another device still signed in announces again within 30s. An account with no visit is left off by all of them: its browser never announced.
- **Stats come from `planet.v1.TileTaken`**, one event per tile, so a spread of seven is seven tiles. **The streak day is UTC**: a take on the day after `streak_last_day` extends the streak, a take on the same day changes nothing, a gap starts it again at 1, and a late event older than the last day counts a tile and leaves the streak alone (`Stats.WithTake`). `GetStats` reads it as of today (`Stats.AsOf`): a streak whose last day is before yesterday reads 0, and `streak_best` keeps it. `stats_test.go` pins the rules across UTC midnight.
- **An admin of the game is `player.profiles.admin`**, false by default. **The game never sets it**: an operator flips it in the database (below), and `SaveProfile` never writes it, so a rename keeps it. It only means something beside a username: `GetAuthor` and `GetAuthors` answer `admin` for the chat, which shows the crown on a message whose account is an admin **now** — it is read when the message is shown, not stamped on it, so a player that stops being an admin stops looking like one on what it already said; `presence.RosterOf` sets `Admin` on a roster entry only when it is not a guest; `GetPlayer` answers it too. The frontend draws a crown on all three. An announce reads the profile, so a new admin shows on the roster within 30s, and in the chat the moment anybody reloads it.

  ```bash
  ssh deploy@YOUR_IP
  cd /opt/clickplanet/deploy/vps
  docker compose exec postgres psql -U clickplanet -c \
    "UPDATE player.profiles SET admin = true WHERE name = 'TheUsername'"
  ```

  An account with no username has no profile row, so it cannot be one: pick the name first.
- **`GetPlayer(name)` is what anybody may know about a player with a username**: the name as typed, the stats as of today, and `created_at_unix_ms`, when auth made the account (as a guest or by a first sign-in, so a guest who signs in keeps its first day). It needs no token (the session interceptor does not list it), is `NO_SIDE_EFFECTS`, and answers `public, max-age=10`. **It never answers the account id.** The name is found ignoring case (`Store.ProfileNamed`, on the unique index on `name_folded`). A name no account holds is `NotFound` (`ErrNoProfile`), and so is one no account may hold, a guest's included, which reads nothing. A guest has no username, so it has no answer here: the client shows its name and flag only. `rpc_account_reader.CreatedAt` asks `auth.v1.InternalService/GetAccount` on each call, which now also answers `created_at_unix_ms` (zero for an account auth does not know, and the answer then carries zero). **A failure to ask auth is a real error**, the error net's `internal`, as for `SetName`.
- **`auth.v1.AccountDeleted` deletes both rows.** A take that arrives after, on a token minted before the delete, makes a new stats row; the token lives an hour at most.
- **Events are at most once.** A take dropped by a full buffer (`events_dropped_total`) or lost in a crash is a tile the stats never count. Stats start the day the module is turned on: takes before are not replayed.
- **No memory copy: every call reads or writes postgres.** This is not the tile map's pattern on purpose. The map is in memory so a click never waits on the database; a take reaches this module over the event bus, so a click already never waits on it, and the calls are few (production is ~15 takes a second at peak). A memory copy would load every account that ever took a tile at boot, and cost a dirty set, a flush loop and a window a hard kill loses.
- **`RecordTake` is one transaction** under `pg_advisory_xact_lock` on the account: read the stats, apply `Stats.WithTake`, upsert. The rule stays in Go, and two takes never read the same stats. An advisory lock rather than `FOR UPDATE`, because a first take has no row to lock; `TestConcurrentFirstTakesAreAllCounted` fails without it.
- **Errors are real errors.** Every `Store` method returns one; absence is `ErrNoProfile` or `ErrNoStats`, which the use cases answer as an empty profile or empty stats. On a request a database error is the error net's `internal`. On an event it is counted in `events_failed_total`, logged by `log_subscriber`, and the take is lost. Each event's write has a 5s timeout (`subscribers.Timeout`), so a stuck database cannot hold a subscriber.
- **`players.StoreContractSuite`** is the port's behaviour, run on `inmemory_player_store` and on postgres. The use cases are tested over the in-memory one, which can `FailWith` an error.
- **The pool closes after the subscribers stop** (`cppg.CloseAfter`), so what they drain from their buffers at shutdown is still written.
- **No foreign key to `auth.accounts`**: the schemas are each module's own, and the event is how a deletion crosses.
- **If takes ever outgrow one transaction each**, the subscriber can batch them; nothing else changes.

### Bonus boxes (`internal/planet/internal/bonuses/`)

A question-mark box flies past the planet every so often; whoever catches it
gets one of four bonuses. Each box draws its kind from `bonus.kinds`, a weight
per kind — a kind's chance is its weight over the sum of the weights, so the
strong ones can be made rare (production runs 5 : 2 : 1 : 2):

- **`refill`** — a charge: fills the caller's click bank to full, when the
  caller chooses. Up to one bank, 60 clicks. See [What a refill does to the bucket](#what-a-refill-does-to-the-bucket).
- **`spread_clicks`** — 1 to `bonus.spread.maxPerBox` (4) clicks added to a pool
  of at most `bonus.spread.clicks` (8). While the player switches spread on, each
  click spends one and also takes the tiles touching the one clicked. See [What a spread does to a click](#what-a-spread-does-to-a-click).
- **`bomb`** — a charge: one bomb, kept until it is dropped. It
  clears a circle of `bonus.bomb.rings` tile spacings around where it lands,
  whoever holds the tiles. See [What a bomb does](#what-a-bomb-does).
- **`enclose_clicks`** — 1 to `bonus.enclose.maxPerBox` (3) enclosures added to a
  stack of at most `bonus.enclose.held` (3). While the player switches enclose on,
  a click that closes a shape of the caller's own tiles also takes the tiles
  inside it, at most `bonus.enclose.maxTiles` (25), and spends one.
  See [What an enclose does to a click](#what-an-enclose-does-to-a-click).

Every kind is a **charge**, worth about one bank, rather than a timer — see
[Charges](#charges-refill-bomb-enclose-spread). There used to be a `triple_clicks`
that multiplied the refill for two minutes. With a bank of 60 it was worth "some
clicks, maybe": nothing on a full bank, and a different amount for every
country's pace. The refill is the same good, clicks, with the moment chosen by
the player.

Boxes are always on: there is no switch.

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
- **Caught** — a charge has no end, so the next is due a window after the
  **claim**, not after it is spent: a bomb held for an hour does not hold back
  every other box for an hour.

**Only callers who have clicked inside `activeWithin` are offered anything.** A
tab left open overnight is not playing, and it is also what keeps the miss rule
from needing a back-off of its own. A caller whose turn comes up while they are
away **loses the slot rather than banking it** — otherwise they are handed a box
the instant they come back.

**A schedule outlives its stream by `forgetAfter`.** Without that, closing the
tab and opening it again draws a fresh wait, and a player could reload until
they got a short one.

**`maxChargesPerHour` bounds what a caller can be granted.** Nothing here is a
race any more, but catch rate is where an advantage is left: a script catches
every box it is offered where a person catches some. This makes the worst case
a number you choose rather than a function of reflexes. Past 12 charges in the
last hour the slot is lost, like a caller's who was away; 12 is above the ten
boxes an hour a person catching every one would get. A kind another box would
add nothing to (a refill or a bomb held, a full stack or pool) is not offered.

**A caller is a scope, not a connection.** `Attend` is keyed on `cpipscope.Of`,
the same unit the session token binds to, and holds every stream
sharing it — so twenty tabs are one entrant on one schedule, and all of them are
sent the box. Offers stay on the scope; the charge a box grants lands on the
claiming account. The handler's `defer` is what removes it; there is no
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

The catch is published **after** the charge is held, so a catch announced to the
planet that then failed to apply is the one lie this cannot tell.

#### Charges (refill, bomb, enclose, spread)

A timer rewarded speed rather than planning: with a bank of clicks, a 10s spread
let a player dump the whole bank at seven tiles a click, more than a bomb, and a
bomb held 30s was dropped on the first target in sight. So every kind is a
**charge**, kept until the player uses it: nothing lapses and nothing is used on
its own.

- **Only an account holds a charge.** `bonuses.Holder` is the account the click
  token names, and `HolderOf(clicks.PayerOf(ctx))` derives it, so the claim, the
  click chain and the drop agree on whose charge it is. A charge is the account's
  so it survives closing the tab, a new address and another device. A caller with
  no account is `NoHolder`: it holds nothing, and a scope where no account plays
  is offered nothing. Every client mints a guest account, so this leaves
  out only a caller with no token at all.
- **The rules are a value, `bonuses.Held`**: a refill and a bomb (held or not), a
  stack of enclosures and a pool of spread clicks. `Granted`, `AfterRefill`,
  `AfterBomb`, `AfterEnclose` and `AfterSpreadClick` build a new `Held` and change
  nothing; the storage swaps it in. `Count(kind)` is how many of a kind are held,
  and `Full(config)` the kinds another box would add nothing to.
- **How much fits.** A refill and a bomb are one: a second replaces the first,
  which is what stops a stockpile of bombs being dropped all at once. Enclosures
  stack to `bonus.enclose.held` (3) and spread clicks pool to `bonus.spread.clicks`
  (8). A box draws its amount evenly from 1 to `maxPerBox` (`Registry.amountOf`,
  `crypto/rand`) and the grant caps it at the size. The claim answers
  `ClaimBonusResponse.amount` as **what was kept**, `Count` after less `Count`
  before, so a player is never told of clicks that did not fit.
- **The schedule offers no kind that is full** for any account that clicked from
  the scope within `activeWithin` (`Registry.offerable`). The schedule is by scope
  and a charge is by account, so `bonus_click` tells the registry both on every
  accepted click: `Clicked(scope, holder)`. The registry reads the charges through
  its `Holdings` port, which the storage satisfies.
- **Off by default, one at a time.** `ClickRequest.spread` and `enclose` say what
  the player switched on for this click, and `click_usecase.In` carries them.
  `spread_click` spends a spread click only with `Spread`, `enclose_click` an
  enclosure only with `Enclose`. **Both at once is refused** by the rule itself
  (`clicks.ErrBonusesTogether`, answered `InvalidArgument`) before anything is
  written or spent: a spread's tiles and an enclose's pocket would each take what
  the other decides. The client keeps one on at a time too, and aiming the bomb
  switches both off.
- **Nothing lapses.** Migration `20260919120000_charges_never_lapse` replaced the
  `*_until` columns with `refill` and `bomb` booleans and an `enclosures` count,
  keeping what was still in time. `bonus.chargeTTL` is gone.
- **A restart keeps them.** `bonuses/inmemory_charge_storage` holds every account's
  `Held` in memory, so a click reads and spends under one lock with no round trip,
  and writes the ones that changed through its `Persistence` every
  `chargeStorage.flushInterval` (1s): `bonuses/postgres_charge_store`, one row per
  account in `planet.charges` (`refill`, `bomb`, `enclosures`, `spread_clicks`). An
  empty hand is a deleted row. Like the tile map: boot loads it and a failed load
  refuses the boot, shutdown flushes once more, and a hard kill loses at most the
  last second. A deleted account's row stays until something removes it. The rest
  of the bonus state (schedules, offers, the hourly caps) is still memory only, so
  a restart gives everyone a fresh schedule.
- **Each spend is atomic**: the storage's `SpendRefill`, `SpendBomb`,
  `SpendEnclose` and `SpendSpreadClick` check and take under one lock, so two tabs
  racing for the last one get one.
- **Nothing pushes them.** They are not live news: `GetCharges` answers what the
  caller holds, read by the client at load and when its account changes, and
  `ClaimBonusResponse.charges` answers the claim. After that the client follows
  its own calls: a drop spends the bomb, an accepted click sent with spread on a
  spread click, its own `tiles_enclosed` an enclosure. A charge spent in another
  tab shows until the next read. **`Click` answers nothing about them on purpose**:
  a shadow-banned click never reaches the spread, so a count on the answer would
  tell a banned caller its clicks are dropped. `planetv1controller/chargesheld`
  encodes the message both procedures answer.
- **How big a charge is, is a rule, not state.** `GetBonusRules` answers the
  blast radius, the enclose's `maxTiles`, the spread pool's size and the
  enclosure stack's size (`bonuses.Rules`, built in `module.go`). It is
  `NO_SIDE_EFFECTS`, a GET the cache interceptor marks for 5 minutes, and the
  client reads it once per page load. A page open across a deploy that changes
  them shows the old sizes until it reloads.

#### What a spread does to a click

**The server picks the tiles, off its own map.** A client that named the tiles
a click spreads to could name any tiles it liked — that is why the spread waited
for [Map geography](#map-geography). The client paints the tile it clicked, as it
always has, and the neighbours reach it over the stream like anyone else's.

`claim_bonus_usecase` adds the box's clicks to the pool with
`Charges.Grant(holder, KindSpreadClicks, amount)`. Each accepted click sent with
spread on then spends one with `Charges.SpendSpreadClick`, after the rule accepted
it: a refused click spreads nothing and costs nothing, and a click with spread off
never touches the pool.

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

#### What a bomb does

`claim_bonus_usecase` hands the bomb over with `Charges.Grant(holder, KindBomb)` and
the client draws its aiming ring at `GetBonusRules.blast_radius`, the width of
what it will clear. `DropBomb` spends it through `drop_bomb_usecase` with `Charges.SpendBomb`.

**The client names a point, never a tile.** The sea has no tiles, and whether an
aim is on land is the server's call: `Geography.Nearest` finds the closest tile,
and an aim further than one tile spacing from it is **in the sea** — the bomb is
spent, nothing is cleared, and the blast is still broadcast with tile 0 so every
screen draws a splash. That was a product decision: a bad aim costs the bomb.

On land the tiles are `Geography.Within(centre, radius)`: every tile within
`radius` of arc of the tile hit — a true circle, ~230 tiles inland, found by a
straight scan (~0.5ms, once per bomb). The radius is `bomb.rings × Geography.Spacing()` (`bonuses.NewBombRules`, and `BombRules.Blast` decides land or sea),
the mean arc between touching tiles measured at boot (0.0040 rad on the
257,948-tile map, so 0.032), rather than a number in the config that could drift
from the map; the same radius goes to clients, so the ring they draw is the clear.

**It is not `Disc`, deliberately.** Rings of neighbours on a honeycomb make a
hexagon, which showed in production as a hexagonal crater inside a round ring —
and a walk over neighbours stops at water, so an island just offshore survived a
bomb that visibly covered it. A circle has neither problem.

`drop_bomb_usecase` checks the country and the target **before** taking the bomb, so a
malformed request does not cost one. The drop does not move the schedule: the next
box was already due a window after the claim.

**The blast is one event, and it rides the tile feed.** `inmemory_tile_storage.Clear`
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
**Every bomb that went off is told to the other modules** as `planet.v1.BombLanded`
by `publishing_drop_bomb`, inside the count, and the chat announces it — see
[Announcements](#announcements).
**The shadow ban applies**: `antibot_drop_bomb` marks a banned caller's drop as a
`Dud`, which spends the bomb, clears nothing and publishes nothing, and is answered
OK — a bomb left in hand would tell the caller it was refused. It sits outside the
count so the counter can label it.
`prom_drop_bomb` counts `bonus_bombs_dropped_total{outcome=land|sea|refused|shadowbanned}`
and `bonus_bomb_tiles_cleared_total`.

#### What an enclose does to a click

**It looks for a small inside, never for the outline.** On a sphere every loop
cuts the planet in two, and both halves are inside it. So after an accepted
click, `click/enclose_click` floods out from each neighbour of the clicked tile
that is not the caller's, over tiles that are not the caller's. A flood that
runs out before passing `enclose.maxTiles` found a pocket; one that passes it is
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
- **An enclosure is one shape**, spent through `Charges.SpendEnclose` only by a
  click sent with enclose on that closed a pocket, so a click that closes nothing
  (too big, open to the coast, no inside) keeps it. With enclose off, closing a
  shape takes nothing. The spend settles two clicks racing for the last one. A
  click that closes two shapes takes the first.
- **Why 25 tiles**: the wall around a 25-tile pocket costs about 20 clicks, so the
  reward matches the planning it took.

**The use case only wires three objects together.** `bonuses.Terrain` is the map as
the search sees it — who holds a tile, what touches it — and finds the pockets a
click closed. `bonuses.Charges` says whether the caller holds the charge and how
big a shape may be. `Annexer` spends the charge on the first pocket, takes its
tiles and announces them. `Execute` asks whether the charge is held, lets the
rule write, and hands the pockets to the annexer.

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
with `yours`: an enclosure is one shape, so a shape of your own is one spent.

`prom_enclose` wraps that publisher, so it counts exactly the shapes that were
closed: `bonus_enclosures_total` and `bonus_enclosed_tiles_total`.

#### What a refill does to the bucket

`UseRefill` spends it, through `use_refill_usecase`, and `cpratelimit.Limiter.Fill(key)`
tops the bucket up to its capacity. **It fills the account's bucket, never the
scope's** (`Buckets.Own`): the scope's is shared with every other player behind
the address, so a refill that filled it would be a refill for all of them. A
refilled player still spends from the scope's bucket, so it is bounded by it —
`TestARefillDoesNotFillTheScopesBucket`. The bank never grows past its size, and
the pace is left as it was. `Fill` is **additive**: nothing that never calls it
can tell it exists, which matters because the same limiter type throttles chat
and session mints.

- **A full bank is refused before the charge is touched**: `ErrBankFull`,
  answered `FailedPrecondition`, spends nothing. The client checks first and says
  "Bank already full" without asking; the server check is what holds.
- **No refill, or no account**: `ErrNoRefill`, answered `NotFound`.
- **The answer carries the budget**, full, and the charges, so the meter jumps at
  once rather than on the next click.
- `UseRefill` is session-gated like `Click` and `DropBomb`, and not throttled:
  holding the refill the server granted is the gate.

### Anti-bot (`internal/antibot/`)

What is left after sessions. A player who solves Turnstile in a real browser and
then runs a userscript holds a genuine session, and no address- or token-based
check can tell them from a player. The signal that survives is **behavioural**.

**The whole of its API is ten names**, and `internal/antibot/antibot.go` is all
of it: `Config`, `Observer`, `Guard`, `New` and `Description` to wire it, plus
`Click`, `Report`, `Sentence`, `Examination` and `Reading` — the types a caller writes down, because it builds one
and is handed the others. A caller hands over the block and the two hooks it wants
findings reported through, and gets back a `Guard` — one that drops and bans nothing when the block is off, so
the DI sequence wires it the same way either way — that answers `Attempted`, `Inspect`, `Committed`, `Caught`, `Missed`, `Fetched`, `Listened`, `Flagged`, `Banned`, `LoadState`, `Run` and `Enabled`, plus `Ban`,
`Sentence`, `Enforcing` and `Examine` for the operator tools (see [Operator tools](#operator-tools-adminservice)). It is
**one** `Run` whatever the file turned on: how many sweepers there are is this
package's business, which is why `planet` registers one runner rather than one per sweeper.

**A caller is never taught this package's vocabulary.** The edge does two things
with a watchdog's opinion — count it if it argued for the ban, and put it in the
log line — so an `Opinion` answers `Fired()` and renders itself with `String()`,
and `Verdict`, `Evidence`, `Field` and the `clear`/`suspect`/`certain` ladder stay
inside. `Examine` follows the same rule: an `Examination` carries `Reading`s whose
level and evidence are already strings, so the edge copies them onto the wire and
never compares against the ladder. The alternative shipped briefly and is what this rule is written against:
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
could not assemble a jury out of watchdogs even if it wanted to. The shadow-ban step of
the DI sequence in `internal/planet/module.go` is the whole of the clicks side, and what is left in it is genuinely the edge's:
the metric names, the message and attribute names of the ban line, and where
in the chain it sits.

**Detection and consequence are separate, and the consequence is the boring
half.** `antibot/internal/shadowban` takes a scope or an account and a clock and runs a ban. It knows
nothing about tiles, reactions or what earned it, which is why the same sentence
serves three different findings and would serve a fourth.

**A flagged caller's clicks are answered `OK` and dropped.** That is the whole
point — a refusal names the check that tripped, and the author fixes it in an
afternoon; a silent no-op names nothing. It is not permanent (the caller reads
the map back over the same stream and will notice), but it moves the cost of
the next round onto them.

#### Seven watchdogs, one jury

A `Watchdog` measures one behaviour over one caller and returns a `Verdict`:

- **`retaker`** — takes a tile back moments after losing it, over and over.
- **`sequencer`** — walks the tile ids rather than the map: 1, 2, 3, 4, on and on.
- **`metronome`** — never varies and never stops (`cadence`), or sleeps a random time between clicks (`shape`).
- **`defender`** — nearly every take is a retake, however slowly it comes.
- **`catcher`** — catches every bonus box, at once.
- **`cohort`** — starts, paces and stops in step with other scopes, group after group.
- **`scraper`** — reads the whole map again and again, which the web app never does.

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

#### What survives a restart

**Bans and evidence both, in postgres, in the `antibot` schema.** Bans are
`antibot.bans`, one row per scope ever banned, and `antibot.account_bans`, one row per
account, written by `antibot/internal/shadowban`. What
each watchdog is tracking and the jury's record of each caller — its tally and
the last opinion of every watchdog — are `antibot.evidence`, one row per section, written by
`antibot/internal/evidence`. This exists because of 2026-09-14: production
restarted 23 times, every 5-10 minutes, during a bot attack, and with the evidence
in memory no window of 10m (`suspicionWindow`), 15m (`trackWindow`) or 30m
(`certainFor`, the catcher's `trackWindow`) ever filled. Nothing was banned.
`TestALoopRestartedEveryFewMinutesIsStillCaught` replays that, both ways.

- **The antibot owns its schema, its migrations and its pool**, like a module:
  `antiBot.database` (a `cppg.Config`, `schema: antibot`) and
  `antibot/internal/migrations`. It is a library, not a module, but the planet's
  migrations sit behind `planet/internal/` where it cannot reach them, and its
  tables are its own business. `antibot.New` builds the pool and connects nothing;
  `Guard.LoadState(ctx)` connects, migrates and loads, and **an error refuses the
  boot**: a boot that forgets the bans unbans every bot. The guard's `Run` closes the
  pool after the last flush, the way `cppg.CloseAfter` does for the tile map.
- **The tile map's pattern.** `shadowban.Persistence` and `evidence.Persistence`
  are the ports, `postgres_ban_store` and `postgres_evidence_store` the adapters,
  `MemoryPersistence` (behind the `testing` tag) the fakes. State lives in memory
  and is flushed every `saveInterval` (1m), with a 10s timeout, and once more on
  shutdown. `antibot.NewInMemory` (behind the tag) builds a guard over the fakes.
- **Bans are flushed by key.** Scopes and accounts are two `Banner`s over two
  tables, each with its own ladder. A flag or a ban marks the key dirty; a flush
  upserts the dirty keys, as they are then, in one statement per table. A failed flush
  marks them again for the next tick. A row is never deleted: offences are never
  forgotten. `nextFlagAt` is not kept, as it never was.
- **Evidence is flushed whole.** One section per watchdog and one for the jury,
  each encoded by its own package (`state.go` beside it), so a watchdog's fields
  stay unexported. A flush replaces every row in one transaction, so a section
  left out (a watchdog turned off, or one that fails to encode) is deleted, and
  a failed flush is simply retried whole on the next tick. A section that does
  not decode at load is reported through `OnStateError` and starts that one
  watchdog empty; the others load. That is not a failed boot: a changed section
  shape after a deploy should cost one watchdog's windows, not the start.
- **`antiBot.evidence.retention` (72h) is a ceiling on top of every window**:
  evidence older than it is dropped on load and by a sweep before each flush. The
  watchdogs' own sweeps still forget at their `trackWindow`, which is far
  shorter; retention is what bounds the rows after a long outage or a
  `trackWindow` set in days. Since each flush rewrites the rows from memory,
  nothing older than retention outlives one flush.
- An entry goes when its last event does: a metronome
  run or a jury tally still being added to is kept whole, however old its start.
- **Timestamps are wall clock, and the windows stay wall clock.** A restart of
  20 seconds costs every window 20 seconds, and a caller away for an hour is away
  whether or not the process was. Stretching every window by the outage would buy
  nothing measurable and let a long outage hide a real absence.
- **Except where a gap is the signal.** `metronome` ends a run on any gap over
  `maxGap` (3s), and no restart is that short — so a gap spanning the outage
  would read as a break every time, and the half hour of `certainFor` could never
  be reached across restarts. So the outage is taken out of that one gap
  (`detect.Outage`): it runs from the last flush (`saved_at`) to the moment
  the new guard's `Run` starts — not to the load, because the boot is not over
  then, and runners start before the server listens. If what is left is still
  under `maxGap` — the caller was clicking when the process went down and again
  as soon as it came back — the run goes on. That gap is **not a sample**: it is
  stitched from two pieces, and a stitched 0.4s inside a 950ms loop would widen
  the spread like a burst. The outage is not counted in `sustained` either. A
  crash is the same, with the outage starting at the last periodic flush (the
  latest `saved_at`).
- The jury does the same for `longestGap` and `activeFor`, which only feed the
  log line: a restart is not the caller pausing, nor time it was active.
- The jury also keeps when each watchdog last reached each level, so a reading
  standing before the restart is not reported again through `OnRise` after it.
  `cohort` saves its members and rebuilds its indexes; its cached judgement is
  not saved, so a loaded member is judged again on its next click.
- **What this does not change:** the jury refreshes every watchdog's opinion on
  every click before it deliberates, so a saved opinion carries its words into the
  next ban line but never decides one — the verdicts come back because each
  watchdog's own evidence does.

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

**Speed on one tile is not the signal, and speed on many tiles is (`roam`).** The
band missed the Bulgaria recapture bot of 2026-09-14 from the other side: 62
retakes on 61 different tiles, median ~280ms, but a p90-p10 of ~750ms, because its
retakes queue behind the throttle. A player at war is fast only on the tile its
cursor is already on. So reactions on `minTiles` different tiles with a median at
or under `roamMedian` read `Suspect` whatever the spread, and `certainTiles` reads
`Certain`. The ban line names the stronger rule (`reflex` or `roam`) and carries
`tiles` either way. `TestTheSameSpeedOnAFewTilesIsATileWar` pins the player
clicking back at a bot; `TestTheRecaptureBotOfSeptember14IsCaught` replays the bot.

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
the caller is fast. What is measured is `maxSpread` over an unbroken run, where a
pause longer than `maxGap` ends the run and the evidence starts again from nothing.

**The run is timed off every click tried, not every click accepted.** A loop firing
just above the refill rate has some of its tries refused, unevenly, so the gaps
between the survivors are 0.95s, 1.9s, 2.85s — a spread no clock shows. That is
exactly the bot of 2026-09-14: a try every 950ms (±60ms) for twenty minutes, half
of them 429s, read `clear`. So `antibot_attempt_click` sits **outside** the
throttle and hands every try to `Guard.Attempted`; the metronome records there
and only judges in `Watch`. It also helps a player: someone spam-clicking into the
throttle is timed by their own hand, not by the refill rate.

**That run is the answer to a jittered delay.** A spread test is beatable by
construction — randomise and the band widens to look human. What is not cheap to
fake is *stopping*: a person's session has breaks in it. `activeFor` and
`longestGap` still feed no rule, because deciding on them alone would ban the
genuinely obsessed; they go in the log, beside a rule that did fire.

**Stopping was cheap to fake after all, and `shape` is the answer.** The bot of
2026-09-16 slept a random 0.6-2.1s between tries and paused up to 38s every so
often: no run lasted, its spread was ~1.2s, and every watchdog read `clear` for
hours while it rotated Free Mobile /64s. What it did not fake is the *shape* of
its gaps. A sleep drawn evenly from a range sits evenly around its median; a
hand's gaps are mostly short with a long tail. The rule is the quantile skew
`(p90 + p10 - 2·p50) / (p90 - p10)` over the last `shape.clicks` gaps.

- **Its window is not the run.** A gap over `shape.maxGap` (10s) is skipped and
  ends nothing, so a pause does not reset it; a gap over `maxGap` still ends the
  `cadence` run. A gap stitched across a restart is not a sample here either.
- **Measured before it was written.** Over two days of access log, every player
  with 500 gaps read 0.34 or more over any 500-gap window, and the bot's 12
  scopes read under 0.25 in nearly all of them. Replayed through the watchdog,
  `maxSkew` 0.25 and `certainSkew` 0.15 caught all 12 in ~12 minutes and no player.
- **It ships measuring.** `maxSkew` and `certainSkew` are pointers, unset in
  production: 0 is a skew, so leaving a bound out is the only off. The sweep
  reports every caller with a full window through `Observer.OnGapSkew`, into
  `click_gap_skew`. Set the bounds from that histogram.
- **One watchdog, one opinion.** It is a rule of `metronome` and not a watchdog of
  its own because it reads the same gaps: two timing rules in two watchdogs
  could reach `Suspect` together and ban on one behaviour. The stronger level is
  reported, `cadence` on a tie, and the evidence names the rule.
- **It is beatable too**: sleep a lopsided random time and it reads like a hand.
  It buys time, like every rule here.
- A gap that never varies (p90 = p10) has no skew and is `cadence`'s.

**`defender`: what is clicked, not when.** The bots of 2026-09-14 retook from a
queue behind the throttle: tiles came back 0.4s, 1.5s, 2.5s … 40s after they were
lost, one refill at a time, so the retaker's reaction window saw almost none of
it. The rule is the share of a caller's takes, over `trackWindow`, that win a
tile back for the country that lost it within `retakeWindow`. A take for another
country, a take of what the same caller took, a refused click and a no-op are not
retakes.

**It ships measuring.** `minShare` and `certainShare` default to zero, and zero
never reads anything: the watchdog only reports each caller's share once a sweep
through `Observer.OnRetakeShare`, into the `click_retake_share` histogram (a caller
held there five minutes is five samples). That is not caution for its own sake —
two people fighting over one tile retake on every click, and
`TestTwoPlayersFightingOverOneTileReadAsRetakes` pins it. Set the shares from the
histogram, and expect the tile war to be the case that decides them.

Production reads `Suspect` since 2026-09-14 (`minShare` 0.6, `minClicks` 40 over
10m): the histogram held 26 callers under 0.1 and the recapture bot at 0.7-0.8.
`certainShare` stays unset for the tile war. A bot that only answers attacks
clicks a few times a minute, so the old 60 takes in 5m never judged it at all.

**`catcher`: every box, and fast.** A box is addressed to one caller and flies
a slow orbit that is rarely in view, so a person has to zoom out to orbit height
and often drag the globe round to click it, and some boxes go by unseen. A script
reads `bonus_offered` off the stream and claims at once. Over the last
`minCatches` boxes offered (5), **all of them must be caught** — one lapse clears
the caller — and the median delay from offer to claim reads `Suspect` at or under
`maxMedian` (3s) and `Certain` at or under `certainMedian` (1.5s). Neither half is
enough alone: a player already zoomed out gets lucky once, and a keen player
catches a lot.

It is the one watchdog that does not read clicks. The registry reports each box
through `bonuses.Report` — `Caught(scope, after)` from `Claim`, timed from the
offer being sent, and `Lapsed(scope)` from the sweep — and `internal/planet/module.go`
hands both to `Guard.Caught` and `Guard.Missed`. The watchdog keeps the outcomes
and answers from them on the caller's next click, since the jury only asks on a
click. The delay includes the round trip, which only makes a person look slower.
`bonus_catch_seconds` is the same delay as a histogram, whether the antibot is on or not.

The counter-move is cheap — wait a random few seconds, or let one box in five go
— and that is fine: a bot that does either has stopped taking every box the
moment it is offered.

**`cohort`: between scopes, not within one.** Every other watchdog judges one
scope, and a scope is only as long-lived as the caller wants it to be. On
2026-09-14 a pool painted `bg` through Firefox's built-in VPN
(`2a00:8c40:f000::/36`): pairs of /64s whose first takes were milliseconds apart,
~30 tiles a minute each for ~476s, followed at once by the next pair on new /64s.
Each identity started clean and none lived long enough to read anything — zero
`antibot ban` lines all day.

Two scopes are **in step** when both have `minClicks` for one flag
(`minFlagShare`), their first clicks are within `startWindow`, their paces are
within `rateRatio`, and — once the shorter one has been quiet for `quietAfter` —
their lengths are within `lengthRatio`. The first click is timed from `Attempted`,
since a pool starts its tries together and the throttle only blurs that.

- **`Suspect` (`lockstep`)** is `minMembers` scopes in step, from anywhere. Two
  friends joining a flag war in the same second are exactly this, which is why it
  never bans alone. It usually clears on its own: people who start together do not
  stop together, and `TestAPartnerWhoLeavesClearsTheOneWhoStays` pins it.
- **`Certain` (`chain`)** is the scope's group being the `certainCohorts`th
  separate group inside `chainWindow`, painting the same flag from the same wider
  prefix (`v4Bits`/`v6Bits`, a /24 and a /44). **A person keeps their address when
  they come back and a pool does not**, so a chain of fresh scopes is the pattern
  no crowd produces. The prefix is what separates it from a raid — waves of people
  answering one link start together too, but from all over, and
  `TestARaidFromAllOverIsNeverCertain` pins that they never reach it.
- **`Certain` (`crowd`)** is `certainMembers` scopes in step in one group from one prefix.

**How a finding reaches every member.** The jury asks per click of one scope, and
nothing here is pushed to anyone. The watchdog keeps one table of every scope,
indexed by the second of its first click and by its wider prefix, and each member
answers for itself from that table on its own next click (re-judged at most every
2s). A member that has already rotated away needs no ban; it still counts as a
link in the next group's chain. `TestTheRotatingPoolIsCaught` replays the
production pool: the third group is dropped under a minute into its eight, and
the first two are left alone because nothing about them yet is more than a suspicion.

**The flag is an input here**, unlike `topCountry` in the ban line: a pool that
paints another flag to dodge this has stopped painting the one it came for. The
counter-moves that are left cost the pool something too — stagger each start
past `startWindow`, draw its identities from unrelated ranges, or vary pace and
stay length between them. `click_cohort_scopes` is the gauge of scopes in step
right now, set once a sweep through `Observer.OnCohortScopes`: a floor that never drops to zero is a pool, whether or
not its groups have chained yet.

The chain bounds are the only ones in the antibot that `Validate` refuses at
boot (`antiBot.cohort.detector`), because a `minMembers` or `certainCohorts` of 1
would read one scope, or one group, as a pattern.

**`scraper`: what a caller reads, not what it clicks.** The web app reads the map
once per page load, in 26 `GetMap` batches, and follows `ListenForEvents` after
that; the same page load opens that stream. On 2026-09-15 and 16 a script in a
real page read one map chunk after every click, a whole map every half minute,
and painted dz and bg for twenty hours. It jittered its delay, stayed under the
throttle, picked tiles off the map and never retook one: every other watchdog
read `clear`, and the catcher's lone `suspect` banned nothing.

- **The count is maps read beyond one per stream opened**, over `trackWindow`.
  `antibot_get_map` reports each read as a share of the map (two bytes per tile
  in the batch), and `antibot_listen_for_events` each stream as it opens. A page
  load is one of each, so sixty players behind one carrier NAT read `clear` —
  the raw read count would not. A stream reopened after a drop reads nothing,
  and the count never goes below zero.
- **A read outside the map forfeits that credit entirely.** Tile ids start at 1
  and the web app clamps its last batch, so neither bound of a real client's read
  can land outside the map — whatever batch size it picks, which is why this needs
  no agreement with the frontend's `TILES_PER_BATCH` and cannot drift from it.
  `antibot_get_map.offMap` decides it, because the planet module owns the map and
  its bounds; the watchdog only counts. It is reported in the ban line as `offMap`.
- **`certainOffMap` (2) such reads are `Certain` on their own**, however little was
  read, under the rule name `offMap` rather than `poll`. Forfeiting the credit was
  not enough: it only ever lowers a count, so a caller that reads the map twice and
  stops sits under `minMaps` and reads `clear`. One stray read is still forgiven —
  `TestOneReadOffTheMapIsNotEnoughOnItsOwn` — but a walk overruns the end on every
  pass, so two is already a walk. Measured over the access log of 2026-09-16: the
  7 scopes with any off-map read are exactly the 7 known bots, the lowest of them
  at 4 in its worst 15 minutes, and every other scope sits at 0.
- `minMaps` (5) reads `Suspect` and `certainMaps` (15) reads `Certain`. Measured
  over 27 hours of the access log: the bots read 23 to 31 maps in their busiest
  15 minutes, and no human scope read more than 6 — six page loads, in the web
  app's own batches. The log cannot show stream opens (Caddy writes a stream's
  line when it closes), so that 6 is before any credit.
- It is counted in 30 slices of `trackWindow`, so a caller costs the same
  however fast it reads: `GetMap` is not throttled.
- **It only observes.** Reads and the streams are refused by nothing here; a
  banned caller still loads the planet and watches it.
- **The counter-move landed on 2026-09-16, ~1.5h after this shipped**: one scope
  began opening a `ListenForEvents` in the seconds before each 26-chunk read, so
  the credit netted to nothing — 7 maps against 6 streams in its worst 15 minutes,
  against a bound of 5. It jittered its clicks past the metronome too and sat one
  suspect short of a ban while it took 4,829 tiles. What it did not change was the
  walk: every one of its reads started at tile 0 and ran six tiles past the end,
  which no page load does. Measured over the same log, all 22 human scopes read
  from 1 and stopped at the last tile, and all 6 bots did neither.
  `TestAStreamBeforeEveryReadBuysNothingOffTheLattice` pins it, and
  `TestReloadingOverAndOverIsClear` pins that a real reload still costs nothing.
- **The second counter-move landed on 2026-09-16, the same day**: a scope from the
  same carrier range walked the map four times in its first half hour, then
  **stopped reading it altogether** and clicked from the copy — 349 clicks in the
  last window against zero reads. That put it at 5.00 maps in its busiest 15
  minutes, exactly `minMaps`, so the scraper read `suspect`; every other watchdog
  read `clear` and `minSuspects: 2` was never met. It took 1,651 `ps` tiles and 640
  `dz` ones, all of them painted over other flags, and was banned by hand. Caching
  the map is what `certainOffMap` answers: the reads stop, but the walk that made
  the copy is already proof. `TestTheWalkOffTheLatticeIsCaughtBeforeItPaints` and
  `TestACallerThatStopsReadingKeepsItsVerdictForTheWindow` pin it.
- The counter-move left is to walk the map exactly as the web app does — from 1,
  in its steps, clamped — which is also to stop reading it faster than a reload.
  `click_map_reads` is each clicking caller's count once a sweep, through
  `Observer.OnMapReads`.

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

**Only the watching is outside it.** `antibot_attempt_click` wraps the throttle and
reports each try through `Attempted`, which judges and drops nothing. Moving the
whole guard out instead would let a banned caller skip its 429s.

**`antiBot.shadowBan.enforce` is the rollout switch,** the same shape as
`auth.enforce`: false judges, logs and counts without dropping anything. The
two surfaces are built for a box with no dashboard — `click_reaction_seconds` is
a histogram whose raw bucket counts show the bot band by eye, and each flag
writes one `antibot ban` log line carrying the scope, every watchdog's verdict
and numbers (**including the ones that said `clear`** — what did not fire is half
of reading a line that did), the tiles, and the country the caller painted with
most. **The address is never a metric label** — unbounded cardinality, and
personal data in every scrape. That holds for `clicks_total` too, which
`prom_click` labels by `country_id` and `status` only. Per-address click data
is in three places instead: the ledger (every take, read with `FindPlayers` and
`TopPlayers`), `InspectPlayer` (what the antibot holds on one scope), and the
Caddy access log (every request). See [Operator tools](#operator-tools-adminservice). `topCountry` is context for a human reading the
log and never an input to a rule on its own: the client declares it, so it is changed by
editing one string, and real players paint the same flags a bot does. `cohort` only uses the flag
to group scopes that already started together — see above.

**A flag repeats, and that is most of its value.** `reflagInterval` is how soon a
caller already serving a ban can be judged again; at or above a watchdog's
`trackWindow`, each flag rests on evidence the previous one never saw, so
`flags=6` on a line is six independent judgements agreeing rather than one
verdict repeated.

**A day with no ban still says how close it came.** A ban is the only thing
`OnFlag` reports, so on 2026-09-14 a bot attack produced zero lines and zero
metrics about the watchdogs. The jury now also reports, through
`Observer.OnRise` and `Observer.OnStanding`, into `antibot_opinions_total` and
`antibot_opinions_standing`, both `{watchdog, level}`:

- **The counter counts rises, not clicks.** A rise is a watchdog's reading of a
  caller reaching a level it has not held within `jury.suspicionWindow` — the
  same window the jury expires a reading on. A reading flapping across a bound
  every click counts once a window; one that lapses and comes back counts again.
  Per-click would say how often the watchdog was asked, and a sweep sample would
  count a standing suspicion once a minute for as long as it stands.
- **The gauge is set once a jury sweep** (`sweepInterval`, 1m): how many
  callers each watchdog reads at the level now, by the jury's own rule (latest
  reading, expired past the window). Every watchdog and level is set, zero
  included, or a gauge would hold its last non-zero value forever.
- **Levels are cumulative**, like histogram buckets: `level="suspect"` includes
  every `certain`, so a caller going straight to certain rises through both, and
  `suspect − certain` is the near misses.

The level leaves as the string `Verdict.String()` gives, not as a type — the
same rule as `Opinion`: the edge puts it on a label and never compares it.
`jury.Hooks` carries the typed verdict inside the package, and `antibot.New`
words it on the way out.

**The watchdogs judge a scope; a ban falls on the scope and the account.** The
evidence stays keyed on `cpipscope.Of`, so a v6 caller cannot serve a ban on one
address and click from the next in its own /64. A click carries the account its
token names (`Click.Account`), and `shadowban.Bans` passes a ban on:

- **the account**, when the token names one, so it is dropped from any scope;
- **the scope too, when there is no account or the account is a guest's**
  (`Click.SignedIn` false: the token is not linked). A guest can
  shed its account with a new cookie, and must not shed the ban with it. A
  signed-in account's ban leaves its scope alone, so a campus is not banned for one
  player on it.

**A click, and a bomb, are dropped when the scope or the account is banned**
(`Guard.Banned(scope, account)`). `TestAGuestBannedByTheJuryCannotShedItWithAFreshCookie`
and `TestABannedGuestWithAFreshCookieIsStillDropped` pin it. The `antibot ban` line
carries `account`, and `shadowban_flagged` counts running bans on scopes and on
accounts, so a guest banned on both counts twice.

A scope ban only bites a bot with a stable address — against a residential proxy
pool it evaporates for exactly the reason the scope's bucket does; the account ban
is what follows a guest across addresses until it drops its cookie. `cohort` is the
one watchdog that reads across scopes, and it is the answer to a pool that rotates
inside one range.

`inmemory_tile_storage.Owner` exists for this: one indexed read under the existing
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
`send_message_handler` for `messages.ErrInvalidMessage`, `create_session_handler` and `sessionv1controller`
for `attestation.ErrAttestationFailed`. The net never sees those, because a `*connect.Error`
is passed through untouched.

The session one keeps a log line of its own: a refused mint is logged at **Info**
there, because the net logs at Error and a refusal is the check doing its job
rather than a fault of this server — on a public endpoint it is the common case.

### Shared interceptors

`shared/cpconnect` holds the two interceptors both contexts need, because the policy is the same whatever the procedure is — only the procedure names and the wording of the refusal differ, and those are arguments:

- `NewRateLimitInterceptor(limiter, refusal, procedures...)` — a `shared/cpratelimit` bucket keyed on `cpctx.RateLimitKey`, answering `CodeResourceExhausted` (429). It reports nothing about what is left: a context that shows a player their allowance throttles inside its own use case instead, where the reading is a return value. This is for the procedures where a refusal is the whole story — chat and sessions
- `NewIPBlockInterceptor(blocklist, refusal, onBlocked, procedures...)` — a `cpipblock.Blocklist` lookup answering `CodePermissionDenied` (403), with an optional hook the click counter hangs on
- `NewSessionInterceptor(verifier, clock, refusal, enforce, onVerdict, procedures...)` — a `shared/cpsession` signature check answering `CodeUnauthenticated` (401), which puts the session id and the account (when the token names one) on the context and, with `enforce` false, counts without refusing
- `NewSessionReaderInterceptor(verifier, clock, procedures...)` — the same check where a token is optional: a valid one puts the same values on the context, and nothing is refused or counted. `GetBudget` and the chat's `SendMessage` use it
- `NewErrorInterceptor(logger, mapper)` — the net, applied by `cpbootstrap` rather than by any module (see [The error net](#the-error-net)). It is a full `connect.Interceptor` rather than a `UnaryInterceptorFunc`, so it covers the streaming handlers too; without that, a stream would be the one procedure whose raw error the caller sees. The `Mapper` is optional and nothing passes one any more.

Each context keeps a thin named constructor over these — `planetv1controller.NewVPNBlockInterceptor`, `chatv1controller.NewRateLimitInterceptor`, `NewBlocklistInterceptor` and `NewSessionInterceptor` — which is where the procedure list, the refusal wording and the metric live. **A context names its own policy; neither reimplements the mechanism.**

Both chains order them the same way: error mapping outermost, then the blocklist, then the limiter. **The blocklist has to sit outside the limiter** — a refused address must not also spend a token, or its next call would come back 429 and the client would report the wrong reason. `TestVPNBlockRunsBeforeTheThrottle` pins that for clicks.

### Durability

**The map lives in memory; postgres is where it is kept.** A click never waits on the database.

- **Boot loads it.** `inmemory_tile_storage.Load` reads every row of `planet.tiles` — one per owned tile, `(id, country)`; an unowned tile has no row. 180k rows load in about 60ms. **A failed load refuses the boot**: an empty map that then flushes would be every player's territory gone. A row past `gameMap.maxIndex` is skipped and logged.
- **A flush writes what changed.** Every write under the tiles lock sets the tile's bit in a `dirty` bitmap (one bit per tile, ~32 KB). Every `tilesStorage.flushInterval` (1s), `Flush` takes the bits, reads each tile's owner **as it is now**, and hands them to `postgres_tile_store.Save`: one transaction, an upsert for owned tiles and a delete for freed ones, in chunks of 10k. A tile clicked five times between flushes is written once. A failed save puts the bits back; the next tick retries. Each flush has a 10s timeout, so a stuck connection cannot stall the loop.
- **Shutdown flushes once more**, from `Run`. That is also why the tile map's pool is closed by its runner rather than registered on `props.Closers`: closers run before the runners stop, so the last flush would find the pool already closed.
- **What a hard kill loses** (`SIGKILL`, OOM, power loss) is bounded by `flushInterval`. The state is still per-process, so **this is single-instance only**: two API replicas would each hold their own divergent map.

**Each module owns a postgres schema**, set in its own database block (`database.schema: planet`), and its migrations: `internal/planet/internal/migrations`, embedded golang-migrate pairs. The module migrates it while it builds, right after connecting — `cppg.Migrate` creates the schema if it is missing and keeps the history in that schema's own `schema_migrations`, so two modules never share a migration history or a migration lock. Every connection sets `search_path` to the module's schema, so the SQL says `tiles`, not `planet.tiles`, and no module's adapter can reach another's tables by accident. `TestEachSchemaHoldsItsOwnTablesAndMigrationHistory` pins it.

**`shared/cppg` is the client**, ported from on-core-platform's `onpg`: `Config` (one module's database block, schema included), `New`, `ConnectCtx`, `Migrate`, and the `Querier`/`Beginner` interfaces a store depends on. Cut from the original: gorm (`make deadcode` rejects what nothing calls, and plain SQL is enough), the SSM tunnel, tracing and lazy config. **Every module that stores something has its own database block and its own pool.** The planet's is `database:` at the top of the file, because `planet.Config` is squashed there; another module's would sit inside its own section (`chat.database:`). Nothing is handed between modules. `Config.String` leaves the password out of the boot's config log line.

**The chat has its own block, `chat.database` (schema `chat`), its own pool and its own migrations** (`internal/chat/internal/migrations`). It connects and migrates inside chat's DI sequence, and `chat.Config.Validate` refuses an incomplete block. Its pool is closed by `cppg.CloseAfter` around the prune runner, like the planet's around its storage runners. In production both blocks point at the same postgres and user. See [Chat](#chat-internalchat).

**The ledger follows the same pattern**, through `inmemory_ledger_storage.Persistence` and `ledger/postgres_ledger_store`, on the tile map's pool.

- **Four tables.** `ledger_takes` is one row per take, keyed by its position, with the take's `account` (NULL for none, and for every take made before accounts). `ledger_head` is one row: the oldest position kept, so positions carry on past a ledger the retention emptied. `ledger_forgotten` is a reverted scope's mark, and `ledger_forgotten_accounts` a reverted account's.
- **Boot loads it**, takes in position order, then the marks. **A failed load refuses the boot.** Measured at 1M takes on a laptop: 95 MiB of table, 0.8s to load, 1.3s to copy in — so ~380 MiB, ~3s and ~5s at the 4M cap.
- **A flush appends, it never rewrites.** Every `ledgerStorage.flushInterval` (1s), `Flush` hands the takes past the last flush to `Save`: one transaction that `COPY`s them in, deletes the takes before the head (what the retention or the cap dropped), moves the head, and upserts the marks set since. It first deletes any row at or past the first new position, so a flush whose commit answer was lost writes again without a conflict. A take dropped before it was flushed is never written. A failed save keeps it all for the next tick; each flush has a 10s timeout, and shutdown flushes once more.
- **One pool for every runner.** `cppg.CloseAfter(db, logger, tilesStorage, takings, charges)` runs them together and closes the pool after the last flushes.

**The charges follow it too**, through `inmemory_charge_storage.Persistence` and `bonuses/postgres_charge_store`, on the same pool: one row per account in `planet.charges`, written every `chargeStorage.flushInterval`. See [Charges](#charges-refill-bomb-enclose-spread).

The antibot's bans and evidence are in postgres too, in the `antibot` schema, the same way — see [What survives a restart](#what-survives-a-restart). The player module's profiles and stats are in postgres too, in the `player` schema, but with no memory copy: each call reads or writes the table — see [Player](#player-internalplayer).

Nothing lives in files any more: the container mounts no state volume.

### Operator tools (`AdminService`)

**A second router, on a loopback listener.** `props.AdminRPC.Mount` is `props.RPC.Mount` for services an operator calls: same builder, same error net, but `cpbootstrap` serves them on `httpServer.adminBindAddress` instead of the public router — logging middleware only, no CORS. Empty serves no admin listener; anything but a loopback `host:port` refuses the boot, both in `ServerConfig.Validate` and again in `Run`, and a port already taken refuses it too. They have no authentication, so loopback is their whole protection, and they are off the router Caddy forwards to on purpose: one Caddyfile edit would otherwise let anybody repaint the map. In production they are reached with `docker compose exec backend wget`; see `deploy/vps/README.md`, "Operator tools".

`planet.v1.AdminService` is the one there today, in `proto/planet/v1/admin.proto`. `planetv1controller.AdminService` is its bag of handlers, the way `ClickService` is. `ReassignCountry` runs `clicks/usecases/reassign_country_usecase`, wrapped in `audit_reassign`: every tile `from_country_id` holds goes to `to_country_id`, while the game runs.

- **The move is paced.** `inmemory_tile_storage.Reassign` moves one batch under the lock and returns where to resume; the use case sleeps 50ms between batches. A batch is a quarter of `tilesStorage.subscriberBuffer`, because each tile is one update on every open stream and the clicks still arriving need the rest of the buffer.
- **Each tile is an ordinary `TileUpdate`** with `Previous` set, not a new event kind: open clients repaint with no frontend release, `counts` move so the toll prices the next click right, and `dirty` puts it in the next flush.
- **A tile `from` retakes behind the scan stays theirs.** The answer reads both counts again at the end, so `from_after` says whether to run it again.
- **`audit_reassign` logs every call at Warn**, dry runs and failures included: it is the only record that those tiles did not change hands through play. It is a decorator for the reason `prom_click` is — handlers here do not log.

Measured on a copy of production's map, before postgres: 22,040 tiles in 4.4s, all 22,040 updates delivered to an open stream, none dropped.

#### `PaintRandomTiles`

`PaintRandomTiles(flag, area, count, proximity, dry_run)` runs `clicks/usecases/paint_random_tiles_usecase`, wrapped in `audit_paint_random`: it paints `count` tiles with `flag`, starting on `area`'s ground (from `clicks.Borders`), or anywhere on the map when `area` is empty.

- **The area is where a patch starts, not a wall.** A fresh draw (a seed) is a tile of the area not wearing the flag; `eligible` counts those. With no area every tile of the map is a seed, `proximity` still grows patches, and `outside_area` is 0. A patch grows into any tile not wearing the flag, across the border too; `outside_area` counts what it took there.
- **`clicks.Pick` is the rule.** Before each draw, with probability `proximity`, it takes an eligible tile touching one already picked (`Geography.Neighbours`); otherwise a seed. 0 is uniform over the area; 1 grows one patch and draws a new seed only when the patch is walled in. Between the two you get a few patches. When the seeds run out, patches keep growing; `picked` is below `count` only when nothing is left.
- **The paint is `Restore`**, the revert's compare-and-set, against the owner read at the pick. A tile somebody takes in between stays theirs, so `painted` can be below `picked`. Paced like the reassign, each tile an ordinary `TileUpdate`. It does not write the ledger, like the reassign.
- The draw is `clicks.SystemRandom`, math/rand/v2's global source; tests pass a seeded `*rand.Rand`.

#### Manual bans: `FindPlayers`, `TopPlayers`, `BanPlayer`, `RevertPlayer`, `InspectPlayer`

For the patterns no watchdog catches but a person sees on the map. A player is an **account on a scope** (`cpipscope`: the address over IPv4, the /64 over IPv6), or a scope alone for takes made with no account. `BanPlayer`, `RevertPlayer` and `InspectPlayer` take a `scope` (any address) **or** an `account_id`, never both (`ledger.ParseCaller`; both, neither or a malformed id is `InvalidArgument`).

- **`ledger` remembers every take**: tile, scope, account, country, previous owner and time, oldest first. `ledger.Recording` wraps the storage the click chain writes through — the rule, `spread_click` and the enclose annexer — so every tile a click takes is recorded, a no-op is not, and a click the shadow ban drops never reaches it. Recording appends through `publishing_ledger_storage`, which publishes `planet.v1.TileTaken` for each take with an account, after it is recorded. A take by somebody else is one more take, not a replacement: a bot painted over as fast as it paints is still in the ledger. Bombs and reassigns write nothing; they show as a change the ledger never saw.
- **The rules are in the `ledger` root, and its package doc states them**. A caller (`ledger.Caller`, a scope or an account) **holds** a tile when the tile's latest take is its own and the tile still wears that paint. A revert gives a held tile back to what it held before the caller's **current run** on it: its own latest takes, walking back while each took the tile from the paint of the one before. An account's run follows it across scopes. Another scope's take breaks the run (A il→ps, B ps→de, A de→ps goes back to `de`), and so does a change the ledger never saw (A il→ps, bomb, A ""→ps goes back to nobody). `Tally` gathers players for `FindPlayers` and `TopPlayers`, `Runs` computes the revert, `ByTakes` and `Top` rank and cut. The use cases only replay the ledger into these, filter through their ports, and call them. The tests for each interleaving are in `ledger_test.go`.
- **Kept in memory and flushed to postgres** (`inmemory_ledger_storage`, behind `ledger.Storage`; see [Durability](#durability)). In memory it is an append-only log of 20-byte records in 1.25 MiB chunks, scopes and accounts interned in one table per chunk and countries in another, so an old chunk takes its strings when it goes. A record is never changed once written, so `Replay` copies the chunk headers under the lock and reads without it: a `TopPlayers` over 4M takes takes ~1s and never blocks a click. `Forget(caller, position)` hides a reverted scope's or account's takes up to the replay it was computed from, so a take made mid-revert still counts.
- **Bounded twice.** `ledger.retention` (72h) drops takes by age, each `ledger.sweepInterval`; `ledgerStorage.maxTakes` (4M) drops the oldest first when a busy stretch fills it, and logs "the ledger is full" once. Production is thousands of clicks per 5 minutes (`clicks_total`), and a spread click takes up to 7 tiles: 15 takes a second fill 4M in three days. Measured at 4M before the account: ~85 MiB heap. The account adds 4 bytes a take, about 15 MiB more at the cap (not measured).
- **`FindPlayers(flag, area, limit)`** lists every player that took a tile for `flag` on `area`'s ground (empty is the whole map), latest take first, with any running ban on its scope or its account — whichever ends last. Each account on a scope is its own row, with `account_id`. The ground comes from `clicks.Borders`, built by `embedded_geodesic_map.Loader.LoadBorders` from the borders blob the frontend paints flags from — see [Map geography](#map-geography). A blob for another map refuses the boot.
- **`TopPlayers(limit)`** is the same over every flag and the whole map, **most takes first, then most tiles held**, then latest take. Takes lead because they are what a painted-over bot cannot hide. Every player, in both answers, carries `tiles` (held) and `takes` (every take, a tile taken twice counting twice), `active_for` (last take minus first take), and `tiles_per_minute` and `takes_per_minute` over that. A player with `takes` high and `tiles` near zero is painting and being painted over.
- **`BanPlayer(scope | account_id, duration)`** is `shadowban.Bans.Ban`, and drops the caller's clicks and bombs alike: the same record, ladder and table as a watchdog's ban, and it counts as an offence. It skips `reflagInterval`, and an empty duration takes the ladder's step. Any address is accepted and banned as its scope (`cpipscope.Parse`). **An account is banned alone**: the operator named no scope, and a guest that sheds it with a new cookie is a second ban on its scope away. **It follows `antiBot.shadowBan.enforce`**, and says so in `enforced`. With `antiBot.enabled` false it answers `FailedPrecondition`.
- **`RevertPlayer(scope | account_id, dry_run)`** gives each tile the caller holds back to what it held before its run, by the rule above. A scope reverts every account's takes on it; an account reverts its takes from every scope. `touched` is the tiles it took, `held` those it still holds. **Only a tile still wearing the scope's paint changes** — `inmemory_tile_storage.Restore` is a compare-and-set under the lock, so a tile retaken mid-revert stays retaken. Paced like the reassign (`clicks.Pacing`), each tile an ordinary `TileUpdate`. A tile that was nobody's goes back to nobody, as an update with an empty country. It then forgets the caller's takes, so a second run does nothing; an interrupted one forgets nothing and can be run again.
- **Ban before reverting**: an unbanned player repaints behind the revert.
- **`InspectPlayer(scope | account_id)`** answers how close the antibot is to a caller, which the `antibot ban` log line cannot: it is only written when a ban fires, so on 2026-09-14 a day of bots and no bans left nothing to read. It is `Guard.Examine`. **An account is read on the scope of its latest take** in the ledger, because the watchdogs judge scopes, with the bans on both; an account with no take inside the retention answers its bans alone and an empty `scope`. It changes nothing — no caller record is created, no watchdog is asked again, no ban is passed. It answers any running ban (`banned`, `bannedUntil`, `offence`, `flags`); per watchdog its `level` and `evidence`, aged the way the jury ages them (past `suspicionWindow` a verdict reads `clear` but keeps its evidence); `suspects` against `minSuspects` and `guilty`, what the jury would decide on a click now (the ban itself would still wait for `reflagInterval`); and the click summary the ban line carries. `tracked` false is a scope the jury has not seen inside its `trackWindow`. Parsed with `ledger.ParseCaller` and refused with `FailedPrecondition` when `antiBot.enabled` is false, as `BanPlayer` is. **A watchdog that reads `clear` has no evidence**: watchdogs only word the rule that tripped, so it says how close a caller is only once some rule has.
- `audit_ban` and `audit_revert` log every call at Warn, as `audit_reassign` does. `FindPlayers`, `TopPlayers` and `InspectPlayer` are reads and log nothing.

### Shared (`internal/shared/`)

Shared infrastructure: `cpbootstrap` (the composite layer), `cpcountries`, `cpconfigs` (YAML + env config via koanf), `cphttpserver` (middleware, formats), `cpprom` (Prometheus), `cptime`, `cpctx`, `cpconnect`, `cpratelimit`, `cpipblock`, `cpipscope`, `cpsession`, `cpsecrets`, `cppg` (postgres — see [Durability](#durability)), `cpcolls` (collections).

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

`cpbootstrap` runs the modules — see [The composite layer](#the-composite-layer). It knows nothing about this game: it takes a list of modules, builds each one under a startup deadline, mounts what they claimed on one router, and serves until the process is signalled.

Two of these are here because both bounded contexts need them and neither should depend on the other:

- `cpsecrets` — the random hex a config may leave it to the server to invent. The player module's tag salt is the only one left, and it costs nothing: the tag never leaves memory. The click token's key is not invented, because a key the server made up could not verify what another restart had minted — see [Sessions](#sessions-internalauth).

`cpsession` mints and verifies the click token — see [Sessions](#sessions-internalauth). It is here because **both** contexts read it: `auth` mints with its `Signer`, `planet` verifies with its `Verifier`, and neither may depend on the other. The two halves are separate types over separate config, so what each context can do with it is decided at compile time.

The siteverify client it is fed by is **not** here. `turnstile` sat here on the same "both contexts need it" rule, but only one ever did, so it now lives at `session/internal/turnstile` where the compiler keeps it. **The bar is not that a package is shareable, it is that it would read the same in any other program and that two modules actually import it** — "shared" names the symptom, and a directory admitted on the weaker reading becomes a dumping ground. `cpsecrets` passes narrowly — chat is its only caller today, but it is twenty lines of `crypto/rand` with no domain in it at all.

`cpcountries` is the ISO country list both the tile game and the chat validate against. `cpipblock` is the VPN prefix set — see [VPN blocklist](#vpn-blocklist). `cpratelimit` is a keyed token bucket held in this process, like the tile map it protects — with one API instance, a shared counter would buy nothing. Its `Run` loop periodically forgets the buckets that have refilled to capacity, which is free: such a bucket holds exactly what a freshly created one would, and without it the map would keep an entry per address that ever clicked.

`cpipscope` decides what a scope's bucket is keyed on, and every throttle goes through it. Over IPv4 that is the address; over IPv6 it is the surrounding **/64**, because the smallest allocation a subscriber receives is a /64 and most receive far more — a bucket per v6 address is one the same line walks out of by picking its next address, turning one home connection into thousands of callers with a throttle each. The session token binds to the same unit, so the address a token is valid for and the address that spends a budget cannot diverge. Blocking deliberately does **not** use it: the VPN and datacenter lists are precise prefixes already, and widening a hit to the surrounding /64 would refuse neighbours who are not on them.
`cpcolls` holds the collections the standard library does not — today `Set[T]`. **A set is a `*cpcolls.Set`, never a map.** A `map[T]struct{}` or a `map[T]bool` written `m[k] = true` fails `make lint`: a ruleguard rule in `tools/ruleguard/rules.go`, run by gocritic, reports both everywhere but in `cpcolls` itself. A nil `*Set` reads as empty, like a nil map, so a lookup in a map of sets needs no `ok` check.

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
- **`clicks/embedded_geodesic_map`** — the recovery. Everything in it exists because the
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
`embedded_geodesic_map` is tested against the real blob, and holds every number below.

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
1-based, which is what `clicks.Board`, `inmemory_tile_storage`'s unused slot 0 and the
frontend's `integerToColor(i + 1)` all agree on. Getting it wrong shifts every neighbourhood by one
tile, **symmetrically, with a degree histogram that still looks right** —
`TestTileIDsAreOneBasedOverTheBlob` is what catches it.

#### Checked at boot, not just in the tests

`embedded_geodesic_map.Loader.LoadGeography` is the first thing the planet module loads and **fails the boot** on a blob whose
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

`clicks.Borders` is the other half of the geography: which country's ground a tile sits on, from
`generated/map/borders-<hash>.bin`, the table the frontend's `npm run borders` writes to `/map`.
Only the operator tools read it — see [Manual bans](#manual-bans-findplayers-topplayers-banplayer-revertplayer-inspectplayer).

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

Config is loaded from a YAML file, with environment variables overriding it — `.` is the nesting delimiter, so `database.password=...` in the environment overrides the file. See `cmd/api/example.yaml` for the full schema.

**A config that implements `Validate() error` is asked to check itself**, and the load fails with its sentence wrapped in `cpconfigs.ErrValidation`. That is where a bad setting is refused out loud rather than becoming a zero value nothing reports.

**Every block validates its own, and `cmd/api` only fans out:**

```go
func (c Config) Validate() error {
	return errors.Join(
		c.HTTPServer.Validate(),
		c.Clicks.Validate(),
		c.Auth.Validate(),
		c.Chat.Validate(),
		c.Player.Validate(),
	)
}
```

The binary never reads inside a block to check it, so a new bound is added in the module that owns it and nothing here changes. `errors.Join` also means a broken file reports **everything** wrong at once rather than one line per restart.

- `cpbootstrap.ServerConfig` — `bindAddress` empty listens on port 80; `adminBindAddress` set to anything but loopback
- `shared/cppg.Config` — a connection setting or the schema left empty, or a schema that is not a plain lowercase identifier. Each module checks its own block, starting with `planet.Config`
- `planet.Config` — `gameMap.maxIndex` zero is a map that refuses every click, plus whatever `bonus` and `antiBot` refuse of their own
- `shared/cpsession.SignerConfig` — `secret` empty, not hex or not 32 bytes while `enabled`, and a negative `ttl`. Checked by `auth.Config`, which is the only block that declares it. `cpsession.VerifierConfig`, which planet declares, has nothing to check: two switches and no key
- `chat.Config` — an incomplete `chat.database` block. Every other chat setting has a usable default

There is no struct-tag validation and therefore no validator dependency — a hook the config implements covers this app's needs.

**Each module owns its own config struct** — `planet.Config`, `chat.Config`, `auth.Config`, `player.Config` — and `app.Config` is the four of them plus `httpServer`. The planet keys stayed at the top level of the file rather than moving under a `planet:` section: `app.Config` squashes that struct (`koanf:",squash"`), so the file and every `deploy/` environment variable are unchanged.

- `httpServer.bindAddress` — the encoding is negotiated per request, so there is no format setting.
- `httpServer.streamHeartbeat` — how often a silent live stream sends a heartbeat (default 30s). **Must stay well under the proxy's idle cut**: Cloudflare answers 524 at ~125s, and a stream that never speaks is one it kills.
- `httpServer.adminBindAddress` — where the operator services listen (see [Operator tools](#operator-tools-adminservice)); empty serves none, and a non-loopback address refuses the boot
- `httpServer.internalBindAddress` — where the services other modules call listen (see [Calling another module](#calling-another-module)); empty serves none, a caller that dials it then fails, and a non-loopback address refuses the boot
- `httpServer.allowedOrigin` — the frontend's exact origin (`scheme://host[:port]`, no path). The public router answers CORS for it alone, with `Access-Control-Allow-Credentials: true`, so the `cp_sid` cookie travels on the web client's mint. **Empty, `*` or anything that is not an origin refuses the boot**: a browser drops a credentialed answer that allows every origin, so a wrong value would only show up as every mint starting a new guest. In production it comes from `FRONTEND_ORIGIN`, the value the Caddyfile allows, which sets the same headers and overwrites these.
- `gameMap.maxIndex` — total number of tiles
- `database.host`, `port`, `user`, `password`, `dbName`, `sslMode`, `schema`, `pool.*` — the planet module's postgres and the schema its tables live in; any of them but `password` and `pool` empty refuses the boot. `database.password` belongs in the environment
- `tilesStorage.flushInterval` — how often the tiles changed since the last flush are written to postgres (1s)
- `ledger.retention`, `ledger.sweepInterval` — how long the operator tools can trace and revert a take (72h)
- `ledgerStorage.flushInterval` — how often new takes are written to postgres (1s, and on shutdown)
- `ledgerStorage.maxTakes` — the most takes kept (4M, ~100 MiB); past it the oldest go before the retention
- `tilesStorage.subscriberBuffer` — per-subscriber channel capacity, which is now per connected client rather than per fanout; updates for a subscriber that cannot keep up are dropped, not blocked on
- `rateLimiter.perSecond`, `rateLimiter.burst`, `rateLimiter.sweepInterval` — one account's click allowance, and a token with no account's scope bucket (defaults 1/s, burst 10, swept every minute; `perSecond` is a float, so 0.2 is one click every 5s)
- `rateLimiter.scopeMultiplier` — the scope's bucket over one account's, shared by every account behind the address (default 10). Below 1 refuses the boot. See [Two buckets per click](#two-buckets-per-click)
- `vpnBlocklist.enabled`, `vpnBlocklist.includeDatacenters`, `vpnBlocklist.allow` — the VPN refusal (see [VPN blocklist](#vpn-blocklist)); disabled parses nothing and allocates nothing
- `bonus.interval` — how often a box is put in front of somebody; a ceiling, since nothing is offered while nobody is watching
- `bonus.offerTTL` — how long the token stays good; **must outlast the flight the client draws**, or a box caught on its last frame is refused
- `bonus.kinds` — a weight per kind (`refill`, `spread_clicks`, `bomb`, `enclose_clicks`); a kind's chance is its weight over the sum. Left out or 0 is never offered, empty offers every kind equally, and an unknown kind, a negative weight or all zeros refuse the boot
- `bonus.spread.clicks`, `bonus.spread.maxPerBox` — the most spread clicks held (8, about 56 tiles, a bomb's worth), and the most one box adds (4; it draws 1 to that). A count, not a time: a timed spread let a full bank of clicks be dumped inside it
- `bonus.enclose.held`, `bonus.enclose.maxPerBox`, `bonus.enclose.maxTiles` — the most enclosures held (3), the most one box adds (3; it draws 1 to that), and the most tiles one shape may take (25)
- `chargeStorage.flushInterval` — how often the charges that changed are written to postgres (default 1s); also flushed on shutdown
- `bonus.maxChargesPerHour` — the most charges one caller may be granted per hour (12); past it the slot is lost
- `antiBot.enabled` — off registers nothing and measures nothing
- `antiBot.shadowBan.enforce` — off judges, logs and counts without dropping; the mode to deploy in
- `antiBot.shadowBan.banDurations` — the ban for each offence (the last step repeats). An offence is a ban that starts while none is running; a flag on a running ban only extends it. **Offences are never forgotten**
- `antiBot.database` — the antibot's own `cppg.Config`, `schema: antibot`; required when `antiBot.enabled`. A failed connection, migration or load refuses the boot
- `antiBot.shadowBan.saveInterval` — how often changed bans are written to `antibot.bans` and `antibot.account_bans` (1m, and on shutdown)
- `antiBot.shadowBan.reflagInterval` — how soon a banned caller can be judged again
- `antiBot.jury.minSuspects` — how many watchdogs at `suspect` make a ban; one at `certain` bans alone
- `antiBot.jury.suspicionWindow`, `trackWindow`, `sweepInterval` — how long a verdict stands while another watchdog catches up, and how long a silent caller is remembered
- `antiBot.evidence.saveInterval`, `retention` — how often every watchdog's evidence and the jury's record are written to `antibot.evidence` (1m, and on shutdown), and the oldest kept (72h) on load and in memory. See [What survives a restart](#what-survives-a-restart)
- `antiBot.retaker.enabled`, `detector.reactionWindow`, `minReactions`, `maxSpread`, `maxMedian` — what counts as a reaction, how many are needed, and the band that reads `suspect` then `certain`
- `antiBot.sequencer.enabled`, `detector.minSteps`, `minShare`, `certainSteps`, `certainShare` — how long a run of constant-stride clicks must be, and how much of it must sit at that stride
- `antiBot.metronome.enabled`, `detector.maxGap`, `maxSpread`, `minClicks`, `certainFor`, `certainClicks` — what ends a run, how tight its gaps must be, and how long it must hold; `detector.shape.maxGap`, `clicks`, `maxSkew`, `certainClicks`, `certainSkew` — the longest gap sampled, and the skew of the last gaps that reads each level (unset, it only measures)
- `antiBot.defender.enabled`, `detector.retakeWindow`, `minClicks`, `minShare`, `certainClicks`, `certainShare` — what counts as a retake, and the share of takes that reads `suspect` then `certain`; a zero share never reads
- `antiBot.cohort.enabled`, `detector.startWindow`, `minClicks`, `minFlagShare`, `rateRatio`, `lengthRatio`, `quietAfter`, `minMembers` — what makes two scopes in step, and how many of them read `suspect`
- `antiBot.cohort.detector.v4Bits`, `v6Bits`, `certainCohorts`, `certainMembers`, `chainWindow` — the prefix a chain must share, and how many groups, or scopes in one group, read `certain`. Its `trackWindow` is raised to `chainWindow` if shorter; bad bounds refuse the boot
- `antiBot.catcher.enabled`, `detector.minCatches`, `maxMedian`, `certainMedian` — how many boxes in a row must all be caught, and the median offer-to-claim delay that reads `suspect` then `certain`. Its `trackWindow` must hold `minCatches` boxes at `bonus.maxInterval` plus `bonus.offerTTL`
- `antiBot.scraper.enabled`, `detector.minMaps`, `certainMaps` — the whole maps read beyond one per stream opened, inside `trackWindow`, that read `suspect` then `certain`
- `antiBot.scraper.detector.certainOffMap` — the reads off the map, inside `trackWindow`, that read `certain` however little was read; one stray read is forgiven
- every watchdog also takes `detector.trackWindow` and `detector.sweepInterval` — how far back its evidence counts, and how often what can no longer matter is forgotten
- `auth.enabled` — off registers nothing, so `auth.v1` and `session.v1` 404 and clicks are judged on address alone
- `auth.enforce` — off counts what enforcing would refuse without refusing it; the mode to deploy in
- `auth.secret` — the Ed25519 seed the tokens are signed with, 32 bytes as 64 hex characters (`openssl rand -hex 32`); **required once `auth.enabled` is true**, and an empty or malformed one refuses the boot rather than being invented. It is the only key in the file: `planet` asks `auth` for the verifying half over the internal listener
- `auth.ttl` — how long a minted token is accepted (default 1h)
- `auth.rateLimiter.*` — the per-IP throttle on both `CreateSession` paths together, same shape as `rateLimiter`
- `auth.turnstile.enabled` — off mints for anyone who asks, which is how a local backend runs without a widget
- `auth.turnstile.secret` — the widget's secret half, from the environment
- `auth.turnstile.hostnames` — the frontend origins siteverify must report; **empty refuses every token** rather than accepting any, and a production value must not include `localhost`
- `auth.turnstile.action` — must match the widget's `data-action` (default `session`)
- `auth.database.*` — accounts and their sessions, same shape as `database`, schema `auth`; required when `auth.enabled`. `auth.database.password` belongs in the environment
- `auth.sessions.guestTTL`, `auth.sessions.linkedTTL`, `auth.sessions.extendEvery` — how long an idle guest keeps its cookie (90 days), how long an idle signed-in account does (30 days), and how often using a session extends it (24h)
- `auth.signIn.enabled` — off, `StartSignIn` and `CompleteSignIn` answer `Unimplemented` (404). On, at least one provider needs a `clientId`
- `auth.signIn.redirectUrl` — the frontend callback page, registered with every provider exactly (production `https://clickplanet.lol/auth/callback`); not an absolute URL, or one with a query, refuses the boot
- `auth.google.clientId`, `auth.discord.clientId` — a provider is offered once its id is set
- `auth.google.clientSecret`, `auth.discord.clientSecret` — from the environment; empty beside a set `clientId` refuses the boot
- `auth.prune.idleFor`, `auth.prune.interval` — how long a guest goes unused before it is deleted (default `guestTTL`, never less), and how often the prune runs (1h)
- `chat.database.host`, `port`, `user`, `password`, `dbName`, `sslMode`, `schema`, `pool.*` — the chat module's postgres, same shape as `database`; any of them but `password` and `pool` empty refuses the boot. `chat.database.password` belongs in the environment
- `chat.storage.historySize`, `chat.storage.retention`, `chat.storage.pruneInterval`, `chat.storage.subscriberBuffer`
- `chat.service.maxTextLength` — the bound on a message in runes (280)
- `chat.rateLimiter.*` — the per-IP `SendMessage` throttle, same shape as `rateLimiter`
- `chat.reactionLimiter.*` — the per-IP `React` throttle, same shape; its defaults suit it
- `chat.blockedIPs` — prefixes refused every chat RPC, parsed by `shared/cpipblock` exactly as `vpnBlocklist.allow` is
- `player.tagSalt` — salts the hash of an address the roster caps its visits with. It is never shown, so empty, which generates one at boot, costs nothing. Production reads it from `CHAT_TAG_SALT`, the chat's old variable
- `player.database.*` — profiles and stats, same shape as `database`, schema `player`; required. `player.database.password` belongs in the environment

### Protobuf

API contracts live in the monorepo-shared [`/proto`](../../proto) (also used by the frontend), one package per bounded context: [`planet/v1/planet.proto`](../../proto/planet/v1/planet.proto), [`chat/v1/chat.proto`](../../proto/chat/v1/chat.proto), [`auth/v1/auth.proto`](../../proto/auth/v1/auth.proto) and [`player/v1/player.proto`](../../proto/player/v1/player.proto). A module's in-process events sit beside them in `events.proto` (`planet/v1`, `auth/v1`), and what other modules call in `internal.proto`. Generated code goes to `generated/proto/`. Use `make proto` to regenerate after editing `.proto` files (requires the `buf` CLI, plus `protoc-gen-go` and `protoc-gen-connect-go` on `PATH`).

`generated/` is for everything the root owns and this app carries a committed copy of, because the Docker build context is this directory: `generated/proto` from [`/proto`](../../proto) via `make proto`, and `generated/map` from [`/map`](../../map) via `make map` — see [Map geography](#map-geography). Nothing in there is edited by hand; run the target.

The proto package is the **only** version number: Connect derives each route from it, and the controller and connect package names follow — `planet.v1` gives `planetv1controller` and `planetv1connect`, `chat.v1` gives `chatv1controller` and `chatv1connect`. There is no gRPC here — Connect serves the service definitions over ordinary HTTP/1.1 POSTs (and h2c, for clients that want it).

### Testing

Tests use `testify`. **A postgres store's own tests need Docker**, and nothing else does: a suite starts one `postgres:16-alpine` container in `SetupSuite` with `cppg.StartTestServer(t)` (behind the `testing` tag), opens and migrates its schema with `OpenSchema(t, schema, migrations.FS)`, and empties it in `SetupTest` with `Purge`. The container stops when the suite ends. There is no container shared across packages: `go test` runs each package as its own process, up to `-p` (GOMAXPROCS) at once. Everything above a store is tested against a fake of its port (`inmemory_tile_storage.MemoryPersistence`), so it runs without Docker.

**A path through a booted module is tested in `e2e/`**, never in `cmd/api`, which only holds the config and the module list. A test there boots the modules it needs with `cpbootstrap.Run` on a test postgres (`TestServer.ConfigFor(schema)`) and calls them over the wire: `accounts_test.go` mints a guest account, brings it back with its cookie, and checks the deprecated `session.v1` path still mints with none; `sign_in_test.go` links a provider, signs in to a known identity, signs out and deletes, over fake providers; `player_test.go` boots auth (with fake providers), planet, player and chat, clicks with a guest's token and reads `GetStats`, links an account to set a username (and sees a guest refused and a taken name refused), and deletes the account to see both go; `chat_test.go` posts on the same stack as a player with a username and a guest, sees the guest keep one code, and a sender with no token refused.

On macOS, testcontainers asks the Docker credential helper before it pulls an image, and that can hang with no prompt in a non-interactive shell. `docker pull postgres:16-alpine` once from a terminal avoids it.

**Tests build with `-tags testing`, so use `make test` rather than a bare `go test ./...`.** Anything else that loads test files needs the tag too: `go vet -tags testing ./...`, and an editor's language server (`gopls` `buildFlags: ["-tags=testing"]`, or `go.buildTags` in VS Code), which otherwise reports the helpers as undefined. A helper that more than one package needs cannot live in a `_test.go` file, so it lives in an ordinary `.go` file carrying `//go:build testing`. The tag, not a filename convention, is what keeps such a helper out of the production binary — and what lets `make deadcode` tell a helper apart from production code. `cpctx.GetSessionID`, `cpconfigs.FromFile`, `cppg.StartTestServer`, `cpsession.TestKeyPair` (one fixed Ed25519 pair, so a test names both halves without carrying two magic strings), `cpbootstrap.RecordedEvents` (the bus, for a use case that only publishes) and `cptime.FixedClock` — the stand-still clock a dozen test packages drive time with — are the shared ones.

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

- `nilnil` — three constructors return `(nil, nil)` for **"this feature is off"**,
  and the caller checks for nil and mounts nothing. A sentinel would make every
  caller unwrap one. Annotated per site so an *accidental* `nil, nil` is still caught.
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
