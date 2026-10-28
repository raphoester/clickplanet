# The composite layer

Each context wires **itself**, in a `module.go` at its root (`internal/planet/module.go`, `internal/chat/module.go`, `internal/auth/module.go`, `internal/player/module.go`). That file is the context's manifest: its `Config`, whether it is on, and its DI sequence. **A module takes its config and nothing else, and builds every object it needs itself** — there is no `Deps` struct and nothing is handed down from `main`. A module is a `cpbootstrap.Module` — a name, an `Enabled` flag and a DI sequence — and the sequence is handed a `cpbootstrap.Props` carrying registrars and nothing else:

- `props.RPC.Mount(build, interceptors...)` — the module hands over what *builds* the handler, plus the interceptors it wants. `cpbootstrap` builds it, so it can put its own interceptor outside every module's — see [The error net](error-net.md)
- `props.AdminRPC.Mount(build, interceptors...)` — the same, for an operator service: served on the loopback console listener, and on the admin listener behind `Options.AdminGate` — see [Operator tools](admin.md)
- `props.InternalRPC.Mount(build, interceptors...)` — the same again, for what other modules call: served only on the loopback internal listener
- `props.Internal.Dial()` — the HTTP client and base URL a generated `New<Service>Client` takes, to call another module — see [Calling another module](#calling-another-module)
- `props.Runners.Add(runner)` — a `Runner` (`Name()` and `Run(ctx)`), run as a goroutine with the process-lifetime context
- `props.Closers.Add(name, close)` — a cleanup, run in reverse registration order
- `props.Logger`, `props.Metrics`
- `props.Server` — the bind address and the stream heartbeat, **the only config a module reads that is not its own**. It is the transport every module answers over, so it belongs to the layer that owns the server rather than to any context.

**A constructor builds and a `Load…` method reads.** Nothing that reads a file or parses a blob happens inside `New`: the DI sequence calls `New`, then the load, one line each — `embedded_geodesic_map.New` then `LoadGeography`/`LoadBorders`, `inmemory_tile_storage.New` then `LoadSnapshot`, `cpipblock.New` then `Load`, `guard.New` then `Load` (the antibot's evidence), `inmemory_ledger_storage.New` then `Load`.

A module never sees the router, the signal handler or another module's objects. **There is no mutable app object and nothing to leave a half-built dependency on**: `internal/shared/cpbootstrap` builds every module in order, then serves.

**Shutdown goes in this order**: end every open stream, `http.Server.Shutdown` (the public server, then the console, admin and internal ones — a public call in flight may still be waiting on an internal one), the closers in reverse order, then cancel and wait on the runners. A module closes its own Redis client and pool once its runners returned. The first step is the drain interceptor — see [Ending the streams on shutdown](live-streams.md#ending-the-streams-on-shutdown). There are no storage closers left: the tile map, the ledger and the antibot's evidence all flush from their runners, after the closers, once the server has stopped taking writes. The antibot's flushes once its subscribers handled their last facts (`Guard.Hearing`).

**`cmd/api/main.go` is the composition root, and the server's only one** — there is no `internal/app`, because a package whose whole job is to be called once by `main` was a level of indirection and nothing else. It does three things: load the config, build the admin gate (`auth.NewAdminGate`, see [Operator tools](admin.md)), and list the modules. It builds no other object, derives nothing, and reads inside no block.

**The aggregation is the whole of it — a flat slice, no branches, no dependencies threaded through:**

```go
return []bootstrap.Module{
	auth.NewModule(config.Auth),
	planet.NewModule(config.Planet),
	antibot.NewModule(config.AntiBot),
	chat.NewModule(config.Chat),
	player.NewModule(config.Player),
	scoring.NewModule(config.Scoring),
	gamemaster.NewModule(config.Gamemaster),
	missions.NewModule(config.Missions),
	support.NewModule(config.Support),
}
```

**Every module is always listed; a module with a switch reads its own.** `NewModule` sets `Enabled` and `cpbootstrap` skips the ones that are off, so turning auth off is a config change and never an edit here. `planet`, `chat`, `player`, `scoring` and `support` have no switch and are always on; `auth` and `antibot` have one. A disabled module is never built, so its routes are **absent** rather than present and refusing — `/auth.v1.AuthService/` 404s.

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
| `planet`, `player`, `chat`, `scoring`, `support`, `missions` | `auth.v1.InternalService/GetVerifyingKey` | the public half of the click token key, once per boot | `shared/cpsessionverifier` |
| `player` | `auth.v1.InternalService/GetAccount` | whether an account is linked, on each `SetName`; when it was made, on each `GetPlayer` and each title award (`StatsChanged`); both, on each operator's `AdminService/GetPlayer` | `players/rpc_account_reader` |
| `player` | `auth.v1.InternalService/GetAccounts` | whether each account of a page is linked and when it was made, once per page of a title reconciliation | `players/rpc_account_reader` |
| `chat` | `player.v1.InternalService/GetAuthor` | the name a sender is shown under, on each `SendMessage` | `messages/rpc_player_authors` |
| `chat` | `player.v1.InternalService/GetAuthors` | who everyone in the window is, once per `GetHistory`; who reacted, once per `React`; who wrote each message of a page, once per `ListMessages` | `get_history_handler/history_query/rpc_player_authors`, `messages/rpc_player_authors`, `list_messages_handler/messages_query/rpc_player_authors` |
| `scoring` | `planet.v1.InternalService/GetTerritories` | the tiles each country holds and the size of the map, once when a round is closed to score it, and for each read of an open round's live standings (every 10s while a stream follows it, and each `GetRound`) | `rounds/rpc_planet_territories`, for both |
| `scoring` | `player.v1.InternalService/GetAuthors` | the name and color of each account of a page of standings, and whether it is a guest, on each `GetBoard` and `GetMyTally`, and each read of a live board; a round's best players, on each read of a round | `get_standings_handler/standings_query/rpc_player_authors`, `get_my_season_handler/my_season_query/rpc_player_authors`, `listen_for_events_handler/race_query/rpc_player_authors` |
| `support` | `player.v1.InternalService/GetAuthors` | the name of both accounts of each report of a page, on each `ListReports` | `list_reports_handler/reports_query/rpc_player_authors` |
| `support` | `player.v1.InternalService/GetAuthors` | the name of each account a page of reports names, on each `ListReports` | `list_reports_handler/reports_query/rpc_player_authors` |
| `auth`'s admin gate, which `main` builds | `auth.v1.InternalService/GetOperator` | who calls the admin listener, and whether that account operates, on each admin call | `admingate/admission/rpc_operators` |
| `player` | `planet.v1.InternalService/GetTakesByCountry` | the flags an account took tiles for and the flags that held them, from the ledger, on each `GetFronts`, on each `GetPlayer` the cache does not answer, and on each operator's `AdminService/GetPlayer` | `playerv1controller/rpc_planet_fronts` |
| `missions` | `planet.v1.InternalService/GetTerritories` and `GetCountrySizes` | what each flag holds and where, and how many tiles of land each country has, once per account and day, to draw its missions | `missions/rpc_planet_world` |
| `missions` | `planet.v1.InternalService/GetAllegianceOf` | the flag an account pledged, once per account and day, to draw its missions | `missions/rpc_planet_allegiance` |
| `missions` | `auth.v1.InternalService/GetAccount` | whether an account is signed in, when a take finishes one of its missions | `missions/rpc_auth_accounts` |
| `gamemaster` | `planet.v1.InternalService/GetRuleSetNames` and `antibot.v1.InternalService/GetRuleSetNames` | the rule sets each module keeps, on each schedule entry an operator adds or edits | `schedule/rpc_planet_rule_sets`, `schedule/rpc_antibot_rule_sets` |
| `chat` | `scoring.v1.InternalService/GetRound` and `GetTable` | a round's results or a book's table, by the id a gamemaster event names, to announce a podium or a champion | chat's rpc scoring adapter |

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

**Over Redis Streams.** An emitter publishes an event and never knows who listens, so it has no reason to load a listener's config, and the problem the loopback rule solves does not arise. Redis keeps what was published until every subscriber has handled it: a restart, a crash or a slow subscriber loses nothing that reached it.

- **The payload is a proto-generated message**, declared in the emitter's proto package (`proto/<name>/v1/events.proto`), and travels as its protobuf bytes. A subscriber imports `generated/proto`, never the emitter's root package, which would also hand it the emitter's `Config`.
- **One stream per type of event** (`events:planet.v1.TileTaken`), not per emitter: a subscriber hears one type, so it reads only that one, and each stream is trimmed at the pace of its own subscribers. In one stream per emitter, the 40k takes a day would push planet's rare bombs out of the trim window.
- **One consumer group per subscriber**, so each gets every event, in its own copy, in order, in its own runner. No subscriber code runs in the emitter's stack trace: a click waits on one `XADD`, never on a listener.
- **Delivery is at least once and in order per subscriber.** A handler's error is tried again in place after `redis.retryAfter` (2s), doubling up to 64 times that, and the events behind it wait, so the order holds. After `redis.maxDeliveries` (8, about four minutes) the event is buried in `events:dead` with its subscriber and the next one is read. A crash leaves what was read pending; the next boot takes it back first (`XAUTOCLAIM` from `0-0`, whatever consumer held it), before anything new.
- **So a handler sometimes runs twice, and must be safe to.** The bus does not dedupe: no event id, no inbox table. **The domain models the uncertainty**: a fact that may come twice carries the identity of what happened, and the receiver's own state remembers what it applied. See [Idempotent receivers](#idempotent-receivers).
- **A publish Redis does not take is lost, and counted** (`events_unpublished_total{event}`). A fact whose loss would leave two modules disagreeing for good goes through an outbox instead — see [What may not be lost](#what-may-not-be-lost-the-outbox).
- **A relationship on the context map is one integration, not one pair of modules.** `chat` asks `player` who posts, so `player` is upstream of `chat` for authors; `chat` publishes `MessageSent`, so `chat` is upstream of `player` for messages. Two modules linked both ways is fine. What is not: a shared model, or a synchronous call cycle — `player` never calls `chat`. The context where a thing happens publishes it in its own language, and the listener translates it at its edge (`message_sent_subscriber`), as it does `TileTaken`. A downstream module never calls its upstream to report a fact instead.
- **A downstream module that must make its upstream act sends a command in the upstream's language.** The antibot is downstream of planet: it hears planet's facts about callers and decides whom to ban, but the refusal is planet's. Planet defines `RefuseCaller` and `AdmitCaller` in its own package (`proto/planet/v1/commands.proto`), on command streams it handles, and the antibot's adapter (`bans/outbox_planet_enforcer`) translates its bans into them. Planet never names the antibot. See [Asking another module to change something: commands](#asking-another-module-to-change-something-commands).

**Each module builds its own connection, as it builds its own pool.** `cpbootstrap` knows nothing of Redis. The reusable parts live in `shared/cpredis`, and a module's `Config` declares a `redis` block (`cpredis.Config`). Every module's block names the same Redis — `TestEveryModuleOfProductionTalksToTheSameRedis` pins it for the production file. Right after its migrations, a module calls `cpredis.New(config.Redis, props.Metrics)` then `ConnectCtx`, which pings: a module that cannot reach Redis refuses the boot.

- **`Publish(events...)`** is one pipelined `XADD` per call, bounded by `redis.publishTimeout` (500ms), and never fails to its caller: what Redis did not take is counted. **`Send(ctx, events...)`** is the same with an error, for a relay that keeps the events until they are sent.
- **`cpredis.Subscribe[T](ctx, streams, name, handler)`** creates group `name` on the stream of type `T` and answers the `Runner` that reads it and calls `handler.Handle(ctx, event)`. The module adds that runner. A second subscriber of one type with the same name on one connection refuses the boot.
- **A group is made from the start of the stream**, and an existing one is kept: what another module published before this one subscribed is read too, so the order of modules in `cmd/api` does not matter. A publish always adds to the stream, since no module knows every subscriber. **So a group keeps its name when its module is renamed**: a new name is a new group, which reads the stream again from its start. The scoring module's groups on `TileTaken` and `AccountDeleted` are still `seasons-standings`, `seasons-standings-accounts`, `seasons-rounds` and `seasons-rounds-accounts`.
- **A group nobody subscribes with any more stays**, and holds its stream's trim back to `redis.maxLen`. A renamed or removed subscriber needs `XGROUP DESTROY events:<type> <old name>` by hand once the release is out.
- **Memory is bounded twice.** Every minute each subscription cuts its stream (`XTRIM MINID ~`) to the oldest entry one of the stream's groups has not handled, so a stream holds only what is still in hand while every subscriber keeps up. And each `XADD` caps its stream at `redis.maxLen` (100k, ~15 MB of takes), for a stream whose subscriber stopped. Redis runs with 64 MB and `noeviction`: a full Redis refuses events, counted as unpublished, rather than drop unread ones.
- **Metrics**: `events_failed_total{event, subscriber}` what a handler refused, `events_dead_total{event, subscriber}` what was buried, `events_lag{event, subscriber}` what is in the stream unread, `events_pending{event, subscriber}` what was read and not acked (read from `XINFO GROUPS` at each scrape), and `events_unpublished_total{event}`, one counter every module's connection shares. The bus logs nothing: a subscriber that wants a log line wraps its handler in a decorator (`player`'s `log_subscriber`).
- **At shutdown** a runner finishes and acks what it read, reads what is left without waiting (ten batches at most), and stops; what it does not reach waits in Redis for the next boot. A handler runs on a context the shutdown does not cancel. A store a handler writes to must outlive it: `cpredis.CloseAfter(streams, logger, runners...)` closes the client once the module's subscribers returned, inside `cppg.CloseAfter`, which closes the pool after that.
- **Tests:** `redis_test.go` runs against a real Redis (`cpredis.StartTestServer`, behind the tag, a `redis:7.4-alpine` container per test), each module on its own connection: every module reads every event in order though it subscribed after the publish, a copy each, no subscriber code in the publisher's stack trace, a duplicate name refused, no Redis refuses the connection, a failure tried again before the events behind it, a poison event buried where `XRANGE events:dead` shows it, what a crash left pending handled first at the next boot, and the shutdown. `subscription_internal_test.go` pins the trim. `cpredis.RecordedEvents` (behind the tag) stands in for the bus where a use case only publishes.

### Idempotent receivers

| subscriber | why a second delivery changes nothing |
|---|---|
| `player-stats`, `player-stats-bombs`, `player-stats-fortifies` | `planet` tells its map facts in ledger order with their `position`, and each take with its `index` in its event. The stats row keeps a mark past the last take, bomb and fortify it counted (`players.Counted`); one below it was counted already. A take's mark is its position and index (`players.TakeMark`): a spread or an enclose is one ledger event, at one position, that holds several takes |
| `player-stats-messages`, `player-stats-bonuses`, `player-stats-quizzes` | the same with the time of the last message, box and right answer counted, compared to the microsecond postgres keeps |
| `seasons-standings` | the tally keeps the position and index past the last take it counted (`standings.Tally.TakesBelow`: two columns of each contributions row, the highest pair is the tally's) |
| `seasons-rounds` | each round of its own keeps the position and index past the last take it counted (`rounds.counted_takes_below`, `counted_takes_below_index`), moved in the statement that counts the take |
| `missions-progress` | an account's day keeps the position and index past the last take it counted (`missions.Sheet`, `mission_sheets.counted_takes_below`, `counted_takes_below_index`) |
| `missions-sign-ins` | pays only the missions not paid yet (`mission_payments`) |
| `planet`'s `planet.v1.GrantCharges` handler | keeps each `grant_id` it applied, in the transaction of the grant (`charge_grants`) |
| `chat-announcements-*` | an announcement is named after the fact it tells (the bomb's or the fortify's ledger position, the mute's id, the round's season and number, the champion's season) and kept with `AppendOnce`: a second one is neither kept nor shown |
| `player-titles` | grants only what the account does not hold |
| `player-names-sign-ins` | names only an account that has no name |
| each `AccountDeleted` subscriber | deletes or blanks what is left, which is nothing the second time |
| `player-presence-*` | moves or forgets a visit in memory; a second time finds nothing to move |
| `antibot-clicks`, `antibot-map-reads`, `antibot-streams`, `antibot-boxes-*` | planet tells each kind of fact at a later time than the one before (`facts.Log`), from one sender. The listener keeps the latest time it heard of each kind (`listener.Marks`), saved with the evidence as of the same moment, and skips a fact at or before it |
| `commands:planet.v1.RefuseCaller`, `commands:planet.v1.AdmitCaller` (planet's `handler`) | each command names when it was decided; planet keeps the latest decision on each caller (`refusals.Refusal.Decided`), so a command sent again, or an older one from the other stream, changes nothing |
| `commands:scoring.v1.ScoringCommand` (scoring's `handler`) | each command names its book or round: a book or round opened twice is one, a round closed twice is scored once, a book closed twice is ranked once, and a close of a round never opened scores nothing |
| `commands:planet.v1.PlayRuleSet`, `commands:antibot.v1.PlayRuleSet` (each module's `handler`) | each command names when it was decided; the module keeps the latest choice (`rules.Choice.Overrules`, `rule_set_choice`), so a command sent again, or an older one, changes nothing |
| `commands:planet.v1.ResetMap` (`handler`) | planet keeps each reset id in `map_resets`, in the wipe's transaction, so a reset told again wipes nothing |
| `gamemaster-rounds`, `gamemaster-books` | the gamemaster keeps each round scored and each book closed it heard (`scored_rounds`, `closed_books`, inserted once), and records its own event only for a new one |

The marks need each source in order per account, which is why the map facts come from the ledger relay and not from each click: two clicks of one account could reach Redis in the other order. **What stays uncertain, on purpose**: two messages, boxes or answers of one account in the same microsecond, or a clock that jumps back, count once.

### What may not be lost: the outbox

A fact goes through an outbox when its loss would leave the modules disagreeing for good, with nothing to tell it again:

- `auth.v1.AccountDeleted`: a lost one keeps a deleted player's personal data in every other module. `delete_account_usecase` and `prune_guests_usecase` record it in the transaction that deletes the account.
- `auth.v1.SignedIn`: a lost one leaves a linked account with no username, shown as a guest, until its next sign-in. `signin.Admitter` records it in the transaction that saves the sign-in.
- `planet.v1.RefuseCaller` and `AdmitCaller`, from the antibot: a lost one leaves a ban the antibot keeps and planet never refuses, or a lifted one planet keeps refusing. `bans.Banner` records the command in the transaction that changes the ban row, under the ban's lock. At each rule set it takes, the first at boot, the antibot records one again for every running ban (`bans.Book.Restate`), so the bans running at a deploy reach planet's new table too, and a set that stops enforcing admits them.
- `scoring.v1.RoundScored` and `BookClosed`: the round is scored once and the book closed once, so a lost one is never sent again and the gamemaster never tells that day, battle or season ended. Scoring records each in the transaction that scores the round or closes the book, only when this call did it.
- `gamemaster.v1.DayEnded`, `BattleEnded` and `SeasonEnded`: the gamemaster records each in the transaction that keeps what it heard from scoring, only when it is new, so the chat announces each podium and champion once.
- `planet.v1.GrantCharges`, a command: a lost one is a mission paid that planet never granted. `missions`' `payroll` records it with the payment.
- **Not the gamemaster's commands** (`PlayRuleSet`, `ResetMap`, `ScoringCommand`): the clock decides them, not a transaction. The
  gamemaster's runners send them with `streams.Commands().Send` and keep what they told only once Redis took it, so a
  refused send is sent again the next second, and each boot sends again what holds (see [The gamemaster](gamemaster.md)).
- `planet.v1.TileTaken`, `BombLanded` and `Fortified`: **the ledger is their outbox.** `ledger/usecases/relay_ledger_usecase` reads what the ledger flushed past `planet.ledger_relayed`, sends it, then moves the row. A take a hard kill loses before its flush is never told, so the stats and the standings never count what the ledger does not hold; a crash between the send and the move tells the batch again, which the marks make harmless. The migration put the row after the last take the old server had published.

Not in an outbox, because a loss is harmless or the next event repairs it: `SignedOut` (the roster drops the visit when it expires), `MessageSent`, `BonusClaimed` and `QuizAnswered` (a stat short by one), `AccountMuted` (an announcement missed), `StatsChanged` (the next one grants the title), and planet's facts about callers (`ClickTried` and the rest: a try or a read short, in a watchdog's window). Those go through `facts.Relay`, which sends what `facts.Log` holds every 100ms with `Publish`, in order: a crash loses the last 100ms of them.

**`shared/cpoutbox`** is the mechanism. **Recording an event is the use case's decision**: it opens a transaction and calls `Outbox.Record(ctx, events...)` beside its stores — see [Transactions](transactions.md). `cpoutbox.New(db).Record` inserts each event in the module's `outbox` table (`position bigserial`, `event outbox_event`, `payload bytea`) in the transaction `ctx` carries, and refuses outside one. `outbox_event` is an enum of the events that module keeps there, so a new one is a migration. `cpoutbox.NewRelay(name, db, streams, logger)` is a runner that, every second and once more at shutdown, locks the oldest rows (`FOR UPDATE SKIP LOCKED`, 100 at a time), `Send`s them in order and deletes them in the same transaction: Redis down keeps the rows for the next tick. A row that does not decode is dropped and logged rather than left to hold every row behind it. `cpoutbox/outbox_test.go` pins it; each use case's test pins what it records, with `cpatomic.FakeOutbox`.

### Asking another module to change something: commands

**A command, over Redis Streams, through the sender's outbox.** A module that needs another to change its own state
does not call it: it records a command in the receiver's language (`proto/<receiver>/v1/commands.proto`,
`planet.v1.GrantCharges`) in its outbox, in the transaction of what decided it, and the relay sends it. The receiver
never learns who sent it, as an emitter never learns who listens. The rule schedule's commands, which the clock
decides, are sent straight (see [The rule schedule](gamemaster.md)).

- **One stream per type of command** (`commands:planet.v1.GrantCharges`), apart from the events, with **one group**
  (`handler`): the module that owns the command reads each one once, in order, and one sent before it listened too.
  `cpredis.Handle[T](ctx, streams, handler)` is that runner; a second `Handle` of one command on one connection refuses
  the boot. `streams.Commands()` is what an outbox relay sends with: `cpoutbox.NewRelay(name, db, streams.Commands(),
  logger)`.
- **Delivery is the events' own**: at least once, a failure tried again, a poison command buried in `events:dead`. So
  **a command names itself** (`grant_id`), and the receiver keeps what it applied in the transaction of the change.
- **No answer comes back.** The sender decides with what it knows; what the receiver refuses or caps is its own rule: a
  grant past the stack is not granted.
- **A read stays an RPC** (see [Calling another module](#calling-another-module)): a command is for a change.
- `redis_test.go` pins a command sent before its handler listened, a command that is no event, and one handler.

The commands today:

| command | sent by | when | handled by |
|---|---|---|---|
| `planet.v1.GrantCharges{grant_id, account_id, kind, amount}` | `missions`, through its outbox | a mission finished: its reward | `planet` (`apply_grant_handler`), which grants what fits the account's stack, once per `grant_id` |
| `planet.v1.RefuseCaller{scope \| account_id, until, decided_at}`, `AdmitCaller{scope \| account_id, decided_at}` | `antibot`, through its outbox, from `bans.Banner` (a flag, a ban, an unban) and `bans.Book.Restate` (each rule set taken, the first at boot) | a ban changed: refuse the caller until a time, or admit it from now | `planet` (`refuse_caller_handler`, `admit_caller_handler`), which keeps the latest decision on each caller by `decided_at` in `planet.refused_callers` and refuses clicks, bombs and shields from it |
| `planet.v1.PlayRuleSet{rule_set, decided_at}`, `antibot.v1.PlayRuleSet{rule_set, decided_at}` | `gamemaster`, straight to the command streams, from `play_schedule_usecase` | the schedule's turn changed, and once at each boot | `planet` and `antibot` (`play_rule_set_handler`), which each play the set when it is newer than the choice kept, and refuse a set they keep none of |
| `planet.v1.ResetMap{reset_id, decided_at}` | `gamemaster`, straight to planet's command stream, from `play_resets_usecase` | a season after season 0 runs, once per process | `planet` (`reset_map_handler`), which wipes the map once per reset id it keeps in `map_resets` (see [The map reset](map-reset.md)) |
| `scoring.v1.ScoringCommand{open_book \| open_round \| close_round \| close_book}` | `gamemaster`, straight to **one** command stream, from `play_scoring_usecase` | a season's start and end (a book), each day's and each battle's start and end (a round), in that order at one instant; each boot sends again what holds | `scoring`, which opens, scores and ranks by the ids it is given. **One stream for four kinds, on purpose**: a round must be scored before its book is ranked, and two streams would not keep that order |

**In process still, on purpose**: the live fanouts to open browser streams (the tile map, the chat feed, the roster and the titles, the season boards and the race) are not module-to-module; they are lossy by design, and a client reads again when its stream reopens.

The events today:

| event | published by | when | heard by |
|---|---|---|---|
| `planet.v1.TileTaken{account_id, tile_id, country, taken_at, position, previous, ground, index}` | `planet`, `ledger/usecases/relay_ledger_usecase` | each take the ledger flushed with an account: a click, a spread or an enclose, one event per tile, in ledger order; `previous` is the flag that held it, `ground` the country it is on (`clicks.Borders`); `index` is the take's place in its ledger event, from 0, since a spread's or an enclose's takes share one `position` | `player`, for the stats; `scoring`, for the books' boards and each round's takes and players; `missions`, for the day's progress |
| `planet.v1.BombLanded{country, tile_id, ground, cleared, landed_at, account_id, position}` | `planet`, `ledger/usecases/relay_ledger_usecase` | each bomb the ledger flushed, on land or in the sea; a refused drop is not in the ledger | `chat`, which announces it; `player`, for the stats |
| `planet.v1.Fortified{country, landmass_id, tiles, ground, fortified_at, account_id, position}` | `planet`, `ledger/usecases/relay_ledger_usecase` | each landmass a take fortified (see [Fortify](fortify.md)); `account_id` is the account that took the last tile, empty for a click with no account | `chat`, which announces the ones of 50 tiles or more; `player`, for the stats |
| `planet.v1.BonusClaimed{account_id, claimed_at}` | `planet`, `claim_bonus_usecase` | after a caught box's charge is held; a refused claim, or a caller with no account, publishes nothing | `player`, for the stats |
| `planet.v1.QuizAnswered{account_id, correct, answered_at}` | `planet`, `answer_quiz_usecase` | each quiz answered, right or wrong, by a caller with an account; a right answer after its charge is held | `player`, which counts the right ones |
| `planet.v1.ClickTried{scope, account_id, linked, account_created_at, tile_id, country, tried_at, outcome, owner, position}` | `planet`, `try_click`, through `facts.Relay` | every click tried, refused and throttled ones included, with what came of it and the flag the tile wore before | `antibot`, which times every try and judges the ones the throttle let through |
| `planet.v1.MapRead{scope, maps, off_map, read_at}` | `planet`, `get_map_usecase`, through `facts.Relay` | each map batch read, as a share of the map, and whether a bound lies off it | `antibot`, for the `scraper` |
| `planet.v1.StreamOpened{scope, opened_at}` | `planet`, `listen_for_events_usecase`, through `facts.Relay` | each live stream opened, even one that fails | `antibot`, for the `scraper` |
| `planet.v1.BoxCaught{scope, after, caught_at}`, `BoxLapsed{scope, lapsed_at}`, `ForeignBoxClaimed{scope, claimed_at}` | `planet`, the box registry's report, through `bonuses/box_facts` and `facts.Relay` | a box claimed by the scope it was offered to, a box nobody claimed, and a claim of a box offered to another scope | `antibot`, for the `catcher` |
| `auth.v1.AccountDeleted{account_id}` | `auth`, through its outbox, from `delete_account_usecase` and `prune_guests_usecase` | with the account row's deletion; a pruned guest is a deleted account | `player`, which forgets the profile, the stats, the titles, the title worn and the visit; `scoring`, which forgets the account's tallies and its takes in the rounds; `planet`, which takes the account off every take postgres keeps; `chat`, which forgets the seen mark; `support`, which takes the account off the notes it sent and deletes the reports that name it; `missions`, which forgets the account's sheets and collections |
| `auth.v1.SignedIn{previous_account_id, account_id}` | `auth`, through its outbox, from `signin.Admitter`, for `complete_sign_in_usecase` and `complete_email_sign_in_usecase` | with a sign-in or a link; `previous_account_id` is empty for a browser that had no live session | `player`, which moves the browser's visit to the account, and gives the account a username when it has none; `missions`, which pays the day's missions the account finished as a guest |
| `auth.v1.SignedOut{account_id}` | `auth`, `sign_out_usecase` and `sign_out_everywhere_usecase` | after the session, or every session, is deleted; a cookie with no session publishes nothing | `player`, which takes the account off the roster |
| `chat.v1.MessageSent{message_id, account_id, sent_at}` | `chat`, `send_message_usecase/publishing_send_message` | after each message is kept; a refused or failed post publishes nothing | `player`, for the stats |
| `chat.v1.AccountMuted{account_id, muted_at, duration, mute_id}` | `chat`, `mute_usecase/publishing_mute` | after each mute is kept; a refused or failed mute publishes nothing. Never the network it holds | `chat`, which announces it |
| `player.v1.StatsChanged{account_id}` | `player`, `record_stats_usecase` | after each take, message, bomb, fortify, box or right answer is counted on the account's stats; a failed write publishes nothing | `player`, which grants the titles the stats now earn |
| `scoring.v1.RoundScored{round_id, book_id, ended_at, results}` | `scoring`, through its outbox | a round closed: scored from the map then, its points times its times | `gamemaster`, which tells which day or battle it was |
| `scoring.v1.BookClosed{book_id, ended_at, table}` | `scoring`, through its outbox | a book closed: its table, the champion first | `gamemaster`, which tells which season it was |
| `gamemaster.v1.DayEnded{season, day, round_id, ended_at}`, `BattleEnded{season, number, entry_id, times, ends_season, round_id, ended_at}`, `SeasonEnded{season, book_id, ended_at, finale_round_id}` | `gamemaster`, through its outbox, once scoring scored it | a day, a battle or a season ended, in the game's words, with the ids to read its numbers by | `chat`, which reads the numbers from scoring and announces the podium or the champion |

Adding a context that callers talk to means a `proto/<name>/v1`, an `internal/<name>/` with a `module.go`, and one line in the slice. Connect derives the route from the proto package, so there is no prefix to allocate and no router to edit.

**`cmd/api`** is the server, and the only binary the image holds: `main.go` is the composition root, about 85 lines of config and a module list, and `cpbootstrap` is the rest. **`cmd/clips`** is the other binary, a job a workflow runs with `go run` — see [The clips job](clips-job.md).

It runs as **one container beside a postgres and a Redis**: the tile map lives in process, and postgres is where it is kept between boots — see [Durability](durability.md); Redis carries the events between modules and nothing else — see [Telling other modules what happened](#telling-other-modules-what-happened). There is no cache and no second API process.
