# Rules: the game's rule sets (`internal/planet/internal/rules/`)

**Every game rule is kept in postgres, not in the config.** A game rule is what a player feels: how fast a bank
refills, what a box holds, how big a bomb is. A rule set is a name and every one of those values, **all written**:
no set inherits from another, and no value has a default. Each set is one row of `planet.rule_sets` (migration
`20261026210000_rule_sets`). **The migration keeps the game's own rules as `default`**: production's values on
26 October 2026, the ones `deploy/vps/backend.yaml` held before they moved. Every other set is made from the
backoffice. The game plays `default`. **A rule set holds planet's words only**: the anti-bot's bounds are its own
config (see [Anti-bot](antibot.md)).

## What a set holds

The paths are `planet.v1.RuleSet`'s JSON names: the backoffice shows them, and a refusal names one.

- `tempo` — how many times faster than `clicks.perSecond` every bucket refills (1).
- `clicks.perSecond`, `clicks.burst` — one account's refill and bank (0.2, one click every 5 s; 60). See
  [Rate limiting](rate-limiting.md).
- `clicks.scopeMultiplier`, `clicks.v6ScopeMultiplier`, `clicks.guestScopeMultiplier` — the address's bucket, an
  IPv6 /64's, and the one every guest behind an address shares, in accounts (10, 4, 1).
- `clicks.linkedMultiplier` — how many times as fast a signed-in account refills (2).
- `clicks.newAccountClicks` — what a new account's bank holds; it earns the rest at `clicks.perSecond` (10).
- `tollSteps` — rising `{share, slowdown}`: from `share` of the map, a country refills `slowdown` times slower
  (10% 1.5×, 20% 2.5×, 30% 4×). No step is no toll. See [A big country refills slower](rate-limiting.md#a-big-country-refills-slower-clickstoll).
- `boxes.minInterval`, `boxes.maxInterval`, `boxes.maxChargesPerHour` — a box's wait and the hourly cap (4–8 min, 12).
- `bonuses.kinds.*` — what a box is drawn as: a weight per kind, `refill`, `spreadClicks`, `encloseClicks`, `bomb`,
  `shields` (5, 3, 2, 1, 3). 0 is never offered.
- `bonuses.offerTtl` — how long a box flies; **it must outlast the flight the client draws**, or a box caught on its
  last frame is refused (15 s).
- `bonuses.missRetry` — a box missed brings the next one this soon, once (2 min).
- `bonuses.activeWithin` — no box and no quiz for a caller that has not clicked for this long (2 min).
- `bonuses.spread.clicks`, `.maxPerBox` — the spread pool and what one box adds at most (8, 4). See [Spread](spread.md).
- `bonuses.bomb.rings` — the blast's radius in tile spacings (4). See [Bomb](bomb.md).
- `bonuses.enclose.maxTiles`, `.held`, `.maxPerBox` — the biggest shape, the stack, a box's most (25, 3, 3). See
  [Enclose](enclose.md).
- `bonuses.shield.held`, `.minPerBox`, `.maxPerBox`, `.perTile` — the stack, a box's least and most, and a tile's
  most (30, 5, 20, 10). See [Shields](shields.md).
- `bonuses.welcome.within`, `.boxAfter` — how young an account is welcomed, and when its first box comes (10 min,
  1 min); `0s` is off. See [A newcomer's first box and first quiz](welcome-bonus.md).
- `quiz.enabled`, `quiz.minInterval`, `quiz.maxInterval`, `quiz.maxChargesPerHour`, `quiz.offerTtl`,
  `quiz.answerWindow`, `quiz.missRetry`, `quiz.leaderBias` — the quizzes (on, 4–8 min, 12, 25 s, 8 s, 2 min, 1). See
  [Quizzes](quizzes.md).

**What stays in the config is the machine**: addresses, `database`, `redis`, secrets, `gameMap.maxIndex`,
`tilesStorage.*`, `ledger.*`, `ledgerStorage.*`, `vpnBlocklist.*`, `auth`, `rateLimiter.sweepInterval`,
`bonus.forgetAfter` and `bonus.sweepInterval` (see [Configuration](configuration.md)). koanf drops a key no struct
reads, so a rule written back in a file would change nothing, silently: `TestNoConfigFileHoldsAGameRule` refuses
every moved key in the three YAML files.

## How postgres keeps a set

**One `jsonb` document per set: the `planet.v1.RuleSet` message as protojson, with no name** (the row's `name` is
it). `rule_set_message` is the one codec: `Encode` and `Decode` between the message and `rules.Snapshot`, `Document`
and `Parse` between the message and the document. The save handler, the store, the `GetRules` query and the logs
all use it. Why a document and not columns:

- About forty values, a list (`tollSteps`) and a map (the kinds) would be forty columns and a child table.
- A new rule is a proto field and a data migration that writes it into each set, not a new column in a table and
  in every query.
- `GetRules` answers the document as it is kept (`rule_sets_query`), with no mapping of its own.

Postgres checks only that the name is not empty and the document is an object: **`Snapshot.Restore` is the rule,
on the way in and on the way out.** It refuses a name that is not a lowercase word, and every value a set cannot
play (`ErrInvalidSet`, with the path of the first one): a tempo, a refill, a bank, a box, a pool or a stack of 0,
a multiplier under 1, a new bank over the bank, shares or slowdowns that do not rise, a weight under 0, every
weight 0, a kind not weighed, a longest wait shorter than the shortest, a shield box whose most is under its least,
more than 255 shields on a tile (a tile keeps them in a byte), a welcome or a lean under 0. A document with a field
this build does not know is refused too (`protojson` is strict): a renamed field needs a migration.
`postgres_rule_set_store` finds a set by name (`ErrUnknownSet`) and upserts one; `rules.StoreContractSuite` runs
on it and on `MemoryStore`.

## The set in force: one atomic pointer

**`rules.InForce` holds the set in force in one `atomic.Pointer`, and each part of the game reads it where it uses
it.** Nothing is pushed: `Enforce` stores the set, stamps when it took over, and wakes the `Watch()`ers. The readers,
each through a small port of its own package:

- `Policy()` — the click limiter (`cpratelimit.NewRuled`): `clicks.perSecond`, `clicks.burst`, `tempo`.
- `Throttle()` — `clicks.Buckets`: the multipliers and `newAccountClicks`, so the throttle, `GetBudget` and a refill
  name the same buckets.
- `Toll()` — `clicks.Toll`: the price of each click, and `GetBonusRules.toll_steps`.
- `Bonuses()` — the box registry (waits, kinds, flight, miss, activity, welcome, quizzes, the quiz lean),
  `bonuses.Charges` (the stacks), `bonuses.Bombing` (the radius), `enclose_click` (the biggest shape),
  `place_shield_usecase` and `GetBonusRules`.
- `ShieldsPerTile()` — `clicks.Claiming`, for a fortify.
- `Tempo()` — `try_click`, which tells the anti-bot the tempo of each try on `ClickTried.tempo`.

**Before the first set every value is 0**: no click passes, no box is offered. The boot enforces `default` before
anything is served, so this is never seen.

## What changes for what is already there

A value read where it is used applies from the next use. **A value that carries state has a rule of its own:**

- **A bucket's rate and size** (`cpratelimit.Limiter.follow`): at each call the limiter compares the policy with
  the one it applied. On a change it refills every bucket at the old policy up to when the new one took over
  (`Policy.Since`), then plays the new one. A smaller burst cuts every bank to it; a bigger one is earned, not
  handed over. A bucket follows its key's scale from its next take, so a new scope multiplier reaches a busy
  address at its next click.
- **The toll** is set by each click, as before: a new step slows a player from its next click.
- **Charges held above a smaller stack** are kept: `Held.Granted` never takes from a stack, and a full or fuller
  stack is offered no box of its kind.
- **Shields on a tile above a smaller `perTile`** stay: a placement is refused (`ErrTileFull`) and a fortify adds
  none until the tile is under it.
- **A box or a quiz due later than the new longest wait** is brought in to a wait at the new pace, at the next
  sweep (the registry compares at each sweep, every second).
- **Quizzes switched on**: each caller's next quiz is a whole wait away, so a wait long past is not a quiz for
  everyone at once. **Switched off**: no quiz is asked; one asked already can still be opened and answered until it
  lapses. A newcomer is owed no welcome quiz while they are off.
- **A box or a banner in flight** keeps the expiry it was given; **an account welcomed** keeps its welcome.

## Following postgres

**What postgres keeps is the truth, and the game follows it** (`enforce_rules_usecase`): it reads `default` and,
when its values changed (`Snapshot.Equal`), hands it to `rules.InForce`. It runs once while the module is built,
and **a missing or broken `default` refuses the boot**, then every second (`planet-rules`). So an edit reaches the
game within a second, and a restart plays what was kept. A read that fails changes nothing and is logged;
`log_enforce_rules` logs each set enforced at Warn, with its document.

**Clients follow too.** `Watch()` feeds each planet stream, which sends `PlanetEvent.rules_changed`; the web client
reads `GetBudget` and `GetBonusRules` again, so an open tab shows a new refill and new sizes with no reload.
`GetBonusRules` is a GET no cache keeps for that reason.

## An operator writes them

`AdminService/SaveRuleSet` makes a set or replaces its values (`audit_save_rule_set` logs every save at Warn, with
its document; a set that cannot play is `InvalidArgument`, with the refusal's words), and `GetRules` lists every
set, `default` first. The backoffice's Game rules page calls both (see the backoffice's [Ops](../../backoffice/docs/ops.md)).

The rules are pinned by `Set`'s tests (one refusal per impossible value), the codec's, the store's contract suite and
its migration test, the limiter's policy tests, the registry's pace and quiz tests, `Held`'s, the handlers' and
`InForce`'s; the e2e suite holds smoke tests only (see [Testing](testing.md)). Its stack writes its own
`clicks` into `default` after the boot, since every test clicks from one address.
