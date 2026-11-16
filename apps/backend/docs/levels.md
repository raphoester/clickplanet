# Levels (`progression/internal/levels/`)

**One level per linked account, from XP computed from what its stats row counts.** XP is never stored: it is computed
again at each read, so a change of weights is one reconciliation. Only the level is kept (`progression.levels`), and it
has no top. A guest has no XP and no level. The design is [Player level and unlocks](https://claude.ai/artifact/8bvS9FGScmFpbz4kfCVsy2);
its first slice is [XP slice 1: hidden level bootstrap](https://claude.ai/artifact/Pf6X1m7gNk72F42q5U8n5A).

**The level runs hidden: players see nothing.** Each fact counted raises it, each raise is told silent, and only the
operator reads it (`AdminService/GetProgress`). No public read, no stream and no reward know it yet.

```
levels/                            XP, Level, Record, Source, MissionKind (stats.MissionKind), MissionRate, Curve, Standing,
                                   Scale (XPOf, Reach, StandingOf, ReconciliationOf), Visibility, ByAccount; the Store port and
                                   its contract suite
  postgres_level_store/            the Store over progression.levels, under the rows' locks
  inmemory_level_store/            the same port in a map, behind the testing tag
  usecases/raise_level_usecase/    on progression.v1.StatsChanged: the level raised, then progression.v1.LevelReached
  usecases/forget_level_usecase/
  usecases/reconcile_levels_usecase/  progression.v1.AdminService/ReconcileLevels
    reconcile_levels_usecase/audit_reconcile_levels/  logs every reconciliation at Warn
```

## The flag: `levels.shown`

**A top-level YAML key, `levels: {shown: false}`, read once by `cmd/api`** (`LevelsConfig`) and handed to each module it
changes in that module's words: `progression.NewModule(config.Progression, progression.Levels{Shown: …})`, which
progression reads as a `levels.Visibility` (`Hidden`, `Shown`). It is in `deploy/vps/backend.yaml`, `deploy/backend.yaml`
and `cmd/api/example.yaml`; `TestThePlayersSeeNoLevelInAnyConfig` pins `false` in each. A flip is an edit and a deploy,
and the restart reloads every page (see [A server restart reloads the page](../../../docs/reload-on-restart.md)).

- **The boot refuses `shown: true`** (`progression.Levels.Validate`, `ErrNoUnlockLadder`): no level is shown before a
  level unlocks something. Nothing does yet, so the guard is that alone.
- **While hidden, a level-up is silent** (`LevelReached.silent`) and the reconciliation sets the exact level, lower too.
  **Once shown, a level never goes down.**

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

**A guest's standing is empty** (`Standing.Empty`): no XP, no level, no row and no `LevelReached`, whatever it did. The
test is the titles' one: `accounts.Account.Linked`, as player tells it. A linked account that did nothing is at level 1.
**A guest's history counts once it links** (the owner, 2026-11-16): a link keeps the account row, so its next fact
raises it on the whole stats row, as its titles are earned on its whole career. A guest that signs in to another
account leaves its history behind.

## The record

**`Record` is what the level reads: the stats row's counts, and `stats.Stats` is one.** It answers each under the same
name (`TilesTaken`, `DaysPlayed`, `MissionGoals(kind)`, `BombsDropped`, `BonusesClaimed`, `RightAnswers`,
`TilesFortified`). `levels.MissionKind` is `stats.MissionKind`: levels reads the stats, never the other way.

- The days played are in [Progression](progression.md#the-days-played).
- **`MissionGoals(kind)` is the goals of the missions done of that kind, summed, a guest's too**, from
  `missions.v1.MissionsDone.by_kind` (group `progression-stats-missions-done`, `missions_done_subscriber`): the kind is
  the enum's name without `MISSION_KIND_`, in lower case, `UNSPECIFIED` skipped, and a number this build does not know
  refused. The stats keep the highest total heard per kind (`Stats.WithMissionsDone`, the `missions_by_kind` jsonb
  column, `{"take_for": {"done": 3, "goals": 540}}`), so a total told twice, late or lost changes nothing. Done, not
  paid: a guest's missions count from the day they were done, once the account links. `missions_done` and
  `MissionsPaid` stay as they were: the Missions title track counts payments.

## The level kept

**`progression.levels(account_id, level >= 1, reached_at)`, one row per linked account a fact raised.** The `Store`
port (`StoreContractSuite`, run on postgres and in memory):

- **`Raise(account, level, at)`** writes only above the level kept and answers the level kept before it, 0 for none. In
  postgres: an insert that does nothing on a row there, then that row read `FOR UPDATE` and updated only below the
  level, all in one transaction. Raises at once never lower and each one tells the level the one before left
  (`TestRaisesAtOnceKeepTheHighestAndEachJumpStartsWhereTheOneBeforeEnded`).
- **`RaiseAll(levels, at)`** is a page's raise in one upsert, `WHERE kept.level < excluded.level`: it never lowers a
  level a live raise wrote. **`Set(levels, at)`** writes each level as it is. **`Forget(accounts...)`** deletes.
- `reached_at` is the time of the last write that changed the level.

## Raised live

**`raise_level_usecase`, group `progression-levels` on `progression.v1.StatsChanged`** (`stats_changed_subscriber`, as
the titles' `player-titles`):

1. Read the stats. None (`ErrNoStats`): nothing to do.
2. The level the stats reach (`Scale.Reach`) not above the level kept: stop. No call to player, no write. Most facts end
   here.
3. Ask player whether the account is linked (`rpc_player_accounts`, as the titles do). A guest stops here.
4. In one transaction, `Raise`, and record `progression.v1.LevelReached{account_id, level, previous, reached_at,
   silent}` in progression's outbox, only when `Raise` answered a level below: a raise another write beat writes and
   tells nothing.

- **One event per jump**: from 1 to 9 is one event, `previous: 1`. `silent` is `!levels.shown`.
- **Through an outbox** (`cpoutbox`, relay `progression-outbox`), because the rewards to come pay one bonus per
  `LevelReached` that is not silent: a lost one would be lost for good. See [What may not be
  lost](composite-layer.md#what-may-not-be-lost-the-outbox).
- **Nobody listens yet**, so `events:progression.v1.LevelReached` fills up to `redis.maxLen`. A listener added later
  reads it from its start, the silent ones too, which is why each says it.
- A failure to read or to ask player is an error: the event is tried again, then buried.

## The reconciliation

**`progression.v1.AdminService/ReconcileLevels(dry_run)` → `{raised, lowered, removed}`**, by hand, after a deploy that
changes the scale or the curve, and once after the one that added the level. Connector tool
`progression_reconcile_levels`. `reconcile_levels_usecase`, built as `ReconcileTitles`:

- It pages `progression.stats` 500 at a time, asks player the page's accounts (`GetAccounts`, one call) and reads their
  levels (one query); `Scale.ReconciliationOf` says what to raise, lower and remove.
- **Hidden**: every linked account gets the exact level its stats reach, lowered where a stricter scale no longer
  reaches it (`Set`), and a guest's row is removed. **Shown**: it only raises.
- **Raises go through `RaiseAll`**, so a page never lowers a level a live raise wrote after the page was read. A lowering
  can still undo one, while hidden only: the account's next fact raises it again, silent.
- **It publishes nothing**: a level it gives is never announced nor rewarded. `dry_run` counts the same and writes
  nothing; a second run changes nothing. `audit_reconcile_levels` logs every run at Warn.

## Forgotten

`player.v1.PlayerForgotten` deletes the account's level (group `progression-levels-forgotten`, `forget_level_usecase`).
A raise from a fact read before the forget, or a reconciliation page read before it, can write the row again, as a
late fact does for the stats.

## The operator's read

`AdminService/GetProgress` answers `level` (the level kept, 0 for none), `xp` (`XpBreakdown`: the total and one field
per source, from the stats row, a guest's too) and `missions_by_kind` (`MissionTally{kind, done, goals}`, by kind's
name). The XP goes through `inprocess_level_scale`, the scale behind a port as `inprocess_title_catalog` puts the
catalog; its test pins that the fields add up to the total, so a new source with no field fails it.

## After the deploy that added it

1. `missions_retell_missions_done`, dry run first, so progression hears every account's totals. Its group reads the
   stream from its start, so this is only needed if the stream no longer holds them.
2. `progression_reconcile_levels`, dry run, then for real.
3. Spot-check a few accounts with `progression_get_progress` against the design's ship-day numbers: a median about
   level 9, the top about 29.
