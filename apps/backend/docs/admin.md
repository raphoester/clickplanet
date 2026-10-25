# Operator tools (`AdminService`)

**A second router, on a loopback listener: the console.** `props.AdminRPC.Mount` is `props.RPC.Mount` for services an operator calls: same builder, same error net, but `cpbootstrap` serves them on `httpServer.consoleBindAddress` instead of the public router — logging middleware only, no CORS. Empty serves no admin listener; anything but a loopback `host:port` refuses the boot, both in `ServerConfig.Validate` and again in `Run`, and a port already taken refuses it too. They have no authentication, so loopback is their whole protection, and they are off the router Caddy forwards to on purpose: one Caddyfile edit would otherwise let anybody repaint the map. In production they are reached with `docker compose exec backend wget`; see `deploy/vps/README.md`, "Operator tools".

**The admin listener serves every admin service again, for the admin page.**
`httpServer.adminBindAddress` is not loopback (Caddy reaches it on the Docker network), so `cpbootstrap`
builds every admin route a second time behind `Options.AdminGate`: its middlewares around the router, its
interceptors inside the error net (`rpcRoutes.mountBehind`). `cpbootstrap` knows nothing of what the gate checks.
An address with an empty gate, or a gate holding a nil, refuses the boot: the admin services have no authentication
of their own. Empty serves no admin listener.

**The gate is auth's (`internal/auth/internal/admingate`), and `cmd/api` builds it** with `auth.NewAdminGate`,
from `auth.admin.origin` and the internal listener's address, since it runs outside the module: it asks auth
who calls over the internal listener, as another module would. No origin is no admin page, and so no gate; an
origin with auth off refuses the boot.

- **The admin page's origin alone** (`admission.Page`): a call whose `Origin` is another, or none, is
  `permission_denied` before anything is asked. Scripts on the box use the console listener. The router answers CORS
  for that origin alone, with credentials.
- **Then who calls**, but for the admin sign-in calls (`admission.Page.SignIn`: start, complete, sign out), which
  are made before a session exists: `admit_usecase` asks `auth.v1.InternalService/GetOperator` with the cookie
  (`rpc_operators`, 2s), and `guard` maps the answer: no session `unauthenticated`, anybody but an operator
  `permission_denied`. `log_admit` logs each refusal (`admin call refused`), with its origin and never the cookie.
- **A line per admitted call** (`log_calls`, inside the guard): `admin call` with the operator's account id, the
  procedure, the request as JSON and the code, reads included. The audit decorators say what a call did; this line
  says who asked it.

Caddy sends `/<package>.v1.AdminService/*` on the API's domain to it, so `cp_admin`, which is host-only on that
domain, travels from the admin page, a page of the same site.

The admin page is `apps/backoffice` (its [`CLAUDE.md`](../../backoffice/CLAUDE.md)). `planet.v1.AdminService` is the main one, in `proto/planet/v1/admin.proto`; `player.v1.AdminService` (`proto/player/v1/admin.proto`) has `ReconcileTitles`, and the reads that name players, `GetAuthors` and `GetPlayer`, and who plays now, `GetRoster` — see [Player](player.md); `chat.v1.AdminService` (`proto/chat/v1/admin.proto`) has `Mute`, `RemoveMessage`, `GetMessage` and `ListMessages` — see [Mutes](chat.md#mutes) and [Removals](chat.md#removals); `support.v1.AdminService` (`proto/support/v1/admin.proto`) has `ListNotes`, `ListReports` and `DecideReport` — see [Support](support.md); `clips.v1.AdminService` (`proto/clips/v1/admin.proto`) has `KeepSnapshot`, which the clips job calls over ssh, and `ListSnapshots` and `ListRows`, which the admin page reads — see [Clip stats](clip-stats.md); `auth.v1.AdminService` (`proto/auth/v1/admin.proto`) has the admin page's own sign-in and `WhoAmI` — see [Operators](auth.md#operators); `seasons.v1.AdminService` (`proto/seasons/v1/admin.proto`) has `GetRace`, and `metrics.v1.AdminService` (`proto/metrics/v1/admin.proto`) `GetMetrics` — see [The battle reads](#the-battle-reads-listconvictions-getmetrics-getrace). `planetv1controller.AdminService` is its bag of handlers, the way `ClickService` is. `ReassignCountry` runs `clicks/usecases/reassign_country_usecase`, wrapped in `audit_reassign`: every tile `from_country_id` holds goes to `to_country_id`, while the game runs.

- **The move is paced.** `inmemory_tile_storage.Reassign` moves one batch under the lock and returns where to resume; the use case sleeps 50ms between batches. A batch is a quarter of `tilesStorage.subscriberBuffer`, because each tile is one update on every open stream, the clicks still arriving need the rest of the buffer, and a stream that overflows it is closed.
- **Each tile is an ordinary `TileUpdate`** with `Previous` set, not a new event kind: open clients repaint with no frontend release, `counts` move so the toll prices the next click right, and `dirty` puts it in the next flush.
- **A tile `from` retakes behind the scan stays theirs.** The answer reads both counts again at the end, so `from_after` says whether to run it again.
- **`audit_reassign` logs every call at Warn**, dry runs and failures included: it is the only record that those tiles did not change hands through play. It is a decorator for the reason `prom_click` is — handlers here do not log.

Measured on a copy of production's map, before postgres: 22,040 tiles in 4.4s, all 22,040 updates delivered to an open stream, none dropped.

## `GrantCharges`

`GrantCharges(account_id, refill, bomb, enclosures, spread_clicks, shields)` runs `bonuses/usecases/grant_charges_usecase`, wrapped in `audit_grant_charges`: it gives one account any of the five charges, as a box would.

- **The caps hold**: `Held.Granted` is the rule, so a refill and a bomb are one, enclosures stop at `bonus.enclose.held`, spread clicks at `bonus.spread.clicks` and shields at `bonus.shield.held`. The answer is what the account held `before` and `after`, so the operator sees what fit.
- **An account id, never a scope**: a charge is an account's (`bonuses.ParseHolder`). A grant of nothing is `InvalidArgument` (`ErrNothingToGrant`). Planet does not know which accounts exist, so an id nobody holds gets a row nobody reads.
- **Nothing pushes it**: the player sees it on the next `GetCharges`, at its next page load.

## `GetReplay`

`GetReplay(since, until)` runs `ledger/usecases/replay_usecase`: the map as it was at `since` (`opening`, a
`GetMapResponse`), and every act on it from `since` until `until` (unset: now), each with its time and as the live
stream sent it. The frontend's clip generator plays it back (see its [Clips](../../frontend/docs/clips.md)).

- **An event shows what every screen saw** (`Event.Show`), as `ledger.Scene`s: a click, a strike, a shield placed
  and each tile a spread or an enclosure took or struck is a `Change` (a `clicks.TileUpdate` and the shields the tile
  held before); a bomb is its `clicks.Blast`; a spread or an enclosure is then one `Bonus` too, for its effect. So
  a replay goes out as `tile_update`, `bomb_dropped`, `tiles_spread` and `tiles_enclosed`, which
  `planetv1controller/planetmessage` encodes for the stream and the replay alike. An enclosure has no wall: the
  ledger does not keep it.
- **`ledger.Footage` rewinds the map**: each tile opens on what it held before its first scene at or after `since`,
  owner and shields apart, and every other tile on the map now. It reads the scenes past `until` too, to rewind
  across them. The use case reads the map before the ledger, so a change made between the two reads is rewound.
- **What it cannot see it cannot rewind**: a reassign, a paint and a revert write no event, and a reverted
  caller's takes are forgotten, so a tile one of them changed since `since` opens as it is now.
- **It reads memory**, like the other tools here: a `since` older than `ledger.retention` (72h) opens on the map at
  the oldest event kept. It changes nothing and logs nothing. A `since` that is not before `until` is
  `InvalidArgument` (`replay_usecase.ErrInvalidWindow`).
- **Nothing in it names a player**: no scope and no account goes out, as on the live stream.
- A day of production is about 40,000 events, a few MB of JSON.

## `GetLedger`, `GetTerritories`

**What the admin page's home reads, none of it shaped for it**: the page counts, charts and draws on its own side
(`apps/backoffice`'s [The home page](../../backoffice/docs/home.md)), so a new view of the game needs no new endpoint.

- **`GetLedger(since, until)`** is a query (`get_ledger_handler/changes_query`), as `takes_query` is: every change
  the live stream showed in the window, oldest first, and who made it, straight from `planet.ledger_events` in SQL,
  with no domain type on the way. A take and a strike are the click (`clicked`); the tiles a spread or an enclosure
  took or struck, the tiles a bomb emptied, and a shield placed or raised, are changes of their own, unfolded from
  each kind's payload. A shield a bomb struck is left out: the ledger keeps no owner for it.
  - **A parity test runs every kind** (`ledger.Kinds()`) through the store and asks the query for what `Event.Show`
    shows, so the SQL cannot drift from the stream. A new kind fails it until the SQL reads it.
  - **It reads from the head on** (`ledger_head`): past it the address is blanked, so a change with no account would
    have nobody. The primary key finds the window from there; `plan_test.go` pins it.
  - **Who** is the account, or for a change with no account its address, as every operator tool names a player, in
    one string to compare and not to read: `account:<id>` or `scope:<address>`. It names addresses, so it is an operator's read only: never the replay, which goes into the clips' public
    artifacts and names nobody.
  - **Each change is a `LedgerChange`**: when, the `TileUpdate` as the live stream sent it, and who. The answer is
    the changes alone: the window is the caller's, so nothing of the request is sent back. Plain messages
    rather than columns of indexes, on purpose: the contract reads as what it is. Measured with production's volume
    (about 50,000 changes a day): 8 MB of JSON a day but 0.9 MB once Caddy compresses it, the same as columns or
    binary protobuf would send, decoded in about 100 ms in the browser (300 ms for 72 hours).
  - **A window of 72 hours at most**, the ledger's retention, and `since` before `until` (unset: now); anything else is
    `InvalidArgument` (`changes_query.ErrInvalidWindow`). It changes nothing and logs nothing. The last second of
    changes, not flushed yet, is not in it.
- **`GetTerritories`** is the internal service's, handler and all: the tiles each country holds now, and the map's
  size.

## The battle reads: `ListConvictions`, `GetMetrics`, `GetRace`

**What the admin page's Battle page reads, besides `GetLedger` and `GetRoster`** ([The battle page](../../backoffice/docs/battle.md)). Each is a raw read; the page counts and charts.

- **`planet.v1.AdminService/ListConvictions(since, until)`**: every ban the jury passed in the window, oldest first,
  with its readings and its click summary (see [A ban is kept with its reasons](antibot.md#a-ban-is-kept-with-its-reasons)).
  Unset `until` is now; no `since`, or one not before `until`, is `InvalidArgument`; the antibot off is
  `FailedPrecondition`. `list_convictions_handler` maps `Guard.Convictions` straight onto the wire.
- **`metrics.v1.AdminService/GetMetrics`**: every series the process's registry gathers, as `/metrics` would say it,
  with `started_at` (when the counters started) and `read_at`. **The page subtracts two answers**, as one reads the
  metrics poller's files, and never two with different `started_at`. It is served by `cpbootstrap`
  (`cpprom.AdminService`, `mountMetricsAdmin`) on the admin routes, since the registry is the process's, not a
  module's; `/metrics` itself is on the public router, which Caddy does not route. A type the wire cannot carry
  (untyped) is an error (`cpprom.ErrUnknownType`). **The registry carries the Go and process collectors**
  (`cpprom.NewRegistry`): goroutines, CPU seconds, resident memory. The poller's `grep` keeps them out of its files.
- **`seasons.v1.AdminService/GetRace`** is `SeasonService/GetRace`'s handler mounted again: Caddy answers CORS on the
  game's routes for the game's origin alone, so the admin page cannot read the public one.
- `e2e/battle_test.go` signs an operator in, lets a quick `sequencer` ban a guest walking the ids, and reads all
  three from the page.

## `PaintRandomTiles`

`PaintRandomTiles(flag, area, count, proximity, dry_run)` runs `clicks/usecases/paint_random_tiles_usecase`, wrapped in `audit_paint_random`: it paints `count` tiles with `flag`, starting on `area`'s ground (from `clicks.Borders`), or anywhere on the map when `area` is empty.

- **The area is where a patch starts, not a wall.** A fresh draw (a seed) is a tile of the area not wearing the flag; `eligible` counts those. With no area every tile of the map is a seed, `proximity` still grows patches, and `outside_area` is 0. A patch grows into any tile not wearing the flag, across the border too; `outside_area` counts what it took there.
- **`clicks.Pick` is the rule.** Before each draw, with probability `proximity`, it takes an eligible tile touching one already picked (`Geography.Neighbours`); otherwise a seed. 0 is uniform over the area; 1 grows one patch and draws a new seed only when the patch is walled in. Between the two you get a few patches. When the seeds run out, patches keep growing; `picked` is below `count` only when nothing is left.
- **The paint is `Restore`**, the revert's compare-and-set, against the owner read at the pick. A tile somebody takes in between stays theirs, so `painted` can be below `picked`. Paced like the reassign, each tile an ordinary `TileUpdate`. It does not write the ledger, like the reassign.
- The draw is `clicks.SystemRandom`, math/rand/v2's global source; tests pass a seeded `*rand.Rand`.

## Manual bans: `FindPlayers`, `TopPlayers`, `BanPlayer`, `UnbanPlayer`, `RevertPlayer`, `InspectPlayer`

For the patterns no watchdog catches but a person sees on the map. A player is an **account**, or a **scope** (`cpipscope`: the address over IPv4, the /64 over IPv6) for takes made with no account. `BanPlayer`, `UnbanPlayer`, `RevertPlayer` and `InspectPlayer` take a `scope` (any address) **or** an `account_id`, never both (`ledger.ParseCaller`; both, neither or a malformed id is `InvalidArgument`).

- **`ledger` remembers every act of a player on the map**: who, when, the flag, the tile, and what it did to each tile, oldest first. `ledger.Recording` is the tile writer the click chain, `drop_bomb_usecase` and `place_shield_usecase` write through, with one method per act (`Click`, `Spread`, `Enclose`, `Clear`, `Shield`), so each act is one event, a click that changed nothing is none, and a click or a bomb the ban refuses never reaches it. Recording appends through `publishing_ledger_storage`, which publishes `planet.v1.TileTaken` for each take with an account, after it is recorded. A take by somebody else is one more take, not a replacement: a bot painted over as fast as it paints is still in the ledger. Reassigns, paints and reverts write nothing; they show as a change the ledger never saw.
- **The ledger is a log of `ledger.Event`s, and nothing outside a kind's own file asks which kind it holds.** An event does two things: `Replay(see)` hands over each tile it changed, as a `Taking`, and `Entry()` writes it down. `ledger.EventOf(entry)` reads one back through `kinds`, the catalog, which is the only list of kinds. So the storage, the store, `Runs`, `Tally` and the publisher see events and changes, never a kind, and **a new kind is a type and one line in `kinds`**. A take replays as itself.
- **The kinds**: a click that took a tile is a `Taking`, and one a shield stopped a `Striking`, which replays nothing. A spread or an enclosure is one event beside the click's (`Spreading`, `Enclosing`): the `clicks.Impact` of each tile it took or struck, sorted by tile, replayed as a take of each tile it took. A shield placed is a `Shielding`, and replays nothing. **`Tally` counts acts**: a click with a spread that took a neighbour is two takes in `TopPlayers`, and a strike or a shield is none. `TileTaken` is still one per tile taken, so the stats and the seasons count tiles as before.
- **A bomb is one event** (`ledger.Bombing`): who dropped it, when, and the blast as the map cleared it. `drop_bomb_usecase` clears through `Recording.Clear`, and the map's `Clear` reports the flag each tile wore under its own lock (`clicks.Blast.Owners`). A refused drop never reaches it; a bomb in the sea is kept too, and clears nothing. It replays as a take with no country on each tile it cleared, by the bomber. **So a revert undoes a bomb**: a bomber's run on each tile it cleared goes back to the flag it wore, while the tile is still empty, and a bomb breaks the run of whoever it hit. **`Tally` counts takes by event**: a bomb is one take for the bomber in `TopPlayers` and holds nothing, it ends the hold of each tile it hit, and it matches no flag in `FindPlayers`. It is never published, since every tile it changed is a clear: `BombLanded` already says it.
- **The rules are in the `ledger` root, and its package doc states them**. A caller (`ledger.Caller`, a scope or an account) **holds** a tile when the tile's latest take is its own and the tile still wears that paint. A revert gives a held tile back to what it held before the caller's **current run** on it: its own latest takes, walking back while each took the tile from the paint of the one before. An account's run follows it across scopes. Another scope's take breaks the run (A il→ps, B ps→de, A de→ps goes back to `de`), and so does a change the ledger never saw (A il→ps, a revert of A to nobody, A ""→ps goes back to nobody). `Tally` gathers players for `FindPlayers` and `TopPlayers`, `Runs` computes the revert, `ByTakes` and `Top` rank and cut. The use cases only replay the ledger into these, filter through their ports, and call them. The tests for each interleaving are in `ledger_test.go`.
- **Kept in memory and flushed to postgres** (`inmemory_ledger_storage`, behind `ledger.Storage`; see [Durability](durability.md)). Postgres keeps every event for good, without its scope once memory has dropped it; the tools here read memory alone. In memory it is an append-only log of 24-byte records in 1.5 MiB chunks, one per entry, scopes and accounts interned in one table per chunk and countries and kinds in others, so an old chunk takes its strings when it goes. An entry's payload sits beside its record in the chunk, in position order, and a take has none. `Replay` reads each entry back through `EventOf`; `Load` refuses the boot on one that does not read back (`ErrUnknownKind`), so a rollback past a new kind is refused, not skipped. A record is never changed once written, so `Replay` copies the chunk headers under the lock and reads without it: a `TopPlayers` over 4M takes takes ~1s and never blocks a click. `Forget(caller, position)` hides a reverted scope's or account's takes up to the replay it was computed from, so a take made mid-revert still counts.
- **Memory is bounded twice; postgres is not.** `ledger.retention` (72h) drops takes from memory by age, each `ledger.sweepInterval`; `ledgerStorage.maxTakes` (4M) drops the oldest first when a busy stretch fills it, and logs "the ledger is full" once. Production is thousands of clicks per 5 minutes (`clicks_total`), and a spread click takes up to 7 tiles: 15 takes a second fill 4M in three days. Measured at 4M before the account: ~85 MiB heap. The account and the kind add 4 bytes each a take, about 30 MiB more at the cap (not measured).
- **`FindPlayers(flag, area, limit)`** lists every player that took a tile for `flag` on `area`'s ground (empty is the whole map), latest take first, with any running ban on its scope or its account — whichever ends last. **An account is one row across every scope it took from**, with `account_id`, and its `scope` is the one of its latest take, as `InspectPlayer` reads it. Until 2026-10-23 each account on a scope was its own row, so a phone on a mobile network, which gets a new address every few hours, filled the list with one player. The ground comes from `clicks.Borders`, built by `embedded_geodesic_map.Loader.LoadBorders` from the borders blob the frontend paints flags from — see [Map geography](map-geography.md). A blob for another map refuses the boot.
- **`TopPlayers(limit)`** is the same over every flag and the whole map, **most takes first, then most tiles held**, then latest take. Takes lead because they are what a painted-over bot cannot hide. Every player, in both answers, carries `tiles` (held) and `takes` (every take, a tile taken twice counting twice), `active_for` (last take minus first take), and `tiles_per_minute` and `takes_per_minute` over that. A player with `takes` high and `tiles` near zero is painting and being painted over. **`flag_country_id` is the flag it took the most tiles for**, held or painted over since, and of a tie the first by id (`Tally.pickFlags`); a bomb takes no tile for any flag, so a player that only bombed has none. In `FindPlayers` it counts only the takes the list counts, so it is the flag asked. No row carries a name: the admin page asks `player.v1.AdminService/GetAuthors` for those.
- **`BanPlayer(scope | account_id, duration)`** is `bans.Book.Ban`, and refuses the caller's clicks, bombs and shields alike: the same record, ladder and table as a watchdog's ban, and it counts as an offence. It skips `reflagInterval`, and an empty duration takes the ladder's step. Any address is accepted and banned as its scope (`cpipscope.Parse`). **An account is banned alone**: the operator named no scope, and a guest that sheds it with a new cookie is a second ban on its scope away. **It follows `antiBot.bans.enforce`**, and says so in `enforced`. With `antiBot.enabled` false it answers `FailedPrecondition`.
- **`RevertPlayer(scope | account_id, dry_run)`** gives each tile the caller holds back to what it held before its run, by the rule above. A scope reverts every account's takes on it; an account reverts its takes from every scope. `touched` is the tiles it took, `held` those it still holds. **Only a tile still wearing the scope's paint changes** — `inmemory_tile_storage.Restore` is a compare-and-set under the lock, so a tile retaken mid-revert stays retaken. Paced like the reassign (`clicks.Pacing`), each tile an ordinary `TileUpdate`. A tile that was nobody's goes back to nobody, as an update with an empty country. It then forgets the caller's takes, so a second run does nothing; an interrupted one forgets nothing and can be run again.
- **Ban before reverting**: an unbanned player repaints behind the revert.
- **`UnbanPlayer(scope | account_id)`** lifts the caller's latest ban, for a false positive: its offence is
  forgotten, so the next ban takes the step the lifted one took, and a ban still running ends now. **A ban that
  already ended is lifted too** (`Record.liftable`): its end is left as it was, and only the offence goes. Until
  2026-10-11 an ended ban kept its offence, so the people a rule banned by mistake for a day were one flag away
  from a week. **One call forgets one offence**, the latest: an offence before it still counts, since the scope
  may be shared and the account may have run a script last week, and a second call forgets that one. The answer is
  the ban that was lifted (`LastSentence`, read before the change). The row is written before the call
  answers, so a restart keeps the caller unbanned: no restart and no SQL. It lifts the key it is given, so **a guest the jury banned needs two
  calls**, its account and its scope. A caller with no offence left is `NotFound` (`ErrNotBanned`), which a typo or the wrong
  half shows loudly; with `antiBot.enabled` false it is `FailedPrecondition`, as `BanPlayer` is.
  - **It pardons the key it lifts, for `antiBot.evidence.retention` (72h)** (`bans.Pardon`, in the same row
    change). Until then, the jury flags nothing on that key, however guilty it reads. The watchdogs kept what they
    read before the ban, and until 2026-10-20 a rule that still read `certain` banned the caller again five
    minutes after the unban (`reflagInterval`). No evidence outlives retention, so when the pardon ends every
    reading rests on play after the unban.
  - **It does not touch the evidence.** Forgetting it would take a forget in each of the ten watchdogs, keyed by
    scope, by account or by wider prefix. The cost: a rule that still reads the same play the same way bans again
    once the pardon ends, as a first offence. **Fix the rule** if it is the rule. `InspectPlayer` says which one.
  - **A pardoned caller convicts nobody** (`Book.Pardoned`, every key the jury would flag): not even the accounts
    it rotates with. An accomplice that is pardoned is skipped and the rest of the ring still falls.
  - **An operator's `BanPlayer` still bans a pardoned key**: the pardon only holds off the jury.
  - The answer carries `pardonedUntil`. `TestAnUnbanPardonsTheCallerUntilTheEvidenceThatBannedItIsGone` pins it.
- **`InspectPlayer(scope | account_id)`** answers how close the antibot is to a caller, which the `antibot ban` log line cannot: it is only written when a ban fires, so on 2026-09-14 a day of bots and no bans left nothing to read. It is `Guard.Examine`. **An account is read on the scope of its latest take** in the ledger, because the watchdogs judge scopes, with the bans on both; an account with no take inside the retention answers its bans alone and an empty `scope`. It changes nothing — no caller record is created, no watchdog is asked again, no ban is passed. It answers any running ban (`banned`, `bannedUntil`, `offence`, `flags`) and pardon (`pardonedUntil`); per watchdog its `level` and `evidence`, aged the way the jury ages them (past `suspicionWindow` a verdict reads `clear` but keeps its evidence); `suspects` against `minSuspects` and `guilty`, what the jury would decide on a click now (the ban itself would still wait for `reflagInterval`); and the click summary the ban line carries. `tracked` false is a scope the jury has not seen inside its `trackWindow`. Parsed with `ledger.ParseCaller` and refused with `FailedPrecondition` when `antiBot.enabled` is false, as `BanPlayer` is. **A watchdog that reads `clear` has no evidence**: watchdogs only word the rule that tripped, so it says how close a caller is only once some rule has.
- `audit_ban`, `audit_unban` and `audit_revert` log every call at Warn, as `audit_reassign` does. `FindPlayers`, `TopPlayers` and `InspectPlayer` are reads and log nothing.
