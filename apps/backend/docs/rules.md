# Rules: the game's rule sets (`internal/planet/internal/rules/`)

**What can change while the game runs is kept in postgres, not in the config.** A rule set is a name and every
value a moment of the game may change: how fast every bucket refills (`tempo`, times `rateLimiter.perSecond`), and the
boxes' shortest and longest wait and their hourly cap. Each is one row of `planet.rule_sets` (migration
`20261026210000_rule_sets`), every value written: no set inherits from another, and the config no longer holds
`bonus.minInterval`, `bonus.maxInterval` or `bonus.maxChargesPerHour`. **The migration keeps the game's own rules as
`default`**, production's values on 25 October 2026 (tempo 1, a box every 4 to 8 minutes, 12 an hour); every other
set is made from the backoffice. The game plays `default`. **A rule set holds planet's words only**: the anti-bot's
bounds are its own config (see [Anti-bot](antibot.md)).

- **`rules.Set` is the rule** (snapshot pattern): `Snapshot.Restore` refuses a name that is not a lowercase word
  (`ErrInvalidSet`), a tempo that is not above 0, a shortest wait of 0 or a longest one under it, an hourly cap of 0.
  `postgres_rule_set_store` finds a set by name (`ErrUnknownSet`) and upserts one;
  `rules.StoreContractSuite` runs on it and on `MemoryStore`.
- **What postgres keeps is the truth, and the game follows it** (`enforce_rules_usecase`): it reads `default` and,
  when its values changed, hands it to `rules.InForce`. It runs once while the module is built, and **a missing
  `default` refuses the boot**, then every second (`planet-rules`). So an edit reaches the game within a second, and
  a restart plays what was kept. A read that fails changes nothing and is logged; `log_enforce_rules` logs each set
  enforced at Warn, with its values.
- **`rules.InForce` pushes a set to what plays it, under one lock**: `cpratelimit.Limiter.SetTempo` (every bucket
  is multiplied from then on, and what each earned before was refilled at the tempo it had, so `GetBudget` reads the
  new rate at once), `bonuses.Registry.SetPace` (each wait is drawn at the new pace, and a box due later than the new
  longest wait is brought in). Before the first set the limiter is at its own tempo and no box is offered (an hourly
  cap of 0): the boot enforces before anything is served. `Tempo()` is read by `try_click`, which tells the anti-bot
  the tempo of each try on `ClickTried.tempo` (see [Anti-bot](antibot.md)), and `Watch()` by each planet stream, which sends `PlanetEvent.rules_changed`: the client reads `GetBudget` again,
  so an open tab shows a new refill with no reload.
- **An operator writes them**: `AdminService/SaveRuleSet` makes a set or replaces its values (`audit_save_rule_set`
  logs every save at Warn, with its values; a set that cannot play is `InvalidArgument`), and `GetRules` lists every
  set, `default` first (`rule_sets_query`, straight from postgres). The backoffice's Game rules page calls both.
- The rules are pinned by `Set`'s tests, the use cases' over `MemoryStore`, the store's contract suite, the
  handlers' and `InForce`'s; the e2e suite holds smoke tests only (see [Testing](testing.md)).
