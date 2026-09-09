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
  `previousCountry` an event reports. It also holds the third source, **your own
  clicks, painted before the server has agreed to them** — see [Rolling back a
  refused click](#rolling-back-a-refused-click).
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

### `src/backends/` — three contracts, one transport

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

The backend refuses a click in three ways, and `clickTile` translates all of
them into errors declared beside the interfaces — the first two in `backend.ts`,
the third in `session.ts` — so `globe.ts` recognises them without knowing what a
Connect code is:

- `resource_exhausted` → `RateLimitedError`. The per-IP token bucket is spent.
- `permission_denied` → `VPNBlockedError`. The address is in the backend's VPN
  and proxy blocklist.
- `unauthenticated` → `SessionUnavailableError`, **and only after a retry** —
  see [Sessions](#sessions).

They are separate classes rather than one with a field because the dialogs give
different advice: ease off for a second, turn the VPN off, or reload and unblock
the challenge. Everything else is a transport fault and still reaches the
console.

`FakeBackend` reproduces all three, so every dialog is reachable in dev: it
enforces the same bucket with the backend's defaults, and takes `vpnBlocked` and
`sessionUnavailable` options that refuse every click (there is no address and no
widget there to judge). Its own simulated traffic bypasses all of them, standing
in for other players rather than for this one.

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

### Sessions

The backend gates `Click` on a token it minted, and refuses one that carries
none. `session.ts` declares the contract — `SessionProvider`, the
`X-Session-Token` header name, `SessionUnavailableError`, and `NoSession` —
and `turnstileSession.ts` implements it in two halves that are worth keeping
apart:

- **`SessionClient`** holds one session and mints another when it is close to
  lapsing. No DOM, no script tag, no network of its own — which is why all of
  its behaviour is under test.
- **`turnstileAttester`** is the half that touches Cloudflare: it loads
  `challenges.cloudflare.com/turnstile/v0/api.js` on first use, renders a widget,
  resolves with its token and removes it again.

**A fresh widget per attestation**, not one reset between uses. Turnstile tokens
are redeemed exactly once, and a widget that is created and destroyed has no
lifecycle left to get wrong.

**`appearance: "interaction-only"`** — the widget draws nothing for almost every
visitor. `.turnstile-host` in `index.css` is where it would appear if Turnstile
decides this one has to tick a box; it is `pointer-events: none` so an empty host
never swallows a click meant for the globe.

**Concurrent clicks share one mint.** A page load fires a flurry, and without
coalescing the first second of play would spend the whole per-IP mint budget on
one Turnstile round trip per click.

**Minting is refreshed a minute before the server would stop accepting the
token**, rather than after a refusal — a token that lapses between the check and
the server reading it costs a round trip the player can feel.

**A refused click is retried once against a freshly minted session, silently.**
A token that lapsed mid-session, or one bound to an address that changed when a
phone moved onto cellular, is not worth a dialog. The retry is not a loop: a
second `unauthenticated` is reported as `SessionUnavailableError` and raises
`SessionUnavailableModal`.

**Every failure inside the session client leaves as `SessionUnavailableError`.**
A refused mint answers `permission_denied`, which is also what a VPN-blocked
click answers — left bare it would reach the dialog telling the player to turn
off a VPN they may not be using.

**`VITE_TURNSTILE_SITEKEY` is what switches this on.** Unset, `main.tsx` wires
`NoSession` and the client sends no header, which is what a local backend with
`session.enabled: false` expects. The sitekey is public — it is read off the
page — and useless without the secret, which only the backend holds. A server
that *enforces* sessions refuses every click from a build with no sitekey:
the two are configured together.

**It is a *build* variable, and Vite inlines it.** Setting it on the Workers
project changes nothing until a build runs: with it unset, `import.meta.env
.VITE_TURNSTILE_SITEKEY` folds to `undefined`, the `SessionClient` branch in
`main.tsx` becomes dead code, and Rollup drops `turnstileSession.ts` wholesale.
So a bundle built without it contains neither the sitekey nor the Turnstile
script URL — while still containing `X-Session-Token`, which comes from
`planetBackend.ts` and ships either way. That combination is the signature of a
stale build, not of a missing variable.

**The project only rebuilds on changes under `apps/frontend/`**, which is its
build watch path. A change to `deploy/` or `apps/backend/` that turns sessions
on server-side will therefore *not* ship a frontend that can mint one. Grep the
deployed bundle rather than trusting the dashboard:

```bash
B=$(curl -s https://clickplanet.lol | grep -oE '/assets/index-[A-Za-z0-9_-]+\.js' | head -1)
curl -s "https://clickplanet.lol$B" | grep -c 'challenges.cloudflare.com/turnstile'
```

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
   A refused click is taken back off the map and raises a flag in `useGlobe` that
   `Viewer` renders as `RateLimitModal`, `VPNBlockedModal` or
   `SessionUnavailableModal`. The globe reports every refused click,
   so each flag is a boolean and not a queue — a burst is one thing to say, once.
   `reportClickFailure` in `globe.ts` is the four-way branch that picks which,
   split out of the click handler because it is the one piece of that handler
   worth testing: sending a refusal to the wrong dialog leaves a working page
   giving the wrong advice.

   Dismissing `VPNBlockedModal` only closes it. Unlike a spent bucket, that
   refusal does not clear on its own — the player has to change network — so the
   next click raises it again.

### Rolling back a refused click

A click is painted before the server has agreed to it, and the server refuses
plenty of them — a spent bucket, a blocked VPN, a session it would not mint. The
paint has to come back off, or a throttled player spends the rest of the session
looking at tiles nobody gave them, with a leaderboard counting them.

`TileOwnership.applyOptimistic` paints and hands back a **claim**; `rollback`
takes that claim and returns the tile to what it would hold had the click never
happened. `globe.ts` calls it from the same `catch` that raises the dialog, and
feeds what comes back through `applyChanges` — the same path a batch or a live
update takes, so the repaint and the re-rank need no special case.

What makes this more than an undo is everything that can happen to a tile while
a click is in flight, and the rule is the same in every case: **a rollback only
ever takes back what that click itself painted.**

- **The server settled the tile** — its echo, or someone else's claim, arrived
  first. `applyUpdates` drops the claim, and the refusal that lands afterwards
  changes nothing. A late refusal of a click the server did in fact record
  therefore costs nothing either.
- **A later click on the same tile is still in flight.** It owns the paint, so
  rolling back the earlier one leaves the map alone. Refuse them all and the
  tile ends up where it started, whatever order the refusals arrive in — each
  claim remembers what it painted, so it falls back to the newest one left.
- **A batch landed while the click was in flight.** It cannot paint over the
  optimistic claim, but it is what the rollback restores instead of the stale
  value from before the click — unless a live update had already claimed that
  tile, which still wins permanently.
- The tile's claimed-live flag is restored too, so a tile whose only claim was
  rolled back accepts a later batch again rather than staying blank.

`OwnerChange.country` is `string | undefined` for this: `undefined` is a tile
going back to unowned, which `TileField` writes as a zero-sized atlas region —
what the fragment shader already draws as an unclaimed tile.

Rolling back on *any* failure, including a transport fault, is deliberate: if
the click did land and only the response was lost, the websocket echo repaints
it, and if the echo arrives first the rollback is already a no-op.

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
- [`session/v1/session.proto`](../../proto/session/v1/session.proto) —
  `CreateSessionRequest`, `CreateSessionResponse`

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
