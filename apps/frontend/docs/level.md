# The level

A signed-in player has a level, from XP the server counts (see the backend's [Levels](../../backend/docs/levels.md)).
`app/level/` draws it. **The page computes no level and no XP**: it draws what `GetLevel`, the card's
`Progress.level` and the stream's `level_reached` answer, all the same `PlayerLevel` (the level kept, the XP now, and
the two thresholds its bar runs between).

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
- **What the bar is, and how to gain XP**: `XpHelp`, a clear button laid over the dock's line and over the Progress
  tab's level. A mouse over it, or a press, opens a `Bubble` above it. The bubble says what the bar is: "XP to level 13",
  the XP on the bar ("1,366 / 5,558 XP") and that each new level gives a bonus ([the
  level-up bonus](../../backend/docs/levels.md#the-level-up-bonus)). Then it lists the ways to gain XP: take tiles, play
  every day, missions, bombs, boxes, quizzes, fortify. A press keeps it open until a press elsewhere, Escape or a
  scroll. No "?" icon: a line holds two things at most ([`DESIGN.md`](../DESIGN.md)). It names the ways, not the XP
  each gives, since the page holds none of the server's weights: a source added to the backend's
  [scale](../../backend/docs/levels.md#what-gives-xp) needs its line in `WAYS`. Asked by the owner on 2026-11-16,
  after players asked what the bar was.
- **The level-up moment**: `LevelUp`, in the announcements queue (`{kind: "level"}`, keyed by the level), one
  full-screen moment at a time. It is the title unlock's stage (`TitleUnlocked.css`, `TITLE_REVEAL`, the `title`
  sound) with the level's number in a gold block where the medal lands. Nothing closes it before it has landed. A jump
  of several levels is one `level_reached`, so one moment. A level reached while no tab is open is never announced.

## The numbers

- **One read of `GetLevel` serves the whole page.** `LevelReader.readLevel` (`ConnectProgressionBackend`) answers the
  level on its bar, the level each gated kind opens at and what each kind of mission pays (`LevelRead`), with the
  session held, and nothing with none, so it never makes one. No other part of the page calls `GetLevel`.
- **`useLevel`** (`levelStore.ts`) holds it: the page's own level (`own`), the unlocks, which the dock, the Progress
  tab, the profile and the bonus bar's locks all read, and the missions' rates (`missionRates`, see
  [Missions](missions.md)). **`useLevelFeed`**, which `PlayScreen` calls, reads at load, on
  each new session (`listenForSessions`: a guest's first click makes one) or account, and, signed in, once its clicks
  settle (`useReadsAfterClicks`, as the titles); it forgets both when the account signs out or changes. The stream's
  `level_reached` goes to the same store (`reached`), from `PlayScreen` (`useProgressionEvents`, the titles' stream),
  so a level reached opens the bar's kinds at once.
- **`newerLevel`** (`domain/level.ts`): a read sent before a raise never takes the level shown back down.
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
answers it with enclose at 4 and the bomb at 5, as the server's ladder. `giveXp(500)`
adds XP to the signed-in player and plays the level-up when it passes a level; XP below a level shows at the next
read, after a run of clicks (see [Commands](commands.md)).
