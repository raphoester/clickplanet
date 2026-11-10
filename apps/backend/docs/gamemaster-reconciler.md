# The gamemaster as a reconciler

**Each second the gamemaster compares what the game wants with what it knows, and sends only the commands that close
the gap** (the user, 2026-11-10). Before, each step sent a command once per process and waited for a fact that could
never come: an arena battle whose round never opened in scoring stalled every round, day and season after it, and a
season with no break could wipe the map before its Final Battle was scored.

## The rule

- **Wanted**: from the plan (`moments.Plan`, read once a tick, see [The gamemaster](gamemaster.md)) and the clock.
- **Known**: what the receivers told, kept in the gamemaster's postgres, and what Redis took from this process.
- **A close is sent again each minute until its answer comes** (`sendAgain` in `module.go`). An open is sent once
  per process: a boot sends again what is open now, and each receiver keeps it once.
- **Every wait is on the answer to a close, and every receiver answers every close.** So nothing waits for ever.

## What each module is told, and how the gamemaster knows

| What | Wanted | Known from | Command |
|---|---|---|---|
| planet's and the antibot's rule set | the turn's set | the module compares `decided_at` | `PlayRuleSet`, at boot, at each change, and again each minute with the same `decided_at` |
| a scoring book | open while its season runs, closed once it ended | `BookClosed` or `BookMissed` (`closed_books`, `missed_books`) | `OpenBook` once, `CloseBook` until an answer |
| a scoring round | open while it runs (judged, the map back), scored once it ended | `RoundScored` or `RoundMissed` (`scored_rounds`, `missed_rounds`) | `OpenRound` once, `CloseRound` until an answer |
| a planet arena | open while its arena battle runs and scoring answered the round before it; closed once scoring answered the battle's round | `ArenaClosed` (`closed_arenas`) | `OpenArena` once, `CloseArena` until an answer |
| a map reset | done once a season after season 0 runs, scoring answered the book before it and no arena waits | Redis took it | `ResetMap` once per process |

**Answered**: a round scored or missed, a book closed or missed, an arena closed (`Plan.settled`, `Known`). **Every round
of an answered book is answered**: scoring closes a book after its rounds, on the one stream.

**Scoring tells a close it cannot apply**: a `CloseRound` of a round it never opened records `scoring.v1.RoundMissed`, and
a `CloseBook` of a book it never opened `scoring.v1.BookMissed`, through its outbox, at each such close (see
[Scoring](scoring.md#the-commands)). The gamemaster keeps each once (`gamemaster-rounds-missed`,
`gamemaster-books-missed`) and tells the players nothing: nothing scored it. The answer comes on scoring's one ordered
stream, after every command before it, so "never opened" is exact. A read of scoring over the internal listener was
not chosen: it does not see a command still in the stream, and would load scoring's pool every second.

**Each command is idempotent on its receiver**: a book, a round and an arena by its id, a reset by its id, a rule set by
its `decided_at` (the latest decision wins). A missed round or book is told again at each close, and kept once.

## The two stalls

- **An arena battle whose round never opened in scoring** (judged in its last second, or the open refused and the
  server down at its end): `CloseRound` → scoring tells `RoundMissed` → the round is answered → `CloseArena` → planet
  closes an arena that never opened with the map left as it is → `ArenaClosed` → the day after opens. About 5 seconds.
- **A season with no break, or a restart across its start**: the reset waits for scoring's answer to the book before
  (`Plan.ResetAt`). Scoring closes the book after the finale's round on the same stream, so the finale is read on the
  map it was fought on, and only then does planet wipe it.

## Boot, downtime, refusal

- **A boot** sends the opens due now, the closes not answered yet, the reset if it is due (planet keeps its id), and the
  turn. Not every close since season 0, as before: those rounds and books are answered.
- **After downtime**: what ended meanwhile gets its close. A round that never opened comes back missed. A battle that
  started and ended while the server was down was never judged, so it has no arena.
- **A refusal for good**: scoring refuses no close any more, it answers missed. A refused open (scoring's book not
  open, planet's arena past its end) holds nothing back: its round is missed at its end. A refused rule set leaves the
  module on its own set, logged at Error there; the turn sent again each minute plays it once the operator makes the
  set, with no new entry.

## Cost

Per tick, on the gamemaster's pool of 2: the plan reads the seasons, the entries, then what scoring answered (one
statement, `postgres_moment_store.Kept`) and the arena battles (one statement, `postgres_arena_store.ArenasKept`): 4
reads, 6 before. With the starts' transaction (5 round trips) and the schedule's 2 reads, about 11 round trips a second.
No command in a steady state; one `XADD` a minute for each close with no answer, and 2 a minute for the turn. No call
on the internal listener but the judgement, once per battle.

## Known gap

**An arena battle whose arena never opens** (judged too late, or `OpenArena` refused) still counts from nothing on the
world's map, so each country gains its whole land. The fix is to open its round only once planet tells a new
`ArenaOpened`.

## Tests

`play_scoring_usecase`'s tests pin a close sent again each minute until scoring answers it, a boot after a season whose
book closed that sends nothing, and an arena battle judged in its last second that scoring tells missed;
`play_arenas_usecase`'s the arena closed until planet tells it, and closed after a missed round; `play_resets_usecase`'s
a season with no break reset once the book before it is closed, or told missed; `play_schedule_usecase`'s the turn sent
again each minute with its decision time, and a set refused that plays once it is made. `arenanight` plays an arena
battle whose open Redis refused, with the server down to an hour after its end; `finalnight/no_break_test.go` plays
season 1 right after the Final Battle, and a restart across that end, with scoring answering late. Scoring's
`season_end_test.go` pins a close of a round or book never opened told missed at each close.
