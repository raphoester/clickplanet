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
window is a round of that season's book. The Final Battle is an entry like any other, worth three days, the round that
ends the season. It is where a game stop and a map reset go next. Its name was `rules` for a day; the user renamed it,
since it organises the rules and holds none. PR #475 tried a loopback call from scoring, a baseline table and an
override button; this replaced it.

```
internal/gamemaster/
  module.go                         Config (dailyCutoff, database, redis), NewModule
  internal/
    migrations/                     schedule_entries, entry_kind, entry_description, seasons (season 0 seeded),
                                    entry_times, scored_moments (scored_rounds, opened_rounds, closed_books, outbox)
    schedule/                       EntryID, SetName, Kind, Description, Times, Priority, Draft, Entry (NewEntry, Edited,
                                    Changeable), Schedule (New, TurnAt, SinceAt, BattlesAfter), Turn; the Store, Keeper and
                                    IDProvider ports, Playable; the store's contract suite; FakeKeeper and FakeIDs (testing tag)
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
    season/                         Number, Origin, Draft, Season, Window, Calendar (Added, Edited, Deletable, Windows,
                                    RunningAt, NextAfter); Cutoff, DayNumber, Day, Window.Days; the Store port and its
                                    contract suite
      inmemory_season_store/        the Store in a map, behind the testing tag
      postgres_season_store/        the Store over gamemaster.seasons
      usecases/add_season_usecase/     audit_add_season/
      usecases/edit_season_usecase/    audit_edit_season/
      usecases/delete_season_usecase/  audit_delete_season/
    moments/                        BookID, RoundID (BookOf, DayRoundOf, BattleRoundOf, SeasonOf, NamedOf), Battle, Book,
                                    Plan (DueAt, BookAt, DayAt, BattleOf, RoundEnded, BookClosed), Planner, Command, Action,
                                    Key, Moment (DayEnded, BattleEnded, SeasonEnded); the Store and Events ports, the store's
                                    contract suite
      inmemory_moment_store/        the Store in maps, behind the testing tag
      postgres_moment_store/        the Store over gamemaster.scored_rounds and closed_books
      momentmessage/                DebriefOf and EventOf: a moment in gamemaster.v1's words
      outbox_moment_events/         Events, into the gamemaster outbox
      redis_scoring_commands/       one scoring.v1.ScoringCommand per command, on scoring's command stream
      round_scored_subscriber/      hears scoring.v1.RoundScored (gamemaster-rounds)
      book_closed_subscriber/       hears scoring.v1.BookClosed (gamemaster-books)
      usecases/play_scoring_usecase/ each command whose time has come, once per process; Runner
        log_play_scoring/           logs each command sent, and a failure
      usecases/hear_round_usecase/  keeps a scored round, and records its day's or battle's end once; log_hear_round/
      usecases/hear_book_usecase/   keeps a closed book, and records its season's end once; log_hear_book/
    gamemasterv1controller/              AdminService and GamemasterService (bags), NewCacheInterceptor
      get_schedule_handler/schedule_query/   PostgresQuery: every entry, and what plays now
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
battle, 0 for a plain entry, which scores nothing; anything else is refused with the draft's other reasons
(`ErrInvalidEntry`), and postgres checks it too. Migration `20261028130000_entry_times` gave every battle kept 3: the
only one planned was the Final Battle, worth three days.

**The kind is what the entry is to the players** (`schedule.Kind`, the postgres enum `entry_kind`): `battle`, which
the game client gives a war-like look, or `plain`, a change of the rules with no look for them (a quieter antibot
bound for a day). The set's name is the operator's word and never crosses to the players; the kind is the published
language the client reads (see [The game client](#the-game-client)). A kind nobody knows is refused
(`ErrUnknownKind`). More kinds will come: each is a value of the enum, in the proto and in postgres, and a word the
client interprets.

**The description is the entry's words for the players** (`schedule.Description`): what its rules are, in plain
text, which the client shows behind the battle chip ("Triple points. A bonus box every minute."). The client cannot
know a set's values, so the operator writes them. At most `MaxDescription` (300) characters, counted in runes; a
longer one is refused with `ErrLongDescription`, among the draft's other reasons. It may be empty.

**At any time the game plays the running entry with the highest priority** (`Schedule.TurnAt`). A tie goes to the
entry made last (`made_at`, then the id). An entry runs from its start to just before its end. With no entry running,
`default` plays, as a plain turn. `Schedule.SinceAt` is when the turn at a time began: the last start or end that
changed it.

**A turn is the entry that plays, its set, its kind, its description and its window** (its start and its end, in UTC,
zero when none runs). An entry is a turn of its own even when it plays the set already playing, so it is commanded
again.

## Only an entry to come changes

- **An operator adds an entry, or edits or deletes one that has not started** (`Entry.Changeable`). A start must be
  after now, an end after the start, the name a set's, and the description short enough (`Draft.check`, every reason
  at once, each reason found by `errors.Is`).
- **A running entry runs to its end.** To fix one, the operator adds an entry at a higher priority. There is no "end
  now", no override button, and no call that changes the rules at once.
- **An edit keeps the entry's id and its `made_at`**, so it keeps its place in a tie.
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

**No set plays while it is edited**: each module's `SaveRuleSet` refuses the set in force with `FailedPrecondition`
(`ErrPlaying`). To change the rules that play, the operator saves another set and schedules it.

## The operator

`gamemaster.v1.AdminService`, on the console and admin listeners (see [Operator tools](admin.md)), and the backoffice's
Rule schedule page (see the backoffice's [Ops](../../backoffice/docs/ops.md)):

- **`GetSchedule`** is a query (`schedule_query`): every entry (the past ones too), by start, each with its kind and
  its description, and `playing`: the set, the kind, since when, and the entry that plays (empty when none runs; the
  kind is plain then).
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
  `break_before`, `length`, two intervals), apart from the entries: it has a lifecycle of its own. The type stays open
  to one optional finale (a length, a rule set, a description), added in a later change.
- **The seasons are a chain, appended to** (the owner, 2026-10-28): each starts its break after the end of the one
  before, and ends its length later (`Calendar.Windows`). So none can overlap, and **moving the running season's end
  moves every season after it**. The break is the time no season runs: no book is open and no day runs. Freezing the
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
- **The operator** calls `GetSeasons` (each season's break, length and window), `AddSeason{break_before, length}`,
  `EditSeason` and `DeleteSeason` on `gamemaster.v1.AdminService`, from the backoffice's Seasons page; each change is
  logged at Warn by its audit decorator (`admin season added`, `edited`, `deleted`, or `refused`). A refusal is
  `InvalidArgument`, `FailedPrecondition` or `NotFound` (`seasonmessage.Refusal`).

## The days

**A day is a round of its season's book, scored at the daily cutoff** (`gamemaster.dailyCutoff`, a time of day in UTC,
from 0s to under 24h: `season.Cutoff`; 21h in production). **Day K of a season ends at the K-th cutoff after the
season's start, and starts at the cutoff before it**, or at the season's start for day 1 (`Window.Days`). A day that
ends exactly at the season's end is its last. **No day runs from the season's last cutoff to its end**: season 0's day
24 ends 2026-10-31 21:00, and from then to 23:00 only the Final Battle runs.

## The ids

The gamemaster's format; scoring treats each as an opaque string (`moments`):

- the book of season N: `season-N` (`BookOf`, read back by `SeasonOf`);
- day K of season N: `season-N/day-K` (`DayRoundOf`);
- a battle's round: its entry's uuid, as a string (`BattleRoundOf`). `NamedOf` reads a round id back to a day or an
  entry; anything else, `season-0/own-1` included, names nothing (`ErrUnknownRound`).

## Commanding scoring

**`moments.Plan` is every book**: one per season, its days, and its battles, each an entry of kind battle with times
of 1 or more that **starts while the season runs**, a round of that book cut at the season's end, numbered by start
among the season's battles (a tie goes to the entry made first). A battle that starts in a break is no round. **A
book's finale** is the round that ends exactly at the season's end: the battle if one does, else a day, else none; it is
the tiebreak scoring ranks the book by.

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
- **A moved season end** moves its book's close, its days (a day may start now, from the old last cutoff) and its
  battles' cut; a battle that no longer ends the season is no finale. **A day that opened keeps running when the end
  moves before its cutoff**: it is cut at the season's end (`Window.CutDay`), closed just before the book, and no day
  opens after it; it is the finale unless a battle ends there. A running battle is cut at the new end the same way. The
  runner keeps each round it opens in `opened_rounds` before it sends the open, so a boot still knows the cut day.

## Hearing scoring

**The gamemaster hears `scoring.v1.RoundScored` (`gamemaster-rounds`) and `scoring.v1.BookClosed`
(`gamemaster-books`).** `hear_round_usecase` and `hear_book_usecase` plan the books, read what the round or book was,
and in one transaction keep it (`scored_rounds`, `closed_books`, inserted once) and, **only when it is new**, record
its moment in the gamemaster outbox, relayed by `cpoutbox.NewRelay` (`gamemaster-outbox`):

- `gamemaster.v1.DayEnded{season, day, round_id, ended_at}` for a day;
- `gamemaster.v1.BattleEnded{season, number, entry_id, times, ends_season, round_id, ended_at}` for a battle;
- `gamemaster.v1.SeasonEnded{season, book_id, ended_at, finale_round_id, finale_battle}` for a book; `finale_battle` is
  the `BattleEnded` of the finale when it is a battle, unset otherwise.

They carry ids, not numbers: whoever needs the points reads scoring by id. **A round or book id no day, battle or
season has is logged at Warn and dropped** (acked, not kept). A second delivery finds the row and records nothing.

## The game client

**`gamemaster.v1.GamemasterService`, on the public router, is what the client reads to give a battle its look, to
count down to the next one, and to play a debrief.** Its `Turn` is the kind of the running entry (plain when none runs),
its `description` (empty when none runs), `since` when the turn began (unset before any entry), `ends_at`, when the
running entry ends (unset when none runs), `entry_id`, and for a running battle that is a round, `round_id` and
`times`. **Never the rule set's name**: the query (`turn_query`) reads the entries and the seasons and computes the
turn, the plan and the timeline by the domain's rules, and answers in the published language only.

**The `Timeline` is the battles to come**, next to the turn: each entry of kind battle that has not started
(`Schedule.BattlesAfter`), by start, a tie going to the entry made first, as a `Battle`: its entry id, its start, its
end, its description, its `times`, and, when it starts in a season, its `round_id` and `ends_season` (its round ends at
the season's end: the finale). **Never a plain entry**: a quiet antibot change stays the operator's. An entry to come
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
the memory and postgres stores. `season`'s tests pin the calendar's rules and the days (season 0's day 1 ends
2026-10-08 21:00, day 24 2026-10-31 21:00, none from 21:00 to 23:00), and `season.StoreContractSuite` runs on both
stores. `moments`' tests pin the ids read back; `moments.StoreContractSuite` (each round and book kept once) runs on
both stores.

**`play_scoring_usecase`'s test is the Final Battle night**: season 0 to 2026-10-31 23:00, cutoff 21h, a battle rush × 3
from 21:00 to 23:00; second by second through the night, exactly `CloseRound season-0/day-24` then `OpenRound <entry>
× 3` at 21:00, `CloseRound <entry>` then `CloseBook season-0` tiebreak `<entry>` at 23:00, nothing in between, and a
boot at 23:05 that sends every close again and no open. It also pins a day cut by a season end moved before its cutoff
(its close then the book's, tiebreak the day, again after a boot), a running battle cut the same way, a boot mid-day (the book and the running day
opened, the ended days closed), a moved season end, a send Redis refused sent again in order, and nothing before any
season. `hear_round_usecase`'s and `hear_book_usecase`'s tests pin the event recorded once (a day, the Final Battle as
the season's second battle that ends it, the season's end with its finale and its finale battle), a second delivery that records nothing,
an unknown id dropped, and a failed keep. `redis_scoring_commands`' test pins each command in scoring's words, and
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
test `TestAnEntryOfTheScheduleIsPlayedByPlanetTheAntibotAndSeasons` adds a battle entry on the console listener, waits
for the modules to play it, reads its kind and its end on the public listener, and opens the stream, whose first event
is it.

## Before the Final Battle (Sat 31 Oct 2026, 21:00 UTC)

Nothing plays `rush` until an entry names it, and **the Final Battle is that entry**: season 0's end (23:00) is the
gamemaster's season, and the last daily cutoff (21:00) ends day 24; neither is the battle. Before 21:00, on the
backoffice:

1. Ops → Game rules: Copy `default`, name it `rush`, tempo 3, box waits 60–120 s, 30 an hour. Save.
2. Ops → Anti-bot rules: Copy `default`, name it `rush`, `cohort.certainMembers` 50. Save.
3. Ops → Rule schedule: Add an entry: `rush`, kind battle, times 3, a description of its rules for the players
   ("Triple points."), from 31 Oct 21:00 UTC to 23:00 UTC, priority 1. It is refused until both sets exist. The kind is
   what the game client shows, and a battle is a round of season 0's book: a plain entry would play `rush` with no look
   and no points. It ends at the season's end, so it is the finale and breaks a tie.

Once `rush` plays, no module saves `rush`; an edit is a new set and a new entry.
