# Boards: a map's geometry as files of its own

**Designed 2026-11-17. PR 1 is built (2026-11-17): `earth-1` is a board, and every app reads its geometry from it. PR 2
is built (2026-11-17): every tile and event names its board and map, and the boot refuses a map on a board it does not
carry. PR 3 is built (2026-11-18): every app holds more than one board, the page draws the board an answer names and
reloads when it changes, every threshold sized from the board reads it, an arena plays the board its set names, and a
history draws each era on its own board, with the cut. PR 4 is built (2026-11-18): `earth-2` and `siege-1` are
generated and committed, warped and rotated and in canonical order, and every app reads them to the bit. PR 5 is built
(2026-11-18): `charges.spread.off`, and Capital Siege's set ready to name `siege-1` with no spread, which the owner
saves.**
The rest is the reference the later PRs at the end of this doc build from. Each PR turns the part of this doc it
builds into "how it works", and updates the docs it makes stale: [`map/README.md`](../map/README.md),
[Map geometry](map-geometry.md), and the apps' tile docs (`tile-positions.md`, `tile-layout.md`, `zoomed-out-view.md`,
`map-geography.md`, `arenas.md`).

## Why (the owner, 2026-11-17)

- **The world is cramped.** 260,503 tiles, and players freeze their borders and get angry when attacked. The world
  goes to about 1,000,000 tiles. The click economy does not change, so the same ground costs about 3.9× the clicks.
- **The grid is uneven.** The tiles are a flat icosahedron's grid pushed onto the sphere (gnomonic). A tile in the
  middle of a face covers about twice the ground of a tile at a corner.
  - A bomb of 4 rings takes 48 to 89 tiles depending on where it lands, and 91 on a pentagon, against a click bank of 60.
  - Three pentagons are on land: Tibet (31.7°N 90°E), Mississippi (31.7°N 90°W) and Uganda (0°, 31.7°E).
  - The virus's pods overlap near the pentagons, where tiles are closest.
- **The geometry has changed twice already.** Each time — the remaps of 2026-09-21 and 2026-11-03, seats and lakes —
  it rewrote the ledger onto the new ids, silently, so a replay from before drew its takes on ground they were not made
  on. It will change again: more players, or regions inside the big countries.
- **Events want boards of their own.** Capital Siege on a coarse board, the United States alone, New York, the Moon.

So a geometry becomes a set of files of its own, a **board**, kept and served forever. Everything that names a tile
also says which board the tile is on.

## Words

- **Board**: a geometry. A set of immutable files, named (`earth-1`) and content-addressed.
- **Map**: what is played on a board: the world, or an arena (`tiles.map` today). A map plays one board at a time.
- **Era**: a stretch of a map on one board, from a moment on. A switch of board starts one.
- **Run**: a stretch of tiles side by side in one row of the lattice, written as a range of lattice ids: "face 7, row
  1042, columns 33 to 96". The ocean costs nothing, a coast a few runs a row, and a New York board a few thousand runs.
- **Carry-over**: a file between two boards that names, for each tile of the new board, the tile of the old board it
  comes from. A switch reads it once.

| board | tiles | apart | plays |
| --- | --- | --- | --- |
| `earth-1` | 260,503 (`cols` 301) | 25.9 km | the world until the switch, and all history before it |
| `earth-2` | 1,004,682 (`cols` 590) | 13.1 km | the world after the switch |
| `siege-1` | 27,259 (`cols` 97) | 79.4 km | Capital Siege's arena |

A board's tile count is fixed by its `cols` and the land the generator finds: 28.9% of `earth-2`'s lattice and 29.0% of
`siege-1`'s, against 28.75% of `earth-1`'s. `siege-1`'s `cols` is the one whose count is nearest 27,000 (the owner,
2026-11-17: 15 players at a click a second fill it in about 30 minutes with no spread). The apart column is the median
arc between two touching tiles, the manifest's `spacing`.

## The geometry

**Built in PR 4** (2026-11-18): `lattice.ts` in the page (`positionWriter`, `warpedWeights`, `unwarp`, `walkCanonical`),
its copy in the backoffice, `lattice.go` in the backend, and the generator's `scripts/map/warp.mjs`; the measurements
below are `npm run map:measure -- <board>`'s.

**Every board is an icosahedral lattice.** Each of the 20 faces is a triangular grid of `cols` steps per edge, so
`cols + 1` points along it and `10·cols² + 2` on the sphere (906,012 for `earth-1`'s 301). A point is `(face, i, j)`,
and its neighbours are the ±1 exchanges of its barycentric coordinates: 6 everywhere, 5 at the 12 corners. Built:
`lattice.ts` walks it, `latticeNeighbours` takes its neighbours, and the backend's `embedded_board` does the same in
Go.

**The 12 five-neighbour points cannot go.** On a sphere, a grid where nearly every point has 6 neighbours has exactly
12 with 5: Σ(6 − degree) = 12 (Euler). So they go to sea. One flat grid per continent would avoid them but stretches
worse. Afro-Eurasia is about 150° across, twice a face, so the best flat grid over it puts tiles up to 1.57× apart, or
squashes them 1.57× in one direction.

**The position of `(face, i, j)`** on a face with corners A, B, C:

1. The barycentric `w = (wa, wb, wc)` of `(i, j)`, as `lattice.ts` computes it today.
2. **The warp** (new boards only):
   - `h_k = w_k · (1 + a·w_k + b·w_k²) + c·wa·wb·wc`
   - `w'_k = h_k / (ha + hb + hc)`
   - with each board's own `a`, `b`, `c` in its manifest (below).
3. `p = w'a·A + w'b·B + w'c·C`, then normalised as today (`× 1 / length`).
4. A, B, C are three's 12 unnormalised icosahedron corners, **rotated once** (new boards only), the 36 float64s of the
   manifest's `corners`.

**The order of operations**, the same in every reader: `w = ((cols − i − j) / cols, j / cols, i / cols)`;
`h_k = w_k · ((1 + a·w_k) + b·(w_k·w_k)) + ((c·wa)·wb)·wc`; `w'_k = h_k / ((ha + hb) + hc)`;
`x = (w'a·Ax + w'b·Bx) + w'c·Cx`, and so for y and z; `× 1 / √((x·x + y·y) + z·z)`, then to float32. A point on an edge
or a corner comes out the same bits from every face it is on: its third weight is 0, which adds nothing and is added in
any order, so the canonical walk may reach it from any face.

**Only `+ − × ÷ √`**, so the rule of [Tile positions](../apps/frontend/docs/tile-positions.md) holds: every engine
rebuilds every position bit for bit. A lattice with no warp and no rotation is `earth-1`'s, built with three's own lerps
as before. In Go each product is converted to `float64` on its own, or the compiler fuses it into the add (arm64 does,
and the hash moves): an arm64 build of the package has no fused instruction, and as an arm64 binary under qemu it
rebuilds every tile of all three boards to their manifests' hashes (see the backend's
[Map geography](../apps/backend/docs/map-geography.md)).

**The fit, per board.** The design's coefficients were fitted at `cols` 24 (`a = −0.05520744`, `b = −0.16849639`,
`c = 0.30479447`). Each board's are fitted again at its own `cols` (`fitWarp` in `scripts/map/warp.mjs`): Nelder-Mead
from those, on the cells' areas, largest over smallest (a cell is a third of each of its triangles, the twelve
pentagons left out), rounded to 8 decimals. `warp.test.mjs` pins each: the spread at the manifest's coefficients is
below a nudge of 0.002 to any of them (a fit at 590 takes 3 minutes, too long for a test), and a fit at `cols` 24 run
twice gives the same numbers.

| board | `a` | `b` | `c` | spread at its fit | at the design's |
| --- | --- | --- | --- | --- | --- |
| `earth-2` (590) | −0.09142428 | −0.13039461 | 0.26096331 | 1.00088 | 1.00539 |
| `siege-1` (97) | −0.08982266 | −0.13216128 | 0.26291902 | 1.00129 | 1.00460 |

No lattice triangle folds (`lattice.test.ts`, at `cols` 97 and 400).

**Measured** (`npm run map:measure`, 2026-11-18). A bomb is every point within 4 × the mean arc between touching tiles
of the point hit (the backend's `Geography.Spacing`, which its radius reads), counted on the whole lattice so the sea
cuts no circle. A crease is how far a row of cells turns at a point: 180° less the angle between a neighbour and the one
opposite it, on land.

| | `earth-1` (gnomonic) | `earth-2` | `siege-1` |
| --- | --- | --- | --- |
| tile area, largest ÷ smallest | 1.98 | 1.001 | 1.001 |
| closest two tiles ÷ median spacing, on land | 0.733 (19.0 km) | 0.900 (11.7 km) | 0.899 (71.4 km) |
| tiles a 4-ring bomb takes, wherever it lands | 55 to 95 | 57 to 63 | 57 to 63 |
| … on a pentagon | 91 | 61 | 61 |
| the creases: a row's turn on a face edge, median / 99% / most | 9.7° / 17.8° / 17.9° | 6.8° / 16.4° / 17.0° | 6.9° / 16.6° / 17.0° |
| … inside a face, most | 0° | 0.09° | 0.55° |

Snyder's equal-area projection was measured too, at design: area ratio 1.03, closest two tiles 0.85. It needs sin and
cos, which break the bit-for-bit rebuild.

**The territory shader inverts the warp, and so does the pick.** The gnomonic inverse, as today, gives `w'`. Two Newton
steps starting from `w = w'` (`unwarp`) reach 5e-8 on every point of `earth-2`'s faces, against 1.7e-3 for one cell at
`cols` 590 (`lattice.test.ts`); one step reaches 3.4e-4. The shader's `unwarped` does the same in float32. **A tile its
cell's inverse misses refuses the globe**: `cellsOf` runs every tile's position through `cellOf` as the page loads and
throws on one found in another cell, about 200 ms on `earth-2` in the worker (see the page's
[Tile positions](../apps/frontend/docs/tile-positions.md)); `latticeCells.test.ts` runs it on every tile of every
shipped board, and on a warped board read without its inverse, which it refuses.

**The rotation puts the 12 corners at sea**, each at least 184 km from an `earth-1` tile, 181 km from an `earth-2` tile
and 189 km from a `siege-1` tile. It was found by a random
search over rotations, then refined, maximising the smallest distance to land. A corner `c` goes to `R·c`:

```
R = [[0.09926508736568841, -0.49847683751896155, -0.8612010711136958],
     [0.5914752627857613,   0.725540318816103,   -0.35177870783312387],
     [0.8001896374877626,  -0.47445978568190805,  0.36685754159875256]]
```

| lat, lon | nearest land | | lat, lon | nearest land |
| --- | --- | --- | --- | --- |
| 17.83, −59.98 (east of the Leewards) | 209 km | | 43.48, 137.76 (Sea of Japan) | 184 km |
| 68.15, 2.63 (Norwegian Sea) | 444 km | | 18.55, 67.14 (Arabian Sea) | 427 km |
| −68.15, −177.37 (Ross Sea) | 611 km | | −18.55, −112.86 (south-east Pacific) | 1,932 km |
| −17.83, 120.02 (off north-west Australia) | 210 km | | −43.48, −42.24 (South Atlantic) | 1,246 km |
| −42.90, 50.04 (south Indian Ocean) | 1,058 km | | 4.71, 3.60 (Gulf of Guinea) | 200 km |
| −4.71, −176.40 (central Pacific) | 645 km | | 42.90, −129.96 (off Oregon) | 449 km |

Latitude is `asin(y)` and longitude `atan2(z, −x)`, as everywhere in the map scripts. The generator computes the
corners once from `R` (`rotatedCorners` in `scripts/map/warp.mjs`), and every manifest carries them; no reader knows `R`.

Against each board's own coasts (`warp.test.mjs` holds them at sea, 180 km from `earth-2`'s land and 185 from
`siege-1`'s at least):

| lat, lon | `earth-1` | `earth-2` | `siege-1` |
| --- | --- | --- | --- |
| 17.83, −59.98 | 209 km | 194 km | 226 km |
| 68.15, 2.63 | 444 km | 431 km | 498 km |
| −68.15, −177.37 | 611 km | 595 km | 628 km |
| −17.83, 120.02 | 210 km | 203 km | 226 km |
| −42.90, 50.04 | 1,058 km | 1,066 km | 1,618 km |
| 4.71, 3.60 | 200 km | 190 km | 226 km |
| −4.71, −176.40 | 645 km | 567 km | 638 km |
| 42.90, −129.96 | 449 km | 448 km | 450 km |
| 43.48, 137.76 | 184 km | 181 km | 189 km |
| 18.55, 67.14 | 427 km | 423 km | 448 km |
| −18.55, −112.86 | 1,932 km | 1,019 km (Easter Island, which `earth-1` misses) | 1,944 km |
| −43.48, −42.24 | 1,246 km | 1,214 km | 1,240 km |

**What it leaves: creases.** Where hex rows cross one of the 30 face edges they turn: by 6.8° at the median on land,
up to 17° next to a corner, against `earth-1`'s 9.7° and 17.9° (measured above). They stay for now (the owner,
2026-11-17): look at `earth-2` before deciding; the screenshots are below ([Screenshots](#screenshots)). A relaxation
table could smooth them later and keep the bit-for-bit rebuild.

**Tile order.** A lattice point's canonical id is `canonicalId`: the corners, then the 30 edges, then each face's
inside, row by row. A board's runs are ranges of canonical ids whatever its order.
- **New boards** number their tiles in canonical id order. Built: a tile's id is its rank among the board's tiles in
  canonical id order, plus one (`walkCanonical` in each reader: the 12 corners, then the 30 edges step by step from
  their lower corner, then each face's inside, row by row).
- **`earth-1` keeps three's walk order**, because its ids are the ones the ledger holds. Built: a tile's id is its rank
  among the tiles in `walkLattice`'s order, plus one, and the regenerated `earth-1` has the same 260,503 tiles in the
  same order.
- Above `cols` of about 20,000 a canonical id passes 2³². The file stores them in 64 bits; tile ids, which are ranks,
  stay 32-bit.

## The files of a board

Built in PR 1; the formats are in [`map/README.md`](../map/README.md#format). Each board lives in `map/boards/<name>/`,
content-addressed like every file in `/map`:

- **`board-<hash>.json`**, the manifest:
  - `name`, and `body` (`earth`);
  - the lattice: `cols`, `warp` (`null` or `{a, b, c}`), `corners` (`null`, or the 36 rotated float64s) and `order`
    (`three` or `canonical`). Every reader reads both kinds since PR 4, and refuses an order it does not know;
  - `tiles`, the median `spacing` in radians (the arc between two touching tiles: 0.00406 for `earth-1`, 25.9 km), and
    `positionsHash`, FNV-1a over the float32 positions in tile order, a uint32;
  - the names of its other files.
- **`runs-<hash>.bin`**: `"CPRN" | uint32 version | uint32 count | count × (uint64 first, uint64 last)` canonical ids,
  ascending. It replaced the landmask: 7,556 runs, 121 KB for `earth-1`; 17,628 runs, 282 KB for `earth-2`; 1,855 runs,
  30 KB for `siege-1`.
- **`borders`, `capitals`, `bridges`**, in their formats, named by the board's tile ids. `earth-1`'s are the files of
  before, byte for byte.
- **The frontend's own**: the border lines and the landmass names, per board since PR 3: `static/boards/<name>/` beside
  the board's copy, content-addressed, named in the page's board table (`boardsAsset.ts`), with the landmass names'
  translations in `src/i18n/landmasses/<name>/`. The terrain is a file of the board, since it is cut from its runs.

**The generator** (`npm run map:generate -- <board>`, then `npm run bridges -- <board>` and the rest of the chain in
[`map/README.md`](../map/README.md)) writes a board without three, each board as `scripts/map/board.mjs` defines it: it walks the lattice,
takes neighbours and triangles from it, and reads latitude as `asin(y)` and longitude as `atan2(z, −x)` from each point.
Regenerating `earth-1` writes its runs, borders, capitals, bridges and manifest byte for byte, and the border lines
and landmass names too.

**No coordinates file, on any board.** Every app rebuilds positions from the lattice and the runs:
- the frontend, in its worker, or on the page when the worker fails;
- the backoffice, with its own copy of the lattice module (it imports nothing from the frontend);
- the scripts, with the frontend's module;
- the backend, in Go, to the bit, and it refuses the boot on positions that do not hash to the manifest's. It needs
  positions only for `Within`, `Nearest` and `Spacing`, so a tolerance would have done, but the exact rebuild costs
  nothing and checks the board. It takes neighbours straight from the lattice, with no position matching, and loads
  `earth-1` in about 0.1 s.

**A hash that does not match is logged, not fatal**, on a page: it logs the mismatch with its user agent and keeps the
positions it computed; the ids come from the runs, so an ulp of difference moves a tile by under a metre.

**Copies:** each app carries every board, committed, through `npm run map` and `make map`: the frontend every file in
`static/boards/<name>/` and each manifest in a table by name, the backend every file embedded, the backoffice each
manifest and its runs. The frontend serves every board's files forever. A board costs the backend about 57 bytes a
tile (47 MB kept for `earth-2`, measured), loaded when it is first needed (`embedded_board.Registry`, PR 3).

**A fixture for the tests** (PR 3): `fixture-1`, a coarse earth cut from `earth-1` at `cols` 31 (2,998 tiles, 9.7 times
as far apart, another positions hash), which the page's `npm run map:fixture` writes into each app's test data alone:
the backend's `testing` tag, the page's and the backoffice's `src/testdata`. It is never in `/map/boards`, `generated/`
or `static/`, and planet keeps a row of it only in a test's database.

## Eras and the board column

**Built in PR 2 (2026-11-17)**; the backend's side is [Boards and eras](../apps/backend/docs/boards-and-eras.md).

- **`boards`** `(id smallint, name text unique, hash text unique)` in planet's schema: a board's row, its name (the
  directory in `/map/boards`) and its manifest's `positionsHash` in 8 hex (`3fb28bdf` for `earth-1`): what its tile ids
  mean, which a terrain or borders drawn again leaves alone while it rewrites the manifest. A board's row ships with the
  migration that ships the board.
- **`eras`** `(map text, board smallint, started_at timestamptz)`: a map's latest era is the board it plays. The world's
  first era started at the epoch, on `earth-1`; an arena's map gets its era as it opens, on the board its set names
  (`arena.board`, PR 3) or the world's.
- **A `board smallint NOT NULL` column**, a foreign key to `boards`, on `tiles`, `landmasses`, `mines`, `viruses` and
  `ledger_events`, and **a `map` column on `ledger_events`**. Every write names the board of the map that plays, and
  every event its map: an arena's events are told apart by that column, no longer by the arena's times
  (`arenas.Played`, gone).
- **The vandal squad's `graffiti` keeps its board by name** (`board text`), not as planet's row id: its schema cannot
  reference planet's `boards`, and the board reaches it by name, on planet's `GetLedger`. A drawing is seen again only on
  its own board.
- **The backfill wrote `earth-1` everywhere**, which is true: both past remaps rewrote all history onto today's ids, and
  arenas played on the world's board. It moved each past arena's events to its map, from its window in `planet.arenas`,
  the last time those times were read.
- **The boot refuses** a map whose era plays a board the process does not carry (by name and hash), and a map whose
  tile, lock, mine or virus rows name a board other than its era's: a skipped remap would boot old ids onto new ground.

**Every answer that names tiles names their board, by name:**
- `GetMap` (`board`), and `GetReplay`'s opening;
- the stream as it opens (the first `heartbeat`'s `board`; a change of board cuts every stream, as a new epoch);
- `GetMapHistory`, `GetRecentMapHistory`, `GetReplay` and the season replay: `eras`, each era of the world the window
  covers with its board;
- `GetLedger` (`board`: the world's changes on that board alone) and the vandal squad's `ListGraffiti` (each graffiti's
  `board`).

**A page that reads a board other than the one it drew reloads** (the owner: a reload, not a rebuild in place). Built in
PR 3: the page builds its globe from the board `GetMap` names, and reloads when `GetMap` or the stream's opening
heartbeat names another. The backoffice has no globe, and reads again on the board an answer names.

**Every reader of the ledger resolves each event's board and map from its columns:** replays and clips, the archive,
the last 24 hours and `GetLedger` keep the world's map on the board it plays; the holder and landmass cards the map that
plays on its board; the relay takes each tile's `Ground` and `Position` from its event's board. FindPlayers, TopPlayers
and RevertPlayer read the current board's events only, and so does the vandal squad, through `GetLedger`.

**Each era on its own board, with the cut** (PR 3): `GetMapHistory` and the season replay keep each world event on the
board of the era in force at its time; the first frame of an era holds every tile held on its board, as the window's
first frame does, and each tile's hits name their era (`TileHits.era`). The page's `/archive` and the player's video
draw each era on its board, with a visible cut. The rewinders (`GetReplay`, `GetRecentMapHistory`) keep the board the
world plays now: no rewind crosses into another board, and PR 6 starts them at the switch. So a clip plays one board,
its replay's: `GetReplay` has one opening, and a clip whose history crosses a cut opens on the cut and tells the last
era alone, its sagas measured on it (no visible cut in a clip yet).

## Arena boards

**`arena.board`** is a rule set's knob: the board an arena's map plays on (default: the world's). **Built in PR 3**
(2026-11-18): `default` names none, a set refuses a board its build does not carry, and an open on a board planet keeps
no row of is refused; every rule reads the arena's board while it plays, and its seed is laid on it. See the backend's
[Arena battles](../apps/backend/docs/arenas.md#the-board-an-arena-plays-arenaboard).

**Capital Siege plays `siege-1`, with no spread at all**: `charges.spread` is off in its set, and none comes in its
boxes (the owner, 2026-11-17). **Built in PR 5** (2026-11-18): `charges.spread.off`, a knob of its own on bombs off's
pattern (the backend's [Spreads off](../apps/backend/docs/spread.md#spreads-off-chargesspreadoff)), and the set's exact
JSON in the backend's [Capital Siege's set](../apps/backend/docs/arenas.md#capital-sieges-set), which the owner saves. On 260,503 tiles its unlimited spread was inflation on purpose, to fill the map fast,
and its animation spammed the screen. With no spread, 15 players at a click a second fill `siege-1` in about 30
minutes; France is about 100 tiles.

**The arena's era opens with it, and the world's resumes at its close.** Pages reload during the 8-second freeze.

## The world's switch to `earth-2`

**Two steps.** First a deploy: the code knows `earth-1` and `earth-2`, and the world still plays `earth-1`. Then the
operator calls `AdminService/SwitchBoard{map, board, dry_run}`, an MCP tool, at a quiet hour.

**When:** mid-season, as soon as the rehearsal is clean, and not within 48 hours of the Final Battle. Season 1's
finale starts 2026-11-28 17:00 UTC, so the switch is before 2026-11-26 17:00 UTC.

**The carry-over file** (`carry-earth-1-earth-2-<hash>.bin`) gives each new tile a parent: the nearest old tile, when
it is within about 0.6 old spacings. Otherwise the tile has none, so new coasts and islands start blank. The generator
writes it with the board.

**The switch runs in one transaction, under the flush lock and the tiles lock:**

- each new tile takes its parent's owner and holder;
- **each old tile's shields are split across its children, keeping the total** (10 → 3, 3, 2, 2, the remainder going
  to the first children in id order), so the switch creates no protection;
- each mine moves to the child nearest its old tile;
- live viruses end;
- fortify locks are dropped and settled again;
- a new era, and one `board_switched` event in the ledger. **It is not a take**: no `TilesTaken`, no XP, no mission,
  no stat.

**Then:**
- a new epoch: every stream is cut, and every page reloads;
- a full-screen moment ("The world grew: 4× more ground"), a line in the World chat, and a Discord `@everyone`, as
  for a season start.

**The dry run** answers the counts (tiles carried, tiles blank, shields, mines moved, viruses ended) and changes
nothing.

**Scoring that day is as usual**: rankings compare countries, and every count jumps about 3.9× at the same moment.

**History across the switch** (the owner, 2026-11-17):
- The season-1 archive plays through it, with a visible cut and a caption.
- For their first 24 hours after it, player videos, clips and the last 24 hours start at the switch, as after a reset.
- RevertPlayer and the operator tools act only on the current board's events.

**The rehearsal** uses production's public `GetMap` (owners and shields) on a local postgres, with made-up holders and
ledger: the switch, the boot, and the measurements.

## Rules at the new density

**A bonus never pays more than a full click bank** (the owner, 2026-11-17; past bonuses inflated the economy). So
whatever pays keeps its tile count, and only what describes the map scales with the density.

**What stays:**
- `bomb.rings` 4: about 60 tiles. The radius is rings × the board's measured spacing, so it follows the board on its
  own.
- `mine.rings` 2, `spread.rings` 1, `enclose.maxTiles` 25, `virus.infections` 128, `seed.capitals.shields` 100, and
  `urf`'s bomb at 8 rings and enclose at 35.
- XP per tile taken or fortified, the titles, and the mission goals.
- The antibot's thresholds, because they count clicks.
- The toll, because it counts shares of the map.

**What changes with `earth-2`:**

| what | where | `earth-1` | `earth-2` |
| --- | --- | --- | --- |
| `rally.rings` | rule sets, edited by the operator at the switch | 18 | 36, the same ground (a rally pays nothing itself) |
| mission floors | `missions/draw.go` (`rivalAtLeast`, `groundAtLeast`, `openAtLeast`, `raidAtLeast`) | 1,000 / 3,000 / 1,000 / 300 | 3,900 / 11,700 / 3,900 / 1,170 |
| fortify news | `chat/…/announcement.go` `fortifyNewsTiles`, and the frontend's `newsFromTiles` | 50 | 200 |
| largest drawing | vandal squad `reading.go` `maxTiles` (12 at least stays) | 150 | ×3.9, about 585 (a count of tiles: until PR 3 this row said 300) |
| clip thresholds | frontend `clip/look.ts`, `clip/changes.ts`, `clip/findings.ts` | 20, 100, 400, 1,000 | ×3.9 |
| attack ping zoom | alerts `pings/view.go` `closestZoom` | 20 | 40 |
| heatmap cell | frontend `archive/heatmap.ts`, backoffice `home/heat.ts` | 0.4°, 0.3° | 0.2°, 0.15° |
| max zoom | frontend `zoom.ts` | 50 | 100 |

**The thresholds read the board, not constants**, so they hold on `siege-1` too. **Built in PR 3** (2026-11-18), with one
rule in every app: a board's length ratio is its spacing over `earth-1`'s (0.004063093462255102), its density ratio
that ratio's square inverted; a count of tiles is `max(1, round(n × density))`, a length `x × length`, a zoom `z ÷
length`. On `earth-1` every value comes out as before, to the bit. See the backend's
[Sized from the board](../apps/backend/docs/board-scale.md).
- **Counts of tiles** scale with the board's density divided by `earth-1`'s: the mission floors, the fortify news, the
  largest drawing, the clip thresholds.
- **Lengths** scale with its spacing: the heatmap cell, the zoom, the attack ping's zoom.
  - **Max zoom** keeps a tile at max zoom as wide as on `earth-1`: 100 on `earth-2`, about 16 on `siege-1`.
  - **The frontend's hard-coded spacings** become the board's spacing: `REACH`, `TILE_SPACING`, `tilePointSize`,
    `tileSpacing`, `LEAST_FLAG_RADIUS`, `VIRUS_LIFT`, and the fake backend's constants.

## What `earth-2` costs

Built in PR 4 where it says so, measured 2026-11-18.

- **Frontend:**
  - **GPU tile data:** 5 → 21 MB in the compact layout, 36 MB in the plain one (58 before PR 0).
  - **`tileData`: 1,024 columns since PR 4.** At 512 `earth-2`'s 1,004,682 tiles would fill 1,963 of the 2,048 rows
    WebGL2 guarantees; at 1,024 they fill 982. **The terrain's texture went to 2,048 columns too**: a byte a lattice
    point, `earth-2`'s 3,481,002 points would need 3,400 rows at 1,024, and fill 1,700 at 2,048. `tileCells` (425 rows),
    the landmasses' rows (1,175 of 2,048) and the aimed outlines fit as they were (the page's
    [What a tile costs on the GPU](../apps/frontend/docs/tile-layout.md)).
  - **`GetMap`:** 0.5 → 2 MB raw, in batches of 10,000.
  - **The position rebuild:** 55 ms for the positions and 200 ms for the cells and their check in Node on 4 vCPUs,
    against 60 and 40 ms on `earth-1`; about 6 s in iOS Lockdown Mode at the design's estimate (not measured).
  - **The board's files**, served as they ship: runs 282 KB (47 KB packed with brotli), borders 2.0 MB, terrain
    197 KB, border lines 810 KB, landmass names 27 KB.
  - **The work done per tile on every frame** grew 4× until PR 0, which made it follow what is on screen: see
    [Slow devices](#slow-devices), and [PR 4, measured](#pr-4-measured).
- **The terrain:** a file of the board, 3,481,002 points in 197 KB. `npm run terrain` reads each tile's colour from the
  4,096 px photo, where a tile is 1.5 px. **Built:** the photo's coast was cut to earth-1's tiles, so a tile where
  earth-1 had sea reads the photo's blue; it takes the kind most of its land neighbours have, wave by wave from the
  coast in (10,264 tiles on `earth-2`), and an island earth-1 did not have at all, earth-1's nearest tile's kind (692)
  (the page's [The terrain](../apps/frontend/docs/terrain.md)).
- **Natural Earth stays at 1:50m** (the owner), pinned. **Seats are per board** (`SEATS_BY_BOARD` in `seats.mjs`):
  `earth-2` keeps 13 of `earth-1`'s 34 (20 have ground of their own at 12.5 km, and `io` too), gives Christmas Island
  Australia's ground (Natural Earth draws it inside Australia, and a vertex of `earth-2` falls on it), and seats the
  British Virgin Islands, whose vertices fell on `earth-1`'s lattice and not on `earth-2`'s: 15. `siege-1`, at 79 km,
  seats 52: `earth-1`'s 33 but `io`, and 19 countries smaller than its cell (Luxembourg, Malta, Singapore, Hong Kong,
  Jersey, the Windward Islands…), each at its capital. Two islands nearest one vertex take the next one beside it
  (`beside`: Guernsey beside Jersey, the British Virgin Islands beside the US ones). Every flag has ground on all three
  boards, and `npm run map:audit -- <board>` reads zero faults on each.
- **Sea lanes are per board too** (`LANES_BY_BOARD` in `bridges.mjs`): on `siege-1` the Rock is Gibraltar's one tile and
  Morocco's coast is nearer Tarifa than Spain's next, so the lane leaves from Málaga. A lane lands where a stone does, a
  capital when its land has no other coast (Puerto Rico is one tile on `siege-1`), and its point may lie as many of the
  board's spacings from its coast as 150 km is of `earth-1`'s (`snapKmOf`). `earth-1`'s bridges are unchanged.
  `earth-2` has 1,422 bridges over 975 pieces of land, `siege-1` 272 over 173.
- **The landmass names** come from the generator in English: 936 of `earth-2`'s 1,175 landmasses, 138 of `siege-1`'s
  374. **`earth-2`'s translations wait for `make translate`**, which needs the translator's key and was not run in PR 4:
  the page names them in English meanwhile, and its catalogue test lists the board as awaiting one (the backend's
  [The page's translator](../apps/backend/docs/translate.md)). `siege-1` is done by hand (PR 5).
- **The last 24 hours:** its first frame grows 4×, past its 1 MiB budget, so it needs a new format.
- **The backend** (measured on 4 vCPUs, `load_new_boards_test.go`):
  - **The boot of `earth-2`:** 0.8 to 0.9 s, against the design's 4 to 6 s and a one-minute deadline; 2,966,323 edges.
  - **Memory:** about 47 MB kept, and 302 MB allocated (about 260 MB of process memory) at the peak while it builds the
    edges. A board loads only when a map plays it.
  - **`Within` and `Nearest`:** tens of milliseconds per bomb or rally (the design's estimate, not measured).

## Slow devices

**The pixel work does not grow; the work per tile does** (the owner, 2026-11-17: "won't it be a hard hit on the slower
devices?"). At the same zoom a tile covers a quarter of the pixels, so the pixels shaded stay about the same. What grows
4× is what the page does once per tile on every frame, whether the tile shows or not. Before PR 0 that was three things:

- **The idle spin drew every tile under the painted flags.** Zoomed out, which is the landing view and the idle spin,
  the territory layer paints the flags, at a cost per pixel. The display points were drawn under it anyway, every frame
  at 60 fps.
- **The far side was not really skipped.** The vertex shader dropped a point behind the earth on one dot product, but
  it still ran for every point and read every attribute.
- **A hover redrew every tile.** `GpuPicker.pick` rendered all the points into a window around the pointer and read a
  pixel back, which made the CPU wait on the GPU, on every hover tick.

**Built in PR 0** (2026-11-17), on `earth-1`, before any board:

1. **No tile points where the painted flags cover the globe.** At `flagPaint` 1 a tile draws only its rings (the front's
   and the fortify pulse's), so only the pieces holding a ringed tile are drawn; with none, no tile is drawn. The
   landing view and the spin now cost the same at any tile count.
2. **Only the visible pieces are drawn.** The tiles are cut into chunks of consecutive ids, at most 0.1 rad from their
   first tile (a piece of a row), each with a bounding cap, and blocks of 32 chunks with a cap of their own. The CPU
   skips the chunks behind the earth (with the vertex shader's limb) or off screen, and draws the rest as at most 16
   ranges. It keeps them while the camera moves by less than a quarter of the screen's half-height, and does not cull
   at all while the screen shows more than half the globe, where the shader's own far-side drop is as cheap. The
   border-line passes are not chunked: they are instanced, three draws no range of instances, and they grow with the
   coastline, not with the tiles.
3. **Picking on the CPU.** The pixel's ray meets the tiles' sphere, and `cellOf` gives the lattice cell; anywhere in a
   tile's cell is that tile, at every zoom (a tile's disc is 0.76 of its cell, and the gap between two discs is the
   ground the territory layer paints there). The GPU picker, its scene, buffers, shaders and the plain layout's picking
   colour are gone.
4. **Less memory per tile:**
   - the plain layout is 36 bytes a tile, not 56, of 32-bit floats alone: the atlas rectangle packed in two floats, the
     marks in one `vec4`, and no picking colour;
   - the owners are a `Uint16Array` of flag numbers, not a JS array of strings;
   - a change sends only the rows of `tileData` and the ranges of a buffer it touched, not the band between the
     lowest and the highest.

See the frontend's [The tiles out of sight are not drawn](../apps/frontend/docs/far-side.md),
[The tile under the pointer](../apps/frontend/docs/picking.md) and [What a tile costs on the GPU](../apps/frontend/docs/tile-layout.md).

**Measured** in headless Chrome 157 on an M2 Max, 1280×800 at a device ratio of 2, the fake backend, before and after
in turn on the same machine: on `earth-1` and on a test map of 1,000,749 tiles (the points of
`IcosahedronGeometry(1, 589)` whose nearest `earth-1` lattice point is a tile, on that tile's landmass; built for
measuring only). The CPU slowed 4 times, the median of three runs; GPU time from `EXT_disjoint_timer_query`, which the
other work on the machine made noisy by about a millisecond.

| per frame, CPU slowed 4× | `earth-1` before → after | 1M before → after |
| --- | --- | --- |
| idle spin at the landing view, GPU | 2.8 → 2.2 ms | 8.8 → 1.9 ms |
| idle spin at the landing view, CPU | 1.1 → 1.1 ms | 1.2 → 0.9 ms |
| vertices drawn at the landing view, every pass | 1.27M → 1.01M (no tile) | 2.01M → 1.01M (no tile) |
| idle spin with the flags off, GPU | 3.3 → 3.4 ms | 9.8 → 9.6 ms |
| turning at zoom 3 / 10 / 40, GPU | 2.7 / 1.4 / 1.3 → 3.5 / 2.8 / 1.2 ms | 5.6 / 2.2 / 1.9 → 2.5 / 1.9 / 1.7 ms |
| turning at zoom 3 / 10 / 40, CPU | 1.0 / 0.8 / 0.8 → 1.1 / 1.0 / 1.0 ms | 0.9 / 0.9 / 0.8 → 0.8 / 0.9 / 0.8 ms |
| hover pick in orbit, median / 90th percentile | 2.0 / 2.8 → under 0.1 / 0.3 ms | 2.2 / 3.5 → under 0.1 / 0.3 ms |
| hover pick zoomed in | 2.7 / 4.2 → under 0.1 / 0.1 ms | 2.1 / 2.7 → under 0.1 / 0.2 ms |

With the CPU slowed 6 times (one run): the landing spin's GPU time 2.8 → 2.3 ms on `earth-1`, 9.7 → 2.5 ms on 1M; a
pick in orbit on 1M 9.8 / 20.5 → under 0.1 / 0.1 ms. The `earth-1` GPU figures at zoom 3 and 10 are the machine's
noise: at zoom 3 the page draws no tile at all after, and four more runs of zoom 10 alone gave 1.3 ms before and 1.1 ms
after (medians).

| memory | `earth-1` before → after | 1M before → after |
| --- | --- | --- |
| JS heap | 23.9 → 22.9 MB | 38.0 → 34.0 MB |
| array buffers | 38.0 → 39.8 MB | 91.1 → 97.8 MB |
| GPU, HD graphics on (compact) | 84.2 → 84.2 MB | 102.7 → 102.7 MB |
| GPU vertex buffers, HD graphics off (plain) | 20.8 → 15.5 MB | 62.2 → 42.2 MB |

The new arrays are the tile at each texel (4 bytes a tile), the chunks (0.7 MB at 1M) and the owners (2 bytes a tile);
the strings' array they replace was in the JS heap.

| time to the globe, DevTools' Fast 4G | `earth-1` before → after | 1M before → after |
| --- | --- | --- |
| CPU as is | 1.88 → 1.90 s | 2.41 → 2.49 s |
| CPU slowed 4× | 2.79 → 3.07 s | 5.41 → 5.73 s |

Five runs each, a production build with the fake backend, against a local server that packs files with brotli.

**Against the bar:**
- the idle spin's frame time no worse than today: met, and on 1M its GPU time is a fifth of what it was;
- a hover pick under 2 ms: met, under 0.1 ms;
- memory at most about 30 MB above today's on 1M: met, about 3 MB more with HD graphics, 20 MB less without;
- time to the globe at most 0.5 s more on Fast 4G: met, 0.02 to 0.08 s with the CPU as is, about 0.3 s slowed 4×.

**One thing on screen changed**, against "rendering stays identical": the tiles under the painted flags drew no colour
but wrote their depth, and cut the outline drawn over them where the globe slopes away. With them gone, those outline
pixels show: 0.04 to 0.5% of a view from orbit, all on the outline. Kept (the owner, 2026-11-17: "i don't see a
difference"). Every view where the tiles have the frame (zoom 6,
12, 40, the flags off, a bomb, a virus, the front's rings) is the same to the pixel, with HD graphics on and off.

**Still to do:** a real low-end phone, and a school-style laptop with integrated graphics, which is much of CrazyGames'
audience. A headless M2 Max shows the work removed; it cannot show what a phone's GPU makes of it.

**If a device still struggles**, it cannot have a smaller board: everyone shares the tile ids. But with HD graphics
off, the painted flags can stay on until the player is closer in, and they cost per pixel, not per tile.

## PR 4, measured

**Measured 2026-11-18**, in the container PR 4 was built in: 4 vCPUs and **no GPU**. Headless Chromium at 1280×800, a
device ratio of 1, on SwiftShader (software GL); the production build with the fake backend on each board
(`/play?fakeBoard=<board>`, see the page's [Commands](../apps/frontend/docs/commands.md)), served by a local server that
packs files with brotli, the cache off. Unlike PR 0's M2 Max there is no GPU: a frame takes the software renderer one to
ten seconds, `EXT_disjoint_timer_query` is absent, and so **no GPU time is recorded**: only what does not depend on it,
each board under the same conditions. The CPU per frame is the main thread's in the frame callback, the median of the
frames drawn in 40 seconds with the CPU slowed 4 times; vertices are the sum of each draw's count of a frame.

| per frame, CPU slowed 4× | `earth-1` | `earth-2` | `siege-1` |
| --- | --- | --- | --- |
| idle spin at the landing view, CPU | 7.3 ms | 8.8 ms | 5.8 ms |
| … vertices drawn | 1.02M | 1.89M | 0.51M |
| turning at zoom 3 / 10 / 40, CPU | 6.8 / 6.9 / 4.9 ms | 5.9 / 7.4 / 6.7 ms | 6.1 / 4.4 / 5.3 ms |
| … vertices drawn | 1.01M / 1.73M / 1.72M | 1.88M / 5.09M / 3.49M | 0.71M / 0.69M / 0.69M |
| a pick (`TilePicker.pick` in Node, CPU as is), median / 90%, in orbit and zoomed in | 0.3 / 3.3 and 0.4 / 0.7 µs | 0.4 / 0.5 and 0.4 / 0.5 µs | 0.4 / 0.5 and 0.4 / 0.4 µs |

| memory, loaded at the landing view | `earth-1` | `earth-2` | `siege-1` |
| --- | --- | --- | --- |
| JS heap | 13.6 MB | 24.1 MB | 10.1 MB |
| array buffers | 24.7 MB | 75.3 MB | 6.8 MB |
| GPU buffers + textures allocated, compact layout | 8.2 + 26.7 MB | 24.6 + 32.6 MB | 2.5 + 24.8 MB |

The GPU column is summed from the WebGL calls that allocate (`bufferData`, `texImage2D`, `texStorage2D`), as PR 0's was,
but without the framebuffers, so it is lower than PR 0's figures; the boards compare on it.

| time to the globe, DevTools' Fast 4G (the first WebGL draw) | `earth-1` | `earth-2` | `siege-1` |
| --- | --- | --- | --- |
| CPU as is | 2.06 s | 3.00 s | 1.74 s |
| CPU slowed 4× | 4.19 s | 8.40 s | 3.15 s |
| the board's files the globe waits for, packed | 401 KB | 960 KB | 110 KB |

Three runs each, the median. The files are the runs, the borders, the border lines and the terrain, fetched together.

**What it says:**
- **The tiles still cost nothing at the landing view.** No tile is drawn there on any board. `earth-2`'s 0.87M more
  vertices are its outlines: the border lines draw 12 vertices for each cell edge of an outline (4 corners, 3 passes),
  and `earth-2`'s coasts at 13 km have 129,431 edges against `earth-1`'s 56,790. Less 12 a edge, every board draws the same
  335k. PR 0's test map of 1M tiles borrowed `earth-1`'s outlines, so it read 1.01M; the outlines grow with the coast,
  not with the tiles, as [Slow devices](#slow-devices) says.
- **Zoomed in, `earth-2` draws its tiles in sight**: 5.1M vertices at zoom 10 against 1.7M. On a real GPU PR 0 measured
  this case at 1.9 ms on a 1M map; here only the CPU's side is measured, and it stays at 6 to 9 ms a frame on every board.
- **A pick costs the same on every board**, under a microsecond: the warp's two Newton steps do not show.
- **Memory:** `earth-2` holds about 10 MB more JS heap, 51 MB more array buffers (the positions, the cells, the chunks,
  the tile attributes, at four times the tiles) and 22 MB more on the GPU than `earth-1`.
- **The globe shows 0.94 s later on `earth-2`** with the CPU as is, 4.2 s later slowed 4×: past PR 0's bar of 0.5 s. Of
  the 0.94 s, about 0.56 s is the files, and most of that the border lines (742 KB packed: their int16 corners pack
  poorly); the rest is the rebuild, 255 ms against 100 ms in Node (the positions, then the cells and their check of
  every tile, see [The geometry](#the-geometry)), which the slowed CPU multiplies.

**Still to do before PR 6** switches the world: the border lines' file smaller (its corners delta-coded along each run),
and the cell check of every tile weighed against its 0.2 s; then the time to the globe measured again, with the GPU rows
of PR 0's protocol, on a machine with a GPU and on a low-end phone.

## Screenshots

For the owner to judge the creases before deciding on them (they stay as they are for now, the owner looks first).
Taken 2026-11-18 as the page draws each board, on the fake backend (every tile French), the page's panels hidden,
1280×800 at a ratio of 1, in software GL. Each board's landing view, then a face edge crossing land at three zooms, so
a cell spans about 10, 25 and 60 to 80 pixels: on `earth-2` and `siege-1` the edge across Kansas (39.2°N, 100.9°W) and
the one by the Caspian (47.5°N, 49.9°E); on `earth-1`, whose edges lie elsewhere, the one across Montana (48.2°N,
109.3°W). `/play?fakeBoard=<board>&lat=…&lon=…&zoom=…` opens each again.

| | landing | about 10 px a cell | about 25 px | about 70 px | the Caspian |
| --- | --- | --- | --- | --- | --- |
| `earth-1` | [landing](boards/earth-1-landing.jpg) | [Montana, zoom 6](boards/earth-1-montana-z6.jpg) | [zoom 15](boards/earth-1-montana-z15.jpg) | [zoom 40](boards/earth-1-montana-z40.jpg) | |
| `earth-2` | [landing](boards/earth-2-landing.jpg) | [Kansas, zoom 12](boards/earth-2-kansas-z12.jpg) | [zoom 30](boards/earth-2-kansas-z30.jpg) | [zoom 80](boards/earth-2-kansas-z80.jpg) | [zoom 30](boards/earth-2-caspian-z30.jpg) |
| `siege-1` | [landing](boards/siege-1-landing.jpg) | [Kansas, zoom 2](boards/siege-1-kansas-z2.jpg) | [zoom 5](boards/siege-1-kansas-z5.jpg) | [zoom 16, its most](boards/siege-1-kansas-z16.jpg) | [zoom 5](boards/siege-1-caspian-z5.jpg) |

## The PRs, in order

Each one ships on its own, and players see nothing new until the PR that says so. PR 0 is numbered so the rest keep
their numbers.

0. **Performance, on `earth-1`** (the owner, 2026-11-17: one PR, shipped on its own, today). **Built 2026-11-17.** The
   four fixes of [Slow devices](#slow-devices), measured against its bar. Players see the same globe, faster.
1. **Boards, `earth-1` only. Built 2026-11-17.**
   - The format, and a generator without three that writes `earth-1` (three's order, no warp, no rotation).
   - Every app reads its geometry from the board.
   - The coordinates file, its readers, and the backend's position matching go.
   - Nothing visible.
2. **Eras and the board column. Built 2026-11-17.**
   - `boards`, `eras`, the `board` and `map` columns, the backfill, and the boot's guard.
   - The board on every answer that names tiles, and every reader resolving its events' board.
   - Arenas told apart by column instead of by time.
   - Nothing visible.
3. **More than one board. Built 2026-11-18.**
   - The backend's registry of boards.
   - The page builds its globe from the board an answer names, and reloads when it changes.
   - Everything sized from the spacing or the density reads the board.
   - `arena.board`, and replays that draw each era on its board, with the cut.
   - Nothing visible: only `earth-1` exists.
4. **The new geometry: `earth-2` and `siege-1`. Built 2026-11-18.**
   - The warp, fitted per board, the rotation and the canonical order, in every reader.
   - Both boards generated and committed, with their capitals, bridges, terrain, border lines and English names, and
     their rows in planet's `boards`.
   - The territory shader's inverse and the pick's. The coast is the terrain file's since #864 (the earth shader this
     item first named has no coast left to draw): a tile where earth-1 had sea takes its kind from its land neighbours.
   - The frontend measured on both, and the backend's load of `earth-2`.
   - Nothing plays them yet. The landmass names' translations wait for `make translate`.
5. **Capital Siege on `siege-1`. Built 2026-11-18.**
   - `charges.spread.off`: a click with spread on takes its tile alone and spends nothing, a grant keeps none, the
     spread clicks held are kept, and the page hides the Spread slot.
   - Its set names `siege-1` and drops the spread: the owner saves planet's `capital-siege` once it is deployed (the
     backend's [Capital Siege's set](../apps/backend/docs/arenas.md#capital-sieges-set)); the offers', the boxes' and the
     quizzes' sets need no change.
   - Before 2026-11-21 19:00 UTC, `siege-1`'s landmasses are translated in every language by hand (PR 4 ran without the
     translator's key; no `make translate` run, to save its credits), and `siege-1` comes off the page's
     `AWAITING_TRANSLATION`; `earth-2` still waits.
   - The first board players see, at the next Capital Siege and before the world's switch. It tries the reload and an
     arena on another board for real.
   - **Rehearsed locally** (2026-11-18, the PR's screenshots): the arena opened on `siege-1` and both pages, one framed as
     on CrazyGames, reloaded inside the freeze and again at the close; every playable flag's capital (235) started with
     50 shields; the reach crossed a bridge (Malta's one tile to Cap Bon); a capital taken knocked its flag out (the
     Vatican, by Italy); a click with spread on, 8 spread clicks held, spread nothing and spent nothing; and the battle's
     ledger events name the arena's map and `siege-1`. **A world history across the battle stays on `earth-1`**: the
     replays, the last 24 hours and the player's video skip the arena's events, as the backend's
     [The ledger's readers](../apps/backend/docs/arenas.md#the-ledgers-readers) has it, so nothing draws the battle on
     `siege-1` and no cut shows.
6. **The switch.**
   - The carry-over file, and `SwitchBoard` with its dry run.
   - The moment, the chat line and the Discord ping.
   - The 24-hour readers starting at the switch, and the new format of the last 24 hours.
   - The rehearsal's report.
   - Then the operator's steps: deploy, dry run, switch at a quiet hour, and `rally.rings` 18 → 36 in every set that
     holds it.
