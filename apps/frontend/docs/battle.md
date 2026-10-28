# Battles

**The game shows when the schedule plays a battle.** The backend's gamemaster keeps a
schedule of rule-set entries, and each entry has a kind (the backend's
[gamemaster](../../backend/docs/gamemaster.md)). The client never hears a rule set's
name, only the kind of the entry that plays, from `gamemaster.v1.GamemasterService/ListenForEvents`:
`battle` or `plain`, since when, when the running entry ends, the entry's id, the
battle's round and times, the battles to come, and the last debrief (a day, a battle
or a season scoring scored). A battle is the one kind with a look.

## Reading what plays

- `backends/turn.ts` is the contract: `Turn` (`kind`, `since`, `endsAt`,
  `description`, `entryId`, `roundId`, `times`), `BattleToCome` (also `times`,
  `endsSeason`, `roundId`), `Debrief` (`day`, `battle` or `season`, each with its ids
  and the gamemaster's words: the day's number, the battle's number and times, the
  season's book and its finale round) and `TurnBackend`, whose `follow` hands on the
  turn, the battles to come, the seasons (`onSeasons`, see [The season](season.md))
  and the last debrief (`onDebrief`) at once, then each change of any. `turnBackend.ts` follows the stream (`openStream`:
  heartbeats, reconnects, a page wake); no read polls. A server without the
  gamemaster (`Unimplemented`, `NotFound`) reads as a plain turn with no battle to
  come, no season and no debrief, and the stream is not opened again. A turn with no kind, or a kind this build
  does not know, is refused: the switch names every value. `fakeTurnBackend.ts` is the
  fake, with `startBattle(entryId, since, endsAt, description, times)`, `endBattle()`,
  `endDay()`, `endBattleRound(…)` and `endSeason(finaleRoundId, finaleBattle)` for the debriefs, and
  `scheduleBattle`, `moveBattle`, `deleteBattle` for the battles to come, and
  `moveSeasonEnd`, which tell of the change at once. It answers Season 0 as the
  season that runs (`SEASON_ZERO`, book `season-0`), and today as the gamemaster counts
  days (day K ends at the K-th 21:00 UTC cutoff after the start).
- **A battle is its entry's id** (`entryId`, empty when no entry runs): the battle's
  key and the start moment's "seen" mark. **Its round is `roundId`**, scoring's id for
  it: the board's battle column reads it, and a debrief is matched to the battle the
  page saw by it. Two battles are never told apart by their times (the owner,
  2026-10-27).
- **`description` is the entry's, the operator's words for the players**, written
  on the backoffice's Rule schedule page (at most 300 characters, the backend's
  rule). Empty when the operator wrote none, or when no entry runs.
- `app/battle/useSchedule.ts` holds the turn and the battles to come, as the stream
  sends them. The end of a battle needs no read of its own: the stream sends the next
  turn within a second.
- `domain/battle.ts` — `battleOf(turn)`, the `Battle` a battle turn is (nothing for
  a plain turn or a battle with no end), its key (its `entryId`), its clock
  (`battleLeft`, `timeLeftTo` from the season clock, stopped at zero), and
  `battleEndsLine`, "Ends at 23:00" in the player's time, with the weekday when the
  end is not today.

## The battles to come

**The next battle is counted down to, on the season chip and on the home page**
("Next battle in", see [The season](season.md)). `domain/battle.ts`'s `nextBattleOf`
is the first battle to come that has not started at the client's now: the server's
list is of its last read, and a battle whose start passed leaves the countdown before
its turn arrives. With none to come there is no face and no pill. During a battle the
battle chip is the chip, as below. The countdown says "next battle", never "final":
an entry of the schedule carries no fact of the season.

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
battle ended while the page was open. **When the battle has a round, its results
follow as the debrief** (below): until the gamemaster's last debrief names that round
(`briefAwaited`, a `BattleEnded` or a `SeasonEnded` whose finale it is), the line adds
"The results are coming" and stays, with no timer. Then it goes, as the debrief comes.

## The debrief

**A battle debriefed plays three full screens, one after the other**
(`BattleDebrief`, rules in `domain/battleBrief.ts`): the words are the debrief's, the
numbers its round's, read from scoring's `GetRound` (see [The cutoff,
revealed](cutoff.md) for when a debrief plays, and once per browser). Each has the battle's look: the warning tape along the top of the
screen and the rays in `--war`.

- **Which battle**: the round's `roundId` is the battle the page saw in the turn
  (`useSeenBattle` keeps the last one), or the page saw none (it opened after the
  battle). Another battle's round plays no debrief (`debriefedFor`); the season's
  results after a finale still play.
- **The screens** (`debriefScreensOf`), each its own announcement, so each closes on
  its button and the next one starts:
  1. `over` — "Battle finished!", or "Final Battle finished!" for the round that ended
     the season (`battleFinishedLine`), over the swords disc, with the battle's words
     when the page saw it play (`briefDescriptionOf`, by its round), and the
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
- **When the battle ends the season (`endsSeason`), the season's results follow** as
  a fourth announcement: the table before and after, then the champion (see
  [The cutoff, revealed](cutoff.md)). The gamemaster tells only its last debrief, and
  scoring scores the Final Battle and closes the book at one instant: the page may
  hear `SeasonEnded` alone. So `SeasonEnded` carries the Final Battle's debrief
  (`finale_battle`) when a battle finished the season, and plays its three screens
  before the results; with none, only the results play. The screens are keyed by the
  round, so they never play twice.

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
board's battle column follow at once. `endBattle()` leaves the battle's round
undebriefed, so "The results are coming" stays; `closeBattle()` scores it and the fake
gamemaster debriefs it, so the debrief plays; `closeFinale()` debriefs it as the Final
Battle, then the season, and the season's results follow.
