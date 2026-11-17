# Tile positions, rebuilt from the board

The tiles are points of the board's lattice, the points of `THREE.IcosahedronGeometry(1, cols - 1)` (see
[`/map/README.md`](../../../map/README.md)). The lattice never changes, so the browser only downloads **which of its
points are tiles**: the board's runs, `static/boards/<board>/runs-<hash>.bin`, 121 KB for `earth-1`. It works every
position out itself. There is no file of positions to fall back on, on any board.

The landmask the runs replaced was measured in headless Chrome on DevTools' Fast 4G, empty cache: the globe showed
at 1.3s rather than 3.75s with the 5.2 MB coordinates file it had replaced.

## How

`points.ts` fetches the runs named by `BOARD` (`boardAsset.ts`, which `npm run map` writes from the board's manifest)
and hands them to `latticeWorker.ts`, which runs `rebuildTiles` (`domain/board.ts`, over `domain/lattice.ts`): it walks
the lattice in three's order and keeps the points whose canonical id is in a run. It sends back the positions as a
transfer, with the cells the territory layer and the pick find a pixel's tile by (`cellsOf`, built from each tile's
canonical id; see [The zoomed-out view](zoomed-out-view.md)), the chunks the page culls the tiles by (`chunksOf`; see
[The tiles out of sight are not drawn](far-side.md)) and the positions' hash. **It is a worker because of the engines
with no JIT**: 60-90 ms on V8 or JavaScriptCore, 1.5s on JavaScriptCore with its JIT off, as in iOS Lockdown Mode — a
frozen page for that long. **A worker that does not start or fails is logged, and the page rebuilds on its own
thread** from the same runs, which it copied to the worker rather than moved. `loadPointGeometryData` still returns
`{positions, size, cells}`, with the chunks, so nothing that reads `positions` knows where they came from.

## The positions must be three's, to the bit

A tile's position is where it is drawn, and the borders, the outlines and the earth texture were cut from the same
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
