# Rules: the game's rule sets (`planet.rule_sets`)

**Every game rule is kept in postgres, not in the config**, and edited from the backoffice (Ops → Game rules). A game
rule is what a player feels: how fast a bank refills, how many shields a hand holds, how big a bomb is. Until
2026-10-26 they were in the YAML, read once at boot, so a change was a deploy and a restart. The design is the
antibot's (see [The antibot's rules](antibot-rules.md)); **a rule set holds planet's words only**: the antibot keeps its
own sets, and so do the offers since 2026-11-11 (when a box and a quiz come, and what each grants: see
[The offers](bonuses.md#its-rule-sets)).
Which set each plays is the schedule's (see [The gamemaster module](gamemaster.md)).

## A rule set

**`default` holds every value; every other set holds only the values it changes, and plays default's for the rest**
(the user's rule, 2026-10-31). Until then a set held every value: a change to `default` reached no other set, and
each rule added took a migration into every set. Every value of `planet.v1.Rules` is `optional`, so a set can hold a
zero or a `false` of its own; nothing falls back to a default in Go or in SQL. Its snapshot is the wire's message:
`cprules.Snapshot[*planetv1.Rules]{Name, Rules}`, which the store keeps and the operator's calls carry.
`Snapshot.Restore(parts, defaults)` is the one check. The machinery is `cprules`, shared with the antibot and the offers
(see [Shared](shared.md)); planet keeps its parts, its bounds and its fixtures in `rules`.

- **What a set plays is default's values with its own written over them** (`cpproto.Overlay`). A message it holds is
  overlaid value by value. A list, a `Duration` and a message of lists only are one value each: a set that holds `toll`
  holds every step, and `toll: {}` is no toll. `default` plays alone, whatever `defaults` it is restored with.
- **`Set.TakeSnapshot()` is what the set holds**: what is kept and edited. `TakePlayedSnapshot()` is what it plays,
  which the log lines print. `Set.Equal` compares what two sets play.

- **Each part is read by the package that plays it.** `clicks` owns five parts, `clicks` (`Throttle`), `toll`
  (`TollSteps`), `frozen` (`Freeze`), `reach` (`Reaching`) and `capitals` (`CapitalRules`); `allegiances` owns
  `allegianceLocked` (`Lock`), `arenas` `arena` (`Rules`) and `seeds` `seed` (`Pattern`). Each part has its `Validate`, its
  `…Of(message)` (`message.go`) and a `Part` (`cprules.PartOf[*planetv1.Rules, R]{Key, Read}`). `cprules` names none
  of them.
- **`charges` is the wallet's part** (`charges.Part`, `planet.v1.ChargeRules`: `spread.clicks`, `.rings`, `.unlimited`, `bomb.rings`,
  `bomb.off`, `enclose.maxTiles`, `.held`, `.unlimited`, `shield.held`, `.perTile`). Until 2026-11-11 those were the
  wallet's half of `bonuses` (`planet.v1.BonusRules`), whose other half a second part read for the offers
  (`parts.Grants`, see [Offers and the wallet](offers-and-wallet.md#the-rules-split)). The offers keep that half in rule
  sets of their own now ([The offers](bonuses.md#its-rule-sets)); `Rules.bonuses` (3) is `reserved`, and the wallet's
  reasons are named `charges.…`. Migration `20261111200000_charges` wrote each set's wallet values under `charges`, and
  left `bonuses` in the stored sets, which planet reads no more, so the offers' own migration copies it whichever
  module migrates first; a later release strips it. Its down removes `charges`: a set saved since loses its wallet values.
- **`parts.All()` is the one list of parts** (`rules/parts`). A concept that reads the rules adds its field to
  `planet.v1.Rules`, its part, and one line there; nothing in `rules`, the store or the handlers changes.
- **A part may refuse a value in `default` alone** (`PartOf.InDefault`): `arena.on`, which every set would play.
- **`Restore` reads and validates every part of what the set plays**, and keeps each part's rules, so a read never
  converts again. It refuses a set any part refuses, with every reason, each named by its path as `planet.v1.Rules`
  writes it: `clicks.burst is 0, under one click (1); charges.spread.clicks is 0, and must be above 0`. A value
  neither the set nor `default` holds reads as zero, which its `Validate` refuses. A set's own value is checked against
  default's too, since what the set plays is checked whole.
- **The name** is a lowercase letter, then up to 31 of a-z, 0-9, `-` and `_`.
- **`default`** is seeded by migration `20261026210000_rule_sets` with production's values of that day, the ones
  `deploy/vps/backend.yaml` held. `TestTheMigrationSeedsDefaultWithProductionsRules` compares it with `rules.Seeded()`,
  the fixture the tests share (`rules.Named(name, change)` is a set that holds every value, `Seeded()`'s changed).

### The rules, by part

- `clicks.tempo` — how many times faster than `clicks.perSecond` every bucket refills (1)
- `clicks.perSecond`, `clicks.burst` — one account's refill and bank (0.2, one click every 5 s; 60). See
  [Rate limiting](rate-limiting.md)
- `clicks.scopeMultiplier`, `v6ScopeMultiplier`, `guestScopeMultiplier` — the address's bucket, an IPv6 /64's, and the
  one every guest behind an address shares, in accounts (10, 4, 1)
- `clicks.linkedMultiplier` — how many times as fast a signed-in account refills (2)
- `clicks.newAccountClicks` — what a new account's bank holds; it earns the rest at `clicks.perSecond` (10)
- `clicks.fastStart` — how long a new account refills at the signed-in pace, guest or not; `0s` is off (30 min, since
  migration `20261106120000_fast_start`). See [Rate limiting](rate-limiting.md)
- `toll.steps` — rising `{share, slowdown}`: from `share` of the map, a country refills `slowdown` times slower (10%
  1.5×, 20% 2.5×, 30% 4×). No step is no toll. See [A big country refills slower](rate-limiting.md#a-big-country-refills-slower-clickstoll)
- `charges.spread.clicks` — the spread pool a hand holds (8). See [Spread](spread.md)
- `charges.spread.rings` — how far a spread reaches: every tile within this many steps of the tile clicked, 1 to 3 (1:
  7 tiles; 2: 19; 3: 37). See
  [The disc](spread.md#the-disc-chargesspreadrings)
- `charges.spread.unlimited` — a click with spread on needs no charge and spends none (off). See
  [Unlimited spreads](spread.md#unlimited-spreads-chargesspreadunlimited)
- `charges.bomb.rings` — the blast's radius in tile spacings (4). See [Bomb](bomb.md)
- `charges.bomb.off` — no bomb: a bomb held is not dropped, and no hand has room for one, so no box or quiz offers one
  (off). See [Bomb](bomb.md)
- `charges.enclose.maxTiles`, `.held` — the biggest shape and the stack (25, 3). See [Enclose](enclose.md)
- `charges.enclose.unlimited` — a click with enclose on needs no charge and spends none (off). See
  [Unlimited enclosures](enclose.md#unlimited-enclosures-chargesencloseunlimited)
- `charges.shield.held`, `.perTile` — the stack and a tile's most (30, 10; a tile keeps them in a byte, so 255 at
  most). See [Shields](shields.md)
- **The offers' rules are not here since 2026-11-11**: a box's wait, its kinds and its flight, what a box or a right
  answer grants, the welcome, the quizzes and the sign-in refill are the bonuses module's own sets (see
  [Its rule sets](bonuses.md#its-rule-sets)). Until then they were `bonuses.*` here, and promo's cards still name them
  so (`{bonuses.boxes.minInterval}`, see [Promo](promo.md))
- `frozen` — the map takes no play (off). See [A frozen map](#a-frozen-map)
- `allegianceLocked` — no account changes its flag; a first flag is still taken (off; on in a battle's set). See
  [Allegiance](allegiance.md)
- `arena.on` — a battle that plays the set is an arena battle, played on a fresh map; the gamemaster reads it, planet plays no
  rule of it (off; **refused in `default`**, or every battle would be one: `PartOf.InDefault`). `arena.freeze` — how
  long an arena takes no play after its map plays and before the battle ends, 0 to 30 s (8 s), read from the battle's
  set as its arena opens. See [Arena battles](arenas.md)
- `seed` — how an arena's fresh map starts: one pattern of a `oneof`. `capitals {shields}`, each capital held by its
  country with that many shields, past `charges.shield.perTile`, 0 to 255 (100). Until 2026-11-10 it was
  `arena.capitalShields`. See [Seeds](arenas.md#seeds)
- `reach` — a take lands only next to the flag's land or across a bridge from it (off). See [The reach](reach.md)
- `capitals.knockOut`, `capitals.fixedShields` — a capital taken by another flag knocks its flag out; no shield goes on a
  capital (off, off). See [The capitals' rules](capitals.md)

## A frozen map

**A set with `frozen` takes no play** (`clicks.Freeze`, part `frozen`): while it is in force,
`planetv1controller.NewFreezeInterceptor` refuses Click, DropBomb, PlaceShield and UseRefill with `FailedPrecondition`
and a `planet.v1.FrozenRefusal` detail (`clicks.ErrFrozen`), before any of them spends a click or a charge, and
`InternalService/GetFreeze` answers frozen. The map still loads and streams; tile cards, the board and PledgeAllegiance go on, and so
do the operator's tools. A freeze is played like any set: the schedule picks it (a plain entry over the break between
two seasons, see [A freeze over the break](gamemaster.md#a-freeze-over-the-break-between-two-seasons)), so no
operator call freezes the map at once.

- **The client learns it from planet, the module that refuses**: `GetBudget` answers `ClickBudget.frozen` (only
  GetBudget sets it), and a set taking over sends `rules_changed`, so an open tab reads it again within a second.
- **The offers refuse their own calls** (ClaimBonus, OpenQuiz, AnswerQuiz) with a `bonuses.v1.FrozenRefusal` while
  their set is frozen, or while planet's `GetFreeze` answers frozen, one loopback call each: see
  [The freeze](bonuses.md#the-freeze). Their set of the same name holds `frozen` too (copied from planet's on
  2026-11-11), and a frozen offers' set offers nothing. A box or a quiz offered while planet alone is frozen is refused
  when taken, and lapses. Until 2026-11-11 a frozen planet still offered boxes on their timer and refused the claims.
- **The antibot and the offers need a set of the same name** for the entry to be scheduled (an empty one does: each
  plays default's).

## Where it is kept

**One row per set, the rules as one `jsonb` document**: `planet.rule_sets(name, rules)`, the values the set holds as
protojson writes a `planet.v1.Rules`: a value held is written, a zero too; a value left unset is not. A key present is a
value the set holds. About forty values and a list (the toll) would be forty columns and a child table. The proto
message is already the wire's, so the store, the RPC and the page read one shape.

- **The store answers sets, never snapshots** (`cprules.Store`, the user's rule of 2026-10-26).
  `cprules.PostgresStore` is built with the parts: `Set(name)` reads the set's document and default's and restores the
  one over the other, so a kept set that cannot play is `ErrInvalidSet` with every reason, and a set with no `default`
  kept is `ErrUnknownSet`; `Names` lists the sets kept; `Save` writes a set's snapshot. The snapshot never leaves the
  adapter. `cprules.MemoryStore` is the fake (`Keep` puts in a snapshot as a hand edit in SQL would), and
  `cprules.StoreContractSuite` runs on both, the postgres one on planet's migrations with planet's values
  (`rules/postgres_store_test.go`). A key the proto no longer has is ignored on read.
- **A rule added later needs a migration** that writes its value into `default` (`jsonb_set`); every other set plays
  it from there. Without one, default reads the value as zero; where zero cannot play, no set restores, and the next
  boot is refused.
- **Migration `20261031180000_rule_sets_inherit_default`** made every set but `default` hold only what it changes. A
  value equal to default's was dropped; a value it left out or kept as `null`, which it read as zero, was written as
  that zero. So each set played the same before and after; only a later change to `default` now reaches it. It also
  wrote `frozen: false` into `default`, written before the map could freeze. The down migration writes default's values
  back into each set, for a build from before.

## The rules in force

**Planet holds the set it plays in one `atomic.Pointer` (`cprules.InForce`).** Each part is handed out by
`cprules.Reader(inForce, <package>.Part)`, a `func() R` that reads its part of the set in force each time it is called.
The readers call it where they use the rules, not when they are built: a set taken applies from the next use.

- `clicks.ThrottlePart` — `clicks.Buckets` (the multipliers, `newAccountClicks`, `fastStart`), `try_click` (`ClickTried.tempo`),
  and the click limiter: `cpratelimit.NewRuled` reads `Throttle.Policy(inForce.Since())`, its rate, bank and tempo.
- `clicks.TollPart` — `clicks.Toll`: the price of each click, and `GetChargeRules.toll_steps`.
- `clicks.FreezePart` — the freeze interceptor (`planetv1controller.NewFreezeInterceptor`), `get_budget_usecase` and
  `InternalService/GetFreeze`, which the offers ask.
- `allegiances.LockPart` — `allegiances.Book` (a pledge and a click for another flag) and `allegiance_query`
  (`Allegiance.locked`).
- `charges.Part` — `charges.Charges` (the stacks, a grant capped at them, and the room `GetRoom` answers, a bomb while
  bombs are off included), `charges.Bombing` (the radius, and the refusal while bombs are off), `enclose_click`,
  `place_shield_usecase`, `clicks.Claiming` (a tile's most shields, for a fortify) and the rest of `GetChargeRules`.
- `arenas.Part` and `seeds.Part` — read from the set an arena is opened for, never from the set in force: `arenas.Setups`
  takes the freeze, which the opening keeps, and `seeds.Catalog` the seed (`cprules.PartOf.Of(set)`).
- `clicks.ReachPart` — `clicks.Reach` (`reach_click`, `spread_click`) and `GetChargeRules.reach`. The spread's rings
  are `charges.Part`'s, read by `spread_click` after the click.
- `clicks.CapitalsPart` — `clicks.CapitalPlay` in the tile storage (a shield on a capital, a fortify, the knock-out, the
  standing) and `GetChargeRules` (`knock_out`, `capital_shields_fixed`).
- **`cprules.PlayUseCase` plays when something changes, never on a timer**: at boot and after each save (`Execute`) the
  set the schedule chose last, and `default` until it chose one; then each `planet.v1.InternalService/PlayRuleSet` the
  schedule calls (`Play`, from `planetv1controller/play_rule_set_handler`); the same turn called again, with the same
  `decided_at`, is no newer and changes nothing (see [The gamemaster as a reconciler](gamemaster-reconciler.md)). A missing or broken set at boot refuses the boot. Each time it reads the set to
  play from the store and enforces it when its values changed (`Set.Equal`), one play at a time under a lock. A failed
  read, or a kept set that cannot play, keeps the rules in force, and `cprules.LoggedPlay` logs it; each set taken is
  logged at Warn (`rules in force`, `set`, `rules`: every value it plays, as protojson). A save of `default` changes
  what the set in force plays when it leaves the value unset, so it is enforced again. A hand edit in SQL takes effect
  at the next play or boot.
- **The choice is kept** in `planet.rule_set_choice` (`cprules.Choice`: the set and when the schedule decided it, one
  row, `cprules.PostgresChoices`): a later decision overrules it, and an older one or one sent again changes nothing. A
  set planet keeps none of, or one that cannot play, is refused (`FailedPrecondition`) and the choice is not kept; the
  schedule calls again each minute.
- `InForce.Enforce` also stamps when the set took over (`Since`) and wakes each `Watch()`er: every planet stream sends
  `PlanetEvent.rules_changed`, and the web client reads `GetBudget` and `GetChargeRules` again, so an open tab shows a
  new refill and new sizes with no reload. `GetChargeRules` is a GET no cache keeps for that reason. Until 2026-11-11 it
  was `GetBonusRules`, and also said `sign_in_refill`, which the offers' `GetGrantRules` says now.

## What changes for what is already there

A value read where it is used applies from the next use. **A value that carries state has a rule of its own:**

- **A bucket's rate and size** (`cpratelimit.Limiter.follow`): at each call the limiter compares the policy with the
  one it applied. On a change it refills every bucket at the old policy up to when the new one took over
  (`Policy.Since`), then plays the new one. A smaller burst cuts every bank to it; a bigger one is earned, not handed
  over. A bucket follows its key's scale from its next take, so a new scope multiplier reaches a busy address at its
  next click.
- **The toll** is set by each click: a new step slows a player from its next click.
- **Charges held above a smaller stack** are kept: `Held.Granted` never takes from a stack, and a full or fuller stack
  has no room for its kind (`GetRoom`), so the offers grant none.
- **Shields on a tile above a smaller `perTile`** stay: a placement is refused (`ErrTileFull`) and a fortify adds none
  until the tile is under it.
- **A box or a quiz due later than the new longest wait**, in the offers' sets, is brought in to a wait at the new pace, at the next sweep
  (the registry compares at each sweep, every second).
- **Quizzes switched on** (the offers' `quiz.enabled`): each caller's next quiz is a whole wait away, so a wait long past is not a quiz for
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
`ledger.*`, `ledgerStorage.*`, `vpnBlocklist.*`, `auth` and `rateLimiter.sweepInterval` (see
[Configuration](configuration.md)). The offers' `forgetAfter` and `sweepInterval` are the `bonuses:` block's since
2026-11-11, when planet's top-level `bonus:` block went. `TestEveryPlanetKeyIsOneTheConfigReads` decodes each
YAML file's planet keys with unused keys refused: a rule left in a file would be read by nothing.

## What another module asks

`planet.v1.InternalService`, on the internal listener: **`GetRuleSetNames`**, every set postgres keeps, for the
gamemaster's check of an entry and the promo module's cards, and **`GetRuleSet(name)`**, the values a set plays,
default's with the set's own written over them (`cpproto.Overlay`), for the words of the promo module's teaser cards
(see [Promo](promo.md)). NotFound for a set postgres does not keep. `rule_set_query` reads both rows in one statement
and overlays them; it does not check that the set can play. Its test pins a set over default, default alone and a set
not kept.

## The operator

`planet.v1.AdminService` (see [Operator tools](admin.md)):

- **`GetRules`** answers every set postgres keeps, `default` first, as kept (one that cannot play too, so it can be
  mended), and `playing`, the name of the set in force (`rule_sets_query` beside the name `InForce` holds).
- **`SaveRuleSet{set}`** makes a set or replaces the values it holds: `cprules.SaveUseCase` restores the snapshot the
  call carries over the `default` kept, and hands the store the set. A set `Restore` refuses is `InvalidArgument`, with
  every reason. **A save of `default` is refused while another set cannot play over it**, each such set named with its
  reasons, so a battle's set never fails when the schedule picks it; the check reads every set in the save's
  transaction, so nothing is kept. A set that already could not play blocks a save of `default` until it is mended.
  **Then it plays the set chosen last again**
  (`PlayUseCase.Execute`, through `LoggedPlay`): a save of the set in force takes over at once, and a save of
  another set changes nothing. A play that fails after the save is an error; the set stays kept and plays at the next
  play or boot.
  `cprules.AuditedSave` logs every save at Warn (`admin rule set saved` or `refused`, with the rules).

The rules are pinned by each part's tests, `Set`'s (one refusal per impossible value), the store's contract suite and
its migration tests, the limiter's policy tests, `Held`'s, the handlers', and `cprules`' own (the sets, the use cases,
`InForce`); the e2e suite holds smoke tests only (see [Testing](testing.md)). Its stack migrates
planet's schema first (`planet.Migrate`, behind the `testing` tag) and writes its own `clicks` into `default` before
planet boots, since every test clicks from one address.
