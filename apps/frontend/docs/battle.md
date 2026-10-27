# Battles

**The game shows when the schedule plays a battle.** The backend's gamemaster keeps a
schedule of rule-set entries, and each entry has a kind (the backend's
[gamemaster](../../backend/docs/gamemaster.md)). The client never hears a rule set's
name, only the kind of the entry that plays, from `gamemaster.v1.GamemasterService/GetTurn`:
`battle` or `plain`, since when, and when the running entry ends. A battle is the
one kind with a look.

## Reading what plays

- `backends/turn.ts` is the contract: `Turn` (`kind`, `since`, `endsAt`,
  `description`) and `TurnBackend`, with `turn()` and `changes()`, which tells when
  what it answers may have changed. `turnBackend.ts` reads `GetTurn` as a cached GET
  and tells of a change on a page wake (`pageWakeups`: a phone that locked may have
  slept through a start). A server without the gamemaster (`Unimplemented`,
  `NotFound`) reads as a plain turn. A turn with no kind, or a kind this build does
  not know, is refused: the switch names every value. `fakeTurnBackend.ts` is the
  fake, with `startBattle(since, endsAt, description)` and `endBattle()`, which tell
  of the change at once.
- **`description` is the entry's, the operator's words for the players**, written
  on the backoffice's Rule schedule page (at most 300 characters, the backend's
  rule). Empty when the operator wrote none, or when no entry runs.
- `app/battle/useTurn.ts` holds the turn: read at mount, read again on a change,
  at the running entry's end, and once a minute otherwise (`domain/battle.ts`,
  `nextReadAt`): an entry at a higher priority may start before the end, and the
  answer is 5 s fresh. A read that fails is logged and tried again at the next
  tick.
- `domain/battle.ts` — `battleOf(turn)`, the `Battle` a battle turn is (nothing for
  a plain turn or a battle with no end), its key (`since:endsAt`), its clock
  (`battleLeft`, `timeLeftTo` from the season clock, stopped at zero), and
  `battleEndsLine`, "Ends at 23:00" in the player's time, with the weekday when the
  end is not today.

## The look

**The battle is a mood** (see [`DESIGN.md`](../DESIGN.md#moods)): `useMood` writes
`data-mood="battle"` on the root while a battle plays, and `tokens.css` redefines
the panels to charcoal and the secondary button to steel under it. The warning tape
(`--tape`) runs along the top of the dock (`ClickBudgetMeter.css`, under the mood)
and of the battle chip. **The globe's halo turns to ember**: `Viewer` hands
`useGlobe` a `halo` (`calm` or `war`), the globe's `setHalo` repaints the
atmosphere's colour (`app/viewer/atmosphere.ts`) and asks for a frame. The sky, the
gold and the text do not move: a coin, a header or a reward looks the same in a
battle.

## The chip

**While a battle plays, the status zone's chip is the battle's** (`BattleChip`),
in the season chip's place and shape (its `season-chip` classes, with its own
colour, glow and tape in `Battle.css`): crossed swords, "Battle ends in", and the
clock to the second. The season chip, or the champion's, is back when the battle
ends.

- On a desktop a hover opens its facts under it, and a press holds them open
  (Escape closes a pressed one; a hover-only one closes when the pointer leaves).
  On a phone it is the right end of the status bar, "Battle" over the two largest
  units, and a press opens the "Battle" sheet (`BattleDetails`): the clock and the
  same facts.
- **The facts are the description, when the operator wrote one, and when the battle
  ends.** The season's countdowns follow (today's end and the season's), so a player
  loses no clock while the chip is the battle's.
  The facts explain no rule on their own: what a battle changes is the operator's
  description, read from the server.
- The chip's name holds the clock in full, and the visible clock is hidden from a
  screen reader, as the season chip's is.

## The start moment, and the end

**The start plays once per battle per browser** (`BattleReveal`, `useBattleReveal`):
the title unlock's slam, "Battle!", a disc of crossed swords in the battle's
colour, the description and when it ends, then "To the globe" once it settled
(`TITLE_REVEAL.ready`). Close keeps the battle's key in `clickplanet-battle-seen`,
so a reload does not play it again; a player who arrives mid-battle sees it, since
the battle is on. It plays through the announcements queue (below).

**At the end** the mood goes, the halo turns blue again, and one line in the moments
zone says "The battle is over" (`BattleOver`, `useBattleOver`, 5 s), only when the
battle ended while the page was open. **When the battle has a round of its own, its
results follow as the brief** (below), within the race feed's 10 s: while the race
still shows that round in progress (`briefAwaited`: `Race.round` is of its own and
ends at or before the battle's end), the line adds "The results are coming" and stays,
with no timer, until the race carries the round closed. Then it goes, as the brief
comes.

## The brief

**A battle's round of its own closed plays its brief, full screen** (`BattleBrief`,
rules in `domain/battleBrief.ts`): from the race's `closed` when it is `own` (see
[The cutoff, revealed](cutoff.md) for when a closed round plays, and once per
browser). It has the battle's look: the warning tape along the top of the screen,
the rays in `--war`, and "Points ×3" (the round's `times`) in a war chip under the
head. The head is `roundOverLine`: "The battle is over!", or "The Final Battle is
over!" for the round that ended the season.

- **Three stages, each skipped when it has nothing**: who won (the podium of
  `standings`, `RoundReveal`'s `Podium`, the points already ×times), the tiles each
  flag took (`takes`, a bar each against the flag that took the most, the first
  `TAKES_SHOWN` (8) and the player's under them), then the top players (`players`:
  rank, flag, name in its colour, the title worn, tiles; the player's own row
  marked). Each stage moves on after `BATTLE_BRIEF` (3.4 s, 4.2 s), or on Next; the
  last one shows Close after `BATTLE_BRIEF.settle` (1.8 s). Escape closes it then.
- **The description is the turn's, when the page saw the battle play**
  (`useSeenBattle` keeps the last battle seen, `briefDescriptionOf` takes its words
  when its window is the round's: one end or one start). A page opened after the
  battle shows the brief with none.
- **When the closed round is also the finale, the season's results follow** as a
  second announcement: the table before and after, then the champion (see
  [The cutoff, revealed](cutoff.md)).

## The announcements queue

**One full-screen moment at a time, in the order they arrive.** `domain/announcements.ts`
and `app/announcements/useAnnouncements.ts` keep the queue: a title unlocked, a
battle's start, a day closed (the cutoff's reveal), a battle's brief, the season's
results. `Viewer` announces each as it learns of it, shows the head of the queue once
the map is loaded, and the next starts when the player closes the current one. An
announcement is queued once, by its key. So a round that closes while the player
reads a battle's screens waits its turn, the season's results wait for the brief
queued before them, and nothing is drawn over anything.

The round's "once per browser" and the battle's keep their own rule (`useRoundReveal`,
`useBattleReveal`): the queue only orders what is due.

## In fake mode

`startBattle(minutes = 120, description, times = 3)` and `endBattle()` on the console
(see [Commands](commands.md)): the mood, the halo, the chip, the moment and the
board's battle column follow at once. `endBattle()` leaves the round of its own in
progress, so "The results are coming" stays; `closeBattle()` closes it and plays the
brief, `closeFinale()` closes it as the Final Battle, and the season's results follow.
