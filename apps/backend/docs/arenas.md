# Arena battles: a battle on a fresh map

**A battle whose rule set has `arena.on` is an arena battle: it is fought on a fresh map, and the map from before comes
back, tile for tile, when it ends** (the owner, 2026-11-09). The replays, the map history and the clips skip it: the
world's map before and after it joins up (see [The ledger's readers](#the-ledgers-readers)).

**An arena is a shell** (the owner, 2026-11-10). It switches the tile map that plays, which empties it, and it starts
the fresh map from a seed ([Seeds](#seeds)). It plays no game rule. Each rule a battle plays on the fresh map is a knob
of its own in its rule set, which plays on any map, the world's too: [the reach](reach.md), [the capitals' rules](capitals.md)
(the knock-out and fixed capital shields). **Capital Siege is one rule set**: `arena.on`, the capitals seed, the reach,
the knock-out and fixed capital shields, no bombs (`charges.bomb.off`), and urf's speed with bigger spread boxes (see
[Capital Siege's set](#capital-sieges-set)). Until 2026-11-10 the arena carried those rules, and a rule could not be
used in another mode or on the world's map.

Planet calls the fresh map an **arena**; the gamemaster calls the battle an **arena battle**. Planet never learns the
word battle: it opens and closes arenas by the ids it is given.

## The order

Day K ends as the arena battle starts (T1), and day K+1 starts as it ends (T2). The gamemaster's schedule, its
[plan tick](gamemaster.md#one-plan-a-tick) and its map tick run each second with no order between them, so **every
step waits on an answer kept in postgres, never on a time margin**. Each call answers at once; one cut by its timeout is
made again a minute later, and planet's swap goes on meanwhile:

| when | who | what | waits for |
|---|---|---|---|
| T1 | play_schedule | the antibot's `PlayRuleSet` of the arena battle's set; planet and the offers keep the set before | the clock |
| T1 | judge_battles | judges the battle once: `GetRuleSet(set).arena.on` → `battle_modes` | the clock |
| T1 | play_scoring | `CloseRound` day K: scoring scores it on the real map → `scored_rounds` | the clock |
| ~T1+1 s | play_arenas | `OpenArena{arena_id, ends_at, rule_set}` → `opened_arenas` | day K answered (`scored_rounds` or `missed_rounds`), no arena before it still open |
| | planet | saves what changed to the world's map, keeps the arena, then plays the arena's empty map: new epoch, every stream cut | — |
| ~T1+2 s | play_schedule | planet's `PlayRuleSet` of the battle's set, decided at T1, then the offers' | the arena in `opened_arenas` |
| | play_scoring | `OpenRound` of the battle with `from_nothing`; its start is T1 | the arena in `opened_arenas` |
| T2 | play_scoring | `CloseRound` of the battle: scoring scores it on the arena, from nothing → `scored_rounds` | the clock |
| ~T2+1 s | play_arenas | `CloseArena{arena_id}` → `closed_arenas` | the battle answered (`scored_rounds` or `missed_rounds`) |
| | planet | deletes the arena's map, plays the world's again: new epoch, every stream cut | — |
| ~T2+2 s | play_scoring, tell_starts | `OpenRound` day K+1 on the world's map, `DayStarted` K+1 | the arena in `closed_arenas` |

**The one rule behind the waits is the plan's** (`moments.Plan`, which the Planner builds `With` the arena battles
kept): **nothing opens on the map while an arena battle that ended before it has its arena open** (`Plan.Waits`). It
holds a round's and a book's open (`DueAt`), the day told (`DayAt`, so `DayStarted` and the missions' draw come after
the map is back), a season's reset (`play_resets_usecase`: a reset before the restore would be undone by it) and the
next arena (`ArenasDueAt`). **Each wait is on the answer to a close, and every close is answered** (see [The gamemaster
as a reconciler](gamemaster-reconciler.md)): a round scoring never opened is answered missed, and an arena planet never
opened closes all the same, so nothing waits for ever. A battle's round opens only once the battle is judged, in the same tick (`Plan.Judged`
hands the judgement to the steps after the judge), and an arena battle's only once planet answered its arena opened
(`opened_arenas`): it counts from nothing, so its gain is the tiles held on the arena when it ends. A plan built with no arena battles (the zero
`Arenas`, as `turn_query` builds it) waits for nothing.

**The battle's set plays on planet only once its arena opened** (`Plan.SetWaits`, asked by `play_schedule_usecase`; the
owner, 2026-11-10). A rule of the set plays on any map, so at T1 it would play 2 to 4 s on the world's map: a knock-out
there would clear a real country's land. Until the arena opened the antibot alone is called; planet keeps the set before,
and is called as soon as the arena is in `opened_arenas`, with the turn's decision time. The offers wait with planet
(2026-11-11), so a battle's boxes come with its map. Every battle's set waits for
its judgement the same way, a plain battle's for about a second. The arena's start freeze covers the moment between
the fresh map and the set: nobody plays the day's rules on it. A battle the plan has no round of (no season runs)
waits for nothing. At T2 the next set plays at once: the end freeze holds the arena until its map is back.

**Retries, crashes and restarts**: the judgement, the rounds scored or missed and the arenas opened and closed are rows; the steps
keep only what was answered in this process, so a boot calls again what is due, and planet keeps each arena by its id. A
call with no answer stops its step, and it and every one after it are made a minute later. **A swap cut by the call's
timeout (30 s) goes on**: planet runs the open, the close and the reset on a context detached from the request, so the
next call finds it done and answers it.

**An arena battle whose round never opened in scoring** (judged in its last second, or its open refused and the server
down at its end) no longer stalls the game: its close is answered missed, the arena closes (it never opened
either), and the day after opens on the world's map. Until 2026-11-10 its round was never scored, its arena never
closed, and every round, day and season after it waited for ever.

**An arena battle whose arena never opens** (judged too late, or `OpenArena` refused because it lands at or after the
battle's end) **gets no open in scoring**: planet answers `OpenArena` only once the map went blank, and the battle's
round waits for it. At its end its `CloseRound` is answered missed, the arena closes with the map left as it is,
and the battle scores nothing. Until 2026-11-10 its round opened at T1 with `from_nothing` all the same, so it was scored
from nothing on the world's map, and each country gained its whole land.

**An arena battle is judged once, as it starts, and kept** (`battle_modes`, enum `battle_mode`): a set edited during the
battle changes no judgement. **A battle that opens its book is plain** (`Plan.Judge`): the season's reset blanked the
map already, and an arena opened then would keep the old season's map aside. NotFound from planet is plain.

## Planet: the arena (`internal/planet/internal/arenas/`)

**One table keeps every map** (`tiles` and `landmasses`, keyed by `map`): `world`, the game's own (`clicks.World`), and
each open arena's, named by the arena's id (`arenas.ID.Map`). The tile map plays one at a time: memory holds it, the
flush writes it, and the boot loads the open arena's map, or the world's (`Arena.Played`). Nothing is copied: the
world's rows stay as they are while an arena plays, and a column added to a tile, or a remap of the ids, reaches every
map.

- **`OpenArena`** (`open_arena_usecase`) names the battle's rule set (`rule_set`). `arenas.Setups` reads that set, never
  the set in force (it plays only once the arena opened): its `arena.freeze` and its seed ([Seeds](#seeds)). A set planet
  keeps none of, or one that cannot play, is refused. Then, under the flush lock and the tiles lock, `Storage.Blank`
  saves what changed to the world's map (the tiles not flushed yet), and `Persistence.Open` keeps the arena (`arenas`,
  with its freeze) and writes the seed's tiles, in one transaction, and the arena's map plays. Memory is emptied as
  `Wipe` empties it: a new epoch and every stream cut, so every page reads the blank map again. **An arena opens once,
  never after it closed, never at or after its end, and never while another is open** (`Arena.Open`): a refusal is
  `FailedPrecondition`, logged at Error. An arena open already answers open.
- **`CloseArena`** (`close_arena_usecase`): under the same locks, `Storage.PutBack` runs `Wipe` on the arena's map in
  one transaction with the arena kept closed (`closed_arenas`). Only after the commit does the world's map play again,
  read as the boot reads it (`Load`'s steps: every tile, every lock, each landmass settled), with a new epoch and every
  stream cut. **An arena that never opened closes too**, with the map left as it is, and answers closed, so day K+1
  never waits for ever. One closed already changes nothing, and answers closed again.
  The world's map is never an arena's (`ParseID` refuses `world`, `InvalidArgument`, and `Blank` and `PutBack` refuse
  it).
- **Two calls on planet's `InternalService`** (`open_arena_handler`, `close_arena_handler`), each run on a context
  detached from the request (`context.WithoutCancel`): a caller that gives up never stops a swap halfway. Planet orders
  them under the map's locks. Until 2026-11-11 they were one ordered command stream, `commands:planet.v1.ArenaCommand`,
  answered by `ArenaOpened` and `ArenaClosed` through planet's outbox, which went with them.
- **Neither writes a ledger event.** The ledger keeps the battle's takes as any other; its readers tell them apart by
  time (see [The ledger's readers](#the-ledgers-readers)).
- **The freeze**: while an arena is open, every play (Click, DropBomb, PlaceShield, UseRefill, and the offers'
  ClaimBonus, OpenQuiz, AnswerQuiz) is refused with a `FrozenRefusal` for the battle's `arena.freeze` after the map went blank and from it
  before the battle's end until the map is back (`Opening.Freezes`, read through `InPlay`, an atomic pointer loaded at
  boot). The freeze is the opening's, read from the battle's set as it opened and kept in `planet.arenas.freeze_ms`, so
  a boot keeps it; `InPlay` reads no rules.
  Only the freeze interceptors refuse: planet's, and the offers', which ask planet's `GetFreeze` on each claim,
  opening and answer, since only planet knows an arena's freeze (see [The offers](bonuses.md#the-freeze)). `GetBudget` does not say it, since a page that reads `frozen` locks its clicks
  until it reads the budget again; a click refused shakes as a throttled one. The start's freeze also outlasts GetMap's
  5 s cache.
- **A deleted account** (`anonymize_takes_usecase`) is taken off every map, in memory and then in postgres
  (`Storage.ForgetHolder`), so the world's map played again names it no more.

## Seeds

**An arena's fresh map starts from a seed its rule set names** (`seed`, a `oneof` of patterns; the owner, 2026-11-10).
The arena defines the contract and knows no pattern (`arenas.Seed`: `Tiles() []clicks.StartTile`; `arenas.Seeds`:
`SeedOf(rules.Set) Seed`). The patterns are in `internal/planet/internal/seeds/`: `seeds.Part` reads the set's
pattern, and `seeds.Catalog`, built with what the map gives (`seeds.Ground`: the capitals), lays it.

- **`capitals {shields}`** (`seeds.Capitals`): each capital, held by its country, with `shields` shields (0 to 255,
  past `charges.shield.perTile`). `default` holds `shields` 100. The capitals are the map's (`clicks.Capitals`, from
  `/map`'s `capitals-<hash>.json`): each country's tile nearest its capital city, picked by the map generator (see
  [`/map/README.md`](../../../map/README.md)). Planet keeps the 235 of the flags a player can play for; a region flag
  (England, Scotland, Wales, Brittany) has no ground, so no capital. No ledger event and no holder: no player took them.
  A one-tile landmass held whole settles locked to its holder, as at boot.
- **A new pattern** is a message in `Seed`'s `oneof`, a type in `seeds` and one case in `seeds.Part`. The arena does not
  change. A set that holds the pattern default holds is overlaid value by value; a set that holds another plays its own
  alone (`cpproto.Overlay`, pinned in its tests).
- The seed is written **in one transaction with the arena kept** (`open_arena_usecase`'s keep), then memory is filled
  from it as the boot does. Before 2026-11-10 the arena's rows waited for the next flush: a crash in between would have
  booted an arena with no start.

## The rules a fresh map is played with

Each is a knob of its own, read from the rules in force, on whatever map plays:

- **The reach** (`reach`): a take lands only next to the flag's land or across its waters. See [The reach](reach.md).
- **The knock-out** (`capitals.knockOut`) and **fixed capital shields** (`capitals.fixedShields`): see
  [The capitals' rules](capitals.md).
- **The page** learns them from `GetChargeRules` (`reach`, `knock_out`, `capital_shields_fixed`), read again on
  `rules_changed`: see the frontend's [The reach](../../frontend/docs/reach.md) and
  [The capitals](../../frontend/docs/capitals.md). No map read says whether the map is an arena's: `GetMapResponse.arena`
  went with slice 5 (field 6 reserved), as nobody read it.

## Capital Siege's set

`capital-siege` in production (saved by hand, not scheduled): urf's values (tempo 3, its toll, boxes and quizzes every
30 to 60 s, its enclose, shield and welcome values, flags locked), plus `arena {on, freeze 8s}`, `seed.capitals.shields`
100, `reach`, `capitals {knockOut, fixedShields}`, `charges.bomb.off`, and a faster spread: a pool of 50
(`charges.spread.clicks`), up to 25 a box (the offers' `spread.maxPerBox`), weighed 2 against 1 for refill, enclose
and shields. Since 2026-11-11 the boxes, the quizzes, the welcome and what a box grants are in the offers' set of the
same name, which their migration copied from planet's (see [The offers](bonuses.md#where-the-sets-came-from)). Every
value is edited on the backoffice. Its antibot set of the same name is still to make before it is scheduled.

## Gamemaster: the arena battles (`internal/gamemaster/internal/arenas/`)

- `judge_battles_usecase` (the first step of `gamemaster-plan`, each second): judges the battle that runs
  (`Plan.Unjudged`, `Plan.Judge`, `rpc_planet_arenas`' read of `GetRuleSet`), once.
- `play_arenas_usecase` (the second step of `gamemaster-maps`, each second): calls each arena command due
  (`Plan.ArenasDueAt`) until planet answers it, by time, a close before an open at one instant (`rpc_planet_arenas`),
  and keeps the answer right after the call (`opened_arenas`, `closed_arenas`). An open names the battle's set
  (`ArenaCommand.Set`, `OpenArena.rule_set`). A refusal is kept for the process: the battle's round never opens.
- `play_schedule_usecase` asks the planner whether planet's set waits (`Planner.SetWaits`, `Plan.SetWaits`) while
  planet has not played the turn's set, and only for a battle's turn.
- `postgres_arena_store` over `battle_modes`, `opened_arenas` and `closed_arenas`, read in one statement (`ArenasKept`);
  `moments.Plan` reads them with what scoring answered (`Kept`).

## The ledger's readers

**The replays, the map history and the clips skip an arena: the world's map before it and after it joins up** (the
owner, 2026-11-10; slice 5). A replay of the battle itself is not built. No ledger event marks the open or the close,
and no copy of the ledger is made: each reader takes the stretches from planet's own rows (`arenas.opened_at`,
`closed_arenas.closed_at`, `postgres_arena_store.Stretches`; an arena that never opened played no stretch).

- **The one rule** (`arenas.Stretch.Holds`): a ledger entry was made on an arena's map when it is after the arena's
  open and before the second it closed. The ledger keeps whole seconds and the rows full time: a world take right after
  the close can share the close's second, and an arena take never can, since the freeze holds play from `ends_at −
  freeze` until the close. `arenas.Maps` gathers the stretches: `World()` is when the world's map played (around every
  stretch), `Playing()` when the map on screen now played (the open arena's since it opened, or the world's), and
  `Played.Bounds` hands the same rule to SQL as two arrays of times.
- **World readers keep only the world's entries**:
  - `GetReplay` (`replay_usecase`, `ledger.Footage`): no scene of an arena, so each tile opens on the world's flag and
    the replay plays straight through the battle. **While an arena plays**, it rewinds from the world's map as postgres
    keeps it (`Storage.Kept`: the world's rows on a board of their own, exact from the open on, since no flush writes
    them). The answer lists each stretch of the window (`GetReplayResponse.asides`, `ReplayAside {opened_at,
    closed_at}`, no close while it plays). The clips' replay, cast window and history read it; so does the backoffice's
    replay file.
  - `GetMapHistory` (`history_query`, `ledger.History`): the frames stand still over the battle, and no hit counts its
    fighting. It feeds the clips' sagas and chapters, and the gamemaster's `GetSeasonReplay` (the `/archive` replay and
    heatmap). An arena finale's 10-second replay shows the world's map standing still.
  - `GetLedger` (`changes_query`, in SQL, so a page still holds 20,000 events): the clips' cast names nobody for a take
    in an arena, and the backoffice Home counts no battle's change as won or lost.
  - `RevertPlayer` (`ledger.Runs`): an arena take neither breaks nor makes a run, so a revert after a battle never
    blanks a world tile its flag happens to wear. **Refused while an arena plays** (`ErrArenaPlays`,
    `FailedPrecondition`): `Forget` hides by position, and would hide the world's takes the revert did not undo.
- **Map-that-plays readers keep the entries of the map on screen**: `GetTileHolder` (`holder_query`) and `GetLandmass`'s
  last fortify (`landmass_query`). While an arena plays, a capital nobody took there names nobody; after it, no arena
  take names a holder.
- **The clip generator tells no round that ended in a stretch** (`roundStoryOf`, from the replay's `asides`): its
  fighting is in none of the replay, and the 3 hours before its end are the world's takes from before it.

**What counts takes sees the arena's**, as the world's: scoring's round numbers (`GetRound`, Battle Top 1-3, the
chat's podium) and season standings, progression's `tiles_taken`, streaks, fortifies and fronts, the Hall of Fame's
totals (`GetLedgerTotals`), `TopPlayers` and `FindPlayers`' takes, the antibot and the alerts' attack pings. Missions
count none: a fact counts only inside a day, and a battle is no day.

## Known gaps

- **At T1, about 2 to 4 s** between the cutoff and the blank: clicks land on the real map after day K was read. They
  stay on the world's map, and count as no gain.
- **At T2, about 2 to 3 s** between the restore and day K+1's open: clicks become part of K+1's start.
- **An operator's paint during an arena** is lost with it.
- **`TopPlayers` and `FindPlayers`' tiles held** match each tile's last take in the ledger to the map that plays, arena
  takes included: just after a battle, a tile the arena's taker's flag also wears on the world's map counts as held.
  Telling them apart would take the arenas into `find_players_usecase`, past five ports.
- **An arena with a freeze under 1 s** may read a take of its first second as the world's.
- **A set with `arena.on` on a plain entry** plays its rules on the real map: nothing blanks, since only a battle is
  judged.
- **A set with `arena.on` and a freeze of 0**: the fresh map plays the day's rules for the second before planet plays
  the battle's set.
- **The knock-out without the reach**: a flag that is out can still take tiles. With the reach, a flag with no land
  reaches nothing. Fix it when a mode needs it: the allegiance refuses a take for a flag that does not stand.
- **A season end moved during an arena battle** cuts the battle; the arena's `ends_at` is the one planet was told, so
  the end's freeze starts late (the close still comes once the cut battle is scored).

## Tests

Slice 5: `arenas`' `maps_test.go` pins the rule at the open, at the close's second and for an open arena, the map that
plays and the asides of a window; `StoreContractSuite` the stretches of every arena that opened. `ledger`'s
`history_test.go`, `footage_test.go` and `ledger_test.go` pin a battle's takes and knock-out on no frame, no hit, no
scene and no run; `replay_usecase` a replay across an arena and one while it plays, rewound from the world's map kept
(`inmemory_tile_storage`'s `kept_test.go`); `revert_player_usecase` the refusal and a revert after a battle that blanks
no world tile; `history_query`, `changes_query`, `holder_query` and `landmass_query` the same on postgres, with their
plans; the handlers the asides and the `FailedPrecondition`; the frontend's `round.test.ts` no round story in a stretch.

The shell (2026-11-10): `arenas`' `seed_test.go` pins a setup read from the set named, a set that cannot play refused,
and a failed read that is no refusal; `arena_test.go` and `in_play_test.go` the opening's own freeze; `seeds`'
`seeds_test.go` default's capitals, a set's own shields, the pattern held alone, the bound, and a default with no
pattern; `cpproto`'s `overlay_test.go` a oneof case merged with the same and played alone over another;
`open_arena_usecase` opens seeded and frozen as the battle's set says and refuses a set planet keeps none of; the
migrations' `rule_knobs_test.go` the capital shields moved to the seed and back, and an arena kept before its freeze;
`postgres_arena_store` keeps the freeze. In the gamemaster, `moments`' `arenas_test.go` pins the set that waits for its
judgement and for its arena opened, and a turn of no battle that waits for nothing; `play_schedule_usecase` the antibot
called at T1 and planet once the wait ends with the turn's decision time, a plain turn never asked, a wait that calls
the antibot once, and a failed read that calls nothing.

Slice 3: `clicks`' `knock_out_test.go` pins the rule and the standing; `inmemory_tile_storage`'s `knock_out_test.go`
the take of a capital that clears the flag's other tiles in one change after the take, the taker keeping the capital,
a flag out once, a blank capital that still falls, nothing on the world's map, an enclosure that knocks out once its
pocket is written, the flag out saved by the flush (again after a failed one) and read at boot, and gone with its
arena; `postgres_tile_store`'s `knockouts_test.go` the rows of each map, kept once, deleted by a wipe, none on the
world's; the migrations' `knockouts_test.go`; `ledger_test.go` the event's entry, takes and scenes, and `Recording`
noting it after the take; the relay's `FlagKnockedOut`; the stream's sink and `GetMap`'s out flags;
`allegiances`' arena rule and `Book`.


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
opened or not, and the freeze's two windows; `close_arena_usecase`'s each close answered, the map left as it is for one
never opened; `open_arena_usecase`'s an arena opened once; `open_arena_handler`'s and `close_arena_handler`'s each code,
and a swap that goes on when the caller gives up; the arena part (default never plays in an arena, a freeze past 30 s
refused); `StoreContractSuite` on memory and postgres. `inmemory_tile_storage`'s `arena_test.go` pins the world's map
saved with the tiles not flushed yet, the arena's map saved apart, the world played again tile for tile with shields,
holders and locks, the arena's rows deleted, a holder forgotten on the world while an arena plays, the new epoch and the
cut streams; `postgres_tile_store`'s `maps_test.go` each map's rows apart, a wipe of one map, a holder forgotten on
every map; the migration's test every row kept before it as the world's, and its way back. The use cases' tests run the
real storage. In the gamemaster, `moments`' `arenas_test.go` pins the judgement, the waits, the battle's round that waits for its arena
opened, a battle whose arena never opens with no open and its close due, and the arena commands of two arena battles
back to back; `play_scoring_usecase`'s `arena_test.go` the open that waits for its judgement, `from_nothing` once the arena
opened, day K+1 held through a boot, a battle judged in its last second told missed, and a battle whose arena never opens
told missed with no open; `judge_battles_usecase`'s a battle judged once; `play_arenas_usecase`'s each answer kept, a
call planet did not answer made again a minute later, a refused open, and an arena closed after its round was missed;
`arenanight` plays an arena battle second by second, at once and with planet slower than the call, with boots in the
middle, one whose day before scoring could not answer with the server down until an hour after its end, and one whose
open planet refuses at its end, so its round never opens.
