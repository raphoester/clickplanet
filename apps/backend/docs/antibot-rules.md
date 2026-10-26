# The antibot's rules (`antibot.rule_sets`)

**Every bound the antibot judges with is a rule, kept in postgres and edited from the backoffice** (Ops → Anti-bot
rules). The YAML `antiBot` block keeps the machine only. Until 2026-10-26 every bound was in the YAML, read once when
each watchdog was built, so a change was a deploy and a restart. For the Final Battle the operator needs a looser crowd
for one evening, from the page.

## A rule set

**A set is a name and every rule value, all written.** No value falls back to another set, to a default in Go, or to
one in SQL. `rules.Set` holds them (snapshot pattern: `Snapshot.Restore` is the one check), one value object per part,
each owned by the package that judges with it: `bans.Rules`, `jury.Rules`, and one `Rules` per watchdog.

- **`Restore` refuses a set any part refuses**, with every reason, each named by its path as `antibot.v1.Rules` writes
  it: `cohort.certainMembers is 1, under minMembers (2); rotator.maxShare is 0, and must be above 0`. Each part's
  `Validate` refuses what the watchdog's `withDefaults` used to repair: a zero that meant "the default", a certain bound
  under its suspect bound, a sample count over what the watchdog keeps.
- **A zero that never reads is a rule of its own** where it was one before: `defender.minShare`,
  `retaker.minTiles`, `rotator.minSlots`, the churner's bounds, `stamina.minBusy`. The four bounds of `shape` and `clock`
  are unset rather than zero (`optional` in the proto, a pointer in Go): 0 is a skew.
- **The name** is a lowercase letter, then up to 31 of a-z, 0-9, `-` and `_`: the names planet's rule sets take.
- **`default`** is seeded by migration `20261026230000_rule_sets` with production's values of that day:
  `deploy/vps/backend.yaml`, plus what each watchdog's `withDefaults` filled in for the keys it left out (the catcher's
  `foreign.window` 1h, the scraper's `certainOffMap` 2, the clock's 1s, 120, 600 and 30m).
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
`antibot.v1.Rules` as protojson writes it, its zeros too (`rulesmessage.Document`). About 85 values in a dozen parts
would be a column each, and a migration for every rule added. The proto message is already the wire's, so the store,
the operator's call and the backoffice read one shape.

- `postgres_rule_set_store` finds a set by name and upserts one; `rules.MemoryStore` is the fake, and
  `rules.StoreContractSuite` runs on both. A key the proto no longer has is ignored on read.
- **A rule added later needs a migration** that writes its value into every set (`jsonb_set`). Without one, the set
  reads the value as zero; where zero cannot be judged with, the set no longer restores, the runner keeps the rules in
  force, and the next boot is refused.
- `rulesmessage` maps a set to and from `antibot.v1.Rules`, for the store, the handlers and the log lines.

## The rules in force

**The antibot holds the set it plays in one `atomic.Pointer` (`rules.InForce`).** Each watchdog, the jury and the ban
book are built with a reader of their own part (`inForce.Retaker`, `inForce.Jury`, `inForce.Bans`…) and read it when
they judge, not when they are built: a set taken applies from the next click. `TestASetTakenJudgesFromTheNextClick` pins
it through the guard. A watchdog reads its rules once per call, so one judgement sees one set.

- **`play_rules_usecase` is level-triggered**: every second (`antibot-rules`, a runner closed before the pool), and once
  at boot, it reads the set to play and enforces it when its values changed (`Set.Equal`). A missing `default` refuses
  the boot. A failed read keeps the rules in force, and `log_play_rules` logs it.
- **Each set taken tells planet every running ban again** (`bans.Book.Restate`), under the set's `bans.enforce`: a set
  that stops enforcing admits the banned callers at once, and one that enforces refuses them again. The first take is at
  boot, before the antibot hears anything, so the bans running at a deploy reach planet's table as before. A failed
  restatement is tried again at the next tick.
- **`log_play_rules` logs each set taken** at Warn (`antibot rules in force`, `set`, `asked`, `rules` as protojson), and
  the start line (`antibot enabled`) names the set.

## Which set plays

**The antibot plays the set named like the planet rule set in force, and `default` when it keeps no set of that
name.** The antibot is downstream of planet: it takes planet's word for which set plays and keeps its own values under
it, so planet never hears of a crowd.

- **`play_rules_usecase.UseCase.Play(name)`** is the entry point: it keeps the name, and the next tick plays it. On
  `main`, planet does not tell its set yet, so the name stays `default`. The subscriber that hears
  `planet.v1.RuleSetEnforced` (planet's rule sets, PR #475) calls `Play` with its `rule_set`.
- **For the Final Battle** (Sat 31 Oct 2026, 21:00–23:00 UTC), planet plays `rush`. An antibot set `rush` with
  `cohort.certainMembers` 50 makes the crowd bound 50 for the evening; until it is made, the antibot keeps `default`.

## The machine, in the YAML

The config keeps what decides what runs and what bounds memory, and the keys that shape the evidence:

- `enabled`, each watchdog's `enabled`, `database`, `redis`;
- `evidence.saveInterval` and `retention` (also how long an unban's pardon lasts);
- every `trackWindow` and `sweepInterval`, `jury`'s included. The retaker, the sequencer, the defender, the scraper and
  the hopper also read their `trackWindow` as the window their rule counts over, so it is changed by a deploy. Where a
  rule's window must fit inside it, the sweep keeps the longer: the cohort keeps `chainWindow`, the churner
  `window + relay.handoff`;
- `metronome.detector.stamina.slice` and `rotator.detector.slot`: the evidence is counted per slice or slot and saved
  with its length, and a section saved under another length is dropped;
- `cohort.detector.v4Bits`/`v6Bits` and `churner.detector.relay.v4Bits`/`v6Bits`: read when a scope or an account is
  first seen, to index it by its wider prefix. A live change would split one network across two indexes for a
  `trackWindow`.

`TestEveryAntibotKeyIsOneTheConfigReads` decodes each YAML file's `antiBot` block with unused keys refused: a rule left
in a file would be read by nothing.

## The operator

`antibot.v1.AdminService` (see [Operator tools](admin.md)):

- **`GetRules`** answers every set postgres keeps, `default` first, as kept (one that cannot play too, so it can be
  mended), and `playing`, the name of the set in force. It is a query (`rule_sets_query`) beside the name `InForce`
  holds.
- **`SaveRuleSet{set}`** makes a set or replaces its values. A set `Restore` refuses is `InvalidArgument`, with every
  reason. `audit_save_rule_set` logs every save at Warn (`admin antibot rule set saved` or `refused`, with the rules).
  Saving the set in force takes effect within a second.
