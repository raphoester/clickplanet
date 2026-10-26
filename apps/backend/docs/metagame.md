# The metagame module (`internal/metagame/`)

**The game's meta: which rules play together, and when.** Planet keeps its game rules in its sets (see
[Rules](rules.md)), the antibot its bounds in its own (see [The antibot's rules](antibot-rules.md)). The metagame
module never holds a rule: it picks which set each module plays, by name, so a set of planet and a set of the antibot
play as one. Neither module picks the set it plays: the metagame's schedule does, and no operator call changes it at once (the user's rule,
2026-10-26: "there needs to be a schedule of overrides with a hierarchy … they have to use schedules and priority
orders, always").

**A battle is not bound to a season** (the user, 2026-10-26): an operator may launch an unplanned battle tonight
without touching the seasons. So the metagame is a module of its own, and it knows nothing of seasons: the Final
Battle is an entry like any other. It knows the sets' names, never their values. It is where a game stop and a map
reset go next. Its name was `rules` for a day; the user renamed it, since it organises the rules and holds none. PR #475 tried a loopback call from seasons, a baseline table and an override button; this replaced it.

```
internal/metagame/
  module.go                         Config (database, redis), NewModule
  internal/
    migrations/                     schedule_entries
    schedule/                       EntryID, SetName, Priority, Draft, Entry (NewEntry, Edited, Changeable),
                                    Schedule (New, TurnAt, SinceAt), Turn; the Store, Keeper and IDProvider ports,
                                    Playable; the store's contract suite; FakeKeeper and FakeIDs (testing tag)
      inmemory_entry_store/         the Store in a map, behind the testing tag
      postgres_entry_store/         the Store over metagame.schedule_entries
      uuid_id_provider/             a uuidv7 for each entry
      rpc_planet_rule_sets/         Keeper, from planet.v1.InternalService/GetRuleSetNames
      rpc_antibot_rule_sets/        Keeper, from antibot.v1.InternalService/GetRuleSetNames
      redis_rule_set_commands/      planet.v1.PlayRuleSet and antibot.v1.PlayRuleSet, on their command streams
      usecases/add_entry_usecase/     audit_add_entry/
      usecases/edit_entry_usecase/    audit_edit_entry/
      usecases/delete_entry_usecase/  audit_delete_entry/
      usecases/play_schedule_usecase/ the turn at now, commanded when it changes; Runner
        log_play_schedule/          logs each turn told, and a failure
    metagamev1controller/              AdminService (a bag)
      get_schedule_handler/schedule_query/   PostgresQuery: every entry, and what plays now
      add_entry_handler/  edit_entry_handler/  delete_entry_handler/
      schedulemessage/              DraftOf, and Refusal: the code of each refusal
```

## An entry

**An entry is a rule set, a start, an end and a priority**, kept in `metagame.schedule_entries`. Each has an end: one that
never ended could never be stopped.

**At any time the game plays the running entry with the highest priority** (`Schedule.TurnAt`). A tie goes to the
entry made last (`made_at`, then the id). An entry runs from its start to just before its end. With no entry running,
`default` plays. `Schedule.SinceAt` is when the turn at a time began: the last start or end that changed it.

**A turn is the entry that plays and its set.** An entry is a turn of its own even when it plays the set already
playing, so it is commanded again.

## Only an entry to come changes

- **An operator adds an entry, or edits or deletes one that has not started** (`Entry.Changeable`). A start must be
  after now, an end after the start, and the name a set's (`Draft.check`, every reason at once).
- **A running entry runs to its end.** To fix one, the operator adds an entry at a higher priority. There is no "end
  now", no override button, and no call that changes the rules at once.
- **An edit keeps the entry's id and its `made_at`**, so it keeps its place in a tie.
- **A new or edited entry must name a set planet and the antibot both keep** (`schedule.Playable`): the module asks
  each, in its own words, which sets it keeps (`GetRuleSetNames` on its internal service), and the refusal names each
  module that lacks it (`planet keeps no rule set of this name: "rush"`). **An antibot that is off** serves no route
  (`Unimplemented`): it judges with no set, so it lacks none.
- Each change is logged at Warn by its audit decorator (`admin schedule entry added`, `edited`, `deleted`, or
  `refused`, with the set, the times and the priority).

## A set a module lacks once the entry runs

An entry is checked when it is saved. If a module lacks its set when it starts (a set kept that no longer restores, or
the antibot turned on after the entry was saved), it is loud, not silent:

- **The module refuses the command** and plays on with its set; its log says so at Error (`failed to play the rules`,
  `no rule set has this name`).
- **The backoffice warns** on each entry that has not ended and names a set a module lacks ("No set rush in planet or
  the antibot."), from planet's and the antibot's `GetRules`, and says when a module does not play what the schedule
  plays ("Planet does not play rush.").
- **The fix is an entry**: mend or make the set, then add an entry that starts in a minute, at a higher priority. It is
  a turn of its own, so it is commanded again even when it names the set refused.

## Commanding the turn

**`play_schedule_usecase` runs every second** (`metagame-schedule`): it reads the entries that end after now
(`Store.Unended`) and takes the turn at now. **It commands the turn when it changes, and once at each boot**, when it
remembers none: planet and the antibot may have kept a stale set. Otherwise it sends nothing. `log_play_schedule` logs
each turn told at Warn (`the schedule plays`, `set`, `entry`).

**Level-triggered, with no outbox and no resend loop.** The runner keeps the last turn it told in memory, and moves it
only once Redis took both commands. A send Redis refuses is tried again the next second; a crash is followed by the
boot's send. The outbox is for a command decided in the transaction of a change (see
[the outbox](composite-layer.md#what-may-not-be-lost-the-outbox)): here the clock decides, and nothing is written.

**`redis_rule_set_commands` sends one command to each module, in its words**: `planet.v1.PlayRuleSet{rule_set,
decided_at}` and `antibot.v1.PlayRuleSet{rule_set, decided_at}`, each on its own command stream
(`commands:planet.v1.PlayRuleSet`, `commands:antibot.v1.PlayRuleSet`). `decided_at` is the runner's now. Planet and
the antibot never hear of the metagame module.

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

`metagame.v1.AdminService`, on the console and admin listeners (see [Operator tools](admin.md)), and the backoffice's
Rule schedule page (see the backoffice's [Ops](../../backoffice/docs/ops.md)):

- **`GetSchedule`** is a query (`schedule_query`): every entry (the past ones too), by start, and `playing`: the set,
  since when, and the entry that plays (empty when none runs).
- **`AddEntry`**, **`EditEntry`**, **`DeleteEntry`**: an entry that cannot be scheduled is `InvalidArgument`, a set a
  module lacks or an entry that started `FailedPrecondition`, an unknown id `NotFound`.

## Tests

The rule is pinned by `schedule_test.go` (priorities, ties, the end, since when, two entries of one set) and
`entry_test.go` (only an entry to come changes, every reason). `schedule.StoreContractSuite` runs on the memory and
postgres stores. The runner's test pins a send at boot and at each change, none while the turn holds, and a send Redis
refused sent again. Each module's `play_rule_set_handler` test pins the latest decision, a redelivery and an unknown
set; the choice stores have their own contract suite (`rules.ChoiceStoreContractSuite`). The smoke test
`TestAnEntryOfTheScheduleIsPlayedByPlanetAndTheAntibot` adds an entry on the console listener and waits for both
modules to play it.

## Before the Final Battle (Sat 31 Oct 2026, 21:00 UTC)

Nothing plays `rush` until an entry names it. Before 21:00, on the backoffice:

1. Ops → Game rules: Copy `default`, name it `rush`, tempo 3, box waits 60–120 s, 30 an hour. Save.
2. Ops → Anti-bot rules: Copy `default`, name it `rush`, `cohort.certainMembers` 50. Save.
3. Ops → Rule schedule: Add an entry: `rush`, from 31 Oct 21:00 UTC to 23:00 UTC, priority 1. It is refused until both
   sets exist.

Once `rush` plays, neither module saves `rush`; an edit is a new set and a new entry.
