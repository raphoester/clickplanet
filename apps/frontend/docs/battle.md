# Battles

**The game shows when the schedule plays a battle.** The backend's gamemaster keeps a
schedule of rule-set entries, and each entry has a kind (the backend's
[gamemaster](../../backend/docs/gamemaster.md)). The client never hears a rule set's
name, only the kind of the entry that plays, from `gamemaster.v1.GamemasterService/GetTurn`:
`battle` or `plain`, since when, when the running entry ends, and the entry's id. A
battle is the one kind with a look.

## Reading what plays

- `backends/turn.ts` is the contract: `Turn` (`kind`, `since`, `endsAt`,
  `description`, `entryId`) and `TurnBackend`, with `turn()` and `changes()`, which tells when
  what it answers may have changed. `turnBackend.ts` reads `GetTurn` as a cached GET
  and tells of a change on a page wake (`pageWakeups`: a phone that locked may have
  slept through a start). A server without the gamemaster (`Unimplemented`,
  `NotFound`) reads as a plain turn. A turn with no kind, or a kind this build does
  not know, is refused: the switch names every value. `fakeTurnBackend.ts` is the
  fake, with `startBattle(entryId, since, endsAt, description)` and `endBattle()`,
  which tell of the change at once.
- **A battle is its entry's id** (`entryId`, empty when no entry runs): the battle's
  key, the start moment's "seen" mark, and how a closed round is matched to the
  battle the page saw. The scoring round of its own repeats the id the gamemaster
  sent it (the backend's [gamemaster](../../backend/docs/gamemaster.md)). Two battles
  are never told apart by their times (the owner, 2026-10-27).
- **`description` is the entry's, the operator's words for the players**, written
  on the backoffice's Rule schedule page (at most 300 characters, the backend's
  rule). Empty when the operator wrote none, or when no entry runs.
- `app/battle/useTurn.ts` holds the turn: read at mount, read again on a change,
  at the running entry's end, and once a minute otherwise (`domain/battle.ts`,
  `nextReadAt`): an entry at a higher priority may start before the end, and the
  answer is 5 s fresh. A read that fails is logged and tried again at the next
  tick.
- `domain/battle.ts` — `battleOf(turn)`, the `Battle` a battle turn is (nothing for
  a plain turn or a battle with no end), its key (its `entryId`), its clock
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
results follow as the debrief** (below), within the race feed's 10 s: while the race
still shows that round in progress (`briefAwaited`: `Race.round` is of its own and
of the battle's entry), the line adds "The results are coming" and stays, with no
timer, until the race carries the round closed. Then it goes, as the debrief comes.

## The debrief

**A battle's round of its own closed plays three full screens, one after the other**
(`BattleDebrief`, rules in `domain/battleBrief.ts`), from the race's `closed` when it
is `own` (see [The cutoff, revealed](cutoff.md) for when a closed round plays, and
once per browser). Each has the battle's look: the warning tape along the top of the
screen and the rays in `--war`.

- **Which battle**: the round's `entryId` is the battle the page saw in the turn
  (`useSeenBattle` keeps the last one), or the page saw none (it opened after the
  battle). A round of another entry plays no debrief (`debriefedFor`); the season's
  results after a finale still play.
- **The screens** (`debriefScreensOf`), each its own announcement, so each closes on
  its button and the next one starts:
  1. `over` — "Battle finished!", or "Final Battle finished!" for the round that ended
     the season (`battleFinishedLine`), over the swords disc, with the battle's words
     when the page saw it play (`briefDescriptionOf`, by the entry's id), and the
     fanfare.
  2. `points` — "Points won": each country that scored (`pointsRowsOf`, the first
     `POINTS_SHOWN` (10) and the player's under them), its rank, its points before the
     times and ×times ("25 ×3"), and the total, drawn as a bar against the best; "Points
     ×3" in a war chip under the head. Skipped when no country scored.
  3. `players` — "Top players": rank, flag, name in its colour, the title worn, tiles;
     the player's own row is found by account id, never by name. Skipped with none.
- **Each shows its button after `BATTLE_DEBRIEF`** (2.4 s, 2.2 s, 2.2 s): Next while a
  screen follows (`revealFollowed`), "To the globe" on the last. Escape closes it then.
  The tiles each flag took (`takes`) are still on the wire and no longer shown.
- **When the closed round is also the finale, the season's results follow** as a
  fourth announcement: the table before and after, then the champion (see
  [The cutoff, revealed](cutoff.md)).

## The announcements queue

**One full-screen moment at a time, in the order they arrive.** `domain/announcements.ts`
and `app/announcements/useAnnouncements.ts` keep the queue: a title unlocked, a
battle's start, a day closed (the cutoff's reveal), a battle's three debrief screens,
the season's results. `Viewer` announces each as it learns of it, shows the head of the queue once
the map is loaded, and the next starts when the player closes the current one. An
announcement is queued once, by its key. So a round that closes while the player
reads a battle's screens waits its turn, the season's results wait for the debrief
queued before them, and nothing is drawn over anything.

The round's "once per browser" and the battle's keep their own rule (`useRoundReveal`,
`useBattleReveal`): the queue only orders what is due.

## In fake mode

`startBattle(minutes = 120, description, times = 3)` and `endBattle()` on the console
(see [Commands](commands.md)): the mood, the halo, the chip, the moment and the
board's battle column follow at once. `endBattle()` leaves the round of its own in
progress, so "The results are coming" stays; `closeBattle()` closes it and plays the
debrief, `closeFinale()` closes it as the Final Battle, and the season's results follow.
