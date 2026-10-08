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
npm run map:generate # Rewrite both shared map blobs from the ground oracle (see ../../map/README.md)
npm run map:audit  # Check the tile set, Natural Earth and the globe texture against each other
npm run quiz:generate # Rewrite the shared quiz bank (see ../../quiz/README.md). The backend embeds it; this app never does
npm run borderLines # Trace the countries' outlines onto the tile lattice (see "The countries' outlines")
npm run earth      # Cut the globe's texture from the tile field (see "The globe's texture")
npm run flagFit    # Work out which flags stretch, and where each one is cropped
npm run mobile     # Screenshot/inspect a URL as a phone (see "Debugging mobile layout")
npm run clip:fetch -- --ssh <user@host> --out replay.json  # A replay of the last 72h from production (see "Clips")
npm run clip -- --replay replay.json --count 3  # The 3 best stories in it, as vertical videos and captions
npm run clip:anthems # Vendor the anthems only the clips play (Europe's, Palestine's) into scripts/clip/anthems
npm run regions    # Rewrite each country's continent and sub-region from Natural Earth, for the clips' headlines
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

Every tile starts French in the fake, and it keeps shields on its tiles as the
server does: in its map batches, its tile updates and its bombs' struck tiles.

In fake mode the console has a few commands: `giveBomb()` puts a bomb in the
inventory as if a box holding one had just been caught, `giveBonus("refill")` does
the same for any other bonus (the fake holds charges as the server does: a refill
and a bomb at most, a pool of 8 spread clicks, a stack of 3 enclosures and 12
shields, a box adding 1 to 4, 1 to 3 and 1 to 3 of them, spread and enclose
spent only while switched on,
both at once refused, a refill refused on a full bank), `giveQuiz()` puts a quiz
banner up at once, `giveTitle("warlord")` plays the unlock of any title (the fake
wires no account, so the overlay offers Close only), and `fakeBackend.botBomb(tile, "fr")`,
`fakeBackend.botSpread(tile, "fr")` and `fakeBackend.botShield(tile, "fr")` play
somebody else's bomb, spread click or shield. `fakeBackend.shareClicks("guests")` (or
`"network"`) reads the bucket as shared, and `fakeBackend.shareClicks()` as the
player's own again. `fakeBackend.closeShapes(false)` makes every enclose click
close nothing, for the bubble that says so, and `closeShapes(true)` puts it back.
`localStorage.removeItem("clickplanet-bonus-guide")` makes the bonuses new again.

A local backend is the quickest way to exercise the real chat: `cmd/api`'s
`example.yaml` runs chat (it is always on), and the Go server answers
`Access-Control-Allow-Origin: *` itself, so `VITE_API_BASE_URL=http://localhost:8080
npm run dev` works with no proxy in between.

Docker (only for the local full stack in `deploy/`; production is Cloudflare Pages):
```bash
npm run dBuild   # Build the image as clickplanet-front:local
```

The image is self-contained — `nginx.conf` is baked in and mirrors what the
deployed site does: the caching rules from `public/_headers`, the game at
`/play` and `/auth/callback`, and the fallback to the home page that
`wrangler.jsonc` sets with `not_found_handling`.

## Pages and routes

Two pages, built by Vite as a multi-page app (`build.rollupOptions.input` in
`vite.config.ts`):

| Path | File | What |
|---|---|---|
| `/` | `index.html` | The home page. Plain HTML, and one small module for the Final Battle. |
| `/play` | `play.html` | The game. |
| `/auth/callback` | `auth/callback.html` | The game again, for the sign-in callback. |
| `/privacy`, `/terms` | `privacy.html`, `terms.html` | Plain pages, no bundle. |
| anything else | `index.html` | `not_found_handling` fallback. |

**Why `/` is not the game.** Google's brand verification, which lets anyone
sign in with Google, was refused twice: the checker runs JavaScript, saw only
the globe, and found no text that says what the app is and no visible link to
the privacy policy. A plain intro inside `#root` did not help: hidden as soon as
JavaScript ran (it flashed), the checker never saw it. So `/` is a plain page
built like `privacy.html`: what the game is, "no account needed", a Play button,
and links to both pages, the contact address and Discord. **Keep that text and
those links on it**; they are what the verification reads.

**Only a first visit sees it.** An inline script, first in the home page's
`<head>`, sends a browser that holds `COUNTRY_STORAGE_KEY` in local storage to
`/play` with `location.replace`, before anything is painted. The game writes
that key on its first render, so a browser that opened the game once never
sees the home page again. `location.search` and `location.hash` go along, so
a share link such as `/?f=de` still reaches the game with its query. Crawlers
have no storage and read the home page. `homePage.test.ts` pins the key in the
script to the constant. **`/#home` does not redirect**: it is the "Home page"
link at the bottom of the About modal.

**A first visit keeps the query too.** `home.ts` copies `location.search` onto
every Play link (`app/home/playLinks.ts`), so a newcomer who opened a share link
reaches the game with its flag rather than with their time zone's.

**The game is a real file at each path**, not a fallback. The Workers fallback
is `index.html`, the home page, so a `/auth/callback` that relied on it would
land on the home page and lose the code. The `gameRoutes` plugin in
`vite.config.ts` writes `play.html` a second time as `auth/callback.html`, and
in `npm run dev` rewrites both paths to `play.html`. The registered redirect URI
is unchanged. Paths under `/play/` have no file and get the home page: the game
has no routes of its own.

**The bundle is named `play-*.js`** now, not `index-*.js`, and it is linked from
`/play`, not from `/`. The home page links `home-*.js` (`src/home.ts`), which
shares the season client's chunk with the game.

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
  id and `idsSince` names those messages, for the sound and for highlighting
  them once they are on screen. `unseenAfter` counts the messages and
  announcements after a time, for the badge, and `newestAt` is the time of the
  newest line. `nameSentUnder` is
  the name the server gave the latest message this client sent — the only way
  it learns a guest's name.
- `authorColor.ts` — `NAME_COLORS`, the 12 a player with a username may pick
  from (`player.v1.NameColor`, one hue each), and `hueOf`, the hue of a pick.
  No pick, or a color this build does not know, has no hue. **Only the hue is
  chosen**: the saturation and the lightness are fixed in the CSS, so no pick
  can produce a colour that is unreadable against the dark panel.
  `app/chat/authorStyle.ts` turns a line into the style: a guest, or a name with
  no hue, gets `--author-chroma: 0`, which every rule multiplies its saturation
  by. So **guests are grey** whatever they hold, and **so is a player who has
  not picked a color**: there is no color from the name, so the grey is a
  reason to pick one.
- `streak.ts` — `streakShown`: a flame is drawn from a streak of 3 days. Every
  player of today has 1, so a short run would mean nothing.
- `shareCard.ts` — everything about a shared image that is decided before a
  pixel is drawn: the `?f=<code>` link, the text that rides with it, the line
  under the flag, and the size the card comes out at. See [Sharing the
  globe](#sharing-the-globe).
- `shields.ts` — `TileShields`, how many shields stand on each tile, and
  `outcomeOf` and `placementOf`, what this player's click does to a tile. See
  [Shields](#shields).
- `clickOrDrag.ts` — `ClickOrDrag`, whether a press was a click or a drag of
  the globe. The browser sends `click` after a drag too, so turning the globe
  claimed the tile under the cursor on release. A press that moves more than
  6px (12px for a finger, which rolls as it lifts), or that a second finger
  joins, is a drag, and `globe.ts` drops the click that ends it — the bonus box
  included.
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

Beside them: `session.ts` (the click token, see [Sessions](#sessions)),
`account.ts` (who the cookie belongs to, see [Sign-in](#sign-in)) and
`player.ts` — `PlayerBackend`, the player's `Profile`, `PlayerError` and
`isValidUsername`, the username rule, plus `PresenceBackend` (see [Who is
playing](#who-is-playing)). `playerBackend.ts` implements both against
`player.v1.PlayerService`.

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
enforces the same bucket as production's `rateLimiter` (one click every 5s, 60 in hand), and takes `vpnBlocked` and
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

`PlanetBackend` learns it three ways — `GetBudget` at load and on a switch of
country, `ClickResponse.budget` on every accepted click, and a **connect error
detail** on a refused one, which is the reading that matters most. **`GetBudget`
carries the token already held** (`SessionProvider.held()`, never a mint):
without it the server reads the bucket of an address with no account, which is
never spent and always full, and the next click contradicts it. It also subtracts its own clicks in
flight, so the counter only ever *under*-promises: a counter that says 1 and is
refused is a bug the player sees, and one that says 0 and works is a click they
still get.

**One bank, one click per token.** The bank's size never moves: not with the
country, a bonus or signing in. The more of the map a country holds, the slower
its players refill; signing in refills faster, and a refill charge fills the bank. The
server sets that pace on each click, from the country clicked for, so a switch
of flag moves nothing on the meter until the next click. The reading carries the
selected country's slowdown beside it (`ClickBudget.price`): `useClickBudget`
calls `priceFor(country)` whenever the selected country changes, and
`PlanetBackend` keeps the count of every reading but only the price of one for
that country. The meter only explains the price (`domain/clickPrice.ts`): it
says nothing at the plain rate unless the country is within 80% of the first step.

A server that reports nothing — no throttle, or one too old for the call —
leaves the counter hidden rather than showing a made-up allowance, so this ships
ahead of the backend. `FakeBackend` implements the same interface off its own
bucket, so the counter is live in dev.

`listenForUpdates` goes through `openStream`, which reopens with a capped
exponential backoff. It is the only source of live changes, so a drop that is not
retried freezes the globe until a reload.

**The planet stream carries the click token it can have without a mint.** Each
(re)connect puts `SessionProvider.held()` in `X-Session-Token`, so the server
knows which account the stream serves; with none held it opens without one and
the server follows the address. It never calls `token()`: watching the planet is
not worth a Turnstile check. The server reads the token only when the stream
opens, so `followSession` **reopens the stream** when a click, a claim or a bomb
the server accepted went out under a token the stream was not opened with — the
first click of a page load, and the first one after a sign-in or a sign-out. The
hourly re-mint for the same account reopens it too: telling the two apart would
mean reading the token, which is the server's business. A refused call reopens
nothing.

### The screen: four zones

**Nothing is drawn over the globe but four zones**, and a new feature goes in one
of them rather than in a fifth panel. The rule is in [`DESIGN.md`](DESIGN.md);
this is where each zone lives. `Viewer` composes them, and `useCompact`
(`compact.ts`, the 768px query, kept live) picks the phone or the desktop form.

| Zone | Phone | Desktop |
|---|---|---|
| **Status** | `hud/StatusBar`: logo, flag, country and rank (opens the board), `SeasonChip` at its end | `Menu`'s header and "playing for", `SeasonChip` at the top centre |
| **Moments** | under the status bar (`--status-bottom`) | under the season chip |
| **Play** | the dock (`ClickBudgetMeter` + `Inventory`) above the tab bar, the chat's peek above it | the dock at the bottom centre |
| **Places** | `hud/TabBar` (Board, Chat, Sign in / You, Settings, More), each a `hud/Sheet` | `Menu`'s tabs (Board, You, Settings, More) on the left, the chat on the right |

- **One sheet at a time on a phone.** `Viewer` holds which (`sheet`): the four
  tabs, and the season and "Your clicks", which the status bar and the dock open.
  A tab pressed again closes it. The sheets are the same places the desktop
  shows: `BoardPlace`, `YouPlace` and `MorePlace` in `Menu.tsx`, `SettingsPlace`,
  `SeasonDetails`, `ClicksPanel`, and the chat's own sheet.
- **Settings is every switch the player keeps**: the display (`useDisplaySettings`,
  `domain/displaySettings.ts`, in `clickplanet-display-settings`) and the sound.
  More is for things to do, not things to set.
- **A sheet sits above the tab bar** and is as tall as what it holds, up to the
  room under the status bar; the chat's is that tall always, for its log to
  scroll. It covers the dock: a sheet is for reading, the dock for playing.
- **The desktop menu is as tall as what it holds**, up to the screen (less the
  dock under 1444px, where the two would meet): a short place makes a short
  panel, and only the board, or the account, scrolls inside it.
- **Escape closes the innermost** (`useEscape` keeps a stack): a sub-panel such
  as the country picker steps back, then the sheet closes. A modal dialog
  still takes Escape first.
- **Every `Modal` is drawn in a portal on the body**, so a dialog opened from
  inside the menu or a sheet covers the page and not its panel.
- **The other player's card** is the same `Modal`; on a phone its CSS makes it a
  bottom sheet as tall as the card.

### Live chat

The client for the backend's second bounded context: `chat.ts` declares
`ChatSender`, `ChatHistoryGetter`, `ChatListener`, `ChatReactor` and
`ChatSeenMarker` (plus `ChatBackend`, the five together), `chatBackend.ts`
implements them against `/chat.v1.ChatService/` alone — `SendMessage`,
`GetHistory`, `React`, `MarkSeen` and the `ListenForEvents` stream — and
`fakeChatBackend.ts` is the dev stand-in. On a desktop `ChatPanel` is the
right-hand column, folded and unfolded from its own header. On a phone it is the
Chat tab's sheet, and `Viewer` holds whether it is open (`open`,
`onOpenChange`); see [The screen](#the-screen-four-zones). **It is always
mounted**, open or not, on both: it owns the history load, the stream, the unread
count and the sound. Closed on a phone it draws only the peek (below) and hands
the unread count up through `onUnread`, for the tab's badge.

**The chat and who is online share the panel**: with a roster wired, its header
is two tabs, Chat and "Online · N" (see [Who is playing](#who-is-playing)).

**The open panel is resized from its top edge, its left edge or its top-left
corner**, and a double-click on one puts the default back (`useChatSize`). Not
on a phone, where it is a sheet. What the player dragged to is kept
in `clickplanet-chat-size` and written on `:root` as `--chat-wanted-width` and
`--chat-wanted-height`; `index.css` clamps them into `--chat-width` and
`--chat-height`. **The clamp keeps the chat 16px clear of the dock**, centred at
the bottom: the width stops at `50vw - var(--dock-width) / 2 - 32px`, and under
1100px wide, where that leaves too little, the chat sits above the dock instead
(`--chat-lift`). The log stays pinned
to its newest line while the panel changes size (a `ResizeObserver` in `ChatLog`).

**`MAX_TEXT_LENGTH` in `chat.ts` mirrors `chat.service.maxTextLength` on the
backend**, counted in code points as the server counts runes. It is the
composer's bound, not a defence — the server sanitizes and rejects on its own.

**Message text is rendered as text, never as HTML.** The backend stores it raw
and says so; React escaping is what makes that safe, so never reach for
`dangerouslySetInnerHTML` here.

**The server names every sender; the client sends no name.** A player with a
username posts under it. Every other account is a guest, shown as `guest_` and a
6-hex code the server drew once for that account (`guest_a1b2c3`, `GUEST_PREFIX`)
— no username starts with the prefix, so a guest cannot pass for a player, and
no two accounts share a code, so a name is one account. **Nothing about the
sender's address is public**: the `#tag` beside every name is gone from the wire.
The composer asks nobody for a name; its foot reads "as <username>", or for a
guest "as a guest" until its first message comes back, then the name the server
gave it (`nameSentUnder`). The client still keeps a UUID in
`clickplanet-chat-identity` (`chatIdentity.ts`, `useChatIdentity.ts`, held by
`ChatPanel`) because `SendMessageRequest.author_id` carries it; the server
trusts it for nothing, and a name stored there by an older build is dropped.

**`SendMessage` and `React` always carry the click token**, guests included:
the server refuses a caller with no account (`unauthenticated`). So
`ChatServiceBackend` calls `token()`, which mints when none is held, as a click
does — chatting before the first click costs a Turnstile check. A token the
server refuses is dropped and the call made once more with a fresh one (a
refused call posted nothing, so this cannot post twice); a second refusal, or a
mint that failed, is `ChatNoSessionError`, and nothing is sent. `Viewer` reads
the username off the `AccountStore` and hands it to `ChatPanel`, for the
composer's foot and the sound. "Your own message never pings" checks the ids in
`mine` and the name: the username, or a guest's name once `nameSentUnder` knows
it. A guest's very first message can ping if its broadcast beats the answer.

**`sendMessage` is the one call that is not wrapped in `retrying`.** A retry
after a connection dropped mid-request would post the message twice, visibly, to
everyone; a message the player can retype is the cheaper failure. `getHistory`
is retried like every other read.

The refusals map to their own error classes and are reported **inline in the
composer, not as a modal** — unlike a refused click, the text is still in the box
and the advice is one line:

- `resource_exhausted` → `ChatRateLimitedError`, chat's own bucket (one message
  every 3s), unrelated to the click bucket
- `permission_denied` → `ChatBlockedError`, the address is in `chat.blockedIPs`
- `permission_denied` with a `MuteRefusal` detail → `ChatMutedError`, an operator
  muted the account or its network; the composer says until when (the time, or
  the date and time past 20 hours). A muted reaction is undone and says the same
- `invalid_argument` → `ChatRejectedError`, the server refused the content
- `unauthenticated` twice, or no token to be had → `ChatNoSessionError`

The server always runs chat, so there is no "chat is off" error. **A history that
cannot be loaded hides the panel entirely** rather than showing a broken box:
`useChat` goes to `unavailable` and `ChatPanel` renders nothing. So does a build
with no chat backend wired.

The composer **clears the box when the send starts, not when it lands**, and puts
the text back only if the box is still empty when a refusal comes in. Clearing on
success instead wipes whatever was typed while the message was in flight, which
is exactly what a fast typer does.

#### Reactions

A message carries reactions from a fixed set, `chat.v1.Reaction`: the proto enum
is the list, and the backend refuses any other.

- **Drawn from our own images, never the system's emoji font**, which looks
  different, or broken, on every platform (Windows most of all). They are
  Google's Noto Emoji (Apache 2.0), vendored by `npm run reactions`
  (`scripts/generateReactions.mjs`) from a pinned commit into
  `static/reactions/` under content-addressed names, with
  `app/chat/reactionsAsset.ts` generated beside them. **A new reaction** is a
  value at the end of the proto enum, a line in the script's `REACTIONS`, and a
  run of the script. A reaction this build has no image for is not shown.
- `ChatLog` shows the counts under each balloon (`ReactionBar`), and a button
  beside the balloon (`AddReactionButton`) that opens the picker. The button
  shows on hover; a touch screen has no hover, so there it stays, faint. The
  picker closes on a pick, on Escape and on a click elsewhere.
- **A chip says who reacted while the pointer rests on it**, or while it holds
  the focus: the reaction's name, then the players under it (`whoReacted`,
  `ReactionWho`). The names come from the server on the count itself
  (`ReactionCount.reactors`), oldest first, cut at 20; the server reads them
  from the accounts under the reaction rather than any stored name, so a rename
  shows here too. `count` is what says how many gave it, so the popup ends on
  "and N more" whenever it has fewer names than that — a long list the server
  cut, or somebody it could not name at all. It is drawn in a portal on the body, not beside the chip: the log
  both scrolls and clips. Anything that
  moves the chip — a scroll, a resize — closes it rather than making it follow.
- **`mine` is only known from a call.** `GetHistory` sends the reader's token
  (`SessionProvider.identity()`, never a mint) so the server can mark the
  player's own; `React` answers the counts with `mine` set. The stream is
  nobody's, so `mergedReactions` keeps what the log already knew. A player
  whose click token is not held yet when the history loads is named by its
  identity token instead (see [Sessions](#sessions)); one with neither sees its own reactions
  unmarked; the server treats a second "on" as nothing, so a click still ends
  right.
- `React` goes out with the click token, minted when none is held, like a
  message: every caller reacts as its account, guests included.
- **Each message keeps its reactions' version** (`reactionsVersion`). The
  server publishes tallies with no lock, so two frames can arrive in the wrong
  order: `applyReactionsChange` (a frame) and `applyReactionsAnswer` (the answer
  to this player's own reaction) drop one older than what the log holds.
- `useChat.react` shows the change at once (`toggledReactions`), then takes the
  server's answer, or undoes it when refused. A message the server no longer
  shows reads as `ChatMessageGoneError`. It puts this player's own name in and
  out of the popup's list too: `useChat` derives `displayName` — the `username`
  prop, or the name the server posted under (`nameSentUnder`) for a guest — and
  reads it through a ref, so a guest learning its name builds no new callback.
  Without one the reaction is counted and nobody new is named, until the
  answer lands.
- A reaction is not a new message: `ChatLog` shows the "New messages" pill only
  when the last message changes, the unread count counts messages and
  announcements, and the sound only messages.

#### Announcements

The chat also shows lines nobody sent: `ChatEvent.announcement` on the stream,
and `GetHistoryResponse.announcements` beside the messages. Two kinds today:
`bomb`, every bomb that went off, and `mute`, every mute an operator gave.

- **Decoded, not trusted**: `decodedAnnouncement` reads the `kind` and parses
  the JSON `payload` into a typed `ChatAnnouncement`. A kind this build does not
  know, or a payload that is not the kind's, is dropped, so the server can ship
  a new kind first.
- **The payload is values, the client writes the sentence**: a bomb line is
  `describeBlast`, the same words as `BombNews`, so the chat and the news line
  never disagree. A mute line is the name and `describeMute` of its seconds
  (`domain/mute.ts`): "one hour", "90 minutes", "2 days".
- **Kept apart from the messages** (`useChat`'s `announcements`,
  `addAnnouncements`) and put in one list only to draw (`interleave`, by time).
  So a burst of bombs never pushes a message out of the log, and the sound and
  the "New messages" pill count messages alone. The unread count counts
  announcements too: a bomb while away is something missed.
  **Once the message log is full, `interleave` leaves out every announcement
  older than its oldest message**: the two logs are capped apart, so in a long
  session the older bombs piled up on top of the chat. The server does the same
  for the history.
- **Not a balloon**: `ChatLog` draws a centred line (`.chat-announcement`) with
  the bomber's flag and the time. It ends the run above it, so the next message
  says again who is talking.
- In fake mode `main.tsx` hands every `FakeBackend` bomb to
  `FakeChatBackend.announceBomb`, with no ground: the fake has no borders.
  `fakeChat.mute()` in the console announces a mute of the player for an hour
  and refuses its posts and reactions until it ends; `fakeChat.mute(600, "Ana")`
  only announces somebody else's.

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
  blink: this sits over a game. On a phone the badge is on the Chat tab.
- **The newest line is quoted under the folded header**, in its author's colour.
  It is `aria-hidden` — a screen reader gets the count from the badge and the
  text from the log, and the quote would only say it a third time. **On a phone
  it is a peek over the dock instead** (`.chat-toast`): the newest line, as a
  balloon, for `TOAST_MS` (4s) after it lands, and a press opens the chat. It is
  a button named by the line it quotes, since it is the only thing on screen
  that opens the chat from it.

#### What was missed since the last visit

**The server keeps until when each account saw the chat**, so the badge on
arrival counts what was said while the player was away: messages and
announcements, not the player's own messages. See the backend's CLAUDE.md
(Chat, Seen).

- **A time, not an id.** `GetHistory` answers `seenUntil`; `useChat` hands
  `ChatPanel` the time to count from (`seenAtLoad`), and `unseenAfter` counts
  the lines after it, open or not. A history with no mark is a first visit:
  nothing counts, and the newest line of the history is sent as the mark, so
  the next visit has one even if the chat is never opened in this one.
- **Opening the chat marks nothing.** On a desktop it is open from the start,
  and a phone's sheet used to open on the newest line: either way the player
  was shown the bottom and nothing above it. So `ChatLog` opens on an **Unread
  line** (`.chat-unseen`) above the first line after the mark, with a little of
  what was read above it, and shows the "New messages" button while there is
  more below. The line is placed once, as the log opens, and stays while the
  player reads down from it; nothing missed, no line, and the log opens on the
  newest as before.
- **Seen is what was scrolled into view.** On every scroll, resize and new
  line, `ChatLog` finds the newest line whose whole height has been on screen
  (`data-at` on each line) and reports its time through `onSeen`. A line that
  lands while the log is pinned to the bottom is on screen at once, so it is
  seen at once. "New messages" goes to the newest line, and so marks
  everything. Nothing is reported while the page is hidden (`watching`, from
  `usePageVisible`): a tab left open overnight would mark everything seen.
- **The mark is the time of the newest line shown, never "now"**, which would
  count as seen a line that lands while the call is in flight.
- **`useSeenMark` sends it at most every `SEEN_DELAY_MS` (2s)**, so a fast
  scroll is one call, and at once on `pagehide`, over a keepalive client.
- **It is read and kept as the identity token** (`identity()` for the history,
  `heldIdentity()` for the mark; see [Sessions](#sessions)): a click token lives
  an hour, so on the next day's visit the identity the cookie resumes, with no
  Turnstile check, is what says who is asking. The mark sends the token in hand
  and never resumes or mints one, since it also goes out as the page closes. A
  refusal for want of an account is silent: a page with no session has nobody
  to keep a mark for.
- In fake mode the mark starts 2.5 minutes back, so two of the opening lines
  count as missed.
- jsdom lays nothing out, so every line counts as on screen in a test. The
  tests of the Unread line stub the layout: lines 50px tall in a log 150px tall.

**The log is never yanked down under someone who scrolled up to read.** It
auto-scrolls only while it is pinned to the bottom (`PINNED_SLACK_PX`);
otherwise a "New messages" pill appears and scrolling back down, by the pill or
by hand, dismisses it.

Every one of these animations is dropped or reduced under
`prefers-reduced-motion: reduce`, keeping the colour and losing the movement.

### Who is playing

The "Online · N" tab beside Chat, in the chat's header, lists
everyone playing: players with a username, then guests, each with a flag, a name
in its chat colour (`authorStyle`, the same hue as in the chat). A line's name
is the one the chat shows for that account: the username, or `guest_` and its
code. No address, and no hash of one, is on it.

- `backends/player.ts` — `PresenceBackend`, `Presence`, `PlayerLine` (what a
  card needs), `RosterEntry` (a `PlayerLine` with its `key`), `RosterEvent`, and
  `PlayerInfoBackend` with `PlayerInfo`. `ConnectPlayerBackend` implements it over
  `player.v1.PlayerService/Announce`, `Leave` and `ListenForEvents`;
  `fakePresenceBackend.ts` is the dev stand-in, with players coming and going on
  their own shifts, diffed into live events every second.
- `domain/presence.ts` — `PresenceSchedule`, when to announce. No clock and no
  network, so every rule is under test. `domain/roster.ts` applies one live event
  (`applyRosterEvent`) and splits the roster into the two groups.
- `app/players/` — `usePresence`, `useRoster` and `usePlayerInfo`, thin hooks
  over the above, `PlayersPanel` and `PlayerCard`.

**A name wears its color and its streak** everywhere it is drawn: the chat log,
the folded quote, the roster and the card. Both come from the server with the
name (`ChatMessage.authorColor` and `authorStreak`, `RosterEntry.color` and
`streak`, `PlayerInfo.color`), read from the account when shown, so a new pick
shows on everything its player ever said once the chat is read again. The
flame (`StreakFlame`) is the Noto fire of the reactions, `role="img"` named
"12-day streak", and is left out under 3 days (`streakShown`). It is a button: a
press, or a mouse resting on it, says "Played 12 days in a row" in a `Bubble`,
which goes when the log under it scrolls. **A guest has
neither**: the server sends it no color and a streak of 0, so a signed-in player
shows a flame only once it has a username.

**A name wears its title too**, in the chat log, the roster and the season board: the medal of the
title its player wears, small, after the crown (`TitleBadge`, a `TitleEmblem`
with `role="img"` named by the title). The server sends it with the name
(`ChatMessage.authorTitle`, `RosterEntry.wornTitle`, `Standing.wornTitle`), read when shown, so a new
pick shows on the roster at once, on the next message, and on older ones once the
chat is read again. A guest
wears none. `backends/title.ts` holds `PlayerTitle` and `titleOf`, the one
decoder of `player.v1.Title`, shared by the chat, the player and the standings backends.

**An admin of the game wears a crown** (`AdminCrown`, gold, `role="img"` named
"Admin") beside its name in the chat log, the roster and the card's title.
The server says so: `ChatMessage.authorAdmin`, `RosterEntry.admin` and
`PlayerInfo.admin`. The card crowns from what was clicked, and from the read
once it lands. In fake mode, Ana is the admin.

**A name opens a player card**, in the roster, on a chat message and on the
season's board (`app/players/PlayerCard.tsx`, a `Modal`). `Viewer` holds the one
card open and hands `onOpenPlayer` to `ChatPanel` → `ChatLog` and `PlayersPanel`,
and to the board's `BoardStandings`;
without a `PlayerInfoBackend` wired the names are plain text. The card shows the
flag and the country, then, for a player with a username, what
`player.v1.PlayerService/GetPlayer` answers: the title it wears and the titles
it shows (see [Titles](#titles)), then tiles taken, the current and best
streak, and "Playing since", the day the account was made (left out when the
server does not know it). The fake gives its players titles of its own over
their fake stats and creation date. **A guest's card asks nothing**: a guest has no
username, so there is nothing to look up, and the card shows the name and the
flag alone. It says nothing to the viewer, who may well be signed in. The chat tells
a guest by `GUEST_PREFIX`, which no username starts with. `GetPlayer` needs no
token and goes out as a GET, like `GetRoster`; `NotFound` (renamed, or the
account is gone) reads as `undefined`. `usePlayerInfo` reads it once per card.
Escape closes the card and leaves the roster open: `useEscape` does nothing
while a modal dialog is on the page.

**It announces only with a token it already holds.** `SessionProvider.held()`
answers the click token in hand and never mints: a mint is a Turnstile check,
and presence is not worth one. So a visitor who has never clicked is not listed,
by design; one whose kept token is still live is listed from the load — see
[Sessions](#sessions). The schedule announces as soon as a token is held that the last
announce did not go out under (the first click, a re-mint, a sign-in), once
the flag or the username has held still for a second, and every
30s — the server drops a player 90s after its last one. An announce carries the
flag alone: the server reads the name off the account. An announce refused
`unauthenticated` drops the token and is **not** retried with a fresh one, which
would mint; the next click brings one. `NoSession` holds nothing, so a build
without a sitekey never announces.

**The roster is streamed**, over `ListenForEvents` through `openStream`, with
the token already held (`held()`, never a mint) when there is one. The roster
needs none; the token is what lets the same stream bring this player's own
titles (`titleEarned`, see [Titles](#titles)). The server reads it when the
stream opens, so `useRoster` checks `heldSession()` every `SETTLE_MS` and
reopens the stream when it changes, keeping the list it has. Every connection starts with the whole roster, then
sends one line that joined or changed (`entry`) or one key that left (`left`).
A line is named by its `key`, which the server keeps through a new flag, a
sign-in and a new name, so a guest who signs in is one row renamed in place;
rows are keyed by it in React too. `applyRosterEvent` puts a changed line where
the server's sort would (`compareRosterEntries`); a rare disagreement about
case in a non-ASCII name lasts until the next reconnect's whole roster. While
the stream is down the last list stays. A 404 (read as `unimplemented`) calls
`onUnavailable` once and stops the stream for good, which hides the button, as
does a build with no presence backend wired. `GetRoster` is no longer called.

**A closing page says it left.** `usePresence` calls `leave` on `pagehide`,
unless the page is only kept in the back-forward cache (`persisted`). `Leave`
carries the held token, never a fresh one, and goes out on a second player
client whose `fetch` sets `keepalive` (`newKeepalivePlayerServiceClient`), so it
is still sent after the page is gone. The server takes the account off at once.
Two tabs of one browser are one account: closing one takes the line off until
the other's next announce, at most 30s later.

### Titles

A linked account earns titles; the server decides which and keeps them (see
the backend's CLAUDE.md). `app/titles/` draws them and `app/account/ProgressTab.tsx`
is the player's own view.

- **Most titles are ranks on a track** (`TitleRank`: the track, the rank's
  number, how many ranks). Conquest counts tiles taken, Devotion the streak; OG
  stands alone. **Only the highest rank of each track is ever shown or worn**, so
  the client never filters: it draws what the server sends, in its order.
- **Every title is a medal** (`TitleEmblem`): an SVG per id, drawn by the
  medal rules of [`DESIGN.md`](DESIGN.md), in the metal `titleArt.ts` gives it
  (`TITLE_METALS`, bronze up to prism) and the colors of its track (`enamelOf`,
  `ribbonOf`). An id this build has no picture for gets the first letter of its
  name in silver, so a new title shows before its art ships. `locked` greys it
  with a padlock, for a rank not held. Each medal names its masks with `useId`:
  two on one page cannot share one. Its colors are tokens set through `style`,
  since an SVG presentation attribute does not resolve `var()`.
- **The public card** wears the worn title: a banner (`TitleBanner`, the rank line
  and the name), and a ring of its metal inside the card's ink border
  (`title-frame-<metal>` on the `Modal`). Under it, one medal per title shown.
  OG is also a stamp beside the name (`OgStamp`).
- **The Progress tab** is the worn title (a compact banner), "Wear a title" (a
  `radiogroup` of the titles that can be worn; a press sends `WearTitle` and reads
  the titles again), and one `TrackPath` per track: every rank, its threshold
  (`stepLabel`), a bar filled up to the progress (`filledOf`), and in its header
  what is left to the next rank (`leftLabel`). The path scrolls sideways and opens
  centred on the next rank. `useTitles` reads `GetTitles` (the click token, as
  `GetProfile`) each time the panel opens and after a run of clicks, at the pace
  of `useMySeason`; a failed read says so. Tiles taken counts each take at once
  (`useOwnTakes`): `GetPlayer` is cached 10s, so it is read only once.
- **The unlock moment is live.** The player stream carries `titleEarned` to a
  stream opened with this player's token. `useRoster` hands it to `Viewer`, which
  queues them and shows `TitleUnlocked` over the game, one at a time. It is not a
  `Modal` but the whole screen: it goes dark, "New title!" slams in, the medal
  spins in and lands with a flash, a shockwave and confetti, then the name and the
  rank line. The `title` sound is a drum roll that lands on the same beat; both
  read `TITLE_REVEAL` (`domain/titleReveal.ts`). **Nothing closes it before
  `TITLE_REVEAL.ready`**, and a click beside it never does: a player spamming the
  globe would otherwise dismiss it unseen. Then "Close" and "Wear it"
  (`AccountStore.wearTitle`; left out when no account store is wired) appear, and
  Escape works. The server sends only the highest
  rank per track of what one take earned, so a jump of two ranks is one overlay.
  A title earned while no tab is open is never announced; it is simply there next
  time.
- **The worn title follows the name** in the chat, the roster and the season
  board, as a small medal (see [Who is playing](#who-is-playing)).
- **UI copy is not documentation.** The card and the tab say nothing about the
  rules ("one per track", "others see the title you wear"): what is drawn is the rule.

### Season standings

`backends/standings.ts` is the contract: `StandingsBackend`, `Standing` (a
ranked player: rank, name, color, worn title, the flag its tiles are for, tiles)
and `MySeason` (the caller's line on one board: that flag, its tiles, its rank
and its worn title). `standingsBackend.ts` implements it over
`seasons.v1.SeasonService/ListenForEvents` and `GetMySeason`, and
`fakeStandingsBackend.ts` stands in for it in fake mode, counting the player's
own clicks and moving the other players a few tiles every 1.5s. `app/standings/`
draws it.

- **A player's season is the tiles it took this season for its main flag**, the
  flag it took the most for: that is the Players board. **A country's board is
  every player who took tiles for that country, by those tiles**, so a player is
  on the board of each flag it took for, from its first take. The server ranks
  only signed-in players (each has a username; a guest has none), and ties share
  a rank (1, 2, 2, 4). `RankCoin` draws the rank, as on the countries' board.
- **The board has three views** (`BoardViews`): Countries, the `Leaderboard` as
  it was; Players; and the players of the country played for, named by its flag
  and name. `Viewer` holds the view, so a closed sheet or
  another menu tab keeps it. With no `StandingsBackend` wired the board has no
  views.
- **The view is picked from the board's heading** (`HeadingSelect`, a gold
  section title that opens a listbox), not from tabs: the board is already a
  tab of the menu, and tabs in a tab read as one row of places. **The list is
  `position: fixed` under its button**: absolute, it was clipped by the menu's
  scrolling body and made a short board scroll. It follows the button every
  frame (a sheet grows upward as the board loads), opens upward with no room
  below, and closes when something that holds it scrolls.
- **The board is streamed.** `useStandings` follows `ListenForEvents` for the
  view shown (`country_id`, empty for every player) through `openStream`, with
  no token and `NO_TIMEOUT`, and stops when the view changes or closes. The
  server sends the view's whole top 10 when the stream opens and again each time
  it changes, at most once a second; a `board` replaces what is shown, a
  `heartbeat` is skipped. While the stream is down the last board stays, and a
  reconnect starts with a whole board. A server without the stream is retried
  with `openStream`'s backoff. `GetStandings` is no longer called.
- **`GetMySeason` reads as the identity token** (`identity()`: a fresh token, or
  one resumed from the cookie, never a Turnstile mint), so a player back the next
  day sees its season at once. With none to be had it is not sent, and
  `unauthenticated` reads as unknown and keeps the token. **It is asked for the
  board on screen**: no country on Players, which reads the main flag, its tiles
  and the global rank; the country on its board, which reads the tiles taken for
  it and the rank there. `useMySeason` reads it when a players' view opens, when
  the view's country changes (and shows nothing of the other board meanwhile),
  when the account or the username changes, 2s after the last of a run of
  accepted clicks, and at least every 10s while they keep coming
  (`useReadsAfterClicks`).
- **The caller's own numbers move on every take.** The globe already knows
  which click takes a tile (one on a tile its flag does not hold and no shield
  stands on), and once
  the server accepts one it tells `acceptedClicks`, which tells its listeners
  with no render of `Viewer` per click. `useOwnTakes` counts them by flag, and
  `liveSeason` adds the ones made since the read was sent to the line's flag:
  the country shown on its board, the main flag on Players. A player with no
  main flag yet gets the server's rule there (the first flag, until another has
  strictly more). On Players a take for another flag adds nothing: only the
  server knows whether it became the main one. A spread or an enclosure's extra
  tiles, and the ranks, wait for the next read.
- **The caller's row is drawn from that count** (`boardWith`): it climbs into
  the top 10 and pushes the last out, its rank counted among the rows it passes,
  and the rank in "Your season" follows it. The other rows move with the
  stream: a `TileUpdate` does not say who took the tile.
- **The caller's own line.** "Your season" sits over the table and shows only
  the board on screen: on Players the season's tiles and, with a username, the
  rank among all players; on a country's board the tiles taken for that country
  and the rank there, under its name. No tile names another country than the
  one shown. A guest gets a Sign in button beside its tiles, which opens
  `SignInPitchModal`. In the table the caller's row is the same line: marked
  when it is in the top 10, and otherwise added under it with its rank there.
  Its name and color come from the profile, its title from `GetMySeason`: read
  again after clicks, it follows a rank-up, and a title picked meanwhile shows
  at the next read.
- **A name opens the player card**, as in the roster and the chat, with the
  flag of its row and its title.
- **The head's line is a background**, 7px above its bottom edge: the gap under
  it was the first row's top padding, and the caller's highlight took it in
  when it was first.

### The season

`backends/season.ts` is the contract, `seasonBackend.ts` reads
`seasons.v1.SeasonService/GetSeason` once per page load (a cached GET), and
`fakeSeasonBackend.ts` answers Season 0 in fake mode. A 404 reads as no season.

- `domain/seasonClock.ts` — `seasonClock`, the time left to the second
  (`27d 14h 05m 12s`, `13h 05m 12s`, `52m 10s`, nothing once over) and whether the finale runs, and `finaleWindow`,
  the finale's day and hours in the player's own time zone.
- `app/season/` — `useSeason`, which drops the season at its end (a page open
  across it goes back to no season), `SeasonChip`, `SeasonDetails` and
  `SeasonFacts`, the rows both of them open on.

**The season is a chip in the status zone.** On a desktop it sits at the top
centre: "Season 0 ends in 28d 14h 05m 12s", and a press opens a dropdown (Escape
closes it). On a phone it is the right end of the status bar, the two largest
units alone ("28d 14h", named in full for a screen reader), and a press opens the
same details as a sheet. During the finale it glows and says "Final Battle ends
in"; it still opens.

**The desktop chip has a fixed width** (368px, the widest countdown plus a
little). Luckiest Guy has no equal-width digits, so the countdown changes width
every second: a chip as wide as its text moved every second and wrapped the
dropdown's rows again with it.

**What it opens is the one place the UI explains the rules**, asked for on
purpose: four rows (`SeasonFacts`), the Final Battle with its day and hours, held
ground counted at the end, the winner's trophy in the Hall of Fame, and the
titles. During the finale the first row is the power-ups instead of the date.
**Say only what is decided**: a battle full of power-ups, points for held ground
counted at the end, a trophy for the winning country, a title for every
signed-in player and one more for the winning country's. Guests get no title, so
the row says "signed-in".

**The home page counts down to the Final Battle.** `src/home.ts` reads the
season with the same client, and `app/home/finale.ts` fills the pill in the hero
and the `#final-battle` section, both `hidden` until a season is known and again
once it is over. `finaleClock` counts to the battle's start, then to the season's
end, when both go live and glow. The page's text stays in `index.html`; the
module only writes the numbers and toggles `finale-live`. **It imports no CSS**:
a stylesheet shared with `play.html` becomes its own file, linked before
`play-*.css`, and moves the game's cascade. The art's styles are copied into the
page's `<style>`, as `.panel` and `.button` are.

**The desktop chip writes its bottom edge on `:root` as `--status-bottom`**
(`useBottomEdge`), and on a phone the status bar does: the quiz and the bomb
news sit under it. With neither, the property is unset and they sit at the top.

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

**It mints through `auth.v1.AuthService/CreateSession`**, which also gives the
browser an account: a guest one, kept in the `cp_sid` cookie the answer sets
(HttpOnly, on the API's host). The next mint sends the cookie back and gets the
same account. The page never reads the cookie; the menu learns what it holds
from `GetMe` — see [Sign-in](#sign-in). `session.v1` is deprecated on the
backend and this build no longer calls it.

**`invalidate` also drops a mint in flight.** A sign-in or a sign-out changes
the account the cookie names, and a token minted before that would name the old
one for its hour. A generation counter keeps such a mint from being stored.

**`newAuthServiceClient` is the only transport that sends credentials.** Its
`fetch` wrapper adds `credentials: "include"`; without it connect-web sends
`same-origin`, and a cross-origin mint neither sends the cookie nor keeps the
one it is given — every mint would start a new guest. The click, map and chat
clients stay without it: who asks is in the token they carry, the identity
token included, and a read that carries a cookie is one no shared cache serves. Both halves are pinned in
`turnstileSession.test.ts`. A credentialed call needs the API to name the exact
origin and send `Access-Control-Allow-Credentials: true` — Caddy does in
production, and a local backend does from `httpServer.allowedOrigin`, which
must be the dev server's origin (`http://localhost:5173`).

**A held token outlives the page it was minted on.** `localTokenStore` keeps it
in local storage (`clickplanet-session`) and the client takes it back at
construction, by the same margin `held` applies to one it minted. Nothing mints
at load, so without this a reload held nothing until its first click and
everything that reads the token without minting read as a caller with no
account: the inventory came back empty, the meter showed the scope's bucket
rather than the player's, the stream followed the address, and presence listed
nobody. An invalidation and a failed mint both drop what was kept, so a reload
after a sign-out does not bring the old account's token back.

**Two tokens, one held at a time: the click token and the identity token.** The
click token comes from `CreateSession`, after a Turnstile check, and is the only
thing that may act. The identity token comes from `ResumeSession`, off the
`cp_sid` cookie with no Turnstile check: it names the same account and proves
no check, so the server takes it for reads alone (the backend's CLAUDE.md, "The
click token and the identity token"). A click token names the reader too, so
one token in hand serves both.

- **`token()`** answers the click token, minting through Turnstile when what is
  held is only an identity. **`held()`** never answers the identity token, so
  nothing that acts can send it by mistake: a click, a post, a reaction, a
  claim, an announce.
- **`identity()`** answers any fresh token, and otherwise resumes one silently;
  **`heldIdentity()`** is the same without the call. The reads use them: the
  history, the budget, the charges, both streams, the roster. `PlanetBackend`
  resumes at load (`followIdentity`), so a player back the next day is named
  from the first frame, with no Turnstile check and no click.
- **A cookie with no live session resumes nothing**: a first visit stays
  anonymous until its first click, as before. A resume that fails is no
  identity, logged, never an error a read would surface.
- **A resume that lands after a click token was minted leaves the click token**:
  it names the same account and proves more.
- **Kept like the click token**, with `identity: true` beside it, so a reload
  still knows a click must pass Turnstile. A kept token with no flag is a click
  token, as every token an older build kept was.

**It keeps the tokens and never the account.** The account is the `cp_sid`
cookie, which is HttpOnly and out of this page's reach either way. A token
lapses within the hour, is bound to the address that minted it, and a page that
could read this could mint one of its own off that cookie. A token restored on
another network is refused, which is the case a click already retries against a
fresh mint; a read that carries it answers for nobody, exactly as no token did.

**The store is injected, not read in the client.** `SessionClient` stays the
half with no DOM and no network — the same split as `turnstileAttester` — so
every rule about what is kept and when is under test without a browser.

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
`auth.enabled: false` expects. The sitekey is public — it is read off the
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
B=$(curl -s https://clickplanet.lol/play | grep -oE '/assets/play-[A-Za-z0-9_-]+\.js' | head -1)
curl -s "https://clickplanet.lol$B" | grep -c 'challenges.cloudflare.com/turnstile'
```

### Sign-in

Optional from end to end: a player who never signs in plays exactly as before,
on the guest account the mint gives every browser. Signing in with Google,
Discord or a code sent to an email address keeps that account on every device.

- `backends/account.ts` — the contract: `AccountBackend`, `Provider` (and
  `OAuthProvider`, the ones the page leaves for: every one but `email`), and
  `AuthError`, whose `failure` says why the server said no. **One class and not
  one per reason**, unlike a refused click: every one is shown the same way, as
  one line beside the button that was pressed.
- `backends/accountBackend.ts` — `ConnectAccountBackend`, over the client
  `newAuthServiceClient` builds, so every call carries the cookie. It maps
  Connect codes: `unimplemented` → `off`, `invalid_argument` → `notOffered`,
  `resource_exhausted` → `tooManyTries`, `failed_precondition` → `startAgain`,
  `permission_denied` → `refused`, `unauthenticated` → `notSignedIn`, anything
  else → `failed`. A refused link is matched on its `LinkRefusal` detail, not
  its code: `linkedElsewhere` or `alreadyLinked`. **Only the two reads are retried**: a retried
  `CompleteSignIn` would spend a code that is good once.
- `domain/signInCallback.ts` — `callbackOf`, what the provider sent to
  `/auth/callback`: a code and a state, a refusal (`error`, which wins), or a
  link with a part missing.
- `backends/player.ts` / `playerBackend.ts` — the username. `ConnectPlayerBackend`
  sends the click token with every call, and a call refused `unauthenticated` is
  sent once more with a fresh session, as a click is. `invalid_argument` →
  `invalid`, `already_exists` → `taken`, `permission_denied` → `guest`, a second
  `unauthenticated` → `notSignedIn`, anything else (a mint that failed included)
  → `failed`. Only `GetProfile` is retried.
- `app/account/accountStore.ts` — `AccountStore`, the section's state machine:
  `loading`, `hidden`, or `ready` with the offered providers, the linked ones,
  the action in flight and the last failure, and the username with its own save
  in flight and its own failure. No DOM and no network of its own, like
  `SessionClient`, so every transition is under test.
- `app/account/` — the rest is React: `AccountPanel` (the menu's You tab, named
  "Sign in" for a guest; a sheet on a phone), `DeleteAccountModal`,
  `SignInCallback` and `SignInGate`.

**The buttons come from `GetSignInOptions`**, which answers the providers the
server offers and is not throttled. Production runs with `auth.signIn.enabled`
off, so the list is empty and the menu looks as it did. **Never probe with
`StartSignIn` instead**: it spends the mint budget, which `CreateSession` needs.
A server without the RPC, or with the whole auth module off, answers 404, which
reads as no provider. So does any other failure of that read: a sign-in that
cannot say what it offers is better absent. A linked account still shows when
nothing is offered, so a player can sign out after sign-in is turned off.

**Sign-in shares the mint budget** (one every 30s, ten in hand): `StartSignIn`
and `CompleteSignIn` each spend one. `tooManyTries` says to wait a minute.

**The flow:**

1. "Sign in with Google" calls `StartSignIn` with the intent `signIn`, keeps
   the provider and the intent in session storage (`rememberedSignIn.ts`) and
   sends the browser to the URL it answers. "Link Discord" (`AccountStore.link`)
   sends the intent `link`. **The intent matters**: a sign-in with an identity
   another account uses moves the browser to that account, and a link is
   refused instead, so the player stays on the account they linked from.
2. The provider sends the browser to `/auth/callback?code=…&state=…`. The
   Workers asset handler serves `auth/callback.html` there, a copy of the game
   the build writes (see [Pages and routes](#pages-and-routes); `nginx.conf` has
   a route of its own), and the project's build watch path, `apps/frontend/`,
   covers every file involved.
3. **The code must not leak.** An inline script at the top of `play.html` adds
   `<meta name="referrer" content="no-referrer">` on that path before any other
   request is made, `public/_headers` sends the same `Referrer-Policy`, and
   `main.tsx` takes the query out of the address bar with `history.replaceState`
   before it renders anything.
4. `SignInGate` renders `SignInCallback` instead of the game. It calls
   `CompleteSignIn` **once** — a ref guards it, since StrictMode runs the effect
   twice and a second trade would fail and hide the first one's success.
5. On success, `AccountStore.completeSignIn` **invalidates the click token** and
   reads the account again, and the gate swaps in the game in place, at `/play`,
   with no reload. The next click mints a token that carries the new account.
6. On failure the page says why in one line. `retryOf` picks what "Try again"
   does: send the same code again when the server did not use it
   (`tooManyTries`, `failed` — a spent budget is refused before the code is
   read), or go back to the remembered provider when the code is spent
   (`startAgain`, `refused`). With no remembered provider there is only "Back to
   the game".
7. A refused link (`linkedElsewhere`, `alreadyLinked`) is titled "Not linked"
   and offers only "Back to the game": the same identity would be refused
   again. The server changed nothing, so the player is still on their account.
   `linkedElsewhere` tells them how to move the identity: sign in with it,
   delete that account, then link it here.

**A signed-in player is given a username when it signs in** (the server draws
one, see the backend's CLAUDE.md), **and may pick another** in `AccountPanel`: 3 to 15
code points of letters of any script, digits, `_` and single spaces, not
starting with `guest_` in any case, unique ignoring case (the server's rule is
`players.NameOf`, see the backend's CLAUDE.md). `usernameOf` puts the draft in
NFC and cuts the spaces at its ends, as the server does, and that is what is
sent. `isValidUsername` mirrors the rule for the Save button with `\p{…}`
classes, and counts code points (`[...name].length`), never `length`. The input's
`maxLength` counts UTF-16 units, so it is twice the rule: the rule is the bound.
One part is the server's alone: JavaScript cannot name a character's script, so
the client refuses only Latin, Greek and Cyrillic mixed (the lookalikes), and a
name mixing other scripts comes back `invalid`. The server is the
authority and alone knows what is taken. The chat shows it (see [Live
chat](#live-chat)). **It is read after the account, not with it**: `GetProfile`
needs a click token, which can mean a mint, so the section shows as soon as
`GetMe` answers and the name follows. Only a linked account reads it — a guest
has none and should not mint to learn that — and a failed read leaves the name
unknown with the form still there. A save and the other actions never run at
once. A sign-in reads it again; a sign-out or a delete forgets it, and a read or
a save that lands after the account changed is dropped.

**A player with a username picks its name color** in `AccountPanel`, under the
username: the 12 of `NAME_COLORS` in a `role="group"` named "Name color", each
`aria-pressed`, none pressed before the first pick. There is no way back to no
color. `AccountStore.setColor` shows the pick at once, sends `SetColor` and
keeps the color the server answers, or the old one on a refusal; `GetProfile` answers it with the name, and `readProfile` reads
both. A color is refused without a username (`FailedPrecondition` → `unnamed`),
which is why the picker only shows with one. `usePresence` announces again once
the color held still for a second (`SETTLE_MS`), so the roster line follows.

**A signed-in account's panel has two tabs**: Progress, open first (see
[Titles](#titles)), and Account, which holds the username, the color, linking
and signing out. A guest has no tabs: its panel is the sign-in buttons alone,
and it reads no titles.

**Signing in by email stays on the page.** The server offers `email` beside the
providers when `auth.email.enabled` is on, and `EmailSignIn` draws it under the
provider buttons, in `AccountPanel` and in `SignInPitchModal`: an address, then
a six-digit code. There is no callback page and nothing is remembered across a
trip, so the steps are store state: `AccountStore.sendCode` asks for a code and
keeps `code: {address, intent}` in the ready state, `checkCode` sends what was
typed (digits only), and `cancelCode` goes back to the address.

- **Each `StartEmailSignIn` needs a fresh Turnstile token**, since each sends an
  email: `ConnectAccountBackend` takes the same attester as `SessionClient`,
  and calls it once per code. Without a sitekey it sends an empty token, which
  a local backend with Turnstile off accepts.
- **A wrong code keeps the code step** (`wrongCode`), and so does a spent mint
  budget or a failure on the way. A lapsed code, or five wrong ones, is
  `newCode`: the step closes and the player asks again. Refused addresses come
  back with their reason in an `EmailRefusal` detail: `invalidEmail` or
  `disposableEmail`. `tooManyCodes` is the address's budget or the network's.
- **On success the account is read again** and the click token invalidated,
  exactly as after `CompleteSignIn`. An email account is linked: it clicks
  faster and is given a username it may change.
- To try it locally, run the backend with `auth.email.enabled` and
  `auth.email.delivery: log` (no Cloudflare account needed): the code is in the
  server log.

**Every way out of an account invalidates the click token too**: sign out, sign
out everywhere, and delete. The player plays on, and the next click mints a new
guest. `unauthenticated` on one of these means the account was already gone, and
is treated as done. Delete sits behind a dialog that lists what goes.

**There is no fake.** `VITE_FAKE_BACKEND` wires no `AccountStore`, so the menu
offers no sign-in there. To try it locally, run the backend with `auth.enabled`,
`auth.turnstile.enabled: false`, `auth.signIn.enabled` and any Google
`clientId`, and the dev server with Turnstile's always-passing test sitekey:

```bash
VITE_API_BASE_URL=http://localhost:8080 VITE_TURNSTILE_SITEKEY=1x00000000000000000000AA npm run dev
```

The start leg is real — the button goes to Google, which refuses a made-up
client — and `/auth/callback?code=x&state=y` exercises the callback page's
refusal. A whole sign-in needs a real client registered with
`http://localhost:5173/auth/callback`. To see the signed-in panel without one,
mint a guest and insert a row into `auth.identities` for its account.

### `src/app/viewer/` — the GPU layer

- `globe.ts` — `createGlobe(options): Promise<Globe>`. Builds the scene, wires
  input and the backends to it, starts rendering. Returns `{tilesCount,
  setCountry, dispose}`. Its loop draws on demand — see [Drawing only when
  something changed](#drawing-only-when-something-changed).
- `useGlobe.ts` — owns one globe for the lifetime of the component. **Its effect
  must not depend on anything that changes per render**; the selected country and
  the player's hue are pushed into the running globe through separate effects
  rather than rebuilding it.
- `useLeaderboardFeed.ts` — the one place React hears about the board, and
  **it samples rather than follows**. See [Sampling the
  leaderboard](#sampling-the-leaderboard).
- `tileField.ts` — owns both point clouds and the three attributes that change at
  runtime (`regionVector`, `hover`, `shield`). Mutates them in place and reports only the
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
- `borderLines.ts` / `borderLinesAsset.ts` — the grey outline between one
  country's ground and the next, at a width that does not move with the zoom.
  See [The countries' outlines](#the-countries-outlines).
- `pointSize.ts` — how big a tile is drawn, how far apart two of them sit, and
  the single schedule that hands the frame from the painted flag to the tiles.
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
- `enclosureEffect.ts` — what every screen shows when somebody closes a shape
  with the enclose bonus (`tilesEnclosed` on the stream): the outline lights up
  and meets at the closing tile, the inside pours in, and a gold ring runs out
  over the ground. **The curves are TypeScript, written into attributes per
  frame**, not GLSL: a shape is a few dozen points, and that is what lets
  `enclosureEffect.test.ts` pin the timing. Marks and the ring have a minimum
  size in pixels, so a shape closed while zoomed out is still seen. A shape that
  arrives while the tab is hidden is not played — it would all start at once on
  return. **A mark is pulled toward the camera by its own size**
  (`unitsPerPixel`): a sprite has one depth, so off the middle of the globe the
  curve of the ground hid half of it. The camera is orthographic, so the pull
  moves only the depth, never the place on screen.
- `clickEffects.ts` — the same, for every click made with spread on
  (`tilesSpread`: a green burst, a spark popping onto each tile around it in
  turn, two rings). It reuses the enclosure's shaders, with normal rather than
  additive rings, which vanished on the white of a flag. A busy planet spreads a
  lot, so at most `MAX_PLAYING` run at once.
- `clickGlints.ts` — **every other click glints on its tile**: this player's at
  once, and anyone else's when its `TileUpdate` says `clicked` — the server sets
  it only on the tile a click named, never on a spread's neighbours, an
  enclosure's inside or a moderator's write. Own clicks echoed back are skipped
  through `OwnClicks`. **A glint is one soft glow and no ring**, gone in 0.7s
  and mostly gone by 0.35s: it is the most frequent thing on the map, a flash
  with a ring was too much at that rate even at its smallest, and a puff that
  held the tile for half a second got in the way of play zoomed in. Before that, a lone faint ring at 20px
  could not be seen, and a full-strength ring of at least 44px with a dark edge
  looked like a bonus. **It is never under `MIN_GLINT_PX`**, so from orbit a
  click is a spark that keeps the planet alive, and otherwise 1.8 tiles wide, so
  pushed in it stays on its tile (`glintSize`). **A hit on a shield is the
  same glint, red and shrinking as it fades** (`playHit`), and a shield placed
  is steel and grows (`playShielded`). **This player's glints are in its
  color's hue** (`hueOf`, handed down through `Globe.setClickHue`); a player
  with no color glints sky blue. Everyone else's are sky blue: a
  `TileUpdate` does not say who clicked. Not white, which vanished on the white
  of a flag. **A click out of view is not played** (`inView`): on the far side
  or off the screen it would cost frames and show nothing. With less motion a
  hit and a placement fade without changing size.
- `earth.ts` — the opaque sphere under the tiles, in the globe's light with
  `?gfx=earth`. See [The light](#the-light).
- `graphics.ts` — `graphicsOf`, which parts of the sharper, lit globe are on:
  the HD graphics setting, unless the URL names them. See [HD graphics](#hd-graphics-the-sharper-lit-globe-and-gfx).
- `shaders/` — GLSL for the display, picking, earth, star, enclosure and glint passes.
  `light.glsl` is not a pass but the light they share, pulled in with
  `#include ../light.glsl;` (vite-plugin-glsl's own include, not three's).

### Drawing only when something changed

**The loop draws a frame only when the one on screen has stopped being right.**
It used to draw every frame the browser offered, for as long as the tab was
open, and almost all of them were the same picture: measured at rest, before
any interaction, the drawing buffer was byte-identical across a second while
262k tiles and 98k pieces of outline were redrawn 60 times through it. On a
phone that is a flat battery for a still image, and it is why the game was
reported as a battery eater.

`drawsFrame` in `globe.ts` is the whole rule, kept out of the loop so it is
under test. Three things can ask for a frame:

- **The camera turned** — for a drag, the damping glide after one, a zoom, or
  the idle spin. The loop reads this from OrbitControls' `change` event as well
  as from its own `controls.update()`, because a wheel zoom is applied inside
  the wheel handler: `_handleMouseWheel` calls `update()` there, and that call
  is the one that moves the camera and clears the pending scale. Asking the
  loop's own `update()` afterwards gets `false`, and a zoom used to look to the
  loop like a still globe. `change` is dispatched by whichever `update()`
  actually moved the camera, so it is the signal that survives. It is cleared on
  the frame that draws it rather than the tick that reads it, so the cap below
  can hold a frame back without losing the move that asked for it.
- **Something the loop drives is still moving** — every `update` that animates
  answers a boolean: `blasts`, `bonusBox`, `enclosureEffect`, `clickEffects`,
  `clickGlints` and `TileField.setHover`. **The frame an effect *ends* on counts**: it is the
  one that takes the flash, the box or the highlight off the screen, so each one
  answers `true` on the tick it stops as well as while it runs.
- **Something outside the loop touched the scene** — `invalidate()`, which
  `applyChanges` calls for every claim, batch and rollback, and which the resize
  listener, the lapsed-box branch and `capture()` call for themselves. It is
  read and cleared once per tick, and only ever set from outside the loop, so
  clearing it on a tick that then skips its frame cannot lose one.

**The idle spin is capped at `IDLE_FRAME_MS`, and it is the one animation the
cap has to be generous with.** It is the only thing that moves with nobody
touching the page, and it is the first thing anybody sees. At a turn in thirty
seconds the surface goes by at a little over a hundred pixels a second: two
pixels a frame at sixty, four at thirty. Thirty was tried and the step is
visible as a step, so the cap is sixty — which still leaves half of what a
120Hz phone or a ProMotion Mac offers. The cap is lifted while the globe is
being handled and for `INTERACTION_GRACE_MS` after — the damping glide is
looked at closely enough to be worth every frame — and it never holds back a
frame that something *changed*: a claim or a blast is drawn as it comes.

**The cap takes the display's nearest frame, not the first one past it.** A cap
in milliseconds lands between two of the display's own frames, and waiting for
the first one strictly past it leaves the answer to a fraction of a
millisecond: 16ms against a 16.7ms frame came out 60fps, then 40, then 60
again. That unevenness is seen where the rate itself is not, so `drawsFrame`
takes `sinceLastTick` and allows half a frame of slack.

**The spin turns by the clock, not by the frame.** `controls.update()` is
handed `spinStep(sinceLastTick)`, and `autoRotateSpeed` is then read as **turns
per minute** — `SPIN_TURNS_PER_MINUTE`, 2, a turn in thirty seconds. Handed
nothing, OrbitControls advances a fixed angle per call instead, which assumes
every display runs at 60: the globe went round in fifteen seconds on a 120Hz
screen and thirty on a 60Hz one, and so travelled twice as far between the
frames that were drawn — the very distance the cap exists to keep small.
`spinStep` caps one tick at `MAX_SPIN_STEP_MS`, because a hidden tab is offered
no frames at all and the whole of that wait would otherwise arrive as one step,
with the globe somewhere else by the time it is looked at again.

**The uniforms the tile pass reads are written before the render, not after
it.** They used to be written at the end of the loop, for the next frame; with
a frame skipped whenever nothing moved, a size worked out for a frame that is
never drawn is a size that never arrives, and the tiles would be left drawn for
a zoom the outline had already moved off.

`preserveDrawingBuffer` stays off (see [Sharing the
globe](#sharing-the-globe)). A skipped frame draws nothing at all, so nothing is
composited and the last frame stays on screen; a drawn frame always clears and
redraws everything, so nothing accumulates either.

### The far side of the globe is not drawn

The earth's own sphere is opaque at radius 0.999, so **half of every pass is
behind it** — and was being run through its whole vertex shader before the depth
test threw it away. The display and border-line vertex shaders now drop it on
one dot product, before the four vertex texture fetches the painted flag costs.

What may not be dropped is what the earth's silhouette does not cover. A point
at radius *r*, an angle *θ* past the limb, projects to a screen radius of
*r·cos θ*, so it still shows while *θ < acos(0.999 / r)* — about 0.045 for the
tiles at 1, and `limbOf(lift)` for each outline pass. On top of that comes half
the tile's own disc, half the line's own width, and how far a blast may throw a
tile outward. `borderLines.test.ts` pins `limbOf` against the earth's radius.

Measured against the same frame with the culling off, the limb comes out
pixel-for-pixel identical. It is worth a few percent of those two passes and no
more — the vertex shader still runs for every point and still reads every
attribute, and only its body is skipped.

### HD graphics: the sharper, lit globe, and `?gfx=`

The two sections below — the screen's pixel ratio and the light — shipped on in
#254 and turned the globe **almost white, flickering as it turned**, for players
on Windows Chrome with an Intel GPU (ANGLE on Direct3D 11). Antialiasing off
(#255) did not fix it, and both were reverted (#256). It could not be reproduced
on a Mac (Metal), on SwiftShader, or on a Windows laptop with the same GPU
(Iris Xe, ratio 1.25) in either a dev or a production build. So the cause can
only be found on the screens that have it.

**A player turns them on as "HD graphics" in Settings**, off by default
(`Rendering`, `"plain"` or `"sharp"`, in the display settings): turned on for
everyone, a player who gets the white globe would have to find the switch. **Changing it rebuilds the globe**:
`antialias` is fixed when the WebGL context is made, so `rendering` is in
`useGlobe`'s dependencies and the map loads again.

**The URL still wins over the setting**, part by part, for the bisection below
(`graphicsOf` in `graphics.ts`, read once in `createGlobe`). With `gfx` in the
query only the parts it names are on, so `?gfx=` alone is the plain globe:

| `?gfx=` | Turns on |
|---|---|
| `ratio` | the screen's pixel ratio, capped at 2, instead of 1 |
| `aa` | `antialias` on the context |
| `earth` | the earth's own shader, in the light, with the glint |
| `tiles` | the light on the tiles |
| `halo` | the light on the halo |
| `light` | `earth`, `tiles` and `halo` |
| `all` | all of the above: #254 as it shipped |

Words add up (`?gfx=ratio,tiles`), and sit beside the rest of the query
(`?f=fr&gfx=halo`). A word it does not know turns nothing on.

**Off is the code from before #254, not the new code multiplied by zero.** The
light is compiled out with `#ifdef LIT` (three's `defines`, which leaves out a
`false` one), so a driver that miscompiles `light.glsl` never sees it; the plain
earth is three's standard material under the ambient light again; the plain halo
has its own `colour` uniform. What is left on every page is the ratio
arithmetic, which is a multiplication by 1.

**To use it**, send a player who has the bug the links, one per part, and ask
which come out white: `https://clickplanet.lol/play?gfx=all` first, which must
show the bug, then `ratio`, `aa`, `earth`, `tiles`, `halo`. Ask for
`chrome://gpu` too: it names the driver. **Once the culprit is fixed, take the
URL switches out**; the setting stays.

### CSS pixels in, drawing-buffer pixels out

**HD graphics draws the canvas at the screen's pixel ratio, capped at 2**
(`pixelRatio()` in `scene.ts`, `?gfx=ratio` alone); without it the ratio is 1. At
1, on a phone or a laptop the browser stretches every frame over twice its
pixels and the whole globe is soft. Past 2 is more than twice the work again for
a difference nobody sees at arm's length.

**Antialiasing comes with HD graphics** (`?gfx=aa` alone). #254 turned it on below a ratio of 2,
and it was the first suspect for the white globe on Intel; turning it off
(#255) did not fix that, so it is one of the switches rather than a verdict.

**Every size in pixels in this viewer is a CSS pixel**, and is multiplied by the
ratio on its way to the GPU: the tile's point size, the outline's width
(`halfWidthOf`), the keyline around a painted flag, the smallest a mark or a
ring of the bonus effects may be, the smallest debris. So are the thresholds:
`coarseHandover` and `flagPaint` are worked out in CSS pixels, or a sharper
screen would hand over at half the zoom. What is measured against
`gl_PointSize` stays in drawing-buffer pixels — `pixelsPerRadian`, the picker's
window, the one-pixel feathers that soften an edge.

The click is already in drawing-buffer pixels: `canvasPosition` scales the
pointer by `canvas.width / rect.width`. `resize` sets the ratio again, because a
browser zoom changes it. **Read the ratio back from the renderer, never from
`window.devicePixelRatio`**: a ratio that changes with no `resize` then leaves
the frame no sharper, but every size still agrees with every other.

### The light

**Only with HD graphics, or `?gfx=light` or one of its three parts** — see [HD graphics](#hd-graphics-the-sharper-lit-globe-and-gfx).
Without it the earth is three's standard material under an ambient light, the
tiles are unlit, and the globe reads as a flat blue disc.

**The globe is lit by one light, and everything on its surface calls the same
function for it** — `shaders/light.glsl`, included by the earth, the tiles and
the halo. Lighting only the earth would have left the flags floating flat on a
shaded ball.

**It is a studio light, not the sun.** It sits with the camera, up and to the
left, so the same side is always lit however the globe is turned: a real sun
would put half the players' countries in the dark. Half-Lambert, squared, wraps
it round the globe with no terminator, and the far limb keeps about half.

**`shadeOf` is exactly 1 at the middle of the disc**, which is what the camera
looks straight at and, zoomed in, the whole screen. A player at work on a
country sees its flags as bright as before there was a light; the lit side of
the globe seen whole comes out brighter still.

**The air is lit too.** `hazeOf` lays the halo's colour over the ground seen
edge-on, and `lit` shades it with the ground, so the rim is bright on the lit
side and fades on the far one. The halo itself is shaded by the same function,
taking the limb under it as its normal. `AIR` is the colour, and `COLOUR` in
`atmosphere.ts` is the same one for the unlit halo, until the switches go.

**Only the sea shines.** The earth adds a glint, read off the texture: the sea
is one deep blue whose blue stands clear of its red and green, and no land does
that. The photo itself is drawn as the standard material used to draw it,
`sRGB(texel · 2/π)`, so the sea is still the blue it was.

**None of it moves on its own**, so none of it costs a frame: the light turns
with the camera, and a still globe is still the same picture. See [Drawing only
when something changed](#drawing-only-when-something-changed).

### The zoomed-out view

Zoomed out, a tile is about a pixel and a half sampling a 100px flag out of one
shared 1300×1232 atlas, so the GPU picks a mip level that is the whole atlas
averaged and every country comes out the same mud. Turning mipmaps off swaps mud
for sparkle. Neither end works, so from far enough away the globe is drawn from
something coarser than a tile.

Every tile resolves **offline** to a *landmass*: a country's tiles split into the
separate pieces of land they actually form (Natural Earth 1:50m, 205 countries →
658 landmasses). Neither borders nor tiles move, so `npm run map:generate` writes
the whole table once and nothing recomputes it at runtime. Two tiles are the same
piece when they **touch on the lattice** and carry the same country — not when
they are within a tuned radius, which reached past the neighbours in places and
joined two islands across a strait into one flag painted over the water between
them. A landmass rather than a
country because a flag belongs to a piece of ground — one frame spanning mainland
France, Corsica, Guiana and Réunion would stretch the tricolour across half the
planet and paint nothing recognisable anywhere.

Each landmass flies the flag of whoever holds most of it, painted onto the sphere
with distance measured **along the surface**, so it bends with the globe and is
cropped by its own coastline. `BorderField` keeps the running count per landmass
and writes one row per landmass into a `DataTexture` the vertex shader reads.

**The painted flag is read under the pixel, not per tile.** It still reaches the
screen through the discs — that is what the widening is for — but what each
fragment shows is the flag at the point of ground beneath it, so overlapping
discs agree and the landmass comes out as one image at the resolution of the
screen. The vertex shader hands the fragment shader the frame rather than a
colour: where the tile's own centre falls in the flag, and how far that slides
under one screen pixel across and up. A pixel is a step on the *screen*, so the
ground step behind it is the one whose projection is a pixel — the tangent part
of the camera's axis over how much of itself the projection keeps, floored near
the limb where that divisor runs to zero. Those same two vectors are the
footprint the atlas is sampled over (`textureGrad`), which is the only reason a
mip level can be chosen at all: the frame is flat across a sprite, so without
them the driver takes the top one and point-samples a 100px flag into a few
dozen pixels.

One sample per disc was the whole flag's resolution before, and a landmass is
often only a few tiles across at the zoom where its flag is painted — six or
seven samples of Macedonia's sun or Serbia's arms, with neighbouring discs each
landing on a different one. Bands survived it; anything carrying a device came
out as pixel soup on exactly the countries that are too small to zoom past.

The lookup is skipped outright while `flagPaint` is 0, which pays for it: zoomed
in it was four vertex texture fetches and a frame per tile for a colour the
fragment shader mixed straight back out, and the fragment shader now skips the
tile's own atlas fetch at the other end, where the painted flag has the frame to
itself.

**Opacity is the leader's share, and the curve it goes through is not a free
knob.** Zoomed in, that share is already on screen as the fraction of discs
wearing the holder's flag, so the tiles show `share * 0.7` of ink no matter what;
the painted flag shows `share^contrast * 0.94`. Bend the curve and the summary
becomes fainter than the tiles it hands over to, and a country gets *brighter* as
you zoom into it — measured at 5x for Sudan at contrast 3. `borderField.test.ts`
pins this.

**One schedule owns the whole handover** — `coarseHandover` in `pointSize.ts`
drives the flag fading out, the tiles fading in, the disc widening being undone,
and the outline moving from over the tiles to under them. They only work
together: the flag reaches the ground only through the discs, so while it is
painted they must cover the ground (circles on this hex lattice cover it at
1.155x the spacing), and a tile you are about to aim at must not be fattened.
Running them on separate schedules left a band where the flag was painted
through a lattice with holes in it. `pointSize.test.ts` pins that too, and those
tests fail if the two are split again.

**A player can turn the painted flags off** — "Big country flags" in
Settings (`MapView` in the display settings). Players draw pictures with the
tiles, and the painted flag hides them. With it off
(`MapView` `"tiles"`), the handover in `pointSize.ts` is held at 1 at every zoom, so the
globe is drawn as it was before #52: each tile its own flag at its own size,
and the outline always under the tiles. Zoomed out that is the mud described
above, and that is the price of seeing every tile. The switch reaches the
running globe through `Globe.setMapView`, so it never rebuilds it.

`npm run flagFit` decides the rest: a flag that is only bands can be pulled to
the country's own shape and still say what it is, while one carrying a device is
cropped, anchored on the part that names it rather than on its middle. The
result is `static/countries/flagFit.json`.

**Every tile is in a country**, because both blobs come from one Natural Earth
query: a tile exists exactly where `groundOf` answers a code, and that same answer
is what the borders blob records. The 2,191 tiles that used to fall outside every
country are what the globe drew as discs floating on open water with no outline
round them. See [`/map/README.md`](../../map/README.md).

### The globe's texture

The sphere under the tiles is a photograph, and it used to be the third thing
that answered "sea or land" — disagreeing with both the tile set and Natural
Earth. It cannot be an authority: at 4096×2048 a one-tile island is three pixels,
so the mosaic blends it into open water however carefully the polygons are drawn.
A player must never see green with nothing to click on it, or a disc floating on
open water.

So `npm run earth` cuts it from the tile field instead. Each tile's Voronoi cell —
a hexagon of circumradius `spacing/√3`, the same 1.155× the renderer widens the
discs by when they have to cover the ground for the painted flag — is rasterised
onto an equirectangular image as `cover`, and the photo is corrected toward it:

```
out = photo + (cover - opinion) * (landColour - seaColour)
```

**It is the identity wherever the photo already agrees.** `opinion` is what the
pixel's own colour says, read off the same two colours, so the correction is zero
where the two agree — most of the globe — and full on a three-pixel island. The
two colours are the photo's own local averages over the land and over the water,
in 64-pixel blocks, so they follow latitude, depth and biome and nothing is
painted in a palette somebody chose.

**The two colours are found twice.** Taken straight off `cover`, a block whose only
land is an island the photo drew as water learns that land here looks like water,
and then leaves that island exactly as it found it — the one case this exists for.
The second pass weights each pixel by whether the photo and the tile field already
agree about it, and a block with no agreement left takes its colours from a
neighbour that has some.

Where land and water are the same colour — under cloud, on an ice shelf — there is
no direction to move a pixel along, and nothing is moved. `scripts/map/recolour.mjs`
holds the rule and `recolour.test.mjs` pins every case above.

An earlier version was a frequency separation, `mix(sea, land, cover) + (photo -
average)`. That is only the identity inside a block that is all land or all water;
along a coast it pushed the land further from the water in both directions and
moved 37% of the image.

### The countries' outlines

A thin dark-grey line runs between one country's ground and the next, and it is
the **same width at every zoom** — `borderLines.ts` builds each piece as a quad
laid out in screen pixels, not on the sphere, so pulling the view back thins
nothing. That is the only way a border survives the range this camera covers: at
0.5 the whole planet is half a screen tall, at 50 a single tile is a disc you can
aim at. Grey and not black: black carried fine at map scale but up close, where
the line is the only thing between two rows of discs, it read as a bar drawn over
the planet rather than a border on it.

**The line is not the administrative border.** It is the boundary of each
country's *tiles*. The tiles are the vertices of a geodesic sphere, so their
cells are its dual — a honeycomb — and the outline runs along the cell edges
between two tiles that belong to different countries. Natural Earth's border is
therefore moved onto the lattice, by up to half a tile, about 12 km.

That move is the point. A line on the real border crosses tiles, and a crossed
tile belongs to one side while reading as split between both. A line on the cell
edges passes *between* the discs: the tile field is 76% covered once zoomed in,
and the gap it leaves is exactly where this runs. So a tile is never cut, and
which side of a border it is on is never a matter of where the line happened to
fall across it. `borderLines.test.ts` pins the width against that gap.

**What is drawn is not that polyline but its quadratic B-spline**, four straight
pieces to a cell edge (`SAMPLES`). The lattice is a honeycomb, so the bare
outline turns 60° at every corner and reads as a staircase from a few zooms in;
the spline is the curve through the middle of every cell edge, reaching a quarter
of the way toward each corner without touching it.

That curve is not a compromise on the tile rule — it is the most a curve can be
smoothed and still obey it. The narrowest the corridor between two tiles of
different countries ever gets is at the middle of the cell edge between them, and
any line separating them has to thread that point; the spline goes through it
exactly, and at the corners, where there is half as much room again, it spends a
fraction of what it has. `borderLines.test.ts` measures the drawn line against a
lone tile's own cell and pins its near edge clear of the disc at every zoom the
fine pass is drawn at.

**Each pass carries the outline at the resolution it is looked at.** The over
pass is on screen only while the flag is painted, where a cell edge is at most
six pixels, so two pieces put it within a tenth of a pixel of the fine one and it
draws half the geometry — and it is the pass that is up whenever the whole globe
is. The fine one is only ever drawn pushed in.

**A run ends at every junction** — a corner three countries share, or where a
land border reaches the sea — which is the one corner the smoothing may not round
off. The generator cuts the runs there so the renderer clamps the curve to it,
and the three branches meeting there meet on the point rather than a fraction of
a tile apart.

**It is drawn twice, on either side of the tiles, and `flagPaint` picks.**
Zoomed in the outline sits just inside the tile shell, so the discs' own depth
hides whatever they cover and the line only ever shows in the gaps — which is
the whole width of it. Zoomed out that gap is gone: the discs are widened until
they cover the ground so the painted flag can reach it, and a line underneath
them would be invisible. So the same outline is drawn just outside the shell as
well, at the painted flag's own opacity, and the two hand over on the one
schedule that owns the rest of the handover. Neither pass writes depth, and both
are the same grey, so the stretch where they overlap — a coast, which has no
tiles on the sea side to hide the under pass — only ever comes out that grey.

`npm run borderLines` writes the geometry, from the coordinates blob and the
borders blob and nothing else. It rebuilds the whole `IcosahedronGeometry(1, 300)`
the tiles were cut from, because the coordinates blob holds only the land
vertices and a coast needs the sea around it; every tile has to land on a lattice
vertex or it refuses. A cell corner is the circumcentre of a lattice triangle —
the normal of the plane through its three vertices — which makes the cells a true
Voronoi diagram of the tiles and the corners meet exactly, so there are no seams
to cover up at the joins. The edges are then chained into runs, which is what
keeps the file to one corner per edge rather than two: 49,632 edges in 302 KB.
The smoothing is not baked in — the blob is the outline on the lattice, and how
finely it is rounded off is the renderer's business and four times the size.

**Pinholes are filled before the outline is traced.** A vertex in no country
takes its neighbours' country when at least four of the six agree and none
disagrees, twice over. Without it every one-tile lake, every strait one tile
wide gets an outline of its own, and every coast frays. It closes about 1,650 of
them — it was 2,500 before the two blobs agreed on where the land is — and takes a
fifth off the coastline's length.

It stays out of `/map`, unlike the two blobs it is built from: the backend has no
use for it. Where a tile is and who owns the ground under it are the game's rules
and are shared; how thick a line is drawn between them is this app's.

### Data flow

1. `useGlobe` calls `createGlobe`, which fetches the coordinates and borders
   blobs — in parallel, and before allocating any GPU resource, so an abandoned
   load never opens a context.
2. Ownerships are fetched in batches and fed to `TileOwnership`. **`createGlobe`
   resolves only once the last batch is in**, so `useGlobe` stays `loading` —
   with `territories`, the share fetched, for the progress bar — and the menu,
   the leaderboard and every other control stay hidden until then. The scene
   already turns and fills in behind the loader, but a click claims nothing
   before the map is complete: an empty map with an empty board is not a game.
   A fetch that fails after its retries fails the whole globe, and an abandoned
   one disposes it. On ready, `publishLeaderboard` shows the full board at once
   rather than up to a sample later. Each batch brings its tiles' shields
   too (see [Shields](#shields)).
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

6. `ClickBudgetMeter` shows what is left of the bucket, in the dock at the bottom
   (see [The screen](#the-screen-four-zones)). It warns
   before the wall, and shakes when a click hits it. **Its shape is read off the server's policy**: one
   pip per click in the burst (one bar past 12 of them), and the partly-filled
   pip is the click being granted back, at the server's own rate. Change
   `rateLimiter.burst` on the backend and this follows with no release here.

   **A slow refill gets a countdown.** When a click takes 1.5s or more to come
   back (`COUNTDOWN_FROM_S`; production is one every 5s), the meter says
   "+1 in 4s" under the bar, and says nothing at a full bucket. Past 12 pips
   the bar moves a sixtieth per click, too little to see, so a blue strip under
   it (`--click-budget-next`) fills once per click, as a pip would.

   The refill is animated from **one CSS custom property written per frame**,
   and each pip works out its own share of it with a `clamp()`; the count, the
   colour and the aria value are written only when the whole number changes.
   `useClickBudget` therefore re-renders when the *server* says something, not
   as the bucket refills — this sits beside a WebGL scene that wants the main
   thread. Under `prefers-reduced-motion` the fill steps four times a second
   instead of gliding.

   **It says when the toll slows the refill**: a chip before the countdown,
   "4× slower" (on a phone "4×", named in full), whenever `ClickBudget.price`
   says a slowdown above 1. The chip, the countdown and the offer below are a
   line beside the meter, never inside it.

   **The reading opens "Your clicks"** (`ClicksPanel`): a button laid over the
   reading (`.click-budget-open`), so `role="meter"` stays a leaf. On a desktop it
   is a popover over the dock, closed by the reading or Escape; on a phone, a
   sheet. It holds the count, the country's share of the map, the whole toll
   table from `BonusRules.toll` with the step the country is on marked, who else
   spends from the bank, and the offer. **It shows no seconds per click**: the
   reading's rate is the pace of the last click, which a change of flag only
   moves at the next click (see the backend's toll), so a time worked out from it
   would be wrong exactly when the player is looking.

   **A guest is offered to click faster.** A signed-in account refills
   `ClickBudget.linkedMultiplier` times faster (2 in production), into a bank of
   the same size. For a guest the
   server offers sign-in to, `Viewer` passes `onSignIn` and the dock shows
   "Sign in: clicks 2× faster" in its line — a button beside the meter, not in
   it, since the meter is a reading. It glows when the bucket is empty or a click
   is refused, the moment a guest meets the wall. **It is in the line only where
   it fits**: on a desktop with no slowdown chip. Otherwise it is in "Your
   clicks" and the Sign in tab, so the line never holds more than two things. It opens `SignInPitchModal`,
   which has the account panel's sign-in buttons. **The pitch sells more than
   speed**: a list of what a guest does not have — the clicks, a name, a color
   (guests are grey), a place on the board (players with a name are listed above
   the guests) and a streak flame (guests have none). Keep each line true: it names what
   the game does today, not what is planned. The account panel's guest text
   says the same. **Nothing is offered without the server's number**, nor with
   sign-in off.

   **It says when somebody else spends from the bucket.** The guests behind one
   address share one bank, and every player behind it shares the scope's, so a
   count can drop by clicks this player never made: another tab, or a stranger
   on the same carrier. `ClickBudget.sharedWith` is the server's answer to whose
   bucket the reading is: the dock's line shows a players icon named "Shared with
   the guests on your network" or "…everyone on your network", and "Your clicks"
   says it in words. Absent, it is the
   player's own and nothing is said. A guest who shares is offered "Sign in:
   your own clicks" instead, and `SignInPitchModal` says why.

   **It is one panel, and its width is set rather than grown.** The reading, its
   line and the inventory all live in `.click-budget-dock`, one row at the bottom
   centre that carries the only border and background; `.click-budget` itself
   draws nothing, so `role="meter"` stays a leaf with no button inside it. The
   dock's width is `--dock-width` (620px, 460px under 1280px, the screen less 24px
   on a phone): shrink-to-fit handed the widest part the say before, and the
   others trailed dead space. The reading is `flex: 1` and the slots keep their
   size, so the bar takes what is left. The panel's border is what carries state: the inventory's glow
   first, then low, empty and refused, in that source order so red at the wall
   beats a bonus being switched on. The reading still jolts on a refusal
   (`.click-budget-refused`, taken off on its own `animationend`), but the red
   flash is the dock's, through `:has()`.

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

**A click predicted to be shielded paints nothing**, so it has no claim to
take back.

Rolling back on *any* failure, including a transport fault, is deliberate: if
the click did land and only the response was lost, the stream's echo repaints
it, and if the echo arrives first the rollback is already a no-op.

## Charges

A bonus box holds a **charge**, which has no clock and is never used on its own:
a refill (the click bank, filled when the player presses it), a bomb (one drop), a
stack of enclosures (one shape each, `maxTiles` at most), a pool of spread
clicks and a pool of shields. The server keeps them per account, in postgres:
a refill and a bomb at most, up to 3 enclosures, up to 8 spread clicks and up to
12 shields. A box adds a random 1 to 3 enclosures, 1 to 4 spread clicks or 1
to 3 shields, capped at the size; the reward it announces
is what was kept (`+2 spread clicks`), which is what the server answers in
`ClaimBonusResponse.amount`.

**`PlanetBackend` holds `Charges` (`domain/bonus.ts`) and nothing pushes them.**
It reads `GetCharges` at load, with the token in hand and never a fresh one, and
again when a click goes out under a new token (`followSession`): the charges are
the account's. `ClaimBonusResponse.charges` replaces them on a claim, and
`UseRefillResponse.charges` on a refill, which also brings the full budget, and
`PlaceShieldResponse.charges` on a shield placed. Otherwise
it follows its own calls: an accepted click **sent with spread on** takes a spread
click off, this player's own `tilesEnclosed` takes an enclosure off, and a drop
takes the bomb off at once and gives it back only if the call never reached the
server; a shield is taken off the same way. The click answer says nothing about charges, on purpose (see the backend's
CLAUDE.md). A charge spent in another tab stays on screen until the next read. It
reaches the globe through `BonusHandlers.onCharges`, and `useGlobe` hands it to the
inventory.

**The sizes are rules, read once**: `GetBonusRules` (a cached GET) answers the
blast radius, the enclose's `maxTiles`, the spread pool's size, the enclosure
stack's size, the shields' pool size and how many can stand on one tile
(`tileShields`) as `BonusRules`, through `onRules`. A page open across a change of rules shows the old sizes
until it is reloaded.

### Off by default, one at a time

**Nothing is used until the player says so.** Spread, enclose and shield are
switches (`Switches`), off at load and never turned on by the client. Every click carries
them: `TileClicker.clickTile(tile, country, switches)` sends
`ClickRequest.spread` and `enclose`, and the server spends a charge only when its
switch is on. Shield is not sent: it changes what a click on the player's own tile
does (see [Shields](#shields)). A switch goes off by itself when its pool runs out
(`switchesHeld`), so it never says a click does something it will not.

**One bonus at a time**, the bomb included: `switched` turns the other switches off
when one goes on, aiming the bomb switches them all off, and switching one on puts the
bomb away. `globe.ts` holds the one copy (`setSwitch`, `onSwitchesChange`). The
server refuses a click with both switches on (`INVALID_ARGUMENT`), before it
writes or spends anything.

### The inventory

`components/Inventory.tsx` is the section that shows them, the **right half of
the dock** (the meter takes it as `children`, and shows it even with no budget).
It draws no border or background of its own: the one panel is `.click-budget-dock`.
One slot per kind, always shown, each drawn with its box's
icon (`BonusIcon`) in its box's colours (the `--bonus-*` properties in
`BonusAward.css`, shared with the announcement). An empty slot is dimmed, and a
press on it does nothing but say how to get one. A pool shows its count against
its size (`5/8`, `2/3`), and a word over the icon says what the slot is doing
(`On`, `Aim`, `Full`, or `New` for a kind held and never pressed).

- **Refill** fills the bank (`Refiller.useRefill`). **On a full bank it sends
  nothing** and says "Full": a refill there would be wasted. The
  server refuses it too, `FailedPrecondition`, read as `BankFullError`, and spends
  nothing.
- **Bomb** aims it, or puts it away (`Globe.setArmed`).
- **Spread**, **Enclose** and **Shield** switch (`Globe.setSwitch`),
  `aria-pressed`.

**A slot speaks in a bubble, never a `title`.** Most of the players who came from
TikTok play on a phone, where a `title` never shows, and they asked in the chat
what enclose and shields do. So `Bubble` (`components/Bubble.tsx`, drawn in a
portal on the body like the reaction popup) says one line over a slot: what it
does while a mouse rests on it, how to get one on a press of an empty slot, and
how to use it on a press that switches it on or aims it, for its first
`LEARNING_USES` (3). The line goes after a few seconds, or as soon as the slot's
charge is spent. The same line is in the slot's `aria-describedby`.

**The globe says when a bonus did not do what the player meant**
(`GlobeOptions.onNotice`, a `BonusNotice`), and the slot says it in its bubble:

- **Shield on, a tile of another flag**: the click is an ordinary one, so the
  bubble says "Tile taken. Tap it again to shield it", or that shields go on the
  player's own tiles when a shield stopped the click.
- **Shield on, a full tile**: "Full" on the slot and a line saying so.
- **Enclose on, a click that took a tile and closed no shape**: "Shape is not
  closed or is too big (25 tiles max)". **This is guessed on the client, on
  purpose**: `Click` answers nothing about bonuses so a shadow-banned caller
  cannot tell its clicks are dropped (see the backend's CLAUDE.md), and the
  server cannot tell an open shape from one too big either. The globe waits
  `ENCLOSURE_WAIT_MS` (1.5s) after the click is accepted for this player's own
  `tilesEnclosed` closed at that tile, and says it when none came. A banned
  player learns nothing new: it sees no enclosure either way.

**What a player has learned is kept in the browser** (`clickplanet-bonus-guide`,
`domain/bonusGuide.ts`, `useBonusGuide`): how many times each kind was switched
on (up to `LEARNING_USES`) and which kinds it has won. `Viewer` holds the one
copy and hands it to the inventory and the award. **The first box of each kind
stays up** until "Got it" or Escape (`BonusAward`'s `kept`): the passing award
lasts 2.9s and a tap anywhere closes it, which in a clicking game is before the
line under it is read. Later boxes of that kind pass as before.

**It does not fold**: five slots are one row, labelled from 1280px and icons
with their counts below that (the name stays in `aria-label`). A fold on a row
this small hid the one thing that says the next click does more than paint.
The dock glows while something is on or aimed.

## Quizzes

A banner at the top of the screen: press it and you get a question with three
choices and **a short clock**. A right answer is worth a charge, the same as a
caught box; a wrong one, and running out of time, cost nothing.

`src/app/quiz/` is the whole of it — `useQuiz.ts` drives, `Quiz.tsx` draws,
`Quiz.css` styles — plus `domain/quiz.ts` for the shapes and the two pieces of
arithmetic a countdown needs. The backend half is `QuizMaster` in
`backends/backend.ts`.

**It never touches `globe.ts`.** A quiz is DOM at the top of the screen, not an
object in the scene, so `useQuiz` subscribes to the feed itself rather than
being handed offers down through the globe the way a flying box is. What a right
answer wins reaches the inventory the way every other charge does: the backend
holds the charges and tells whoever is listening.

**The client never knows an answer before it gives one.** The bank lives on the
server and is deliberately not shipped to the browser (see
[`/quiz/README.md`](../../quiz/README.md)). `listenForQuizzes` brings a banner
carrying **a token and a deadline and nothing else**; `openQuiz` brings the
question and its three choices and **starts the server's clock**; `answerQuiz` is
the first thing that says which of the three was right.

**One state machine, `QuizState`:**

```
idle → offered → opening → asking → answered → idle
```

Each phase is a different thing on screen *and* a different thing to a player:
`offered` is an invitation that costs nothing to ignore, `asking` is a clock
already running. Anything that goes wrong — a token the server will not honour, a
stream that dropped — falls back to `idle`, because a quiz nobody can answer
should leave nothing behind. **The token rides through the state** rather than
sitting beside it, so there is no way to answer one quiz with another's token
while a banner and a question are changing places.

**A banner arrives only into an empty screen.** The server will not offer a
second, but a stale one arriving mid-question would take the clock away from
under somebody already reaching for a choice.

**The banner gives nothing away** — not the question, not the choices, and not
what it is about. It named the subject country and flew its flag once, which
read as a harmless teaser and was not: that flag was the answer to **417 of the
bank's 1014 questions**, every "Tallinn is the capital of which country?" and
every "which of these has the most people?". The fix is not to pick safer
templates, because a teaser that has to be checked against every question in the
bank leaks again the first time a template is added. What makes the banner worth
pressing is the charge behind it.

**The charge is teased as a slot reel** (`PrizeReel`): the five bonus boxes roll
past under a gold "?", on the banner and again beside the clock. The server picks
the kind when it offers the quiz and does not send it, so the reel shows every
kind and stops on none. A right answer is the first time the client knows, and
the box then spins in as the bonus won.

It draws no countdown of its own either, because it is free to ignore, and it
goes away by itself.

**The countdown bar starts at what is actually left, not at full.** The five
seconds are the server's and they began when it answered, so a slow round trip
has already spent some of them; a bar that started full would promise time the
player does not have. From there it is **one CSS animation on `transform`** to
empty, started that far in with a negative `animation-delay` — on the compositor,
so a whole window of continuous animation costs nothing beside a WebGL globe
drawing at the same time, where a `width` transition would relayout every frame.
An animation rather than a transition because **a pressed choice pauses it**
(`animation-play-state`): the bar stops where the answer was given. The seconds
beside it are drawn by React once a second (`secondsLeft`, `untilNextSecond`), and
in the last three the coin and the card's glow turn red. Under
`prefers-reduced-motion` the countdown **stays**: it is information, not
decoration.

**Running out of time is sent as a choice past the end of the three.** The server
reads it as wrong, which it is, and answers with the right one — so a question
nobody managed to answer still says what it was. There is no other way to learn
it.

**The result says which one was right whether or not that was the one pressed.**
A wrong answer costs nothing, so the only thing left to give back is the answer.
It draws the three choices again where they were, the right one green and a
wrong pick red, **as list items and not buttons**: nothing on the result can be
pressed, so nothing on it reacts to the pointer.

**The quiz and `BombNews` both want the band at the top**, and the bomb line is
the one that gives it up (`lowered`): four seconds of news nobody presses moves,
a question somebody is answering does not.

**Three sounds, one switch.** `quiz` when the banner arrives (the same reason
`bonusSpawn` exists: it is at the top of the screen and the player is looking at
the globe), `quizRight` and `quizWrong` when the answer lands. All three answer
to one `quiz` switch through `switchOf` — a quiz is one feature making three
noises inside ten seconds, and three lines in the settings panel for that is two
lines too many. `quiz` is the banner's sound, which is also what the panel plays
when the switch is turned on.

`quizWrong` is gentle on purpose, and rounder and higher than `refused`: a wrong
answer costs nothing, so it says "ah well" rather than "no". A sound that
punished a guess would make guessing feel expensive when it is free.

`useQuiz` takes the player and holds it in a ref, so a settings toggle does not
resubscribe the feed; it is optional, and a page with no sound wired still works.

`giveQuiz()` in the console puts one up at once against the fake backend, beside
`giveBomb()` and `giveBonus()`. The fake's bank is three questions — it is there
to develop the banner and the card against, not to be played.

## Bombs

A bonus box can hold a bomb (`BonusReward` kind `bomb`), kept until it is dropped
anywhere on the planet. It clears every tile within `radius` of where it lands —
the server's call, not this client's.

**A bomb is aimed or put away.** It is kept until it is dropped, so it cannot stay
aimed: while aimed, a click claims no tile. It is never aimed on its own, not
even when it is caught: the inventory's bomb slot aims it and puts it away
(`Globe.setArmed`), and Escape puts it away. The radius comes from the rules, so a bomb still in hand
after a reload can be aimed. The pieces:
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
and the colours stay. A blast off screen gets a red edge pointer with a drawn burst on it
(`blastMark.ts`) — the same component as the bonus box's, which carries a drawn
question mark (`questionMark.ts`). Neither is a character: a glyph is a
different picture on every platform.

## Shields

A shield is a charge (`Charges.shields`) placed on a tile the player's flag
holds, up to `BonusRules.tileShields` on one tile. **A tile's shields are
part of the tile**, as its owner is, and every screen shows them. The server
decides what a click does; this client predicts its own. The pieces: `Shielder`
in `backends/backend.ts`, `domain/shields.ts` the count per tile and the rule,
`tileField.ts` and the display shaders the drawing.

- **The count comes with the tile.** Each map batch lists its shielded tiles
  (`GetMapResponse.shields`, read into `Ownerships.shields`), and every
  `TileUpdate` says the tile's count after it. `TileShields` keeps one number
  per tile and, like `TileOwnership`, lets a live update win over a batch that
  was already in flight.
- **The server zeroes the count on every change of owner**, so an update that
  changes the owner carries 0, and nothing here ties a count to a flag. **An
  update whose country is its previous country changes the shields alone**: a
  strike or a placement. It moves no tile on the board and plays no click
  glint.
- **A bomb strikes the shielded tiles in its blast** (`BombDrop.struck`): each
  loses one shield and keeps its flag, and is not in `cleared`. The strikes
  wait for the impact with the clear, and a tile update that arrives meanwhile
  wins its tile, as it does for the clear.
- **Shield is a switch.** While it is on, a click on a tile the flag holds sends
  `PlaceShield` (session-gated, not throttled) instead of a click, and spends
  no click. A tile already at its most sends nothing and the slot says "Full".
  Any other tile gets an ordinary click.
- **A click on another flag's shielded tile is shielded** (`outcomeOf`): it costs
  a click and takes nothing, so the globe paints no flag and plays the hit; the
  update brings the count. A take is predicted only on a tile with none, so a
  paint never has a count to drop. When a prediction is wrong, the update that
  follows carries the flag the server kept, and replaces the paint as any update
  does.
- **Spread and enclose follow the same rule on the server**; their updates bring
  the counts.
- **It is drawn in the tile shader**: one more per-tile attribute (`shield`)
  and a texture of marks (`shieldMarks.ts`), drawn once on a canvas: the
  inventory slot's shield in steel and ink, then the same shield with each
  count from 1 to `tileShields` on it, in the display face. The shader picks
  the tile's cell, and shows the plain shield until the tile is 16px across
  and the count from 22px. **The count is on the shield, not on a badge in
  its corner**: a badge leaves a digit a third of the tile, unreadable
  until about 40px. The shield is 80% of the tile, so the flag still shows
  around it.
- **A shield fades out with the tiles** (`1 - flagPaint`): under the painted
  flags from orbit there is none. With the painted flags off, the tiles stay
  and so do their shields. Nothing is allocated per tile and nothing
  animates, so a shield costs no frame.
- **A count that drops glints red, one that rises glints steel**
  (`clickGlints.ts`). This player's own hit and placement play at once, and their
  echo is skipped (`ownHits`, `ownPlacements`).

## Sharing the globe

The game's own map is the marketing material, so the globe can be photographed
and the picture taken out of the browser. `src/app/share/` holds it:
the "Take a picture" tile under More (`MorePlace` in `Menu.tsx`) takes the shot,
`takePicture.ts` captures and composes it,
`drawShareCard.ts` draws the card, `SharePreview.tsx` shows it, `ShareActions
.tsx` is the row of buttons under it and `deliverShare.ts` is what they do.
`useSharePicture.ts` holds the one picture there is at a time.
`src/domain/shareCard.ts` holds everything decided before a pixel is drawn.

**The camera is a tile under More, not a button on the globe.** It used to sit
on the canvas, bottom-left, beside its subject; it was one of six things over
the globe on a phone, and it is pressed rarely. On a phone the press closes the
sheet first, so the globe is what the preview shows the player framed.

**Its label never changes.** What answers the press is the preview opening.
Working is said by the tile dimming (`aria-busy`, disabled).

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

**The link switches whoever opens it to its flag.** `main.tsx` reads `?f=`
with `sharedCountry`, before the first render, and the game starts on that
country over the one in storage, then stores it as if it had been picked. An
unknown code is ignored. The parameter is then taken out of the address bar,
as the sign-in code is: left there, a reload would undo a flag the player has
switched since.

**The link is drawn into the image**, not only attached to it: a picture is what
survives being reposted. It is drawn in the text face (`--font-text`, Rubik)
rather than the title face, which has no lowercase — a query parameter reading `?C=PS` is a link that
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

**The capture is the drawing buffer**, at a ratio of 1, or at the screen's
capped at 2 with HD graphics (see [CSS pixels in, drawing-buffer pixels
out](#css-pixels-in-drawing-buffer-pixels-out)): a phone captures around 390×844,
or 780×1688 with HD graphics, and a ratio-1 desktop its CSS size. `cardSize` lifts a small one to a
short edge of 720 — the globe softens a little and the flag and the counts stay
crisp, which is the half anyone reads — and caps the long edge at 2400 so a
share sheet will still take the file.

**And the card is the middle of the frame, not all of it.** A phone's frame is
a 1:2.2 column that every timeline either shows as a sliver or crops for you;
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

## Clips

**A clip is the real globe playing back a war, for TikTok, Shorts and Reels**: 1080×1920, 8.5 to 22s, a headline,
the map moving under it from the first frame, and the link at the end. `npm run clip` makes them from a replay
and **chooses everything itself**: the stretch of time, the place, the headline, the camera, the look and the
length. Each comes with a `.txt` holding the caption to post. `--count 3` makes the three best stories, for a
person to pick one; `--plan` prints what they would be and renders nothing, and `--preview 3` renders only the
first 3 seconds; every choice can be forced (`npm run clip -- --help`).

- **The replay is the backend's** (`planet.v1.AdminService/GetReplay`, see the backend's CLAUDE.md): the map as it
  was at the start, and every act after it as the live stream sent it. `npm run clip:fetch` asks for the last
  72 hours over SSH (`--ssh`, or `CLICKPLANET_SSH`); `npm run clip:record` follows the public stream for a few
  minutes instead, to try the generator without the operator tool.
- **It is the game's own globe**, so a clip shows exactly what a player sees: `clip.html` is a page of its own,
  served by `npm run dev` and **left out of the build** (it is not in `rollupOptions.input`). `src/clip/main.ts`
  builds the globe with `createGlobe` over `backends/replayBackend.ts`, which plays the replay through the same
  decoders as `PlanetBackend` (`updateOf`, `bombOf`, `spreadOf`, `enclosureOf`): tile updates in batches, and a
  bomb, a spread or an enclosure as an effect. `ReplayBackend.cut` opens on the map at a later time.
- **The recorder owns the clock.** `src/clip/virtualClock.ts` replaces `performance.now` and
  `requestAnimationFrame`, so a frame is drawn only when `window.clip.frame(i)` asks, and a bomb takes its second
  of video however long a frame takes to capture. **No CSS animation or transition on the page**: they run on the
  real clock. The overlay is set from JS each frame.
- **The camera is the clip's** (`GlobeOptions.director`): with one, `createGlobe` turns OrbitControls off and
  aims the camera at the shot it is given every frame. The game passes none.
- **The text stays where TikTok draws nothing** (`--safe-*` on `#clip` in `clip.css`, measured on a phone): a tall
  phone crops the sides, the tabs cover the top, the buttons run down the right from the middle, and the name and
  the caption cover the bottom. The counter sits left of the buttons, and the call to act in the top half, darkening only that half: the map
  pulled back out stays in sight under it. The call
  shows the flag it asks the viewer to fight for: the country to defend, the flag that strikes back, both sides of a
  battle, or a continent's own flag (Europe's alone, `static/countries/svg/eu.svg`, from the same set as the others).

**`src/domain/clip/` is the director**, pure and under test:

- **`window.ts` finds the candidates**: for each length from 1 to 24 hours, the busiest stretch of a few places
  far apart, counting the tiles taken from another flag in each 10° cell and the eight around it. Filling empty
  ground is not war. A window asked for (`--since`, `--until`) is split into its places the same way, so a war
  next door is a story of its own.
- **`front.ts` finds the front** of a candidate: the point where most tiles changed hands, and every change within
  about 2,900 km of it, so a war in France brings in England, Spain and Germany. Once the story is known, the front
  is found again from its own flags' fighting alone, so a war next door (Israel in Turkey) does not pull the
  camera off Belgium's; with `--country`, it is that flag's fighting from the start.
- **`story.ts` writes the story** (`storyOf`): the flag that took the most there, the flags it took from, and
  where. Nearly all in one country (90%) is **"X IS INVADING FRANCE"**; spread over several, it is **"X IS
  ATTACKING"** the continent holding 70% of it (`static/countries/regions.json`, written by `npm run regions` from
  the snapshot the map is cut from), else **"X IS INVADING EGYPT AND TURKEY"** when two countries hold 70% of it,
  else the world. Not a sub-region: "defend Western Europe" is not how anybody talks. A flag taking back its own ground, or its own continent from a flag from elsewhere (Belgium taking Europe
  back from Palestine), is **"X STRIKES BACK"**; a flag that already held most of the country
  when the story starts is **"X IS KICKING Y OUT OF AUSTRALIA"**, since the opening shot shows its flag there
  already; a second flag taking 60% as much makes it **"X VS Y"**, but only when the two are at war, a quarter of
  what one took taken from the other: Israel and Belgium both taking Europe from Palestine are allies, not a battle.
  `src/clip/overlay.ts` words it.
- **A story is about who leads the fighting** (`castOf`): its flags have to take 35% of everything taken around
  it. Below that, flags of one continent taking it back together, with half of it between them, are the story,
  **"EUROPE STRIKES BACK"**, under the continent's flag with one counter for them all. Otherwise nobody leads it
  and it is skipped: Germany taking its own land back while Belgium and Israel made the war around it.
- **A flag thrown out is its own story** (`routOf`): once the flag a story takes most from has lost half of what it
  held in the place (300 tiles or more), a continent's flags taking it back together become **"PALESTINE GETS
  KICKED OUT OF EUROPE"**, told from its side with its counter falling against the continent's, and one attacker
  becomes **"ISRAEL IS KICKING PALESTINE OUT OF EUROPE"**. A flag taking its own ground back still strikes back.
  Two stories about one flag thrown out of one place are told once, the better one.
- **The words say nothing the map says better.** Only a battle has a line under its headline, "The battle for
  France": no count of tiles and no "in 3 hours", which the counter shows and which read as written by a machine.
  The caption is the headline and the question its call to act asks ("Who stops them?" to defend, "Who joins
  them?" to fight for a flag striking back, "Pick a side"), the site as plain text (a caption's
  link cannot be clicked) and the account's tags with the place's. Never the attacker's: a flag's tag can be a
  political feed.
- **`music.ts` picks the anthem under the clip**, since YouTube Shorts cannot add a sound to an upload: the anthem
  of whoever makes the moves, the leading flag's, else the other side's in a battle, or the continent's when a
  continent strikes back. Never a loser's: a clip with none plays none. They are the game's own anthems (see [The
  leader's anthem](#the-leaders-anthem)), US Navy Band recordings in the public domain, and two the game does not
  play, vendored by `npm run clip:anthems` into `scripts/clip/anthems/`: the Anthem of Europe (Navy Band too) and
  Palestine's Fida'i, which the Navy Band never recorded, in an instrumental under CC BY 3.0. A recording under a
  licence carries its credit, and the caption of every clip it plays under ends with it. It fades out over the
  last 1.2s. `--silent` leaves it out, for TikTok and Instagram, where a sound is added when posting.
- **`score.ts` ranks the candidates**: the tiles taken from another flag, over the square root of the hours, times
  the countries they were taken in (up to 4). A short war over several countries beats a long filling of one. A
  story already told by a better candidate (same attacker, same place) is dropped.
- **`solidity.ts` skips graffiti.** For each tile the attacker took and holds at the end, the share of its 6
  neighbours it holds too: about 1 for land taken, 0.56 for names written across Canada. Under 0.75 the story is
  skipped, and `--plan` says so. So is a story placed in "the world": its tiles are spread over several continents
  and there is no one place to show, and "Israel is attacking the world" is a line no clip may carry.
- **`look.ts` picks when the camera comes back out of the tiles.** From far, a landmass's painted flag only
  changes when its biggest holder does (`flipsOf`, over the borders blob, read 8 times along the changes, so a
  landmass taken and taken back counts too). A front too wide to frame closer than
  `TILES_ZOOM` (3) is **flags** when at least 2 landmasses changed their biggest holder and those hold 2,000 tiles
  or more: a steamroll, which comes back out halfway so its painted flags change on screen. Everything else is a
  **dive**, which comes back out at the end.
- **`camera.ts` opens on the map and dives into the tiles**: every clip opens on the middle of the front's
  changes, zoomed until 95% of them fit but never closer than a continent (`openingOf`), so the first frame is the
  map with its painted flags. It holds there 0.3s (`--hold`), then dives in 0.7s past the zoom the painted flags
  are gone at (`tilesZoomOf`), to where the most tiles change hands, so the fight is seen tile by tile. It follows
  the densest fighting, and comes back out to the opening as the look says. **Every clip ends pulled back out**,
  and holds there 1s before the call to act while the last tiles change hands: close-ups are for the middle, and
  the end shows the rest of the map as it is now. **It never sits still**: where the
  fighting crosses less than 0.4 screens a second, it breathes, out to where the painted flags show and back into
  the tiles every 3s. **It flies to every bomb on the
  front**, close enough for the blast to be a fifth of the screen, and holds there while it goes off. The globe is
  always drawn with the painted flags on, so the zoom alone hands them over to the tiles, as in the game.
- **`pace.ts` spends the clip on what happens and nothing else**: the replay's clock jumps over every quiet
  stretch, so the map moves from the first frame to the last. **A clip is as long as its camera has somewhere to
  go**: 6s for a fight in one place, however many hours it lasted (room for the dive, the close-ups and the pull back
  out), and 1.2s more for every screen the fighting
  crosses up close (`screensOf`, on the camera's smoothed path), up to 15s. A bomb adds the 2s it holds the clip
  still for, while it falls and goes off. The last 2.5s are the call to act, and nothing runs past 22s.

**`scripts/clip/render.mjs` is the recorder**: it starts Vite, serves the replay at `/__clip/replay.json`, opens
headless Chrome at 540×960 at 2×, waits for `window.clip.ready`, then for each frame calls `window.clip.frame(i)`,
takes a screenshot and pipes it to ffmpeg (libx264, `yuv420p`). It needs ffmpeg and Chrome (`CHROME_PATH`).

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
- `SoundSettingsPanel.tsx` — the switches, in the Settings place
  (`settings/SettingsPlace.tsx`). Turning a sound on previews it.

**Audio is locked until a gesture.** The `AudioContext` is only created by the
first `pointerdown`/`keydown` on the window, so a bonus box or a chat message
before the player has touched the page is silent, by design of the browser.

Where each one fires: the click and the refusal in `globe.ts`'s click handler
(`reportClickFailure` returns whether the server refused — a transport fault is
not a "nope"); the box appearing and being caught at the same places the box
itself does; the bomb when its broadcast arrives, with the boom scheduled
`IMPACT_DELAY` later so it lands with the tiles, quieter for someone else's,
and a splash instead of a blast when the drop has no tile under it (the ocean);
your own spread click and closed shape when their broadcast comes
back, so each lands with its effect on screen. **Only the player who made one
hears it.** An enclosure carries `yours`; a spread says nothing of
whose it is, so `domain/ownClicks.ts` remembers the tiles this client clicked in
the last 3s and a broadcast on one of them, for the same country, is taken as
ours. They have no switch of their own: `switchOf` puts them under the tile
click's;
the chat in `ChatPanel` for a message that is not yours. **Your own message is
filtered on your name as well as on `mine`**: its broadcast can arrive before
the send answer that fills `mine` in. A guest's name is only known once a
message of its own came back, so its first can still ping.

### The leader's anthem

The one sound that **is** a file. `src/app/anthem/` plays the national anthem of
the country leading the map, on a loop, with a player in the leader's frame on the board.
The recordings are the US Navy Band's (public domain); `npm run anthems` downloads
them, normalises loudness, trims the silence so the loop has no gap, and writes
`static/anthems/<code>-<hash>.m4a` plus `anthemsAsset.ts`, which pairs each file
with the anthem's title (`TITLES` in the script, written by hand: Wikidata's
labels mix titles with "National Anthem of X"). It needs ffmpeg.
Territories share their country's file (`SHARES` in the script); a country with
no recording shows the player with its play button off.

- `domain/anthemLeader.ts` — `followLeader`: a new leader must hold first place
  for `HOLD_MS` (15s) before the music follows it. The first leader plays at once,
  so `Viewer` hands `useAnthem` no board until the map is loaded: the board is
  sampled while the batches arrive, and a half-loaded map's leader is not the real one.
- `anthemPlayer.ts` — two `<audio>` elements crossfade through Web Audio gain
  nodes. **Not `audio.volume`: iOS ignores it.** A muted or hidden tab fades out
  and pauses rather than streaming silence.
- `useAnthem.ts` — ties the board and the settings to the player, which is built
  once, so a tick or a toggle never restarts the music.
- `AnthemControls.tsx` — the player, in the board's frame for the first country (`Leaderboard`'s `anthem`): the anthem's title, then the country. Play and volume write `SoundSettings.anthem`, so
  it and the settings panel never disagree. Pressing play lifts the master switch.
  It plays the country the music follows, which holds first place `HOLD_MS` before
  it changes, so for that long after a new leader the frame shows the last one's
  anthem, under its own flag.

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
- [`auth/v1/auth.proto`](../../proto/auth/v1/auth.proto) — the mint, the
  account and sign-in (`AuthService`)
- [`session/v1/session.proto`](../../proto/session/v1/session.proto) — the
  deprecated mint, no longer called
- [`player/v1/player.proto`](../../proto/player/v1/player.proto) — the
  username and who is playing (`PlayerService`)
- [`seasons/v1/seasons.proto`](../../proto/seasons/v1/seasons.proto) — the
  current season (`SeasonService`)

`ChatMessage.sentAtUnixMs` is an `int64`, which `protoc-gen-es` gives you as a
`bigint` — `chatBackend.ts` converts it at the edge so nothing above it deals in
two number types.

## Static assets

These assets are **content-addressed**, because `public/_headers` caches
`/static/*` for a week and a regenerated file under a stable name would be
served stale. Each has a generated TS module holding its current URL — do not
edit those by hand, and do not add a `?ts=` cache-buster, which defeats the
cache entirely:

- `/static/coordinates-<hash>.bin` and `/static/borders-<hash>.bin` — where every
  tile is, and whose ground it sits on. Fetched at runtime by `points.ts` and
  `borderField.ts`; URLs in `coordinatesAsset.ts` and `bordersAsset.ts`. Formats
  in `coordinatesBinary.ts` and [`/map/README.md`](../../map/README.md).
  **These two are not ours alone.** The source of truth is the monorepo-shared
  [`/map`](../../map/README.md), which the backend builds its tile adjacency from
  and its admin tools read; `static/` holds a generated copy, exactly as
  `src/gen/grpc/` holds a copy of the proto contract. `npm run map` re-copies the
  coordinates blob, and `npm run map:generate` rewrites **both** — one command,
  because a tile exists exactly where the borders blob says a country does and the
  two must not be able to disagree. **Run the backend's `make map` after it, and
  commit all three copies of each**, or the two apps disagree about what a tile id
  means. It renumbers every tile, so it also writes the postgres migration that
  follows the owned ones across; see `/map/README.md`.
- `/static/borderLines-<hash>.bin` — the countries' outlines, traced onto the
  tile lattice, fetched at runtime by `borderLines.ts`. URL in
  `borderLinesAsset.ts`. Regenerate with `npm run borderLines`, which reads both
  blobs above and so needs them in place first — and **regenerate it whenever
  either of them changes**, or the outline is drawn around a map nobody is
  playing on. Unlike them it is this app's alone and is not copied to `/map`.
- `/static/earth/earth-<hash>.jpg` — the globe's texture, its coastline cut from
  the tile field. URL in `earthAsset.ts`. Regenerate with `npm run earth`, which
  reads the coordinates blob — so **after `npm run map:generate`**, or the coast
  has discs in the water again. See [The globe's texture](#the-globes-texture).
- `/static/countries/atlas-<hash>.png` — the flag sprite atlas. URL and pixel
  size in `atlasAsset.ts`. Regenerate with `npm run atlas`.
- `/static/reactions/<name>-<hash>.svg` — the chat's reaction images. URLs in
  `app/chat/reactionsAsset.ts`. Regenerate with `npm run reactions` — see
  [Reactions](#reactions).

`/static/og-source.png`, the raw screenshot the social preview is built from, and
`/static/earth/earth-source.jpg`, the satellite mosaic the texture is cut from,
are kept in the repo but **not deployed** (`copy:static` deletes both from
`dist/static/`).

`/static/og-image-<hash>.jpg` is that preview, generated by `npm run og-image` at
the 1200×627 scrapers ask for; the script also rewrites its URL in `index.html`
and `play.html`. It is content-addressed because **scrapers cache a preview by
its URL**: under a fixed name, a link shared after a new screenshot still showed
the old one. If you regenerate it at a different size, update
`og:image:width` / `og:image:height` in both pages to match.

The source is the home page's hero, taken from the live site with a fresh
profile (a returning player is sent to the game) at a 1840×962 window — about
1.91:1, and the narrowest the whole hero fits in — at 2× for a sharp downscale.
The script letterboxes onto black rather than crop, so a source of another
shape keeps its edges. Like the home page's screenshots, it must not show the
chat. To retake it:

```bash
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new \
  --hide-scrollbars --force-device-scale-factor=2 --window-size=1840,962 \
  --user-data-dir="$(mktemp -d)" --virtual-time-budget=5000 \
  --screenshot=static/og-source.png https://clickplanet.lol/
```

Chrome may not exit after it writes the file; stop it once the file is there.

`public/` holds the files that must be served as themselves rather than as the
app: `_headers`, `robots.txt` and `sitemap.xml`. Vite copies them to the root of
`dist/`, and the Workers asset handler serves a real file before
`not_found_handling` applies — **without them, every unmatched path including
`/robots.txt` answers 200 with `index.html`**, so a crawler asking for the rules
got an HTML document. That was the leading suspect for LinkedIn refusing to
fetch the preview image, though it was never proven to be the only cause.
Nothing under `/static/` may be disallowed in robots.txt; that is where scrapers
fetch the preview from.

**`privacy.html` is the privacy policy**, linked from the home page and from the bottom of the About modal.
It is a plain page, not a component: it loads with no WebGL and no bundle, and a
crawler reads it as it is. The Workers asset handler serves it at `/privacy`
(`html_handling` drops the extension) and `nginx.conf` does the same with
`$uri.html`; `npm run dev` serves it at both `/privacy` and `/privacy.html`. **It states
retention periods, so it goes stale when the backend's do**: `ledger.retention`,
`antiBot.evidence.retention`, `chat.storage.retention` and
`auth.sessions.guestTTL` in `deploy/vps/backend.yaml`, and `roll_keep_for` in
the Caddyfile. Change one, change the page and its date.

**Its "Google user data" section is what Google's brand verification reads**:
what we ask Google for, why, that nobody else gets it, and the Limited Use
sentence. A new Google scope changes that section.

**Cloudflare Web Analytics counts the page views**, with no cookie. Each of the
four pages ends with the beacon tag, its token written once in
`src/webAnalytics.ts` and fed to the pages as `%WEB_ANALYTICS_TOKEN%`, like the
Discord invite. The token is public. **`/auth/callback` has no beacon**: its URL
holds the one-time code, and its referrer is Google or Discord, which would read
as visitors sent from there. `gameRoutes` strips the tag from the copy and fails
the build if one is left. `"spa": false`, because the pages are real pages and
the home page's section links are not views. The site is a manual one (no
`auto_install`), so nothing is injected at the edge. No page sends a
Content-Security-Policy; one that is added must allow
`static.cloudflareinsights.com` (script) and `cloudflareinsights.com` (connect).

**`index.html` is the home page** and says what the game is in plain HTML — see
[Pages and routes](#pages-and-routes). It is a full landing page (header,
hero, how it works, the board and the map, phone, Discord, and a footer that
carries the creator) in the game's look, with one stylesheet inline and no
script but the redirect. **It sells the game, not its mechanics**: country
against country on one live planet. What only makes sense once playing (signing
in to click faster, bonus boxes, the camera) stays in the game. The hero's
counts are written by hand, since plain HTML cannot read them: 262,000 tiles is
`gameMap.maxIndex` in `deploy/vps/backend.yaml`, 239 flags is
`static/countries/countries.json`. Its screenshots are `static/home/*.jpg`,
taken from the live game at 1440×900 (390×844 for the phone) at 2× with a fresh
profile; a new one must not show the chat, which carries players' own words, and
a fresh profile opens it on a desktop, so fold it first. They are fixed names
under a week of cache, so a retake gets a new name. The section links (`#how`,
`#board`, `#creator`) work in the page, but a returning player who opens one
directly is sent to the game like any other hash but `#home`.

**The Discord invite and the TikTok and Instagram profiles are written once, in
`src/links.ts`.** The menu's More tab imports them; `vite.config.ts` hands them
to the plain pages, which write `%DISCORD_INVITE%`, `%TIKTOK_PROFILE%` and
`%INSTAGRAM_PROFILE%` (Vite's own HTML replacement, fed through `define`).
`links.test.ts` fails on one of them pasted into a page.

**`terms.html` is the terms of service**, linked beside it and built
the same way, at `/terms`. Discord asks for its URL to allow OAuth sign-in. The
privacy policy says what Google and Discord send us; a new provider, or a new
field kept from one, changes that page and its date.

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

**Read [`DESIGN.md`](DESIGN.md) before touching a style.** It is the design
system: what the game looks like, which token does which job, and the rules
every piece follows. The values themselves are in `src/tokens.css`, imported
first in `main.tsx`.

**`designTokens.test.ts` fails on a color or a font named anywhere else**: in a
stylesheet, in the home page's style, or in a component (the Google and Discord
sign-in colors aside). Code that draws outside CSS reads the tokens as well:
`drawShareCard.ts` with `getComputedStyle` at draw time, the medals through
`style`. The home page, `privacy.html` and `terms.html` link `src/tokens.css`
themselves.

Plain CSS files co-located with components. No CSS preprocessor or CSS-in-JS.

`ChatPanel.css` is the one file with a custom property contract: each message
and the folded peek carry `--author-hue` from `hueOf`, and the CSS builds
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
the old Discord button ended up a 56px slab next to a 40px About button on
mobile: its CSS set its own `height`, and the mobile rule it also carried
overrode the shared mobile size. Anchors styled as buttons (`BuyMeACoffee`, the
menu's Home link) legitimately need `display: flex` with both axes centred and
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
npm run mobile -- http://localhost:5173/play --open-menu --out /tmp/shot.png \
  --eval 'JSON.stringify(document.querySelector(".sheet").getBoundingClientRect())'
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

Nor does it draw the browser's toolbars, so **`100vh` looks right in the
emulator and is not on a phone**: there it is the height with the toolbars
hidden, and this page never scrolls them away. A `Modal` sized `100vh` put the
bottom of the player card under the toolbar, with nothing left to scroll and a
drag that pulled the page to refresh. Size a full-screen layer with
`position: fixed; inset: 0` instead. `overscroll-behavior: none` on the root
stops pull-to-refresh everywhere.

`--open-menu` dismisses the donation modal and opens the Board sheet:
`DonationModal` rolls a coin on **every** load (`SHOW_PROBABILITY`), so without
it you will screenshot the donation modal half the time.
