# Scoring (`internal/scoring/`)

**A points accountant: it knows three words, book, round and grant, by the ids it is given.** A book gathers the points
of its rounds and grants and each player's takes; a round credits its book once it closes; a grant credits every book
open at its time. Scoring knows no season, day, battle, finale,
clock, calendar, cutoff or rule set, and never names the gamemaster. **The gamemaster owns time and the game's words**
(see [The gamemaster](gamemaster.md)): it commands scoring over one command stream, and scoring tells what it scored as
events. Missions grants points the same way, by a command, and scoring never learns the word mission. Planet never
imports, calls or hears scoring: scoring asks planet what each country holds
(`planet.v1.InternalService/GetTerritories`), when a round closes and for an open round's live standings.

Until 2026-10-28 scoring kept a calendar (seasons, daily cutoff), rule sets (`scoring.v1.PlayRuleSet`, a round of its
own per entry), an operator `AdminService` and a runner that closed rounds on its own clock. All of it is gone: the
gamemaster sends the moments, and scoring has no rule set any more.

```
internal/scoring/internal/
  books/                          BookID, RoundID, GrantID (ParseBookID, ParseRoundID, ParseGrantID: 1 to 64 characters),
                                  Book (NewBook, Holds, ClosedAt, Table), Round (NewRound, Holds, ScoredAt, Results),
                                  Grant (NewGrant), Scored, Result, Score, Placed (ScoresOf, TableOf), Take, ErrRefused;
                                  the Books, Rounds, Grants and Takes ports and their contract suites
    postgres_book_store/          Books over books, closed_books, book_tiebreaks
    postgres_grant_store/         Grants over grants
    postgres_round_store/         Rounds and Takes over rounds, scored_rounds, round_results, round_takes,
                                  round_players
    inmemory_book_store/  inmemory_round_store/  inmemory_grant_store/   the same ports in maps, behind the testing tag
    rpc_planet_territories/       the map now, from planet.v1.InternalService/GetTerritories
    usecases/open_book_usecase/   open_round_usecase/   close_round_usecase/   close_book_usecase/
    usecases/grant_points_usecase/
    usecases/count_take_usecase/  forget_account_usecase/   a round's numbers
  standings/                      AccountID, Country, Take, Tally (WithTake); the Store port (RecordTake by book,
                                  DeleteAccount) and its contract suite
    postgres_contribution_store/  inmemory_contribution_store/
    usecases/record_take_usecase/  forget_account_usecase/
  commands/scoring_command_handler/  Dispatcher (one case, one use case) and Handler (drops a command refused, of
                                  either stream)
  commands/grant_points_handler/  parses a GrantPoints for its use case
  commands/log_command/           logs each command applied
  scoringv1controller/            ScoringService and InternalService (bags), the cache and session interceptors
    get_board_handler/board_query/          a book's top 10, of every flag or of one; View; Live for the stream
    get_my_tally_handler/my_tally_query/    the caller's tally in a book
    get_round_handler/round_query/          a round, live or scored — Territory, Authors, Scores
    get_scores_handler/scores_query/        a book's scores, the scores around a round, the table (GetTable)
    get_table_handler/                      InternalService/GetTable, through scores_query
    (each */rpc_player_authors/           Authors, from player.v1.InternalService/GetAuthors)
    listen_for_events_handler/    the stream: a book's board and scores, the rounds asked, a heartbeat
      inprocess_feed/             Feed[K, V]: each key the streams follow, read once a period, sent when it changed
      log_reader/                 logs a live read that failed
    caller/                       the account on the context, or Unauthenticated
  subscribers/                    tile_taken_subscriber/  round_take_subscriber/  account_deleted_subscriber/
                                  log_subscriber/
  migrations/
```

## The commands

**`scoring.v1.ScoringCommand`**, one stream (`commands:scoring.v1.ScoringCommand`, group `handler`), its four cases
handled in order: a round closed, then its book, is always scored in that order. `scoring_command_handler.Dispatcher`
parses the ids and the times and hands each case to its use case, behind `log_command` (an Info line for each command
that changed something), behind `log_subscriber` (an Error line for each one that failed or was refused). **A command
refused (`books.ErrRefused`) is dropped** by `Handler`: delivered again, it would hold back every command after it on
the one stream. Any other failure is delivered again.

Every command is idempotent by its ids, checked by the use case on what the store found (`Find`), and once more by the
store's insert (`ON CONFLICT DO NOTHING`, which answers whether this call did it):

- **`OpenBook{book_id, starts_at}`**: a book kept is left as it is (its first start stays).
- **`OpenRound{round_id, book_id, times, starts_at}`**: a round kept is left as it is. `times` is 1 to
  `books.MaxTimes` (10), or the command is refused. A round whose book is not open (never opened, or closed) is refused
  (`open_round_usecase.ErrBookNotOpen`).
- **`CloseRound{round_id, ends_at}`**: a round never opened, or scored, is not scored. Else `close_round_usecase` reads
  the map once (`GetTerritories`), ranks it (`Round.Results`), keeps the results and records **`scoring.v1.RoundScored`**
  (round, book, end, each country's rank, points and share) in the outbox, in one transaction. A failure to read the map
  scores nothing and is tried again. Nothing freezes the map: a round closed late is scored from the map as it is then.
- **`CloseBook{book_id, ends_at, tiebreak_round_id}`**: a book never opened, or closed, is not ranked. Else
  `close_book_usecase` reads every round of the book scored, ranks them (`Book.Table`), closes the book and records
  **`scoring.v1.BookClosed`** (book, end, the table with each tiebreak's points), in one transaction. Scoring ranks by
  the tiebreak round it is given and never picks one: the gamemaster names its season's finale, or none when the season
  had no finale (see [The gamemaster](gamemaster.md#the-finale)).

**`scoring.v1.GrantPoints{grant_id, country, points, granted_at}`** is a stream of its own
(`commands:scoring.v1.GrantPoints`, group `handler`): missions sends one when a flag finishes its flag mission.
`grant_points_handler` parses it for `grant_points_usecase`, behind `log_command` and `log_subscriber`, and the same
`Handler` drops a refusal: an id that is not one, no country, no points or no time (`books.NewGrant`, `ErrEmptyGrant`).
A grant is kept once per `grant_id` (`grants`, `ON CONFLICT DO NOTHING`); it needs no book open when it comes, since
the books read the grants by time.

`scoring_command_handler/season_end_test.go` plays season 0's end: day 24 closed at 21:00, the battle ×3 from 21:00 to
23:00 as the tiebreak, then the book; the outbox holds the two `RoundScored` then `BookClosed`, and every command sent
again changes nothing.

## The rules

- **`Round.Results` ranks a round**: countries by the tiles held at its end, ties sharing a rank and its points, the top
  10 scoring 25, 18, 15, 12, 10, 8, 6, 4, 2, 1, times the round's `times`. A win is worth the same at 1% as at 30%, so
  an early lead cannot be banked. Until 2026-10-27 (season 0) a round was scored on each country's average share; those
  rounds keep their results.
- **`books.TableOf` ranks a book**: points, then rounds won, then the points of its tiebreak round (`ScoresOf`; none
  while the book is open, or when the close named none). Countries level on all three share the rank.
- **A grant counts in every book open at its time** (`Book.Table` keeps the grants it `Holds`): its points add to the
  country's, and it wins no round and gives no tiebreak points. A country with only grants is on the table. A grant
  told after its book closed, at a time before the end, shows in the book's scores but not in its `BookClosed`.
- **A take counts in every book open at its time** (`Book.Holds`: from its start, and before its end once closed), and
  in every round of such a book open at its time (`Round.Holds`: from its start, and before its end once scored). A take
  told after its round or book closed still counts when it happened before the end. Every round counts its takes and
  players, a day too.

## The standings

**A player's tally in a book is the tiles it took for its main flag**, the flag it took the most tiles for, so a player
cannot add up several flags. **A country's board is every player who took tiles for that country, ranked by those
tiles**, so a player is on the board of each flag it took for.

- **`standings.Tally` is the rule**: an account's tiles per flag in one book, and its main flag. `WithTake` adds one tile
  and makes the flag the main one when it has strictly more tiles than the main one; a tie keeps the flag that got there
  first.
- **Takes come from `planet.v1.TileTaken`** (`seasons-standings`), one tile per event, in ledger order.
  `record_take_usecase` reads every book and records the take in each one that holds it. The tally keeps the position
  and index past the last take it counted (`standings.TakeMark`: `counted_takes_below`, `counted_takes_below_index`),
  so a take told twice counts once, and each take of a spread or an enclose, which share one position, counts. A strike on a shield, a
  click the ban refuses and a revert publish nothing, so they change no score.
- **One transaction per take and book**, under `pg_advisory_xact_lock` on the account: read the account's rows for the
  book, apply `WithTake`, upsert the flag's row and move `main`.
- **Kept in `scoring.contributions`** (`book_id`, `account_id`, `country`, `tiles`, `main`, `counted_takes_below`,
  `counted_takes_below_index`;
  primary key `(book_id, account_id, country)`), with its own block, `scoring.database`. `main` marks one row per account
  and book (a partial unique index); `contributions_board` serves the whole map's board and
  `contributions_country_board` a country's.
- **`auth.v1.AccountDeleted`** (`seasons-standings-accounts`) deletes the account's rows in every book; in the rounds
  (`seasons-rounds-accounts`) it forgets the account's rows, and its flag keeps its tiles.
- The consumer groups keep their names from before scoring was renamed: a new name is a new group, which reads the
  stream again from its start.

## A round's numbers

`round_take_subscriber` hears `planet.v1.TileTaken` (`seasons-rounds`) and `count_take_usecase` counts each take in the
rounds that hold it: the tiles each flag's players took (`round_takes`) and each account's, by flag (`round_players`),
in one statement that moves each round's mark past the take's ledger position and index (`counted_takes_below`,
`counted_takes_below_index`).

## Kept

`books` (`book_id`, `starts_at`), `closed_books` (`ended_at`), `book_tiebreaks` (`round_id`: a book closed with no
tiebreak has no row), `grants` (`grant_id`, `country`, `points`, `granted_at`, by `granted_at`), `rounds` (`round_id`,
`book_id`, `times`, `starts_at`, `counted_takes_below`, `counted_takes_below_index`), `scored_rounds`
(`ended_at`), `round_results` (`rank`, `points`, `share`), `round_takes`, `round_players`, and the outbox
(`outbox_event`: `scoring.v1.RoundScored`, `scoring.v1.BookClosed`). An end is a table of its own, not a nullable
column: an open book or round has none. The rules stay in Go; SQL keeps what they said.

**Migration `20261028130000_books` converted production** (only season 0 had run; any other season refuses it):

- `contributions.season` became `book_id` = `'season-' || season`; `ended_seasons` became `closed_books`, with the
  finale round as the tiebreak. Season 0's book starts at the gamemaster's 2026-10-08 00:00 UTC.
- A day became `season-0/day-K`, K = 1 + the days between its end and 2026-10-08 21:00 UTC (day 1), starting a day
  before its end. A round of its own became its `own_rounds.entry_id`, or `season-N/own-n` (n by end) with none. The
  day in progress stayed open: the gamemaster's `OpenRound` for it changes nothing, and its `CloseRound` scores it.
- `rule_sets`, `rule_set_choice` and `own_rounds` were dropped. Pending outbox rows of the old kinds were deleted, and
  the enum's values renamed. The unmerged migration `20261028120000_seasons` was deleted: it never shipped.
- `migrations_test.go` seeds the old schema and checks the ids, the results, the contributions and the open day.

## The reads

The reads are queries (see [Reads are queries](queries.md)), by the ids the gamemaster's timeline names. A book or round
id that is not one is `InvalidArgument`.

- **`GetBoard(book_id, country_id)`** (`board_query`): the book's top 10, of every player by its main flag's tiles, or
  of every player who took tiles for `country_id` by those tiles: rank, account id, name, color, worn title, flag and
  tiles. **Only a signed-in player is ranked**: Go skips what `GetAuthors` answers as a guest or not at all; a tie
  shares the rank. Rows come 200 at a time from a keyset on `(tiles, account_id)`, one `GetAuthors` a page, until 10
  are named. A country that is not one is `InvalidArgument`. A GET, `public, max-age=15`.
- **`GetMyTally(book_id, country_id)`** (`my_tally_query`): the caller's main flag, tiles and rank among every player,
  and its tiles and rank on `country_id`'s board, and the title it wears; 0 for a guest's ranks. Behind
  `NewSessionInterceptor`, always enforcing, on the identity token (`scoring_session_checks{verdict}`).
- **`GetRound(round_id)`** (`round_query`), on the public and the internal listener: NotFound for a round never opened.
  While open: each country's share of the map now and the points it would score (`Round.Results`). Once scored: its
  results (the countries that scored), its end, and the book's scores before and after it (`scores_query.Around`:
  rounds ended before it, and through it; both with the grants made before its end, so they differ by the round
  alone). Always: the tiles each flag took during it (most first) and its top 10
  signed-in players by the tiles each took for the flag it took the most for, from the best 50 and one `GetAuthors`.
  A GET, `public, max-age=10`.
- **`GetScores(book_id)`** (`scores_query`): the book's countries with points, grants included, ranked by
  `Book.Table`. A GET, `public, max-age=10`. A book never opened has none.
- **`InternalService/GetTable(book_id)`**: the same table with each tiebreak's points, and `closed`; NotFound for a book
  never opened.
- **`ListenForEvents(book_id, country_id, round_ids)`**: the book's board and scores, each round asked (10 at most, a
  round asked twice followed once), then each one again once it changed, and a `heartbeat` every
  `httpServer.streamHeartbeat`. No `book_id` follows no board and no scores. A round not opened yet is sent once it
  opens. **One read per key, whatever the number of streams**: `inprocess_feed.Feed` keeps the last value of each key a
  stream follows, reads it at once when first followed and then every period on its own clock (boards 2s, scores and
  rounds 10s), up to 8 at once, sends it only when it changed, and forgets the key when its last stream closes. A slow
  stream skips to the newest value. A failed read is logged (`log_reader`) and read again next period. Nothing tells a
  feed that something moved: a read sees every change the same way. The feeds run under the pool's `cppg.CloseAfter`.

The contract suites (`BooksContractSuite`, `RoundsContractSuite`, `TakesContractSuite`, `standings.StoreContractSuite`)
run on the memory stores and on postgres; the queries are tested on postgres, seeded through the stores.

A question-mark box flies past the planet every so often; whoever catches it
gets one of five bonuses. Each box draws its kind from the rule set's `bonuses.kinds` (see [Rules](rules.md)), a weight
per kind — a kind's chance is its weight over the sum of the weights, so the
strong ones can be made rare (production runs 5 : 3 : 1 : 2 : 3):

- **`refill`** — a charge: fills the caller's click bank to full, when the
  caller chooses, and gives the network's bucket as much. Up to one bank, 60 clicks. See [What a refill does to the bucket](refill.md).
- **`spread_clicks`** — 1 to `bonuses.spread.maxPerBox` (4) clicks added to a pool
  of at most `bonuses.spread.clicks` (8). While the player switches spread on, each
  click spends one and also takes the tiles touching the one clicked. See [What a spread does to a click](spread.md).
- **`bomb`** — a charge: one bomb, kept until it is dropped. It
  clears a circle of `bonuses.bomb.rings` tile spacings around where it lands,
  whoever holds the tiles. See [What a bomb does](bomb.md).
- **`enclose_clicks`** — 1 to `bonuses.enclose.maxPerBox` (3) enclosures added to a
  stack of at most `bonuses.enclose.held` (3). While the player switches enclose on,
  a click that closes a shape of the caller's own tiles also takes the tiles
  inside it, at most `bonuses.enclose.maxTiles` (25), and spends one.
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
`AnswerQuiz` and `PublishEnclosed` are all keyed on it, and it holds every
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
