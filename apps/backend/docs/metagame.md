# The metagame module (`internal/metagame/`)

**The game's meta: which rules play together, and when.** Planet keeps its game rules in its sets (see
[Rules](rules.md)), the antibot its bounds in its own (see [The antibot's rules](antibot-rules.md)), seasons how it
counts points in its own (see [The season rules](season-rules.md)). The metagame module never holds a rule: it picks
which set each module plays, by name, so a set of planet, one of the antibot and one of seasons play as one. No module
picks the set it plays: the metagame's schedule does, and no operator call changes it at once (the user's rule,
2026-10-26: "there needs to be a schedule of overrides with a hierarchy … they have to use schedules and priority
orders, always").

**A battle is not bound to a season** (the user, 2026-10-26): an operator may launch an unplanned battle tonight
without touching the seasons. So the metagame is a module of its own, above seasons as it is above planet and the
antibot (the user, 2026-10-27): it commands seasons a set in seasons' words, and seasons counts points by it. The Final
Battle is an entry like any other, whose seasons set makes its window a round of its own worth three days, the round
that ends the season. The metagame knows the sets' names, never their values: it never hears of a point. It is where a
game stop and a map reset go next. Its name was `rules` for a day; the user renamed it, since it organises the rules and holds none. PR #475 tried a loopback call from seasons, a baseline table and an override button; this replaced it.

```
internal/metagame/
  module.go                         Config (database, redis), NewModule
  internal/
    migrations/                     schedule_entries, entry_kind, entry_description
    schedule/                       EntryID, SetName, Kind, Description, Priority, Draft, Entry (NewEntry, Edited, Changeable),
                                    Schedule (New, TurnAt, SinceAt), Turn; the Store, Keeper and IDProvider ports,
                                    Playable; the store's contract suite; FakeKeeper and FakeIDs (testing tag)
      inmemory_entry_store/         the Store in a map, behind the testing tag
      postgres_entry_store/         the Store over metagame.schedule_entries
      uuid_id_provider/             a uuidv7 for each entry
      rpc_planet_rule_sets/         Keeper, from planet.v1.InternalService/GetRuleSetNames
      rpc_antibot_rule_sets/        Keeper, from antibot.v1.InternalService/GetRuleSetNames
      rpc_seasons_rule_sets/        Keeper, from seasons.v1.InternalService/GetRuleSetNames
      redis_rule_set_commands/      planet.v1, antibot.v1 and seasons.v1 PlayRuleSet, on their command streams
      usecases/add_entry_usecase/     audit_add_entry/
      usecases/edit_entry_usecase/    audit_edit_entry/
      usecases/delete_entry_usecase/  audit_delete_entry/
      usecases/play_schedule_usecase/ the turn at now, commanded when it changes; Runner
        log_play_schedule/          logs each turn told, and a failure
    metagamev1controller/              AdminService and MetagameService (bags), NewCacheInterceptor
      get_schedule_handler/schedule_query/   PostgresQuery: every entry, and what plays now
      add_entry_handler/  edit_entry_handler/  delete_entry_handler/
      get_turn_handler/turn_query/  PostgresQuery: the turn now, in the players' words, for the game client
      listen_for_events_handler/    the stream: the turn, then each change, and heartbeats
        inprocess_turn_feed/        reads the turn each second while a stream follows it; log_turn_reader
      schedulemessage/              DraftOf, KindOf, and Refusal: the code of each refusal
```

## An entry

**An entry is a rule set, a kind, a description, a start, an end and a priority**, kept in
`metagame.schedule_entries`. Each has an end: one that never ended could never be stopped.

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
- **A new or edited entry must name a set planet, the antibot and seasons all keep** (`schedule.Playable`): the module
  asks each, in its own words, which sets it keeps (`GetRuleSetNames` on its internal service), and the refusal names
  each module that lacks it (`planet keeps no rule set of this name: "rush"`). **An antibot that is off** serves no route
  (`Unimplemented`): it judges with no set, so it lacks none.
- Each change is logged at Warn by its audit decorator (`admin schedule entry added`, `edited`, `deleted`, or
  `refused`, with the set, the kind, the description, the times and the priority).

## A set a module lacks once the entry runs

An entry is checked when it is saved. If a module lacks its set when it starts (a set kept that no longer restores, the
antibot turned on after the entry was saved, or an entry saved before seasons kept sets), it is loud, not silent:

- **The module refuses the command** and plays on with its set; its log says so at Error (`failed to play the rules`,
  `no rule set has this name`).
- **The backoffice warns** on each entry that has not ended and names a set a module lacks ("No set rush in planet, the
  antibot or seasons."), from each module's `GetRules`, and says when a module does not play what the schedule plays
  ("Seasons does not play rush.").
- **The fix is an entry**: mend or make the set, then add an entry that starts in a minute, at a higher priority. It is
  a turn of its own, so it is commanded again even when it names the set refused.

## Commanding the turn

**`play_schedule_usecase` runs every second** (`metagame-schedule`): it reads the entries that end after now
(`Store.Unended`) and takes the turn at now. **It commands the turn when it changes, and once at each boot**, when it
remembers none: a module may have kept a stale set. Otherwise it sends nothing. `log_play_schedule` logs
each turn told at Warn (`the schedule plays`, `set`, `entry`).

**Level-triggered, with no outbox and no resend loop.** The runner keeps the last turn it told in memory, and moves it
only once Redis took all three commands. A send Redis refuses is tried again the next second; a crash is followed by the
boot's send. The outbox is for a command decided in the transaction of a change (see
[the outbox](composite-layer.md#what-may-not-be-lost-the-outbox)): here the clock decides, and nothing is written.

**`redis_rule_set_commands` sends one command to each module, in its words**: `planet.v1.PlayRuleSet{rule_set,
decided_at}`, `antibot.v1.PlayRuleSet{rule_set, decided_at}` and `seasons.v1.PlayRuleSet{rule_set, decided_at,
starts_at, ends_at}`, each on its own command stream (`commands:planet.v1.PlayRuleSet`, `commands:antibot.v1.PlayRuleSet`,
`commands:seasons.v1.PlayRuleSet`). `decided_at` is the runner's now. **Seasons is told the entry's window** (both unset
when `default` plays): a running entry's window never changes, so a set with a round of its own makes it of exactly
that window, and the round's end is known from its start. No module hears of the metagame module.

## Playing it

Planet, the antibot and seasons do the same thing, each in its own `play_rules_usecase`:

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
- Seasons opens the round of its own a set asks of the entry's window before it keeps the choice, so a round that
  failed to open is opened when the command comes again (see [The season rules](season-rules.md)).

**No set plays while it is edited**: each module's `SaveRuleSet` refuses the set in force with `FailedPrecondition`
(`ErrPlaying`). To change the rules that play, the operator saves another set and schedules it.

## The operator

`metagame.v1.AdminService`, on the console and admin listeners (see [Operator tools](admin.md)), and the backoffice's
Rule schedule page (see the backoffice's [Ops](../../backoffice/docs/ops.md)):

- **`GetSchedule`** is a query (`schedule_query`): every entry (the past ones too), by start, each with its kind and
  its description, and `playing`: the set, the kind, since when, and the entry that plays (empty when none runs; the
  kind is plain then).
- **`AddEntry`**, **`EditEntry`**, **`DeleteEntry`**: an entry that cannot be scheduled (a description past 300
  characters included), or a kind the wire does not name (`ENTRY_KIND_UNSPECIFIED` included), is `InvalidArgument`, a
  set a module lacks or an entry that started `FailedPrecondition`, an unknown id `NotFound`. The kind crosses the
  wire as `metagame.v1.EntryKind` (`schedulemessage.DraftOf` reads it, `KindOf` writes it, and a kept kind this build
  does not know fails the read).

## The game client

**`metagame.v1.MetagameService`, on the public router, is what the client reads to give a battle its look.** Its
`Turn` is the kind of the running entry (plain when none runs), its `description` (empty when none runs), `since`
when the turn began (unset before any entry), and `ends_at`, when the running entry ends (unset when none runs). **No
entry to come crosses**: the client counts down to the end of the day and of the season (seasons'), not to a battle,
since an entry to come may still be moved or deleted (the user, 2026-10-27).
**Never the rule set's name**: the query (`turn_query`) reads the entries and computes the turn by `Schedule`'s rule,
as `GetSchedule` does, and answers in the published language only.

**`ListenForEvents` is how the client follows it: no polling.** A `MetagameEvent` is a `oneof` of `turn` and
`heartbeat`, as the other live streams (see [The live streams](live-streams.md)). The stream sends the turn as its
first event, then each change of it within a second, and a heartbeat every `httpServer.streamHeartbeat` meanwhile.
So the client needs no read at the entry's end: when the entry ends, or one above it starts, the next turn arrives
and the look changes then; `ends_at` is for a countdown. `inprocess_turn_feed` is the source: a runner
(`metagame-turn`) that, while at least one stream follows, reads the turn each second through `turn_query` (the
schedule runner reads the entries each second too) and hands it to every stream when it differs from the last one
sent; a new stream gets the last turn at once, and the first one after nobody followed waits for a fresh read. A
failed read is logged by `log_turn_reader` and the next second's read sends. Nobody following, it reads nothing. The
handler returns when its context ends, so the drain ends it cleanly.

**`GetTurn` is the same turn as a cached GET**, for a reader that does not follow the stream. It is
`NO_SIDE_EFFECTS`, so Connect sends it as an HTTP GET, and `metagamev1controller.NewCacheInterceptor` sets
`Cache-Control: public, max-age=5, s-maxage=5` on the answer (the map's pattern, see [The map load](map-load.md)):
an answer read within five seconds of a change may still say the turn before it.

## Tests

The rule is pinned by `schedule_test.go` (priorities, ties, the end, since when, two entries of one set, the turn's
kind, its window) and `entry_test.go` (only an entry to come changes, every reason, a kind nobody knows). `schedule.StoreContractSuite`
runs on the memory and postgres stores. `turn_query`'s test pins the public read while an entry runs, after the last
one ended and before any; `cache_interceptor_test.go` pins the GET and its header. `inprocess_turn_feed`'s test pins
the feed (a stream gets the turn once read, another at once, each change to every stream, a turn that holds not sent
again, nothing read with nobody following, a failed read) and `listen_for_events_handler`'s the stream (the turn
case, the heartbeat, the feed left when the stream closes, a clean end when the feed closes). The runner's test pins a send at boot and at each change, none while the turn holds, and a send Redis
refused sent again. Each module's `play_rule_set_handler` test pins the latest decision, a redelivery and an unknown
set; the choice stores have their own contract suite (`rules.ChoiceStoreContractSuite`). The smoke test
`TestAnEntryOfTheScheduleIsPlayedByPlanetTheAntibotAndSeasons` adds a battle entry on the console listener, waits for
the three modules to play it, reads its kind and its end on the public listener, and opens the stream, whose first event is it.

## Before the Final Battle (Sat 31 Oct 2026, 21:00 UTC)

Nothing plays `rush` until an entry names it, and **the Final Battle is that entry**: seasons' calendar keeps the
season's end (23:00) and the daily cutoff (21:00), not the battle. Before 21:00, on the backoffice:

1. Ops → Game rules: Copy `default`, name it `rush`, tempo 3, box waits 60–120 s, 30 an hour. Save.
2. Ops → Anti-bot rules: Copy `default`, name it `rush`, `cohort.certainMembers` 50. Save.
3. Ops → Season rules: Copy `default`, name it `rush`, a round of its own on, points × 3. Save. **An entry saved
   before seasons kept sets names a set seasons lacks**: the Rule schedule page says "No set rush in seasons." on it.
   Without it, seasons refuses the command, no round of its own opens, and the season ends at 23:00 after its last
   day, with no Final Battle points.
4. Ops → Rule schedule: Add an entry: `rush`, kind battle, a description of its rules for the players ("Triple
   points."), from 31 Oct 21:00 UTC to 23:00 UTC, priority 1. It is refused until the three sets exist. The kind is
   what the game client shows: a plain entry would play `rush` with no look.

Once `rush` plays, no module saves `rush`; an edit is a new set and a new entry.
