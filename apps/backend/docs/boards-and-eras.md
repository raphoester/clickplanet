# Boards and eras

**A tile id means a place only on its board, so every row and every event of a tile names its board, and every event
its map** (PR 2 of [Boards](../../../docs/boards.md), 2026-11-17). Only `earth-1` exists, so every value is `earth-1`:
what is built is the model and the guard, ready for more boards.

## The words in the code

- **`clicks.BoardID`** is a board's row in `planet.boards`, a `smallint`: what each row and event names its board by.
  **`clicks.BoardName`** is its name everywhere else: its directory in `/map/boards`, and the wire (`"earth-1"`).
- **`clicks.Stage`** is a map on the board it plays (`{Map, Board}`; `clicks.World.On(board)`): where a write lands,
  and what each event of the ledger names.
- **`eras`** (`internal/planet/internal/eras/`) keeps the boards planet knows and each map's eras: `eras.Board` (id,
  name, hash), `eras.Era` (map, board, start), `eras.Store` (`Boards`, `Latest`, `Covering`, `Start`) and
  `postgres_era_store`, with `MemoryStore` and `StoreContractSuite` behind the `testing` tag. `eras.Boards` are the
  boards this process carries, each matched with planet's row of it, and it names a board by its row (`Name`).

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
- **A `board smallint NOT NULL` column**, a foreign key to `boards`, on `tiles`, `landmasses`, `mines` and `viruses`, and
  **`map text` and `board smallint`** on `ledger_events`. Each was added with a constant default, a change of the catalog
  alone, and the default dropped right after, so every write must name it. The ledger's foreign key and its `map`
  check are `NOT VALID`: they hold for every row written since, and the rows before hold the defaults, so no scan of
  millions of rows runs under the table's lock at boot.
- **The backfill** names `earth-1` everywhere, and moves each past arena's events to its map: an event after the arena's
  open and before the second it closed (the old `arenas.Stretch.Holds`), or still open. It is the last read of those
  times: one scan of the ledger, and an update of the arenas' events alone.
- The vandal squad's `graffiti` got `board text NOT NULL` (migration `20261117133227_graffiti_board`): see
  [The vandal squad](vandalsquad.md).

`internal/planet/internal/migrations/boards_and_eras_test.go` runs it on a real postgres from the version before it:
`earth-1` everywhere, each arena's events on its map (the second of an open and of a close included), the defaults gone,
an unknown board refused, and the way back and up again. It is the one test of a migration's rows (see
[Testing](testing.md)): a wrong window would misfile history for good.

## Writes

- **The tile map plays one stage** (`inmemory_tile_storage.Storage.stage`, read by `Stage()`). `Load` takes it, and
  every flush writes it on each tile, lock, mine and virus (`Persistence.Save`, `SaveMines`, `SaveViruses` and `Open`
  take a `clicks.Stage`). `Blank` plays an arena on the board the map in play plays, and hands that stage to `keep`, so
  `open_arena_usecase` starts the arena's era in the open's transaction (`eras.Store.Start`). `PutBack` plays the world
  again on the board the arena played.
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

1. **The board this process carries** (`embedded_board.Loader.Carried`): the name of its directory and its manifest's
   `positionsHash`, which the loader has checked the rebuilt positions against.
2. **Matched with planet's row** (`eras.BoardsOf`): a board carried that `boards` has no row of, or names under another
   hash, refuses the boot (`eras.ErrUnkept`): a board regenerated under its name, its tiles moved, is another board.
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
| `GetReplay` (`replay_usecase`, `ledger.Footage`), the clips' replay and cast | the world's map on the board it plays (`eras.Store.Latest`) |
| `GetMapHistory` (`history_query`, `ledger.History`), the season replay | the same |
| `GetRecentMapHistory` (`recent_history_query`, `ledger.Rewind`) | the same |
| `GetLedger` (`changes_query`), the backoffice Home, the clips' cast, the vandal squad | the same, in SQL (`map = $5 AND board = $6`) |
| `GetTileHolder` (`holder_query`), `GetLandmass`'s last fortify (`landmass_query`) | the stage that plays (`Storage.Stage`) |
| `RevertPlayer` (`ledger.Runs`) | the stage that plays, refused unless it is the world's |
| `FindPlayers`, `TopPlayers` (`ledger.OnBoard`) | the board that plays, on any map |
| the relay (`relay_ledger_usecase`) | every event, with each tile's ground and point from its event's board (`Boards.Of`; none for a board not carried) |

Whether an arena plays, for the readers that then rewind from the world's map kept, is the stage that plays too
(`Stage().Map != clicks.World`). `arenas.Stretch` stays for `GetReplay`'s `asides`, which the clip generator reads.

**With one board, each reader draws one board.** A window across a change of board (PR 6) would draw the board the
world plays now; PR 3 draws each era on its own, with the cut.

## The wire

Every answer that names tiles names its board, by name (`eras.Boards.Name`): `GetMapResponse.board` (read with the
batch, `DenseBatch.Board`), the `heartbeat` a planet stream opens with (`Heartbeat.board`, read once subscribed: a swap
cuts every stream, so the stage read after the subscription is every change's), `MapHistory.eras` and
`GetReplayResponse.eras` (`BoardEra {started_at, board}`, from `eras.Store.Covering`: the era in force at the window's
start, then each that started in it), `GetLedgerResponse.board`, and the vandal squad's `Graffiti.board`.

## Tests

Besides the migration's: `eras`' `eras_test.go` (a boot on the era's board, one refused on a board not carried or with
no era, a board carried with no row or another hash) and its contract suite on memory and postgres;
`inmemory_tile_storage`'s `persistence_test.go` (a load refused on rows of another board, a flush naming the stage's
board) and `arena_test.go` (the gate); `postgres_tile_store`'s `maps_test.go` (every row naming its stage's board, the
boards of a map, a board planet does not keep refused); `open_arena_usecase` (the arena's era); `embedded_board`'s
`TestTheBoardCarriedIsNamedByItsDirectoryAndItsPositionsHash`; `ledger`'s tests of each reader by stage (an arena's
events and another board's left out) and of each recording naming its write's stage; the postgres store reading each
event's map and board back; each query's postgres suite with an arena's and another board's events; the relay's tiles
of a board not carried; the stream's opening heartbeat; `GetMap`'s board. A suite that purges planet's schema seeds
`earth-1` and the world's era again (`eras.Reseed`); `ledger.Staged` writes a test's event on another stage.
