# The tile map's geometry

`coordinates-<hash>.bin` says where every tile is and `borders-<hash>.bin` says whose ground it sits
on. Both are shared by the two apps exactly as [`/proto`](../proto) is. Neither app may keep its own
generator or its own copy of the numbers; each carries a **generated copy**, because each is built
from its own directory as its Docker build context and so cannot read anything at the repo root at
build time:

| app | copy | refreshed by |
| --- | --- | --- |
| frontend | `apps/frontend/static/<blob>`, fetched at runtime | `npm run map` |
| backend | `apps/backend/generated/map/<blob>`, embedded | `make map` |

All three copies of each blob are committed and byte-identical, so git stores the bytes once.

The files are **content-addressed** — the name carries the first 8 hex of its sha256 — which is what
makes a stale copy visible by eye rather than by symptom.

## One question, one answer

**A tile exists exactly where a country does.** `apps/frontend/scripts/map/ground.mjs` is the one
oracle: `groundOf(lon, lat)` answers a country code or `SEA`, from Natural Earth 1:50m, and
`npm run map:generate` writes both blobs from that one answer. So "is this sea or land" and "which
country is this" cannot disagree, and **a tile in no country cannot be built**.

That used to be two questions asked of two datasets — a greyscale land mask decided which lattice
vertices became tiles, and Natural Earth decided which country each tile was in. They disagreed
about 6,000 of the 906,012 lattice vertices, all of them on coasts and islands, which is exactly
where anyone looks: 2,093 tiles floating on open water with no border drawn round them, 3,729 islands
drawn in the globe's texture with nothing to click, and the Antarctic ice shelves missing whole.

Three things follow from having one oracle, and they are worth knowing before changing it:

- **The Natural Earth tag is pinned** (`NATURAL_EARTH_TAG` in `scripts/map/naturalEarth.mjs`,
  `v5.1.2`). These files decide where every tile is, so two runs a year apart have to produce the
  same map. Moving the pin renumbers every tile; see below.
- **The ice shelves are ground, and they are Antarctica.**
  `ne_50m_antarctic_ice_shelves_polys` is a file of its own and is inside no country polygon, so
  before this Ross and Ronne were tileless holes in a continent the texture paints solid white.
  They are 2,633 tiles, which is about a tenth of `aq`.
- **The globe's texture follows, it does not decide.** A 4096×2048 photo blends a one-tile island
  into open water, so it can never be an authority. `npm run earth` cuts it *from* the tile field
  instead — see below — and `npm run map:audit` reports what is left but enforces nothing about it.

```bash
cd apps/frontend && npm run map:generate   # both blobs, from the oracle
cd apps/frontend && npm run map:audit      # the three sources against each other
```

The audit fails when a fault is worse than `apps/frontend/scripts/map/audit.baseline.json`. Today
every fault between the tile set and the oracle is zero, and the baseline says so, so any drift is
a failure rather than a number nobody reads.

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

What the migration cannot give back: a take on a tile the new map does not have is deleted, because
leaving it would let a revert write to an id that now means other ground.

**The remap is computed against the blob currently in `/map`**, so run it with `--remap` on the
first run. A second run over its own output would write an identity migration.

**The rest of the checklist**, which the generator prints:

```bash
cd apps/backend && make map          # copy both blobs, then commit all three copies of each
cd apps/frontend && npm run borderLines   # traced from both blobs, stale until it is run
cd apps/frontend && npm run earth         # the globe's texture, cut from the new tile field
cd apps/frontend && npm run map:audit -- --save
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
The uvs are dead weight — neither shader reads them any more — and dropping them would take ~2 MB
off a 5 MB download, but it touches the encoder, both decoders, the generator and the tests.

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
[`apps/frontend/CLAUDE.md`](../apps/frontend/CLAUDE.md), "The countries' outlines".

## The grid is a regular honeycomb

The tiles are the land vertices of `THREE.IcosahedronGeometry(1, 300)`, deduplicated by position.
That makes them a geodesic sphere's vertices, so **every interior vertex has exactly 6 neighbours**
and the 12 icosahedron corners have 5 — which is what lets the backend compute adjacency exactly
rather than by searching. See [apps/backend/CLAUDE.md](../apps/backend/CLAUDE.md), "Map geography".

| | |
| --- | --- |
| detail | 300, so `cols = 301` |
| full sphere | `10 * 301² + 2` = 906,012 vertices |
| land tiles | 262,119, which is `gameMap.maxIndex` |
| landmasses | 658, across 205 countries |
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

`cover` is the tile field at full resolution, so a one-tile island is corrected all the way, and
`opinion` is what the photo's own colour already says — so the correction is **zero wherever the two
already agree**, which is most of the globe. The two colours are the photo's own local averages over
the land and over the water, so nothing is repainted in a palette somebody chose. It took the land
the photo draws as water from 8,599 lattice vertices to 1,972.

`static/earth/earth-source.jpg` is the mosaic, kept in the repo and **not deployed**;
`static/earth/earth-<hash>.jpg` is what the globe loads.

## What is left

- **225 tiles have no neighbours at all** — single-tile islands. Anything reading adjacency has to
  have an answer for an empty neighbour set. This is not a fault: an island a tile wide is an island
  a tile wide, and there are more of them now because the small ones finally have tiles.
- **1,972 land vertices are still drawn as water, and 1,171 water vertices as land.** What is left is
  where the photo gives the correction nothing to work with — land and water the same colour under
  cloud or on an ice shelf — and where the audit's own "does this pixel look blue" is stricter than
  an eye is. `npm run map:audit` is where those numbers live.

The antimeridian fault this file used to record — "the row carries about a quarter of the tiles it
should" — was measured at a 36-tile deficit and is gone: the audit reads 1.000 tiles per land vertex
within a degree of the dateline, the same as everywhere else. Its cause was the old land mask, not
the lattice; the uvs are the equirectangular projection of their own positions to within 0.001°.
