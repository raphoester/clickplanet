# Scoring (`internal/scoring/`)

**The season's scorekeeper: when a season and each day end, how points are counted, and who leads.** Planet never imports, calls or hears it: scoring asks planet what each country holds (`planet.v1.InternalService/GetTerritories`), at each round's end and for the live race, never the other way. **The gamemaster commands it** a rule set, as it does planet and the antibot (`scoring.v1.PlayRuleSet`, see [The season rules](season-rules.md)): a battle's set makes the battle's window a round of its own, and the Final Battle is the one that ends the season.

```
internal/scoring/internal/
  calendar/                       Number, Season (LastDayEndsAt), Entry, Config (its Validate), Calendar (Current, Ended,
                                  CutoffAfter)
    usecases/get_season_usecase/  the calendar and the clock
  standings/                      AccountID, Country, Take (Season), Tally (WithTake); the Store port (RecordTake,
                                  DeleteAccount) and its contract suite
    postgres_contribution_store/  the Store over scoring.contributions
    inmemory_contribution_store/  the same port in a map, behind the testing tag
    usecases/record_take_usecase/  forget_account_usecase/
  rounds/                         Country, Round (Day, Own, Results, Closed), Snapshot, Result, Closed, Score (ScoresOf),
                                  Placed (TableOf), SeasonEnd (SeasonEndOf), Take; the Store port (Open, Unclosed,
                                  FirstEnd, Number, Close, ClosedIn, End, Ended) and the Takes port (Count,
                                  ForgetAccount), each with its contract suite; Close and End answer whether this call
                                  did it
    postgres_round_store/         both ports over scoring.rounds, round_results, round_takes, round_players and
                                  ended_seasons
    inmemory_round_store/         the same ports in maps, behind the testing tag
    rpc_planet_territories/         the map now, from planet.v1.InternalService/GetTerritories
    usecases/close_round_usecase/  closes one round from the map at its end and records it in the outbox, in one
                                  transaction
    usecases/close_season_usecase/ ends a season from every round it closed and records its table, in one transaction
    usecases/close_ended_rounds_usecase/ closes the rounds that ended, ends the seasons that ended, then opens the day
                                  in progress; Runner (Wake)
      log_close_ended_rounds/       logs each round closed, at Warn when late, each season ended, and a failure
    usecases/open_own_round_usecase/ opens the round of its own of an entry's window, and wakes the runner
    usecases/count_take_usecase/  forget_account_usecase/   a round of its own's numbers
  rules/                          SetName, Set, OwnRound, Snapshot (Restore), Window, Choice; the Store and Choices
                                  ports, their fakes and contract suites — see [The season rules](season-rules.md)
    postgres_rule_set_store/  postgres_choice_store/
    usecases/play_rules_usecase/ (log_play_rules/)  save_rule_set_usecase/ (audit_save_rule_set/)
  commands/play_rule_set_handler/ scoring.v1.PlayRuleSet, from the gamemaster
  scoringv1controller/            SeasonService (a bag), the cache interceptor, the session interceptor
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
      race_query/                 PostgresQuery: the Race from SQL and the map, ranked by rounds.Round.Results and
                                  rounds.TableOf — Territory, rounds/rpc_planet_territories
    get_my_season_handler/my_season_query/   PostgresQuery: GetMySeasonResponse from SQL, ranked — Authors
      rpc_player_authors/         the same adapter, for this query
    get_hall_of_fame_handler/hall_of_fame_query/   PostgresQuery: each season ended, its table by rounds.TableOf and
                                  its top players, named — Authors
      rpc_player_authors/         the same adapter, for this query
    caller/                       the account on the context, or Unauthenticated
  subscribers/                    tile_taken_subscriber/  round_take_subscriber/  account_deleted_subscriber/
                                  log_subscriber/
  migrations/
```

- **The calendar is the config**: `scoring.list`, each season's number and end, and `scoring.dailyCutoff`, the time
  of day (UTC) every day ends at (21:00 in production, evening in Europe, so every day ends while people play). An
  empty list is no season. **The calendar holds no Final Battle**: since 2026-10-27 the Final Battle is a schedule
  entry of the gamemaster (see [The gamemaster module](gamemaster.md)), and `finale` is no key of a season.
- **`Calendar.Current(now)` is the first season whose end is after now.** A season starts when the one before it ends; the first started before anything. `Season.LastDayEndsAt` is its last cutoff, at or before its end.
- **The boot is refused** when the numbers do not count up from 0, the ends do not go up, the cutoff is not a time of
  day, or a season holds no end of a day.
- **`GetSeason` is a GET** (`NO_SIDE_EFFECTS`), answered `public, max-age=60`: the number, the end, and
  `last_day_ends_at`. `finale_starts_at` says the same as `last_day_ends_at`, for the clients from before it: the
  Final Battle followed the last day, and does in Season 0. No current season is an empty answer.

**A player's season score is the tiles it took this season for its main flag**, the flag it took the most tiles for, so a player cannot add up several flags. **A country's board is every player who took tiles for that country, ranked by those tiles**, so a player is on the board of each flag it took for, and a change of flag shows at the first take rather than when the main flag moves.

- **`standings.Tally` is the rule**: an account's tiles per flag in one season, and its main flag. `WithTake` adds one tile and makes the flag the main one when it has strictly more tiles than the main one; a tie keeps the flag that got there first. **The season of a take is the one whose interval holds `taken_at`** (`Take.Season`, over `Calendar.Current`); a take after the last season counts nothing. `tally_test.go` and `take_test.go` pin both.
- **Takes come from `planet.v1.TileTaken`** (`seasons-standings`), one tile per event, in ledger order. The tally keeps the position after the last take it counted (`Tally.TakesBelow`, `counted_takes_below` on each row), so a take told twice counts once. A strike on a shield, a click the ban refuses (it never reaches the ledger) and a revert publish nothing, so they change no score. Takes before this module had a database are not replayed: counting started the day it shipped.
- **One transaction per take**, under `pg_advisory_xact_lock` on the account: read the account's rows for the season, apply `WithTake`, upsert the flag's row and move `main`. The rule stays in Go; the column only keeps what it said.
- **Kept in `scoring.contributions`** (`season`, `account_id`, `country`, `tiles`, `main`; primary key `(season, account_id, country)`), with its own block, `scoring.database`, its own pool and migrations. `main` marks one row per account and season (a partial unique index). A partial index on it serves the whole map's board, and `contributions_country_board` (`season, country, tiles DESC, account_id`, every row) serves a country's.
- **`auth.v1.AccountDeleted`** (`seasons-standings-accounts`) deletes the account's rows in every season.
- **The reads are queries** (see [Reads are queries](queries.md)): the write model is `Take`, `Tally` and a `Store` that only records and deletes. Each query reads `scoring.contributions` itself and asks `GetAuthors` through its own `rpc_player_authors`. The module dials the internal listener once, at build, and both adapters share that one `player.v1` client; with no internal listener the boot is refused.
- **Only a signed-in player is ranked**: every one has a username (see [Player](player.md)), and a guest has none. SQL cannot tell them apart, so a query reads the rows and Go skips what `GetAuthors` answers as a guest, or does not answer at all. A tie shares the rank (1, 1, 3).
- **`GetStandings(country_id)`** (`standings_query`) is the current season's top 10 (`standings_query.Shown`): of every player by its main flag's tiles, or of every player who took tiles for `country_id` by those tiles. Each is rank, account id, name, color, worn title, the flag its tiles are for (the main flag, or `country_id`) and tiles. It needs no token, is a GET (`NO_SIDE_EFFECTS`) and answers `public, max-age=15`. The rows (the main rows, or the country's) come best first, then by account id, 200 at a time from a keyset on `(tiles, account_id)`, with one `GetAuthors` a page, until 10 are named. A country that is not one is `InvalidArgument` (`standings_query.ErrUnknownCountry`); no season is an empty answer. The web client no longer calls it: it stays for clients from before the stream.
- **`ListenForEvents(country_id)`** is the same top 10, live. It needs no token. The first event is the view's whole `board` (`Board`, the `Standing`s of `GetStandings`), each one after is the whole board again once it changed, and a `heartbeat` comes every `httpServer.streamHeartbeat`. A country that is not one is `InvalidArgument` before anything is followed. **One read per view, whatever the number of streams**: `inprocess_board_feed` keeps, for each view a stream follows, the last board, and forgets the view when its last stream closes.
  - **Nothing tells it a board moved.** A read sees every change the same way: a take, a deleted account, a rename, a new color or title, a guest who signed in, the season's end. The standings use cases know nothing of who reads their rows, so the subscribers call them straight.
  - **When it is read** (`Feed.Refresh`, every 250ms): a view at once when it is first followed, then every 2s (`Every`) on its own clock, so a new view is no reason to read the others. A board equal to the last one is not sent. Up to 8 views are read at once, each through `standings_query.Board`, one query and one `GetAuthors` a page, as `GetStandings`. So a board is at most 2s late, and each followed view costs that read every 2s whether anything moved or not: at most one view per country, and the whole map's.
  - **A slow stream skips to the newest board**: each holds one, and a new one replaces the one it did not take. A board is whole, so nothing is lost.
  - **A failed read is tried again on the next read**, 2s later, and logged (`log_board_reader`), but not when the shutdown cut it. The feed runs under the pool's `cppg.CloseAfter`, so it stops before the pool closes.
- **`GetMySeason(country_id)`** (`my_season_query`) is the caller's main flag, its tiles for it and its rank among every player, and its tiles for `country_id` and its rank on that country's board (`country_tiles`, `country_rank`), and the title it wears, from the `GetAuthors` that tells it is not a guest. A rank is 0 and there is no title for a guest, and every number is 0 for an account with no take this season; the country's are 0 with no `country_id`, or no take for it. A country that is not one is `InvalidArgument` (`my_season_query.ErrUnknownCountry`). It sits behind `scoringv1controller.NewSessionInterceptor`, always enforcing, on the key `auth` hands over (`scoring_session_checks{verdict}`), and takes the identity token (`cpconnect.Identified`): it only reads. It counts the ranked players above the caller: SQL reads the main rows with more tiles than the caller's main flag, and the country's rows with more than the caller's for it, each 500 at a time and both at once, and costs one `GetAuthors` a page.
**A country's season score is the points of the rounds it won places in**, counted from the ground it held at
each round's end: the standings rank players, the rounds rank countries.

- **A round is a day, or a round of its own.** A day ends at each cutoff (`rounds.Day`); none runs from the season's
  last cutoff to its end. **A round of its own is the window of a schedule entry whose scoring set asks for one**
  (`rounds.Own`, opened by `open_own_round_usecase` when scoring plays the set, see
  [The season rules](season-rules.md)): of the season its window starts in, cut at that season's end, and worth its
  set's times a day's points. A day and a round of its own run at once and are scored apart, each from the map at its
  end. A round is kept by its season, its end and whether it is of its own (`Round{Season, EndsAt, Own}`, with
  `Finale`, `StartsAt` and `Times`); a day and a round of its own may end at one time. Its number is the read side's,
  among its season's rounds of its kind (day 5, the first round of its own), so a season with no start (Season 0)
  still says "day 1".
- **The finale is the round that ends the season**: the round of its own whose window runs to the season's end (the
  Final Battle), or a day when the season ends at a cutoff. A season with no such round has none.
- **A round is scored from the map at the moment it ends** (`close_ended_rounds_usecase`): one `GetTerritories` at
  the cutoff. Until 2026-10-27 (Season 0) it was scored on each country's average share; those rounds keep their
  results.
- **`Round.Results` is the rule**: countries ranked by the tiles held at the end, ties sharing a rank and its points,
  and the top 10 scoring 25, 18, 15, 12, 10, 8, 6, 4, 2, 1, times the round's `Times` (1 for a day). A day's win is
  worth the same at 1% as at 30%, so an early lead cannot be banked.
- **The day in progress is opened** (`Store.Open`); **the runner waits until the first end of a round or of the
  season** (`Store.FirstEnd`, the season's end), reads the map once and closes every round that ended
  (`Store.Unclosed`, a day before a round of its own that ends with it), each in one transaction that marks it closed
  and records its event; a second close changes nothing. A round of its own opened wakes the runner
  (`Runner.Wake`), since it may end before the round it waits for. A failure is tried again 5s later and logged
  (`log_close_ended_rounds`, which logs each round closed).
- **A round that ended while the server was down is closed at boot, from the map as read then**, logged at Warn
  (`late` over a second). Nothing freezes the map: the owner accepted the risk. A round never opened is not played,
  and a round of its own heard of once it ended is never opened.
- **Each round closed is told to the other modules** as `scoring.v1.RoundClosed`: its season, its number, whether it
  is the finale, whether it is of its own and its times, its end and its results. The use case records it in the
  outbox in the transaction that closes the round: a round that failed to close is not published, and is closed and
  published by the next try. The chat announces it — see [Announcements](chat.md#announcements).
- **A season ends at its end, whether a round ends with it or none does** (`close_season_usecase`), once every round
  that ended is closed: so the Final Battle's `RoundClosed` comes first, then `scoring.v1.SeasonClosed`, its season,
  its end and its whole table, the champion first, for the chat and for the titles. `rounds.SeasonEndOf` is the rule:
  its table is `TableOf(ScoresOf(...))` of every round the season closed, the rounds of their own among them.
  `ScoresOf` adds the results up as `race_query`'s SQL does, and `race_query`'s test pins the two against each other.
  **Everything is read before the season is ended** (`Store.ClosedIn`), so a failed read leaves it to end at the next
  try. **`ended_seasons` keeps each season ended** (`Store.End`, one row, in the transaction that records the event):
  nothing tells a season twice; the chat keys its line by the season anyway.
- **`rounds.TableOf` ranks the season**: points, then rounds won, then the finale's points. Countries level on all
  three share the rank. The champion is the country ranked 1st; the Hall of Fame reads the same table, and the titles
  at the season's end read it from `SeasonClosed`.
- **A round of its own counts its numbers** for the client's brief: the tiles each flag's players took during its
  window (`round_takes`), and each account's, by flag (`round_players`). `round_take_subscriber` hears
  `planet.v1.TileTaken` (`seasons-rounds`) and `count_take_usecase` counts each take in every round of its own whose
  window holds its time, in one statement that moves the round's mark past the take's ledger position
  (`counted_takes_below`), so a take told twice counts once. `auth.v1.AccountDeleted` (`seasons-rounds-accounts`)
  forgets the account's rows; its flag keeps its tiles. A day counts no numbers.
- **Kept in `scoring.rounds`** (`season`, `ends_at`, `own`, `finale`, `starts_at`, `times`, `closed`,
  `counted_takes_below`; key `(season, ends_at, own)`), **`round_results`** (`rank`, `points`, `share` held at the
  end; for a round closed before 2026-10-27, its average), **`round_takes`**, **`round_players`** and
  **`ended_seasons`**, migration `20261027210000_rounds_of_their_own`. The rule stays in Go: SQL keeps what `Results`
  said.
- **The race is live on the season stream** (`SeasonEvent.race`), to every stream whatever its view. `race_query`
  reads **the round in progress that ends first**, a day or a round of its own, which wins a tie (its number, its end,
  whether it is of its own and its times, each country's rank, share of the map now and the points it would score if
  the round ended now, from `GetTerritories`; none from the last cutoff to the season's end when no round of its own
  runs), and the season's scores, best first, of every country with points. **It also carries the round closed
  last** (`closed`), in any season, so the finale still shows once its season is over: its number, its end, the
  countries that scored in it with their share, and its season's table before and after it, both ranked by
  `rounds.TableOf`; and, for a round of its own, its start, its times, the tiles each flag took (`takes`, most first)
  and its top 10 players (`players`, by the tiles each took for the flag it took the most for, named as the boards
  name them, guests skipped, from the best 50 and one `GetAuthors`). A client plays the cutoff, and a battle's brief,
  from it. **One read for every stream**:
  `inprocess_race_feed` reads it when the first stream opens and then every 10s (`Every`) while a stream follows it, and
  sends it only when it changed. **Nothing tells it a round closed**: reading on its own clock is at most 10s late, and
  the close use case knows nothing of who reads its rows. It forgets the race when the last stream closes. A failed
  read is logged (`log_race_reader`) and the next tick reads again. No season is a race with no round and no scores,
  and no round closed yet is a race with no `closed`.
- **`GetRace` is the same race, once**, for a reader that follows no stream: the clips job, which tells the day closed
  last (see [The clips job](clips-job.md)). It needs no token, is a GET (`NO_SIDE_EFFECTS`)
  and answers `public, max-age=10`. It reads through `race_query` itself, logged by `log_race_reader` as the feed is,
  and never through the feed: a feed with no stream holds no race.
- **`GetHallOfFame` is every season that ended**, newest first: a season is in it once it is in `ended_seasons`,
  whether a round ended with it or none did, so before Season 0 ends the answer is empty. Each is its number, its end,
  its final table (`rounds.TableOf` over the results of its closed rounds, days and rounds of their own, every country
  with points) and its top 10 players by their main flag's tiles, signed-in only, named and ranked as `GetStandings`
  does. Raw facts: the client crowns the countries ranked 1. It needs no token, is a GET (`NO_SIDE_EFFECTS`) and
  answers `public, max-age=60`. `hall_of_fame_query` reads `scoring.ended_seasons`, `rounds`, `round_results` and
  `contributions` itself; its tests seed through `postgres_round_store` and `postgres_contribution_store`.
- `rounds.StoreContractSuite` and `rounds.TakesContractSuite` run on `inmemory_round_store` and on postgres.
  `race_query`'s suite ranks a round from a fake planet's map, picks the round that ends first and names a round of its
  own's players; `close_ended_rounds_usecase` closes a round of its own at ×3, ends the season with and without a round
  at its end, and tells its table once; chat's `season_closed_subscriber` names the champion once.
- `standings.StoreContractSuite` runs on `inmemory_contribution_store` and on postgres. It reads what a store kept through a `TallyOf` hook each adapter's test fills, since the write side reads nothing back. The use cases are tested over the in-memory one; the queries on postgres, seeded through `postgres_contribution_store`.


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
