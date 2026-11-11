# The gamemaster as a reconciler

**Each second the gamemaster compares what the game wants with what it knows, and calls the modules only to close the
gap** (the user, 2026-11-10). Before, each step sent a command once per process and waited for a fact that could never
come: an arena battle whose round never opened in scoring stalled every round, day and season after it, and a season
with no break could wipe the map before its Final Battle was scored.

**Each call is a Connect RPC that answers at once** (the user, 2026-11-11): the gamemaster calls until it gets a definite
answer, and keeps the answer in its own postgres right after the call. Before, the commands went over Redis Streams and
the answers came back as events through the receiver's outbox. The reconciler already sent each close again until
answered, so the broker added no durability: it only added places where an answer was lost (the outbox relay, the
stream, the subscriber that buries an event after 8 tries). A buried `ArenaClosed` held every round, day, season and
reset after it.

## The rule

- **Wanted**: from the plan (`moments.Plan`, read once a tick, see [The gamemaster](gamemaster.md)) and the clock.
- **Known**: what the receivers answered, kept in the gamemaster's postgres.
- **A call is made until its receiver answers it** (`moments.Calls`): once per process for an answer kept in memory (an
  open, a refusal), and until it is kept in postgres for a close.
- **A call that got no answer is made again a minute later** (`callAgain` in `module.go`): `Unavailable`,
  `DeadlineExceeded`, `Internal`, any error but a refusal for good.
- **No call goes before the ones ahead of it are answered.** A step calls in the plan's order and stops at the first call
  not answered, or waiting for its minute. A round opens after its book; a book closes after its rounds.
- **Every wait is on the answer to a close, and every receiver answers every close.** So nothing waits for ever.

## What each module is asked, and how the gamemaster knows

| What | Wanted | Known from | Call |
|---|---|---|---|
| planet's, the antibot's and the offers' rule set | the turn's set; planet's and the offers' a battle's only once the battle is judged and its arena opened | all three answered the turn (in memory) | `planet.v1`, `antibot.v1` and `bonuses.v1` `InternalService/PlayRuleSet`, once a turn, again a minute after an error or a refusal |
| a scoring book | open while its season runs, closed once it ended | `closed_books`, `missed_books` | `scoring.v1.InternalService/OpenBook` until answered, `CloseBook` until answered |
| a scoring round | open while it runs (judged, the map back, an arena battle's arena opened), scored once it ended | `scored_rounds`, `missed_rounds` | `OpenRound` until answered, `CloseRound` until answered |
| a planet arena | open while its arena battle runs and scoring answered the round before it; closed once scoring answered the battle's round | `opened_arenas`, `closed_arenas` | `planet.v1.InternalService/OpenArena` until answered, `CloseArena` until answered |
| a map reset | done once a season after season 0 runs, scoring answered the book before it and no arena waits | `reset_seasons` | `planet.v1.InternalService/ResetMap` until answered |

**Answered**: a round scored or missed, a book closed or missed, an arena closed (`Plan.settled`, `Known`). An arena
opened is not an answer to a close: it is what an arena battle's round waits for before it opens.

## The answers

- **Scoring**: `CloseRound` answers `scored_at` (the end the round was scored at, by this close or one before) or
  `missed_at` (it never opened); `CloseBook` answers `closed_at` or `missed_at` (see [Scoring](scoring.md#the-calls)). A
  round scored and a book closed are kept with their moment (`DayEnded`, `BattleEnded`, `SeasonEnded`) in one
  transaction, the moment told only when the answer is new. A missed round or book is kept and told nobody.
- **Planet**: an empty answer is the arena open or closed, the map reset, the set played.
- **A refusal for good** (`FailedPrecondition`, `InvalidArgument`, read by `moments/rpcrefusal`) is a definite answer:
  kept in memory, logged at Error, and the call is not made again in this process; the calls after it go on. A refused
  open holds nothing back: its round is missed at its end. Planet refuses an arena past its end, closed already, or
  beside another; scoring refuses an open of a book not open, or of a round or book it answered missed. A close or a reset
  is refused only for a bad id or time.
- **A rule set is called until every module plays it** (planet, the antibot and, since 2026-11-11, the offers). A set a module keeps none of is refused (`FailedPrecondition`),
  and called again each minute with the turn's decision time: it plays once the operator makes the set, with no new
  entry. Each keeps the latest decision by `decided_at`, so a call made again changes nothing.

## A call cut by its timeout

**A call that times out may still run on its receiver.** So:

- The gamemaster sends no call that depends on an earlier one until that one has a definite answer (above).
- **Scoring never lets an open land after a close it answered missed**: it keeps the missed round or book, and the open's
  check and insert, and the close's, each run under one lock per id (see [Scoring](scoring.md#the-calls)).
- **Planet's swaps of the map run detached from the request** (`context.WithoutCancel`): `OpenArena`, `CloseArena` and
  `ResetMap` never stop halfway when the caller gives up. The next call, a minute later, finds the work done and answers
  it. Planet keeps one arena open at a time and refuses to open one closed already.

**A slow call holds no other step**: the map swaps, a call of seconds, run in a runner of their own
(`gamemaster-maps`: the reset, then the arenas), beside `gamemaster-plan` (the battle's judge, scoring, then what
starts) and `gamemaster-schedule` (the rule sets). Each reads its own plan; what one waits on from another is kept in
postgres. The judge is in `gamemaster-plan`, so a battle's round opens in the tick it is judged.

## The two stalls

- **An arena battle whose round never opened in scoring** (judged in its last second, or the open refused and the
  server down at its end): `CloseRound` → missed → the round is answered → `CloseArena` → planet closes an arena that
  never opened with the map left as it is → the day after opens. About 2 seconds.
- **A season with no break, or a restart across its start**: the reset waits for scoring's answer to the book before
  (`Plan.ResetAt`), and the book closes only after its last round is answered, so the finale is read on the map it was
  fought on, and only then does planet wipe it.

## Boot, downtime

- **A boot** calls the opens due now, the closes not answered yet, the reset if it is due and not kept, and the turn.
  Every receiver keeps what it did by its id, so an open called again answers open.
- **After downtime**: what ended meanwhile gets its close. A round that never opened comes back missed. A battle that
  started and ended while the server was down was never judged, so it has no arena.

## Cost

Per tick, on the gamemaster's pool of 2: `gamemaster-plan` and `gamemaster-maps` each read the plan (the seasons, the
entries, what was answered in one statement, `postgres_moment_store.Kept`, and the arena battles in one,
`postgres_arena_store.ArenasKept`): 8 reads. With the starts' transaction (5 round trips) and the schedule's 2 reads,
about 15 round trips a second. In a steady state, no call: a call per open and per close as each comes, and one a minute
for a call with no answer. A call is a loopback HTTP request on the internal listener.

## Tests

`play_scoring_usecase`'s tests pin a close called again each minute until scoring answers it, with the book held back,
an answer that cannot be kept called again, a refusal for good not called again, the moments told once, and an arena
battle judged in its last second that scoring answers missed; `play_arenas_usecase`'s each answer kept, a call planet did
not answer made again a minute later, and a refused open; `play_resets_usecase`'s a season with no break reset once the
book before it is closed or missed, the reset kept for the next boot, and a refusal; `play_schedule_usecase`'s a turn
played once, a set refused that plays once it is made, and a call made again with the turn's decision time.
`arenanight` plays an arena battle with planet slower than the call (cut, then answered a minute later), with boots in
the middle, scoring down at its start and the server back an hour after its end, and an open planet refuses at its end;
`finalnight/no_break_test.go` plays season 1 right after the Final Battle, and a restart across that end, with scoring
slower than the call. Scoring's use case and handler tests pin a close of a round or book never opened answered missed at
each close, an open after it refused, and the race under the lock; planet's handler tests the swaps that go on when the
caller gives up.
