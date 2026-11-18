# The level

A signed-in player has a level, from XP the server counts (see the backend's [Levels](../../backend/docs/levels.md)).
`app/level/` draws it. **The page computes no level and no XP**: it draws what `GetLevel`, the card's
`Progress.level` and the stream's `level_reached` and `xp_gained` answer, all the same `PlayerLevel` (the level kept, the
XP now, and the two thresholds its bar runs between), and what the server says each source of XP pays.

**Held for the reveal.** The level shows as soon as this page is deployed and the server answers one, so it ships with
the rest of the reveal (missions paying XP, the level-up bonus, the unlock levels): its PR merges only with them.

## Where it shows

- **The dock**: `LevelLine`, the level's chip, the XP on its bar ("1,366 / 5,558 XP") and the bar across the rest of
  the dock, under the clicks and the bonuses (`ClickBudgetMeter`'s `footer`). The line is a row of its own in the dock's flow: the dock grows
by its height and a gap, and only while there is a level. Drawn over the dock's bottom padding, it ran under the bonus
slots, which are taller than the clicks. Picked by
  the owner on 2026-11-16 over three placements shot at phone and desktop width: on the You button (a bar 40px wide
  barely moves), under the clicks alone, and along the status bar. A bar as wide as the dock moves visibly as the
  player takes tiles. The XP is the bar's label, so the line is still one thing, the level; the owner asked for it the
  same day, with the chip's "Lv".
- **The chip says "Lv 12"** (`level.short`: "Niv. 12", "Poz. 12", … in each language), never the bare number, which
  read as a count beside the clicks. Its `aria-label` is the whole "Level 12".
- **The level's color is `--xp`**, cyan: the chip, the bar and the Progress tab's line. The clicks' bar right above it
  is gold, and the two read as one bar when both were gold (the owner, 2026-11-16). The level-up moment stays gold: it
  is a reward.
- **The public card**: the level's chip before the name (`PlayerCard`, from `Progress.level`). Not beside names in the
  chat, the roster or the board: the streak and the worn title are already there.
- **The Progress tab and the profile**: `LevelProgress`, a large chip ("Lv 12"), "XP to level 13", the bar and the XP
  on it ("1,366 / 5,558 XP"), under the worn title on the phone and in the profile's hero on a desktop.
- **Each XP gain shows on the line, and says why**: `XpGainPop`, in the place of the XP for a moment: the XP in a pill
  of `--xp` ("+100 XP"), then why ("First take of the day", "Tiles taken", "Mission done", "Bomb dropped", "Box
  caught", "Right answer", "Tiles fortified"), and the bar glows while it fills. The pill pops in and bumps at each gain
  added to it. Asked by the owner on 2026-11-18: players did not see where the XP came from, but for the missions, and
  not every player opens a bubble; a full-screen moment was too much, a small toast that bumps was what he asked for.
  It is the line itself and not a toast over the globe: the gain shows where the XP is, and covers nothing (the
  zones, [`DESIGN.md`](../DESIGN.md)).
  - **One at a time, one per source** (`domain/xpPops.ts`): a gain of the source shown, or of one waiting, adds to it.
    **The takes count up while the player clicks** ("+1 XP", "+2 XP", … "Tiles taken"), which says a tile is 1 XP, and
    go 1.8 s after the last; any other gain shows 3.2 s. **Any other source takes the line from the takes at once**,
    and the takes wait behind it and keep counting: a player who keeps clicking would never let a box caught show. The
    day's first take shows its day before its tile.
  - **The pops run on the store's own clock** (`levelStore.ts`), not the line's: a gain told while a sheet covers the
    dock is not shown late.
  - A screen reader hears every gain but the takes, which come at each click.
- **What the bar is, what the levels give, and how to gain XP**: `XpHelp`, a clear button laid over the dock's line and
  over the Progress tab's level. A mouse over it, or a press, opens a `Bubble` above it, 320px wide. The bubble says what
  the bar is: "XP to level 13" and the XP on the bar ("1,366 / 5,558 XP"); what the levels give: the next kind a level
  unlocks ("Next: Mine, unlocks at level 13", with the kind's box), or "Every bonus is unlocked" past the ladder's top,
  and that each level up also fills every bonus the player unlocked
  ([the level-up bonus](../../backend/docs/levels.md#the-level-up-bonus)): "Each level up also fills all your bonuses"
  since 2026-11-18, when a level up stopped drawing one bonus.
  Then it lists the ways to gain XP, each with what it pays, from `GetLevel`'s `xp_rates`: take tiles (+1 XP), the first
  take of each day (+100 XP), missions ("Shown on each": each mission says its own), bombs, boxes, quizzes, and fortify
  (+0.2 XP: 1 per 5 tiles, said per tile). A press keeps it open until a press elsewhere, Escape or a scroll. No "?"
  icon: a line holds two things at most ([`DESIGN.md`](../DESIGN.md)). A source added to the backend's
  [scale](../../backend/docs/levels.md#what-gives-xp) needs its line in `WAYS` and its reason in `XpGainPop`. Asked by
  the owner on 2026-11-16, after players asked what the bar was. Until 2026-11-18 it said "Each new level gives you a
  bonus", for the level-up bonus, which the owner read as a new kind at every level, when 7 levels unlock one (2, 3, 5,
  7, 10, 13 and 15).
- **The level-up moment**: `LevelUp`, in the announcements queue (`{kind: "level", from}`, keyed by the level), one
  full-screen moment at a time. It is the title unlock's stage (`TitleUnlocked.css`, `TITLE_REVEAL`, the `title`
  sound) with the level's number in a gold block where the medal lands. Under the level, each kind the raise unlocked
  ("New bonus unlocked: Mine", `unlockedBy` from the level shown before, `from`), and the next one ("Next: Rally,
  unlocks at level 15", `nextUnlock`), both from the ladder `GetLevel` answers. An account's first level names no kind
  unlocked: the page knew no level before it. Nothing closes it before it has landed. A jump of several levels is one
  `level_reached`, so one moment, which names every kind it passed. A level reached while no tab is open is never
  announced. **The bonuses a level up fills are not named**: they come as one grant on planet's stream, which says no
  reason, and a box or a quiz answered right paid at the same moment would be taken for it.

## The numbers

- **One read of `GetLevel` serves the whole page.** `LevelReader.readLevel` (`ConnectProgressionBackend`) answers the
  level on its bar, the level each gated kind opens at and what each kind of mission pays (`LevelRead`), with the
  session held, and nothing with none, so it never makes one. No other part of the page calls `GetLevel`.
- **`useLevel`** (`levelStore.ts`) holds it: the page's own level (`own`), the unlocks, which the dock, the Progress
  tab, the profile and the bonus bar's locks all read, the missions' rates (`missionRates`, see
  [Missions](missions.md)), what each other source of XP pays (`xpRates`), and the gains to show (`pops`). **`useLevelFeed`**, which `PlayScreen` calls, reads at load, on
  each new session (`listenForSessions`: a guest's first click makes one) or account, and, signed in, once its clicks
  settle (`useReadsAfterClicks`, as the titles); it forgets both when the account signs out or changes. The stream's
  `level_reached` goes to the same store (`reached`), from `PlayScreen` (`useProgressionEvents`, the titles' stream),
  so a level reached opens the bar's kinds at once, and so does `xp_gained` (`gained`): the gains to show, and the bar
  moved at once.
- **`gainedOn`** (`domain/level.ts`): a gain moves the XP on the level shown, never the level. The server tells the gain
  before the `level_reached` the same facts bring, with the level after it; the level changes with `level_reached`
  alone, so the moment knows the level before it. Meanwhile the bar shows full.
- **`newerLevel`** (`domain/level.ts`): a read sent before a raise or a gain never takes the level, or the XP on its
  bar, back down.
- **`barOf`** (`domain/level.ts`) is the bar: the XP gained on the level and what the level asks. The server sends the
  XP as it counts it, which passes the next threshold until the raise lands (the bar shows full) or sits under the
  level's own after a stricter scale (empty): the bar clamps.
- **A guest draws no level**: the read answers its unlocks alone (the bar locks by them), and its card has no level. A signed-in account the server keeps
  no level for draws nothing either, until its first `level_reached`, which shows the level and plays the moment.

## Not here

How a mission shows its XP (see [Missions](missions.md)), how the bonus bar draws its locks (see
[Charges](charges.md#the-inventory)) and the next unlock belong to the missions' and the bonus bar's work.

## The fake

Every fake player has a level from its seed, on the server's curve (`fakeLevels.ts`, the fake's own copy). `FakeAccountBackend.readLevel`
answers it with the server's ladder and its XP rates (`FAKE_XP_RATES`). `FakePresenceBackend.counting` wraps the
clicker: each take is 1 XP, and the page's first one 100 more, as the day's first take, told as gained. `giveXp(500)`
adds XP to the signed-in player as a mission's, `giveXp(30, "boxesCaught")` as any source's: the gain shows on the
line, and the level-up plays when it passes a level (see [Commands](commands.md)).
