# Arena battles: a battle on a blank map

**A battle whose rule set has `arena.on` is an arena battle: it is fought on a blank map, and the map from before comes
back, tile for tile, when it ends** (the owner, 2026-11-09). It is slices 1 and 2 of the Capital Siege game mode: the
blank map and back, then each country's capital and the take rule ([Capitals and reach](#capitals-and-reach)). The
knock-out, the speed and the replays come in later slices. **Do not schedule an
arena battle in production before slice 5**: the replays, the map history and the clips read the ledger across the
battle as if the map never changed.

Planet calls the blank map an **arena**; the gamemaster calls the battle an **arena battle**. Planet never learns the
word battle: it opens and closes arenas by the ids it is told.

## The order

Day K ends as the arena battle starts (T1), and day K+1 starts as it ends (T2). The gamemaster's runners tick each
second with no order between them, so **every step waits on a fact kept in postgres, never on a time margin**:

| when | who | what | waits for |
|---|---|---|---|
| T1 | play_schedule | `PlayRuleSet` of the arena battle's set | the clock |
| T1 | play_arenas | judges the battle once: `GetRuleSet(set).arena.on` → `battle_modes` | the clock |
| T1 | play_scoring | `CloseRound` day K, then `OpenRound` of the battle with `from_nothing` | the judgement, for the open |
| ~T1+2 s | scoring | scores day K on the real map → `RoundScored` → `scored_rounds` | — |
| ~T1+3 s | play_arenas | `OpenArena{arena_id, ends_at}` | day K in `scored_rounds`, and no arena before it still open |
| | planet | saves what changed to the world's map, then plays the arena's empty map: new epoch, every stream cut | — |
| T2 | play_scoring | `CloseRound` of the battle | the clock |
| ~T2+2 s | scoring | scores the battle on the arena, from nothing → `RoundScored` | — |
| ~T2+3 s | play_arenas | `CloseArena{arena_id}` | the battle in `scored_rounds` |
| | planet | deletes the arena's map, records `ArenaClosed`, plays the world's again: new epoch, every stream cut | — |
| ~T2+5 s | gamemaster | keeps it in `closed_arenas` (`gamemaster-arenas-closed`) | — |
| | play_scoring, tell_starts | `OpenRound` day K+1 on the world's map, `DayStarted` K+1 | the arena in `closed_arenas` |

**The one rule behind the waits is the plan's** (`moments.Plan`, which the Planner builds `With` the arena battles
kept): **nothing opens on the map while an arena battle that ended before it has its arena open** (`Plan.Waits`). It
holds a round's and a book's open (`DueAt`), the day told (`DayAt`, so `DayStarted` and the missions' draw come after
the map is back), a season's reset (`play_resets_usecase`: a reset before the restore would be undone by it) and the
next arena (`ArenasDueAt`). A battle's round opens only once the battle is judged, about one tick: an arena battle's
round counts from nothing, so its gain is the tiles held when it ends. A plan built with no arena battles (the zero
`Arenas`, as `turn_query` builds it) waits for nothing.

**Retries, crashes and restarts**: the judgement, the scored rounds and the closed arenas are rows; the runners keep
only what they sent in this process, so a boot sends again what is due, and planet keeps each arena by its id. A command
Redis refused stops the tick, and it and every one after it go the next second.

**An arena battle is judged once, as it starts, and kept** (`battle_modes`, enum `battle_mode`): a set edited during the
battle changes no judgement. **A battle that opens its book is plain** (`Plan.Judge`): the season's reset blanked the
map already, and an arena opened then would keep the old season's map aside. NotFound from planet is plain.

## Planet: the arena (`internal/planet/internal/arenas/`)

**One table keeps every map** (`tiles` and `landmasses`, keyed by `map`): `world`, the game's own (`clicks.World`), and
each open arena's, named by the arena's id (`arenas.ID.Map`). The tile map plays one at a time: memory holds it, the
flush writes it, and the boot loads the open arena's map, or the world's (`Arena.Played`). Nothing is copied: the
world's rows stay as they are while an arena plays, and a column added to a tile, or a remap of the ids, reaches every
map.

- **`OpenArena`** (`open_arena_usecase`): under the flush lock and the tiles lock, `Storage.Blank` saves what changed to
  the world's map (the tiles not flushed yet), keeps the arena (`arenas`), and then plays the arena's map, which has no
  row yet. Memory is emptied as `Wipe` empties it: a new epoch and every stream cut, so every page reads the blank map
  again. **An arena opens once, never after it closed, never at or after its end, and never while another is open**
  (`Arena.Open`): a refusal is dropped by the handler and logged at Error.
- **`CloseArena`** (`close_arena_usecase`): under the same locks, `Storage.PutBack` runs `Wipe` on the arena's map in
  one transaction with the arena kept closed (`closed_arenas`) and `planet.v1.ArenaClosed` recorded in planet's outbox.
  Only after the commit does the world's map play again, read as the boot reads it (`Load`'s steps: every tile, every
  lock, each landmass settled), with a new epoch and every stream cut. **An arena that never opened closes too**, with
  the map left as it is, and is told the same way, so day K+1 never waits for ever. One closed already changes nothing.
  The world's map is never an arena's (`ParseID` refuses `world`, and `Blank` and `PutBack` refuse it).
- **One ordered stream**: `commands:planet.v1.ArenaCommand`, a oneof of the two, like scoring's.
- **Neither writes a ledger event.** The ledger keeps the battle's takes as any other.
- **The freeze**: while an arena is open, every play (Click, DropBomb, PlaceShield, UseRefill, ClaimBonus, OpenQuiz,
  AnswerQuiz) is refused with `FrozenRefusal` for `arena.freeze` after the map went blank and from `arena.freeze` before
  the battle's end until the map is back (`Opening.Freezes`, read through `InPlay`, an atomic pointer loaded at boot).
  Only the freeze interceptor refuses: `GetBudget` does not say it, since a page that reads `frozen` locks its clicks
  until it reads the budget again; a click refused shakes as a throttled one. The start's freeze also outlasts GetMap's
  5 s cache.
- **A deleted account** (`anonymize_takes_usecase`) is taken off every map, in memory and then in postgres
  (`Storage.ForgetHolder`), so the world's map played again names it no more.
- **Planet's outbox** (`planet.outbox`, `outbox_event` `planet.v1.ArenaClosed`, relay `planet-outbox`) is new with
  arenas: the map's facts still go through the ledger.

## Capitals and reach

**Each flag starts the arena on its capital, and takes only next to its land or across its waters** (the owner,
2026-11-09).

- **The capitals** (`clicks.Capitals`, from `/map`'s `capitals-<hash>.json`): each country's tile nearest its capital
  city, picked by the map generator (see [`/map/README.md`](../../../map/README.md)). Planet keeps the 235 of the flags
  a player can play for. A region flag (England, Scotland, Wales, Brittany) has no ground, so no capital.
- **The start** (`arenas.Starts`): each capital, held by its country, with `arena.capitalShields` shields (100), read
  from the rules in force as the arena opens. `Storage.Blank` hands it to `Persistence.Open`, which writes it on the
  arena's map **in one transaction with the arena kept** (`open_arena_usecase`'s keep), then fills memory from it as the
  boot does. Before, the arena's rows waited for the next flush: a crash in between would have booted an arena with no
  capital. No ledger event and no holder: no player took them. A one-tile landmass held whole settles locked to its
  holder, as at boot.
- **No shield is added to a capital while an arena plays**, whoever holds it: `PlaceShield` is refused with
  `clicks.ErrCapital` (`FailedPrecondition`, the charge given back) and a fortify raises every tile of the landmass but
  it. The storage knows the capitals and reads `playing != World` under its lock (`capitalLocked`). On the world's map a
  capital tile is a tile like any other.
- **The reach** (`arenas.Reach`): while an arena is open (`InPlay.Playing`), a take of a flag lands on a tile it holds,
  on a tile next to one it holds (`Geography.Neighbours`), or on a coast tile of a body of water where it holds a coast
  tile (`clicks.Waters`, from `/map`'s `waters-<hash>.json`). The board answers through `Storage.HoldsAny`, under one read
  lock; the ocean's coast is a scan of at most 15,294 tiles, only for a tile that touches none of the flag's land.
  - **A click** out of reach is refused by `reach_click`, outside the throttle as `allegiance_click` is, so it spends no
    token: `FailedPrecondition` with a `ReachRefusal` detail, told to the antibot as `CLICK_OUTCOME_OUT_OF_REACH` (an
    attempt, not inspected).
  - **A spread** takes only the neighbours in reach (`Reach.Around`), read after the clicked tile is written.
  - **An enclosure** needs no check: its pocket is walled by the flag's own tiles alone, see [Enclose](enclose.md).
  - **A bomb** takes no tile for the bomber, so the reach does not touch it; slice 4 refuses bombs.
- **The page** learns the map is an arena's from every map read (`GetMapResponse.arena`, stamped by
  `StateBatchDense` from `playing`), and draws the reach and the capitals itself: see the frontend's
  [The arena](../../frontend/docs/arena.md).

## Gamemaster: the arena battles (`internal/gamemaster/internal/arenas/`)

- `play_arenas_usecase` (`gamemaster-arenas`, each second): judges the battle that runs (`Plan.Unjudged`, `Plan.Judge`,
  `rpc_planet_arenas`), then sends each arena command due (`Plan.ArenasDueAt`), once per process, by time, a close
  before an open at one instant (`redis_arena_commands`).
- `arena_closed_subscriber` (`gamemaster-arenas-closed`) → `hear_arena_usecase` keeps the arena closed once.
- `postgres_arena_store` over `battle_modes` and `closed_arenas`; `moments.Plan` reads them with `scored_rounds`.

## Known gaps

- **At T1, about 2 to 4 s** between the cutoff and the blank: clicks land on the real map after day K was read. They
  stay on the world's map, and count as no gain.
- **At T2, about 2 to 3 s** between the restore and day K+1's open: clicks become part of K+1's start.
- **Until slice 5**: `GetTileHolder` (it reads the ledger), the replays, the map history and the clips are wrong across
  an arena battle; an operator's paint during an arena is lost with it.
- **A set with `arena.on` on a plain entry** plays its rules on the real map: nothing blanks, since only a battle is
  judged.
- **A season end moved during an arena battle** cuts the battle; the arena's `ends_at` is the one planet was told, so
  the end's freeze starts late (the close still comes once the cut battle is scored).

## Tests

Slice 2: `arenas`' `reach_test.go` pins a take next to the land, across a water with a coast held, refused across one
without, no reach outside an arena, and a spread past the tile taken or from a struck one; `rules_test.go` the capital
shields of default and their bound. `inmemory_tile_storage`'s `capitals_test.go` pins the start kept with the arena
before any flush, none when keep says no, no shield on a capital in an arena but one on the world's map, a fortify
that skips a capital, the map's `Arena`, and `HoldsAny`; `postgres_tile_store`'s `maps_test.go` an `Open` in keep's
transaction and none when keep refuses or fails. `open_arena_usecase` opens with each capital and its shields;
`reach_click` refuses before the inner click; `throttle_test.go` keeps the token of three clicks out of reach over
HTTP; `click_handler` answers the `ReachRefusal`; `spread_click` takes only the neighbours in reach. The shipped
`waters` and `capitals` are pinned in `embedded_geodesic_map`'s `waters_test.go`.

Slice 1:

`arenas`' tests pin the id, an arena opened once and refused closed, ended or beside another, closed once whether it
opened or not, and the freeze's two windows; the arena part (default never plays in an arena, a freeze past 30 s
refused); `StoreContractSuite` on memory and postgres. `inmemory_tile_storage`'s `arena_test.go` pins the world's map
saved with the tiles not flushed yet, the arena's map saved apart, the world played again tile for tile with shields,
holders and locks, the arena's rows deleted, a holder forgotten on the world while an arena plays, the new epoch and the
cut streams; `postgres_tile_store`'s `maps_test.go` each map's rows apart, a wipe of one map, a holder forgotten on
every map; the migration's test every row kept before it as the world's, and its way back. The use cases' tests run the
real storage. In the gamemaster, `moments`' `arenas_test.go` pins the judgement, the waits and the arena commands of two
arena battles back to back; `play_scoring_usecase`'s `arena_test.go` the open that waits for its judgement,
`from_nothing`, and day K+1 held through a boot; `arenanight` plays an arena battle second by second with scoring and
planet answering late, and with boots in the middle.
