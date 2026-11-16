# Levels (`progression/internal/levels/`)

**One level per linked account, from XP computed from what its stats row counts.** XP is never stored: it is computed
again at each read, so a change of weights is one reconciliation. Only the level is kept (`progression.levels`), and it
has no top. A guest has no XP and no level. The design is [Player level and unlocks](https://claude.ai/artifact/8bvS9FGScmFpbz4kfCVsy2);
its first slice is [XP slice 1: hidden level bootstrap](https://claude.ai/artifact/Pf6X1m7gNk72F42q5U8n5A).

**Each fact counted raises it.** The public reads answer it (see [The public reads](#the-public-reads)), and the page
draws them (the frontend's [The level](../../frontend/docs/level.md)), merged with the rest of the reveal (see [The
reveal](#the-reveal)).

```
levels/                            XP, Level, Record, Source, MissionKind (stats.MissionKind), MissionRate, Curve, Standing,
                                   Scale (XPOf, Reach, StandingOf, ReconciliationOf), ByAccount; the Store port and its
                                   contract suite
  postgres_level_store/            the Store over progression.levels, under the rows' locks
  inmemory_level_store/            the same port in a map, behind the testing tag
  inprocess_level_feed/            each raise, to the account's open streams, in this process
  usecases/raise_level_usecase/    on progression.v1.StatsChanged: the level raised, told to the feed
  usecases/forget_level_usecase/
  usecases/reconcile_levels_usecase/  progression.v1.AdminService/ReconcileLevels
    reconcile_levels_usecase/audit_reconcile_levels/  logs every reconciliation at Warn
```

## The reveal

**No flag: the level stays hidden because nothing a player sees ships before the end.** The pieces that show it ship
together, as the reveal:

- the UI: the level and its bar on the page and on the card, from the public reads (the frontend's
  [The level](../../frontend/docs/level.md));
- missions paying XP instead of charges;
- `LevelReached`, through an outbox, with slice 3's bonus listener: both are added then, so no level gained before the
  reveal pays a bonus;
- the reconciliation made raise-only, so a level the players saw never goes down.

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

**A rate is exact**: `Rate{XP, Per}`, XP for every `Per` units of the goal, floored, so `take_from` is 3 for 2.
`GetLevel` sends each rate in that form, never as a float, so the page's goal × rate is the server's integer math
(`TestThePagesIntegerMathPaysWhatTheServerDoes`). The server floors once per kind, on the goals summed; a page that
shows one mission floors that mission. The two agree while a `take_from` goal is even, and missions draws them in
steps of 10.

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

**A guest's standing is empty** (`Standing.Empty`): no XP, no level and no row, whatever it did. The
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
4. `Raise`: the store writes only above the level kept, so a raise another write beat writes nothing.
5. Hand the raise to `inprocess_level_feed`: the level raised to, on its bar (`Scale.StandingAt`).

- **One write per jump**: from 1 to 9 is one raise, and one `level_reached`. The use case answers the level it raised
  to, 0 for none.
- **It tells only the account's open streams, in this process, at most once**: the feed drops for a stream 8 raises
  behind, and an account with no stream open hears nothing. Nothing crosses to another module: `LevelReached`, with its
  outbox, comes with the reveal. The use case calls the feed itself, after the raise: a raise another write beat, a
  guest and a failure tell nothing.
- A failure to read or to ask player is an error: the event is tried again, then buried.

## The reconciliation

**`progression.v1.AdminService/ReconcileLevels(dry_run)` → `{raised, lowered, removed}`**, by hand, after a deploy that
changes the scale or the curve, and once after the one that added the level. Connector tool
`progression_reconcile_levels`. `reconcile_levels_usecase`, built as `ReconcileTitles`:

- It pages `progression.stats` 500 at a time, asks player the page's accounts (`GetAccounts`, one call) and reads their
  levels (one query); `Scale.ReconciliationOf` says what to raise, lower and remove.
- **Every linked account gets the exact level its stats reach**: raised, or lowered where a stricter scale no longer
  reaches it (`Set`), and a guest's row is removed. It becomes raise-only with the reveal.
- **Raises go through `RaiseAll`**, so a page never lowers a level a live raise wrote after the page was read. A lowering
  can still undo one: the account's next fact raises it again.
- **It publishes nothing**, not even to the open streams. `dry_run` counts the same and writes nothing; a second run
  changes nothing.
  `audit_reconcile_levels` logs every run at Warn.

## Forgotten

`player.v1.PlayerForgotten` deletes the account's level (group `progression-levels-forgotten`, `forget_level_usecase`).
A raise from a fact read before the forget, or a reconciliation page read before it, can write the row again, as a
late fact does for the stats.

## The public reads

**The level answered is the one kept** (`Scale.StandingAt`): the level of the `progression.levels` row, the XP its
stats give now (`Scale.XPOf`), and the two thresholds of the level kept (`level_xp`, `next_level_xp`). So the card,
`GetLevel` and the stream never disagree: a level shows once a raise or the reconciliation wrote it. The page draws
them: see the frontend's [The level](../../frontend/docs/level.md).

- **`ProgressionService/GetLevel`** → `{level, mission_rates}`, the caller's own: `Identified` behind the session
  interceptor, `NO_SIDE_EFFECTS` (`get_level_handler`, `level_query`).
- **`ProgressionService/GetProgress`**, the public card, answers `Progress.level` the same way (`progress_query`).
- **`ListenForEvents` sends `level_reached`**, a `Level`, to the caller's own open stream at each raise (see
  [Raised live](#raised-live)).
- **`mission_rates`: one `{kind, xp, per}` per `missions.v1.MissionKind` but `UNSPECIFIED`**, for the page to pay goal
  × xp ÷ per at the reveal. The kind is missions' enum, not the lower-case name progression keeps in its stats: the page
  reads the wire in missions' words and matches it to a mission's kind, and the lower-case name is how a row stores it.
  `inprocess_level_scale.MissionRates` maps one to the other, strictly: a kind the enum does not name is an error, and
  `TestEveryKindOfMissionButUnspecifiedHasOneRateOnTheWire` loops the enum.

**A guest has no level.** Neither the raise nor the reconciliation keeps a row for one, so `GetLevel` answers an empty
`level` (its rates still), and the card is `NotFound` as before: a guest has no nameplate.

**A linked account with no level kept answers none, not level 1.** The level answered is the one kept, so no read
shows a level the raise has not written, and its first `level_reached` is the level appearing. Level 1 would need the
linked flag in each read, which is player's to say, and would show 1 while the raise to more is on its way. It does not
last: any fact raises a linked account to at least 1, and `ReconcileLevels`, run after the deploy that added the
level, writes every linked account with stats. Only a linked account that never did anything has none.

**The XP can sit past the bar.** Between a fact counted and the raise that follows it (milliseconds, or the retries of
a failed raise), `xp` is above `next_level_xp`: the bar shows full, then `level_reached` comes. After a deploy that
makes the scale stricter and before `ReconcileLevels`, `xp` can be below `level_xp`: the bar shows empty. The server
sends the numbers as they are, and the page clamps the bar.

## The operator's read

`AdminService/GetProgress` answers `level` (the level kept, 0 for none), `xp` (`XpBreakdown`: the total and one field
per source, from the stats row, a guest's too) and `missions_by_kind` (`MissionTally{kind, done, goals}`, by kind's
name). The XP goes through `inprocess_level_scale`, the scale behind a port as `inprocess_title_catalog` puts the
catalog; its test pins that the fields add up to the total, so a new source with no field fails it. The public reads
use the same port: the level kept on its bar (`Level`) and the rates (`MissionRates`).

## After the deploy that added it

1. `missions_retell_missions_done`, dry run first, so progression hears every account's totals. Its group reads the
   stream from its start, so this is only needed if the stream no longer holds them.
2. `progression_reconcile_levels`, dry run, then for real.
3. Spot-check a few accounts with `progression_get_progress` against the design's ship-day numbers: a median about
   level 9, the top about 29.
