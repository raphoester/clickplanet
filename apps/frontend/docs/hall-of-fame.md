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
  big "?" over its first step.
- **The seasons are the gamemaster's timeline**: `ended`, `current` and `next` (see [The season](season.md)). The
  season that runs is `useSeason`'s, so it leaves the "?" plaque at its end on this clock, and the timeline adds it
  to `ended` within a second.
- **In fake mode** `startBreak()` ends the fake season: it becomes an ended plaque, and the next one the "?".
