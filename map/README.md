# The tile map's geometry

`coordinates-<hash>.bin` is the **single source of truth** for where every tile is, shared by both
apps exactly as [`/proto`](../proto) is. Neither app may keep its own generator or its own copy of
the numbers; each carries a **generated copy** of this file, because each is built from its own
directory as its Docker build context and so cannot read anything at the repo root at build time:

| app | copy | refreshed by |
| --- | --- | --- |
| frontend | `apps/frontend/static/coordinates-<hash>.bin`, fetched at runtime | `npm run map` |
| backend | `apps/backend/generated/map/coordinates-<hash>.bin`, embedded | `make map` |

Both copies are committed and byte-identical to this one, so git stores the bytes once.

The file is **content-addressed** — the name carries the first 8 hex of its sha256 — which is what
makes a stale copy visible by eye rather than by symptom. Regenerate it with the frontend's
`npm run coordinates <detail> <mapFilePath> [threshold]`, which writes here and then syncs
`static/`; run the backend's `make map` afterwards and commit all three.

**Regenerating renumbers every tile.** Tile ids are implicit in array position, so a new blob moves
every player's territory and invalidates the snapshot on disk. It has not been done, and the two
known faults below are not worth doing it for.

## Format

Little-endian, mirrored by [`coordinatesBinary.ts`](../apps/frontend/src/app/viewer/coordinatesBinary.ts)
and [`coordinates.go`](../apps/backend/internal/planet/internal/adapters/secondary/geodesic_map/coordinates.go):

```
"CPCO" | uint32 version | uint32 tile count N | N*3 f32 positions | N*2 f32 uvs
```

The 12-byte header keeps the float sections 4-byte aligned so the browser can take zero-copy views.
The uvs are dead weight — neither shader reads them any more — and dropping them would take ~2 MB
off a 4.9 MB download, but it touches the encoder, both decoders, three scripts and the tests.

**Positions are 0-indexed by array position; wire tile ids are 1-based.** Tile id = array index + 1.
That is what `in_memory_tile_checker` (`tile > 0 && tile <= maxIndex`), `memory_tile_storage`
(slot 0 unused) and the frontend's GPU picking (`integerToColor(i + 1)`) all agree on.

## The grid is a regular honeycomb

The tiles are the land vertices of `THREE.IcosahedronGeometry(1, 300)`, deduplicated by position.
That makes them a geodesic sphere's vertices, so **every interior vertex has exactly 6 neighbours**
and the 12 icosahedron corners have 5 — which is what lets the backend compute adjacency exactly
rather than by searching. See [apps/backend/CLAUDE.md](../apps/backend/CLAUDE.md), "Map geography".

| | |
| --- | --- |
| detail | 300, so `cols = 301` |
| full sphere | `10 * 301² + 2` = 906,012 vertices |
| land tiles | 257,948, which is `gameMap.maxIndex` |
| spacing | ~25 km, ~563 km² per tile |

`detail` is not recorded in the generator's output. It was recovered from this file and then
checked: scaling each tile's barycentric coordinates within its icosahedron face by `cols` lands on
integers with a worst error of 1e-5 — float32's own precision — at `cols = 301`, and gives errors
around 0.49, i.e. noise, at every other `cols` in 280..330. The backend re-runs that check on every
boot and refuses to start if it fails.

## Known faults

Both are inherited from how this file was generated and are **documented rather than fixed**,
because fixing them means regenerating:

- the **antimeridian row carries about a quarter of the tiles it should**, so neighbourhoods near
  the dateline are lopsided
- **2,523 tiles fall outside every country**
- **186 tiles have no neighbours at all** — single-tile islands. Anything reading adjacency has to
  have an answer for an empty neighbour set.
