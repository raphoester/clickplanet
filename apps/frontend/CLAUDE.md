# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
npm run dev        # Start Vite dev server (binds to 0.0.0.0:5173)
npm run build      # TypeScript check + Vite bundle → /dist
npm test           # Run the vitest suite once
npm run test:watch # Re-run affected tests on change
npm run lint       # ESLint check
npm run proto      # Regenerate protobuf types from the shared ../../proto/ using buf CLI
npm run atlas      # Repack the flag sprite atlas from static/countries/png100px
```

`.github/workflows/check-frontend.yml` runs lint, build and tests on every PR
touching this app.

**`npm run dev` cannot reach the production API.** `api.clickplanet.lol` sends
`access-control-allow-origin: https://clickplanet.lol` and nothing else, so the
browser blocks every request from `localhost`. Point `VITE_API_BASE_URL` at a
local backend, or swap `HTTPBackend` for `FakeBackend` in `src/main.tsx` — the
fake serves a full map and simulates live updates.

Docker (only for the local full stack in `deploy/`; production is Cloudflare Pages):
```bash
npm run dBuild   # Build the image as clickplanet-front:local
```

The image is self-contained — `nginx.conf` is baked in and mirrors what the
deployed site does: the caching rules from `public/_headers`, and the SPA
fallback that `wrangler.jsonc` sets with `not_found_handling`.

## Architecture

**Multiplayer globe conquest game** — users click tiles on a 3D globe to claim
them for a country.

The app is **plain Three.js driven imperatively**, not React Three Fiber. R3F is
not a dependency and should not be reintroduced without a reason: scene setup
here is a state machine with an order to it, and the resources it allocates have
to be released by hand.

The layering exists to keep that imperative part as small as possible:

```
domain/    no GPU, no DOM, no React — plain functions and data
backends/  the wire protocol
app/viewer/ WebGL, and the React hook that owns its lifecycle
app/       components
```

### `src/domain/` — the logic worth testing

- `tileOwnership.ts` — the authoritative tile → country map and the per-country
  counts derived from it. The initial load arrives as ~26 sequential batches
  while live updates stream in throughout, so the two sources overlap: **live
  updates win permanently**, and a tile someone has claimed is never taken back
  by a batch that was already in flight. Counts follow this map, never the
  `previousCountry` an event reports.
- `leaderboard.ts` — `rankCountries`, a pure sort over those counts. Ties break
  on country code so equally-placed rows stop swapping.
- `countries.ts` — the country list, keyed by code.
- `warnOnce.ts` — for things that would otherwise warn on every frame.

### `src/backends/` — three interfaces in `backend.ts`

- `TileClicker` — HTTP POST to claim a tile
- `OwnershipsGetter` — batched HTTP fetch of tile → country_code, takes an
  `AbortSignal`
- `UpdatesListener` — live tile changes

`httpBackend.ts` is production, `fakeBackend.ts` is for development, and the
active one is wired in `main.tsx`. Both expose `close()`.

`openUpdatesSocket` reconnects with a capped exponential backoff. It is the only
source of live changes, so a drop that is not retried freezes the globe until a
reload.

### `src/app/viewer/` — the GPU layer

- `globe.ts` — `createGlobe(options): Promise<Globe>`. Builds the scene, wires
  input and the backends to it, starts rendering. Returns `{tilesCount,
  setCountry, dispose}`.
- `useGlobe.ts` — owns one globe for the lifetime of the component. **Its effect
  must not depend on anything that changes per render**; the selected country is
  pushed into the running globe through a separate effect rather than rebuilding
  it.
- `tileField.ts` — owns both point clouds and the two attributes that change at
  runtime (`regionVector`, `hover`). Mutates them in place and reports only the
  changed ranges. Do not replace these attributes: doing so makes the renderer
  rebuild the whole GPU buffer instead of patching it.
- `gpuPicking.ts` — `GpuPicker`. Persistent 1×1 render target and scene;
  `setViewOffset` narrows the projection to the pixel under the cursor. Picks
  are resolved once per frame, not once per mousemove — each one ends in a
  synchronous GPU read that stalls the pipeline.
- `points.ts` — fetches and decodes the tile coordinates blob.
- `atlas.ts` / `atlasAsset.ts` — country code → sprite region, and the generated
  atlas URL and size.
- `shaders/` — GLSL for the display and picking passes.

### Data flow

1. `useGlobe` calls `createGlobe`, which fetches the coordinates blob before
   allocating any GPU resource, so an abandoned load never opens a context.
2. Ownerships are fetched in batches and fed to `TileOwnership`.
3. Live updates arrive over the websocket, batched every 100 ms, into the same
   store.
4. Whatever the store reports as changed is painted, and the leaderboard is
   re-ranked from its counts.
5. A click paints optimistically and POSTs; the server's echo confirms it later.

## Protocol Buffers

Types are defined in the monorepo-shared [`/proto/clicks/v1/clicks.proto`](../../proto/clicks/v1/clicks.proto)
and generated to `src/gen/grpc/clicks/v1/clicks_pb.ts`. Run `npm run proto`
after changing `.proto` files. Key messages: `ClickRequest`, `Ownerships`,
`TileUpdate`, `OwnershipBatchRequest`.

## Static assets

Two assets are **content-addressed**, because `public/_headers` caches
`/static/*` for a week and a regenerated file under a stable name would be
served stale. Each has a generated TS module holding its current URL — do not
edit those by hand, and do not add a `?ts=` cache-buster, which defeats the
cache entirely:

- `/static/coordinates-<hash>.bin` — tile positions, fetched at runtime by
  `points.ts`. Format in `coordinatesBinary.ts`; URL in `coordinatesAsset.ts`.
  Regenerate with `npm run coordinates <detail> <mapFilePath> [threshold]`, or
  rebuild the binary from the existing JSON with `npm run coordinates:convert`.
- `/static/countries/atlas-<hash>.png` — the flag sprite atlas. URL and pixel
  size in `atlasAsset.ts`. Regenerate with `npm run atlas`.

`/static/coordinates.json` is the human-readable generator output, kept in the
repo but **not deployed** (`copy:static` deletes it from `dist/static/`). So is
`/static/og-source.png`, the raw screenshot the social preview is built from.

`/static/og-image.jpg` is that preview, generated by `npm run og-image` at the
1200×627 scrapers ask for. It is letterboxed onto black rather than cropped —
the screenshot is wider than 1.91:1 with the leaderboard against one edge and
the buttons against the other. If you regenerate it at a different size, update
`og:image:width` / `og:image:height` in `index.html` to match.

`public/` holds the files that must be served as themselves rather than as the
app: `_headers`, `robots.txt` and `sitemap.xml`. Vite copies them to the root of
`dist/`, and the Workers asset handler serves a real file before
`not_found_handling` applies — **without them, every unmatched path including
`/robots.txt` answers 200 with `index.html`**, so a crawler asking for the rules
got an HTML document. That was the leading suspect for LinkedIn refusing to
fetch the preview image, though it was never proven to be the only cause.
Nothing under `/static/` may be disallowed in robots.txt; that is where scrapers
fetch the preview from.

The blob still carries a uv per tile that neither shader reads any more;
dropping it would take ~2 MB off a 4.9 MB download. It is not a breaking change
— the blob is content-addressed and its URL is bundled with the decoder, so the
two always ship together, and the existing length check already rejects any
mismatch. The work is just its breadth: encoder, decoder, three scripts, the
tests, and regenerating the asset.

## Testing

`vitest`, co-located as `*.test.ts(x)`. The suite runs on **node by default** —
most of it is domain logic with no DOM. Files that render components opt in with
a `// @vitest-environment jsdom` comment on the first line, so the rest are not
slowed down by a jsdom each.

`TileField` is testable without a GPU because `BufferAttribute` is plain arrays;
only `GpuPicker` genuinely needs a WebGL context, which is why it is kept as
thin as it is.

## Styling

Plain CSS files co-located with components. No CSS preprocessor or CSS-in-JS.
