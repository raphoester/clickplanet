# The cutoff, revealed

When a day closes, who won it and the season's new order play full screen,
once (`app/season/RoundReveal.tsx`; the rules are in `domain/roundReveal.ts`).

- **It plays from the race.** The race on the season stream carries `closed`,
  the round closed last: its season, its number, its end, whether it was the
  finale, the countries that scored in it, and the season's table before and
  after it (`ClosedRound`; the backend's [Seasons](../../backend/docs/seasons.md)). `useRoundReveal`
  reads it from the race `Viewer` already follows (`useRace`).
- **When**: a round that ended less than 6 hours before the page opened
  (`REVEAL_WITHIN_MS`), or one that closes while the page is open. **Once per
  browser**: Close keeps the round's key (`season:number:end`) in
  `clickplanet-round-seen`, and a round with that key does not play again, after
  a reload too. Only the last key is kept.
- **Two stages.** First the podium: "Day 5 is over!" or "The Final Battle is
  over!", then the top 3 countries rise on steps (2nd, 1st, 3rd), each with its
  flag, its name and the points it won. Only places with points; ties share a
  rank. After `ROUND_REVEAL.podium` (3.4s), or on Next, the season table: the
  top 10 in their order before the round, and the player's country, marked,
  under them when it is not in the top 10. Then the rows slide to their new
  order, the points count up to the new total, and each row shows what it won
  and how it moved (▲2, ▼1, New, –). With no podium it opens on the table; with
  no table it stops on the podium.
- **The Final Battle ends on the champion**, a third stage: `ROUND_REVEAL.hold`
  (2.4s) after the table settled, or on Next, the head becomes "Germany wins
  Season 0" (`winsLine`) over a trophy, rays, a flash and the flag, and the
  `title` sound plays again. The champion is every country ranked 1st in
  `closed.after` (`championOf` in `domain/champion.ts`), so countries level on
  everything win together. A day has no third stage.
- **Nothing closes it before it is settled**: no button, and Escape does
  nothing, until the rows have moved (`ROUND_REVEAL.slide` + `settle`), or the
  podium has run when there is no table, or the champion has had
  `ROUND_REVEAL.champion` (2.2s). Then Close appears and Escape works.
  The rule is the title unlock's: a player spamming the globe would close it
  unseen.
- **It waits for a title unlock to close.** `Viewer` mounts it once the map is
  loaded and only while no `TitleUnlocked` is queued, so a title earned at the
  cutoff plays first and the reveal plays after it.
- It looks like the unlock: a portal on the body, the `title-reveal` classes of
  `TitleUnlocked.css`, and the `title` sound. Under
  `prefers-reduced-motion: reduce` the rows do not slide and the steps only fade
  in.
- In fake mode `closeDay()` and `closeFinale()` play it (see
  [Commands](commands.md)).
