# The season rules (`scoring.rule_sets`)

**How scoring counts points while a set plays is a rule set, kept in postgres and edited from the backoffice** (Ops →
Season rules), as planet's game rules (see [Rules](rules.md)) and the antibot's bounds (see
[The antibot's rules](antibot-rules.md)). Which set plays is the gamemaster's schedule (see
[The gamemaster module](gamemaster.md)): scoring sits under it as planet and the antibot do (the user, 2026-10-27), and
the gamemaster never hears of a point. Until 2026-10-27 the Final Battle was a round of scoring's calendar (`finale: 2h`)
and its ×3 a constant, while the battle's look and rules were a schedule entry: two clocks for one battle.

## A rule set

**A set is a name and every rule value, all written**: `rules.Snapshot{Name, Rules *scoringv1.Rules}`, which postgres
keeps as protojson, the operator's calls carry and the log lines print. `Snapshot.Restore` is the one check, and names
each value it refuses by its path (`ownRound.times is 0, and must be 1 to 10`). A value left out reads as zero, which
it refuses; nothing falls back to a default. The name is a lowercase letter, then up to 31 of a-z, 0-9, `-` and `_`.

- `ownRound.enabled` — while the set plays, **the window of the schedule entry that plays it is a round of its own**:
  scored from the map at its end, as a day is, beside the days (see [Scoring](scoring.md)). The round that ends with
  the season is its finale: the Final Battle.
- `ownRound.times` — how many times a day's points the round of its own scores, 1 to `rules.MaxTimes` (10), so a slip
  of the keyboard cannot decide a season. Written when the round is off too.

**`default`** is seeded by migration `20261027200000_rule_sets`: days only (`{"ownRound": {"enabled": false,
"times": 1}}`). A battle's set is a copy of it, with the round on and its times.

**The store answers sets, never snapshots** (`rules.Store`), as planet's: `postgres_rule_set_store` restores what it
reads, and a kept set that cannot play is `ErrInvalidSet`. `rules.MemoryStore` is the fake, and `rules.StoreContractSuite`
runs on both.

## Playing it

**`play_rules_usecase` plays the set the schedule chose last, and `default` until it chose one** (`Execute`, at boot),
then each `scoring.v1.PlayRuleSet` the gamemaster sends (`Play`, from `commands/play_rule_set_handler`):

- **The command carries the entry's window** (`starts_at`, `ends_at`; both unset when `default` plays). A running
  entry's window never changes, so the round of its own is exactly the entry, and its end is known from its start:
  the race shows it, and the runner closes it on time.
- **The latest decision wins** (`rules.Choice.Overrules`): a command decided at or before the one kept changes
  nothing. The choice, its window with it, is kept in `scoring.rule_set_choice` (one row, `postgres_choice_store`).
- **A set with a round of its own opens it before the choice is kept** (`open_own_round_usecase`, through the use
  case's `Opener` port): a round that failed to open fails the handler, Redis delivers the command again, and it
  opens then. Opened twice, it is one round; a window heard of once it ended opens none. Opening it wakes the runner
  that closes the rounds. The same set played by two entries is two rounds, one per window.
- **At boot** the kept choice plays again, and opens its round again (one round still). A missing `default` refuses
  the boot.
- **A set scoring keeps none of, or one that cannot play, is refused**: the set in force stays, the choice is not
  kept, the log says so at Error, and the command is dropped. Scoring then opens no round of its own for the entry.
- `log_play_rules` logs each set taken at Warn (`season rules in force`, `set`, `rules`, and the window).

Nothing else reads the set in force: a day is always worth 1, and a round of its own keeps its times from when it
opened.

## Which set plays

**The schedule picks it**, as for planet and the antibot. `scoring.v1.InternalService/GetRuleSetNames` answers the
names of the sets postgres keeps, so the gamemaster refuses an entry of a set scoring lacks. An entry saved before scoring
kept sets may name one it lacks: the backoffice's Rule schedule page says so on the entry.

## The operator

`scoring.v1.AdminService`, on the console and admin listeners (see [Operator tools](admin.md)), and the backoffice's
Season rules page (see the backoffice's [Ops](../../backoffice/docs/ops.md)):

- **`GetRules`** answers every set postgres keeps, `default` first, as kept, and `playing`, the set in force
  (`rule_sets_query`).
- **`SaveRuleSet{set}`** makes a set or replaces its values. A set `Restore` refuses is `InvalidArgument`, with every
  reason; **the set in force is refused** with `FailedPrecondition` (`ErrPlaying`): to change what counts now, the
  operator saves another set and the schedule plays it. `audit_save_rule_set` logs every save at Warn
  (`admin season rule set saved` or `refused`).

The rules are pinned by `set_test.go`, the store and choice contract suites (memory and postgres) and the seed's test,
`play_rules_usecase`'s (default at first, a boot that opens its round again, a window per entry, a decision sent again,
a set refused, a round that failed to open), the handlers', and the smoke test
`TestAnEntryOfTheScheduleIsPlayedByPlanetTheAntibotAndSeasons`.
