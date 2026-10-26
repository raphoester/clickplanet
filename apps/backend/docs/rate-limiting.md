# Rate limiting

`throttle_click` throttles clicks from `shared/cpratelimit` token buckets. `MapDensity` and `GetMap` are cacheable reads a proxy in front absorbs; limiting them would punish a page load rather than a bot. A refused click answers `CodeResourceExhausted`, i.e. HTTP 429, and never reaches the map.

## Two buckets per click

**A click spends from two buckets at once: its account's and its scope's.** The account is the one the click token names; the scope is the address, or its /64 over IPv6 (`cpipscope`). The account's bucket is the rule set's `clicks.perSecond` and `clicks.burst` (see [Rules](rules.md)): production runs one click every 5s (0.2) with a bank of 60, which fills in five minutes. The scope's is `clicks.scopeMultiplier` (10) times that, because many players can share one address — a campus, a school, a carrier NAT.

- **A guest spends from a third: the one every guest behind its scope shares** (`guests:<scope>`), at `clicks.guestScopeMultiplier` (1) times one account's. A guest account is one per browser, and a private tab or a second browser is a new one, so without it ten tabs were ten banks, up to the scope's ten. Now the guests on one network are one bank. **Nobody is refused for being second** — there is no slot to win and none to free — and two people behind one address share until one signs in, which is a reason to. It takes a guest's pace, so ten tabs on a big country refill no faster than one. A linked account never spends from it. `TestGuestsOnOneScopeShareOneBank` pins it.

- **An IPv6 network's bucket is smaller** (`clicks.v6ScopeMultiplier`, production 4). A /64 is one subscriber line — a home, a phone's hotspot — not a campus behind one IPv4 address, so at 4 it holds two signed-in accounts at their pace and no more. From 2026-10-12 one household on a phone hotspot played ten email accounts in turns, four at once, and made 36% of all clicks: under the scope's 10×, every account it added was a bank more. Replayed over 2.5 days of clicks, 4× takes 40% of that household's clicks and 1-2% from two other networks. IPv4 keeps `clicks.scopeMultiplier`. **Known risk**: a network that gives one /64 to many people (some campus Wi-Fi) shares 4×. `TestTwoSignedInAccountsFillAnIPv6NetworkAndNotAnIPv4One` pins it.

- **A linked account refills faster, into the same bank.** An account that signed in with Google, Discord or an email code refills `clicks.linkedMultiplier` (2) times as fast, to make signing in worth it; its bank is the same size. It is the same `account:<id>` bucket either way — the rate is its pace, set by each click (see below) — so signing in neither tops the bank up nor empties it. The scope's bucket still bounds it. **Planet learns it from the token**, which carries a linked byte (see [Sessions](sessions.md)), so a click costs no call to `auth`. A player who links mid-session refills at 1× until the client mints again. `TestALinkedAccountClicksTwiceAsFastAsAGuest` and `TestSigningInKeepsTheBankAndSpeedsUpItsRefill` pin it.
- **A new account earns its bank; it is not handed one.** Its own bucket starts with `clicks.newAccountClicks` (production 10) and earns the rest at the plain rate from the moment the account was made, so at 0.2/s it is full in under four minutes. From 2026-09-28 to 10-01 one phone took a new mobile /64 and a new guest account about once a minute and spent each fresh 60, about 4.5 times a guest's rate; now each new account is worth 10. A set that wants every account to start full writes `clicks.burst`.
  - **Planet reads the age off the id.** Auth makes every account id a UUIDv7, whose first 48 bits are when it was made, so `cpsession.AccountID.CreatedAt` needs no call and no change to the token. The session interceptor puts it on the context beside the account, and `clicks.PayerOf` reads it into `Payer.Created`. An id of another version says nothing and starts full. `uuid_id_provider`'s test pins that the ids it makes say when.
  - **The limiter computes it when it makes the bucket** (`cpratelimit.Key.Since` and `Start`): `Start` plus what the plain rate earned since `Since`, never past the burst. So a restart, a sweep or a `Peek` cannot hand the bank over early, and nothing is stored per account. The sweep keeps a bucket a refill topped up early until it has earned its burst, or it would come back short.
  - Only the account's own bucket: the guests' and the scope's have no age. A fresh account is still bounded by them. `TestAFreshGuestOnAFreshAddressEveryMinuteIsNoFreshBank` replays the phone over HTTP.
- **A click is refused when either bucket is empty, and a refusal spends from neither.** `Limiter.TakeAll` checks every bucket and spends from all or none under one lock, so a player refused for a busy scope keeps its own tokens.
- **A token with no account spends one bucket, the scope's at 1×** — exactly the throttle from before accounts. The deprecated `session.v1` mint, an invalid token while `auth.enforce` is off, and `auth.enabled` false all land here. It is a separate bucket from the scope's shared one, because a key never changes its scale.
- **One limiter holds both.** A bucket's `Scale` multiplies its burst and rate, and follows its key from each take (what it earned before is kept, cut to a smaller size); `clicks.Buckets` names the keys (`account:<id>` at 1, `scope:<scope>` at the scope multiplier, or the bare scope at 1 with no account). **A key's `Pace` multiplies the payer's own bucket's rate from that take on, and leaves its burst alone**: the linked multiplier for a signed-in account, divided by the country's slowdown (see [A big country refills slower](#a-big-country-refills-slower-clickstoll)). The scope's bucket takes no pace. **The rule set's tempo multiplies every bucket** on top of its pace: the limiter reads its policy from the set in force at each call (`cpratelimit.NewRuled`; see [Rules](rules.md)). `clicks.PayerOf(ctx)` reads the scope and the account off the context, so the throttle, `GetBudget` and a bonus claim cannot disagree on who pays.
- **What this buys.** Before accounts, one address was one allowance, so a campus played as one player and a bot farm with many cookies on one address was no worse off than one tab. Now each signed-in player behind an address has its own allowance, bounded together by the scope's, the guests behind it share one, and a bot moving across addresses keeps spending one account's.
- `TestManyAccountsOnOneScopeShareTheScopesBucket` and `TestOneAccountOnManyScopesSpendsOneAllowance` pin both halves over HTTP.

**It is a decorator over the click use case, not an interceptor over the procedure.** Two things fall out of that. The allowance comes back as a return value (`click_usecase.Out`) instead of being left on the context for a handler to find, which is what `cpctx.AddRateBudgetToContext` existed for and why it is gone. And "a click refused for its address or its session must not also spend a token" stops being a rule about the order of a list and becomes a property of the shape: every interceptor is outside the whole click chain by construction. `MapDensity` and `GetMap` are untouched for free, being other procedures entirely — under an interceptor that took a procedure list.

## Saying what is left

The web app shows the player how many clicks they have in hand, and **the server is the only thing that knows**. A client running its own copy of the bucket would drift within seconds — it cannot see the clicks the same address makes from another tab, and its idea of when a click was spent is a round trip out of date.

Polling for it would be worse, so nothing polls. `Limiter.TakeAll` returns each bucket's state alongside its verdict, and the state carries the **policy** (`Capacity`, `PerSecond`) as well as the reading: given both, a client replays the same refill arithmetic between two answers and is exact without asking. The pip count and the fill rate on screen are therefore the server's burst and refill rate — **changing the rule set's `clicks` changes the display with no frontend release**, and an open tab reads it again on `rules_changed`.

That reading travels two ways, because a refused call has no response message to put it in:

- an allowed call carries it on `click_usecase.Out`, and `click_handler` puts it in `ClickResponse.budget`.
- a refused one carries it on the same `click_usecase.Out`, beside `clicks.ErrThrottled`, and `click_handler` attaches it as a **connect error detail** — a refusal has no response message to put it in.

Either way the decorator decides the policy and the handler decides how to say it. `clickbudget.Encode` is the one place that shape is agreed, because two procedures answer with a `ClickBudget`: the click that just spent a token, and `GetBudget`.

**Every reading also carries `linked_multiplier`**, what signing in multiplies the refill by (`Buckets.BudgetOf`). It is the same for every caller, so the client can tell a guest what signing in is worth with no number of its own.

**The reading is the tighter bucket** (`Buckets.BudgetOf`): the one with fewer tokens, on a tie the smaller one, and on a full tie the payer's own. A player behind a busy campus sees the scope's limit rather than a full meter that refuses. The reading is that bucket's `Tokens` and `PerSecond`, so the client arithmetic does not change. `TestTheBudgetIsTheTighterBucket` pins it.

**Its `Capacity` is always the payer's own bank**, whichever bucket is tighter. A shared bucket is a ceiling, not a bigger bank: on 2026-10-23 two signed-in players on one IPv6 line read "24 of 240", and the client, replaying the scope's rate up to its capacity, would have counted past the 60 either of them could spend. The tighter bucket never holds more than the payer's own, so the count always fits under it.

**It says who else spends from that bucket** (`ClickBudget.shared_with`: nobody, the guests behind the address, or every player behind it). Without it a guest whose other tab, or a stranger on the same carrier address, spent the bank sees its count drop by clicks it never made. A guest alone on its network holds as much in its own bucket as in its guests', and the full tie goes to its own, so it is never told it shares.

`ClickService.GetBudget` covers the cold start — a client that has just loaded and has no click to learn from. It reads through `Limiter.Peek`, which spends nothing and, for an address that never clicked, **creates no bucket**: reading an allowance must not be a way to make the limiter remember a caller. `NewSessionReaderInterceptor` reads a token on it when the client sends one, so the reading is the account's, and **refuses nothing**: with no token or a bad one it reads the scope's bucket from before accounts. The web client sends the token it already holds, never a fresh one; before its first click it holds none, and neither bucket has been spent from. Without the token the meter showed that other bucket, always full, until a click contradicted it. It is deliberately not `NO_SIDE_EFFECTS`, so it is a POST no cache will serve a stale answer to; every click re-anchors the client afterwards, so it is asked once per page load.

**This tells a scripted clicker exactly when to fire**, which is a real cost against [Anti-bot](antibot.md). It is a small one — a script can already infer the same schedule by counting its own 429s — and it is paid to stop honest players being refused with no warning.

The scope is whatever `IPReaderMiddleware` put on the context: `X-Real-IP` if present, otherwise the peer address. **The reverse proxy must set that header itself** — `deploy/vps/caddy/Caddyfile` does, with `header_up X-Real-IP {client_ip}` on every backend route. Merely forwarding it would let a client send its own and buy a fresh bucket per request. The fallback is the peer address rather than a constant precisely so a missing header degrades to per-connection buckets instead of rate limiting the whole game as one player.

## A big country refills slower (`clicks.Toll`)

**Every click costs one token.** What the map share changes is how fast the
tokens come back. The rule set's `toll.steps` is a table of `{share, slowdown}`: from `share` of
**every tile on the map**, a player of that country refills `slowdown` times
slower (production runs 1.5× from 10%, 2.5× from 20%, 4× from 30%). No steps
refills every country at the plain rate. `ClickBudget.slowdown` took the field
number of the old `cost`, whose meaning it replaces.

- **The bank never changes size**: not with the country, a bonus or signing in.
  Only its rate moves, so the number a player sees changes when it clicks, when
  time passes, or when a bonus is caught — never on a switch of flag.
- **The pace is set by each click, from the country clicked for, and applies from
  then on.** The time already past was refilled at the pace in force over it, so
  switching flags moves nothing until the next click. A refused click sets it too.
  A player can refill on a small country and spend the bank on a big one; the
  bank bounds what that buys.
- **The share is of the whole map, not of owned tiles**, so early in a game nobody is slowed.
- **`inmemory_tile_storage` keeps a tile count per country**, moved by `set` and
  `Clear` and rebuilt at boot from postgres, so `Share` is one read and no scan.
- **Only the payer's own bucket is slowed.** The scope's is a ceiling shared by
  players of every flag, so it refills at its plain rate.
- **The budget goes out as the bucket holds it**, in clicks, with the slowdown,
  the share and the next step of the country asked about so the client can say why.
  The whole table goes out once per page load, in `GetBonusRules.toll_steps`.
- **Bonuses compose with it.** A refill fills the bank to its size and leaves the
  pace alone. A spread is one click. A bomb is not throttled, and lowers the
  share of whoever it hits.

`GetBudget` takes the country, for the slowdown it answers. Known risk, not
handled yet: a country sitting on a step can cross it back and forth click to click.

Chat and sessions each have **their own limiter instance** with their own budget, because what each call costs has nothing to do with what a click costs:

- `chat.rateLimiter` — one message every 3s, five in hand. A message fans out to every connected client and lands in a log everyone will read.
- `chat.reactionLimiter` — one reaction a second, ten in hand. A reaction is a short row and a small frame.
- `chat.seenLimiter` — one seen mark a second, ten in hand. The client sends one at most every 2s anyway.
- `auth.rateLimiter` — one mint every 30s, ten in hand, across both `CreateSession` paths. A mint costs a siteverify round trip to a third party, so an unthrottled `CreateSession` is a free way to spend this server's Turnstile quota.
- `auth.resumeLimiter` — one `ResumeSession` a second, ten in hand. It costs one row read and no third party.
- `support.rateLimiter` — one note or report a minute, three in hand. Each is a row the operator reads by hand.
- `support.noteLimiter` — ten suggestions and bug reports a day, per account and per address together.
- `support.uploadLimiter` — a hundred files a day, per account and per address together. Each is room in a bucket we pay for.
