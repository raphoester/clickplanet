# Inside the planet module: concepts, not layers

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
  ledger/                         every take and every bomb, and the operator tools that read it
    inmemory_ledger_storage/
    postgres_ledger_store/
    usecases/
  bonuses/                        the boxes, and what each one grants
    postgres_charge_store/
    usecases/
  placements/                     a shield between the hand it left and the map's flush, and the boot that gives it back
    postgres_placement_store/
    usecases/
  allegiances/                    the flag each account plays for, and when it last changed it
    postgres_allegiance_store/
    log_allegiances/
  rules/                          the rule sets, and the one the game plays
    postgres_rule_set_store/
    usecases/
  arenas/                         the blank map an arena battle is fought on: when it opened and closed; its map is the tile map's
    postgres_arena_store/
    usecases/
  quizzes/                        the quiz bank, and how a round's question is drawn
  resets/                         the map's resets, each applied once
    postgres_reset_store/
    usecases/
  refusals/                       the callers planet refuses, and until when: the latest decision on each
    postgres_refusal_store/
    log_refusals/
  facts/                          what callers did, told in order to whoever listens: the log and its relay
  planetv1controller/             the edge: maps the wire to the use cases, nothing else
    get_takes_by_country_handler/takes_query/   InternalService's one read, straight from ledger_events
    get_ledger_handler/changes_query/           AdminService/GetLedger, straight from ledger_events too
    get_tile_holder_handler/holder_query/       ClickService/GetTileHolder, the last take of one tile, from ledger_events
    get_landmass_handler/landmass_query/       ClickService/GetLandmass: the holders from the tiles, the last fortify from ledger_events
    get_holdings_handler/holdings_query/       ClickService/GetHoldings: what each flag holds, from the tile map's tally
  subscribers/                    the edge for events: player.v1.PlayerForgotten → anonymize_takes_usecase,
                                  player.v1.PlayerCreated → grant_sign_in_refill_usecase
  commands/                       the edge for commands: planet.v1.GrantCharges → apply_grant_usecase,
                                  planet.v1.RefuseCaller and AdmitCaller → refusals.Book,
                                  planet.v1.PlayRuleSet → play_rules_usecase, planet.v1.ResetMap → reset_map_usecase,
                                  planet.v1.ArenaCommand → open_arena_usecase and close_arena_usecase
```

- **`clicks/`** — what a click is worth, what the map looks like, and what
  changes when somebody takes a tile. Its root holds the rules that need no
  port: `Board` (which tile ids exist), `Toll` (what a click costs), `Tile` (a tile's
  owner, shields and holder, and `Claimed`, the one rule a player's claim changes them by),
  `Outcome` (taken, shielded or unchanged), `ShieldError`, `Claiming` (a player's claim
  read, decided and written under one lock of the tiles, then its fortify, as an
  `Impact`; see [A claim is one lock](tile-claim.md)), `Pacing` (how an
  operator's bulk change is spread out), `Geography` and `Borders`.
- **`ledger/`** — every act of a player on the map, oldest first. Its root holds `Event` and its kinds (`Taking`, `Striking`, `Spreading`, `Enclosing`, `Bombing`, `Shielding`), `Entry`, `Scene`, `Footage`, `Player`, `Tally`, `Runs`,
  the `Storage` port, `Recording` (the tile writer that records each act) and `Retention`. `FindPlayers`, `TopPlayers` and
  `RevertPlayer` live here: they are one moderation workflow — find, see the refusal, undo. The ban is the antibot's
  (`antibot.v1.AdminService`).
  **The postgres copy is the whole history**: the memory keeps 72h, while `ledger_events` keeps every event and
  only blanks its address past the retention. `planet.v1.InternalService/GetTakesByCountry` reads it there
  (`takes_query`): an account's tiles by the flag they were taken for and by the flag that held them, counting a
  take's own row and each tile in a spread's or an enclose's payload, as `Taking.Replay` does. A parity test runs
  every kind of event through both. It is what `progression` shows as "plays for" and "plays against".
  **`InternalService/GetLedgerTotals(since, until)` reads it there too** (`totals_query`): the events of each kind in
  the window and the accounts that made them, in one scan of `taken_at` (no index on it: seconds over a season). The
  gamemaster asks it once per ended season and keeps the answer (see [The gamemaster](gamemaster.md)).
  **`InternalService/GetMapHistory(since, until, step)` replays it from its first event** (`history_query`,
  `ledger.History`): the map at each step, the first frame every tile held at `since`, each next one the tiles whose
  owner changed, and each tile's changes and shield hits in the window, by `Event.Replay` and `Event.Strikes`, so no kind
  is asked. The read stops at the first event at or after `until`, since positions follow time, and the map's resets
  (`map_resets`, which write no event) wipe it at their time. What a reassign, a paint or a revert wrote is not in it.
  **`ClickService/GetTileHolder(tile_id)` reads it there too** (`holder_query`): the last change `Taking.Replay`
  makes of the tile (a take, a tile a spread or an enclose took, a bomb's clear), as `{account_id, country_id,
  taken_at_unix_ms}`. A clear is no holder. **A take with no account still holds the tile**, with no `account_id`:
  a deleted account's takes keep their flag and time, and the client shows it as a deleted account. It needs no token. It knows no name: the
  client names the account with `player.v1.PlayerService/GetPlayer`, as the admin page joins `TopPlayers`. **It skips
  the events a revert forgot** (`forgotten`, see [Durability](durability.md)), as memory replays none of them, so a
  tile a revert gave back names the take it went back to. Until 2026-11-10 it named the reverted take, of another
  flag, and the card showed nothing. A reassign, a paint or a revert writes no event, and the last second is not
  flushed yet, so the client names the holder only while the tile still wears its `country_id`. A parity test runs
  every kind through both.
- **`refusals/`** — the callers planet refuses, each a scope or an account (`Caller`), until when, and when that was
  decided (`Refusal`, `Decision`). It keeps the latest decision (`Refusal.Decided`), from the commands another module
  sends in planet's words (`RefuseCaller`, `AdmitCaller`). `Book.Refusal(payer)` is what the click, the bomb and the
  shield are refused from. See [A ban is told](antibot.md#a-ban-is-told).
- **`facts/`** — `Log` stamps each fact of a caller later than the one before and keeps it; `Relay` sends them every
  100ms, in order, from one goroutine. What tells: `try_click`, `get_map_usecase`, `listen_for_events_usecase`,
  `bonuses/box_facts` and `allegiances/allegiance_facts`. Planet does not know who listens.
- **`bonuses/`** — the boxes, their schedule, and the running bonuses they grant.
  Its root also holds the rules a bonus plays by: `Terrain` and `Pocket` (what an
  enclose closes) and `Bombing` (where a bomb lands and what it clears).

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
a shield placed while a claim decides waits for its write, and so on. An adapter's test suite embeds it and sets `NewStorage`, then adds only
what is its own — `inmemory_tile_storage` adds the snapshot and the slow-subscriber
tests. A second tile storage runs the same suite by embedding it the same way.
A port with one adapter and no second one coming (`ledger.Storage`) has no suite.

**The controller is the one exception**, at `internal/planet/internal/planetv1controller/`,
because it serves every concept over one Connect service. It only maps.

## Use cases (`<concept>/usecases/`)

**One package per procedure, and each declares its own ports.** Each exports
`New` and a `UseCase` with one `Execute`:

| package | what it does | what it needs |
|---|---|---|
| `clicks/usecases/click_usecase` | validates the country and the tile, then clicks it and answers what the click did | `TilesChecker`, `Tiles`, `CountryChecker` |
| `clicks/usecases/get_map_usecase` | a range of the map as one dense batch, told as a `MapRead` | `Board`, `DenseMapReader`, `Facts` |
| `clicks/usecases/map_density_usecase` | how many tiles there are | `MaxIndexReader` |
| `clicks/usecases/get_budget_usecase` | a caller's allowance, unspent | `ClickBudgetReader`, `Pricer` |
| `clicks/usecases/listen_for_events_usecase` | one client's live feed, heartbeat included, told as a `StreamOpened` | `UpdatesSubscriber`, `BonusFeed`, `RulesFeed`, `Facts` |
| `clicks/usecases/reassign_country_usecase` | gives one country's tiles to another | `Map`, `CountryChecker` |
| `clicks/usecases/paint_random_tiles_usecase` | paints random tiles with a flag, starting on one country's ground or anywhere | `Borders`, `Neighbours`, `Map`, `CountryChecker` |
| `ledger/usecases/find_players_usecase` | who is painting a flag, and where, and whom planet refuses | `Ledger`, `Owners`, `Borders`, `Refusals`, `CountryChecker` |
| `ledger/usecases/top_players_usecase` | who takes the most tiles, every flag, and whom planet refuses | `Ledger`, `Owners`, `Refusals` |
| `ledger/usecases/revert_player_usecase` | gives back what one caller still holds | `Ledger`, `Map` |
| `ledger/usecases/anonymize_takes_usecase` | flushes the ledger, then takes a deleted account off every take postgres keeps, then forgets it as the holder of its tiles | `Ledger`, `Takes`, `Tiles` |
| `ledger/usecases/replay_usecase` | the map as it was at a time, and every act on it since | `Ledger`, `Resets`, `MaxIndexReader`, `DenseMapReader`, `Clock` |
| `bonuses/usecases/claim_bonus_usecase` | redeems a box | `Registry`, `Charger`, `Publisher` |
| `bonuses/usecases/open_quiz_usecase` | shows the caller a quiz's question and starts its clock | `Registry` |
| `bonuses/usecases/answer_quiz_usecase` | checks an answer and grants the charge of a right one | `Registry`, `Charger`, `Publisher` |
| `bonuses/usecases/drop_bomb_usecase` | spends a bomb where it was aimed | `Bombs`, `Map`, `Clearer`, `CountryChecker` |
| `bonuses/usecases/get_charges_usecase` | what the caller holds | `Charges` |
| `bonuses/usecases/use_refill_usecase` | fills the caller's bank with its refill | `Refills`, `Bank`, `Pricer` |
| `bonuses/usecases/grant_charges_usecase` | the operator gives an account charges | `Charger`, `Streams` |
| `bonuses/usecases/apply_grant_usecase` | grants the charges of a `GrantCharges` command, once per grant | `Grants`, `Charger`, `Streams` |
| `bonuses/usecases/grant_sign_in_refill_usecase` | gives a guest's first sign-in its refill, once per account | `Grants`, `Charger` |
| `bonuses/usecases/place_shield_usecase` | spends a shield on a tile of the caller's flag | `Gate`, `Escrow`, `Tiles` |
| `placements/usecases/return_placements_usecase` | at boot, gives back every shield a stop left between the hand and the map | `Placements`, `Escrow` |
| `resets/usecases/reset_map_usecase` | wipes the map once per reset (see [The map reset](map-reset.md)) | `Tiles`, `resets.Store` |
| `arenas/usecases/open_arena_usecase` | saves the world's map, plays the arena's blank one and tells `ArenaOpened`, once per arena (see [Arena battles](arenas.md)) | `Tiles`, `Starts`, `Arenas`, `Outbox`, `InPlay` |
| `arenas/usecases/close_arena_usecase` | deletes the arena's map, plays the world's again and tells `ArenaClosed`, once per arena | `Tiles`, `Arenas`, `Outbox`, `InPlay` |
| `rules/usecases/play_rules_usecase` | plays the set postgres keeps, at boot and when it changes; `Plays` | `Sets`, `Choices`, `Rules` |
| `rules/usecases/save_rule_set_usecase` | the operator makes a set or replaces its values; never the one that plays | `Sets`, `Playing` |
| `ledger/usecases/relay_ledger_usecase` | tells the other modules what the ledger flushed past its cursor, in ledger order: `TilesTaken` and `ShieldsStruck`, one per event in parts of `PartTiles` (4,096) at most, `BombLanded`, `Fortified` and `ShieldPlaced`, each with its position, each tile with its index in its event and a taken one with its point on the globe; a `Runner` | `Ledger`, `Cursor`, `Borders`, `Positions`, `Sender` |

**The interfaces in that last column are declared by the package that calls
them**, not gathered in a `gateways.go` every use case imports. A shared port
file makes every dependency everyone's, so `Click` ends up compiling against the
map reader it never calls. Here, adding a dependency to `get_map_usecase` is
invisible to the others — `inmemory_tile_storage` happens to satisfy several of
these ports at once, which is why `module.go` hands it over several times.

**`click_usecase` is the only one with an interface of its own (`IUseCase`)**,
because the click chain decorates it, from the inside out — `spread_click`, `enclose_click`, `prom_click`,
`throttle_click`, `bonus_click`, `reach_click`, `allegiance_click`, `refusal_click` and `try_click` (`module.go` builds it). The counting is a wrapper rather than a line inside the rule, so
a process that does not want it leaves it out and the rule does not change.

**`try_click` is outside every gate**, so a try the refusal, the allegiance or the throttle stopped is told too, with
what stopped it (`ClickTried.outcome`). It changes nothing and refuses nothing: it observes, as `prom_click` counts.
A gate in the chain would have to tell its own refusals, and three gates would tell one try.

`Geography` is in the `clicks` root, beside the sentinels and `TileUpdate`: the shape of the map, in `geography.go`. It is a model rather than a port — `embedded_geodesic_map` builds one and hands it over. See [Map geography](map-geography.md).
