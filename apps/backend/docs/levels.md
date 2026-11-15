# Levels (`progression/internal/levels/`)

**One level per linked account, from XP computed from what its stats row counts.** XP is never stored: it is computed
again at each read, so a change of weights is one reconciliation. The level only goes up, and it has no top. A guest
has no XP and no level. The design is [Player level and unlocks](https://claude.ai/artifact/8bvS9FGScmFpbz4kfCVsy2);
this package is the domain of its first slice, [XP slice 1: hidden level bootstrap](https://claude.ai/artifact/Pf6X1m7gNk72F42q5U8n5A).

**Nothing calls it yet.** No store, no event, no read and no flag: those come with the slice's hidden-level change,
behind `levels.shown`, off. Players see nothing.

## What gives XP

One type per source (`Source`), each reading one count of the account's `Record`. `Scale.Sources` lists them, and
`Scale.StandingOf` adds them up.

| Type             | Count read                | XP                                             |
| ---------------- | ------------------------- | ---------------------------------------------- |
| `TilesTaken`     | `TilesTaken()`            | × 1                                            |
| `DaysPlayed`     | `DaysPlayed()`            | × 100: the first take of a UTC day             |
| `MissionsDone`   | `MissionGoals(kind)`      | Σ per kind: the goals paid × the kind's rate   |
| `BombsDropped`   | `BombsDropped()`          | × 50                                           |
| `BoxesCaught`    | `BonusesClaimed()`        | × 30                                           |
| `RightAnswers`   | `RightAnswers()`          | × 30: a wrong answer gives nothing             |
| `TilesFortified` | `TilesFortified()`        | ÷ 5 (0.2 a tile), floored on the total         |
| `MessagesSent`   | none                      | 0: XP for messages would fill the chat with spam |

Integer math only: every fraction is floored.

## What a mission pays

**A mission pays its goal times its kind's rate**, so a bigger goal pays more. Missions tells the goals; progression
owns the rates. One type per kind (`MissionRate`), listed by `MissionsDone.Rates`. A `MissionKind` is missions' own
name for it: `missions.v1.MissionKind` without its prefix, in lower case, as missions keeps it in its briefs.
`TestEveryKindOfMissionHasOneRate` loops the enum, so a new kind of mission with no rate fails the tests.

| Type            | Kind             | XP per unit of the goal |
| --------------- | ---------------- | ----------------------- |
| `TakeFor`       | `take_for`       | 1                       |
| `TakeFrom`      | `take_from`      | 1.5, floored            |
| `TakeIn`        | `take_in`        | 1                       |
| `TakeFromIn`    | `take_from_in`   | 3                       |
| `CatchBoxes`    | `catch_boxes`    | 50                      |
| `AnswerQuizzes` | `answer_quizzes` | 80                      |
| `ShieldTiles`   | `shield_tiles`   | 80                      |
| `FortifyTiles`  | `fortify_tiles`  | 4                       |
| `BombIn`        | `bomb_in`        | 150                     |

The rates are first guesses, sized so a mission pays about the XP of the play it asks for.

## The curve

**The XP to reach level L is 14 × (L − 1)³** (`Curve.Threshold`), so going from L to L + 1 costs 14 × (3L² − 3L + 1).
`Curve.LevelOf(xp)` is the highest level whose threshold the XP reaches, at least 1. `Curve.StandingOf(xp)` is a
`Standing`: the level, the XP, and the two thresholds the bar runs between (`LevelXP`, `NextLevelXP`).

| Level | Total XP  | To the next |
| ----- | --------- | ----------- |
| 3     | 112       | 266         |
| 5     | 896       | 854         |
| 10    | 10,206    | 3,794       |
| 20    | 96,026    | 15,974      |
| 30    | 341,446   | 36,554      |
| 50    | 1,647,086 | 102,914     |

**No top level.** The highest level an XP can hold is 1,096,303; a level past it costs the most an XP holds
(`math.MaxUint64`), so no XP reads it.

## Guests

**A guest's standing is empty** (`Standing.Empty`): no XP and no level, whatever it did. The test is the titles' one:
`accounts.Account.Linked`, as player tells it. A linked account that did nothing is at level 1. Whether a guest's
history counts once it links is open in the slice doc; the record is the whole stats row, so as written it does.

## The record

**`Record` is what the level reads: the stats row's counts.** `stats.Stats` already answers five of them under the same
names (`TilesTaken`, `BombsDropped`, `BonusesClaimed`, `RightAnswers`, `TilesFortified`). The two others come with
changes in progress:

- `DaysPlayed()`: the days played, a stat of its own with its backfill from planet's ledger.
- `MissionGoals(kind)`: the goals of the missions paid, summed per kind, from `missions.v1.MissionsPaid.by_kind`.

Until both are in the stats, nothing hands a `stats.Stats` to the scale. The stats cannot name `levels.MissionKind`
(levels reads the stats, never the other way), so the change that wires the level maps the stats' own kinds to it.
