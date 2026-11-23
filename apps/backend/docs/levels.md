# Levels (`progression/internal/levels/`)

**One level per linked account, from XP computed from what its stats row counts.** XP is never stored: it is computed
again at each read, so a change of weights is one reconciliation. Only the level is kept (`progression.levels`), and it
has no top. A guest has no XP and no level. The design is [Player level and unlocks](https://claude.ai/artifact/8bvS9FGScmFpbz4kfCVsy2);
its first slice is [XP slice 1: hidden level bootstrap](https://claude.ai/artifact/Pf6X1m7gNk72F42q5U8n5A).

**Each fact counted raises it.** The public reads answer it (see [The public reads](#the-public-reads)), and the page
draws them (the frontend's [The level](../../frontend/docs/level.md)), merged with the rest of the reveal (see [The
reveal](#the-reveal)).

```
levels/                            XP, Level, Record, Source (its SourceName), RatedSource, MissionKind (stats.MissionKind),
                                   MissionRate, Rate, Curve, Standing, Scale (XPOf, Reach, StandingOf, EarnedOf,
                                   ReconciliationOf), ByAccount; Earned, Gain, and what a stream hears (Told: Reached,
                                   Gained); the ladder (Kind, Unlock,
                                   SpreadUnlock, ShieldsUnlock, EncloseUnlock, BombUnlock, VirusUnlock, MineUnlock,
                                   TrapUnlock, RallyUnlock, Ladder); Book, the store and the bonuses told in one
                                   transaction; the Store port and its contract suite
  postgres_level_store/            the Store over progression.levels, under the rows' locks
  inmemory_level_store/            the same port in a map, behind the testing tag
  inprocess_level_feed/            each raise and each fact's XP, to the account's open streams, in this process
  outbox_bonuses/                  what a level unlocks and a level-up, as bonuses.v1 commands in progression's outbox
  usecases/raise_level_usecase/    on progression.v1.StatsChanged: the level raised, told to the feed
  usecases/forget_level_usecase/
  usecases/reconcile_levels_usecase/  progression.v1.AdminService/ReconcileLevels
    reconcile_levels_usecase/audit_reconcile_levels/  logs every reconciliation at Warn
```

## The reveal

**No flag: the level stays hidden because nothing a player sees ships before the end.** The pieces that show it ship
together, as the reveal:

- the UI: the level and its bar on the page and on the card, the locked kinds, from the public reads (the frontend's
  [The level](../../frontend/docs/level.md));
- missions paying XP instead of charges (see [Missions pay XP](missions.md#missions-pay-xp));
- the bonuses' gates on (`gates.on` in `default` and each set that is not a battle's): data the operator sets, while
  every set keeps them off before;
- the level-up bonus ([The level-up bonus](#the-level-up-bonus)) and the reconciliation made raise-only, so a level the
  players saw never goes down: one pull request, held as a draft until then, so no level gained before the reveal pays
  one.

## What gives XP

One type per source (`Source`), each reading one count of the account's `Record`, and named (`Name()`, a
`SourceName`: `progression.v1.XpSource` without its prefix, in lower case). `Scale.Sources` lists them, and
`Scale.StandingOf` adds them up. **A rated source** (`RatedSource`) pays one `Rate` on its count, which `GetLevel`
answers for the page to say what each way to gain XP pays; the missions pay by kind instead.

| Type             | Name              | Count read           | XP                                               |
| ---------------- | ----------------- | -------------------- | ------------------------------------------------ |
| `TilesTaken`     | `tiles_taken`     | `TilesTaken()`       | `Rate{1, 1}`                                     |
| `DaysPlayed`     | `days_played`     | `DaysPlayed()`       | `Rate{100, 1}`: the first take of a UTC day      |
| `MissionsDone`   | `missions_done`   | `MissionGoals(kind)` | Σ per kind: the goals paid × the kind's rate     |
| `BombsDropped`   | `bombs_dropped`   | `BombsDropped()`     | `Rate{50, 1}`                                    |
| `BoxesCaught`    | `boxes_caught`    | `BonusesClaimed()`   | `Rate{30, 1}`                                    |
| `RightAnswers`   | `right_answers`   | `RightAnswers()`     | `Rate{30, 1}`: a wrong answer gives nothing      |
| `TilesFortified` | `tiles_fortified` | `TilesFortified()`   | `Rate{1, 5}` (0.2 a tile), floored on the total  |
| `MessagesSent`   | `messages_sent`   | none                 | 0: XP for messages would fill the chat with spam |

Integer math only: every fraction is floored. `TestEverySourceThatGivesXPIsNamedOnTheWire` fails for a source that
gives XP with no `XpSource`, and for an `XpSource` no source gives.

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

The rates are first guesses, sized so a mission pays about the XP of the play it asks for. Once the reveal ships, it
is all a mission pays: missions grants no charge (see [Missions pay XP](missions.md#missions-pay-xp)).

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
  level a live raise wrote. **`Forget(accounts...)`** deletes. Nothing writes a level lower: a level the players saw
  stays.
- `reached_at` is the time of the last write that changed the level.

## Raised live

**`raise_level_usecase`, group `progression-levels` on `progression.v1.StatsChanged`** (`stats_changed_subscriber`, as
the titles' `player-titles`):

1. Read the stats. None (`ErrNoStats`): nothing to do.
2. The level the stats reach (`Scale.Reach`) not above the level kept: stop. No call to player, no write. Most facts end
   here.
3. Ask player whether the account is linked (`rpc_player_accounts`, as the titles do). A guest stops here.
4. `Raise`, through the book: in one transaction, the store writes only above the level kept, so a raise another write
   beat writes nothing, and a raise that writes records `bonuses.v1.KeepUnlocks` with every kind its level unlocks
   ([The unlocks](#the-unlocks)) and, from level 2 on, `bonuses.v1.PayLevelUp` ([The level-up
   bonus](#the-level-up-bonus)).
5. Hand the raise to `inprocess_level_feed`: the level raised to, on its bar (`Scale.StandingAt`).

- **One write per jump**: from 1 to 9 is one raise, and one `level_reached`. The use case answers the level it raised
  to, 0 for none.
- **Every fact tells the XP it gave** (see [Told live: the XP gained](#told-live-the-xp-gained)): at step 2, an account
  with a level kept, so linked, is told what the stats give past what its streams knew; at step 5, before the level
  reached.
- **It tells only the account's open streams, in this process, at most once**: the feed drops for a stream 8 raises
  behind, and an account with no stream open hears nothing. Only the bonuses' command crosses to another module: there
  is no `LevelReached` event. The use case calls the feed itself, after the raise: a raise another write beat, a guest
  and a failure tell nothing.
- A failure to read or to ask player is an error: the event is tried again, then buried.

## Told live: the XP gained

**The account's open streams hear the XP each fact gave, by source, and the level on its bar after it**
(`ListenForEvents`' `xp_gained`), so the page shows each gain with why: "+100 XP, first take of the day". Asked by the
owner on 2026-11-18: players did not see where their XP came from. XP is never stored, and `StatsChanged` names only
the account, so a gain is a difference:

- **`Scale.EarnedOf(record)` is the XP each source gives** (`Earned`, a `Gain{Source, XP}` per source, in the scale's
  order), and **`Earned.Since(before)` what each gives past it**, the sources with nothing more left out.
- **`inprocess_level_feed` keeps, per account with a stream open, what it knows the account earned**, and only then:
  the last stream to close forgets it. **A stream starts from the stats as it opens**: `listen_for_events_usecase` reads
  them (`ErrNoStats` is nothing earned) and hands them to `Subscribe`; a stream that cannot read them fails, since from
  no start the first gain told would be the account's whole XP. A second stream of the account keeps what the first
  knows, if it is more (`Earned.Most`).
- **`Feed.Count(account, earned, standing)`** tells the difference, and keeps the higher of each source: a fact told
  twice, or a record read before one already counted, tells nothing, and a gain lost to a full stream is not told
  again. `raise_level_usecase` calls it on every `StatsChanged` it reads: **only for an account with a level kept**
  (linked: a guest has no row), or after the raise that writes one, so a guest hears no XP.
- **The gain comes before the level it brings**: the feed sends both on one channel (`levels.Told`: `Gained` or
  `Reached`), in the order told, and the use case counts before it publishes the raise. The page moves the XP on the
  gain and the level on `level_reached` alone.
- **A take is one gain**: an enclose of a continent is one batch, so one gain, and a player who clicks fast hears a
  gain per batch. The feed holds 32 per stream, then drops.
- `XpGain.source` is `progression.v1.XpSource`, mapped strictly by name (`progressionmessage.XPSource`): a gain whose
  source the wire does not name is left out, which the test above keeps from happening.

## The reconciliation

**`progression.v1.AdminService/ReconcileLevels(dry_run)` → `{raised, removed}`**, by hand, after a deploy that
changes the scale or the curve, and once after the one that added the level. Connector tool
`progression_reconcile_levels`. `reconcile_levels_usecase`, built as `ReconcileTitles`:

- It pages `progression.stats` 500 at a time, asks player the page's accounts (`GetAccounts`, one call) and reads their
  levels (one query); `Scale.ReconciliationOf` says what to raise and remove.
- **Every linked account is raised to the level its stats reach, and never lowered**: a level the players saw stays,
  though a stricter scale no longer reaches it. A guest's row is removed. `lowered` is `reserved` in the response.
- **Raises go through `RaiseAll`**, so a page never lowers a level a live raise wrote after the page was read.
- **It pays no level-up bonus**: a reconciliation is no play.
- **It tells no stream.** It records a `bonuses.v1.KeepUnlocks` for every account of a page, in the page's transaction
  (`Book.Reconcile`): a linked one with what the level it holds after unlocks, a guest whose row it removed with none.
  Its time is the run's start, so a live raise after it wins at the bonuses. One run is the backfill of the bonuses'
  unlocks. `dry_run` counts the same and writes and records nothing; a second run changes no level.
  `audit_reconcile_levels` logs every run at Warn.

## The level-up bonus

**Each raise that writes fills every stack**: `Book.Raise` records `bonuses.v1.PayLevelUp{grant_id, account_id, level,
unlocked}` in the raise's transaction, through `outbox_bonuses`, and the bonuses module fills each kind the account may
be drawn up to its size ([The bonuses](bonuses.md#a-level-reached)). The owner's call on 2026-11-18: until then it drew
one kind, as a box.

- **One per raise, however many levels it jumps**: from 1 to 9 is one fill.
- **Reaching level 1 is no level-up**: a linked account starts there. A first raise to 2 or more pays, a guest's history
  counted once it links too.
- **A reconciliation pays none**, and a raise another write beat pays none: a fact told twice raises once.
- **The grant is named by the account and the level** (a SHA-1 uuid in `outbox_bonuses`), so a command told twice, or a
  level reached again after a forget, pays once: planet keeps the grant for good.
- **The kinds the level unlocks travel with it** (`unlocked`): `KeepUnlocks` comes on a stream of its own, in no order
  with this one, so the fill does not wait on it.

## The unlocks

**Progression keeps the ladder: which level unlocks which kind of bonus, one type per rung** (`levels.Unlock`), as the
title tracks are. The bonuses module knows no level: it is told the kinds each account unlocked, and draws from them
while its set's gates are on ([The bonuses](bonuses.md#the-gates)).

| Type            | Kind             | Level | Comes for a regular player                  |
| --------------- | ---------------- | ----- | ------------------------------------------- |
| `SpreadUnlock`  | `spread_clicks`  | 2     | the first tile (a day's first take: 100 XP) |
| `ShieldsUnlock` | `shields`        | 3     | about a dozen tiles after the first         |
| `EncloseUnlock` | `enclose_clicks` | 5     | a good first day                            |
| `BombUnlock`    | `bomb`           | 7     | about day 3                                 |
| `VirusUnlock`   | `virus`          | 10    | about a week                                |
| `MineUnlock`    | `mine`           | 13    | about day 19                                |
| `TrapUnlock`    | `trap`           | 15    | about a month (38,416 XP)                   |
| `RallyUnlock`   | `rally`          | 17    | about 45 days (57,344 XP)                   |

The owner set these levels on 2026-11-16 (before: enclose 4, bomb 5, virus 6, mine 8, rally 10, and spread and shields
open to all), and on 2026-11-23 put the trap at 15 and moved the rally from 15 to 17, so the rally stays the last unlock
([The trap](trap.md)). The trap shipped dormant, so its rung shows on the ladder before any box draws it, as the mine's
did. **A player at level 15 or 16 draws no rally from a box until 17** once `ReconcileLevels` runs, as production's `default`
has the gates on and `kinds.rally` at 0.5; the rallies it holds stay, as a gate decides only what is drawn.

- **Only the kinds that exist are on it.** A kind not on it unlocks at no level: whether anyone may draw it is the bonuses' `gates.open`. Refill
  alone is open there in `default`. A guest has no level, so it draws refill alone while the gates are on.
- **A change of the ladder is a deploy, then `ReconcileLevels`**: the reconciliation tells the bonuses every linked
  account's whole list again, at the level it holds, changed or not (`Reconciliation.Reached`). Until it runs, each
  account keeps the list its last raise told. **A kind that leaves the open kinds for a rung closes its
  `gates.open` switch only after the reconciliation**: before it, no account has it unlocked, and a player above its
  level is drawn none.
- **A `Kind` is planet's word**: `planet.v1.BonusKind` without its prefix, in lower case. `planetwords.BonusKind` maps
  it strictly, and `TestEveryKindOnTheLadderIsOneOfPlanets` fails for a rung planet does not name.
- **The bonuses are told as a command in their words**, `bonuses.v1.KeepUnlocks{account_id, kinds, decided_at}`, not an
  event: the gamemaster plays the bonuses' rule sets and progression hears the gamemaster, so the bonuses may not hear
  progression. `levels.Book` holds the transactioner, the store and the `Bonuses` port: each raise that writes and each
  reconciliation page record it in the transaction that writes the levels, through `outbox_bonuses` and
  progression's outbox (`progression.outbox`, its `outbox_event` enum, the relay `progression-outbox` on the commands'
  streams). A level kept is never told late or lost; a raise whose command fails is not kept
  (`TestARaiseWhoseCommandFailsIsNotKept`, on postgres).
- **The list is whole each time**, even on a raise that unlocks nothing new: the bonuses keep the latest by
  `decided_at`, so a list told twice, late or out of order changes nothing, and a lowering before the reveal is a
  shorter list.

`ProgressionService/GetLevel` answers the ladder (`unlocks`) for the page to draw a locked kind with its level.

## Forgotten

`player.v1.PlayerForgotten` deletes the account's level (group `progression-levels-forgotten`, `forget_level_usecase`).
A raise from a fact read before the forget, or a reconciliation page read before it, can write the row again, as a
late fact does for the stats.

## The public reads

**The level answered is the one kept** (`Scale.StandingAt`): the level of the `progression.levels` row, the XP its
stats give now (`Scale.XPOf`), and the two thresholds of the level kept (`level_xp`, `next_level_xp`). So the card,
`GetLevel` and the stream never disagree: a level shows once a raise or the reconciliation wrote it. The page draws
them: see the frontend's [The level](../../frontend/docs/level.md).

- **`ProgressionService/GetLevel`** → `{level, mission_rates, unlocks, xp_rates}`, the caller's own; `unlocks` is the
  ladder, one `Unlock{kind, level}` per rung, the kind in planet's words (`planet.v1.BonusKind`); `xp_rates` what each
  rated source pays, one `XpRate{source, xp, per}` per source (`inprocess_level_scale.XPRates`), for the page's list of
  how to gain XP: `Identified` behind the session interceptor, `NO_SIDE_EFFECTS` (`get_level_handler`, `level_query`).
- **`ProgressionService/GetProgress`**, the public card, answers `Progress.level` the same way (`progress_query`).
- **`ListenForEvents` sends `level_reached`**, a `Level`, to the caller's own open stream at each raise (see
  [Raised live](#raised-live)), and **`xp_gained`** at each fact that gave XP, before the `level_reached` it brings (see
  [Told live: the XP gained](#told-live-the-xp-gained)).
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
makes the scale stricter, `xp` can be below `level_xp` until the account's play catches up, since no level goes down:
the bar shows empty. The server
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

## After the deploy that added the unlocks

1. `progression_reconcile_levels`, dry run, then for real: it tells the bonuses every account's unlocks, the backfill.
2. Nothing changes for a player: every bonuses set keeps `gates.on` off until the reveal.
