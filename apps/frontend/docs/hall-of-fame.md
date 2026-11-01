# The Hall of Fame

**A gold "Hall of Fame" button heads the board place** (`HallOfFameButton`, `app/season/HallOfFame.tsx`), above
the leaderboard's heading on a desktop and under "Playing for" in the phone's Board sheet. It opens a `Modal`, wider
than the others (720px): it is the one place that shows every season.

- **One plaque per season, in a carousel**: a row that scrolls sideways and snaps on each plaque, oldest first, and
  opened scrolled to its end, on the newest. While they all fit, the plaques stand centred.
- **An ended season's plaque is its podium**: its number, its first and last dates in the player's time
  (`spanLine`, "8 Oct – 31 Oct 2026"), and its top three on gold, silver and bronze steps, 2nd, 1st, 3rd, each with
  its flag and name. The top three are the book's table from scoring (`GetScores`, ranked with the finale as the
  tiebreak, see the backend's [Scoring](../../backend/docs/scoring.md)), read when the plaque opens
  (`tablePodiumOf`): countries level share a rank and its step. A book nobody scored in says "No winner".
- **The season that runs, or else the next one, ends the row**: the same plaque on a sunk box, an empty podium and a
  big "?" over its first step. An empty slot holds the place of the View button, so every podium stands on one line.

## A season's slides

**View, under an ended season's podium, turns the modal into that season's slides** (`SeasonStats`,
`app/season/SeasonStats.tsx`, rules in `domain/seasonStats.ts`): the head becomes "Season 0", and "Hall of Fame" at
the top goes back to the plaques. The slides are a row that scrolls sideways and snaps on each, so a phone swipes them;
Previous, Next, a dot per slide and the arrow keys move between them.

1. **Top flags**: the book's table (`GetScores`), the first ten, each with its points and the days it won.
2. **Top players**: the book's board (`GetBoard`, `StandingsBackend.board`), each name in its colour with its title.
3. **Final Battle, top flags**: its round (`GetRound` by the season's `finaleRoundId`), the flags that scored, their
   points (×times in the head) and the tiles their players took (`takes`, which the debrief no longer shows).
4. **Final Battle, top players**: the round's ten best players, as the debrief's last screen.
5. **In numbers**: the season's clicks (the hero), players, days, bombs dropped, shields placed, shield hits, lands
   fortified, the flags that scored and the Final Battle's clicks (`bigNumbersOf`). The ledger's totals are the
   gamemaster's `GetSeasonTotals` (`TurnBackend.seasonTotals`, 60 s, since its first read scans the ledger).

- **A season with no finale has no Final Battle slides** (`slidesOf`).
- **Each read is its own**: a slide fills when its read lands, and a number whose read is not there is left out.
- **Each row's bar** is its value against the first row's (`Barred`), and the rows come in one after the other; under
  `prefers-reduced-motion` they do not move.
- **The seasons are the gamemaster's timeline**: `ended`, `current` and `next` (see [The season](season.md)). The
  season that runs is `useSeason`'s, so it leaves the "?" plaque at its end on this clock, and the timeline adds it
  to `ended` within a second.
- **In fake mode** `startBreak()` ends the fake season: it becomes an ended plaque, and the next one the "?". Run
  `startFinale()`, `endBattle()` and `closeFinale()` before it for the Final Battle's slides; the fake's totals are
  fixed numbers.
