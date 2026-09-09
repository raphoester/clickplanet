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
npm run mobile     # Screenshot/inspect a URL as a phone (see "Debugging mobile layout")
```

`.github/workflows/check-frontend.yml` runs lint, build and tests on every PR
touching this app.

**`npm run dev` cannot reach the production API.** `api.clickplanet.lol` sends
`access-control-allow-origin: https://clickplanet.lol` and nothing else, so the
browser blocks every request from `localhost`. Point `VITE_API_BASE_URL` at a
local backend, or swap `PlanetBackend` for `FakeBackend` in `src/main.tsx` — the
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

- `TileClicker` — claims a tile
- `OwnershipsGetter` — batched fetch of tile → country_code, takes an
  `AbortSignal`
- `UpdatesListener` — live tile changes

`planetBackend.ts` is production, `fakeBackend.ts` is for development, and the
active one is wired in `main.tsx`. Both expose `close()`.

`planetBackend.ts` talks to the API through the generated Connect client
(`src/gen/grpc/planet/v1/planet_connect.ts`). No gRPC is involved — Connect is
an ordinary HTTP POST, or a GET for reads.

**Two transport options are load-bearing and both default to `false`:**

- `useBinaryFormat: true` — without it `GetMapResponse.tiles` travels as base64
  inside JSON, a third bigger.
- `useHttpGet: true` — without it the side-effect-free reads go out as POSTs,
  which no cache will serve.

The map arrives as `GetMapResponse`: `start_tile_id`, a `codes` table, and
`tiles`, two bytes per tile indexing into it. `bindingsOf` reads it. Tile ids
are implicit in the position, so it is far smaller than a keyed map — 516 KB
against 3.6 MB for a full map — and protobuf does all the framing, so there is
no hand-rolled encoding to keep in step with the backend.

Connect does not retry, so `retrying` wraps every call: five attempts while the
server cannot be reached, and never a retry of an answer the server chose to
send.

The backend refuses a click in two ways, and `clickTile` translates both into
errors declared in `backend.ts` beside the interfaces, so `globe.ts` recognises
them without knowing what a Connect code is:

- `resource_exhausted` → `RateLimitedError`. The per-IP token bucket is spent.
- `permission_denied` → `VPNBlockedError`. The address is in the backend's VPN
  and proxy blocklist.

They are separate classes rather than one with a field because the two dialogs
give opposite advice: ease off for a second, versus turn the VPN off. Everything
else is a transport fault and still reaches the console.

`FakeBackend` reproduces both, so each dialog is reachable in dev: it enforces
the same bucket with the backend's defaults, and takes a `vpnBlocked` option
that refuses every click (there is no address there to judge). Its own simulated
traffic bypasses both, standing in for other players rather than for this one.

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
   A refused click raises a flag in `useGlobe` that `Viewer` renders as
   `RateLimitModal` or `VPNBlockedModal`. The globe reports every refused click,
   so each flag is a boolean and not a queue — a burst is one thing to say, once.
   `reportClickFailure` in `globe.ts` is the three-way branch that picks which,
   split out of the click handler because it is the one piece of that handler
   worth testing: sending a refusal to the wrong dialog leaves a working page
   giving the wrong advice.

   Dismissing `VPNBlockedModal` only closes it. Unlike a spent bucket, that
   refusal does not clear on its own — the player has to change network — so the
   next click raises it again.

**The optimistic paint of a refused click is never rolled back.** Nothing takes
a tile back once it is painted: `TileOwnership` marks it claimed-live, the
initial batches are told to leave those alone, and the websocket only carries
changes that did happen. So a throttled player keeps looking at tiles the server
never gave them until they reload. That was already true of any failed click;
the throttle is what makes it routine rather than rare, and the VPN blocklist
makes it permanent for whoever it refuses.

## Protocol Buffers

Types are defined in the monorepo-shared [`/proto/planet/v1/planet.proto`](../../proto/planet/v1/planet.proto)
and generated to `src/gen/grpc/planet/v1/` — `planet_pb.ts` for the messages and
`planet_connect.ts` for the service client. Run `npm run proto` after changing
`.proto` files. Key messages: `ClickRequest`, `GetMapResponse`, `TileUpdate`.

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

`index.css` owns the shared boxes — `.button`, `.button-mini`, `.icon-button`,
`.menu-label` — **including their `max-width: 768px` sizes**. A component's own
CSS file says what makes that component itself (its colour, its icon spacing),
and must not restate height, padding, radius or font-size. Restating them is how
the Discord button ended up a 56px slab next to a 40px About button on mobile:
`DiscordButton.css` set its own `height`, and the mobile rule it also carried
overrode the shared mobile size. Anchors styled as buttons (`BuyMeACoffee`,
`DiscordButton`) legitimately need `display: flex` with both axes centred and
`text-decoration: none` — a `<button>` gets those for free — and nothing more.

### Debugging mobile layout

**Do not check mobile by resizing the browser window.** macOS enforces a minimum
window width well above the 768px breakpoint, so the window stays wide (`window
.innerWidth` around 1900 on this machine), the mobile branch never engages, and
you end up inspecting a narrow desktop. Editing the media conditions in the
CSSOM to force the branch tests the rules but not the device: pointer, hover and
DPR all still say desktop.

Use `npm run mobile`, which drives Chrome's real device emulation over the
DevTools Protocol — the same `Emulation.*` calls the inspector's device toolbar
makes, so viewport, DPR, touch and the iOS user agent all resolve like a phone:

```bash
npm run dev
npm run mobile -- http://localhost:5173/ --open-menu --out /tmp/shot.png \
  --eval 'JSON.stringify(getComputedStyle(document.querySelector(".button-discord")).height)'
```

It runs headless Chrome under a throwaway profile, so it never disturbs the
browser you have open. `--eval` evaluates in the page (top-level `await` works)
and prints the result — measuring boxes with `getBoundingClientRect()` beats
eyeballing a screenshot. `--headed`, `--w/--h/--dpr`, `--full` and `--settle`
cover the rest; `npm run mobile -- --help` lists them.

`--open-menu` exists because two things sit between a fresh load and the menu:
`DonationModal` rolls a coin on **every** load (`SHOW_PROBABILITY`), and the
menu starts folded on mobile. Without it you will screenshot the donation modal
half the time and the folded header the other half.
