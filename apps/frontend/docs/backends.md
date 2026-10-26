# `src/backends/` — three contracts, one transport

The tile game's three interfaces are in `backend.ts`:

- `TileClicker` — claims a tile
- `OwnershipsGetter` — batched fetch of tile → country_code, takes an
  `AbortSignal`
- `UpdatesListener` — live tile changes

The chat's three are in `chat.ts` — see [Live chat](chat.md). The two files
share no types: they are separate bounded contexts on the backend and the split
is worth keeping on this side too.

Beside them: `session.ts` (the click token, see [Sessions](sessions.md)),
`account.ts` (who the cookie belongs to, see [Sign-in](sign-in.md)) and
`player.ts` — `PlayerBackend`, the player's `Profile`, `PlayerError` and
`isValidUsername`, the username rule, plus `PresenceBackend` (see [Who is
playing](who-is-playing.md)). `playerBackend.ts` implements both against
`player.v1.PlayerService`. `support.ts` is what a player sends the operator —
see [Support](support.md).

`transport.ts` holds what both contexts need and neither owns: `retrying`,
`retryingAtOnce`, `NO_TIMEOUT`, and `openStream`, which follows a
server-streaming RPC and reopens it with a capped exponential backoff. It is generic over the message type and
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

**A reopened stream has a gap, and nothing replays it.** A phone that locks, or
a tab the browser freezes, loses its connection, and whatever the server sent in
the meantime is gone. So `openStream` calls `onResumed` on the first message of
a connection when an earlier one had been live; the planet and the chat read
the map and the history again there (see below). The roster and the board need
nothing: each connection starts with the whole of it. Two more things find a
gap sooner:

- **A silent stream is reopened.** A socket killed while the page slept can look
  open forever. Nothing heard for `SILENCE_LIMIT_MS` (60s, two heartbeats) aborts
  it and connects again. A server heartbeat slower than 30s would trip this.
- **The page coming back is a wake** (`pageWakeups`: `visibilitychange` to
  visible, `resume`, `pageshow`, `online`). It connects at once rather than
  waiting out the backoff, and reopens a stream already silent too long. Tests
  pass their own `wakeups`.

**`NO_TIMEOUT` is load-bearing, not decoration.** `main.tsx` builds the clients
with `timeoutMs: 10_000`, and connect-web applies `defaultTimeoutMs` to a stream
exactly as to a unary call — so without passing `timeoutMs: NO_TIMEOUT` on the
call, every live feed would die ten seconds in and reconnect forever. Anything
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

Connect does not retry, so `retrying` wraps every read: eight attempts while the
server cannot be reached or does not answer in time, and never a retry of an
answer the server chose to send. **A failed fetch is `unknown`, not
`unavailable`**: connect-web wraps the fetch's `TypeError` as the cause, and
that is how "cannot be reached" is told from an `unknown` the server sent.
The rule is `outcomeOf`'s miss, in `connection.ts`: the one the connection
card counts by (see [Connection health](#connection-health)).
Until this was known, no dropped connection was ever retried. **It is built for a phone on a train**: the
second attempt goes at once (a dead pooled connection), then it waits 1s, 2s,
4s and 8s, about 30s in all, so a tunnel does not fail the map. The timeout was
2s once, and was not retried: one slow `GetMap` batch of the 26 failed the
whole globe. The click goes through `retryingAtOnce` instead: five attempts
with no wait, and **a click that timed out is not sent again**, since it may
have landed and costs a click and maybe a charge. It is rolled back, and the
echo repaints it if it did land.

The backend refuses a click in four ways, and `clickTile` translates all of
them into errors declared beside the interfaces — the first three in `backend.ts`,
the last in `session.ts` — so `globe.ts` recognises them without knowing what a
Connect code is:

- `resource_exhausted` → `RateLimitedError`. The per-IP token bucket is spent.
- `permission_denied` with a `BanRefusal` detail → `BannedError`, with `until`,
  when the ban ends. The account or its network is banned (see the backend's
  [A ban is told](../../backend/docs/antibot.md#a-ban-is-told)).
- `permission_denied` with no detail → `VPNBlockedError`. The address is in the
  backend's VPN and proxy blocklist.
- `unauthenticated` → `SessionUnavailableError`, **and only after a retry** —
  see [Sessions](sessions.md).

They are separate classes rather than one with a field because each one says
something different: ease off for a second (the click meter shakes — no dialog),
wait for the ban to end, turn the VPN off, or reload and unblock the challenge.
Everything else is a transport fault and still reaches the console.

**A bomb and a shield are refused for a ban the same way** (`asBonusError` reads
the detail), and the bomb or the shield stays in hand, since the server spent
nothing.

`FakeBackend` reproduces all four, so every refusal is reachable in dev: it
enforces the same bucket as production's `rateLimiter` (one click every 5s, 60 in hand), and takes `vpnBlocked` and
`sessionUnavailable` options that refuse every click (there is no address and no
widget there to judge). Its own simulated traffic bypasses all of them, standing
in for other players rather than for this one.

## The click budget

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

`PlanetBackend` learns it four ways — `GetBudget` at load and on a switch of
country, `ClickResponse.budget` on every accepted click, a **connect error
detail** on a refused one, which is the reading that matters most, and
`GetBudget` again when the planet stream says the rules changed
(`rulesChanged`): the server can change the whole game's refill at once (the
backend's [Rules](../../backend/docs/rules.md)), and only a reading says by how much.
The same event reads `GetBonusRules` again (see [Charges](charges.md)). **`GetBudget`
carries the token already held** (`SessionProvider.held()`, never a mint):
without it the server reads the bucket of an address with no account, which is
never spent and always full, and the next click contradicts it. It also subtracts its own clicks in
flight, so the counter only ever *under*-promises: a counter that says 1 and is
refused is a bug the player sees.

**The globe sends no click the counter cannot pay** (`canClick`). It paints
nothing, plays the refusal and shakes the meter, as a 429 would. A painted tile
that flips back is exactly an enemy retake to the eye: on 2026-10-09 a player on
a train read 25 refusals as a Polish bot retaking Antarctica in
seconds. What it costs is the click a counter at 0 would still have won, when a
click in flight fails. With no reading, nothing is held.

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

**A resumed stream makes the globe catch up.** `listenForResumes` is called
after `PlanetBackend` has flushed the updates from before the gap. The globe then
forgets which tiles the stream had set (`forgetLive`), waits
`CATCH_UP_DELAY_MS`, reads the whole map again and applies it with `resync`.
**The wait is `GetMap`'s `max-age`** (5s): a copy from the edge is never older
than that, so after the wait it is never older than the resumed stream, and
what the stream said since still wins. It costs a whole map, about 0.5 MB, per
gap.

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

## Connection health

**Every call is the probe, so there is no ping.** `connectionInterceptor`
(`connection.ts`) sits on every Connect transport (`Config.interceptors`, set in
`main.tsx`) and tells one `ConnectionHealth` how each call went. A health RPC
would only say whether the health RPC gets through.

- **Any answer is the server reached**, a refusal it sent included. A failed
  fetch (connect-web's `Unknown` over a `TypeError`), a timeout
  (`DeadlineExceeded`, the 2s `timeoutMs`) and a proxy with nothing behind it
  (`Unavailable`, a 502 to 504) are misses. A call the page cancelled says nothing.
- **A stream is followed past its opening**: each message is the server reached,
  and a break is a miss.
- `DOWN_AFTER_MISSES` (2) in a row is down, and the first answer is up again. The
  browser's `offline` is down at once (`followBrowser`); `online` waits for an
  answer, which `openStream`'s wake brings at once.
- **`Viewer` shows `ConnectionLost` in the Moments zone while it is down.** Clicks
  still go out and roll back: they are how it finds the way back. A deploy's
  restart reads as down too, which is what it is from here.
- The fakes have no transport. In fake mode `connection` is on `window`:
  `connection.wentOffline()` raises the card and `connection.reached()` drops it.
