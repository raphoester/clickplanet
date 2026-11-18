# The tile map's geometry

A map's geometry is a **board**: a set of immutable files in `boards/<name>/`, content-addressed, that every app reads
its tiles from. There are three (see [Boards](../docs/boards.md) for the design and what comes next):

| board | lattice | tiles | apart | plays |
| --- | --- | --- | --- | --- |
| **`earth-1`** | three's, `cols` 301 | 260,503 | 25.9 km | the world until 2026-11-18, and all history before |
| **`earth-2`** | warped and rotated, `cols` 590, canonical order | 1,004,682 | 13.1 km | the world, since 2026-11-18 |
| **`siege-1`** | warped and rotated, `cols` 97, canonical order | 27,259 | 79.4 km | Capital Siege's arena |

| file | says |
| --- | --- |
| `board-<hash>.json` | the manifest: the lattice, the tile count, the spacing, the positions' hash, and the other files' names |
| `runs-<hash>.bin` | which points of the lattice are tiles, as ranges of canonical ids |
| `borders-<hash>.bin` | whose ground each tile sits on (see [Format](#format)) |
| `capitals-<hash>.json` | which tile is each country's capital (see [Capitals and bridges](#capitals-and-bridges)) |
| `bridges-<hash>.json` | which crossings join two shores over the sea |
| `terrain-<hash>.bin` | what each point of the lattice looks like: a kind of land under a tile, or the sea by its distance from land (see [The terrain follows too](#the-terrain-follows-too)) |
| `provinces-<hash>.bin` | which province each tile is in, the ground fortify plays on (see [Provinces](#provinces)); `earth-2` and `siege-1` only |

**There is no coordinates file.** Every app rebuilds every tile's position from the lattice and the runs, to the bit
(see [The board's lattice](#the-boards-lattice)).

The files are shared by the apps exactly as [`/proto`](../proto) is. No app may keep its own generator or its own
copy of the numbers; each carries a **generated copy** of the boards it reads, because each is built from its own
directory as its Docker build context and so cannot read anything at the repo root at build time:

| app | copy | refreshed by |
| --- | --- | --- |
| frontend | `apps/frontend/static/boards/<name>/`, every file, served beside the page's own border lines and landmass names; every board's manifest also inline in `src/app/viewer/boardsAsset.ts` | `npm run map` (and `map:generate`, `bridges`) |
| backend | `apps/backend/generated/map/boards/<name>/`, every file, embedded | `make map` |
| backoffice | `apps/backoffice/src/gen/map/<name>/`, the manifest and the runs, bundled by Vite | `npm run map` |

**The tests' fixture is no board of `/map`**: `fixture-1`, a coarse earth cut from `earth-1` at `cols` 31, which the
page's `npm run map:fixture` writes into each app's test data alone (the backend's `internal/shared/cpboards/testdata`,
embedded under the `testing` tag; the page's and the backoffice's `src/testdata/boards`). See [Boards](../docs/boards.md).

All copies of a file are committed and byte-identical, so git stores the bytes once. A file's name carries the first
8 hex of its sha256, which makes a stale copy visible by eye rather than by symptom. The manifest names the others, so
its own hash changes whenever any of them does.

## One question, one answer

**A tile exists exactly where a country does.** `apps/frontend/scripts/map/ground.mjs` is the one
oracle: `groundOf(lon, lat)` answers a country code or `SEA`, from Natural Earth 1:50m; a vertex
whose cell is mostly lake is `SEA` too; then each country too small for the lattice gets its seat
(`latticeGround` in `scripts/map/oracle.mjs`), and `npm run map:generate` writes the runs and the borders from that
one answer. So "is this sea or land" and "which
country is this" cannot disagree, and **a tile in no country cannot be built**.

That used to be two questions asked of two datasets — a greyscale land mask decided which lattice
vertices became tiles, and Natural Earth decided which country each tile was in. They disagreed
about 6,000 of the 906,012 lattice vertices, all of them on coasts and islands, which is exactly
where anyone looks: 2,093 tiles floating on open water with no border drawn round them, 3,729 islands
drawn in the globe's texture with nothing to click, and the Antarctic ice shelves missing whole.

Three things follow from having one oracle, and they are worth knowing before changing it, and one exception
is written into it:

- **The Natural Earth tag is pinned** (`NATURAL_EARTH_TAG` in `scripts/map/naturalEarth.mjs`,
  `v5.1.2`). These files decide where every tile is, so two runs a year apart have to produce the
  same board. Moving the pin renumbers every tile; see below.
- **The ice shelves are ground, and they are Antarctica.**
  `ne_50m_antarctic_ice_shelves_polys` is a file of its own and is inside no country polygon, so
  before this Ross and Ronne were tileless holes in a continent the photo painted solid white.
  They are 2,633 tiles, which is about a tenth of `aq`.
- **A cell that is mostly lake is water.** See below.
- **Every country has at least one tile: its seat.** See below.
- **The terrain follows, it does not decide.** `npm run terrain` paints land exactly where the runs put a tile — see
  below — and `npm run map:audit` fails on a point astray. Until November 2026 the globe was a 4096×2048 photo, which
  blends a one-tile island into open water, so it had to be corrected toward the tile field.

```bash
cd apps/frontend && npm run map:generate -- earth-2   # a board's runs, borders and capitals, from the oracle
cd apps/frontend && npm run map:audit -- earth-2      # the three sources against each other
```

Every script of the chain takes the board's name, and works on `earth-1` when none is given. `scripts/map/board.mjs`
defines each board the generator writes (`BOARD_DEFINITIONS`): its name, its `cols`, its warp and corners, its order.

## Lakes are water

Natural Earth draws each country over its lakes, so the Great Lakes, Victoria, Tanganyika, Malawi, Baikal and Ladoga
were all tiles, painted over the water the photo showed. `ne_50m_lakes` is the third layer of the oracle, and
**a vertex is sea when most of its cell is lake** (`mostlyLake` in `scripts/map/ground.mjs`): 19 points over the
cell, the vertex and two rings, and more than half of them in a lake. It is the cell and not the vertex because a
lake is often narrower than a cell — a reservoir down a valley, Lake Champlain — and a vertex that falls in one would
cut water a cell wide out of the land; a cell is a fair answer for a lake as wide as one. 1,644 vertices are lake,
and an island in one stays land.

`DRY` lists the lakes that stay land: Australia's salt pans, which fill a few years a century; Kakhovka, drained in
2023; and the steppe's shallow salt lakes (Tengiz, Qusmuryn, Siletitengiz), which the photo drew as steppe. Natural
Earth draws them full, and a cell with no tile on what the globe draws as land is ground nobody can take.

## A seat for each country the lattice misses

`earth-1`'s lattice is 25 km apart, so a country smaller than a cell — the Vatican, Monaco, Liechtenstein — or an island
nation whose islands all fall between vertices — the Maldives, Palau, Tuvalu, Bahrain, which is bigger than a tile
and still misses every vertex — would have no ground at all: no tile to take, no territory, no place for a first
visit's camera to open on. `scripts/map/seats.mjs` gives each of them **one** vertex, after Natural Earth has
answered for the rest, so `npm run map:generate` and `npm run map:audit` read the same answer. 34 countries have one.

The rule is to follow the geography as closely as a 25 km cell allows, and never to put a tile on what the globe
draws as open sea when the country is on land:

- **A country on a continent takes its neighbour's vertex** nearest it (`takes`), on the coast or inland as the
  country is (`coast`): the Vatican and San Marino from Italy, ringed by it; Liechtenstein from Switzerland, on the
  Austrian border; Monaco from France, on the shore by Italy; Gibraltar from Spain, rather than the vertex in the
  strait that touches Morocco; Macao from China. The neighbour keeps the rest, in one piece.
- **An island country is one tile, on its capital's island**, at the vertex whose cell holds the most of that
  island's land. Never a tile per island: the Seychelles' islands spread over 1,200 km, and a tile for each would
  draw it far past its own waters. The vertex is sea to Natural Earth, because the island is smaller than the cell;
  the terrain then paints the cell as land (see below). Two of them touch a neighbour's tile, as the islands do:
  Anguilla touches Sint Maarten and the US Virgin Islands touch the British.

**Each board has its seats** (`SEATS_BY_BOARD`, `seatsOf`): a lattice of another spacing misses other countries.
`earth-2`, at 13 km, gives 20 of `earth-1`'s 34 their own ground and keeps 13 of the seats; Christmas Island takes
Australia's vertex there (Natural Earth draws the Indian Ocean Territories inside Australia), and the British Virgin
Islands, whose vertices fell on `earth-1`'s lattice and not on `earth-2`'s, get one: 15. `siege-1`, at 79 km, seats 52:
`earth-1`'s but the British Indian Ocean Territory's, and 19 more countries smaller than its cell, each at its capital
(Luxembourg takes France's vertex, Hong Kong a vertex in the water off the New Territories, whose cell holds the city).
On a lattice coarser than the gap between two islands both can be nearest one vertex: **`beside`** names the seat
before it that keeps it, and this one takes the nearest vertex the other did not (Guernsey beside Jersey, the British
Virgin Islands beside the US ones, on `siege-1`).

A seat for a country that has ground of its own, an island seat that would land on a country's ground, or two seats on
one vertex that no `beside` names refuse the generation. **The audit fails on a flag with no ground**, except the four flags of a region
inside a country's ground (England, Scotland, Wales, Brittany: `REGIONS`), which share it.

**Every fault between the tile set and the oracle is zero, and the audit fails on any one of them.**
They are not a budget that creeps: the runs and the borders are cut from one `groundOf` call, so a tile on open sea
or a tile in no country is not a map that drifted, it is a generator that broke.

## Capitals and bridges

The capitals are written by `npm run map:generate` from the same answer as the tiles, and the bridges by `npm run
bridges` from the tiles and the capitals written. Both are read only by the rules of an arena battle (see the backend's
[Arena battles](../apps/backend/docs/arenas.md#seeds) and [The reach](../apps/backend/docs/reach.md)). Tiles are named by their wire id
(index + 1).

- **`capitals-<hash>.json`** — `{"tiles": N, "capitals": {"fr": <tile>, …}}`, one for each country with ground, 239.
  `scripts/map/capitals.mjs` reads Natural Earth's populated places (`ne_10m_populated_places`) and picks each
  country's city: its main capital (`Admin-0 capital`, not the alternate one: La Paz, Abidjan, Pretoria, Naypyidaw),
  else a territory's or a claimed capital (`Admin-0 *`: Nuuk, Stanley, Gaza City), else its biggest place (McMurdo
  Station, Tórshavn, San Juan); among equals, the country's own place before one of a land it absorbs (Mogadishu,
  not Hargeisa), then the biggest. The capital is the country's tile nearest the city. A country of one tile, a seat
  among them, starts on it; one with no place at all (the French Southern Territories) in the middle of its biggest
  landmass. No hand-kept list.
- **`bridges-<hash>.json`** — `{"tiles": N, "bridges": [[<tile>, <tile>], …]}`, each bridge's two end tiles, the
  smaller first, in order. A bridge lands on a coast tile, and on a capital only when its piece of land has no other
  coast. `scripts/map/bridges.mjs` writes two kinds (the owner, 2026-11-11):
  - **Stepping stones**, generated: a piece is land that touches, whatever its countries (505 on this map, 86 with a
    capital). Two pieces get a bridge, between their nearest coast tiles, when no third piece is nearer to both. Every
    piece is joined, a strait is crossed where it is narrowest, and an island chain is crossed island by island: 565
    of them, 361 under 100 km. Europe reaches America by the Faroes, Iceland and Greenland as well as by the Bering
    Strait, and Hawaii by Kiribati and the Aleutians.
  - **Sea lanes**, placed by hand (`LANES`): two points [lon, lat] on either shore, snapped to the nearest coast tile
    that is no capital (or a capital whose land has no other coast, as a stone lands), within 150 km or the generator
    stops (as many of a coarser board's spacings, `snapKmOf`). Each board has its lanes (`LANES_BY_BOARD`): `siege-1`'s
    Gibraltar lane leaves from Málaga, since the Rock is Gibraltar's one tile and Morocco's coast is nearer Tarifa than
    Spain's next. 17, for the crossings the stones do not make: the North
    and South Atlantic, Hawaii to California and Japan, the Tasman, the monsoon, Gibraltar. The points are pinned, like
    the seats: moving one moves a bridge.
  - **Second ways**, generated after both (the owner, 2026-11-11: "each group of islands has two ways of being
    joined, so there are several fronts to defend on each mainland"): while one bridge alone holds some land to the
    rest, the shortest other crossing between its two sides is added, the smallest side first. A second bridge between
    the same two pieces lands 250 km (`APART_KM`) from the first on one side at least, so it is another beach. 162 of
    them: no single bridge's loss cuts any land off, so every island has two bridges at least (Great Britain 8, Ireland
    3, Sicily 3, Cyprus 2 to two beaches of the mainland, Tasmania 3), and Afro-Eurasia 128.

  744 bridges in all on `earth-1`; 1,422 over 975 pieces on `earth-2`, 272 over 173 on `siege-1`.
  Until 2026-11-11 the reach crossed `waters-<hash>.json`, each body of water's coast: the ocean was one body, so a
  flag on any coast reached every coast on Earth, and 130 of the 239 capitals were in reach of every flag at the start.

`npm run map:audit` fails on a country with ground and no capital, a capital off its own ground, bridges that differ
from a fresh `bridgesOf`, and land no bridge reaches. The backend refuses to boot on a capital off its ground or a file of another board's
size.

## Regenerating renumbers every tile

`earth-1`'s tile ids are ranks in three's walk of the lattice, filtered by what the oracle calls land, so a
regeneration that adds or removes one tile moves every id after it — including on a server that is already holding
players' territory. **The ids are the only thing that moves.** A tile's position does not, so a tile that exists in
both tile sets can be followed from one to the next, and `npm run map:generate -- --remap <path.up.sql>` writes the
postgres migration that does it:

```bash
cd apps/frontend
npm run map:generate -- --remap ../backend/internal/planet/internal/migrations/<ts>_remap_tiles.up.sql
```

It writes the `.down.sql` beside it. The mapping is **monotonic** — both tile sets are the same lattice
in the same walk order, filtered by what each calls land — so it run-length encodes to a few
thousand rows instead of a quarter of a million. `apps/backend/internal/planet/internal/migrations
/migrations_test.go` runs it against a real postgres, both ways.

**Every column of a tile goes with it** — its flag, shields and holder, and whatever a later migration adds —
because the migration copies the row whole rather than naming columns. **A landmass id moves too**: it is a row of
the borders file's table, and a regeneration renumbers them. The migration follows each old landmass to the new one
its tiles are on, for the same country, and moves the landmass each fortify in the ledger names (the territory card
reads the last one by it) and copies the fortify locks to the new borders file's name (the boot reads only those).
A landmass that split goes to the piece that holds most of it — Canada keeps its lock when a lake cuts an island off
it — and one with no such piece, or whose piece is another's too, is followed to nothing: its lock is settled at boot
as any other.

What the migration cannot give back: a take on a tile the new map does not have is deleted, because
leaving it would let a revert write to an id that now means other ground.

**The remap is computed against the board currently in `/map`**, so run it with `--remap` on the
first run. A second run over its own output would write an identity migration.

The design keeps a regeneration from ever rewriting the ledger again: a new tile set is a new board, and the old
one stays (see [Boards](../docs/boards.md)). Until the boards are told apart in the database, this is the way.

**The rest of the checklist**, which the generator prints (`<board>` is the board's name; none is `earth-1`):

```bash
cd apps/frontend && npm run bridges -- <board>        # the crossings, from the new tiles and capitals; rewrites the manifest
cd apps/frontend && npm run terrain -- <board>        # each lattice point's look, from the new tiles; rewrites the manifest
cd apps/backend && make map                           # copy the boards, then commit every copy
cd apps/backoffice && npm run map                     # copy the manifests and the runs
cd apps/frontend && npm run borderLines -- <board>    # traced from the board, stale until it is run
cd apps/frontend && npm run landmassNames -- <board>  # each landmass's name, keyed by the new borders file
cd apps/frontend && npm run provinces -- <board>      # its provinces and their names, from the new borders; rewrites the manifest
cd apps/frontend && npm run map:audit -- <board>      # every fault should read zero
# then Claude Code's translate skill (.claude/skills/translate): the landmass and province names in every language
```

**A board written the first time** has no bridges nor terrain until those two commands write them: its manifest leaves
them out, and `npm run map` copies it nowhere until it names every file. A new board's row in planet's `boards` ships
with the migration that ships the board (name, and `positionsHash` in 8 hex).

No config follows the tile count: the backend reads it from the manifest, and refuses a file of the board that holds
another (`gameMap.maxIndex` went on 2026-11-18). A board regenerated is another board, with a row of its own in
planet's `boards`, shipped by the migration that ships it.

**Regenerating `earth-1` as it is writes it byte for byte**: the same runs, borders, capitals, bridges, terrain and
manifest, and the same border lines and landmass names. That was checked when the generator stopped building
three's geometry (2026-11-17), and it is the test of any change to the generator that should change nothing.

## Format

Little-endian throughout.

**The manifest**, `board-<hash>.json`, written by `scripts/writeBoard.ts` and typed in
[`board.ts`](../apps/frontend/src/domain/board.ts):

```json
{
  "name": "earth-1",
  "body": "earth",
  "lattice": {"cols": 301, "warp": null, "corners": null, "order": "three"},
  "tiles": 260503,
  "spacing": 0.004063093462255102,
  "positionsHash": 1068665823,
  "files": {"runs": "runs-<hash>.bin", "borders": "…", "capitals": "…", "bridges": "…"}
}
```

- `lattice`: `cols` steps along each edge of the icosahedron. `earth-1`'s `warp` and `corners` are `null` (no warp, no
  rotation: three's own lattice) and its `order` is `three`, tile ids in three's walk order. `earth-2`'s and `siege-1`'s
  `warp` is `{"a", "b", "c"}`, the coefficients fitted at their `cols`, `corners` the 36 float64s of three's corners
  turned by the rotation (x, y, z of each, in three's order of corners), and `order` is `canonical`, tile ids in
  canonical id order (see [The board's lattice](#the-boards-lattice)). Every reader refuses another order.
- `spacing`: the median arc between two touching tiles, in radians (about 25.9 km).
- `positionsHash`: FNV-1a over the tiles' positions as float32 words, x, y, z per tile in tile order. Every app checks
  its rebuild against it.

**The runs**, `runs-<hash>.bin`, mirrored by [`board.ts`](../apps/frontend/src/domain/board.ts) and
[`runs.go`](../apps/backend/internal/planet/internal/clicks/embedded_board/runs.go):

```
"CPRN" | uint32 version (1) | uint32 count | count × (uint64 first, uint64 last)
```

Each run is a stretch of tiles side by side in one row of the lattice, its first and last canonical id, both in,
ascending and apart. `earth-1` is 7,556 runs, 121 KB (the landmask it replaced was 113 KB, the coordinates file 5.2 MB).

**Positions are 0-indexed by tile order; wire tile ids are 1-based.** Tile id = index + 1.
That is what `clicks.Played.CheckTile` (`tile > 0 && tile <= tiles`), `inmemory_tile_storage`
(slot 0 unused) and the frontend's GPU picking (`integerToColor(i + 1)`) all agree on.

The borders file, written by the same command and read by `borderField.ts` and `clicks.Borders`:

```
uint32 header length | JSON {"tiles": N, "codes": [...]} | N uint16 landmass | 0-2 pad | frames | totals
```

Landmass 0 is no country; `codes[k]` is landmass `k`'s ISO code, so one country has many landmasses.
Nothing reaches slot 0 now that every tile has a country, but the format keeps it.

**The header and the landmass table are both padded to 4 bytes.** The second pad is not decoration:
an odd tile count leaves the table on a 2-byte boundary, and the browser's `new Float32Array(buffer,
at, …)` then throws and the globe does not load at all. Every map until 262,119 tiles happened to
have an even count, so it only appeared when one did not.

**Two tiles are the same landmass when they touch on the lattice and carry the same country.** That
was a distance threshold until it was the exact adjacency, and the threshold is the kind the
backend's own geography notes warn against: the spacing varies about 25% between a face's middle and
its corners, so one radius reaches past the neighbours in places and joined two islands across a
strait into one landmass — a flag painted over the water between them.

The terrain file, written by `npm run terrain` and read by `terrain.ts`, mirrored by
[`terrain.ts`](../apps/frontend/src/domain/terrain.ts):

```
"CPTR" | uint32 version (1) | uint32 points | runs to the end
```

One kind a lattice point, in canonical id order, as runs: a run is one byte, its kind in the high four bits and its
length less one in the low four, and a run of 16 points or more sets the low four to 15 and carries its length less 16
after it as a varint (seven bits a byte, low first). Kinds 0-7 are land (forest, tundra, boreal, arid, savanna, grass,
sand, snow), 8-10 the sea one, two and three steps from land, 11 the open sea. `earth-1` is 66,392 runs, 74 KB; one byte
a point would be 906 KB. `earth-2` is 197 KB, `siege-1` 15 KB.

The frontend draws a black outline along the tile boundary between two countries, and keeps the
geometry for it in a file of each board's own — `apps/frontend/static/boards/<board>/borderLines-<hash>.bin`, written by
`npm run borderLines -- <board>`. It is derived from the board's runs and borders and from nothing else, so **regenerate it
after these**; it stays out of `/map` because the backend has no use for it. See
[`apps/frontend/docs/country-outlines.md`](../apps/frontend/docs/country-outlines.md).

```
"CPBL" | uint32 version (3) | uint32 runs | uint32 corners
       | runs × uint32 corners in the run | runs × 2 uint16 landmasses either side, the sea 0
       | runs × 3 int16 first corner | for each corner after its run's first: 3 zigzag varints, x y z less the corner before
```

Little-endian. A corner is a cell's corner, its unit direction × 32767 rounded. Two corners side by side are one cell
edge apart, a few dozen units, so most differences take a byte (zigzag: `(d << 1) ^ (d >> 31)`, then seven bits a byte,
low first). Version 2 (until 2026-11-18) held every corner as three plain int16s; version 3 holds the same lines in
half the bytes, and packs to a third: `earth-2` 810 → 422 KB raw, 767 → 220 KB with brotli at 5 (`domain/borderLinesFile.ts`).

### Provinces

**`provinces-<hash>.bin`**, written by `npm run provinces -- <board>` (`scripts/generateProvinces.ts`) for a board whose
definition has a `provinceFloor`, read by `domain/provinces.ts` and `embedded_board.loadProvinces`. How the provinces are
cut and named is in [Provinces](../docs/provinces.md).

```
"CPPV" | uint32 version (1) | uint32 tiles | uint32 provinces, none included | uint32 borders
       | provinces × 5 float32 frame | provinces × uint32 tiles | provinces × uint16 landmass | tiles × uint16 province
```

Province 0 is none: no tile is in it, and its frame, count and landmass are 0. `borders` is the borders file the
provinces were cut on, the 8 hex of its name as a number (`borders-0ae1a037.bin` is 0x0ae1a037): its landmass ids are the
ones the provinces name, and a reader refuses another. A frame is the borders' (centre x, y, z and the half extents east
and north, in radians). The arrays of 4-byte values come first, so every one is aligned for the browser's typed arrays.
`earth-2`'s is 2,016 KB (113 KB with brotli at 5), `siege-1`'s 73 KB. The names are the page's alone, beside its copy:
`static/boards/<board>/provinceNames-<hash>.json`, keyed by the provinces file.

### The carry-over

**`/map/carries/carry-<from>-<to>-<hash>.bin`** names, for each tile of a new board, the tile of an old one it comes
from: what the world's switch of board reads once (see [Boards](../docs/boards.md#the-worlds-switch-to-earth-2)).
`npm run map:carry -- earth-1 earth-2` writes it (`scripts/generateCarry.ts`, `scripts/map/carry.mjs`), from the two
boards' runs alone; `make map` copies it to the backend, its one reader (`embedded_board.Registry.Carryover`). The page
has no copy.

```
"CPCA" | uint32 version (1) | uint32 from's positionsHash | uint32 to's positionsHash | uint32 from's tiles |
uint32 to's tiles | to's tiles × varint
```

Each varint is a tile's parent less the parent of the tile before it (0 before the first), zigzagged, seven bits a
byte, low first. A parent is a 1-based tile id of the old board, or 0 for none. The two hashes and counts are the
manifests': the backend refuses a file whose boards are not the ones it carries.

**A new tile's parent** is the old tile whose lattice cell holds the new tile's position (`cellOf` on the old board's
lattice: the tile a click there would have taken). Where that cell is the sea, it is the nearest old tile among the
cell's six neighbours when it lies within 0.6 of the old board's spacing; otherwise none, so a new coast or island
starts blank. `earth-1` to `earth-2`, 1.38 MB: 997,267 new tiles carried and 7,415 blank; 3.85 children for an old tile
whose six neighbours are land (1,004,682 / 260,503 = 3.86); every child within 0.62 of the old spacing of its parent
(median 0.37); 66 old coast tiles with no child. `carry.test.mjs` pins the file against the generator and these bounds.

## The board's lattice

Every board is cut from one lattice: an icosahedron's 20 faces, each a triangular grid; `earth-1`'s are the points of
`THREE.IcosahedronGeometry(1, cols - 1)`. A point is `(face, i, j)`, `i` rows from the face's edge AB towards C and `j`
steps along the row.
That makes them a geodesic sphere's points, so **every point has exactly 6 neighbours, the ±1 exchanges of its
barycentric coordinates, and the 12 icosahedron corners have 5** — which is what lets the backend take adjacency
straight from the lattice rather than by searching.

**A point's canonical id** is its place in a numbering that does not depend on which face reaches it: the 12 corners,
then the `cols - 1` points inside each of the 30 edges, then each face's inside, row by row (`canonicalId` in
[`lattice.ts`](../apps/frontend/src/domain/lattice.ts)). The runs are ranges of canonical ids.

**`earth-1`'s tile ids are ranks in three's walk.** Three builds each face's triangles row by row and reaches each
point first from one of them; `walkLattice` visits the points in that order. A tile's id is its place among the tiles
in that walk, as it was when three built the map, because those are the ids the ledger holds.

**`earth-2`'s and `siege-1`'s tile ids are ranks in canonical id order**, and their points are warped and rotated: each
point's barycentric weights `(cols − i − j, j, i) / cols` warped so every cell covers about as much ground, then summed
over the face's rotated corners and normalised, only `+ − × ÷ √` in one order of operations every reader keeps (see
[Boards](../docs/boards.md#the-geometry)). A point on an edge or a corner comes out the same bits from every face it is
on, so a canonical walk may reach it from any face. The generator's scripts take a point's index in the board's order
(three's walk, or the canonical id itself) wherever they say "vertex" or "point".

**Nothing builds three's geometry any more**, not the generator nor any app: they all walk `lattice.ts`'s arithmetic
(the backoffice and the backend in a copy of their own), and three is only the reference
`scripts/map/lattice.test.mjs` checks the walk, the triangles and the neighbours against, to the bit, up to `cols`
301. Latitude is `asin(y)` and longitude `atan2(z, −x)`, as everywhere in the map scripts; it gives the oracle the
same answer as three's uvs did for every point. See [Tile positions](../apps/frontend/docs/tile-positions.md) for
why every engine rebuilds the same bits.

| | `earth-1` | `earth-2` | `siege-1` |
| --- | --- | --- | --- |
| cols | 301 | 590 | 97 |
| full sphere | 906,012 points | 3,481,002 points | 94,092 points |
| land tiles, the manifest's `tiles` | 260,503 | 1,004,682 | 27,259 |
| landmasses | 694, across 239 countries | 1,175, across 239 countries | 374, across 236 countries (Saint Pierre and Miquelon, Saint Helena and Heard Island, which no flag plays, are smaller than its cell) |
| lakes, points on a country's ground that are sea | 1,644 | 6,503 | 98 |
| seats | 34 | 15 | 52 |
| spacing, median | 25.9 km | 13.1 km | 79.4 km |

See [`apps/backend/docs/map-geography.md`](../apps/backend/docs/map-geography.md) for the backend's half.

## The terrain follows too

`npm run terrain` is the third generator, and its file is the frontend's alone, though every app carries it with the
board. It reads the board's runs, gives each tile one of eight kinds of land from the colour of the photo the game
showed until November 2026 (`apps/frontend/scripts/map/earth.jpg`, read and never served), smooths the kinds over the
lattice, and gives each sea point a band from its distance to land: one, two or three steps, or the open sea. Since the
land is the runs' own tiles, a one-tile island is land across its whole cell, and a one-tile lake is water. See the
frontend's [The terrain](../apps/frontend/docs/terrain.md).

## What is left

- **250 tiles have no neighbours at all** — single-tile islands, 26 of them seats. Anything reading adjacency has to
  have an answer for an empty neighbour set. This is not a fault: an island a tile wide is an island
  a tile wide, and there are more of them now because the small ones finally have tiles.

The antimeridian fault this file used to record — "the row carries about a quarter of the tiles it
should" — was measured at a 36-tile deficit and is gone: the audit reads 1.000 tiles per land vertex
within a degree of the dateline, the same as everywhere else. Its cause was the old land mask, not
the lattice.
