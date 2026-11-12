# The tile map's geometry

`coordinates-<hash>.bin` says where every tile is and `borders-<hash>.bin` says whose ground it sits
on; `capitals-<hash>.json` says which tile is each country's capital and `bridges-<hash>.json` which crossings join
two shores over the sea (see [Capitals and bridges](#capitals-and-bridges)). They are shared by the apps exactly as
[`/proto`](../proto) is. Neither app may keep its own
generator or its own copy of the numbers; each carries a **generated copy**, because each is built
from its own directory as its Docker build context and so cannot read anything at the repo root at
build time:

| app | copy | refreshed by |
| --- | --- | --- |
| frontend | `apps/frontend/static/<blob>`; the coordinates only when the rebuild below fails | `npm run map` |
| frontend | `apps/frontend/static/capitals-<hash>.json` and `bridges-<hash>.json` | `npm run map:generate`, `npm run bridges` |
| backend | `apps/backend/generated/map/<blob>`, all four, embedded | `make map` |
| backoffice | `apps/backoffice/src/gen/map/coordinates-<hash>.bin` alone, bundled by Vite | `npm run map` |

All three copies of each blob are committed and byte-identical, so git stores the bytes once.

The files are **content-addressed** — the name carries the first 8 hex of its sha256 — which is what
makes a stale copy visible by eye rather than by symptom.

## One question, one answer

**A tile exists exactly where a country does.** `apps/frontend/scripts/map/ground.mjs` is the one
oracle: `groundOf(lon, lat)` answers a country code or `SEA`, from Natural Earth 1:50m; a vertex
whose cell is mostly lake is `SEA` too; then each country too small for the lattice gets its seat
(`latticeGround` in `scripts/map/oracle.mjs`), and `npm run map:generate` writes both blobs from that
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
  same map. Moving the pin renumbers every tile; see below.
- **The ice shelves are ground, and they are Antarctica.**
  `ne_50m_antarctic_ice_shelves_polys` is a file of its own and is inside no country polygon, so
  before this Ross and Ronne were tileless holes in a continent the texture paints solid white.
  They are 2,633 tiles, which is about a tenth of `aq`.
- **A cell that is mostly lake is water.** See below.
- **Every country has at least one tile: its seat.** See below.
- **The globe's texture follows, it does not decide.** A 4096×2048 photo blends a one-tile island
  into open water, so it can never be an authority. `npm run earth` cuts it *from* the tile field
  instead — see below — and `npm run map:audit` reports what is left but enforces nothing about it.

```bash
cd apps/frontend && npm run map:generate   # both blobs, from the oracle
cd apps/frontend && npm run map:audit      # the three sources against each other
```

## Lakes are water

Natural Earth draws each country over its lakes, so the Great Lakes, Victoria, Tanganyika, Malawi, Baikal and Ladoga
were all tiles, painted over the water the photo shows. `ne_50m_lakes` is the third layer of the oracle, and
**a vertex is sea when most of its cell is lake** (`mostlyLake` in `scripts/map/ground.mjs`): 19 points over the
cell, the vertex and two rings, and more than half of them in a lake. It is the cell and not the vertex because a
lake is often narrower than a cell — a reservoir down a valley, Lake Champlain — and a vertex that falls in one would
cut water a cell wide out of the land; a cell is a fair answer for a lake as wide as one. 1,644 vertices are lake,
and an island in one stays land.

`DRY` lists the lakes that stay land: Australia's salt pans, which fill a few years a century; Kakhovka, drained in
2023; and the steppe's shallow salt lakes (Tengiz, Qusmuryn, Siletitengiz), which the photo draws as steppe. Natural
Earth draws them full, and a cell with no tile on what the globe draws as land is ground nobody can take.

## A seat for each country the lattice misses

The lattice is 25 km apart, so a country smaller than a cell — the Vatican, Monaco, Liechtenstein — or an island
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
  the texture then paints the cell as land (see below). Two of them touch a neighbour's tile, as the islands do:
  Anguilla touches Sint Maarten and the US Virgin Islands touch the British.

A seat for a country that has ground of its own, an island seat that would land on a country's ground, or two seats on
one vertex refuse the generation. **The audit fails on a flag with no ground**, except the four flags of a region
inside a country's ground (England, Scotland, Wales, Brittany: `REGIONS`), which share it.

**Every fault between the tile set and the oracle is zero, and the audit fails on any one of them.**
They are not a budget that creeps: both blobs are cut from one `groundOf` call, so a tile on open sea
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
    that is no capital, within 150 km or the generator stops. 17, for the crossings the stones do not make: the North
    and South Atlantic, Hawaii to California and Japan, the Tasman, the monsoon, Gibraltar. The points are pinned, like
    the seats: moving one moves a bridge.

  Until 2026-11-11 the reach crossed `waters-<hash>.json`, each body of water's coast: the ocean was one body, so a
  flag on any coast reached every coast on Earth, and 130 of the 239 capitals were in reach of every flag at the start.

`npm run map:audit` fails on a country with ground and no capital, a capital off its own ground, bridges that differ
from a fresh `bridgesOf`, and land no bridge reaches. The backend refuses to boot on a capital off its ground or a file of another map's
size.

## Regenerating renumbers every tile

A tile id is an index into the coordinates blob, so a regeneration moves every id after the first
tile added or removed — including on a server that is already holding players' territory. **The ids
are the only thing that moves.** A tile's position does not, so a tile that exists in both blobs can
be followed from one to the next, and `npm run map:generate -- --remap <path.up.sql>` writes the
postgres migration that does it:

```bash
cd apps/frontend
npm run map:generate -- --remap ../backend/internal/planet/internal/migrations/<ts>_remap_tiles.up.sql
```

It writes the `.down.sql` beside it. The mapping is **monotonic** — both blobs are the same lattice
in the same generation order, filtered by what each calls land — so it run-length encodes to a few
thousand rows instead of a quarter of a million. `apps/backend/internal/planet/internal/migrations
/migrations_test.go` runs it against a real postgres, both ways.

**Every column of a tile goes with it** — its flag, shields and holder, and whatever a later migration adds —
because the migration copies the row whole rather than naming columns. **A landmass id moves too**: it is a row of
the borders blob's table, and a regeneration renumbers them. The migration follows each old landmass to the new one
its tiles are on, for the same country, and moves the landmass each fortify in the ledger names (the territory card
reads the last one by it) and copies the fortify locks to the new borders blob's name (the boot reads only those).
A landmass that split goes to the piece that holds most of it — Canada keeps its lock when a lake cuts an island off
it — and one with no such piece, or whose piece is another's too, is followed to nothing: its lock is settled at boot
as any other.

What the migration cannot give back: a take on a tile the new map does not have is deleted, because
leaving it would let a revert write to an id that now means other ground.

**The remap is computed against the blob currently in `/map`**, so run it with `--remap` on the
first run. A second run over its own output would write an identity migration.

**The rest of the checklist**, which the generator prints:

```bash
cd apps/backend && make map          # copy the four files, then commit every copy
cd apps/backoffice && npm run map    # copy the coordinates blob
cd apps/frontend && npm run borderLines   # traced from both blobs, stale until it is run
cd apps/frontend && npm run landmask      # which lattice points are tiles, what the browser downloads
cd apps/frontend && npm run earth         # the globe's texture, cut from the new tile field
cd apps/frontend && npm run landmassNames # each landmass's name, keyed by the new borders blob
cd apps/frontend && npm run map:audit     # every fault should read zero
```

`gameMap.maxIndex` follows the tile count, in `apps/backend/cmd/api/example.yaml` and
`deploy/vps/backend.yaml`; the backend refuses to start on a blob that disagrees with it.

## Format

Little-endian, mirrored by [`coordinatesBinary.ts`](../apps/frontend/src/app/viewer/coordinatesBinary.ts)
and [`coordinates.go`](../apps/backend/internal/planet/internal/clicks/embedded_geodesic_map/coordinates.go):

```
"CPCO" | uint32 version | uint32 tile count N | N*3 f32 positions | N*2 f32 uvs
```

The 12-byte header keeps the float sections 4-byte aligned so the browser can take zero-copy views.
The uvs are dead weight — nothing reads them. The browser downloads this blob only when it cannot
rebuild the positions from its landmask, so dropping them would save players little for a change to
the encoder, every decoder, the generator and the tests.

**Positions are 0-indexed by array position; wire tile ids are 1-based.** Tile id = array index + 1.
That is what `clicks.Board` (`tile > 0 && tile <= maxIndex`), `inmemory_tile_storage`
(slot 0 unused) and the frontend's GPU picking (`integerToColor(i + 1)`) all agree on.

The borders blob, written by the same command and read by `borderField.ts` and `clicks.Borders`:

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

The frontend draws a black outline along the tile boundary between two countries, and keeps the
geometry for it in a blob of its own — `apps/frontend/static/borderLines-<hash>.bin`, written by
`npm run borderLines`. It is derived from both blobs here and from nothing else, so **regenerate it
after these**; it stays out of `/map` because the backend has no use for it. See
[`apps/frontend/docs/country-outlines.md`](../apps/frontend/docs/country-outlines.md).

## The grid is a regular honeycomb

The tiles are the land vertices of `THREE.IcosahedronGeometry(1, 300)`, deduplicated by position.
That makes them a geodesic sphere's vertices, so **every interior vertex has exactly 6 neighbours**
and the 12 icosahedron corners have 5 — which is what lets the backend compute adjacency exactly
rather than by searching. It is also what lets the browser rebuild every position from one bit per
vertex (`apps/frontend/static/landmask-<hash>.bin`, written by `npm run landmask` from the
coordinates blob) instead of downloading them. See
[`apps/frontend/docs/tile-positions.md`](../apps/frontend/docs/tile-positions.md). See [`apps/backend/docs/map-geography.md`](../apps/backend/docs/map-geography.md).

| | |
| --- | --- |
| detail | 300, so `cols = 301` |
| full sphere | `10 * 301² + 2` = 906,012 vertices |
| land tiles | 260,503, which is `gameMap.maxIndex` |
| landmasses | 694, across 239 countries |
| spacing | ~25 km, ~563 km² per tile |

`detail` is not recorded in the generator's output. It was recovered from this file and then
checked: scaling each tile's barycentric coordinates within its icosahedron face by `cols` lands on
integers with a worst error of 1e-5 — float32's own precision — at `cols = 301`, and gives errors
around 0.49, i.e. noise, at every other `cols` in 280..330. The backend re-runs that check on every
boot and refuses to start if it fails.

## The texture follows too

`npm run earth` is the third generator, and it is the frontend's alone — the backend has no texture,
so nothing of it comes to `/map`. It reads the coordinates blob, rasterises each tile's cell onto an
equirectangular image, and moves the satellite mosaic's coastline onto it:

```
out = photo + (cover - opinion) * (landColour - seaColour)
```

`cover` is the tile field at full resolution — a pixel is land when its nearest lattice vertex is a
tile — so a one-tile island or lake is corrected all the way, and
`opinion` is what the photo's own colour already says — so the correction is **zero wherever the two
already agree**, which is most of the globe. The two colours are the photo's own local averages over
the land and over the water, so nothing is repainted in a palette somebody chose. It took the land
the photo draws as water from 8,599 lattice vertices to 2,053, and the water it draws as land to 13.

**A block learns its land colour only from 128 pixels' worth of land it agrees about**, or takes a neighbour's
(`LAND_ENOUGH`). A block of open ocean whose only land is atolls the photo draws as water used to learn it from the
few pixels that half agreed — a blue a shade off the sea's — and leave a one-tile island looking like open water:
14 of the 28 island seats did. All but Nauru read as land now (Nauru is a dark green-blue dot), and it took the land
drawn as water across the whole globe from 2,053 to 1,903, for 0.08% more of the image moved, all of it on coasts and
islands.

`static/earth/earth-source.jpg` is the mosaic, kept in the repo and **not deployed**;
`static/earth/earth-<hash>.jpg` is what the globe loads.

## What is left

- **250 tiles have no neighbours at all** — single-tile islands, 26 of them seats. Anything reading adjacency has to
  have an answer for an empty neighbour set. This is not a fault: an island a tile wide is an island
  a tile wide, and there are more of them now because the small ones finally have tiles.
- **1,802 land vertices are still drawn as water, and 47 water vertices as land.** Most of the water drawn as land
  is a lake the photo draws dark teal — the Quebec reservoirs, Itaipú, Kariba — which reads as water to the eye and
  not to the audit's "is this pixel blue". What is left is
  where the photo gives the correction nothing to work with — land and water the same colour under
  cloud or on an ice shelf — and where the audit's own "does this pixel look blue" is stricter than
  an eye is. `npm run map:audit` is where those numbers live.

The antimeridian fault this file used to record — "the row carries about a quarter of the tiles it
should" — was measured at a 36-tile deficit and is gone: the audit reads 1.000 tiles per land vertex
within a degree of the dateline, the same as everywhere else. Its cause was the old land mask, not
the lattice; the uvs are the equirectangular projection of their own positions to within 0.001°.
