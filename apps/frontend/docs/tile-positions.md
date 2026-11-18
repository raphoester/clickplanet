# Tile positions, rebuilt from the board

The tiles are points of the board's lattice (see [`/map/README.md`](../../../map/README.md)): `earth-1`'s are the points
of `THREE.IcosahedronGeometry(1, cols - 1)`; `earth-2`'s and `siege-1`'s those of a warped and rotated icosahedral
lattice (see [Boards](../../../docs/boards.md#the-geometry)). The lattice never changes, so the browser only downloads **which of its
points are tiles**: the board's runs, `static/boards/<board>/runs-<hash>.bin`, 121 KB for `earth-1`. It works every
position out itself. There is no file of positions to fall back on, on any board.

The landmask the runs replaced was measured in headless Chrome on DevTools' Fast 4G, empty cache: the globe showed
at 1.3s rather than 3.75s with the 5.2 MB coordinates file it had replaced.

## How

`points.ts` fetches the runs of the board it is given (`loadPointGeometryData(board)`), from the table of every board
the page carries (`BOARDS` in `boardsAsset.ts`, which `npm run map` writes from each board's manifest): the game page's
is the one `GetMap` names (see [The board the page draws](board-reload.md)), the archive's, the clips' and the video's
the board of each era they draw. It hands them to `latticeWorker.ts`, which runs `rebuildTiles` (`domain/board.ts`, over `domain/lattice.ts`): it walks
the lattice in the board's order (three's walk for `earth-1`, canonical ids for the others: `walkLattice`) and keeps
the points whose canonical id is in a run, a tile's id its rank among them plus one. It sends back the positions as a
transfer, with the cells the territory layer and the pick find a pixel's tile by (`cellsOf`, built from each tile's
canonical id; see [The zoomed-out view](zoomed-out-view.md)), the chunks the page culls the tiles by (`chunksOf`; see
[The tiles out of sight are not drawn](far-side.md)) and the positions' hash. **It is a worker because of the engines
with no JIT**: 60-90 ms on V8 or JavaScriptCore, 1.5s on JavaScriptCore with its JIT off, as in iOS Lockdown Mode — a
frozen page for that long. **A worker that does not start or fails is logged, and the page rebuilds on its own
thread** from the same runs, which it copied to the worker rather than moved. `loadPointGeometryData` still returns
`{positions, size, cells}`, with the chunks, so nothing that reads `positions` knows where they came from.

## The positions must be three's, to the bit

A tile's position is where it is drawn, and the borders, the outlines and the terrain were cut from the same
positions. A position one ulp off is a tile that does not line up with them. So `lattice.ts` does three's own
arithmetic from `PolyhedronGeometry` and nothing else:

- each lerp as `a + (c - a) * t`, normalizing as `* (1 / length)`, `length` as
  `Math.sqrt(x * x + y * y + z * z)`, and `Math.fround` into float32;
- only `+ - * /` and `Math.sqrt`, which IEEE 754 rounds the same on every engine. **Never
  `Math.hypot`**, `Math.pow` or a trig function: their last bit is the engine's choice;
- esbuild's default minify, which renames but does not reorder arithmetic.

**Every copy of it holds the same rule**: the backoffice's (`src/home/board.ts`) and the backend's
(`embedded_board/lattice.go`, where each product is converted to `float64` on its own, so the compiler cannot fuse it
into the add: fused on arm64, the hash moves).

It finds each point once by its canonical id — a corner, a step along one of the 30 edges, or a face's inside — the
first time three's walk reaches it. `scripts/map/lattice.test.mjs` checks the walk against three's own geometry, to the
bit, up to `cols` 301, and `board.test.ts` that the shipped runs rebuild the positions the manifest hashes. Before the
coordinates file went, every app's rebuild was checked against it, to the bit, for every tile.

## A warped and rotated board

`earth-2` and `siege-1` take each point's barycentric weights `(cols − i − j, j, i) / cols`, warp them (`warpedWeights`:
`h_k = w_k·((1 + a·w_k) + b·(w_k·w_k)) + ((c·wa)·wb)·wc`, each over `(ha + hb) + hc`) and sum them over the face's
rotated corners, `(wa·A + wb·B) + wc·C`, normalised as three does (`positionWriter`). The manifest carries the warp and
the 36 corners; no reader knows the rotation. It is the same rule: only `+ − × ÷` and `Math.sqrt`, in that order, in
every copy, and a point on an edge or a corner comes out the same bits from every face it is on (`lattice.test.ts`).

**Every tile must find its own cell**: `cellsOf` runs each tile's position through `cellOf`, the inverse the territory
layer and the pick use, and throws on one it finds in another cell, so a wrong inverse does not load the globe rather
than paint a tile's flag on its neighbour (see [The tile under the pointer](picking.md)). It costs the rebuild, in Node
on 4 vCPUs (2026-11-18): `earth-1` 60 ms for the positions and 40 ms for the cells, `earth-2` 55 ms and 200 ms, `siege-1`
2 ms and 10 ms.

## The hash is the safety net

The manifest carries a 32-bit hash (FNV-1a over the float32 words) of the positions the board was made from. A
rebuild that hashes to anything else is **logged with the browser's user agent, and the page keeps the positions it
computed**: the ids come from the runs, so an ulp of difference moves a tile by under a metre, and there is no other
file to read. The arithmetic matched on Chrome (V8) and JavaScriptCore, with and without its JIT, when the landmask
shipped it; the rebuild from the runs was checked again in Chrome.

## The file

```
"CPRN" | uint32 version | uint32 count | count × (uint64 first, uint64 last)
```

Little-endian; ranges of canonical ids, ascending. It belongs to the board in `/map`, and every app carries a copy;
see [`/map/README.md`](../../../map/README.md#format).
