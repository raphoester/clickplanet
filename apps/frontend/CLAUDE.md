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
fake serves a full map and simulates live updates. `FakeChatBackend` is the same
swap for `ChatServiceBackend`, and reproduces every refusal the chat can show.

A local backend is the quickest way to exercise the real chat: `cmd/api`'s
`example.yaml` has `chat.enabled: true`, and the Go server answers
`Access-Control-Allow-Origin: *` itself, so `VITE_API_BASE_URL=http://localhost:8080
npm run dev` works with no proxy in between.

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
- `chatLog.ts` — `addMessages`, the merge of the history fetch and the live
  socket into one bounded list. **The two sources overlap**: your own message
  arrives twice (the `SendMessage` response and the broadcast that follows), and
  history is fetched after the socket is already open, so it is deduplicated on
  message id, ordered on the time the server stamped, and capped at 200. It
  returns the array it was given when nothing was added, so an echo of something
  already shown costs no render. `unreadSince` counts what arrived after a given
  id, for the badge on the folded panel, and `idsSince` names those same
  messages, for highlighting them once they are on screen.
- `authorColor.ts` — `authorHue`, a stable hue per chat author. It hashes the
  identity the log actually displays, the name *and* the `author_tag`, so two
  people typing one name get two colours. **Only the hue is derived**: the
  saturation and the lightness are fixed in `ChatPanel.css`, so no hash can
  produce a colour that is unreadable against the dark panel.
- `warnOnce.ts` — for things that would otherwise warn on every frame.

### `src/backends/` — two contracts, one transport

The tile game's three interfaces are in `backend.ts`:

- `TileClicker` — claims a tile
- `OwnershipsGetter` — batched fetch of tile → country_code, takes an
  `AbortSignal`
- `UpdatesListener` — live tile changes

The chat's three are in `chat.ts` — see [Live chat](#live-chat). The two files
share no types: they are separate bounded contexts on the backend and the split
is worth keeping on this side too.

`transport.ts` holds what both wire protocols need and neither owns:
`websocketUrl(baseUrl, route)`, `retrying`, and `openSocket`, the reconnecting
websocket with the capped exponential backoff. `openSocket` takes raw frames and
knows nothing about what they carry, so each context keeps its own decoder —
`openUpdatesSocket` and `ChatServiceBackend.listenForMessages` are both three
lines over it.

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

### Live chat

The client for the backend's second bounded context: `chat.ts` declares
`ChatSender`, `ChatHistoryGetter` and `ChatListener` (plus `ChatBackend`, the
three together), `chatBackend.ts` implements them against `/chat.v1.ChatService/`
and `/ws/chat`, and `fakeChatBackend.ts` is the dev stand-in. `ChatPanel` docks
bottom-right, opposite the menu, and starts folded under 768px.

**`MAX_TEXT_LENGTH` and `MAX_NAME_LENGTH` in `chat.ts` mirror
`chat.service.max*Length` on the backend**, counted in code points as the server
counts runes. They are the composer's bounds, not a defence — the server
sanitizes and rejects on its own.

**Message text is rendered as text, never as HTML.** The backend stores it raw
and says so; React escaping is what makes that safe, so never reach for
`dangerouslySetInnerHTML` here.

**Identity is a name and a UUID the client keeps** in `clickplanet-chat-identity`
(`chatIdentity.ts`, `useChatIdentity.ts`). The server trusts neither: what
distinguishes two senders with one name is `author_tag`, the salted hash of their
address that it stamps itself. The composer asks for a name before the first
message rather than at page load — nothing else on the page requires one.

**`sendMessage` is the one call that is not wrapped in `retrying`.** A retry
after a connection dropped mid-request would post the message twice, visibly, to
everyone; a message the player can retype is the cheaper failure. `getHistory`
is retried like every other read.

The four refusals map to their own error classes and are reported **inline in the
composer, not as a modal** — unlike a refused click, the text is still in the box
and the advice is one line:

- `resource_exhausted` → `ChatRateLimitedError`, chat's own bucket (one message
  every 3s), unrelated to the click bucket
- `permission_denied` → `ChatBlockedError`, the address is in `chat.blockedIPs`
- `invalid_argument` → `ChatRejectedError`, the server refused the content
- `unimplemented` → `ChatUnavailableError`, i.e. `chat.enabled` is false and the
  route 404s

**`ChatUnavailableError` hides the panel entirely** rather than showing a broken
box: `useChat` goes to `unavailable` and `ChatPanel` renders nothing. That is
what lets this ship against a server with chat switched off, and it is also why
the composer keeps its text when a send fails — the panel may be gone next
frame.

The composer **clears the box when the send starts, not when it lands**, and puts
the text back only if the box is still empty when a refusal comes in. Clearing on
success instead wipes whatever was typed while the message was in flight, which
is exactly what a fast typer does.

#### Saying that a message landed

The globe is what the player is looking at, so an arriving message has to catch
the eye without stealing it. Four things say it, each for a different glance:

- **A message lights up as it comes into view.** `ChatPanel` holds the ids in
  `flashing` for `FLASH_MS`, matched to the `chat-message-glow` animation,
  and `ChatLog` turns that into a class. What lights up is what `idsSince`
  reports as unseen, so the same rule covers one message arriving into an open
  panel and a whole backlog the moment a folded one is unfolded.
- **Your own message never flashes.** `useChat` keeps the ids `sendMessage`
  returned in `mine` and `ChatPanel` filters them out — you know you sent it.
  This is the only reason `mine` exists.
- **A folded panel breathes.** `chat-waiting` puts the accent on the title, pops
  the unread badge (remounted on every count change, so it replays per message)
  and pulses the panel's own border and glow. It is a slow breath rather than a
  blink: this sits over a game.
- **The newest line is quoted under the folded header**, in its author's colour.
  It is `aria-hidden` — a screen reader gets the count from the badge and the
  text from the log, and the quote would only say it a third time.

**The log is never yanked down under someone who scrolled up to read.** It
auto-scrolls only while it is pinned to the bottom (`PINNED_SLACK_PX`);
otherwise a "New messages" pill appears and scrolling back down, by the pill or
by hand, dismisses it.

Every one of these animations is dropped or reduced under
`prefers-reduced-motion: reduce`, keeping the colour and losing the movement.

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

Types are defined in the monorepo-shared [`/proto`](../../proto), one package per
bounded context, and generated to `src/gen/grpc/<package>/v1/` — `*_pb.ts` for
the messages and `*_connect.ts` for the service client. `buf.gen.yaml` points at
the whole `proto` directory, so a new package needs no config change; run
`npm run proto` after changing a `.proto`.

- [`planet/v1/planet.proto`](../../proto/planet/v1/planet.proto) — `ClickRequest`,
  `GetMapResponse`, `TileUpdate`
- [`chat/v1/chat.proto`](../../proto/chat/v1/chat.proto) — `ChatMessage`,
  `SendMessageRequest`, `GetHistoryResponse`

`ChatMessage.sentAtUnixMs` is an `int64`, which `protoc-gen-es` gives you as a
`bigint` — `chatBackend.ts` converts it at the edge so nothing above it deals in
two number types.

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

`ChatPanel.css` is the one file with a custom property contract: each message
and the folded peek carry `--author-hue` from `authorHue`, and the CSS builds
the author's stripe, name colour and arrival glow out of it. Keep the hue in the
TS and the rest in the CSS — that is what stops an author's colour from being
computed in two places with two different saturations.

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
