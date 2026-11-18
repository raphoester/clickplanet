# Boards and eras

**A tile id means a place only on its board, so every row and every event of a tile names its board, and every event
its map** (PR 2 of [Boards](../../../docs/boards.md), 2026-11-17). **The process carries every board it ships, and
plays any of them** (PR 3, 2026-11-18): an arena on a board of its own (`arena.board`), a history that draws each era on
its board, and every threshold sized from the board ([Sized from the board](board-scale.md)). **It ships `earth-2` and
`siege-1`** (PR 4, 2026-11-18), as files and rows: no set names `siege-1` until PR 5. **The world switches to `earth-2`
once, when the operator calls `SwitchBoard`** (PR 6, 2026-11-18, [below](#the-worlds-switch-of-board-switchboard)). **It
plays `earth-2` since 2026-11-18 13:41:31 UTC**: clicks held 7.9 s (see [Boards](../../../docs/boards.md#the-switch-live)).

## The words in the code

- **`clicks.BoardID`** is a board's row in `planet.boards`, a `smallint`: what each row and event names its board by.
  **`clicks.BoardName`** is its name everywhere else: its directory in `/map/boards`, and the wire (`"earth-1"`).
- **`clicks.Stage`** is a map on the board it plays (`{Map, Board}`; `clicks.World.On(board)`): where a write lands,
  and what each event of the ledger names.
- **`eras`** (`internal/planet/internal/eras/`) keeps the boards planet knows and each map's eras: `eras.Board` (id,
  name, hash), `eras.Era` (map, board, start), `eras.Store` (`Boards`, `Latest`, `Covering`, `Start`) and
  `postgres_era_store`, with `MemoryStore` and `StoreContractSuite` behind the `testing` tag. `eras.Boards` are the
  boards this process carries, each matched with planet's row of it, and it names a board by its row (`Name`) or finds
  it by name (`Named`). `eras.Grounds` is each one's ground by row or by name, `eras.Maps` the stage a map plays.
- **`clicks.Ground`** is one board's geometry as the rules read it: its geography, borders, bridges and capitals, and
  its spacing, measured once. **`clicks.Played`** is the ground of the board the map in play plays: the tile map swaps
  it with its stage, and every rule reads it (the neighbours, a bomb's blast and a rally's ring, the borders, the
  bridges, the capitals, the tile count). **`clicks.Grounds`** is what the tile map finds a stage's ground by.
- **`embedded_board.Registry`** is every board this process carries, and **`cpboards`** (`internal/shared/cpboards`)
  reads their manifests for any module, with each board's scale against `earth-1`.

## The boards this process carries

**`embedded_board.Registry` holds every board under `generated/map/boards`** (`cpboards.Files()`, which adds the tests'
fixture under the `testing` tag). At boot it reads each manifest and refuses a lattice it cannot walk: an order but `three` or `canonical`, a warp
that is not three numbers, corners that are not 36 ([Map geography](map-geography.md)). It builds a board's ground (`Registry.Ground`: the positions rebuilt from the
runs and checked against the manifest's hash, the adjacency from the lattice, the borders, bridges and capitals, each
file refused if it holds another tile count than the manifest, and the provinces of a board that has them, refused if cut
on other borders or if a tile's province is on another landmass: `clicks.Provinces`, by `clicks.ProvinceID`, which no
rule reads yet, [Provinces](../../../docs/provinces.md)) **the first time something asks for it**, once, and
keeps it: about 57 bytes a tile, about 0.1 s for `earth-1`. The boot asks for the map in play's board alone, so a board
no map plays costs its manifest. One log line a board as it loads:

```
board loaded board=earth-1 asset=board-725d8920.json tiles=260503 edges=761422 degrees="[250 527 1154 4072 6368 6471 241661]"
  borders=borders-e6851c0e.bin bridges=744 capitals=235 took=95ms
```

`gameMap.maxIndex` went with it: a board's tile count is its manifest's, and the tile map, `GetMap`, `MapDensity`,
`GetTiles` and the territories read the board in play's (`Played.Tiles`, `CheckTile`, `OffMap`).

## The schema

Migration `20261117133227_boards_and_eras`:

- **`boards`** `(id smallint, name text unique, hash text unique)`, seeded with `(1, 'earth-1', '3fb28bdf')`. The hash
  is the manifest's `positionsHash` in 8 hex (1068665823 for `earth-1`; see
  [`/map/README.md`](../../../map/README.md)): every tile's position in id order, so it moves exactly when a tile's id or
  place does. Not the manifest's own hash: the manifest names the board's other files, and a terrain or borders drawn
  again rewrites it (#864 did, `board-f7db567f.json` to `board-725d8920.json`) with every tile where it was. A board
  ships with the migration that adds its row, and its hash never moves.
- **`eras`** `(map, board, started_at)`, keyed by `(map, started_at)`: a map's latest era is the board it plays. Seeded
  with the world on `earth-1` since the epoch, and each arena that ever opened on `earth-1` since its open.
- **A `board smallint NOT NULL` column**, a foreign key to `boards` (on `tiles` until PR 6, below), on `tiles`,
  `landmasses`, `mines` and `viruses`, and
  **`map text` and `board smallint`** on `ledger_events`. Each was added with a constant default, a change of the catalog
  alone, and the default dropped right after, so every write must name it. The ledger's foreign key and its `map`
  check are `NOT VALID`: they hold for every row written since, and the rows before hold the defaults, so no scan of
  millions of rows runs under the table's lock at boot.
- **The backfill** names `earth-1` everywhere, and moves each past arena's events to its map: an event after the arena's
  open and before the second it closed (the old `arenas.Stretch.Holds`), or still open. It is the last read of those
  times: one scan of the ledger, and an update of the arenas' events alone.
- The vandal squad's `graffiti` got `board text NOT NULL` (migration `20261117133227_graffiti_board`): see
  [The vandal squad](vandalsquad.md).

**`tiles.board` has no foreign key since PR 6** (`20261118123254_tiles_board_without_foreign_key`): a switch writes a
million tiles in the transaction that holds every click, and the key's check of each row took 22 of the 26 seconds of
the `COPY` in the rehearsal (see [Boards](../../../docs/boards.md#pr-6-the-rehearsal)). Every tile names its stage's
board, which its era names (whose key stays), and the boot refuses a map whose rows name another board than its era's.
`landmasses`, `mines`, `viruses` and `ledger_events` keep theirs.

Migration `20261118092225_earth2_and_siege1` ships PR 4's boards: `(2, 'earth-2', '5a61c10a')` and `(3, 'siege-1',
'e219e852')`. No era names them, so nothing plays them; the boot matches each carried board with its row (below), and
builds a board's ground only when a map plays it. Its way down deletes the two rows, which a row of a tile, an era or an
event naming either refuses.

`internal/planet/internal/migrations/boards_and_eras_test.go` runs it on a real postgres from the version before it:
`earth-1` everywhere, each arena's events on its map (the second of an open and of a close included), the defaults gone,
an unknown board refused, and the way back and up again. It is the one test of a migration's rows (see
[Testing](testing.md)): a wrong window would misfile history for good.

## Writes

- **The tile map plays one stage, on its board's ground** (`inmemory_tile_storage.Storage.stage`, read by `Stage()`).
  `Load` takes it, sizes the map from its board's ground (`clicks.Grounds`) and makes that ground the one played
  (`clicks.Played`); every flush writes the stage on each tile, lock, mine and virus (`Persistence.Save`, `SaveMines`,
  `SaveViruses` and `Open` take a `clicks.Stage`). `Blank` plays an arena on the stage it is given, its board the one
  its set names (`arena.board`, see [Arena battles](arenas.md)) or the world's, and hands that stage to `keep`, so
  `open_arena_usecase` starts the arena's era in the open's transaction (`eras.Store.Start`). `PutBack` plays the world
  again on the stage of its era, which `close_arena_usecase` reads (`eras.Maps.Playing`): the world's era resumes,
  whatever board the arena played. Each swaps the ground in play with the stage, under the same locks.
- **Every event names the stage its write ran on.** `ledger.Entry.Stage`; `ledger.Storage.Append(on, event)`; the
  memory ledger interns each chunk's stages as it does kinds (a byte a record), and the flush `COPY`s `map` and `board`.
- **No swap of the stage comes between a write and its record.** `Storage.During(write)` read-holds `staging`, a
  `sync.RWMutex` that `Load`, `Blank` and `PutBack` hold while they swap, taken before the flush lock and the tiles
  lock. `ledger.Recording` (a click, a spread, an enclosure, a bomb, a shield), `VirusRecording` and `MineRecording`
  each run their write and their `Append` inside one `During`, and the event takes the stage `During` handed it. A write
  never calls `During` again. Without it a click made on the world's map in the milliseconds an arena opens could be
  filed under the arena's. `arena_test.go`'s `TestASwapOfTheMapWaitsForAWriteAndItsRecordToEnd` pins it.

## The boot's guard

The planet module, before the tile map loads (`module.go`):

1. **Every board this process carries** (`embedded_board.Registry.Carried`): the name of its directory and its
   manifest's `positionsHash`, which `Registry.Ground` checks the rebuilt positions against.
2. **Each matched with planet's row** (`eras.BoardsOf`): a board carried that `boards` has no row of, or names under
   another hash, refuses the boot (`eras.ErrUnkept`): a board regenerated under its name, its tiles moved, is another
   board. A board ships with the migration that adds its row; a test binary carries the fixture, whose row
   `planet.Migrate` and `eras.KeepFixture` keep.
3. **The map that plays** (the open arena's, or the world's), and **its latest era** (`eras.Playing`): no era refuses the
   boot (`eras.ErrNoEra`), and an era on a board this process does not carry too (`eras.ErrNotCarried`).
4. **Its rows** (`inmemory_tile_storage.Load`, `fillLocked`, from `Persistence.Boards`, the boards a map's tiles, locks,
   mines and viruses name): a row on another board than the stage's refuses the load (`ErrOtherBoard`), and so the boot.
   `Blank` and `PutBack` fill the same way.

Until 2026-11-17 a skipped remap booted old ids onto new ground with only a Warn for the ids past the end. That Warn
stays, for a row past the end of its own board.

## Readers

Each reads an event's map and board from its columns (or the memory ledger's record), never from a time:

| reader | keeps |
| --- | --- |
| `GetReplay` (`replay_usecase`, `ledger.Footage`), the clips' replay and cast | the world's map on the board it plays (`eras.Store.Latest`), from its era's start |
| `GetMapHistory` (`history_query`, `ledger.History`), the season replay | the world's map on the board of each era, with the cut |
| `GetRecentMapHistory` (`recent_history_query`, `ledger.Rewind`) | the world's map on the board it plays, from its era's start |
| `GetLedger` (`changes_query`), the backoffice Home, the clips' cast, the vandal squad | the same, in SQL (`map = $5 AND board = $6`) |
| `GetTileHolder` (`holder_query`), `GetProvince`'s last fortify (`province_query`) | the stage that plays (`Storage.Stage`) |
| `RevertPlayer` (`ledger.Runs`) | the stage that plays, refused unless it is the world's |
| `FindPlayers`, `TopPlayers` (`ledger.OnBoard`) | the board that plays, on any map |
| the relay (`relay_ledger_usecase`) | every event, with each tile's ground and point from its event's board (`Boards.Of`; none for a board not carried) |

Whether an arena plays, for the readers that then rewind from the world's map kept, is the stage that plays too
(`Stage().Map != clicks.World`). `arenas.Stretch` stays for `GetReplay`'s `asides`, which the clip generator reads.

**The history draws each era on its own board, with the cut** (2026-11-18). `history_query` reads the world's eras from
the blank map on (`eras.Store.Covering`), replays each event on the era in force at its time, and keeps it only when it
names that era's board. At an era's start (`ledger.History.Cut`) the frames before are on the board before, **the map is
carried over onto the new board** as the switch carried it, each of its tiles taking the owner of the tile of the board
before it comes from (a `ledger.Carry`, which `history_query` reads through its `Carries` port for each cut; 2026-11-18,
PR 6), and the first frame after is whole, every tile held, as the window's first frame is: a reader clears its globe
and draws that frame, on the new board. The carry-over is no take: it hits no tile. **A cut whose carry-over is not read
refuses the history**, never a blank board drawn: the module reads it from the carry-over files this process carries
(`carriesOf`, `embedded_board.Registry.Carryover`, kept once read), and a cut with no file between its boards is
refused. Each tile's hits name their era (`ledger.EraTile`, `TileHits.era`), since two boards' tile ids
collide. A reset and a cut at one time play the reset first.

**The rewinders keep the board the world plays now** (`GetReplay`, `GetRecentMapHistory`): they rewind from the map in
memory, and no rewind crosses into another board. **A window across a switch of the world's board starts at it, as
after a reset** (2026-11-18, PR 6), on the map as the switch carried it, rewound from memory: `replay_usecase` moves
`since` to the start of the world's latest era, and refuses a window that ended before it (`ErrBeforeSwitch`,
`FailedPrecondition`), and `recent_history_query` starts at the first step at or after it (`Recent.Wiped`). So for the
24 hours after the switch, the clips, the players' videos and the last 24 hours start at it, and each covers one era.
Their hits are that era's (`ledger.InEra`).

## The world's switch of board (`SwitchBoard`)

**`planet.v1.AdminService/SwitchBoard{map, board, dry_run}` moves the world's map onto another board, its land carried
over, once** (PR 6 of [Boards](../../../docs/boards.md), 2026-11-18). An MCP tool like every admin method
([Claude as a connector](connector.md)); the operator's steps are the runbook in [Boards](../../../docs/boards.md#the-operators-runbook).

- **The carry-over** (`clicks.Carryover`): one parent on the old board for each tile of the new, read from
  `generated/map/carries/carry-<from>-<to>-<hash>.bin` (`embedded_board.Registry.Carryover`, a copy of `/map/carries`
  by `make map`; the format is in [`/map/README.md`](../../../map/README.md#the-carry-over)). The file names both boards'
  positions hash and tile count, and is refused when either is not the board this process carries
  (`clicks.ErrCarryover`). An old tile's children are the new tiles it is parent of, in id order. `earth-1` to `earth-2`:
  997,267 new tiles carried, 7,415 blank (new coasts and islands), 66 old coast tiles with no child.
- **`switch_board_usecase`** (in `eras/usecases`, logged by `log_switch_board`) refuses another map than the world's
  (`InvalidArgument`), and a board this process does not carry, one with no carry-over to it, an arena open and the
  board the world plays (`FailedPrecondition`). The handler runs it on a context detached from the request, as an
  arena's swap.
- **`Storage.Switch`**, under the staging, flush and tiles locks: it reads the stage again (`clicks.ErrNotInPlay` while
  an arena plays, `clicks.ErrPlaysAlready` for a second call), saves what changed, and carries memory over
  (`carryLocked`): each new tile takes its parent's flag and holder, and its share of the parent's shields
  (`Carryover.Share`: split evenly in id order, the remainder to the first children, 10 → 3, 3, 2, 2); each mine moves
  to its tile's child nearest it (`Carryover.Heir`), and one on a tile with no child is lost; live viruses end; the
  flags out stay. Then **one transaction** (`Persistence.Carry`): the use case's keep starts the world's era on the new
  board (`eras.Store.Start`), the map's tiles, locks, viruses and mines are deleted, the carried tiles are `COPY`ed and
  the mines written. After the commit memory plays the new board's ground, filled from the carried rows as the boot
  fills it from postgres (`fillFromLocked`, so each province is settled again: fortify locks are dropped and the new
  board's taken), and **one `board_switched` event** is appended to the ledger on the new stage while no write runs. A
  new epoch, and every stream cut: every page reloads onto the new board.
- **The event is the one thing outside the transaction**: the ledger is kept in memory and flushed within a second, as
  every event is; the era is the switch's record. Its kind (`ledger.Switching`, migration
  `20261118113619_ledger_board_switched`) is **no take**: no `Taking`, `Strike` or `Scene`, so no `TilesTaken`, no XP, no
  mission's progress, no stat, and no replay draws it. The relay tells it as **`planet.v1.BoardSwitched`** (the map, both
  boards and their tile counts, the time and the position), which chat announces in the World and alerts pings Discord
  with ([Chat](chat.md#announcements), [Discord pings](alerts.md#the-pings-pings)).
- **The dry run** (`Storage.Carried`, under the tiles read lock) answers the same counts and changes nothing:
  `clicks.Carried`, the response's `tiles`, `carried`, `blank`, `held_before`, `held_after`, `dropped`, `dropped_held`,
  `shields_before`, `shields_after`, `mines_moved`, `mines_lost`, `viruses_ended` and `locks_dropped`.
- **A log line times the locks** (`carried the map onto its new board under the locks`: the flush, the carry, the
  transaction, the fill). Rehearsed on production's map: 1.9 to 6.1 s, the transaction most of it.
- **The process holds both grounds from the switch on**, `earth-1`'s for the events before it (the relay, the history);
  a boot after it loads `earth-2`'s alone, until a reader asks for `earth-1`'s.

## The wire

Every answer that names tiles names its board, by name (`eras.Boards.Name`): `GetMapResponse.board` (read with the
batch, `DenseBatch.Board`), the `heartbeat` a planet stream opens with (`Heartbeat.board`, read once subscribed: a swap
cuts every stream, so the stage read after the subscription is every change's), `MapHistory.eras` and
`GetReplayResponse.eras` (`BoardEra {started_at, board}`, from `eras.Store.Covering`: the era in force at the window's
start, then each that started in it; the first frame of each era is whole, a `WholeMap` of two bytes a tile since
2026-11-18, see [The map's last 24 hours](recent-map-history.md)), `GetLedgerResponse.board`, and the vandal squad's `Graffiti.board`. Since
2026-11-18 also: `TileHits.era` (the era of each tile's hits), `TilesTaken.board` and `Fortified.board` (the relay names
each fact's board, empty for one this process does not carry), `GetTerritoriesResponse.board` and
`GetCountrySizesResponse.board` (the board in play's counts, which the missions scale by), `GetTilesResponse.board`
(which the vandal squad checks against the ledger's), and the rule set's `ArenaRules.board`.

## Tests

PR 6's: `clicks`' `carryover_test.go` (parents, children in id order, the split, the heir); `embedded_board`'s
`carry_test.go` (the file read and fitted to both boards, none back or to a board not carried, a file of boards
regenerated since refused); the tile map's `switch_test.go` (the carry in memory and stored, the shares, holders, the
mine's heir, the viruses ended, the locks settled again, the new epoch and every stream cut, keep refusing, the count
changing nothing, refused onto the board it plays, while an arena plays, a second time, and with another board's
carry-over); `postgres_tile_store`'s `carry_test.go` (the rows moved in the era's transaction, and rolled back with it);
`switch_board_usecase`'s tests (the era, the event, the refusals, dry or not); the relay's `BoardSwitched`;
`changes_query`'s every kind.


PR 3's: `embedded_board`'s `registry_test.go` (a second board loaded the first time it is needed and kept, a board not
carried refused, every board carried needing planet's row under its hash, a file of another board refused, a lattice
this build cannot walk refused at boot); `cpboards`' tests (the boards shipped and the fixture, the scale); the tile
map's arena on a board of its own (its ground played, its size, the world kept on its board, the world back) and a
blank on a board not carried; `open_arena_usecase` (an arena on its set's board, a board planet keeps no row of
refused) and `close_arena_usecase` (the world's era resumes); `arenas`' rules (`arena.board`, none in default, a board
not carried refused); `ledger`'s history across two eras and `history_query`'s on postgres; the relay naming each
fact's board.

Besides the migration's: `eras`' `eras_test.go` (a boot on the era's board, one refused on a board not carried or with
no era, a board carried with no row or another hash) and its contract suite on memory and postgres;
`inmemory_tile_storage`'s `persistence_test.go` (a load refused on rows of another board, a flush naming the stage's
board) and `arena_test.go` (the gate); `postgres_tile_store`'s `maps_test.go` (every row naming its stage's board, the
boards of a map, a board planet does not keep refused); `open_arena_usecase` (the arena's era); `embedded_board`'s
`TestEachBoardCarriedIsNamedByItsDirectoryAndItsPositionsHash`; `ledger`'s tests of each reader by stage (an arena's
events and another board's left out) and of each recording naming its write's stage; the postgres store reading each
event's map and board back; each query's postgres suite with an arena's and another board's events; the relay's tiles
of a board not carried; the stream's opening heartbeat; `GetMap`'s board. A suite that purges planet's schema seeds
every board the migrations ship (`eras.Shipped`: `Earth1`, `Earth2`, `Siege1`) and the world's era again
(`eras.Reseed`); `MemoryStore` starts with the same rows; `ledger.Staged` writes a test's event on another stage, such
as `earth-2`'s. The fixture's row is `eras.Fixture`, under an id no migration takes (99).
