# The season

**The season and its days are the gamemaster's**: `backends/season.ts` is `Season`
(`number`, `startsAt`, `endsAt`, `bookId`), `Day` (`number`, `startsAt`, `endsAt`,
`roundId`) and `Seasons` (`current`, `next`, `today`, and the seasons that `ended`, for the [Hall of
Fame](hall-of-fame.md)), which the gamemaster's
`Timeline` carries and `TurnBackend.follow` hands on (`onSeasons`, see
[Battles](battle.md#reading-what-plays)). An end the operator moves reaches the page
within a second. A server without the gamemaster reads as no season. **Scoring only
counts points**, by the ids the gamemaster names: the season's `bookId` and the
rounds' `roundId` (see [Season standings](season-standings.md)).
`scoringBackend.ts` makes the `ScoringService` client.

- `domain/seasonClock.ts` — `countdownsOf`, the chip's faces: the time left to
  today's cutoff to the second (`13h 05m 12s`, `52m 10s`), to the next battle's
  start when one is to come, and to the season's end. **Today's end is the
  gamemaster's**: `Timeline.today.ends_at`, the daily cutoff. With no day
  running (after the season's last cutoff), or a day that
  ends with the season, there is one face, the season's. After the season, none. `endClock` is the season's
  countdown unit by unit, for the home page, and `endsLine` its end in the
  player's own time zone ("Sat 31 Oct · 23:00").
- **The next battle is the schedule's** (see [Battles](battle.md#the-battles-to-come)):
  an operator may move or delete it, and the stream says so within a second.
- `domain/champion.ts` — `championOf`, the countries ranked 1st once the Final
  Battle or the season was debriefed, `reigningChampionOf`, that champion while no newer season is current,
  and `winsLine`, "Germany wins Season 0", for the reveal, the chip and the chat.
- `app/season/` — `useSeason`, the timeline's current season, dropped at its end on
  this clock (a page open across it goes back to no season), `SeasonChip`, `ChampionChip`, `SeasonDetails`,
  `SeasonFacts`, the rows both of them open on, `useRotation`, which turns the
  chip's faces, and `RoundReveal` with `useRoundReveal` (see [The cutoff,
  revealed](cutoff.md)): a day's reveal, and the season's results after the Final
  Battle's brief (see [Battles](battle.md#the-brief)).

**The season is a chip in the status zone, and it turns between its
countdowns**: today's cutoff, because the day is what scores, the next battle's
start ("Next battle in", tag "Battle", or "Final Battle in", tag "Final", when it is the
season's finale), while one is to come, and the season's
end. Each face shows for 6 seconds. A face is a label and its time
together, so a look never shows one countdown's time under the other's label.
The next face slides up into place; under `prefers-reduced-motion` it changes
with no movement. The mouse over the chip, or focus on it, stops the turns. With
one face the chip does not turn.

On a desktop it sits at the top centre: "Today ends in 13h 05m 12s", then "Season
ends in 23d 06h 12m 05s", and a press opens a dropdown (Escape closes it). On a
phone it is the right end of the status bar, the two largest units alone
("13h 05m"), under a short tag ("Today", "Season") when there are two faces. A
press opens the same details as a sheet, with both countdowns at once. The chip's
name holds both countdowns in full, so a screen reader does not wait for a turn.
The faces are hidden from it, and they are not a live region.

**Before the Final Battle, the chip wears the battle's colour.** The "Final Battle in" face is in `--war`, on the
chip and in the sheet's countdowns (`endsSeason` on the face, `season-chip-face--final`). In the two days before its
start (`finaleNear`, `FINALE_NEAR_MS`), the whole chip has the warning tape along its top and an ember glow
(`season-chip--final`); on a phone, the glow alone.

**While a battle plays, the chip is the battle's** (`BattleChip`, see
[Battles](battle.md)): the season's countdowns move into its facts, and the season
chip is back when the battle ends.

**After the season, the chip is the champion's** (`ChampionChip`): a trophy,
"Season 0 champion", the flag and the name, from the last debrief once it is the
Final Battle's or the season's (`SeasonEnded`), with the book's table read from scoring. It replaces the countdown as soon as the finale of the current
season closed, and stays until a newer season is current
(`reigningChampionOf`). It opens nothing: with no season there are no facts to
show. Its name is the chat's line, "Germany wins Season 0". On a phone it is the
trophy and the flag alone, so a long name never squeezes the player's own
country out of the status bar.

**All faces are in one grid cell**, and the faces not shown are
`visibility: hidden`. So the chip is as wide as its widest face, and its width
does not change at each turn.

**The desktop chip has a fixed width** (368px, the widest countdown plus a
little). Luckiest Guy has no equal-width digits, so the countdown changes width
every second: a chip as wide as its text moved every second and wrapped the
dropdown's rows again with it.

**What it opens is the one place the UI explains the rules**, asked for on
purpose: three rows (`SeasonFacts`), the daily rounds ("Win the day"), the
winner's trophy in the Hall of Fame, and the titles. **Say only what is
decided**: points when a day ends for the countries holding the most ground then,
a trophy for the winning country, a title for every signed-in player and one
more for the winning country's. A battle and what it is worth are the schedule's:
the battle chip says them, in the operator's words, while one plays. Guests get no title, so
the row says "signed-in".

**The Final Battle to come opens the details** (`FinaleDetails`, `app/season/FinaleFacts.tsx`), above the
season's rows, in the dropdown and the sheet, while the schedule lists the season's finale (`finaleOf`): "The
Final Battle", when it plays in the player's time, the operator's words, and three rows: "2 hours of war",
"Points ×3" (from its `times`, left out at ×1), and "The last word" (nothing scores after it, and it crowns the
champion). Each is the schedule's fact or what is always true of a finale, so no battle is promised. Once it
plays, the battle chip takes over.

**The home page counts down to the season's end, and to the next battle.** `src/home.ts` follows the
gamemaster's stream once, and `app/home/season.ts` (`seasonShown`) fills the pill in the hero
("Season 0 · ends in") and the `#season` section (when it ends, the countdown,
the rows of the chip), both `hidden` until a season is known and again once it is
over, and follows a moved end. `app/home/battle.ts` (`nextBattleShown`) fills the second pill ("Next battle in", or "Final Battle in" for the season's finale, `[data-battle-label]`, `[data-battle]`), `hidden` while
no battle is to come; an edit of the schedule moves it at once. The page's text stays in `index.html`; the module only writes the numbers.

**The home page tells the Final Battle in a panel of its own**, under the season's, in `#season`.
`app/home/finale.ts` (`finaleShown`) shows it while the season's finale is to come or plays (`finaleOf`, see
[Battles](battle.md#the-battles-to-come)), and hides it with none: the schedule decides the finale, so the page
promises none. It writes only the schedule's facts: when it plays in the player's time (`finaleWhenLine`, "Sat 31
Oct · 21:00 – 23:00"), how long ("2 hours of war", `finaleLengthLine`), its points' times ("Points ×3", the row
hidden at ×1), and the operator's words in a box (hidden when empty). Its clock counts to the start ("Starts in"),
then, live (`data-live`), to the end ("Live now · ends in"), and the panel's ember glow pulses. "The last word" says
what is always true of a finale: nothing scores after it, and the champion is crowned. The panel wears the battle
mood (`data-mood="battle"` on the panel, see [`DESIGN.md`](../DESIGN.md#moods)) and the warning tape along its top.

**It imports no CSS**:
a stylesheet shared with `play.html` becomes its own file, linked before
`play-*.css`, and moves the game's cascade. The art's styles are copied into the
page's `<style>`, as `.panel` and `.button` are.

**The desktop chip writes its bottom edge on `:root` as `--status-bottom`**
(`useBottomEdge`), and on a phone the status bar does: the quiz and the bomb
news sit under it. With neither, the property is unset and they sit at the top.
