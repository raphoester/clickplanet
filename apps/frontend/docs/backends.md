# `src/backends/` — three contracts, one transport

The tile game's three interfaces are in `backend.ts`:

- `TileClicker` — claims a tile
- `OwnershipsGetter` — the map read range by range (`getOwnerships`), the
  server's counts (`getHoldings`) and the fortresses, each with the version it
  was read at; takes an `AbortSignal`
- `UpdatesListener` — live tile changes, the counts that changed, and each
  opening of the stream, each with its version

The chat's three are in `chat.ts` — see [Live chat](chat.md). The two files
share no types: they are separate bounded contexts on the backend and the split
is worth keeping on this side too.

Beside them: `session.ts` (the click token, see [Sessions](sessions.md)),
`account.ts` (who the cookie belongs to, see [Sign-in](sign-in.md)) and
`player.ts` — `PlayerBackend`, the player's `Profile`, `PlayerError` and
`isValidUsername`, the username rule, plus `PresenceBackend` (see [Who is
playing](who-is-playing.md)) and `PlayerReader`, who a player is on its card.
`playerBackend.ts` implements them against `player.v1.PlayerService`.
`progression.ts` is what a player earned: `ProgressionBackend` (its own titles,
the title it wears and its fronts), `LevelReader` (its level and what each gated kind opens at, one `GetLevel` for the
whole page), `ProgressReader` (a player's progress on its
card) and `ProgressionListener` (the titles it earns and the levels it reaches, live, on one stream).
`progressionBackend.ts` implements them, and the badges, against `progression.v1.ProgressionService`
(see [Titles](titles.md), [The level](level.md) and [Badges](badges.md)). A level, on the card's `Progress.level`,
`GetLevel` and the stream's `level_reached`, is the level the server keeps on its bar (`PlayerLevel`), or none when it
keeps none (see the backend's [Levels](../../backend/docs/levels.md#the-public-reads)). `playerCards.ts` reads the
two halves of a public card side by side (see [Who is playing](who-is-playing.md)).
`support.ts` is what a player sends the operator —
see [Support](support.md). `clips.ts` is the clips that name the player — see [In a
clip](clip-appearances.md). `alerts.ts` is whether the player is in the game's Discord server, which a Discord sign-in joins — see [Discord pings](discord-alerts.md).
`flags.ts` is the list of flags: `FlagLister` and `FlagStream`. `flagsBackend.ts` implements both against
`flags.v1.FlagService`, and `FakeFlagsBackend` both in fake mode (see [The flags stream](#the-flags-stream)). `flags.ts`
also has `FlagSubmitter`, the flags a player sends for review, with `SubmissionRefusedError`: `ConnectFlagSubmissions`
(`flagSubmissionsBackend.ts`) over the same service, the sends with the click token and one retry on a refused token
as support's, the read with the token that names the account; `FakeFlagSubmissions` in fake mode (see [Flag
submissions](flag-submissions.md)).
`shop.ts` is time for a regional flag, paid on Stripe's page: `Shop` and `ShopRefusedError`. `ConnectShop`
(`shopBackend.ts`) implements it against `shop.v1.ShopService` with the token that names the account, `ListPlans` with
none; `FakeShop` in fake mode (see [The shop](shop.md)).
`antibot.ts` is the warning before a ban for playing several accounts: `WarningsListener` (the antibot's stream, opened
with the token held as progression's is, and stopped for good on a refused token) and `WarningAcknowledger`; the same
stream's notice that another account played from the connection, and `NoticeAcknowledger`.
`antibotBackend.ts` implements them against `antibot.v1.AntibotService`, and `FakeAntibotBackend` in fake mode (see
[Data flow](data-flow.md)). `bonusesBackend.ts` is the `bonuses.v1.BonusService` client, which `PlanetBackend` holds to read whether a first
sign-in earns a refill (`GetGrantRules`). `boxesBackend.ts` is the boxes': `BoxesBackend` implements `BoxCatcher`
against `boxes.v1.BoxService` (see [The boxes stream](#the-boxes-stream)). `quizzesBackend.ts` is the quizzes':
`QuizzesBackend` implements `QuizMaster` against `quizzes.v1.QuizService` (see [The quizzes stream](#the-quizzes-stream)).

`transport.ts` holds what both contexts need and neither owns: `retrying`,
`retryingAtOnce`, `NO_TIMEOUT`, and `openStream`, which follows a
server-streaming RPC and reopens it with a capped exponential backoff. It is generic over the message type and
knows nothing about what it carries, so each context keeps its own mapping —
`PlanetBackend.listenForUpdates` and `ChatServiceBackend.listenForMessages` are
both a few lines over it.

**One stream per API, carrying an envelope.** `ClickService.ListenForEvents`
sends `PlanetEvent`, `BoxService.ListenForEvents` sends `BoxEvent` and `ChatService.ListenForEvents` sends
`ChatEvent`, each a `oneof`. `updateOf` and `messageOf` unwrap the case each context cares about and
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

**A message the handler throws on is dropped, and the stream stays.** Each reconnect
sends the same message again, so a reconnect only loops on it. On 2026-10-31 a new
`EntryKind` made `kindOf` throw in tabs opened before the deploy: each one reconnected
every 500 ms, since the message had reset the backoff, until the server swapped. The
error goes to the console.

**An attempt that ends is aborted.** connect-es drops `return()` from a stream's
iterator, so leaving the `for await` on an error leaves the fetch open, and the
server's heartbeats keep its end alive. `openStream` aborts each attempt's signal when
its loop ends, whatever ended it.

**A reopened stream has a gap, and nothing replays it.** A phone that locks, or
a tab the browser freezes, loses its connection, and whatever the server sent in
the meantime is gone. So `openStream` calls `onResumed` on the first message of
a connection when an earlier one had been live, and the chat reads its history
again there. The planet's first message on each connection is a heartbeat with
the map's version, which `listenForOpenings` hands the globe (see below), and on a resumed one the planet reads
`GetBudget` and `GetChargeRules` again, since a `rulesChanged` may have fallen in the gap. The roster and the board need
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

`planetBackend.ts` talks to the API through the generated Connect clients
(`src/gen/grpc/planet/v1/planet_connect.ts`, and `bonuses/v1/bonuses_connect.ts` for the sign-in refill's rule). No gRPC is involved — Connect is
an ordinary HTTP POST, or a GET for reads.

**Two transport options are load-bearing and both default to `false`:**

- `useBinaryFormat: true` — without it `GetMapResponse.tiles` travels as base64
  inside JSON, a third bigger.
- `useHttpGet: true` — without it the side-effect-free reads go out as POSTs,
  which no cache will serve.

**Every `GetMap` answer and the stream's opening name the board the map plays.** `readBoard` reads it at the boot from
the first tile alone, and `BoardChanges` (`boards.ts`), which `PlanetBackend` and `FakeBackend` are given, decides what
a batch or an opening on another board than the one drawn does: a batch is read again past the browser's copy, an
opening reloads the page. See [The board the page draws](board-reload.md).

The map arrives as `GetMapResponse`: `start_tile_id`, a `codes` table, and
`tiles`, two bytes per tile indexing into it. `bindingsOf` reads it. Tile ids
are implicit in the position, so it is far smaller than a keyed map — 516 KB
against 3.6 MB for a full map — and protobuf does all the framing, so there is
no hand-rolled encoding to keep in step with the backend.

**The map is read eight batches at a time.** `getOwnerships` keeps
`MAP_READS_IN_FLIGHT` (8) `GetMap` reads open over the ranges it is given
(`tileRanges`), and hands each batch over as it lands, in any order, with its
range and version, since a batch only sets its own tiles. One after the other,
the 27 batches were 27 round trips, plus a CORS preflight each, before the globe
was playable: on DevTools' Fast 4G, 4.7s of a 9.6s load. Eight at a time it is
0.7s, and four at a time 1.2s. A batch that fails after its retries stops the
others, and nothing that lands after that, or after the caller's abort, is handed
over. **The server's end is inclusive**, so a batch ends at `start + batchSize - 1`,
and no read starts at 0 or ends past the board's last tile: the backend's antibot takes a
read off the map for a bot's walk (its `scraper`, in
[antibot.md](../../backend/docs/antibot.md)).

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

The backend refuses a click in five ways, and `clickTile` translates all of
them into errors declared beside the interfaces — `SessionUnavailableError` in
`session.ts`, the others in `backend.ts` — so `globe.ts` recognises them without knowing what a
Connect code is:

- `resource_exhausted` → `RateLimitedError`. The per-IP token bucket is spent.
- `permission_denied` with a `BanRefusal` detail → `BannedError`, with `until`,
  when the ban ends. The account or its network is banned (see the backend's
  [A ban is told](../../backend/docs/antibot.md#a-ban-is-told)).
- `permission_denied` with no detail → `VPNBlockedError`. The address is in the
  backend's VPN and proxy blocklist.
- `unauthenticated` → `SessionUnavailableError`, **and only after a retry** —
  see [Sessions](sessions.md).
- `failed_precondition` with a `FrozenRefusal` detail → `FrozenError`. The rules in
  force freeze the map (see [A frozen map](#a-frozen-map)); the click meter shakes,
  as for the throttle.
- `failed_precondition` with a `ReachRefusal` detail → `OutOfReachError`. While
  the reach plays, the tile is next to none of the flag's land and across no bridge from it
  (see [The reach](reach.md)); the meter shakes and says to take a tile next to it.

They are separate classes rather than one with a field because each one says
something different: ease off for a second (the click meter shakes — no dialog),
wait for the ban to end, turn the VPN off, or reload and unblock the challenge.
Everything else is a transport fault and still reaches the console.

**A bomb and a shield are refused for a ban the same way** (`asBonusError` reads
the detail), and the bomb or the shield stays in hand, since the server spent
nothing.

`FakeBackend` reproduces the throttle, the ban and the frozen map in dev: it
enforces the same bucket as production's `rateLimiter` (one click every 5s, 60 in hand). Its own simulated traffic
bypasses all of them, standing in for other players rather than for this one.

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
(`rulesChanged`) or comes back after a gap: the server can change the whole game's refill at once (the
backend's [Rules](../../backend/docs/rules.md)), and only a reading says by how much.
The same event reads `GetChargeRules` again (see [Charges](charges.md)). **`GetBudget`
carries the token already held** (`SessionProvider.identity()`, never a mint):
without it the server reads the bucket of an address with no account, which is
never spent and always full, and the next click contradicts it. It also subtracts its own clicks in
flight, so the counter only ever *under*-promises: a counter that says 1 and is
refused is a bug the player sees.

## A frozen map

**Planet says when the map takes no play, since planet is what refuses it**:
`ClickBudget.frozen`, set by `GetBudget` alone (see the backend's
[A frozen map](../../backend/docs/rules.md#a-frozen-map)). A set taking over sends
`rulesChanged`, so an open tab reads the freeze, and its end, within a second.

- **The mood is decided in one place**, `domain/mood.ts`'s `moodOf(turn, frozen)`:
  frozen wins over a battle, since planet takes no play whatever the schedule's turn
  says. `useMood` writes it on the root (`data-mood="frozen"`), and the battle look
  goes with it.
- **`globe.ts` sends no click on a frozen map**: a click on a tile shakes the meter
  (`onFrozen`) and plays the refusal sound; a click from orbit still flies down. A
  `FrozenError` from the server, in the second before the reading lands, does the same.
- **The click dock says so** (`ClickBudgetMeter`'s `frozen`): "The map is frozen" in
  place of the clicks and the inventory, and "Season N starts in …" when the
  gamemaster's timeline has a next season.
- **A frozen map is when players have time to bring friends**, so the dock holds the
  recruit link of the flag played ("Invite", `RecruitButton`, see
  [Sharing](sharing.md)) and the Discord, when the game offers it (see
  [Discord pings](discord-alerts.md)). On a phone the Discord is its icon alone.
- **A map reset needs nothing here**: planet cuts every stream and starts a new epoch
  (see the backend's [The map reset](../../backend/docs/map-reset.md)), so the stream
  opens again, the tiles see another start, and the tiles, the holdings and the
  fortresses are read again, as after a restart.
- **In fake mode** `freezeMap()` and `thawMap()` on the console freeze and thaw the
  fake planet, and `startBreak(nextInMinutes = 960)` ends the fake season with the
  next one to come, so the countdown shows, and adds it to the ended seasons.

**The globe sends no click the counter cannot pay** (`canClick`). It paints
nothing, plays the refusal and shakes the meter, as a 429 would. A painted tile
that flips back is exactly an enemy retake to the eye: on 2026-10-09 a player on
a train read 25 refusals as a Polish bot retaking Antarctica in
seconds. What it costs is the click a counter at 0 would still have won, when a
click in flight fails. With no reading, nothing is held.

**A click inside a rally's ring of its flag takes half a token** (`charges.rally.cost`): the globe asks the bank for
that much (`canClick(budget, at, cost)`), `clickTile` counts the click in flight at it, and the answer's
`ClickBudget.spent` says what it took, so the reading carries `rally` while a ring of the flag is on. The bank is shown
in tenths. See [Rally](rally.md).

**One bank, one token per click elsewhere.** The bank's size never moves: not with the
country, a bonus or signing in. The more of the map a country holds, the slower
its players refill; signing in refills faster, and a refill charge fills the bank. The
server sets that pace on each click, from the country clicked for, so a switch
of flag moves nothing on the meter until the next click. The reading carries the
selected country's slowdown beside it (`ClickBudget.price`): `useClickBudgetFeed`
puts each reading in the click budget store (`useClickBudget`, see [Stores](stores.md)) and
calls `priceFor(country)` whenever the selected country changes, and
`PlanetBackend` keeps the count of every reading but only the price of one for
that country. The meter only explains the price (`priceNote` in `domain/clickPrice.ts`, which `ClicksPanel`
words): it says nothing at the plain rate unless the country is within 80% of the first step.

A server that reports nothing — no throttle, or one too old for the call —
leaves the counter hidden rather than showing a made-up allowance, so this ships
ahead of the backend. `FakeBackend` implements the same interface off its own
bucket, so the counter is live in dev.

`listenForUpdates` goes through `openStream`, which reopens with a capped
exponential backoff. It is the only source of live changes, so a drop that is not
retried freezes the globe until a reload.

**A stream that opens again makes the globe read again what it may have
missed.** `listenForOpenings` is called with the opening heartbeat's version,
after `PlanetBackend` has flushed the updates from before the gap, and a
listener that comes late hears at once the version the stream already passed.
The globe reads again, at once, the counts and every range older than that
version, and again 5s later (`max-age`) while a read comes back older. What the
stream said since still wins, by its version. A gap in which nothing moved reads
nothing. See [The counts are the server's](counts.md).

**The planet stream carries the click token it can have without a mint.** Each
(re)connect puts `SessionProvider.heldIdentity()` in `X-Session-Token`, so the server
knows which account the stream serves; with none held it opens without one and
the server follows the address. It never calls `token()`: watching the planet is
not worth a Turnstile check. The server reads the token only when the stream
opens, so `followSession` **reopens the stream** when a click, a claim or a bomb
the server accepted went out under a token the stream was not opened with — the
first click of a page load, and the first one after a sign-in or a sign-out. The
hourly re-mint for the same account reopens it too: telling the two apart would
mean reading the token, which is the server's business. A refused call reopens
nothing. The boxes' and the quizzes' streams follow each token a call carried (`listenForSessions`, below).

## The boxes stream

**The boxes come on a stream of their own, the boxes module's** (2026-11-15): `boxes.v1.BoxService/ListenForEvents`,
which sends `BoxEvent` (the backend's [The boxes](../../backend/docs/boxes-module.md)). `BoxesBackend` opens it at
construction through `openStream`, as the planet's: the same backoff, the same silence limit and wakes, and the clean end
of a server restart opens it again 500 ms later.

- **It carries the token held, never a mint** (`heldIdentity()`), and the server reads it only as it opens. **A box goes
  only to a caller with this stream open**, to the account it names (the backend's
  [Bonus boxes](../../backend/docs/bonus-boxes.md)). So it follows the identity the cookie resumes, and each token a
  planet call carried (`PlanetBackend.listenForSessions`, a `SessionSource`): a signed-in account's stream hears the
  boxes as that account.
- **`box_offered` goes to `listenForBoxes`** (the globe's flying box): a question mark, its token, seed and expiry
  (`boxOf`), and no kind. Until 2026-11-15 the offer named its kind, and `offerOf` dropped one it did not know.
- **`ClaimBox` goes to `BoxService`**, with a click token: minted, and minted again after an `unauthenticated`.
  `asBonusError` reads its refusals; a claim refused while the boxes or the map take no play (`FailedPrecondition` with
  `boxes.v1.FrozenRefusal`) is a failed claim. **`ClaimBox` pays nothing**: the bonuses module draws what the box holds
  and pays it about a second later, and planet's stream tells it (`charges_granted` with what it `gained`).
  `BoxesBackend.claimBox` listens to the grants (`PlanetBackend.listenForGrants`, a `GrantSource`) from before the claim
  goes out, and answers the first one after it as what the box held, or nothing when none comes in
  `BOX_REWARD_WAIT_MS` (2.5 s), and then no card shows. A kind paid into a full slot comes with 0 kept, and the card
  names it ([Charges](charges.md)). A grant of another kind in that window (an operator's) would be taken for the box's,
  as for a right answer.
- **Whether a first sign-in earns a refill, and the gates** are the bonuses module's rules (`GetGrantRules`, a GET as
  `GetChargeRules`). `PlanetBackend` reads them at load and tells them as `BonusRules.signInRefill`, `gatesOn` and
  `openKinds`, with planet's rules, through `onRules`, nothing before planet's are read. The bonuses module has no stream
  since 2026-11-15, so `refreshGrantRules` reads them again when the gamemaster's turn moves to another entry, and tells
  them only when they changed (see [Charges](charges.md#the-inventory)).
- **The ladder** is progression's `GetLevel`: `ConnectProgressionBackend.readLevel()` reads the player's level and each
  gated kind's opening level with the session held, and nothing with none, so it never makes one (see [The
  level](level.md#the-numbers), which reads it once for the whole page).
- **The fakes have no such stream**: `FakeBackend` implements `BoxCatcher` itself, pays a box a moment after its catch,
  and `ReplayBackend` offers none.

## The quizzes stream

**The quizzes come on a third stream, the quizzes module's** (2026-11-13): `quizzes.v1.QuizService/ListenForEvents`,
which sends `QuizEvent` (the backend's [The quizzes](../../backend/docs/quizzes-module.md)). `QuizzesBackend` opens it at
construction through `openStream`, as the boxes'.

- **It carries the token held, and follows a new one**: the identity the cookie resumes, and each token a planet call
  carried (`PlanetBackend.listenForSessions`, a `SessionSource`), so a guest who signs in hears the quizzes as its
  account. **A quiz goes only to a caller with this stream open.**
- **`quiz_offered` goes to `listenForQuizzes`** (the quiz store); `quiz_won` is read by nothing yet.
- **`OpenQuiz` and `AnswerQuiz` go to `QuizService`**, with a click token, minted again after an `unauthenticated` for
  the opening. **`AnswerQuiz` answers right or wrong, and pays nothing**: the bonuses module pays a right answer a
  moment later, and planet's stream tells it (`charges_granted` with what it `gained`). `QuizzesBackend.listenForRewards`
  hands each charge a grant gave the account (`PlanetBackend.listenForGrants`, a `GrantSource`) to the quiz store, which
  takes the first after a right answer as its reward ([Quizzes](quizzes.md)).
- **One more stream on the page**, which the bonuses module's, gone on 2026-11-15, offsets. Fine in production on
  HTTP/2; locally, on plain HTTP/1.1, the browser's six connections are already full.

## The flags stream

**The flags come on a stream of the flags module's** (2026-11-23): `flags.v1.FlagService/ListenForEvents`, which sends
`FlagEvent`: a heartbeat at once and every 30 s, and `flag_changed`, a flag opened, closed or renamed while the stream
is open, as the catalog holds it now. `ConnectFlagsBackend.listenForFlags` opens it through `openStream` while the
flags store listens (`useFlagsFeed`). It carries no token: it is the same for every page.

- **Each connection's first message says the stream opened** (`onOpened`), the first connection's included. The store
  then reads `ListFlags` again past the cache: nothing told before the stream opened is sent again, so a read covers
  what changed while it was closed, and the race between the load read and the stream's opening.
- **`flag_changed` goes into the catalog with no read** (`flagChangedOf`, `onChanged`). Any other case is a heartbeat, or
  one this build does not know, and is dropped.
- **A server with no flags module** answers `Unimplemented`: the stream stops for good (`stopsOn`), and `ListFlags`
  lists none.
- **One more stream on the page: seven for a guest.** Fine in production on HTTP/2. Locally, on plain HTTP/1.1, a guest
  page already held six, which fill Chrome's six connections to a host: the flags stream waits for one, and so does
  every unary call.
- **`FakeFlagsBackend`** says the stream opened a moment after it is listened to, and tells each `openFlag(name)`,
  `addTime(code)`, `endFlag(code)`, `claimElsewhere(code)`, `removeFlag(code)` and `renameFlag(code, name)` as
  `flag_changed`. The archive page opens no stream: it
  reads the list once.

See [The list of flags](flag-catalog.md).

## Connection health

**Every call is the probe, so there is no ping.** `connectionInterceptor`
(`connection.ts`) sits on every Connect transport (`Config.interceptors`, set in
`main.tsx`) and tells one `ConnectionHealth` how each call went. A health RPC
would only say whether the health RPC gets through.

- **Any answer is the server reached**, a refusal it sent included. A failed
  fetch (connect-web's `Unknown` over a `TypeError`), a timeout
  (`DeadlineExceeded`, the 10s `timeoutMs`) and a proxy with nothing behind it
  (`Unavailable`, a 502 to 504) are misses. A call the page cancelled says nothing.
- **A stream is followed past its opening**: each message is the server reached,
  and a break is a miss.
- `DOWN_AFTER_MISSES` (2) in a row is down, and the first answer is up again. The
  browser's `offline` is down at once (`followBrowser`); `online` waits for an
  answer, which `openStream`'s wake brings at once.
- **`PlayScreen` shows `ConnectionLost` in the Moments zone while it is down.** Clicks
  still go out and roll back: they are how it finds the way back. A deploy's
  restart reads as down too, which is what it is from here.
- The fakes have no transport. In fake mode `connection` is on `window`:
  `connection.wentOffline()` raises the card and `connection.reached()` drops it.
