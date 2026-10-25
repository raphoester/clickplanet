# Rules: the game's rule sets (`internal/planet/internal/rules/`)

**What can change while the game runs is kept in postgres, not in the config.** A rule set is a name and every
value a moment of the game may change: how fast every bucket refills (`tempo`, times `rateLimiter.perSecond`), the
boxes' shortest and longest wait and their hourly cap, and how many networks of one prefix the anti-bot bans as a
crowd. Each is one row of `planet.rule_sets` (migration `20261025130000_rule_sets`), every value written: no set
inherits from another, and the config no longer holds `bonus.minInterval`, `bonus.maxInterval`,
`bonus.maxChargesPerHour` or `antiBot.cohort.detector.certainMembers`. **The migration keeps the game's own rules as
`default`**, production's values on 25 October 2026 (tempo 1, a box every 4 to 8 minutes, 12 an hour, a crowd of
6); every other set is made from the backoffice.

**Which set plays is two layers, never mixed** (`rules.Layers.SetAt`): **an operator's override while it runs, then
the baseline, then `default`** when no baseline was ever applied.

- **The baseline is what a module asks for**: `planet.v1.InternalService/ApplyRuleSet{request_id, rule_set}`. Each is
  a row of `planet.rule_baselines` (migration `20261025140000_rule_layers`), and **the newest is the baseline**.
- **The override is the operator's, and always has an end**: `AdminService/OverrideRules{rule_set, until}` puts a set
  over the baseline from now until `until`, a day ahead at most (`rules.NewOverride`, `ErrInvalidOverride`), in place
  of any override running. `EndOverride` ends the one running now; with none running it changes nothing. It is one
  row, `planet.rule_override`, kept once ended. **At its end the baseline plays again by itself**: no override can be
  forgotten on, and the baseline is never overwritten, so the season's word and the operator's never fight over one
  value.
- **Planet never learns why a set is asked for.** The relationship is Customer–Supplier with an Open Host Service and
  a Published Language: planet publishes its rule sets and `ApplyRuleSet`, and a module downstream translates its own
  concepts into them in its own anticorruption adapter. `seasons` is that module (see
  [Seasons](seasons.md)): its finale plays `rush`. It is a command, an intent in planet's words, not a
  fact being told, so the rule of [Telling other modules what happened](composite-layer.md#telling-other-modules-what-happened) holds:
  planet hears no word of seasons, and seasons publishes no event planet listens to.
- **A request applies once** (`rules.NewBaseline` refuses a request the store kept, `ErrAppliedAlready`, which
  `apply_rule_set_usecase` answers as success). So a caller sends it again with no harm, and a baseline applied since
  stands. An unknown set, an empty request id or one past 128 characters is `InvalidArgument`. `audit_apply_rule_set`
  logs every request at Warn.
- **Known debt: the delivery is a stopgap.** Facts between modules go over Redis Streams, but this command still
  goes over the loopback call, so a command sent while planet cannot answer is sent again. `seasons` sends the phase
  of now again at each boot and at each change, and planet ignores a request it already applied. The user called
  this a poor solution on 2026-10-25: **it must move to a command stream of planet's on Redis**, and the code marks
  the places with `TODO(broker)`.

- **`rules.Set` is the rule** (snapshot pattern): `Snapshot.Restore` refuses a name that is not a lowercase word
  (`ErrInvalidSet`), a tempo that is not above 0, a shortest wait of 0 or a longest one under it, an hourly cap of 0
  and a crowd under 2. `postgres_rule_set_store` finds a set by name (`ErrUnknownSet`) and upserts one;
  `rules.StoreContractSuite` runs on it and on `MemoryStore`.
- **What postgres keeps is the truth, and the game follows it** (`enforce_rules_usecase`): it reads the newest
  baseline and the override, asks `Layers.SetAt` the set to play now, reads it, and hands it to `rules.InForce` when
  its values changed. It runs once while the module is built, and **a set to play that is missing refuses the boot**,
  then every second (`planet-rules`). So an edit, a baseline, an override and an override's end all reach the game
  within a second, and a restart plays what was kept. A read that fails changes nothing and is logged;
  `log_enforce_rules` logs each set enforced at Warn, with its values. `postgres_rule_layer_store` is the
  `rules.LayerStore` (a baseline by request, the newest, save; the override, save), and `rules.LayerStoreContractSuite`
  runs on it and on `MemoryLayerStore`.
- **`rules.InForce` pushes a set to what plays it, under one lock**: `cpratelimit.Limiter.SetTempo` (every bucket
  is multiplied from then on, and what each earned before was refilled at the tempo it had, so `GetBudget` reads the
  new rate at once), `bonuses.Registry.SetPace` (each wait is drawn at the new pace, and a box due later than the new
  longest wait is brought in) and `antibot.Guard.SetCrowd`. **The anti-bot runs inside planet**, which builds it and
  hands it every click, so its bound for a moment of the game is a rule like the pace. Before the first set the
  limiter is at its own tempo, no box is offered (an hourly cap of 0) and no crowd is banned: the boot enforces
  before anything is served. `Tempo()` is read by the anti-bot's click reports (see [Anti-bot](antibot.md)),
  and `Watch()` by each planet stream, which sends `PlanetEvent.rules_changed`: the client reads `GetBudget` again,
  so an open tab shows a new refill with no reload.
- **An operator writes them**: `AdminService/SaveRuleSet` makes a set or replaces its values (`audit_save_rule_set`
  logs every save at Warn, with its values; a set that cannot play is `InvalidArgument`), `OverrideRules` and
  `EndOverride` (`audit_override_rules`, `audit_end_override`), and `GetRules` reads every set, `default` first
  (`rule_sets_query`), the last 20 baselines, newest first, and the override (`rule_layers_query`), straight from
  postgres. **What plays now is not on the wire**: the backoffice's Game rules page works it out from those facts, as
  planet does.
- The rules are pinned by the domain's tests (`Layers`, `Baseline`, `Override`, `Set`), the use cases' over the
  memory stores, both stores' contract suites, the handlers' and `InForce`'s; a smoke test sees seasons apply the
  set of its phase over the internal listener (see [Testing](testing.md)). **`make finale`** runs `cmd/api` with `example.yaml` and
  Season 0's finale `FINALE_IN` minutes away (2 by default) and `FINALE_FOR` long (5), to rehearse it; the `rush` set
  must exist, made on the backoffice's Game rules page.
