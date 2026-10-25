# Seasons (`internal/seasons/`)

**When a season ends, when its finale starts, and who leads it.** Planet never imports, calls or hears it: seasons asks planet what each country holds (`planet.v1.InternalService/GetTerritories`), never the other way.

```
internal/seasons/internal/
  calendar/                       Number, Season, Entry, Config (its Validate), Calendar (Current)
    usecases/get_season_usecase/  the calendar and the clock
  standings/                      AccountID, Country, Take (Season), Tally (WithTake); the Store port (RecordTake,
                                  DeleteAccount) and its contract suite
    postgres_contribution_store/  the Store over seasons.contributions
    inmemory_contribution_store/  the same port in a map, behind the testing tag
    usecases/record_take_usecase/  forget_account_usecase/
  rounds/                         Country, Round (Current, Results, Closed), Snapshot, Result, Closed (SeasonEnd),
                                  Score (ScoresOf), Placed (TableOf), SeasonEnd; the Store port (RecordSnapshot,
                                  Unclosed, Held, Number, Close, ClosedIn) and its contract suite; Close answers
                                  whether this call closed the round
    postgres_round_store/         the Store over seasons.rounds, round_holdings and round_results
    inmemory_round_store/         the same port in maps, behind the testing tag
    rpc_planet_territories/         the snapshot, from planet.v1.InternalService/GetTerritories
    usecases/close_round_usecase/  closes one round and records it, and the season's end with its finale, in the
                                  outbox, in one transaction
    usecases/take_snapshot_usecase/ closes the rounds that ended through close_round_usecase, then counts the round in
                                  progress; Runner
      log_take_snapshot/            logs each round closed, and a snapshot that failed
  seasonsv1controller/            SeasonService (a bag), the cache interceptor, the session interceptor
    get_season_handler/
    get_race_handler/             the race of the stream, once, through race_query — Query
    get_standings_handler/standings_query/   PostgresQuery: GetStandingsResponse from SQL, named, and the same top as
                                  a Board, for the live boards — Authors
      rpc_player_authors/         Authors, from player.v1.InternalService/GetAuthors, as player.v1.Author
    listen_for_events_handler/    the stream: the view's board, the race, each new one, a heartbeat — Boards, Races,
                                  CountryChecker
      inprocess_board_feed/       the boards streams follow, each read through standings_query; a Runner
        log_board_reader/         logs a board it could not read
      inprocess_race_feed/        the race every stream follows, read through race_query; a Runner
        log_race_reader/          logs a race it could not read
      race_query/                 PostgresQuery: the Race from SQL, ranked by rounds.Round.Results and rounds.TableOf
    get_my_season_handler/my_season_query/   PostgresQuery: GetMySeasonResponse from SQL, ranked — Authors
      rpc_player_authors/         the same adapter, for this query
    caller/                       the account on the context, or Unauthenticated
  subscribers/                    tile_taken_subscriber/  account_deleted_subscriber/  log_subscriber/
  migrations/
```

- **The calendar is the config**, `seasons.list`. An empty list is no season.
- **`Calendar.Current(now)` is the first season whose end is after now.** A season starts when the one before it ends; the first started before anything.
- **The boot is refused** when the numbers do not count up from 0, the ends do not go up, or a finale is not above 0 and shorter than its season.
- **`GetSeason` is a GET** (`NO_SIDE_EFFECTS`), answered `public, max-age=60`. No current season is an empty answer.

**A player's season score is the tiles it took this season for its main flag**, the flag it took the most tiles for, so a player cannot add up several flags. **A country's board is every player who took tiles for that country, ranked by those tiles**, so a player is on the board of each flag it took for, and a change of flag shows at the first take rather than when the main flag moves.

- **`standings.Tally` is the rule**: an account's tiles per flag in one season, and its main flag. `WithTake` adds one tile and makes the flag the main one when it has strictly more tiles than the main one; a tie keeps the flag that got there first. **The season of a take is the one whose interval holds `taken_at`** (`Take.Season`, over `Calendar.Current`); a take after the last season counts nothing. `tally_test.go` and `take_test.go` pin both.
- **Takes come from `planet.v1.TileTaken`** (`seasons-standings`), one tile per event, in ledger order. The tally keeps the position after the last take it counted (`Tally.TakesBelow`, `counted_takes_below` on each row), so a take told twice counts once. A strike on a shield, a click the ban refuses (it never reaches the ledger) and a revert publish nothing, so they change no score. Takes before this module had a database are not replayed: counting started the day it shipped.
- **One transaction per take**, under `pg_advisory_xact_lock` on the account: read the account's rows for the season, apply `WithTake`, upsert the flag's row and move `main`. The rule stays in Go; the column only keeps what it said.
- **Kept in `seasons.contributions`** (`season`, `account_id`, `country`, `tiles`, `main`; primary key `(season, account_id, country)`), with its own block, `seasons.database`, its own pool and migrations. `main` marks one row per account and season (a partial unique index). A partial index on it serves the whole map's board, and `contributions_country_board` (`season, country, tiles DESC, account_id`, every row) serves a country's.
- **`auth.v1.AccountDeleted`** (`seasons-standings-accounts`) deletes the account's rows in every season.
- **The reads are queries** (see [Reads are queries](queries.md)): the write model is `Take`, `Tally` and a `Store` that only records and deletes. Each query reads `seasons.contributions` itself and asks `GetAuthors` through its own `rpc_player_authors`. The module dials the internal listener once, at build, and both adapters share that one `player.v1` client; with no internal listener the boot is refused.
- **Only a signed-in player is ranked**: every one has a username (see [Player](player.md)), and a guest has none. SQL cannot tell them apart, so a query reads the rows and Go skips what `GetAuthors` answers as a guest, or does not answer at all. A tie shares the rank (1, 1, 3).
- **`GetStandings(country_id)`** (`standings_query`) is the current season's top 10 (`standings_query.Shown`): of every player by its main flag's tiles, or of every player who took tiles for `country_id` by those tiles. Each is rank, account id, name, color, worn title, the flag its tiles are for (the main flag, or `country_id`) and tiles. It needs no token, is a GET (`NO_SIDE_EFFECTS`) and answers `public, max-age=15`. The rows (the main rows, or the country's) come best first, then by account id, 200 at a time from a keyset on `(tiles, account_id)`, with one `GetAuthors` a page, until 10 are named. A country that is not one is `InvalidArgument` (`standings_query.ErrUnknownCountry`); no season is an empty answer. The web client no longer calls it: it stays for clients from before the stream.
- **`ListenForEvents(country_id)`** is the same top 10, live. It needs no token. The first event is the view's whole `board` (`Board`, the `Standing`s of `GetStandings`), each one after is the whole board again once it changed, and a `heartbeat` comes every `httpServer.streamHeartbeat`. A country that is not one is `InvalidArgument` before anything is followed. **One read per view, whatever the number of streams**: `inprocess_board_feed` keeps, for each view a stream follows, the last board, and forgets the view when its last stream closes.
  - **Nothing tells it a board moved.** A read sees every change the same way: a take, a deleted account, a rename, a new color or title, a guest who signed in, the season's end. The standings use cases know nothing of who reads their rows, so the subscribers call them straight.
  - **When it is read** (`Feed.Refresh`, every 250ms): a view at once when it is first followed, then every 2s (`Every`) on its own clock, so a new view is no reason to read the others. A board equal to the last one is not sent. Up to 8 views are read at once, each through `standings_query.Board`, one query and one `GetAuthors` a page, as `GetStandings`. So a board is at most 2s late, and each followed view costs that read every 2s whether anything moved or not: at most one view per country, and the whole map's.
  - **A slow stream skips to the newest board**: each holds one, and a new one replaces the one it did not take. A board is whole, so nothing is lost.
  - **A failed read is tried again on the next read**, 2s later, and logged (`log_board_reader`), but not when the shutdown cut it. The feed runs under the pool's `cppg.CloseAfter`, so it stops before the pool closes.
- **`GetMySeason(country_id)`** (`my_season_query`) is the caller's main flag, its tiles for it and its rank among every player, and its tiles for `country_id` and its rank on that country's board (`country_tiles`, `country_rank`), and the title it wears, from the `GetAuthors` that tells it is not a guest. A rank is 0 and there is no title for a guest, and every number is 0 for an account with no take this season; the country's are 0 with no `country_id`, or no take for it. A country that is not one is `InvalidArgument` (`my_season_query.ErrUnknownCountry`). It sits behind `seasonsv1controller.NewSessionInterceptor`, always enforcing, on the key `auth` hands over (`seasons_session_checks{verdict}`), and takes the identity token (`cpconnect.Identified`): it only reads. It counts the ranked players above the caller: SQL reads the main rows with more tiles than the caller's main flag, and the country's rows with more than the caller's for it, each 500 at a time and both at once, and costs one `GetAuthors` a page.
**A country's season score is the points of the rounds it won places in**, counted from the ground it held: the
standings rank players, the rounds rank countries.

- **A round is a day, and the Final Battle is the last one** (`rounds.Current`). A day round ends at the time of day
  the finale starts (21:00 UTC in production, evening in Europe), so every day ends while people play. The finale is
  a round of its own, from its start to the season's end. A round is kept by its season and its end
  (`Round{Season, EndsAt, Finale}`); its number is the read side's, from the rounds counted before it, so a season
  with no start (Season 0) still says "day 1".
- **A snapshot a minute** (`take_snapshot_usecase`, `seasons.snapshot.interval`): ask planet what each country holds, then
  add each count to the round in progress. **A country's result is its average share**: the sum of its counts over the
  round, over the snapshots taken. A snapshot missed (a restart, planet down) is one sample fewer for every country, so
  it biases nobody. Counting started the day this shipped; nothing is replayed.
- **`Round.Results` is the rule**: countries ranked by the ground held, ties sharing a rank and its points, and the top
  10 scoring 25, 18, 15, 12, 10, 8, 6, 4, 2, 1. The finale scores 3 times that. A day's win is worth the same at 1% as
  at 30%, so an early lead cannot be banked.
- **A round closes on the first snapshot after it ended** (`Store.Unclosed`): its results are written once, in one
  transaction that marks it closed, and a second close changes nothing. Closing does not need a current season, so the
  finale closes after the last season ends. `log_take_snapshot` logs each round closed.
- **Each round closed is told to the other modules** as `seasons.v1.RoundClosed`: its season, its number
  (`Store.Number`, the rounds of its season counted before it, as the read side counts them), whether it is the finale,
  its end and its results. The use case publishes it itself, through its `Publisher` port, once `Close` returned: a
  round that failed to close is not published, and is closed and published by the next snapshot. The chat announces
  it — see [Announcements](chat.md#announcements).
- **The finale closing ends the season, and that is told too** as `seasons.v1.SeasonClosed`: its season, its end and
  its whole table, the champion first, for the chat and for the titles. `Closed.SeasonEnd` is the rule: only a finale
  ends its season, and its table is `TableOf(ScoresOf(...))` of every round the season closed, the finale last.
  `ScoresOf` adds the results up as `race_query`'s SQL does, and `race_query`'s test pins the two against each other.
  **Everything is read before the round is closed** (`Store.ClosedIn`, the rounds of a season closed so far, in
  order, with their results), so a failed read leaves the finale to close, and the next snapshot closes it and tells
  both events once. A closed round is never unclosed, so nothing tells a season twice; the chat keys its line by the
  season anyway.
- **`rounds.TableOf` ranks the season**: points, then rounds won, then the finale's points. Countries level on all
  three share the rank. The champion is the country ranked 1st; the titles at the season's end read the same table,
  from `SeasonClosed`.
- **Kept in `seasons.rounds`** (`season`, `ends_at`, `finale`, `samples`, `map_tiles`, `closed`), **`round_holdings`**
  (the sum of each country's counts) and **`round_results`** (`rank`, `points`), migration `20261009120000_rounds`. The
  rule stays in Go: SQL keeps the sums and what `Results` said.
- **The race is live on the season stream** (`SeasonEvent.race`), to every stream whatever its view. `race_query` reads
  the round in progress (its number, its end, each country's rank, average share and the points it would score if it
  ended now) and the season's scores, best first, of every country with points. **It also carries the round closed
  last** (`closed`), in any season, so the finale still shows once its season is over: its number, its end, the
  countries that scored in it with their average share, and its season's table before and after it, both ranked by
  `rounds.TableOf`. A client plays the cutoff from it. **One read for every stream**:
  `inprocess_race_feed` reads it when the first stream opens and then every 10s (`Every`) while a stream follows it, and
  sends it only when it changed. **Nothing tells it a snapshot was taken**: the race only changes once a minute, so
  reading on its own clock is at most 10s late, and the snapshot use case knows nothing of who reads its rows. It
  forgets the race when the last stream closes. A failed read is logged (`log_race_reader`) and the next tick reads
  again. No season is a race with no round and no scores, and no round closed yet is a race with no `closed`.
- **`GetRace` is the same race, once**, for a reader that follows no stream: the clips job, which tells the day closed
  last (see [The clips job](clips-job.md)). It needs no token, is a GET (`NO_SIDE_EFFECTS`)
  and answers `public, max-age=10`. It reads through `race_query` itself, logged by `log_race_reader` as the feed is,
  and never through the feed: a feed with no stream holds no race.
- `rounds.StoreContractSuite` runs on `inmemory_round_store` and on postgres. `race_query`'s suite ranks a
  round from its snapshots, `take_snapshot_usecase` closes the finale and tells the season's table, and chat's
  `season_closed_subscriber` names the champion once.
- `standings.StoreContractSuite` runs on `inmemory_contribution_store` and on postgres. It reads what a store kept through a `TallyOf` hook each adapter's test fills, since the write side reads nothing back. The use cases are tested over the in-memory one; the queries on postgres, seeded through `postgres_contribution_store`.


A question-mark box flies past the planet every so often; whoever catches it
gets one of five bonuses. Each box draws its kind from `bonus.kinds`, a weight
per kind — a kind's chance is its weight over the sum of the weights, so the
strong ones can be made rare (production runs 5 : 3 : 1 : 2 : 3):

- **`refill`** — a charge: fills the caller's click bank to full, when the
  caller chooses, and gives the network's bucket as much. Up to one bank, 60 clicks. See [What a refill does to the bucket](refill.md).
- **`spread_clicks`** — 1 to `bonus.spread.maxPerBox` (4) clicks added to a pool
  of at most `bonus.spread.clicks` (8). While the player switches spread on, each
  click spends one and also takes the tiles touching the one clicked. See [What a spread does to a click](spread.md).
- **`bomb`** — a charge: one bomb, kept until it is dropped. It
  clears a circle of `bonus.bomb.rings` tile spacings around where it lands,
  whoever holds the tiles. See [What a bomb does](bomb.md).
- **`enclose_clicks`** — 1 to `bonus.enclose.maxPerBox` (3) enclosures added to a
  stack of at most `bonus.enclose.held` (3). While the player switches enclose on,
  a click that closes a shape of the caller's own tiles also takes the tiles
  inside it, at most `bonus.enclose.maxTiles` (25), and spends one.
  See [What an enclose does to a click](enclose.md).
- **`shields`** — `bonus.shield.minPerBox` (5) to `bonus.shield.maxPerBox` (20)
  shields added to a stack of at most `bonus.shield.held` (30). Each is placed on a tile the player's flag
  holds and takes one foreign click there. See [Shields](shields.md).

Every kind is a **charge**, worth about one bank, rather than a timer — see
[Charges](charges.md). There used to be a `triple_clicks`
that multiplied the refill for two minutes. With a bank of 60 it was worth "some
clicks, maybe": nothing on a full bank, and a different amount for every
country's pace. The refill is the same good, clicks, with the moment chosen by
the player.

Boxes are always on: there is no switch.

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

**`maxChargesPerHour` bounds what a caller can be granted.** Nothing here is a
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
