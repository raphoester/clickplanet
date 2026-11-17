# Boards: a map's geometry as files of its own

**Designed 2026-11-17, not built yet.** This is the reference the PRs at the end of this doc build from. Each PR turns
the part of this doc it builds into "how it works", and updates the docs it makes stale: [`map/README.md`](../map/README.md),
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
| `earth-1` | 260,503 | ~25 km | the world until the switch, and all history before it |
| `earth-2` | ≈1,000,000 (`cols` 590) | ~12.5 km | the world after the switch |
| `siege-1` | ≈27,000 (`cols` about 97) | ~79 km | Capital Siege's arena |

The names are proposals. A board's tile count is fixed by its `cols` and the land the generator finds, so the counts
above are estimates from `earth-1`'s land share (28.75% of the lattice).

## The geometry

**Every board is an icosahedral lattice.** Each of the 20 faces is a triangular grid of `cols` points per edge. A point
is `(face, i, j)`, and its neighbours are the ±1 exchanges of its barycentric coordinates: 6 everywhere, 5 at the 12
corners.

**The 12 five-neighbour points cannot go.** On a sphere, a grid where nearly every point has 6 neighbours has exactly
12 with 5: Σ(6 − degree) = 12 (Euler). So they go to sea. One flat grid per continent would avoid them but stretches
worse. Afro-Eurasia is about 150° across, twice a face, so the best flat grid over it puts tiles up to 1.57× apart, or
squashes them 1.57× in one direction.

**The position of `(face, i, j)`** on a face with corners A, B, C:

1. The barycentric `w = (wa, wb, wc)` of `(i, j)`, as `lattice.ts` computes it today.
2. **The warp** (new boards only):
   - `h_k = w_k · (1 + a·w_k + b·w_k²) + c·wa·wb·wc`
   - `w'_k = h_k / (ha + hb + hc)`
   - with `a = −0.05520744`, `b = −0.16849639`, `c = 0.30479447`.
3. `p = w'a·A + w'b·B + w'c·C`, then normalised as today (`× 1 / length`).
4. A, B, C are three's 12 unnormalised icosahedron corners, **rotated once** (new boards only) and kept as float64
   literals.

**Only `+ − × ÷ √`**, so the rule of [Tile positions](../apps/frontend/docs/tile-positions.md) holds: every engine
rebuilds every position bit for bit. With no warp and no rotation it is today's lattice exactly, which is how `earth-1`
is described.

Measured on the same lattice (`cols` 48; the ratios do not depend on `cols`):

| | `earth-1` (gnomonic) | warped |
| --- | --- | --- |
| tile area, largest ÷ smallest | 1.91 | 1.004 |
| closest two tiles ÷ median spacing | 0.74 | 0.90 |
| tiles a 4-ring bomb takes, wherever it lands | 48 to 89 | 54 to 64 (the lattice's own noise) |
| … on a pentagon | 91 | 61 |

Snyder's equal-area projection was measured too: area ratio 1.03, closest two tiles 0.85. It needs sin and cos, which
break the bit-for-bit rebuild.

**The coefficients were fitted at `cols` 24** (Nelder-Mead on the spread of cell areas). Fit them again at each board's
`cols` before generating, and pin the fit in the generator's tests. No lattice triangle folds (checked at `cols` 400).

**The territory shader inverts the warp.** The gnomonic inverse, as today, gives `w'`. Two Newton steps starting from
`w = w'` reach 5e-8, against 1.7e-3 for one cell at `cols` 590. A shader that finds the wrong cell makes `cellsOf`
throw, and the globe does not load, so the inverse's tests matter.

**The rotation puts the 12 corners at sea**, each at least 184 km from an `earth-1` tile. It was found by a random
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

Latitude is `asin(y)` and longitude `atan2(z, −x)`, as everywhere in the map scripts. Check the distances again
against `earth-2`'s own coasts once it is generated.

**What it leaves: creases.** Where hex rows cross one of the 30 face edges they turn, by up to 6 to 9° on land.
`earth-1` has the same. They stay for now (the owner, 2026-11-17): look at `earth-2` before deciding. A relaxation
table could smooth them later and keep the bit-for-bit rebuild.

**Tile order.** A lattice point's canonical id is today's `canonicalId`: the corners, then the 30 edges, then each
face's inside, row by row.
- **New boards** number their tiles in canonical id order, so a board is ranges of canonical ids.
- **`earth-1` keeps three's walk order**, because its ids are the ones the ledger holds.
- Above `cols` of about 20,000 a canonical id passes 2³². The file stores them in 64 bits; tile ids, which are ranks,
  stay 32-bit.

## The files of a board

Each board lives in `map/boards/<name>/`, content-addressed like every file in `/map` today:

- **`board-<hash>.json`**, the manifest:
  - `name`, and `body` (`earth`);
  - the lattice: `cols`, `warp` (`null` or `{a, b, c}`), `corners` (`null`, or the 36 rotated float64s) and `order`
    (`three` or `canonical`);
  - `tiles`, the median `spacing` in radians, and `positionsHash`, FNV-1a over the float32 positions as the landmask
    carries today;
  - the names of its other files.
- **`runs-<hash>.bin`**: `"CPRN" | uint32 version | uint32 count | count × (uint64 first, uint64 last)` canonical ids,
  ascending. It replaces the landmask.
- **`borders`, `capitals`, `bridges`**, in today's formats, named by the board's tile ids. `earth-1`'s are today's
  files.
- **The frontend's own**: the border lines and the landmass names, per board. The earth texture belongs to the body,
  not to the board.

**No coordinates file, on any board.** Every app rebuilds positions from the lattice and the runs:
- the frontend, in its worker as today;
- the backoffice, with the same module;
- the scripts;
- the backend, in Go. It needs positions only for `Within`, `Nearest` and `Spacing`, so a tolerance will do. It takes
  neighbours straight from the lattice, with no position matching.

**A hash that does not match is logged, not fatal.** Today a mismatch downloads the coordinates file. With no such
file, a page logs the mismatch with its user agent and keeps the positions it computed: the ids come from the runs, so
an ulp of difference moves a tile by under a metre.

**Copies:** each app carries the boards it plays or reads, committed, through `npm run map` and `make map` as today.
The frontend serves every board's files forever. A board costs the backend about 57 bytes a tile (about 60 MB for
`earth-2`), loaded when it is first needed.

## Eras and the board column

In planet's schema:

- **`boards`** `(id smallint, name text unique, hash text unique)`, a row added by the migration that ships a board.
- **`eras`** `(map text, board smallint, started_at timestamptz)`: a map's latest era is the board it plays.
- **A `board smallint NOT NULL` column** on `tiles`, `landmasses` (keyed by the borders asset today), `mines`,
  `viruses`, `ledger_events`, and the vandal squad's `graffiti`.
- **A `map` column on `ledger_events` too**, so readers stop telling arenas apart by time (`arenas.Played.After`). A
  reader never guesses a board or a map from a timestamp.
- **The backfill writes `earth-1` everywhere.** That is true: both past remaps rewrote all history onto today's ids,
  and arenas played on the world's board.
- **The boot refuses** a map whose tile rows name a board other than its era's, or a board the process does not
  carry. Today a skipped remap boots old ids onto new ground with no error.

**Every answer that names tiles names their board:**
- `GetMap`;
- the stream as it opens (a change of board cuts every stream, as a new epoch);
- `GetMapHistory`, `GetRecentMapHistory`, `GetReplay` and the season replay, once per era they cover.

**A page that reads a board other than the one it drew reloads** (the owner: a reload, not a rebuild in place).

**Every reader of the ledger resolves each event's board:** replays, clips, the archive, the last 24 hours, the
holder card, the landmass card, and the relay (its `Ground` and `Position`). FindPlayers, TopPlayers, RevertPlayer and
the vandal squad read the current board's events only.

## Arena boards

**`arena.board`** is a rule set's knob: the board an arena's map plays on (default: the world's).

**Capital Siege plays `siege-1`, with no spread at all**: `charges.spread` is off in its set, and none comes in its
boxes (the owner, 2026-11-17). On 260,503 tiles its unlimited spread was inflation on purpose, to fill the map fast,
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
| fortify news | `chat/…/announcement.go` `fortifyNewsTiles`, and the frontend's `NEWS_FROM_TILES` | 50 | 200 |
| largest drawing | vandal squad `reading.go` `maxTiles` (12 at least stays) | 150 | 300 |
| clip thresholds | frontend `clip/look.ts`, `clip/changes.ts` | 20, 100, 400, 1,000 | ×3.9 |
| attack ping zoom | alerts `pings/view.go` `closestZoom` | 20 | 40 |
| heatmap cell | frontend `archive/heatmap.ts`, backoffice `home/heat.ts` | 0.4°, 0.3° | 0.2°, 0.15° |
| max zoom | frontend `zoom.ts` | 50 | 100 |

**The thresholds read the board, not constants**, so they hold on `siege-1` too:
- **Counts of tiles** scale with the board's density divided by `earth-1`'s: the mission floors, the fortify news, the
  largest drawing, the clip thresholds.
- **Lengths** scale with its spacing: the heatmap cell, the zoom, the attack ping's zoom.
  - **Max zoom** keeps a tile at max zoom as wide as on `earth-1`: 100 on `earth-2`, about 16 on `siege-1`.
  - **The frontend's hard-coded spacings** become the board's spacing: `REACH`, `TILE_SPACING`, `tilePointSize`,
    `tileSpacing`, `LEAST_FLAG_RADIUS`, `VIRUS_LIFT`, and the fake backend's constants.

## What `earth-2` costs

- **Frontend:**
  - **GPU tile data:** 5 → 21 MB in the compact layout, 58 MB in the plain one.
  - **`tileData`:** at 512 columns it fills 2,022 of the 2,048 rows WebGL2 guarantees, so it goes to 1,024 columns.
  - **`GetMap`:** 0.5 → 2 MB raw, in batches of 10,000.
  - **The position rebuild:** about 300 ms with a JIT, about 6 s in iOS Lockdown Mode.
  - **Zoomed out**, the display points are still drawn under the painted flags. Hide them there, or they cost 4× the
    vertex work while the globe spins.
- **The earth texture:** a tile is 1.5 px of the 4,096 px photo, so the coast is cut in the earth shader from the tile
  set (the owner: no 8,192 px texture).
- **Natural Earth stays at 1:50m** (the owner). `seats.mjs` loses the countries that now have ground of their own.
- **The last 24 hours:** its first frame grows 4×, past its 1 MiB budget, so it needs a new format.
- **The backend:**
  - **Memory:** about 45 MB more resident, and about 200 MB at the boot's peak while it builds the edges.
  - **The boot:** 4 to 6 seconds, against a one-minute deadline.
  - **`Within` and `Nearest`:** tens of milliseconds per bomb or rally.

## The PRs, in order

Each one ships on its own, and players see nothing new until the PR that says so.

1. **Boards, `earth-1` only.**
   - The format, and a generator without three that writes `earth-1` (three's order, no warp, no rotation).
   - Every app reads its geometry from the board.
   - The coordinates file, its readers, and the backend's position matching go.
   - Nothing visible.
2. **Eras and the board column.**
   - `boards`, `eras`, the `board` and `map` columns, the backfill, and the boot's guard.
   - The board on every answer that names tiles, and every reader resolving its events' board.
   - Arenas told apart by column instead of by time.
   - Nothing visible.
3. **More than one board.**
   - The backend's registry of boards.
   - The page builds its globe from the board an answer names, and reloads when it changes.
   - Everything sized from the spacing or the density reads the board.
   - `arena.board`, and replays that draw each era on its board, with the cut.
   - Nothing visible: only `earth-1` exists.
4. **The new geometry: `earth-2` and `siege-1`.**
   - The warp, the rotation and the canonical order.
   - Both boards generated and committed, with their capitals, bridges, border lines and names.
   - The territory shader's inverse, and the coast in the earth shader.
   - The frontend measured on both.
   - Nothing plays them yet.
5. **Capital Siege on `siege-1`.**
   - Its set names `siege-1` and drops the spread.
   - The first board players see, at the next Capital Siege and before the world's switch. It tries the reload and an
     arena on another board for real.
6. **The switch.**
   - The carry-over file, and `SwitchBoard` with its dry run.
   - The moment, the chat line and the Discord ping.
   - The 24-hour readers starting at the switch, and the new format of the last 24 hours.
   - The rehearsal's report.
   - Then the operator's steps: deploy, dry run, switch at a quiet hour, and `rally.rings` 18 → 36 in every set that
     holds it.
