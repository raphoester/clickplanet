# Rules: the game's rule sets (`planet.rule_sets`)

**Every game rule is kept in postgres, not in the config**, and edited from the backoffice (Ops → Game rules). A game
rule is what a player feels: how fast a bank refills, what a box holds, how big a bomb is. Until 2026-10-26 they were
in the YAML, read once at boot, so a change was a deploy and a restart. The design is the antibot's (see
[The antibot's rules](antibot-rules.md)); **a rule set holds planet's words only**: the antibot keeps its own sets.
Which set each plays is the schedule's (see [The gamemaster module](gamemaster.md)).

## A rule set

**A set is a name and every rule value, all written.** No value falls back to another set, to a default in Go, or to
one in SQL. Its snapshot is the wire's message: `rules.Snapshot{Name, Rules *planetv1.Rules}`, which the store keeps,
the operator's calls carry and the log lines print as it is. `Snapshot.Restore(parts)` is the one check.

- **Each part is read by the package that plays it.** `clicks` owns two parts, `clicks` (`Throttle`) and `toll`
  (`TollSteps`); `bonuses` owns `bonuses` (`Rules`). Each part has its `Validate`, its `…Of(message)` (`message.go`)
  and a `Part` (`rules.PartOf[R]{Key, Read}`). The `rules` package names none of them.
- **`parts.All()` is the one list of parts** (`rules/parts`). A concept that reads the rules adds its field to
  `planet.v1.Rules`, its part, and one line there; nothing in `rules`, the store or the handlers changes.
- **`Restore` reads and validates every part**, and keeps each part's rules, so a read never converts again. It
  refuses a set any part refuses, with every reason, each named by its path as `planet.v1.Rules` writes it:
  `clicks.burst is 0, under one click (1); bonuses.quiz.answerWindow is 0s, and must be above 0`. A part left out of
  the message reads as zeros, which its `Validate` refuses.
- **The name** is a lowercase letter, then up to 31 of a-z, 0-9, `-` and `_`.
- **`default`** is seeded by migration `20261026210000_rule_sets` with production's values of that day, the ones
  `deploy/vps/backend.yaml` held. `TestTheMigrationSeedsDefaultWithProductionsRules` compares it with `rules.Seeded()`,
  the fixture the tests share (`rules.Named(name, change)` changes it).

### The rules, by part

- `clicks.tempo` — how many times faster than `clicks.perSecond` every bucket refills (1)
- `clicks.perSecond`, `clicks.burst` — one account's refill and bank (0.2, one click every 5 s; 60). See
  [Rate limiting](rate-limiting.md)
- `clicks.scopeMultiplier`, `v6ScopeMultiplier`, `guestScopeMultiplier` — the address's bucket, an IPv6 /64's, and the
  one every guest behind an address shares, in accounts (10, 4, 1)
- `clicks.linkedMultiplier` — how many times as fast a signed-in account refills (2)
- `clicks.newAccountClicks` — what a new account's bank holds; it earns the rest at `clicks.perSecond` (10)
- `toll.steps` — rising `{share, slowdown}`: from `share` of the map, a country refills `slowdown` times slower (10%
  1.5×, 20% 2.5×, 30% 4×). No step is no toll. See [A big country refills slower](rate-limiting.md#a-big-country-refills-slower-clickstoll)
- `bonuses.boxes.minInterval`, `maxInterval`, `maxChargesPerHour` — a box's wait and the hourly cap (4–8 min, 12)
- `bonuses.kinds.*` — what a box is drawn as: a weight per kind, `refill`, `spreadClicks`, `encloseClicks`, `bomb`,
  `shields` (5, 3, 2, 1, 3). 0 is never offered
- `bonuses.offerTtl` — how long a box flies; **it must outlast the flight the client draws**, or a box caught on its
  last frame is refused (15 s)
- `bonuses.missRetry` — a box missed brings the next one this soon, once (2 min)
- `bonuses.activeWithin` — no box and no quiz for a caller that has not clicked for this long (2 min)
- `bonuses.spread.clicks`, `.maxPerBox` — the spread pool and what one box adds at most (8, 4). See [Spread](spread.md)
- `bonuses.bomb.rings` — the blast's radius in tile spacings (4). See [Bomb](bomb.md)
- `bonuses.enclose.maxTiles`, `.held`, `.maxPerBox` — the biggest shape, the stack, a box's most (25, 3, 3). See
  [Enclose](enclose.md)
- `bonuses.shield.held`, `.minPerBox`, `.maxPerBox`, `.perTile` — the stack, a box's least and most, and a tile's most
  (30, 5, 20, 10; a tile keeps them in a byte, so 255 at most). See [Shields](shields.md)
- `bonuses.welcome.within`, `.boxAfter` — how young an account is welcomed, and when its first box comes (10 min,
  1 min); `0s` is off. See [A newcomer's first box and first quiz](welcome-bonus.md)
- `bonuses.signInRefill` — a guest's first sign-in earns a refill (on). See
  [A guest's first sign-in earns a refill](sign-in-refill.md)
- `bonuses.quiz.enabled`, `minInterval`, `maxInterval`, `maxChargesPerHour`, `offerTtl`, `answerWindow`, `missRetry`,
  `leaderBias` — the quizzes (on, 4–8 min, 12, 25 s, 8 s, 2 min, 1). See [Quizzes](quizzes.md)
- `frozen` — the map takes no play (off). See [A frozen map](#a-frozen-map)

## A frozen map

**A set with `frozen` takes no play** (`clicks.Freeze`, part `frozen`): while it is in force,
`planetv1controller.NewFreezeInterceptor` refuses Click, DropBomb, PlaceShield, UseRefill, ClaimBonus, OpenQuiz and
AnswerQuiz with `FailedPrecondition` and a `planet.v1.FrozenRefusal` detail (`clicks.ErrFrozen`), before any of them
spends a click or a charge. The map still loads and streams; tile cards, the board and PledgeAllegiance go on, and so
do the operator's tools. A freeze is played like any set: the schedule picks it (a plain entry over the break between
two seasons, see [Before the Final Battle](gamemaster.md#before-the-final-battle-sat-31-oct-2026-2100-utc)), so no
operator call freezes the map at once.

- **The client learns it from planet, the module that refuses**: `GetBudget` answers `ClickBudget.frozen` (only
  GetBudget sets it), and a set taking over sends `rules_changed`, so an open tab reads it again within a second.
- **Boxes and quizzes are offered on a timer** to a caller who clicked within `bonuses.activeWithin`, so one may still
  be offered in the first minutes of a freeze: claiming or answering it is refused, and it lapses.
- **The antibot needs a set of the same name** for the entry to be scheduled (a copy of `default` does).

## Where it is kept

**One row per set, the rules as one `jsonb` document**: `planet.rule_sets(name, rules)`, the document a
`planet.v1.Rules` as protojson writes it, its zeros too (`EmitUnpopulated`). About forty values, a list (the toll) and
a map (the kinds) would be forty columns and a child table. The proto message is already the wire's, so the store,
the RPC and the page read one shape.

- **The store answers sets, never snapshots** (`rules.Store`, the user's rule of 2026-10-26).
  `postgres_rule_set_store` is built with the parts: it reads a document into a snapshot and restores it, so a kept
  set that cannot play is `ErrInvalidSet` with every reason; it saves a set's snapshot. The snapshot never leaves the
  adapter. `rules.MemoryStore` is the fake (`Keep` puts in a snapshot as a hand edit in SQL would), and
  `rules.StoreContractSuite` runs on both. A key the proto no longer has is ignored on read.
- **A rule added later needs a migration** that writes its value into every set (`jsonb_set`). Without one, the set
  reads the value as zero; where zero cannot play, the set no longer restores, and the next boot is refused.

## The rules in force

**Planet holds the set it plays in one `atomic.Pointer` (`rules.InForce`).** Each part is handed out by
`rules.Reader(inForce, <package>.Part)`, a `func() R` that reads its part of the set in force each time it is called.
The readers call it where they use the rules, not when they are built: a set taken applies from the next use.

- `clicks.ThrottlePart` — `clicks.Buckets` (the multipliers, `newAccountClicks`), `try_click` (`ClickTried.tempo`),
  and the click limiter: `cpratelimit.NewRuled` reads `Throttle.Policy(inForce.Since())`, its rate, bank and tempo.
- `clicks.TollPart` — `clicks.Toll`: the price of each click, and `GetBonusRules.toll_steps`.
- `bonuses.Part` — the box registry (waits, kinds, flight, miss, activity, welcome, quizzes, the quiz lean),
  `bonuses.Charges` (the stacks), `bonuses.Bombing` (the radius), `enclose_click`, `place_shield_usecase`,
  `clicks.Claiming` (a tile's most shields, for a fortify) and `GetBonusRules`.
- **`play_rules_usecase` plays when something changes, never on a timer**: at boot and after each save (`Execute`) the
  set the schedule chose last, and `default` until it chose one; then each `planet.v1.PlayRuleSet` the schedule sends (`Play`, from
  `commands/play_rule_set_handler`). A missing or broken set at boot refuses the boot. Each time it reads the set to
  play from the store and enforces it when its values changed (`Set.Equal`), one play at a time under a lock. A failed
  read, or a kept set that cannot play, keeps the rules in force, and `log_play_rules` logs it; each set taken is
  logged at Warn (`rules in force`, `set`, `rules` as protojson). A hand edit in SQL takes effect at the next play or
  boot.
- **The choice is kept** in `planet.rule_set_choice` (`rules.Choice`: the set and when the schedule decided it, one
  row, `postgres_choice_store`): a later decision overrules it, and an older one or one sent again changes nothing. A
  set planet keeps none of is refused, the choice is not kept, and the command is dropped.
- `InForce.Enforce` also stamps when the set took over (`Since`) and wakes each `Watch()`er: every planet stream sends
  `PlanetEvent.rules_changed`, and the web client reads `GetBudget` and `GetBonusRules` again, so an open tab shows a
  new refill and new sizes with no reload. `GetBonusRules` is a GET no cache keeps for that reason.

## What changes for what is already there

A value read where it is used applies from the next use. **A value that carries state has a rule of its own:**

- **A bucket's rate and size** (`cpratelimit.Limiter.follow`): at each call the limiter compares the policy with the
  one it applied. On a change it refills every bucket at the old policy up to when the new one took over
  (`Policy.Since`), then plays the new one. A smaller burst cuts every bank to it; a bigger one is earned, not handed
  over. A bucket follows its key's scale from its next take, so a new scope multiplier reaches a busy address at its
  next click.
- **The toll** is set by each click: a new step slows a player from its next click.
- **Charges held above a smaller stack** are kept: `Held.Granted` never takes from a stack, and a full or fuller stack
  is offered no box of its kind.
- **Shields on a tile above a smaller `perTile`** stay: a placement is refused (`ErrTileFull`) and a fortify adds none
  until the tile is under it.
- **A box or a quiz due later than the new longest wait** is brought in to a wait at the new pace, at the next sweep
  (the registry compares at each sweep, every second).
- **Quizzes switched on**: each caller's next quiz is a whole wait away, so a wait long past is not a quiz for
  everyone at once. **Switched off**: no quiz is asked; one asked already can still be opened and answered until it
  lapses. A newcomer is owed no welcome quiz while they are off.
- **A box or a banner in flight** keeps the expiry it was given; **an account welcomed** keeps its welcome.

## Which set plays

**The schedule picks it**, in the gamemaster module: the running entry with the highest priority, else `default` (see
[The gamemaster module](gamemaster.md)). No operator call picks the set in force (the user's rule, 2026-10-26: an
operator never overrides the schedule, here or anywhere). Its values change with a save: a save of the set in force
takes over at once (2026-10-28, see [The operator](#the-operator)). `planet.v1.InternalService/GetRuleSetNames`
answers the names of the sets postgres keeps, so the gamemaster module refuses an entry of a set planet lacks.

## The machine, in the YAML

The config keeps the machine: addresses, `database`, `redis`, secrets, `gameMap.maxIndex`, `tilesStorage.*`,
`ledger.*`, `ledgerStorage.*`, `vpnBlocklist.*`, `auth`, `rateLimiter.sweepInterval`, `bonus.forgetAfter` and
`bonus.sweepInterval` (see [Configuration](configuration.md)). `TestEveryPlanetKeyIsOneTheConfigReads` decodes each
YAML file's planet keys with unused keys refused: a rule left in a file would be read by nothing.

## The operator

`planet.v1.AdminService` (see [Operator tools](admin.md)):

- **`GetRules`** answers every set postgres keeps, `default` first, as kept (one that cannot play too, so it can be
  mended), and `playing`, the name of the set in force (`rule_sets_query` beside the name `InForce` holds).
- **`SaveRuleSet{set}`** makes a set or replaces its values: `save_rule_set_usecase` restores the snapshot the call
  carries, and hands the store the set. A set `Restore` refuses is `InvalidArgument`, with every reason. **Then it plays the set chosen last again**
  (`play_rules_usecase.Execute`, through `log_play_rules`): a save of the set in force takes over at once, and a save of
  another set changes nothing. A play that fails after the save is an error; the set stays kept and plays at the next
  play or boot.
  `audit_save_rule_set` logs every save at Warn (`admin rule set saved` or `refused`, with the rules).

The rules are pinned by each part's tests, `Set`'s (one refusal per impossible value), the store's contract suite and
its migration test, the limiter's policy tests, the registry's pace and quiz tests, `Held`'s, the use cases', the
handlers' and `InForce`'s; the e2e suite holds smoke tests only (see [Testing](testing.md)). Its stack migrates
planet's schema first (`planet.Migrate`, behind the `testing` tag) and writes its own `clicks` into `default` before
planet boots, since every test clicks from one address.
