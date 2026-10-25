# The composite layer

Each context wires **itself**, in a `module.go` at its root (`internal/planet/module.go`, `internal/chat/module.go`, `internal/auth/module.go`, `internal/player/module.go`). That file is the context's manifest: its `Config`, whether it is on, and its DI sequence. **A module takes its config and nothing else, and builds every object it needs itself** — there is no `Deps` struct and nothing is handed down from `main`. A module is a `cpbootstrap.Module` — a name, an `Enabled` flag and a DI sequence — and the sequence is handed a `cpbootstrap.Props` carrying registrars and nothing else:

- `props.RPC.Mount(build, interceptors...)` — the module hands over what *builds* the handler, plus the interceptors it wants. `cpbootstrap` builds it, so it can put its own interceptor outside every module's — see [The error net](error-net.md)
- `props.AdminRPC.Mount(build, interceptors...)` — the same, for an operator service: served on the loopback console listener, and on the admin listener behind `Options.AdminGate` — see [Operator tools](admin.md)
- `props.InternalRPC.Mount(build, interceptors...)` — the same again, for what other modules call: served only on the loopback internal listener
- `props.Internal.Dial()` — the HTTP client and base URL a generated `New<Service>Client` takes, to call another module — see [Calling another module](#calling-another-module)
- `props.Events` — the in-process event bus: `props.Events.Publish(message)`, and `cpbootstrap.Subscribe(props.Events, name, buffer, handler)` for the runner that delivers one type — see [Telling other modules what happened](#telling-other-modules-what-happened)
- `props.Runners.Add(runner)` — a `Runner` (`Name()` and `Run(ctx)`), run as a goroutine with the process-lifetime context
- `props.Closers.Add(name, close)` — a cleanup, run in reverse registration order
- `props.Logger`, `props.Metrics`
- `props.Server` — the bind address and the stream heartbeat, **the only config a module reads that is not its own**. It is the transport every module answers over, so it belongs to the layer that owns the server rather than to any context.

**A constructor builds and a `Load…` method reads.** Nothing that reads a file or parses a blob happens inside `New`: the DI sequence calls `New`, then the load, one line each — `embedded_geodesic_map.New` then `LoadGeography`/`LoadBorders`, `inmemory_tile_storage.New` then `LoadSnapshot`, `cpipblock.New` then `Load`, `antibot.New` then `LoadState` (which connects to its schema too), `inmemory_ledger_storage.New` then `Load`.

A module never sees the router, the signal handler or another module's objects. **There is no mutable app object and nothing to leave a half-built dependency on**: `internal/shared/cpbootstrap` builds every module in order, then serves.

**Shutdown goes in this order**: end every open stream, `http.Server.Shutdown` (the public server, then the console, admin and internal ones — a public call in flight may still be waiting on an internal one), the closers in reverse order, then cancel and wait on the runners. The first step is the drain interceptor — see [Ending the streams on shutdown](live-streams.md#ending-the-streams-on-shutdown). There are no storage closers left: the tile map, the ledger and the antibot's evidence all flush from their runners, after the closers, once the server has stopped taking writes.

**`cmd/api/main.go` is the composition root, and the server's only one** — there is no `internal/app`, because a package whose whole job is to be called once by `main` was a level of indirection and nothing else. It does three things: load the config, build the admin gate (`auth.NewAdminGate`, see [Operator tools](admin.md)), and list the modules. It builds no other object, derives nothing, and reads inside no block.

**The aggregation is the whole of it — a flat slice, no branches, no dependencies threaded through:**

```go
return []bootstrap.Module{
	auth.NewModule(config.Auth),
	planet.NewModule(config.Planet),
	chat.NewModule(config.Chat),
	player.NewModule(config.Player),
	seasons.NewModule(config.Seasons),
	support.NewModule(config.Support),
}
```

**Every module is always listed; a module with a switch reads its own.** `NewModule` sets `Enabled` and `cpbootstrap` skips the ones that are off, so turning auth off is a config change and never an edit here. `planet`, `chat`, `player`, `seasons` and `support` have no switch and are always on; `auth` has one. A disabled module is never built, so its routes are **absent** rather than present and refusing — `/auth.v1.AuthService/` 404s.

## What two contexts need, without either handing it to the other

`main` builds no objects at all, so a thing two contexts need is **a config block they both declare**, and each builds its own instance from it.

- **`shared/cpsession`** is the click token. The `auth:` block is read by two contexts, but they share no key and no type: `cpsession.SignerConfig` carries the Ed25519 seed and only `auth.Config` declares it, while `planet.Config` declares `cpsession.VerifierConfig`, which is two switches and **no key at all**. `auth` builds a `*Signer`, `planet` a `*Verifier`, **which has no `Mint` on it**. Under the old shared `Config` both built a `*Signer` off one HMAC secret, so `planet` held a live minting object and nothing but discipline stopped it using it.

  **`planet` asks `auth` for the key** over the internal listener, rather than reading one — see [Calling another module](#calling-another-module) and [Sessions](sessions.md). So there is one key in the whole file, `auth.secret`, nothing derived for a human to paste, and nothing to keep in step with it.

  Neither module imports the other's packages, and **the planet context knows nothing about Turnstile** — the siteverify client lives at `auth/internal/attestation/turnstile`, so it *cannot* reach it.
- **`shared/cpcountries`** is the ISO list. It is stateless and hardcoded, so each module just calls `cpcountries.New()`, the way it calls `cptime.SystemClock{}`. It sits in `shared` and not under `planet/internal/clicks/` for exactly the reason that layer exists: neither context may depend on the other — and now could not, since that directory is unreachable from chat.

**This is why `auth.secret` is now required** rather than invented at boot — see [Sessions](sessions.md).

## Calling another module

**Over Connect, on a loopback listener, never in the caller's stack trace.** A module that other modules call mounts an internal service with `props.InternalRPC.Mount`; `cpbootstrap` serves it on `httpServer.internalBindAddress` alone, which must be loopback. The caller dials once, in its `module.go`, builds the generated client over `props.Internal.Dial()`, and hands that one client to every adapter that needs it. An adapter takes a narrow port of the procedures it calls (`rpc_account_reader.Auth`, `rpc_player_authors.Player`), never the dialer, so no call builds a client.

| caller | asks | for | through |
|---|---|---|---|
| `planet`, `player`, `chat`, `seasons`, `support`, `missions` | `auth.v1.InternalService/GetVerifyingKey` | the public half of the click token key, once per boot | `shared/cpsessionverifier` |
| `player` | `auth.v1.InternalService/GetAccount` | whether an account is linked, on each `SetName`; when it was made, on each `GetPlayer` and each title award (`StatsChanged`); both, on each operator's `AdminService/GetPlayer` | `players/rpc_account_reader` |
| `player` | `auth.v1.InternalService/GetAccounts` | whether each account of a page is linked and when it was made, once per page of a title reconciliation | `players/rpc_account_reader` |
| `chat` | `player.v1.InternalService/GetAuthor` | the name a sender is shown under, on each `SendMessage` | `messages/rpc_player_authors` |
| `chat` | `player.v1.InternalService/GetAuthors` | who everyone in the window is, once per `GetHistory`; who reacted, once per `React`; who wrote each message of a page, once per `ListMessages` | `get_history_handler/history_query/rpc_player_authors`, `messages/rpc_player_authors`, `list_messages_handler/messages_query/rpc_player_authors` |
| `seasons` | `planet.v1.InternalService/GetTerritories` | the tiles each country holds and the size of the map, for each snapshot (once a minute) | `rounds/rpc_planet_territories` |
| `seasons` | `player.v1.InternalService/GetAuthors` | the name and color of each account of a page of standings, and whether it is a guest, on each `GetStandings` and `GetMySeason`, and each read of a live board | `get_standings_handler/standings_query/rpc_player_authors`, `get_my_season_handler/my_season_query/rpc_player_authors` |
| `support` | `player.v1.InternalService/GetAuthors` | the name of both accounts of each report of a page, on each `ListReports` | `list_reports_handler/reports_query/rpc_player_authors` |
| `support` | `player.v1.InternalService/GetAuthors` | the name of each account a page of reports names, on each `ListReports` | `list_reports_handler/reports_query/rpc_player_authors` |
| `auth`'s admin gate, which `main` builds | `auth.v1.InternalService/GetOperator` | who calls the admin listener, and whether that account operates, on each admin call | `admingate/admission/rpc_operators` |
| `player` | `planet.v1.InternalService/GetTakesByCountry` | the flags an account took tiles for and the flags that held them, from the ledger, on each `GetFronts`, on each `GetPlayer` the cache does not answer, and on each operator's `AdminService/GetPlayer` | `playerv1controller/rpc_planet_fronts` |
| `missions` | `planet.v1.InternalService/GetTerritories` and `GetGrounds` | what each flag holds and how many tiles each country's ground has, once per flag and day, to draw its missions | `missions/rpc_planet_world` |
| `missions` | `planet.v1.InternalService/GrantCharge` | the charge a mission pays, on each `CollectMission` | `missions/rpc_planet_charges` |

A module cannot import another's interior, so the key client all five need is `shared/cpsessionverifier` rather than a copy in each.

- **Each module's data stays in one place.** The caller holds an address, never the other module's config block, pool, objects or root package. The seed is read by `auth` and nothing else.
- **`Dial` is the one place that knows the transport is loopback HTTP.** Moving to unix sockets changes the listener and `internalDialer`, and no module.
- **`Dial` fails with no internal listener**, and a module that calls another refuses the boot. `cpsessionverifier` is the exception: it dials when it first fetches the key, once per process. An internal service with no listener is not served, and logged, like an admin one.
- **The error net and the drain wrap internal services too.**
- **What travels is a public key**, which is the point: a holder can check a token and cannot mint one, so the hop carries nothing worth stealing. Under the HMAC this replaced there was no such thing to send.
- `TestAModuleCallsAnotherOverTheInternalListener` pins the path, `TestAnInternalServiceIsNotOnThePublicRouter` pins that Caddy cannot reach it.

The proto sits beside the public one in the module's package (`proto/auth/v1/internal.proto`), as `admin.proto` does, and the frontend generates it too without using it.

**The loopback rule is for synchronous calls, and it exists to stop one module loading another's config.** A caller that needs an answer has to reach the module that holds the data. Done in process, that is how `planet` once built a signer off `auth`'s secret and how a key ends up duplicated across blocks in the file. The listener makes the caller hold an address and nothing else.

## Telling other modules what happened

**Over Go channels, in process.** There is no broker, and a modular monolith still needs messaging. An emitter publishes an event and never knows who listens, so it has no reason to load a listener's config, and the problem the loopback rule solves does not arise. A loopback stream would buy nothing here: it adds encoding, reconnects and an address the subscriber must hold, and it loses events on a crash exactly as a channel does.

- **The payload is a proto-generated message**, declared in the emitter's proto package (`proto/<name>/v1/events.proto`). A subscriber imports `generated/proto`, never the emitter's root package, which would also hand it the emitter's `Config`. The proto is also the schema, if AsyncAPI or a real broker ever comes.
- **Each subscriber gets its own copy.** A proto message is a pointer: the emitter builds a fresh one per event and never touches it again, and nothing shared is mutated.
- **Each subscriber reads its own buffered channel, in its own goroutine.** No subscriber code runs in the emitter's stack trace, so a click never waits on a listener.
- **A full buffer drops the event and counts it.** Delivery is at most once and not durable: a crash or a restart loses what was in flight. A flow that cannot lose an event (money, rewards) needs a durable path of its own, not this.
- **Subscribers register while modules are built**, before any runner starts, so no event is published to nobody at boot.
- **A relationship on the context map is one integration, not one pair of modules.** `chat` asks `player` who posts, so `player` is upstream of `chat` for authors; `chat` publishes `MessageSent`, so `chat` is upstream of `player` for messages. Two modules linked both ways is fine. What is not: a shared model, or a synchronous call cycle — `player` never calls `chat`. The context where a thing happens publishes it in its own language, and the listener translates it at its edge (`message_sent_subscriber`), as it does `TileTaken`. A downstream module never calls its upstream to report a fact instead.

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
| `planet.v1.TileTaken{account_id, tile_id, country, taken_at, previous, ground}` | `planet`, `ledger/publishing_ledger_storage` | each take the ledger records with an account: a click, a spread or an enclose, one event per tile; `previous` is the flag that held it, `ground` the country it is on (`clicks.Borders`) | `player`, for the stats; `seasons`, for the standings; `missions`, for the day's progress |
| `planet.v1.BombLanded{country, tile_id, ground, cleared, landed_at, account_id}` | `planet`, `drop_bomb_usecase/publishing_drop_bomb` | each bomb that went off, on land or in the sea; a refused drop publishes nothing | `chat`, which announces it; `player`, for the stats |
| `planet.v1.Fortified{country, landmass_id, tiles, ground, fortified_at, account_id}` | `planet`, `relay_fortifications_usecase` | each landmass a player's take fortified (see [Fortify](fortify.md)); `account_id` is the account that took the last tile, empty for a click with no account | `chat`, which announces the ones of 50 tiles or more; `player`, for the stats |
| `planet.v1.BonusClaimed{account_id, claimed_at}` | `planet`, `claim_bonus_usecase` | after a caught box's charge is held; a refused claim, or a caller with no account, publishes nothing | `player`, for the stats |
| `planet.v1.QuizAnswered{account_id, correct, answered_at}` | `planet`, `answer_quiz_usecase` | each quiz answered, right or wrong, by a caller with an account; a right answer after its charge is held | `player`, which counts the right ones |
| `auth.v1.AccountDeleted{account_id}` | `auth`, `delete_account_usecase` and `prune_guests_usecase` | after the account row is deleted; a pruned guest is a deleted account | `player`, which forgets the profile, the stats, the titles, the title worn and the visit; `seasons`, which forgets the account's standings; `planet`, which takes the account off every take postgres keeps; `chat`, which forgets the seen mark; `support`, which takes the account off the notes it sent and deletes the reports that name it; `missions`, which forgets the account's sheets and collections |
| `auth.v1.SignedIn{previous_account_id, account_id}` | `auth`, `signin.Admitter`, for `complete_sign_in_usecase` and `complete_email_sign_in_usecase` | after a sign-in or a link is saved; `previous_account_id` is empty for a browser that had no live session | `player`, which moves the browser's visit to the account, and gives the account a username when it has none |
| `auth.v1.SignedOut{account_id}` | `auth`, `sign_out_usecase` and `sign_out_everywhere_usecase` | after the session, or every session, is deleted; a cookie with no session publishes nothing | `player`, which takes the account off the roster |
| `chat.v1.MessageSent{message_id, account_id, sent_at}` | `chat`, `send_message_usecase/publishing_send_message` | after each message is kept; a refused or failed post publishes nothing | `player`, for the stats |
| `chat.v1.AccountMuted{account_id, muted_at, duration}` | `chat`, `mute_usecase/publishing_mute` | after each mute is kept; a refused or failed mute publishes nothing. Never the network it holds | `chat`, which announces it |
| `player.v1.StatsChanged{account_id}` | `player`, `record_stats_usecase` | after each take, message, bomb, fortify, box or right answer is counted on the account's stats; a failed write publishes nothing | `player`, which grants the titles the stats now earn |
| `seasons.v1.RoundClosed{season, number, finale, ended_at, results}` | `seasons`, `take_snapshot_usecase` | after each round's results are kept; a failed close publishes nothing | `chat`, which announces the podium |
| `seasons.v1.SeasonClosed{season, ended_at, table}` | `seasons`, `take_snapshot_usecase` | after the finale's results are kept, right after its `RoundClosed`: the season's whole table, ranked by `rounds.TableOf`, the champion first; a failed close publishes nothing | `chat`, which announces the champion |

Adding a context that callers talk to means a `proto/<name>/v1`, an `internal/<name>/` with a `module.go`, and one line in the slice. Connect derives the route from the proto package, so there is no prefix to allocate and no router to edit.

**`cmd/api`** is the server, and the only binary the image holds: `main.go` is the composition root, about 85 lines of config and a module list, and `cpbootstrap` is the rest. **`cmd/clips`** is the other binary, a job a workflow runs with `go run` — see [The clips job](clips-job.md).

It runs as **one container beside a postgres**: the tile map lives in process, and postgres is where it is kept between boots — see [Durability](durability.md). There is no cache and no second API process.
