# The gamemaster module (`internal/gamemaster/`)

**The game's meta: which rules play together, and when; every moment of the game, and what scoring counts in it.**
Planet keeps its game rules in its sets (see [Rules](rules.md)), the antibot its bounds in its own (see [The antibot's
rules](antibot-rules.md)). The gamemaster module never holds a rule: it picks which set each module plays, by name, so
a set of planet and one of the antibot play as one. No module picks the set it plays: the gamemaster's schedule does,
and no operator call changes it at once (the user's rule, 2026-10-26: "there needs to be a schedule of overrides with
a hierarchy … they have to use schedules and priority orders, always").

**The gamemaster owns time and the game's words** (the owner, 2026-10-28): the seasons, the days, the battles and the
finale. Scoring is a points accountant that knows two words, a book and a round, by ids the gamemaster gives it (see
[Scoring](scoring.md)). The gamemaster commands it, in its words, when a book or a round opens and closes; scoring tells
what it scored; the gamemaster tells the other modules what that was, in the game's words, with ids and no numbers.

**A battle is not bound to a season** (the user, 2026-10-26): an operator may launch an unplanned battle tonight
without touching the seasons. A battle is an entry of kind battle with its points' `times`; while a season runs, its
window is a round of that season's book. **The Final Battle is the season's finale, declared on the season** (the
owner, 2026-10-28; see [The finale](#the-finale)): its window follows the season's end and can never drift, and no
battle is ever the finale because its end matches the season's. It is where a game stop and a map reset go next. The
module's name was `rules` for a day; the user renamed it, since it organises the rules and holds none. PR #475 tried a
loopback call from scoring, a baseline table and an override button; this replaced it.

```
internal/gamemaster/
  module.go                         Config (dailyCutoff, database, redis), NewModule
  internal/
    migrations/                     schedule_entries, entry_kind, entry_description, seasons (season 0 seeded),
                                    entry_times, scored_moments (scored_rounds, opened_rounds, closed_books, outbox),
                                    season_finales (season 0's Final Battle seeded)
    schedule/                       EntryID, SetName, Kind, Description, Times, Priority, Draft, Entry (NewEntry, Edited,
                                    Changeable), Schedule (New, With, TurnAt, SinceAt, BattlesAfter), Finale, Turn; the
                                    Store, Keeper and IDProvider ports, Playable; the store's contract suite; FakeKeeper and
                                    FakeIDs (testing tag)
      inmemory_entry_store/         the Store in a map, behind the testing tag
      postgres_entry_store/         the Store over gamemaster.schedule_entries
      uuid_id_provider/             a uuidv7 for each entry
      rpc_planet_rule_sets/         Keeper, from planet.v1.InternalService/GetRuleSetNames
      rpc_antibot_rule_sets/        Keeper, from antibot.v1.InternalService/GetRuleSetNames
      redis_rule_set_commands/      planet.v1 and antibot.v1 PlayRuleSet, on their command streams
      usecases/add_entry_usecase/     audit_add_entry/
      usecases/edit_entry_usecase/    audit_edit_entry/
      usecases/delete_entry_usecase/  audit_delete_entry/
      usecases/play_schedule_usecase/ the turn at now, commanded when it changes; Runner
        log_play_schedule/          logs each turn told, and a failure
    season/                         Number, Origin, Draft, Season, Window, Calendar (Added, Edited, Deletable, With,
                                    Windows, RunningAt, NextAfter, Finales); Finale, Window.FinaleStartsAt;
                                    Cutoff (Is), DayNumber, Day, Window.Days; the Store port and its contract suite
      inmemory_season_store/        the Store in a map, behind the testing tag
      postgres_season_store/        the Store over gamemaster.seasons and season_finales
      redis_map_commands/           planet.v1.ResetMap, on planet's command stream
      usecases/play_resets_usecase/ a season after season 0 starts on a blank map, told once per process; Runner
        log_play_resets/            logs each reset told, and a failure
      usecases/add_season_usecase/     audit_add_season/
      usecases/edit_season_usecase/    audit_edit_season/
      usecases/delete_season_usecase/  audit_delete_season/
    moments/                        BookID, RoundID (BookOf, DayRoundOf, FinaleRoundOf, BattleRoundOf, SeasonOf, NamedOf),
                                    Battle, Book (Tiebreak, FinaleBattle), Plan (DueAt, BookAt, DayAt, DaysOf, BattleOf,
                                    FinaleAt, FinalesAfter, RoundEnded, BookClosed, Clashes), Planner (Plan, With),
                                    Command, Action,
                                    Key, Moment (DayStarted, DayEnded, BattleEnded, SeasonEnded), ToldDay; the Store and
                                    Events ports, the store's contract suite
      inmemory_moment_store/        the Store in maps, behind the testing tag
      postgres_moment_store/        the Store over gamemaster.scored_rounds, closed_books, opened_rounds and told_days
      momentmessage/                DebriefOf and EventOf: a moment in gamemaster.v1's words
      outbox_moment_events/         Events, into the gamemaster outbox
      redis_scoring_commands/       one scoring.v1.ScoringCommand per command, on scoring's command stream
      round_scored_subscriber/      hears scoring.v1.RoundScored (gamemaster-rounds)
      book_closed_subscriber/       hears scoring.v1.BookClosed (gamemaster-books)
      usecases/play_scoring_usecase/ each command whose time has come, once per process; Runner
        log_play_scoring/           logs each command sent, and a failure
      usecases/tell_day_usecase/    the day that runs, told once and again when its end moved; Runner
        log_tell_day/               logs each day told, and a failure
      usecases/hear_round_usecase/  keeps a scored round, and records its day's or battle's end once; log_hear_round/
      usecases/hear_book_usecase/   keeps a closed book, and records its season's end once; log_hear_book/
    finalnight/                     a test alone: season 0's last day and its Final Battle, second by second
    gamemasterv1controller/              AdminService and GamemasterService (bags), NewCacheInterceptor
      get_schedule_handler/schedule_query/   PostgresQuery: every entry, every finale, and what plays now
      add_entry_handler/  edit_entry_handler/  delete_entry_handler/
      get_seasons_handler/seasons_query/   PostgresQuery: every season, by number, planned and with its window
      add_season_handler/  edit_season_handler/  delete_season_handler/
      get_turn_handler/turn_query/  PostgresQuery: the turn now, the battles to come, the seasons, the day and the last
                                    debrief, in the players' words
      listen_for_events_handler/    the stream: the turn, the timeline and the debrief, then each change, and heartbeats
        inprocess_turn_feed/        reads them each second while a stream follows them; log_turn_reader
      schedulemessage/              DraftOf, KindOf, and Refusal: the code of each refusal
      seasonmessage/                DraftOf, ScheduledOf, SeasonOf and Refusal, for the seasons
```

## An entry

**An entry is a rule set, a kind, a description, a start, an end, a priority and its points' times**, kept in
`gamemaster.schedule_entries`. Each has an end: one that never ended could never be stopped.

**The times are how many times a day's points a battle scores** (`schedule.Times`, column `times`): 1 to 10 for a
battle, 0 for a plain or a Halloween entry, which scores nothing; anything else is refused with the draft's other reasons
(`ErrInvalidEntry`), and postgres checks it too. Migration `20261028130000_entry_times` gave every battle kept 3: the
only one planned was the Final Battle, worth three days.

**The kind is what the entry is to the players** (`schedule.Kind`, the postgres enum `entry_kind`): `battle`, which
the game client gives a war-like look, `halloween`, a plain entry the client dresses for Halloween (migration
`20261031130000_halloween_kind`, and `20261031150000_halloween_times` for postgres's check on its times: a kind added
to the enum is a kind added to that check too), or `plain`, a change of the rules with no look for them (a quieter antibot bound for a
day). Only a battle is a round, or on the timeline. The set's name is the operator's word and never crosses to the players; the kind is the published
language the client reads (see [The game client](#the-game-client)). A kind nobody knows is refused
(`ErrUnknownKind`). More kinds will come: each is a value of the enum, in the proto and in postgres, and a word the
client interprets.

**The description is the entry's words for the players** (`schedule.Description`): what its rules are, in plain
text, which the client shows behind the battle chip ("Triple points. A bonus box every minute."). The client cannot
know a set's values, so the operator writes them. At most `MaxDescription` (300) characters, counted in runes; a
longer one is refused with `ErrLongDescription`, among the draft's other reasons. It may be empty.

**At any time the game plays a season's finale while one runs, else the running entry with the highest priority**
(`Schedule.TurnAt`). **A finale is above the priority scale**: it outranks every entry, whatever its priority, so no
entry can take the Final Battle's rules away (a plain entry may overlap it, and waits under it; a battle may not, see
below). The schedule hears the finales from the calendar (`Schedule.With(calendar.Finales()...)`), each a set, words
and a window. A tie between entries goes to the entry made last (`made_at`, then the id). An entry runs from its start
to just before its end. With nothing running, `default` plays, as a plain turn. `Schedule.SinceAt` is when the turn at
a time began: the last start or end, of an entry or a finale, that changed it.

**A turn is the entry that plays, its set, its kind, its description and its window** (its start and its end, in UTC,
zero when none runs). A finale's turn is kind battle with the zero entry and `Finale` set. An entry is a turn of its own
even when it plays the set already playing, so it is commanded again.

## Only an entry to come changes

- **An operator adds an entry, or edits or deletes one that has not started** (`Entry.Changeable`). A start must be
  after now, an end after the start, the name a set's, and the description short enough (`Draft.check`, every reason
  at once, each reason found by `errors.Is`).
- **A running entry runs to its end.** To fix one, the operator adds an entry at a higher priority. There is no "end
  now", no override button, and no call that changes which set plays at once; a save of the set in force changes its
  values at once (see [Rules](rules.md#the-operator)).
- **An edit keeps the entry's id and its `made_at`**, so it keeps its place in a tie.
- **A battle fits between the season's days** (`Plan.Clashes`, `FailedPrecondition`; see [A battle and the
  days](#a-battle-and-the-days)): a new or edited entry is judged in the plan with it (`Planner.With`), in place of the
  one kept. A plain entry is never judged, and may overlap anything.
- **A new or edited entry must name a set planet and the antibot both keep** (`schedule.Playable`): the module asks
  each, in its own words, which sets it keeps (`GetRuleSetNames` on its internal service), and the refusal names each
  module that lacks it (`planet keeps no rule set of this name: "rush"`). **An antibot that is off** serves no route
  (`Unimplemented`): it judges with no set, so it lacks none. Scoring keeps no sets: a battle's times are the entry's.
- Each change is logged at Warn by its audit decorator (`admin schedule entry added`, `edited`, `deleted`, or
  `refused`, with the set, the kind, the description, the times and the priority).

## A set a module lacks once the entry runs

An entry is checked when it is saved. If a module lacks its set when it starts (a set kept that no longer restores, or
the antibot turned on after the entry was saved), it is loud, not silent:

- **The module refuses the command** and plays on with its set; its log says so at Error (`failed to play the rules`,
  `no rule set has this name`).
- **The backoffice warns** on each entry that has not ended and names a set a module lacks ("No set rush in planet or
  the antibot."), from each module's `GetRules`, and says when a module does not play what the schedule plays.
- **The fix is an entry**: mend or make the set, then add an entry that starts in a minute, at a higher priority. It is
  a turn of its own, so it is commanded again even when it names the set refused.

## Commanding the turn

**`play_schedule_usecase` runs every second** (`gamemaster-schedule`): it reads the entries that end after now
(`Store.Unended`) and takes the turn at now. **It commands the turn when it changes, and once at each boot**, when it
remembers none: a module may have kept a stale set. Otherwise it sends nothing. `log_play_schedule` logs
each turn told at Warn (`the schedule plays`, `set`, `entry`).

**Level-triggered, with no outbox and no resend loop.** The runner keeps the last turn it told in memory, and moves it
only once Redis took both commands. A send Redis refuses is tried again the next second; a crash is followed by the
boot's send. The outbox is for a command decided in the transaction of a change (see
[the outbox](composite-layer.md#what-may-not-be-lost-the-outbox)): here the clock decides, and nothing is written.

**`redis_rule_set_commands` sends one command to each module, in its words**: `planet.v1.PlayRuleSet{rule_set,
decided_at}` and `antibot.v1.PlayRuleSet{rule_set, decided_at}`, each on its own command stream
(`commands:planet.v1.PlayRuleSet`, `commands:antibot.v1.PlayRuleSet`). `decided_at` is the runner's now. Scoring hears no
rule set: a battle reaches it as a round (see [Commanding scoring](#commanding-scoring)).

## Playing it

Planet and the antibot do the same thing, each in its own `play_rules_usecase`:

- **The latest decision wins** (`rules.Choice.Overrules`): a command decided at or before the one kept changes
  nothing, so a command delivered again, or an older one, is harmless.
- **A newer one reads its set, keeps the choice, and enforces it** in the atomic pointer (`rules.InForce`). The choice
  is kept in each module's `rule_set_choice` (one row), in the module's own schema.
- **At boot each module plays the choice it kept**, and `default` until it was told one. A kept choice whose set is
  missing or cannot play refuses the boot, as a broken `default` does.
- **A set the module does not keep, or one that cannot play, is refused**: the rules in force stay, the choice is not
  kept, the log says so, and the command is dropped (`play_rule_set_handler` acks it). Delivered again, it would hold
  back the commands behind it, and the schedule commands again only when its turn changes.
- **A choice that cannot be kept** (postgres down) fails the handler, and Redis delivers the command again.
- The antibot also tells planet every running ban again under each set it takes (`bans.Book.Restate`). A failed
  restatement fails the handler, and the command delivered again restates them.

**A save of the set in force takes over at once**: each module's `SaveRuleSet` keeps the set, then plays the set chosen
last again (`Execute`), which enforces the new values. A save of another set plays when the schedule plays it.

## The operator

`gamemaster.v1.AdminService`, on the console and admin listeners (see [Operator tools](admin.md)), and the backoffice's
Rule schedule page (see the backoffice's [Ops](../../backoffice/docs/ops.md)):

- **`GetSchedule`** is a query (`schedule_query`): every entry (the past ones too), by start, each with its kind and
  its description; `finales`, every season's finale with its window (changed on its season only); and `playing`: the
  set, the kind, since when, the entry that plays (empty when none runs; the kind is plain then), and `finale_season`,
  the season whose finale plays, read as the season running now.
- **`AddEntry`**, **`EditEntry`**, **`DeleteEntry`**: an entry that cannot be scheduled (a description past 300
  characters included), or a kind the wire does not name (`ENTRY_KIND_UNSPECIFIED` included), is `InvalidArgument`, a
  set a module lacks or an entry that started `FailedPrecondition`, an unknown id `NotFound`. The kind crosses the
  wire as `gamemaster.v1.EntryKind` (`schedulemessage.DraftOf` reads it, `KindOf` writes it, and a kept kind this build
  does not know fails the read).

## The seasons

**The gamemaster owns game time: the entries, the battles, the days and each season's start and end** (the owner,
2026-10-27). Scoring only counts points in the books and rounds it is told to open and close, and owns no clock. All
commands go one way, gamemaster to scoring; scoring never asks back.

- **A season is planned as a break and a length** (`season.Season`), kept in `gamemaster.seasons` (`number`,
  `break_before`, `length`, two intervals), apart from the entries: it has a lifecycle of its own. It may have one
  finale (see [The finale](#the-finale)).
- **The seasons are a chain, appended to** (the owner, 2026-10-28): each starts its break after the end of the one
  before, and ends its length later (`Calendar.Windows`). So none can overlap, and **moving the running season's end
  moves every season after it**. The break is the time no season runs: no book is open and no day runs. **A season after season 0 starts on a blank
  map**: planet is told to reset it (see [The map reset](map-reset.md)). Freezing the
  map during it is a rule set the schedule plays over the break, not a fact of the season.
- **Season 0's break counts from `season.Origin`, 2026-10-08 00:00 UTC**, so its days keep the numbers scoring gave
  them: day 1 ended 2026-10-08 21:00, when rounds shipped (#360). Migration `20261028120000_seasons` seeds it with no
  break and a length of 23 days 23 hours: it ends 2026-10-31 23:00 UTC, scoring's former YAML end.
- **Numbers count up from 0, and the gamemaster gives the next one** (`Calendar.Added`), after the last season.
- **The rules are the calendar's** (every reason found by `errors.Is`): a length above none and no break below none
  (`ErrInvalidSeason`); an added or edited season to come starts after now, so a break after a season that ended long
  ago must be longer; **a season that started keeps its break** (`ErrStarted`); **one that ended is not edited**
  (`ErrEnded`); **an end moves only to a time after now**. Only the last season, before it starts, is deleted
  (`ErrNotLast`): numbers keep counting with no gap, and scoring never heard of it.
- **The operator** calls `GetSeasons` (each season's break, length, finale, window, and its days by `Window.Days` at
  the daily cutoff, which the backoffice's Timeline draws), `AddSeason{break_before, length,
  finale}`, `EditSeason` and `DeleteSeason` on `gamemaster.v1.AdminService`, from the backoffice's Seasons page; each
  change is logged at Warn by its audit decorator (`admin season added`, `edited`, `deleted`, or `refused`, with the
  finale's fields). A refusal is `InvalidArgument`, `FailedPrecondition` or `NotFound` (`seasonmessage.Refusal`). An
  edit sends the whole plan: an edit with no finale removes it.

## The finale

**A season may declare one finale: its last battle, the Final Battle** (the owner, 2026-10-28). It is the only battle
that ends a season. Before, a battle was the finale when its end equalled the season's: moving the end earlier made a
running battle the finale by accident, and moving it later dropped the planned one. The owner rejected identity by
times. Now the fact "ends season S" is the season's own.

- **A finale is a length, a rule set, a description for the players and its `times`** (`season.Finale`). **Its window
  is computed, never typed**: from the season's end minus its length to the season's end (`Window.FinaleStartsAt`). So
  it follows any move of the end. A season with none has the zero `Finale`.
- **It is kept with its season, in `gamemaster.season_finales`** (`season` → `seasons.number`, cascade on delete;
  `length`, `rule_set`, `description`, `times`): one row per season that has one, so no season row holds an empty
  finale. `postgres_season_store` saves both in one statement.
- **Its rules** (every reason found by `errors.Is`): a length above none and at most the season's, a set's name, a
  description of at most 300 characters and times 1 to 10 (`ErrInvalidSeason`); **once it started, the season's end
  and the finale stay** (`ErrFinaleStarted`, `FailedPrecondition`); a finale that is added, changed or moved starts
  after now; **its set is one planet and the antibot keep** (`schedule.Playable`, `FailedPrecondition`); **it fits
  between the days like any battle** (see [A battle and the days](#a-battle-and-the-days)): it starts at the season's
  last cutoff, or the season ends at a cutoff, and no battle overlaps it. It is judged on the season's add and edit with
  every battle that has not ended, and on every battle added or edited.
- **It is a battle on the same path as an entry's** (`moments.Battle` with `EndsSeason`, the zero entry): a round of
  the season's book, `season-N/finale` (`FinaleRoundOf`, read back by `NamedOf`), worth its times, numbered by start
  among the season's battles; its set plays over its window above every entry (see [An entry](#an-entry)). It is in the
  `Timeline` with `ends_season` and no entry id, and the `Turn` carries `ends_season` while it plays.
- **`EndsSeason` is true for the finale alone.** A battle entry that runs past a season's end is still cut there, but
  is no finale. With a finale, no battle can run past the end: it would overlap.
- **The tiebreak is the finale, or none** (`Book.Tiebreak`). A day is never the tiebreak, even one that ends with the
  season: a season with no finale closes with no tiebreak round, and scoring ranks by points and rounds won.

## The days

**A day is a round of its season's book, scored at the daily cutoff** (`gamemaster.dailyCutoff`, a time of day in UTC,
from 0s to under 24h: `season.Cutoff`; 21h in production). **Day K of a season ends at the K-th cutoff after the
season's start, and starts at the cutoff before it**, or at the season's start for day 1 (`Window.Days`). A day that
ends exactly at the season's end is its last. **No day runs from the season's last cutoff to its end**: season 0's day
24 ends 2026-10-31 21:00, and from then to 23:00 only the Final Battle runs.

## A battle and the days

**No two rounds of a book overlap** (the owner, 2026-10-29): a player always knows which one snapshot scores the map,
and two are never scored at once. Before, a battle ran inside a day and both scored from the map, each at its own end.

- **A battle takes its time from the start or the end of a day** (`moments.NewPlan`): a day ends when a battle that runs
  at its end starts, and starts when a battle that runs at its start ends. A battle 18:00 to 21:00 scores day K at
  18:00 and itself at 21:00; a battle 21:00 to 23:00 scores day K at 21:00, itself at 23:00, and day K+1 runs from
  23:00 to the next 21:00. So something is scored at every cutoff. `today` is unset while a battle runs, and the
  `DayStarted`, the `DayEnded`, the debrief and `GetSeasons` tell the day as it is cut: a battle added or moved during
  a day moves its end, which `DayStarted` tells again. Missions count nothing while a battle runs: no day, no missions.
- **`Plan.Clashes(now)`** judges every battle that has not ended, the finale included, and names each one that breaks
  a rule (every reason found by `errors.Is`, `FailedPrecondition`):
  - it starts or ends at the daily cutoff (`ErrOffCutoff`): a battle across a cutoff would score nothing at it;
  - it overlaps no other battle, nor the finale (`ErrOverlap`);
  - it leaves each day some time (`ErrDayTaken`): a battle of a day or more, or two that meet inside one day, are
    refused.
- **A battle that starts in a break is no round**, so it takes nothing from a day and is not judged. A battle that has
  ended is not judged either: an entry kept before the rule stays as it was planned.
- **An entry changes only before it starts**, so every day edge it moves is still to come: scoring hears a day's end
  only in its `CloseRound`.

## The ids

The gamemaster's format; scoring treats each as an opaque string (`moments`):

- the book of season N: `season-N` (`BookOf`, read back by `SeasonOf`);
- day K of season N: `season-N/day-K` (`DayRoundOf`);
- a battle's round: its entry's uuid, as a string (`BattleRoundOf`). `NamedOf` reads a round id back to a day or an
  entry; anything else, `season-0/own-1` included, names nothing (`ErrUnknownRound`).

## Commanding scoring

**`moments.Plan` is every book**: one per season, its days, and its battles: each entry of kind battle with times of 1
or more that **starts while the season runs**, a round of that book cut at the season's end, and the season's finale,
all numbered by start among the season's battles (a tie goes to the entry made first, then the finale). A battle that
starts in a break is no round. **A book's tiebreak** is its finale's round, or none (`Book.Tiebreak`).

**`play_scoring_usecase` runs every second** (`gamemaster-scoring`): it plans the books (`moments.Planner` reads the
seasons and the entries that end after `season.Origin`) and sends each command whose time has come
(`Plan.DueAt`), **once per process**: the runner keeps each command's key (`moments.Key`, its action and ids) in memory
once Redis took it.

- `OpenBook{season-N, starts_at}` while the season runs; `CloseBook{season-N, ends_at, tiebreak_round_id}` once it ended.
- `OpenRound{id, book, times, starts_at}` while a day (times 1) or a battle (the entry's times) runs; `CloseRound{id,
  ends_at}` once it ended.
- **An open is only sent while what it opens has not ended**: a round that already ended is never opened, since scoring
  would score it from today's map. **The closes are sent for every ended round and book of every started season**, so
  a boot after downtime closes what ended meanwhile; scoring ignores the ones it handled already.
- **In order**: by time, and at one instant OpenBook, CloseRound, OpenRound, then CloseBook. A command Redis refuses
  stops the tick, so it and every one after it go the next second, in order.
- **Level-triggered, no outbox**, like the schedule: the clock decides, and a boot sends again what is due.
  `log_play_scoring` logs each command at Warn (`scoring is commanded`).
- `redis_scoring_commands` sends each as one `scoring.v1.ScoringCommand` on its one stream
  (`commands:scoring.v1.ScoringCommand`), so a round's close always reaches scoring before its book's.
- **A moved season end** moves its book's close, its days (a day may start now, from the old last cutoff), its battles'
  cut and its finale, which keeps its length before the new end. **A day that opened keeps running when the end moves
  before its cutoff**: it is cut at the season's end (`Window.CutDay`), closed just before the book, and no day opens
  after it; it breaks no tie. A running battle entry is cut at the new end the same way, and is no finale. The runner
  keeps each round it opens in `opened_rounds` before it sends the open, so a boot still knows the cut day.

## Telling the days

**The gamemaster tells each day as it starts, so no other module keeps a clock of its own** (the owner, 2026-10-29:
missions had its own cutoff, which only happened to match `dailyCutoff`). `tell_day_usecase` runs every second
(`gamemaster-days`): it plans the books, takes the day that runs now (`Plan.DayAt`), and in one transaction keeps it in
`told_days` (`round_id`, `ends_at`) and, **only when it is new or its end moved**, records
`gamemaster.v1.DayStarted{season, day, round_id, book_id, starts_at, ends_at}` in the outbox. **Level-triggered**: a
boot tells the day that runs if it was never told, as the day that ran when the event shipped. **A moved season end that
cuts the day** tells it again with the new end. No day runs between two seasons, nor from a season's last cutoff to its
end, so nothing is told then. `log_tell_day` logs each day told at Info (`day told`). Missions hears it (see
[Missions](missions.md)).

## Hearing scoring

**The gamemaster hears `scoring.v1.RoundScored` (`gamemaster-rounds`) and `scoring.v1.BookClosed`
(`gamemaster-books`).** `hear_round_usecase` and `hear_book_usecase` plan the books, read what the round or book was,
and in one transaction keep it (`scored_rounds`, `closed_books`, inserted once) and, **only when it is new**, record
its moment in the gamemaster outbox, relayed by `cpoutbox.NewRelay` (`gamemaster-outbox`):

- `gamemaster.v1.DayEnded{season, day, round_id, ended_at}` for a day;
- `gamemaster.v1.BattleEnded{season, number, entry_id, times, ends_season, round_id, ended_at}` for a battle;
  `ends_season` is the finale's alone, and its `entry_id` is empty;
- `gamemaster.v1.SeasonEnded{season, book_id, ended_at, finale_round_id, finale_battle}` for a book; `finale_battle` is
  the `BattleEnded` of the season's finale, unset with none (and `finale_round_id` empty then).

They carry ids, not numbers: whoever needs the points reads scoring by id. **A round or book id no day, battle or
season has is logged at Warn and dropped** (acked, not kept). A second delivery finds the row and records nothing.

## The game client

**`gamemaster.v1.GamemasterService`, on the public router, is what the client reads to give a battle its look, to
count down to the next one, and to play a debrief.** Its `Turn` is the kind of the running entry (plain when none runs),
its `description` (empty when none runs), `since` when the turn began (unset before any entry), `ends_at`, when the
running entry ends (unset when none runs), `entry_id`, and for a running battle that is a round, `round_id` and
`times`. **While a season's finale plays**, the turn is kind battle, its description, its window, `round_id`
`season-N/finale`, its `times` and `ends_season`, with no `entry_id`; which finale is the season running now
(`Plan.FinaleAt`). **Never the rule set's name**: the query (`turn_query`) reads the entries and the seasons and computes the
turn, the plan and the timeline by the domain's rules, and answers in the published language only.

**The `Timeline` is the battles to come**, next to the turn: each entry of kind battle that has not started
(`Schedule.BattlesAfter`) and each season's finale that has not started (`Plan.FinalesAfter`), by start, a tie going to
the entry made first then the finale, as a `Battle`: its entry id (empty for a finale), its start, its end, its
description, its `times`, its `round_id` when it is a round, and `ends_season` for a finale alone. **Never a plain entry**: a quiet antibot change stays the operator's. An entry to come
may still be moved or deleted; the stream sends each change within a second, so a countdown is fixed at once (the user
lifted the rule "no entry to come crosses", 2026-10-27). **The timeline carries the seasons and the day too**:
`current`, the season that runs (unset between two), `next`, the first that has not started, each a
`Season{number, starts_at, ends_at, book_id}`, and `today`, the day in progress (`Day{number, starts_at, ends_at,
round_id}`; unset between two seasons and from a season's last cutoff to its end).

**The `Debrief` is the last moment scoring scored**: the latest row of `scored_rounds` and `closed_books` by its end, a
book first at one time (it closes after its rounds), as the `DayEnded`, `BattleEnded` or `SeasonEnded` the outbox
recorded for it; unset before any, and for a moment the plan no longer knows.

**`ListenForEvents` is how the client follows them: no polling.** A `GamemasterEvent` is a `oneof` of `turn`,
`timeline`, `debrief` and `heartbeat`, as the other live streams (see [The live streams](live-streams.md)). The stream
sends the turn, the timeline and the debrief (when there is one) as its first events, then each one again only when it
changed, within a second, and a heartbeat every `httpServer.streamHeartbeat` meanwhile. So the client needs no read at
the entry's end: when the entry ends, or one above it starts, the next turn arrives and the look changes then (and the
battle that started leaves the timeline); `ends_at` is for a countdown. `inprocess_turn_feed` is the source: a runner
(`gamemaster-turn`) that, while at least one stream follows, reads them each second through `turn_query` and hands them
to every stream when any differs from the last one read; the handler sends the part that changed. A new stream gets the
last read at once, and the first one after nobody followed waits for a fresh read. A failed read is logged by
`log_turn_reader` and the next second's read sends. Nobody following, it reads nothing. The handler returns when its
context ends, so the drain ends it cleanly.

**`GetTurn` is the same turn, timeline and debrief as a cached GET**, for a reader that does not follow the stream. It
is `NO_SIDE_EFFECTS`, so Connect sends it as an HTTP GET, and `gamemasterv1controller.NewCacheInterceptor` sets
`Cache-Control: public, max-age=5, s-maxage=5` on the answer (the map's pattern, see [The map load](map-load.md)):
an answer read within five seconds of a change may still say the turn before it.

## Tests

The rule is pinned by `schedule_test.go` (priorities, ties, the end, since when, two entries of one set, the turn's
kind, its window, the battles after a time and their order) and `entry_test.go` (only an entry to come changes, every
reason, a kind nobody knows, a battle's times 1 to 10 and a plain entry's none). `schedule.StoreContractSuite` runs on
the memory and postgres stores. `finale_test.go` pins the finale's window, its rules, an end that stays once it
started, and a moved end that moves it. `season`'s tests pin the calendar's rules and the days (season 0's day 1 ends
2026-10-08 21:00, day 24 2026-10-31 21:00, none from 21:00 to 23:00), and `season.StoreContractSuite` runs on both
stores. `moments`' tests pin the ids read back, a day cut by a battle at its end and one at its start, the battles that
fit (season 0's Final Battle, battles at a cutoff, a plain entry under the finale, a battle that ended) and each clash;
`moments.StoreContractSuite` (each round and book kept once) runs on both stores. The entry and season use cases' tests
pin a battle over the finale refused, a moved battle judged in place of the one kept, and a finale at no cutoff
refused.

**`finalnight`'s test is the safety net for Sat 31 Oct**: season 0 as the migrations seed it, from the start of day 24
to after 23:00, second by second through `play_schedule_usecase`, `play_scoring_usecase` and a fake scoring that
answers each close through `hear_round_usecase` and `hear_book_usecase`: planet and the antibot are told `rush` at
21:00 and `default` at 23:00 and nothing else; `CloseRound season-0/day-24` then `OpenRound season-0/finale × 3` at
21:00, `CloseRound season-0/finale` then `CloseBook season-0` tiebreak `season-0/finale` at 23:00; and the outbox
holds `DayEnded` day 24, then `BattleEnded` (`ends_season`), then `SeasonEnded` with that finale.
`postgres_season_store`'s test pins the seed: season 0's finale, rush × 3, 31 Oct 21:00 to 23:00.

**`play_scoring_usecase`'s test is the Final Battle night** too: `CloseRound season-0/day-24` then `OpenRound
season-0/finale × 3` at 21:00, the closes at 23:00, nothing in between, and a boot at 23:05 that sends every close again
and no open. It also pins a moved season end that moves the finale, a day cut by a season end moved before its cutoff
(its close then the book's, no tiebreak, again after a boot), a running battle entry cut the same way and no finale, a
boot mid-day (the book and the running day opened, the ended days closed), a send Redis refused sent again in order,
and nothing before any season. `hear_round_usecase`'s and `hear_book_usecase`'s tests pin the event recorded once (a day, the Final Battle as
the season's second battle that ends it, the season's end with its finale and its finale battle), a second delivery that records nothing,
an unknown id dropped (`season-1/finale` of a season with none included), and a failed keep. `play_schedule_usecase`'s
test pins a finale's set played over its window and above an entry at the highest priority. `redis_scoring_commands`' test pins each command in scoring's words, and
`redis_rule_set_commands`' that scoring hears no rule set.

`turn_query`'s test pins the public read while a battle runs (its id and round among it), after the last one ended and
before any, the timeline (a battle to come in the players' words with its times, round and finale, no plain entry, none
once started), the seasons with their books, today, and the debrief (none before any, a day, then the season's end
after the round that ends with it); `cache_interceptor_test.go` pins the GET and its header. `inprocess_turn_feed`'s
test pins the feed (a stream gets the turn once read, another at once, each change to every stream, a battle added,
edited or deleted sent, a new debrief sent, a turn that holds not sent again, nothing read with nobody following, a
failed read) and `listen_for_events_handler`'s the stream (the turn, the timeline then the debrief first, only the part
that changed after, the heartbeat, the feed left when the stream closes, a clean end when the feed closes). The runner's
test pins a send at boot and at each change, none while the turn holds, and a send Redis refused sent again. The smoke
test `TestLastABattleOfTheScheduleIsPlayedByPlanetAndTheAntibotAndScoredByScoring` adds a battle entry that ends at
the next cutoff on the console listener, waits for the modules to play it and scoring to open its round, reads its kind
and its end on the public listener, and opens the stream, whose first event is it. It runs last, by its name: its
battle takes the rest of today, and the other smoke tests need a day. The suite's season 0 ends at 23:00, so its 2 h
finale starts at the cutoff.

## Before the Final Battle (Sat 31 Oct 2026, 21:00 UTC)

**Season 0's finale is the Final Battle**: migration `20261028150000_season_finales` gives season 0 a finale of 2
hours, set `rush`, times 3, "Triple points.". It runs 21:00 to 23:00 UTC because season 0 ends at 23:00; moving the end
moves it. No entry is needed, and none may be a battle over it. Before 21:00:

1. Ops → Game rules: if planet keeps no `rush`, Copy `default`, name it `rush`, tempo 3, box waits 60–120 s, 30 an
   hour. Save.
2. Ops → Anti-bot rules: if the antibot keeps no `rush`, Copy `default`, name it `rush`, `cohort.certainMembers` 50.
   Save.
3. Ops → Rule schedule: season 0's Final Battle shows as a row "Season 0's final battle". **A battle entry added
   earlier for that night** (the former way: `rush`, kind battle, 21:00 to 23:00) **must be deleted**: it would be a
   second round × 3 in season 0's book, and a second battle on the client's timeline. Delete it before 21:00 (only an
   entry to come is deleted). The row warns while a module lacks `rush`.
4. Ops → Seasons: season 0 shows "2 h, rush × 3". To change its words or times, edit season 0 before 21:00; from 21:00
   its end and its finale are locked.

If a module lacks `rush` at 21:00, it refuses the command and plays on with its set (see [A set a module lacks once
the entry runs](#a-set-a-module-lacks-once-the-entry-runs)); the round × 3 still runs in scoring. Once `rush` plays, no
module saves `rush`.

**On prod (2026-10-28) the set is `urf`, not `rush`**: the operator had made `urf` in both modules and a `urf` battle
entry for that night. The entry was deleted and season 0's finale edited to play `urf` (2 h, × 3); the migration's
`rush` seed is what a fresh database gets.

**The freeze after it (Sat 31 Oct 23:00 to Sun 1 Nov 15:00 UTC)**: the map is frozen over the break before season 1
(see [A frozen map](rules.md#a-frozen-map)). Before 23:00:

1. Ops → Game rules: Copy `default`, name it `frozen`, tick "Frozen: the map takes no play". Save.
2. Ops → Anti-bot rules: Copy `default`, name it `frozen`. Save.
3. Ops → Seasons: add season 1 with a break of 16 h, so it starts Sun 1 Nov 15:00 UTC; the client counts down to it
   on the frozen dock.
4. Ops → Rule schedule: add a plain entry `frozen`, from Sat 31 Oct 23:00 to Sun 1 Nov 15:00 UTC, priority 1. It
   starts as the Final Battle ends, so it overlaps no finale.

At 15:00 the freeze ends, season 1 opens its book and day 1, and **the map is reset** (see [The map reset](map-reset.md)):
the gamemaster tells planet `ResetMap{season-1}`, once.
