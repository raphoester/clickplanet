# The flag mission simulator (`cmd/missionsim`)

**It replays a season played and draws each flag's mission every day as the game does, with the numbers the game plays
and with others, to choose the goals and the points from what the players did, not from a guess a day** (the owner,
2026-11-06, after "break 30 shields" gave France 9 points). It only reads production, and changes nothing.

```bash
cd apps/backend
go run ./cmd/missionsim fetch -ssh root@<host> -since 2026-11-01T06:00:00Z -out ~/season-1
go run ./cmd/missionsim run -data ~/season-1 -sizings sizings.json -rows rows.csv
```

## What it does

- **The numbers are a `flagmissions.Sizing`**: the goal is `GoalPercent` of what the flag's players did the day
  before, at least `LeastConquest` tiles or `LeastShields` shields; the points go from `MostPoints` for a flag with no
  tiles to `FewestPoints` for the biggest. `flagmissions.Played` is the one the game plays, and `Draw` takes the
  sizing, so the simulator draws with the game's own code. A change of the formula, not of a number, is a change of
  the code: run the simulator again on the branch that makes it.
- **`fetch`** (`missions.FetchSeason`, `simulation/ssh_season_dump`) asks production's postgres over ssh, with
  `psql` in the `cp-postgres` container and the SQL on its input, for two CSV files: `days.csv`, each day told
  (`missions.mission_days`, with its season and number) that ends after `-since`, and `acts.csv`, every act of `planet.ledger_events` from
  `-since`, unfolded from each kind's payload as planet's ledger kinds write it: a take, a shield struck (a click's,
  a spread's, an enclose's or a bomb's) and a tile a bomb cleared. Shields placed and fortifies change no flag and are
  left out. **The ledger has no index on time**: the query finds its first position with a bisection over the primary
  key (some 40 index reads), so it reads a range of the key and never scans the whole table on the live box: a heavy
  read of the ledger once slowed the game for minutes. `CLICKPLANET_SSH` stands in for `-ssh`.
- **`-since` is a season's start**: its map was blank (see [The map reset](map-reset.md)), so the replay starts from
  a blank map on the land of the map this build ships (`mapdata.DecodeBorders`, the decoder planet builds its
  landmasses with). A reassign, a paint or a revert writes no ledger row, so the replay drifts from the real map by
  them: the report counts the acts that found a tile wearing another flag than the replay said.
- **`run`** (`missions.Simulate`, `simulation.Run`) replays the acts in ledger order (`simulation.Globe`, what each
  flag holds on each country's land). A flag's mission is drawn at its first take with an account or shield broken of
  a day, with the map then and its day before, as `Briefing.Of` does in the game, then counted with
  `Day.WithTake` and `WithStrike`, so the counting is the game's too. A fact between two days (in a battle) counts
  nothing, and a day's day before is the one before it in its season (`days.Day.Follows`), as `DayBefore` finds it. Production draws at a read of
  `GetMissions` too, which the ledger does not keep: the map may differ a little.
- **What it measures is the players playing as they did**, with no mission to aim at. A mission done that way is too
  easy: nobody had to do anything for it. A mission the players went for is done more often in the game than here:
  compare the played sizing's row with what `missions_list_flag_days` says the game gave and the flags did.

## The report

One table a sizing, `played` first, then each one of `-sizings`, a JSON array:

```json
[{"name": "harder", "goal_percent": 80, "least_conquest": 300, "least_shields": 300, "fewest_points": 2, "most_points": 6}]
```

Each line is a group of flag days with a mission: by the part of the map the flag held at the draw (5% or more, 0.5%
to 5%, under 0.5%), by kind (conquests, shields to break) and all of them. It gives how many were done, the median
time from the draw to the fact that finished it, the median goal, the median goal beside all the flag did that day
(tiles taken and shields broken), and the points offered and won. `-rows` writes every flag's day of every sizing as
CSV, with what the flag did the day before and that day, for a spreadsheet.
