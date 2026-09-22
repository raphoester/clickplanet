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

In fake mode the console has a few commands: `giveBomb()` puts a bomb in the
inventory as if a box holding one had just been caught, `giveBonus("refill")` does
the same for any other bonus (the fake holds charges as the server does: a refill
and a bomb at most, a pool of 8 spread clicks and a stack of 3 enclosures, a box
adding 1 to 4 and 1 to 3 of them, spread and enclose spent only while switched on,
both at once refused, a refill refused on a full bank), `giveQuiz()` puts a quiz
banner up at once, and `fakeBackend.botBomb(tile, "fr")` and `fakeBackend.botSpread(tile, "fr")`
play somebody else's bomb or spread click.

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
| `/` | `index.html` | The home page. Plain HTML, no bundle. |
| `/play` | `play.html` | The game. |
| `/auth/callback` | `auth/callback.html` | The game again, for the sign-in callback. |
| `/privacy`, `/terms` | `public/*.html` | Plain pages. |
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
an old link such as `/?c=de` still reaches the game with its query. Crawlers
have no storage and read the home page. `homePage.test.ts` pins the key in the
script to the constant. **`/#home` does not redirect**: it is the "Home page"
link at the bottom of the About modal.

**The game is a real file at each path**, not a fallback. The Workers fallback
is `index.html`, the home page, so a `/auth/callback` that relied on it would
land on the home page and lose the code. The `gameRoutes` plugin in
`vite.config.ts` writes `play.html` a second time as `auth/callback.html`, and
in `npm run dev` rewrites both paths to `play.html`. The registered redirect URI
is unchanged. Paths under `/play/` have no file and get the home page: the game
has no routes of its own.

**The bundle is named `play-*.js`** now, not `index-*.js`, and it is linked from
`/play`, not from `/`.

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
  messages, for highlighting them once they are on screen. `nameSentUnder` is
  the name the server gave the latest message this client sent — the only way
  it learns a guest's name.
- `authorColor.ts` — `authorHue`, a stable hue per chat author. It hashes the
  name the log displays, which the server gives one account only (a username,
  or `guest_` and the guest's code). **Only the hue is derived**: the
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

### Live chat

The client for the backend's second bounded context: `chat.ts` declares
`ChatSender`, `ChatHistoryGetter`, `ChatListener` and `ChatReactor` (plus
`ChatBackend`, the four together), `chatBackend.ts` implements them against
`/chat.v1.ChatService/` alone — `SendMessage`, `GetHistory`, `React` and the
`ListenForEvents` stream — and
`fakeChatBackend.ts` is the dev stand-in. `ChatPanel` docks
bottom-right, opposite the menu, and starts folded under 768px.

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
  both scrolls and clips, and the panel's `backdrop-filter` would hold a
  `position: fixed` child to the panel instead of the screen. Anything that
  moves the chip — a scroll, a resize — closes it rather than making it follow.
- **`mine` is only known from a call.** `GetHistory` sends the token already
  held (`SessionProvider.held()`, never a mint) so the server can mark the
  player's own; `React` answers the counts with `mine` set. The stream is
  nobody's, so `mergedReactions` keeps what the log already knew. A player
  whose token is not held yet when the history loads sees its own reactions
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
  when the last message changes, and the unread count and the sound only count
  messages.

#### Announcements

The chat also shows lines nobody sent: `ChatEvent.announcement` on the stream,
and `GetHistoryResponse.announcements` beside the messages. Today the one kind
is `bomb`, every bomb that went off.

- **Decoded, not trusted**: `decodedAnnouncement` reads the `kind` and parses
  the JSON `payload` into a typed `ChatAnnouncement`. A kind this build does not
  know, or a payload that is not the kind's, is dropped, so the server can ship
  a new kind first.
- **The payload is values, the client writes the sentence**: a bomb line is
  `describeBlast`, the same words as `BombNews`, so the chat and the news line
  never disagree.
- **Kept apart from the messages** (`useChat`'s `announcements`,
  `addAnnouncements`) and put in one list only to draw (`interleave`, by time).
  So a burst of bombs never pushes a message out of the log, and the unread
  count, the sound and the "New messages" pill count messages alone.
- **Not a balloon**: `ChatLog` draws a centred line (`.chat-announcement`) with
  the bomber's flag and the time. It ends the run above it, so the next message
  says again who is talking.
- In fake mode `main.tsx` hands every `FakeBackend` bomb to
  `FakeChatBackend.announceBomb`, with no ground: the fake has no borders.

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

### Who is playing

The "players online" button in the menu's action row opens a `MenuPanel` listing
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

**An admin of the game wears a crown** (`AdminCrown`, gold, `role="img"` named
"Admin") beside its name in the chat log, the roster and the card's title.
The server says so: `ChatMessage.authorAdmin`, `RosterEntry.admin` and
`PlayerInfo.admin`. The card crowns from what was clicked, and from the read
once it lands. In fake mode, Ana is the admin.

**A name opens a player card**, in the roster and on a chat message
(`app/players/PlayerCard.tsx`, a `Modal`). `Viewer` holds the one card open and
hands `onOpenPlayer` to `Menu` → `PlayersPanel` and to `ChatPanel` → `ChatLog`;
without a `PlayerInfoBackend` wired the names are plain text. The card shows the
flag and the country, then, for a player with a username, what
`player.v1.PlayerService/GetPlayer` answers: tiles taken, the current and best
streak, and "Playing since", the day the account was made (left out when the
server does not know it). **A guest's card asks nothing**: a guest has no
username, so there is nothing to look up, and the card says so. The chat tells
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
no token and no header. Every connection starts with the whole roster, then
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
clients stay without it: nothing there needs to know who is asking, and a read
that carries a cookie is one no shared cache serves. Both halves are pinned in
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

**It keeps the click token and never the account.** The account is the `cp_sid`
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
on the guest account the mint gives every browser. Signing in with Google or
Discord keeps that account on every device.

- `backends/account.ts` — the contract: `AccountBackend`, `Provider`, and
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
- `app/account/` — the rest is React: `AccountRow` (one line in the menu),
  `AccountPanel` (a `MenuPanel`, like the sound settings), `DeleteAccountModal`,
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

**A signed-in player picks a unique username** in `AccountPanel`: 3 to 15
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
  return.
- `bonusClickEffects.ts` — the same, for every click made with spread on
  (`tilesSpread`: a green burst, a spark popping onto each tile around it in
  turn, two rings). It reuses the enclosure's shaders, with normal rather than
  additive rings, which vanished on the white of a flag. A busy planet spreads a
  lot, so at most `MAX_PLAYING` run at once.
- `shaders/` — GLSL for the display, picking, star and enclosure passes.

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
  answers a boolean: `blasts`, `bonusBox`, `enclosureEffect`, `bonusClickEffects`
  and `TileField.setHover`. **The frame an effect *ends* on counts**: it is the
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
   rather than up to a sample later.
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

   **A slow refill gets a countdown.** When a click takes 1.5s or more to come
   back (`COUNTDOWN_FROM_S`; production is one every 5s), the meter says
   "+1 in 4s" beside the bar, and says nothing at a full bucket. Past 12 pips
   the bar moves a sixtieth per click, too little to see, so a blue strip under
   it (`--click-budget-next`) fills once per click, as a pip would.

   The refill is animated from **one CSS custom property written per frame**,
   and each pip works out its own share of it with a `clamp()`; the count, the
   colour and the aria value are written only when the whole number changes.
   `useClickBudget` therefore re-renders when the *server* says something, not
   as the bucket refills — this sits beside a WebGL scene that wants the main
   thread. Under `prefers-reduced-motion` the fill steps four times a second
   instead of gliding.

   On a phone it moves to under the folded menu: both ends of the screen are
   full-width sheets there, the menu above and the chat below.

   **A guest is offered to click faster.** A signed-in account refills
   `ClickBudget.linkedMultiplier` times faster (2 in production), into a bank of
   the same size. For a guest the
   server offers sign-in to, `Viewer` passes `onSignIn` and the meter shows
   "Sign in: clicks 2× faster" under the pips — a button beside the meter, not in
   it, since the meter is a reading. It glows when the bucket is empty or a click
   is refused, the moment a guest meets the wall. It opens `SignInPitchModal`,
   which has the account panel's sign-in buttons. The account panel's guest text
   says the same. **Nothing is offered without the server's number**, nor with
   sign-in off.

   **It is one panel, and its width is set rather than grown.** The reading, the
   offer and the inventory all live in `.click-budget-dock`, which takes the
   corner and carries the only border, background and blur; `.click-budget`
   itself draws nothing, so `role="meter"` stays a leaf with no button inside
   it. The dock's width is a number (288px, 240px on a phone) because the three
   parts are three different widths: shrink-to-fit handed the widest one the
   say, and the others then trailed a stripe of dead space to their right —
   worse, the price row wrapping made the meter's max-content that whole row
   *unwrapped*, which is where the empty half of the pill came from. The gauge
   is `flex: 1` and the slots `flex: 1 1 0`, so both fill whatever the panel
   gives them. The panel's border is what carries state: the inventory's glow
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

Rolling back on *any* failure, including a transport fault, is deliberate: if
the click did land and only the response was lost, the stream's echo repaints
it, and if the echo arrives first the rollback is already a no-op.

## Charges

A bonus box holds a **charge**, which has no clock and is never used on its own:
a refill (the click bank, filled when the player presses it), a bomb (one drop), a
stack of enclosures (one shape each, `maxTiles` at most) and a pool of spread
clicks. The server keeps them per account, in postgres: a refill and a bomb at
most, up to 3 enclosures and up to 8 spread clicks. A box adds a random 1 to 3
enclosures or 1 to 4 spread clicks, capped at the size; the reward it announces
is what was kept (`+2 spread clicks`), which is what the server answers in
`ClaimBonusResponse.amount`.

**`PlanetBackend` holds `Charges` (`domain/bonus.ts`) and nothing pushes them.**
It reads `GetCharges` at load, with the token in hand and never a fresh one, and
again when a click goes out under a new token (`followSession`): the charges are
the account's. `ClaimBonusResponse.charges` replaces them on a claim, and
`UseRefillResponse.charges` on a refill, which also brings the full budget. Otherwise
it follows its own calls: an accepted click **sent with spread on** takes a spread
click off, this player's own `tilesEnclosed` takes an enclosure off, and a drop
takes the bomb off at once and gives it back only if the call never reached the
server. The click answer says nothing about charges, on purpose (see the backend's
CLAUDE.md). A charge spent in another tab stays on screen until the next read. It
reaches the globe through `BonusHandlers.onCharges`, and `useGlobe` hands it to the
inventory.

**The sizes are rules, read once**: `GetBonusRules` (a cached GET) answers the
blast radius, the enclose's `maxTiles`, the spread pool's size and the enclosure
stack's size as `BonusRules`, through `onRules`. A page open across a change of
rules shows the old sizes until it is reloaded.

### Off by default, one at a time

**Nothing is used until the player says so.** Spread and enclose are switches
(`Switches`), off at load and never turned on by the client. Every click carries
them: `TileClicker.clickTile(tile, country, switches)` sends
`ClickRequest.spread` and `enclose`, and the server spends a charge only when its
switch is on. A switch goes off by itself when its pool runs out
(`switchesHeld`), so it never says a click does something it will not.

**One bonus at a time**, the bomb included: `switched` turns the other switch off
when one goes on, aiming the bomb switches both off, and switching one on puts the
bomb away. `globe.ts` holds the one copy (`setSwitch`, `onSwitchesChange`). The
server refuses a click with both switches on (`INVALID_ARGUMENT`), before it
writes or spends anything.

### The inventory

`components/Inventory.tsx` is the section that shows them, the **lower half of
the click meter's panel** (the meter takes it as `children`, and shows it even
with no budget). It draws no border, background or blur of its own: the one
panel is `.click-budget-dock`, and a hairline divides the reading from the slots.
One slot per kind, always shown, each drawn with its box's
icon (`BonusIcon`) in its box's colours (the `--bonus-*` properties in
`BonusAward.css`, shared with the announcement). An empty slot is dimmed and
cannot be pressed. A pool shows its count against its size (`5/8`, `2/3`), and a
word over the icon says what the slot is doing (`On`, `Aim`, `Full`).

- **Refill** fills the bank (`Refiller.useRefill`). **On a full bank it sends
  nothing** and says "Full" for two seconds: a refill there would be wasted. The
  server refuses it too, `FailedPrecondition`, read as `BankFullError`, and spends
  nothing.
- **Bomb** aims it, or puts it away (`Globe.setArmed`).
- **Spread** and **Enclose** switch (`Globe.setSwitch`), `aria-pressed`.

**The whole section folds** from a header over the slots: the name on the left,
the count of kinds held while folded (open, every slot already says its own), and
a `ChevronIcon` at the right end that turns over when it opens. That is the
page's one way of folding something — the same icon and the same turn as
`MenuHeader`'s collapse and the chat's header — so it is written the same way
here rather than invented again. The fold is kept in local storage
(`clickplanet-inventory-folded`, read and written in a `try`, since a private
window can throw). The panel glows while something is on or aimed, so a folded
inventory still says the next click does more than paint.

## Quizzes

A banner at the top of the screen: press it and you get a question with three
choices and **five seconds**. A right answer is worth a charge, the same as a
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
with a token and a subject country and *no question*; `openQuiz` brings the
question and its three choices and **starts the server's clock**; `answerQuiz` is
the first thing that says which of the three was right.

**One state machine, `QuizState`:**

```
idle → offered → opening → asking → answered → idle
```

Each phase is a different thing on screen *and* a different thing to a player:
`offered` is an invitation that costs nothing to ignore, `asking` is five seconds
already running. Anything that goes wrong — a token the server will not honour, a
stream that dropped — falls back to `idle`, because a quiz nobody can answer
should leave nothing behind. **The token rides through the state** rather than
sitting beside it, so there is no way to answer one quiz with another's token
while a banner and a question are changing places.

**A banner arrives only into an empty screen.** The server will not offer a
second, but a stale one arriving mid-question would take the clock away from
under somebody already reaching for a choice.

**The banner says what the question is about and nothing else** — a flag and a
country name. That is what makes it worth looking at, and it gives nothing away:
knowing a question is about Estonia is not knowing the capital of Estonia. It
draws no countdown of its own, because it is free to ignore, and it goes away by
itself.

**The countdown bar starts at what is actually left, not at full.** The five
seconds are the server's and they began when it answered, so a slow round trip
has already spent some of them; a bar that started full would promise time the
player does not have. From there it is **one CSS transition on `transform`** to
empty — on the compositor, so five seconds of continuous animation costs nothing
beside a WebGL globe drawing at the same time, where a `width` transition would
relayout every frame. Under `prefers-reduced-motion` the countdown **stays**: it
is information, not decoration.

**Running out of time is sent as a choice past the end of the three.** The server
reads it as wrong, which it is, and answers with the right one — so a question
nobody managed to answer still says what it was. There is no other way to learn
it.

**The result says which one was right whether or not that was the one pressed.**
A wrong answer costs nothing, so the only thing left to give back is the answer.

**The quiz and `BombNews` both want the band at the top**, and the bomb line is
the one that gives it up (`lowered`): four seconds of news nobody presses moves,
five seconds somebody is answering does not.

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
the country leading the map, on a loop, with a player at the bottom of the screen.
The recordings are the US Navy Band's (public domain); `npm run anthems` downloads
them, normalises loudness, trims the silence so the loop has no gap, and writes
`static/anthems/<code>-<hash>.m4a` plus `anthemsAsset.ts`, which pairs each file
with the anthem's title (`TITLES` in the script, written by hand: Wikidata's
labels mix titles with "National Anthem of X"). It needs ffmpeg.
Territories share their country's file (`SHARES` in the script); a country with
no recording shows the player with its play button off.

- `domain/anthemLeader.ts` — `followLeader`: a new leader must hold first place
  for `HOLD_MS` (15s) before the music follows it. The first leader plays at once.
- `anthemPlayer.ts` — two `<audio>` elements crossfade through Web Audio gain
  nodes. **Not `audio.volume`: iOS ignores it.** A muted or hidden tab fades out
  and pauses rather than streaming silence.
- `useAnthem.ts` — ties the board and the settings to the player, which is built
  once, so a tick or a toggle never restarts the music.
- `AnthemBar.tsx` — the player: the anthem's title, then the country. Play and volume write `SoundSettings.anthem`, so
  it and the settings panel never disagree. Pressing play lifts the master switch.

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

`/static/og-image.jpg` is that preview, generated by `npm run og-image` at the
1200×627 scrapers ask for. It is letterboxed onto black rather than cropped —
the screenshot is wider than 1.91:1 with the leaderboard against one edge and
the buttons against the other. If you regenerate it at a different size, update
`og:image:width` / `og:image:height` in `index.html` and `play.html` to match.

`public/` holds the files that must be served as themselves rather than as the
app: `_headers`, `robots.txt`, `sitemap.xml`, `privacy.html` and `terms.html`. Vite copies them to the root of
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
`$uri.html`; `npm run dev` only serves it at `/privacy.html`. **It states
retention periods, so it goes stale when the backend's do**: `ledger.retention`,
`antiBot.evidence.retention`, `chat.storage.retention` and
`auth.sessions.guestTTL` in `deploy/vps/backend.yaml`, and `roll_keep_for` in
the Caddyfile. Change one, change the page and its date.

**Its "Google user data" section is what Google's brand verification reads**:
what we ask Google for, why, that nobody else gets it, and the Limited Use
sentence. A new Google scope changes that section.

**`index.html` is the home page** and says what the game is in plain HTML — see
[Pages and routes](#pages-and-routes). It is a full landing page (header,
hero, how it works, features, creator, Discord, footer) in the game's look, with
one stylesheet inline and no script but the redirect. The hero's `.boost` callout
says signing in clicks 2× faster: plain HTML cannot read the server's number, so
**change it with `rateLimiter.linkedMultiplier` in `deploy/vps/backend.yaml`**. Its screenshots are
`static/home/*.jpg`, taken from the live game; a new one must not show the chat,
which carries players' own words. The section links (`#how`, `#features`,
`#creator`) work in the page, but a returning player who opens one directly is
sent to the game like any other hash but `#home`.

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
