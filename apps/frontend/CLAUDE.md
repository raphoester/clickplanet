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
npm run map        # Copy the shared /map coordinates blob into static/ (see "Static assets")
npm run borders    # Resolve every tile to a landmass (see "The zoomed-out view")
npm run flagFit    # Work out which flags stretch, and where each one is cropped
npm run mobile     # Screenshot/inspect a URL as a phone (see "Debugging mobile layout")
```

`.github/workflows/check-frontend.yml` runs lint, build and tests on every PR
touching this app.

**`npm run dev` cannot reach the production API.** `api.clickplanet.lol` sends
`access-control-allow-origin: https://clickplanet.lol` and nothing else, so the
browser blocks every request from `localhost`. Point `VITE_API_BASE_URL` at a
local backend, or run `VITE_FAKE_BACKEND=1 npm run dev` to play against
`FakeBackend` and `FakeChatBackend` — the fake serves a full map, simulates other
players at a few clicks a second, and reproduces every refusal the chat can show.
The switch is gated on `import.meta.env.DEV` as well, because an unset `VITE_*`
variable is **not** folded away in a build: without the `DEV` check both fakes
ship in the production bundle.

In fake mode the console has two commands: `giveBomb()` arms a bomb as if a box
holding one had just been caught, and `fakeBackend.botBomb(tile, "fr")` drops
somebody else's.

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
- `tileDeltas.ts` — what the leaderboard floats beside a tile count as "+3" or
  "-2". `takeInChanges` folds one board into the badges already up, so a country
  on a run keeps one badge counting up instead of flashing "+1" three times, and
  a country that wins a tile back and loses it again drops its badge rather than
  reading "+0". `expireBadges` retires one `DELTA_HOLD_MS` after its last tile.
  **Both hand back the map they were given when nothing moved**, so a quiet
  leaderboard costs no render.
- `countries.ts` — the country list, keyed by code.
- `chatLog.ts` — `addMessages`, the merge of the history fetch and the live
  stream into one bounded list. **The two sources overlap**: your own message
  arrives twice (the `SendMessage` response and the broadcast that follows), and
  history is fetched after the stream is already open, so it is deduplicated on
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
- `shareCard.ts` — everything about a shared image that is decided before a
  pixel is drawn: the `?c=<code>` link, the text that rides with it, the line
  under the flag, and the size the card comes out at. See [Sharing the
  globe](#sharing-the-globe).
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

`transport.ts` holds what both contexts need and neither owns: `retrying`,
`NO_TIMEOUT`, and `openStream`, which follows a server-streaming RPC and reopens
it with a capped exponential backoff. It is generic over the message type and
knows nothing about what it carries, so each context keeps its own mapping —
`PlanetBackend.listenForUpdates` and `ChatServiceBackend.listenForMessages` are
both a few lines over it.

**One stream per API, carrying an envelope.** `ClickService.ListenForEvents`
sends `PlanetEvent` and `ChatService.ListenForEvents` sends `ChatEvent`, each a
`oneof`. `updateOf` and `messageOf` unwrap the case each context cares about and
**return undefined for everything else** — heartbeats, and any case this build
does not know, which reads as an unset `oneof`. That is what lets the backend
add an event type without a second stream and without breaking a deployed
client, so a new live feature is a new case rather than a new connection.

**Heartbeats are why a quiet stream survives.** Cloudflare cuts a silent
response after ~125s with a 524 — measured, not guessed — so the server sends an
empty heartbeat case every 30s. `openStream` resets its backoff on *any*
message, heartbeats included, so a healthy quiet stream is never mistaken for a
failing one.

**A stream is a one-shot async iterable.** It ends on a dropped connection, a
restarted server or a proxy timeout, and nothing reopens it — Connect carries no
reconnect, which is the whole reason `openStream` exists. The backoff resets on
a received message rather than on connect, because a connection is only known to
work once something has come down it.

**`NO_TIMEOUT` is load-bearing, not decoration.** `main.tsx` builds the clients
with `timeoutMs: 2000`, and connect-web applies `defaultTimeoutMs` to a stream
exactly as to a unary call — so without passing `timeoutMs: NO_TIMEOUT` on the
call, every live feed would die two seconds in and reconnect forever. Anything
`<= 0` means no timeout.

`planetBackend.ts` is production, `fakeBackend.ts` is for development, and the
active one is wired in `main.tsx`. Both expose `close()`, and both implement the
fourth contract in `clickBudget.ts` — see [The click budget](#the-click-budget).

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

They are separate classes rather than one with a field because each one says
something different: ease off for a second (the click meter shakes — no dialog),
turn the VPN off, or reload and unblock the challenge. Everything else is a transport fault and still reaches the
console.

`FakeBackend` reproduces all three, so every refusal is reachable in dev: it
enforces the same bucket with the backend's defaults, and takes `vpnBlocked` and
`sessionUnavailable` options that refuse every click (there is no address and no
widget there to judge). Its own simulated traffic bypasses all of them, standing
in for other players rather than for this one.

#### The click budget

`clickBudget.ts` is the fourth contract: `ClickBudgetSource`, which reports how
many clicks the server will still take. **The count is the server's and nothing
else's** — a bucket kept here would drift within seconds, since it cannot see
the clicks the same address makes from another tab and its idea of when a click
was spent is a round trip out of date.

Nothing polls for it either. The server sends a *reading plus its policy* —
`tokens`, `capacity`, `refill_per_second` — and `tokensAt` replays the same
refill between two readings, so the counter is smooth at 60fps over about one
message per click. Every click answer re-anchors it, which is why the error can
never accumulate: it is exact again the moment the player does the thing the
counter is about.

`PlanetBackend` learns it three ways — `GetBudget` once at load, `ClickResponse
.budget` on every accepted click, and a **connect error detail** on a refused
one, which is the reading that matters most. It also subtracts its own clicks in
flight, so the counter only ever *under*-promises: a counter that says 1 and is
refused is a bug the player sees, and one that says 0 and works is a click they
still get.

**The reading is priced for one country.** The server charges more tokens per
click the more of the map a country holds, and sends the budget already divided
into clicks, with the price beside it (`ClickBudget.price`). `useClickBudget`
calls `priceFor(country)` whenever the selected country changes, and
`PlanetBackend` drops any reading priced for a country other than that one. The
meter only explains the price (`domain/clickPrice.ts`): it says nothing at the
plain rate unless the country is within 80% of the first step.

A server that reports nothing — no throttle, or one too old for the call —
leaves the counter hidden rather than showing a made-up allowance, so this ships
ahead of the backend. `FakeBackend` implements the same interface off its own
bucket, so the counter is live in dev.

`listenForUpdates` goes through `openStream`, which reopens with a capped
exponential backoff. It is the only source of live changes, so a drop that is not
retried freezes the globe until a reload.

### Live chat

The client for the backend's second bounded context: `chat.ts` declares
`ChatSender`, `ChatHistoryGetter` and `ChatListener` (plus `ChatBackend`, the
three together), `chatBackend.ts` implements them against `/chat.v1.ChatService/`
alone — `SendMessage`, `GetHistory` and the `ListenForEvents` stream — and
`fakeChatBackend.ts` is the dev stand-in. `ChatPanel` docks
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
decides this one has to tick a box.

**Turnstile draws the checkbox inside a closed shadow root**, so no selector on
this page reaches it and no rule of ours styles it. That makes
`pointer-events: none` on the host unusable, however tempting: the property
*inherits* across the shadow boundary, and the `.turnstile-host iframe` rule
that would give it back matches nothing. A challenge styled that way is painted
on screen and passes every click straight through to the globe behind it — a
player who is asked to tick a box that cannot be ticked, and so cannot play.
Measured on the deployed site, not deduced.

Nothing is needed in its place: the host shrink-wraps the widget, and Turnstile
renders a **0x0** box while it is not challenging, so an idle host has no area
to swallow a click with. Its `z-index` is above every other layer — the chat
sheet is bottom-centre on a phone, exactly where the widget appears, and a
challenge is the one thing on the page that has to be answerable.

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
- `useLeaderboardFeed.ts` — the one place React hears about the board, and
  **it samples rather than follows**. See [Sampling the
  leaderboard](#sampling-the-leaderboard).
- `tileField.ts` — owns both point clouds and the two attributes that change at
  runtime (`regionVector`, `hover`). Mutates them in place and reports only the
  changed ranges. Do not replace these attributes: doing so makes the renderer
  rebuild the whole GPU buffer instead of patching it.
- `gpuPicking.ts` — `GpuPicker`. Persistent 1×1 render target and scene;
  `setViewOffset` narrows the projection to the pixel under the cursor. Picks
  are resolved once per frame, not once per mousemove — each one ends in a
  synchronous GPU read that stalls the pipeline.
- `points.ts` — fetches and decodes the tile coordinates blob.
- `capture.ts` — `readDrawingBuffer`, the frame the player is looking at. **It
  only works inside the render loop**; see [Sharing the
  globe](#sharing-the-globe).
- `viewport.ts` — `layoutViewport()`, the size the canvas is set to. **Never
  size the renderer from `window.innerWidth`**: on iOS Safari that follows the
  *visual* viewport, so a pinch fires a `resize` reporting the zoomed-in width,
  the canvas shrinks to it, and releasing the zoom fires nothing that would
  widen it again — the globe is left short of the right edge with a black band
  beside it for the rest of the session.
- `atlas.ts` / `atlasAsset.ts` — country code → sprite region, and the generated
  atlas URL and size.
- `borderField.ts` / `bordersAsset.ts` — the zoomed-out view: tile → landmass,
  who holds how much of each, and the per-landmass table the vertex shader reads.
  See [The zoomed-out view](#the-zoomed-out-view).
- `pointSize.ts` — how big a tile is drawn, and the single schedule that hands
  the frame from the painted flag to the tiles.
- `zoom.ts` — how far the camera may pull back and push in. The camera is
  orthographic against a globe of radius 1, so `zoom` reads as the share of the
  viewport's height the globe fills: it opens at 1, edge to edge, and pulls back
  to 0.5, the whole sphere with sky around it. The idle spin runs at or below
  the opening zoom and stops once the view is pushed in past it.
- `stars.ts` — the sky behind the globe, drawn as a **pass of its own**. Its
  camera borrows the main camera's orientation and nothing else, so the sky
  turns with the view and holds still through a zoom. Stars in the main scene
  would do neither: that camera is orthographic, so its box frustum would clip
  them to a tube around the globe, and `camera.zoom` would fan them out across
  the screen on the way in. Its scene is not the one `disposeScene` walks, so
  `startAnimation`'s `stop()` disposes it by hand.
- `shaders/` — GLSL for the display, picking and star passes.

### The zoomed-out view

Zoomed out, a tile is about a pixel and a half sampling a 100px flag out of one
shared 1300×1232 atlas, so the GPU picks a mip level that is the whole atlas
averaged and every country comes out the same mud. Turning mipmaps off swaps mud
for sparkle. Neither end works, so from far enough away the globe is drawn from
something coarser than a tile.

Every tile resolves **offline** to a *landmass*: a country's tiles split into the
separate pieces of land they actually form (Natural Earth 1:50m, 189 countries →
630 landmasses). Neither borders nor tiles move, so `npm run borders` writes the
whole table once and nothing recomputes it at runtime. A landmass rather than a
country because a flag belongs to a piece of ground — one frame spanning mainland
France, Corsica, Guiana and Réunion would stretch the tricolour across half the
planet and paint nothing recognisable anywhere.

Each landmass flies the flag of whoever holds most of it, painted onto the sphere
with distance measured **along the surface**, so it bends with the globe and is
cropped by its own coastline. `BorderField` keeps the running count per landmass
and writes one row per landmass into a `DataTexture` the vertex shader reads.

**Opacity is the leader's share, and the curve it goes through is not a free
knob.** Zoomed in, that share is already on screen as the fraction of discs
wearing the holder's flag, so the tiles show `share * 0.7` of ink no matter what;
the painted flag shows `share^contrast * 0.94`. Bend the curve and the summary
becomes fainter than the tiles it hands over to, and a country gets *brighter* as
you zoom into it — measured at 5x for Sudan at contrast 3. `borderField.test.ts`
pins this.

**One schedule owns the whole handover** — `coarseHandover` in `pointSize.ts`
drives the flag fading out, the tiles fading in, and the disc widening being
undone. They only work together: the flag reaches the ground only through the
discs, so while it is painted they must cover the ground (circles on this hex
lattice cover it at 1.155x the spacing), and a tile you are about to aim at must
not be fattened. Running them on separate schedules left a band where the flag
was painted through a lattice with holes in it. `pointSize.test.ts` pins that
too, and those tests fail if the two are split again.

`npm run flagFit` decides the rest: a flag that is only bands can be pulled to
the country's own shape and still say what it is, while one carrying a device is
cropped, anchored on the part that names it rather than on its middle. The
result is `static/countries/flagFit.json`.

Two known faults, both inherited from the coordinates blob rather than from this:
the antimeridian row carries about a quarter of the tiles it should, and 2,523
tiles fall outside every country. Regenerating the blob would fix both and
**renumber every tile** — ids are implicit in array position — moving every
player's territory, so it has not been done.

### Data flow

1. `useGlobe` calls `createGlobe`, which fetches the coordinates and borders
   blobs — in parallel, and before allocating any GPU resource, so an abandoned
   load never opens a context.
2. Ownerships are fetched in batches and fed to `TileOwnership`.
3. Live updates arrive over the `ListenForEvents` stream, batched every 100 ms, into the same
   store.
4. Whatever the store reports as changed is painted, and the leaderboard is
   re-ranked from its counts — then handed to `useLeaderboardFeed`, which
   publishes it to React twice a second rather than ten times.
5. A click paints optimistically and POSTs; the server's echo confirms it later.
   A refused click is taken back off the map. A throttled one bumps `refusals` in
   `useGlobe`, and `ClickBudgetMeter` shakes and flashes red once per bump — a
   dialog here was annoying, since a player hits the wall mid-burst and the meter
   already says why. The other two raise a flag that `Viewer` renders as
   `VPNBlockedModal` or `SessionUnavailableModal`. The globe reports every refused
   click, so those flags are booleans and not a queue — a burst is one thing to
   say, once.
   `reportClickFailure` in `globe.ts` is the four-way branch that picks which,
   split out of the click handler because it is the one piece of that handler
   worth testing: sending a refusal to the wrong dialog leaves a working page
   giving the wrong advice.

   Dismissing `VPNBlockedModal` only closes it. Unlike a spent bucket, that
   refusal does not clear on its own — the player has to change network — so the
   next click raises it again.

6. `ClickBudgetMeter` shows what is left of the bucket, top-right. It warns
   before the wall, and shakes when a click hits it. **Its shape is read off the server's policy**: one
   pip per click in the burst (one bar past 12 of them), and the partly-filled
   pip is the click being granted back, at the server's own rate. Change
   `rateLimiter.burst` on the backend and this follows with no release here.

   The refill is animated from **one CSS custom property written per frame**,
   and each pip works out its own share of it with a `clamp()`; the count, the
   colour and the aria value are written only when the whole number changes.
   `useClickBudget` therefore re-renders when the *server* says something, not
   as the bucket refills — this sits beside a WebGL scene that wants the main
   thread. Under `prefers-reduced-motion` the fill steps four times a second
   instead of gliding.

   On a phone it moves to under the folded menu: both ends of the screen are
   full-width sheets there, the menu above and the chat below.

### Sampling the leaderboard

The globe re-ranks on every batch it takes in — ten times a second, over every
country on the map — and `useLeaderboardFeed` is what stops each of those
reaching React. Rendering two hundred rows, and restarting two hundred badge
animations, ten times a second is enough to make the whole machine stutter on a
busy planet, and it is unreadable anyway: a count nobody can follow flickering
past.

**The accounting still follows every batch; only the render is sampled.**
`recordLeaderboard` folds each board into the badges in a ref — pure arithmetic
over a couple of hundred numbers, no DOM — and a `SAMPLE_MS` interval publishes
what has accumulated. That split is what makes the badges both cheap and honest:

- A country that won three tiles in half a second says **"+3" once**, rather
  than "+1" three times too fast to read.
- The initial load is told apart from live play **exactly**, which sampling the
  stream alone could not do. `createGlobe` passes `live: false` for the ~26
  batches of the initial fetch, and live updates stream in throughout, so a
  half-second window routinely holds both. Folding per batch keeps the map as it
  loads out of the badges while still counting it into the ground they are
  measured from.
- A tick with nothing to publish and nothing to retire **calls no setter at
  all** — handing React the state it already holds still costs a render before
  it bails out.

The cost is up to half a second of staleness on the tiles count, including your
own click. The tile itself paints immediately, which is the feedback that
matters; the leaderboard is not read that fast.

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
the click did land and only the response was lost, the stream's echo repaints
it, and if the echo arrives first the rollback is already a no-op.

## Bombs

A bonus box can hold a bomb (`BonusReward` kind `bomb`). The player has
`seconds` to drop it anywhere on the planet, and it clears every tile within
`radius` of where it lands — the server's call, not this client's. The pieces:
`backends/backend.ts` declares `Bomber` and `BombDrop`, `domain/blast.ts` the
timeline every screen agrees on, `domain/holdToDrop.ts` the gesture,
`viewer/blasts.ts` the drawing, and `components/BombNews.tsx` the line at the top.

**It drops on a press held still for 0.7s, never on a click.** A bomb is precious
and a click is also what ends every drag of the globe — releasing a drag over the
planet used to drop it. `HoldToDrop` is the whole rule: a press that moves more
than 6px is a drag, a press let go early is a change of mind, and only a press
held to the end drops. The aiming ring fills in clockwise while it is held. Mouse
and touch go through the same pointer events, so there is one flow and no
two-tap variant for phones. While armed, a click claims no tile, and the
`viewer-canvas--armed` class blocks the long-press callout on iOS.

**The client sends where it aimed, not a tile.** The sea has no tiles, and whether
an aim is land is decided by the server: past one tile spacing from the nearest
tile it is the sea, the bomb is spent, and everyone sees a splash (`BombDrop.tile`
undefined, nothing cleared). The aiming ring follows the sphere under the cursor
(a ray against the unit sphere), not the tile picker, which finds nothing between
tiles or over water and made a ring that followed it blink.

**Rings are meshes, not tiles.** The aiming ring and the closing "incoming" ring
are a flat quad laid on the globe with a per-pixel ring shader
(`shaders/ring/`). Drawn out of tile discs they broke up over the sea and crawled
as they moved.

**The ground effects are in the tile shader.** The shock wave that throws tiles
outward, the flash and the scorch are computed per vertex from a few uniforms per
blast (`blasts`, `blastRadii`, up to `MAX_BLASTS` at once), so a blast costs a
uniform write per frame and nothing per tile. Each blast slot also owns a flash
sprite and 140 GPU-animated debris points, allocated once and reused.

**The clear waits for the explosion.** `BombDrop.cleared` arrives in one event,
and the globe holds it for `IMPACT_DELAY` so the ground goes when the bomb hits,
not when the message lands. Two things keep that honest:

- A tile update that arrives while a clear is waiting **wins its tile**: it came
  after the blast on the wire, so the tile is taken out of the waiting clear and
  the rest of the crater still goes on impact. (Flushing the whole clear early
  instead is what made craters appear before the explosion on a busy map.)
- `PlanetBackend` batches tile updates every 100ms but delivers a blast at once,
  so it **flushes the pending batch first** — otherwise a tile taken just before
  the blast would be applied after it and repaint the crater.

A hidden tab draws no frames, so there the clear is applied immediately.

The dropper's own screen shakes on impact; nobody else's does. Under
`prefers-reduced-motion` nothing moves — no shake, no debris, no displaced tiles —
and the colours stay. A blast off screen gets the red 💥 edge pointer, the same
component as the bonus box's.

## Sharing the globe

The game's own map is the marketing material, so the globe can be photographed
and the picture taken out of the browser. `src/app/share/` holds it:
`CameraButton.tsx` takes the shot, `takePicture.ts` captures and composes it,
`drawShareCard.ts` draws the card, `SharePreview.tsx` shows it, `ShareActions
.tsx` is the row of buttons under it and `deliverShare.ts` is what they do.
`useSharePicture.ts` holds the one picture there is at a time.
`src/domain/shareCard.ts` holds everything decided before a pixel is drawn.

**The camera sits on the canvas, bottom-left, not in the menu.** The globe is
what it photographs and the card over it is not in the picture, so the button
belongs beside the subject. It is also the wrong shape for the menu's row of
actions: those are 56px slabs for things you do to the *page*, and three of them
do not fit across the card anyway — an earlier version put the share buttons
there and pushed Discord out over the edge, and moving the camera up beside the
collapse chevron did the same to the chevron. It is the click meter's pill
instead, and on a phone it clears the folded chat the way the meter clears the
menu, losing its label there: a camera needs no caption and the name stays in
`aria-label`.

**Its label never changes.** A pill anchored to a corner that rewrites itself
mid-press resizes under the cursor, and what answers the press is the preview
opening. Working is said by the button dimming.

**Pressing it opens a preview, and the preview is where the choice lives.** The
framing is the player's — the camera takes the globe at whatever angle and zoom
they left it — and that is the one thing they cannot check after the fact. A
capture that fails opens the preview too, saying so: a camera button that does
nothing visible is a button pressed again and again.

**`preserveDrawingBuffer` is deliberately off**, which is why the capture is
where it is. Setting it would have the driver keep a second copy of the buffer
for every frame of every session, permanently, to serve a button most players
press rarely or never. Instead `globe.ts` runs the read from an `afterRender`
hook inside the animation loop, in the same tick as the `render()` that filled
the buffer, and `Globe.capture()` hands back a promise that settles on the next
frame. A read one tick later comes back blank. Two consequences worth knowing:

- **Reading the canvas rather than an offscreen target is also what keeps the
  colours right.** three applies the output colour space conversion only when it
  renders to the canvas — a `WebGLRenderTarget` that is not an XR one is forced
  to `LinearSRGBColorSpace` — so the same scene drawn into a target comes back
  visibly different from what the player was offered.
- **A hidden tab has no frames**, so a capture started and then backgrounded
  sits on "Drawing…" until the tab is looked at again. It settles by itself.

**The card is composed, never screenshotted.** The DOM over the globe is a
translucent panel with a scrolling leaderboard in it; what is good to use is a
poor picture. The badge is drawn from the same numbers the menu is drawn from,
out of the same flag atlas, in hundredths of the card's shortest edge so it
reads the same on a phone in portrait as on a wide desktop.

**The card carries the mark across the top** — the same logo and wordmark the
menu header flies — with the link at the other end of that line, and the
player's badge at the bottom.

**The link is drawn into the image**, not only attached to it: a picture is what
survives being reposted. It is drawn in Oswald rather than the page's title
face, which has no lowercase — a query parameter reading `?C=PS` is a link that
does not work for whoever retypes it — and it sits on the masthead's line rather
than over the badge, so a long country name never has to share a width with it.

**Anything drawn beside a title is aligned on the capitals, not on the box.**
Luckiest Guy carries far more ascent than its capitals use, so `align-items:
center` and canvas's `textBaseline: "middle"` both leave the letters riding above
whatever is centred next to them. `titleFont.ts` holds that one fact and what it
costs; `CountryFlag` is the component that settles it for the DOM and every flag
beside a name goes through it, while the card measures the cap band off the face
at draw time and centres on that. The badge's panel is sized from the same
measurement rather than from font sizes, which is what makes its padding even —
a row measured in `px` of font is mostly leading. **A flex `margin-top` doing
this correction has to be twice the rise**, because centring applies to the
margin box; getting that wrong left the camera icon exactly half-corrected.

**The canvas is sized in CSS pixels** (`renderer.setSize` with no pixel ratio),
so a phone captures around 390×844. `cardSize` lifts that to a short edge of
720 — the globe softens a little and the flag and the counts stay crisp, which
is the half anyone reads — and caps the long edge at 2400 so a share sheet will
still take the file.

**And the card is the middle of the frame, not all of it.** 390×844 is a 1:2.2
column that every timeline either shows as a sliver or crops for you;
`cropToAspect` brings the shape back inside 9:16 … 16:9 first, centred, because
the globe is centred — the camera looks at the origin. The portrait limit is the
loosest of the standard shapes on purpose: at rest the sphere's diameter is the
viewport's *height*, so on a phone it is already wider than the screen and every
row cropped is a row of planet. The tighter 4:5 a feed prefers takes nearly half
the frame.

**The player picks the delivery; nothing picks for them.** `deliveriesOffered`
reads once, when the menu mounts, and puts one button on screen per way this
browser actually has of letting go of the file — a phone gets `Share`, a desktop
gets `Copy` and `Save`. It is never empty: a download needs nothing of the
browser.

This replaced a ladder that tried the three in turn and reported whichever
answered first, and both halves of that went wrong in the first minute of real
use. The desktop share sheet reported success and posted the text with no
picture. A clipboard write the browser had quietly refused came back as a
download, so the button said "Saved!" on a press that asked to copy. **What is
offered is only what this browser can do, and what happens is only what was
asked for** — which is also why a refused copy reports a failure instead of
falling through to the download sitting an inch away from it.

**The share sheet is a phone's button**, and that judgement is the one thing
there that is not a feature test. On a phone the sheet *is* how you send a file
somewhere and it carries one; on a desktop it is a shim over the OS share
services, and Chrome on macOS answers `canShare({files})` true for services that
then keep the text and drop the image — measured, into Telegram, which posted
the sentence and no picture. `canShare` is asked with a stand-in `File`, since
it judges the kind of thing it is handed rather than the bytes. A share sheet
the player *cancels* delivers nothing, rather than handing them a file seconds
after they said no.

**The clipboard carries the picture and nothing else**, for the same reason
wearing a third hat: handed an item with `image/png` *and* `text/plain`, a chat
window pastes the sentence. That is why the link is drawn into the image — it
has nowhere else it has to be, so the copy has one fewer way to be misunderstood.

**The delivery buttons live in the preview's footer**, so the picture is on
screen while the player picks what to do with it. `Modal` takes a `className`
for the panel — `SharePreview.css` widens it past the 360px `Modal.css` sizes a
column of text to, drops the scroll fade that would veil the bottom of the card,
and fits the picture to the room between the header and the buttons rather than
capping it in `vh`, which left a hand's width of empty panel under a portrait
card on a phone.

## Sound

`src/app/sound/` holds it. **There is no audio file**: every sound is built
from oscillators and noise with the Web Audio API at the moment it plays, in
`synths.ts`. Tuning a sound is changing numbers there.

- `soundPlayer.ts` — `createSoundPlayer`: settings check, a per-sound
  `MIN_GAP_MS` (fast clicks and a busy chat would otherwise be one long buzz),
  nothing in a hidden tab. Its context, synths and clock are injectable, so it is
  tested without audio.
- `useSound.ts` — the settings in `clickplanet-sound-settings`
  (`domain/soundSettings.ts` parses them; a sound added later starts on) and
  the one player. **`play` never changes identity** and reads the settings
  through a ref: `useGlobe` rebuilds the globe when an option changes, and a
  toggle must not do that.
- `SoundSettingsPanel.tsx` — the switches, behind the speaker button in the
  menu. Turning a sound on previews it.

**Audio is locked until a gesture.** The `AudioContext` is only created by the
first `pointerdown`/`keydown` on the window, so a bonus box or a chat message
before the player has touched the page is silent, by design of the browser.

Where each one fires: the click and the refusal in `globe.ts`'s click handler
(`reportClickFailure` returns whether the server refused — a transport fault is
not a "nope"); the box appearing and being caught at the same places the box
itself does; the bomb when its broadcast arrives, with the boom scheduled
`IMPACT_DELAY` later so it lands with the tiles, quieter for someone else's,
and a splash instead of a blast when the drop has no tile under it (the ocean);
the chat in `ChatPanel` for a message that is not yours. **Your own message is
filtered on your name as well as on `mine`**: its broadcast can arrive before
the send answer that fills `mine` in.

## Protocol Buffers

Types are defined in the monorepo-shared [`/proto`](../../proto), one package per
bounded context, and generated to `src/gen/grpc/<package>/v1/` — `*_pb.ts` for
the messages and `*_connect.ts` for the service client. `buf.gen.yaml` points at
the whole `proto` directory, so a new package needs no config change; run
`npm run proto` after changing a `.proto`.

- [`planet/v1/planet.proto`](../../proto/planet/v1/planet.proto) — `ClickRequest`,
  `ClickBudget`, `GetMapResponse`, `TileUpdate`
- [`chat/v1/chat.proto`](../../proto/chat/v1/chat.proto) — `ChatMessage`,
  `SendMessageRequest`, `GetHistoryResponse`
- [`session/v1/session.proto`](../../proto/session/v1/session.proto) —
  `CreateSessionRequest`, `CreateSessionResponse`

`ChatMessage.sentAtUnixMs` is an `int64`, which `protoc-gen-es` gives you as a
`bigint` — `chatBackend.ts` converts it at the edge so nothing above it deals in
two number types.

## Static assets

Three assets are **content-addressed**, because `public/_headers` caches
`/static/*` for a week and a regenerated file under a stable name would be
served stale. Each has a generated TS module holding its current URL — do not
edit those by hand, and do not add a `?ts=` cache-buster, which defeats the
cache entirely:

- `/static/borders-<hash>.bin` — tile → landmass and a frame per landmass,
  fetched at runtime by `borderField.ts`. URL in `bordersAsset.ts`. Regenerate
  with `npm run borders`, which needs the coordinates blob to already be in
  place — it resolves *those* tiles.
- `/static/coordinates-<hash>.bin` — tile positions, fetched at runtime by
  `points.ts`. Format in `coordinatesBinary.ts`; URL in `coordinatesAsset.ts`.
  **This one is not ours alone.** The source of truth is the monorepo-shared
  [`/map`](../../map/README.md), which the backend also builds its tile adjacency
  from; `static/` holds a generated copy, exactly as `src/gen/grpc/` holds a copy
  of the proto contract. `npm run map` re-copies it, and the generators —
  `npm run coordinates <detail> <mapFilePath> [threshold]`, or
  `npm run coordinates:convert` to rebuild from the existing JSON — write to
  `/map` first and then sync. **Run the backend's `make map` after either, and
  commit all three copies**, or the two apps disagree about what a tile id means.
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

`Leaderboard.css` has the other one: the delta badge's `animation-duration` is
set inline from `DELTA_HOLD_MS`, so the fade-out ends exactly as the badge is
dropped from the map. The keyframes are written so the badge **never dips below
where it comes to rest** — the rows are 25px and a badge that starts low lands
on the count below it.

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

Chrome's emulation does not reproduce one iOS behaviour that has bitten this
page: **Safari zooms the whole page in when a text field under 16px takes
focus**, and never zooms back out. That is why `.chat-input` is 16px — at 14px
the zoom pushed the send button off the right of the screen. Keep every `input`
here at 16px or more; the emulator will not tell you when one drops below.

`--open-menu` exists because two things sit between a fresh load and the menu:
`DonationModal` rolls a coin on **every** load (`SHOW_PROBABILITY`), and the
menu starts folded on mobile. Without it you will screenshot the donation modal
half the time and the folded header the other half.
