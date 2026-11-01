# The cutoff, revealed

When a day closes, who won it and the season's new order play full screen,
once (`app/season/RoundReveal.tsx`; the rules are in `domain/roundReveal.ts`).

- **It plays from the gamemaster's debrief.** The gamemaster's stream carries the
  last moment scoring scored (`Debrief`: a day, a battle or a season), in its words
  (Day 24, the battle's number and times, the finale, Season 0) and with its ids.
  `useDebriefed` reads the numbers from scoring by them: the round's results, takes,
  players and the book before and after it (`GetRound`), or, for a season no round
  finished, the book's scores (`GetScores`). `domain/debrief.ts`'s `closedOf` joins
  the two into a `ClosedRound` (`moment`: `day`, `battle` or `season`), and
  `useRoundReveal` reads it.
- **A round is scored at its end**, and its debrief reaches the page within a second.
- **What a debrief plays** (`revealsOf`, in this order): a day, its reveal
  (`round`, below); a battle, its debrief (`over`, `points`, `players`, see
  [Battles](battle.md#the-debrief)), when it is the battle the page saw, and the
  season's results after it when it ended the season (`results`); a season, its
  results, after its Final Battle's screens when a battle finished it and the page
  did not play them yet (see [Battles](battle.md#the-debrief)). Each is one announcement.
- **When**: a round that ended less than 6 hours before the page opened
  (`REVEAL_WITHIN_MS`), or one that closes while the page is open. **Once per
  browser**: Close keeps the key of the screen closed in `clickplanet-round-seen`
  (`season:number:end` for a day, `season:battle:round:` followed by the screen's kind
  for a battle's debrief, and `season:results:round` for the season's results, `round`
  being the finale's), and `revealsDue` plays only the screens after it, after a reload
  too. Only the last key is kept. A battle's and a season's debrief of one Final Battle
  share its keys, so nothing plays twice.
- **Two stages.** First the podium: "Day 5 is over!", then the top 3 countries
  rise on steps (2nd, 1st, 3rd), each with its
  flag, its name and the points it won. Only places with points; ties share a
  rank. After `ROUND_REVEAL.podium` (3.4s), or on Next, the season table: the
  top 10 in their order before the round, and the player's country, marked,
  under them when it is not in the top 10. Then the rows slide to their new
  order, the points count up to the new total, and each row shows what it won
  and how it moved (▲2, ▼1, New, –). With no podium it opens on the table; with
  no table it stops on the podium.
- **The finale ends on the champion**, a third stage: `ROUND_REVEAL.hold`
  (2.4s) after the table settled, or on Next, the head becomes "Germany wins
  Season 0" (`winsLine`) over a trophy, rays, a flash and the flag, and the
  `title` sound plays again. The champion is every country ranked 1st in
  `closed.after` (`championOf` in `domain/champion.ts`), so countries level on
  everything win together. A day that is not the finale has no third stage.
- **The season's results are the same screen with no podium** (`results`):
  after the Final Battle's debrief, "Season 0 is over!" over the "Final table",
  before and after the battle, then the champion. The points were the debrief's.
  Until 2026-10-27 the Final Battle played podium, table and champion as one
  reveal; a season that ends with a day still does.
- **Nothing closes it before it is settled**: no button, and Escape does
  nothing, until the rows have moved (`ROUND_REVEAL.slide` + `settle`), or the
  podium has run when there is no table, or the champion has had
  `ROUND_REVEAL.champion` (2.2s). Then Close appears and Escape works.
  The rule is the title unlock's: a player spamming the globe would close it
  unseen.
- **It takes its turn in the announcements queue** (see
  [Battles](battle.md#the-announcements-queue)): `Viewer` announces the round
  closed when it is due, and shows it once the map is loaded and once whatever
  arrived before it, a title unlocked for one, was closed. A battle's start waits for the day it ended, so the day
  plays first (see [Battles](battle.md#the-start-moment-and-the-end)).
- It looks like the unlock: a portal on the body, the `title-reveal` classes of
  `TitleUnlocked.css`, and the `title` sound. Under
  `prefers-reduced-motion: reduce` the rows do not slide and the steps only fade
  in.
- In fake mode `closeDay()` scores today's round and debriefs the day, so its reveal
  plays, and `closeFinale()` the Final Battle's debrief, then the season's results
  (see [Commands](commands.md)).
