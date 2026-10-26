# Tile positions, rebuilt rather than downloaded

The tiles are the land points of `THREE.IcosahedronGeometry(1, 300)`, each point once, in the
order three makes them (`scripts/map/lattice.mjs`; see [`/map/README.md`](../../../map/README.md)).
The grid never changes, so the browser only downloads **which of its 906,012 points are tiles**:
`static/landmask-<hash>.bin`, 113 KB, 12.6 KB with brotli. It works every position out itself. The
coordinates blob it replaces is 5.2 MB, 2.44 MB with brotli.

Measured in headless Chrome on DevTools' Fast 4G, empty cache, against a local server that packs
files as Cloudflare does: the globe showed at 1.3s rather than 3.75s, and 0.63 MB had come down
before it rather than 3.06 MB.

## How

`points.ts` fetches the landmask and hands it to `latticeWorker.ts`, which runs
`rebuildTilePositions` (`domain/landmask.ts`, over `domain/lattice.ts`) and sends the positions
back as a transfer. **It is a worker because of the engines with no JIT**: 60-80 ms on V8 or
JavaScriptCore, 1.5s on JavaScriptCore with its JIT off, as in iOS Lockdown Mode — a frozen page for that
long. `loadPointGeometryData` still returns `{positions, size}`, so nothing that reads
`positions` knows where they came from.

## The positions must be three's, to the bit

A tile id is a place in that list, and the backend, the backoffice and every generator in
`scripts/` read the coordinates blob. A position one ulp off is a tile that does not line up with
the borders, the outlines or the earth texture. So `lattice.ts` does three's own arithmetic from
`PolyhedronGeometry` and nothing else:

- each lerp as `a + (c - a) * t`, normalizing as `* (1 / length)`, `length` as
  `Math.sqrt(x * x + y * y + z * z)`, and `Math.fround` into float32;
- only `+ - * /` and `Math.sqrt`, which IEEE 754 rounds the same on every engine. **Never
  `Math.hypot`**, `Math.pow` or a trig function: their last bit is the engine's choice;
- esbuild's default minify, which renames but does not reorder arithmetic.

It finds each point once by which point of the sphere it is — a corner, a step along one of the
30 edges, or a face's inside — the first time three's walk reaches it. `lattice.mjs` finds it by
`toFixed(6)` of the position. `scripts/map/lattice.test.mjs` checks that the two give the same
906,012 points in the same order, to the bit, and `landmask.test.ts` that the shipped landmask
rebuilds the shipped coordinates blob, to the bit, for every tile.

## The hash is the safety net

The landmask carries a 32-bit hash (FNV-1a over the float32 words) of the positions it was made
from. A rebuild that hashes to anything else throws `PositionsMismatchError`, and `points.ts`
**logs it with the browser's user agent and downloads the coordinates blob instead**, as before. A
worker that does not start goes the same way. That is why `static/coordinates-<hash>.bin` is still
shipped. Verified: Chrome (V8) and JavaScriptCore, with and without its JIT.

## The file

```
"CPLM" | uint32 version | uint32 detail (300) | uint32 points (906,012) | uint32 tiles | uint32 hash | ceil(points / 8) bytes
```

Little-endian; one bit per point in the lattice's order, lowest bit first. `npm run landmask`
writes it from the coordinates blob in `static/`, and fails unless every tile is a point of the
lattice, in order, to the bit. **Run it whenever the coordinates blob changes** (`npm run map`,
`npm run map:generate`); `landmask.test.ts` fails until it is. It is this app's alone and is not
in `/map`: the backend and the backoffice read the coordinates blob, and this file is made from it.
