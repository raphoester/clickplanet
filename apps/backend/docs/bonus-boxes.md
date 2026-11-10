# Bonus boxes

A question-mark box flies past the planet every so often; whoever catches it
gets one of five bonuses. Each box draws its kind from the rule set's `bonuses.kinds` (see [Rules](rules.md)), a weight
per kind — a kind's chance is its weight over the sum of the weights, so the
strong ones can be made rare (production runs 5 : 3 : 1 : 2 : 3):

- **`refill`** — a charge: fills the caller's click bank to full, when the
  caller chooses, and gives the network's bucket as much. Up to one bank, 60 clicks. See [What a refill does to the bucket](refill.md).
- **`spread_clicks`** — 1 to `bonuses.spread.maxPerBox` (4) clicks added to a pool
  of at most `bonuses.spread.clicks` (8). While the player switches spread on, each
  click spends one and also takes the tiles touching the one clicked. See [What a spread does to a click](spread.md).
- **`bomb`** — a charge: one bomb, kept until it is dropped (never offered while `bonuses.bomb.off`). It
  clears a circle of `bonuses.bomb.rings` tile spacings around where it lands,
  whoever holds the tiles. See [What a bomb does](bomb.md).
- **`enclose_clicks`** — 1 to `bonuses.enclose.maxPerBox` (3) enclosures added to a
  stack of at most `bonuses.enclose.held` (3). While the player switches enclose on,
  a click that closes shapes of its flag's tiles also takes the tiles
  inside each, at most `bonuses.enclose.maxTiles` (25) a shape, and spends one.
  See [What an enclose does to a click](enclose.md).
- **`shields`** — `bonuses.shield.minPerBox` (5) to `bonuses.shield.maxPerBox` (20)
  shields added to a stack of at most `bonuses.shield.held` (30). Each is placed on a tile the player's flag
  holds and takes one foreign click there. See [Shields](shields.md).

Every kind is a **charge**, worth about one bank, rather than a timer — see
[Charges](charges.md). There used to be a `triple_clicks`
that multiplied the refill for two minutes. With a bank of 60 it was worth "some
clicks, maybe": nothing on a full bank, and a different amount for every
country's pace. The refill is the same good, clicks, with the moment chosen by
the player.

Boxes are always on: there is no switch.

**The waits and the hourly cap are the rule set's** (`bonuses.Pace`, see [Rules](rules.md)), not the config's: the registry reads them at each sweep, so a set taken reaches every caller within a second.

**The server picks who gets one, and that is the whole design.** A box broadcast
to everyone is caught by whichever client reacts fastest, and that is a script
every time: it reads the event off the stream and answers in twenty milliseconds
while a person is still moving the mouse. Broadcasting would make this a machine
for handing extra clicks to exactly the callers [Anti-bot](antibot.md)
exists to stop.

So an offer is **addressed**: it is sent down one caller's stream, and nobody
else sees it or can claim it. Reflexes buy nothing, and what is left to do —
notice the box and click it — is the part that was meant to be the game.

**Every caller is on a schedule of their own.** A single server-wide ticker
drawing one winner made the rate each player saw `1/(interval × players)`, so
the feature got rarer the busier the game was — a box every couple of minutes at
ten players, one an hour at three hundred. What a player experiences must not
depend on how many other people are online.

That schedule is one `nextOfferAt` per caller and **one sweep for everybody**,
not a timer each: `bonus.sweepInterval` walks the map the way the antibot sweeps
already do.

**The wait is drawn uniformly from `[minInterval, maxInterval]`.** The spread is
for feel and not for defence — a script does not predict the schedule, it
watches the stream, so there is nothing here to hide from one.

**The pace answers what the player did with the last box:**

- **Missed** — the token lapsed unclaimed, which the sweep sees without the
  client saying anything — the next one comes at `missRetry`. That applies to
  **one** miss; a second in a row waits the ordinary window, or a tab that never
  catches anything would collect a box every `missRetry` forever.
- **Caught** — a charge has no end, so the next is due a window after the
  **claim**, not after it is spent: a bomb held for an hour does not hold back
  every other box for an hour.

**Only callers who have clicked inside `activeWithin` are offered anything.** A
tab left open overnight is not playing, and it is also what keeps the miss rule
from needing a back-off of its own. A caller whose turn comes up while they are
away **loses the slot rather than banking it** — otherwise they are handed a box
the instant they come back.

**A schedule outlives its stream by `forgetAfter`.** Without that, closing the
tab and opening it again draws a fresh wait, and a player could reload until
they got a short one.

**The rule set's hourly cap bounds what a caller can be granted.** Nothing here is a
race any more, but catch rate is where an advantage is left: a script catches
every box it is offered where a person catches some. This makes the worst case
a number you choose rather than a function of reflexes. Past 12 charges in the
last hour the slot is lost, like a caller's who was away; 12 is above the ten
boxes an hour a person catching every one would get. A kind another box would
add nothing to (a refill or a bomb held, a full stack or pool) is not offered.

**A caller is an entrant, not a connection.** `bonuses.EntrantOf(payer)` is the
account (`account:<id>`) when the click token says it is linked, and its scope
(`cpipscope.Of`) otherwise. `Attend`, `Clicked`, `Claim`, `OpenQuiz`,
`AnswerQuiz` are all keyed on it, and so is the `yours` each stream tells of an enclosure; it holds every
stream sharing it — so twenty tabs are one entrant on one schedule, and all of
them are sent the box. Two signed-in players behind one address are two
entrants: each gets its own boxes and quizzes, and neither can claim the other's.
The guests behind an address are still one entrant, because a guest account is
free to make and a schedule each would be a box each per private tab. A linked
account keeps its schedule across networks. The charge a box grants lands on the
claiming account either way. **The antibot still hears the scope**: `Caught` and
`Foreign` report the claiming request's, `Lapsed` the one the entrant last
clicked from, since the catcher judges and bans a network. The handler's `defer`
is what removes a stream; there is no context goroutine per connected client,
because the fanout deliberately does not pay that cost.

The stream and the claim must name the same entrant, so the client opens the
stream with the token it clicks with and reopens it when that token changes
(`followSession` in the frontend's `planetBackend.ts`): signing in moves the
stream to the account at the next click.

**The offer is the state, so there is no crypto here.** The registry already has
to remember who it offered what, so the token is 16 random bytes and the map is
the check: unknown, spent, lapsed, or offered to somebody else all fail the same
way. Unlike a session token there is nothing to verify statelessly — and nothing
to sweep either, since both paths that take the lock forget what has lapsed on
the way past.

**A claim answers `CodeNotFound` and says nothing about why.** The difference
between "no such token" and "not yours" is exactly what a script guessing tokens
would measure.

**`ClaimBonus` is session-gated**, appended to `NewSessionInterceptor`'s
procedure list: a bonus is only ever spent as clicks, and clicks need a session,
so the box that grants them should not be the one way to widen an allowance
without proving anything. It is deliberately **not** throttled — a claim is
already gated on holding a token the server addressed to you, and spending a
click token to collect a bonus is backwards.

**Two new cases on `PlanetEvent`, not a second stream** — `bonus_offered`, which
reaches one caller, and `bonus_taken`, which reaches everyone. The private reward
with a public outcome is what keeps the spectacle without the scramble. A client
too old to know either case reads an unset `oneof` and skips it, which is the
whole reason the envelope exists.

The catch is published **after** the charge is held, so a catch announced to the
planet that then failed to apply is the one lie this cannot tell. The same use case then
tells the other modules who caught it, `planet.v1.BonusClaimed`, for the `Loot` titles;
`answer_quiz_usecase` tells every answer as `planet.v1.QuizAnswered`, for `Quiz`.
