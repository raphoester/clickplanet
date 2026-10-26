# The antibot's rules (`antibot.rule_sets`)

**Every bound the antibot judges with is a rule, kept in postgres and edited from the backoffice** (Ops → Anti-bot
rules), each watchdog's on/off switch included. The YAML `antiBot` block keeps the machine only. Until 2026-10-26 every
bound was in the YAML, read once when each watchdog was built, so a change was a deploy and a restart. For the Final
Battle the operator needs a looser crowd for one evening, from the page.

## A rule set

**A set is a name and every rule value, all written.** No value falls back to another set, to a default in Go, or to
one in SQL. Its snapshot is the wire's message: `rules.Snapshot{Name, Rules *antibotv1.Rules}`, which postgres keeps
as a document, the operator's calls carry and the log lines print as it is. `Snapshot.Restore(parts)` is the one check.

- **Each part is read by the package that judges with it.** A watchdog, the jury and the ban book each own their
  `Rules`, its `Validate`, its `RulesOf(*antibotv1.XRules)` and a `Part` (`rules.PartOf[Rules]{Key, Read}`) that takes
  it out of the set's message. The `rules` package names none of them.
- **`guard.Parts()` is the one list of parts**, beside the code that builds every watchdog. A new watchdog adds its
  field to `antibot.v1.Rules`, its package, and one line there; nothing in `rules`, the store or the handlers changes.
- **`Restore` reads and validates every part**, and keeps each part's `Rules`, so a read never converts again. It
  refuses a set any part refuses, with every reason, each named by its path as `antibot.v1.Rules` writes it:
  `cohort.certainMembers is 1, under minMembers (2); rotator.maxShare is 0, and must be above 0`. A part left out of
  the message reads as zeros, which its `Validate` refuses. Each `Validate` refuses what the watchdog's `withDefaults`
  used to repair: a zero that meant "the default", a certain bound under its suspect bound, a sample count over what
  the watchdog keeps.
- **A zero that never reads is a rule of its own** where it was one before: `defender.minShare`,
  `retaker.minTiles`, `rotator.minSlots`, the churner's bounds, `stamina.minBusy`. The four bounds of `shape` and `clock`
  are unset rather than zero (`optional` in the proto, a pointer in Go): 0 is a skew.
- **The name** is a lowercase letter, then up to 31 of a-z, 0-9, `-` and `_`: the names planet's rule sets take.
- **`default`** is seeded by migration `20261026230000_rule_sets` with production's values of that day:
  `deploy/vps/backend.yaml`, plus what each watchdog's `withDefaults` filled in for the keys it left out (the catcher's
  `foreign.window` 1h, the scraper's `certainOffMap` 2, the clock's 1s, 120, 600 and 30m), and every watchdog on.
  `TestTheMigrationSeedsDefaultWithProductionsRules` compares it with `rules.Seeded()`, the fixture the tests share.

### The rules, by part

- `bans.enforce` — off judges, logs and counts, and tells planet to refuse nobody; the mode to deploy in
- `bans.banDurations` — the ban for each offence (the last step repeats). An offence is a ban that starts while none is
  running; a flag on a running ban only extends it. **Offences are never forgotten**, but for the one an operator lifts
  with `UnbanPlayer`
- `bans.reflagInterval` — how soon a banned caller can be judged again
- `jury.minSuspects` — how many watchdogs at `suspect` make a ban; one at `certain` bans alone; `stamina` and `defence`
  together count as one
- `jury.suspicionWindow` — how long a verdict stands while another watchdog catches up
- `<watchdog>.enabled` — off, the jury reads the watchdog as clear (see [A watchdog turned off](#a-watchdog-turned-off))
- `retaker.reactionWindow`, `minReactions`, `maxSpread`, `maxMedian` — what counts as a reaction, how many are needed,
  and the band that reads `suspect` then `certain`; `minTiles`, `roamMedian`, `certainTiles` — the `roam` rule
- `sequencer.minSteps`, `minShare`, `certainSteps`, `certainShare` — how long a run of constant-stride clicks must be,
  and how much of it must sit at that stride
- `metronome.maxGap`, `maxSpread`, `minClicks`, `certainFor`, `certainClicks` — what ends a run, how tight its gaps
  must be, and how long it must hold; `shape.maxGap`, `clicks`, `maxSkew`, `certainClicks`, `certainSkew` — the longest
  gap sampled, and the skew of the last gaps that reads each level (unset, it only measures); `clock.period`, `clicks`,
  `minCoherence`, `certainClicks`, `certainFor`, `certainCoherence` — the beat and the coherence that reads each level
  (unset, it only measures); `stamina.clicks`, `window`, `minBusy`, `certainBusy` — what makes a slice busy, and the
  busy time inside the window that reads each level (zero never reads)
- `defender.retakeWindow`, `minClicks`, `minShare`, `certainClicks`, `certainShare` — what counts as a retake, and the
  share of takes that reads `suspect` then `certain`; a zero share never reads
- `catcher.minCatches`, `maxMedian`, `certainMedian` — how many boxes in a row must all be caught, and the median
  offer-to-claim delay that reads `suspect` then `certain`; `foreign.window`, `minClaims`, `certainClaims` — the claims
  of boxes offered to another scope (zero never reads)
- `cohort.startWindow`, `minClicks`, `minFlagShare`, `rateRatio`, `lengthRatio`, `quietAfter`, `minMembers` — what makes
  two scopes in step, and how many of them read `suspect`; `certainCohorts`, `certainMembers` (the crowd),
  `chainWindow` — how many groups, or scopes in one group, read `certain`
- `scraper.minMaps`, `certainMaps` — the whole maps read beyond one per stream opened that read `suspect` then
  `certain`; `certainOffMap` — the reads off the map that read `certain` however little was read
- `churner.window`, `v6`, `v4` — the guest accounts born on one scope that read each level; `relay.handoff`, `maxLife`,
  `minClicks`, `minFlagShare`, `minLinks`, `certainLinks` — the takeovers
- `hopper.minAngle`, `minGap`, `maxGap`, `minSteps`, `minShare`, `certainSteps`, `certainShare` — what a hop is, and the
  share of hops that reads each level
- `rotator.window`, `minSeconds`, `minByChance`, `maxShare` — how far back slots count, the seconds each account must
  click in for a pair to share a slot, the shared seconds chance must have given, and the share a rotating pair stays
  under; `minSlots`, `certainSlots` — the shared slots that read each level (the jury counts neither)

## Where it is kept

**One row per set, the rules as one `jsonb` document**: `antibot.rule_sets(name, rules)`, the document an
`antibot.v1.Rules` as protojson writes it, its zeros too (`EmitUnpopulated`). About 85 values in a dozen parts would be
a column each, and a migration for every rule added. The proto message is already the wire's, so the store, the RPC and
the page read one shape.

- **The store answers sets, never snapshots** (`rules.Store`: `Set(ctx, name) (Set, error)`, `Save(ctx, Set)`).
  `postgres_rule_set_store` is built with `guard.Parts()`: it finds a set by name and restores it, and upserts a set's
  snapshot. The snapshot never leaves the store. A set kept that cannot play, after a hand edit in SQL or a rule added
  with no migration, is an error that wraps `rules.ErrInvalidSet` and names the set. `rules.MemoryStore` is the fake
  (its `Keep` writes a set that cannot play), and `rules.StoreContractSuite` runs on both. A key the proto no longer has
  is ignored on read.
- **A rule added later needs a migration** that writes its value into every set (`jsonb_set`). Without one, the set
  reads the value as zero; where zero cannot be judged with, the set no longer restores, the antibot keeps the rules
  in force, and the next boot is refused.

## The rules in force

**The antibot holds the set it plays in one `atomic.Pointer` (`rules.InForce`).** Each watchdog, the jury and the ban
book are built with `rules.Reader(inForce, <package>.Part)`, a `func() Rules` that reads their part of the set in force
each time it is called. They call it when they judge, not when they are built: a set taken applies from the next click.
`TestASetTakenJudgesFromTheNextClick` pins it through the guard. A watchdog reads its rules once per call, so one
judgement sees one set.

- **`play_rules_usecase` plays when something changes, never on a timer**: at boot (`Execute`) the set the schedule
  chose last, and `default` until it chose one; then each `antibot.v1.PlayRuleSet` the schedule sends (`Play`, from
  `commands/play_rule_set_handler`). A save never changes the rules in force (see [The operator](#the-operator)), so
  it plays nothing. Each time it reads the set to play from the store, which restores it, and enforces it when its
  values changed (`Set.Equal`), one play at a time under a lock. A missing set at boot refuses the boot. A failed read,
  or a kept set that cannot play, keeps the rules in force; the caller gets the error and `log_play_rules` logs it.
  Until 2026-10-26 a runner read postgres every second; with the schedule as the only thing that changes the set in
  force, a push is enough, and a hand edit in SQL takes effect at the next play or boot.
- **The choice is kept** in `antibot.rule_set_choice` (`rules.Choice`, one row, `postgres_choice_store`): a later
  decision overrules it, and an older one or one sent again changes nothing. A set the antibot keeps none of is
  refused: the rules in force stay, the choice is not kept, and the command is dropped.
- **Each set taken tells planet every running ban again** (`bans.Book.Restate`), under the set's `bans.enforce`: a set
  that stops enforcing admits the banned callers at once, and one that enforces refuses them again. The first take is at
  boot, before the antibot hears anything, so the bans running at a deploy reach planet's table as before. A failed
  restatement is tried again at the next play: the redelivery of the command, or the next boot.
- **`log_play_rules` logs each set taken** at Warn (`antibot rules in force`, `set`, `rules` as protojson), and each
  failure or refusal at Error, and the start line (`antibot enabled`) names the set and the watchdogs on.

## A watchdog turned off

**Every watchdog is built, and `<watchdog>.enabled` decides whether its reading counts.** Off, the jury reads it as
clear: no rise, no suspect, no certain, and `InspectPlayer` shows `clear`. It keeps watching all the same: every try,
take, box and map read still reaches it, its sweep still feeds its histogram, and its evidence is saved. So turning it
on again reads at once, on full windows, and memory is what it was when all ten were on. Until 2026-10-26 a watchdog
off in the YAML was never built. `TestAWatchdogTurnedOffKeepsWatchingAndCountsForNothing` pins it.

## Which set plays

**The schedule picks it**, in the metagame module, and commands it in the antibot's words (`antibot.v1.PlayRuleSet`, see
[The metagame module](metagame.md)). The antibot was to follow the name of planet's set; now each module is told
on its own, so planet never hears of a crowd, and the antibot never follows planet.

- **The antibot has no switch of its own for which set plays.** No operator call picks the set in force.
- **`antibot.v1.InternalService/GetRuleSetNames`** answers the names of the sets postgres keeps, so the metagame module
  refuses an entry of a set the antibot lacks.
- **For the Final Battle** (Sat 31 Oct 2026, 21:00–23:00 UTC), an entry plays `rush`. An antibot set `rush`
  with `cohort.certainMembers` 50 makes the crowd bound 50 for the evening. It must be saved before 21:00: while `rush`
  plays, a save of `rush` is refused, and an antibot with no `rush` refuses the command and keeps `default`.

## The machine, in the YAML

The config keeps what bounds memory and the keys that shape the evidence, one block per watchdog with no `detector`
level since the switches left:

- `enabled` (the module), `database`, `redis`;
- `evidence.saveInterval` and `retention` (also how long an unban's pardon lasts);
- every `trackWindow` and `sweepInterval`, `jury`'s included. The retaker, the sequencer, the defender, the scraper and
  the hopper also read their `trackWindow` as the window their rule counts over, so it is changed by a deploy. Where a
  rule's window must fit inside it, the sweep keeps the longer: the cohort keeps `chainWindow`, the churner
  `window + relay.handoff`;
- `metronome.stamina.slice` and `rotator.slot`: the evidence is counted per slice or slot and saved with its length,
  and a section saved under another length is dropped;
- `cohort.v4Bits`/`v6Bits` and `churner.relay.v4Bits`/`v6Bits`: read when a scope or an account is first seen, to index
  it by its wider prefix. A live change would split one network across two indexes for a `trackWindow`.

`TestEveryAntibotKeyIsOneTheConfigReads` decodes each YAML file's `antiBot` block with unused keys refused: a rule left
in a file would be read by nothing.

## The operator

`antibot.v1.AdminService` (see [Operator tools](admin.md)):

- **`GetRules`** answers every set postgres keeps, `default` first, as kept (one that cannot play too, so it can be
  mended), and `playing`, the name of the set in force. It is a query (`rule_sets_query`) beside the name `InForce`
  holds: it reads the documents, not the store, which refuses a set that cannot play.
- **`SaveRuleSet{set}`** makes a set or replaces its values. A set `Restore` refuses is `InvalidArgument`, with every
  reason. `audit_save_rule_set` logs every save at Warn (`admin antibot rule set saved` or `refused`, with the rules).
- **No operator call changes the rules in force.** What plays is the schedule, with its priorities; an operator
  never overrides it, here or anywhere (the user's rule, 2026-10-26). So `SaveRuleSet` refuses, with
  `FailedPrecondition`, the set that plays now (`ErrPlaying`). To change the live rules, the operator saves another
  set and the schedule plays it. `play_rules_usecase.Plays(name)` answers it.
