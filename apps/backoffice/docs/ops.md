# Ops

`src/ops/`: `ReconcileTitles` (`TitlesPanel`, after a deploy that adds a title or changes a rule). It asks first, and
it is harmless twice.

**Game rules** (`RulesPanel`, `rules.ts`) is `planet.v1.AdminService/GetRules`, `SaveRuleSet`, `OverrideRules` and
`EndOverride` (the backend's [Rules](../../backend/docs/rules.md)). The game follows any change within a second, for
every player.

- **Playing now**: the set the game plays and why, worked out from what `GetRules` answers as planet works it out
  (`playingAt`): the override while it runs, with its end and **End override**; else the newest baseline, with its
  request (`season-0:finale`) and since when; else `default`. The sets table marks it (`aria-current`).
- **Override**: a set and a number of minutes, a whole number from 1 to a day (`overrideMinutesValid`), asked first.
  The end sent is that many minutes from the click; it replaces any override running.
- **The sets**: every set postgres keeps, `default` first, with its refill (`3× as fast`), its boxes and how many
  networks make a crowd. **Edit** opens a set in a form, its name fixed; **New set** opens an empty one. The waits are
  in seconds. Save asks first (`ConfirmAction`) and is off until every value is a number and the set has a name; the
  server holds the rest (a refusal quotes it, and the form stays).
- **The baselines**: the last 20 applied, the newest first, each with its request, its set and when.
