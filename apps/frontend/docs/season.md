# The season

`backends/season.ts` is the contract, `seasonBackend.ts` reads
`scoring.v1.SeasonService/GetSeason` once per page load (a cached GET), and
`fakeSeasonBackend.ts` answers Season 0 in fake mode. A 404 reads as no season.

- `domain/seasonClock.ts` — `countdownsOf`, the chip's faces: the time left to
  today's cutoff to the second (`13h 05m 12s`, `52m 10s`), and to the season's
  end. **A day ends at the time of day the season's last day ends**
  (`Season.lastDayEndsAt`, the backend's `rounds.Day`): worked out here from the
  season, so the chip needs no race. From the last day's end to the season's end
  there is no day, and one face, the season's; a last day that ends with the
  season is one face too. After the season, none. `endClock` is the season's
  countdown unit by unit, for the home page, and `endsLine` its end in the
  player's own time zone ("Sat 31 Oct · 23:00").
- **No battle is counted down to.** Since 2026-10-27 the Final Battle is an entry of the
  gamemaster's schedule (see [Battles](battle.md)), which an operator may still move: the
  client knows a battle only once it plays. `GetSeason`'s `finale_starts_at` is read only
  from a server from before `last_day_ends_at`, which sent the same time under it.
- `domain/champion.ts` — `championOf`, the countries ranked 1st once a finale
  closed, `reigningChampionOf`, that champion while no newer season is current,
  and `winsLine`, "Germany wins Season 0", for the reveal, the chip and the chat.
- `app/season/` — `useSeason`, which drops the season at its end (a page open
  across it goes back to no season), `SeasonChip`, `ChampionChip`, `SeasonDetails`,
  `SeasonFacts`, the rows both of them open on, `useRotation`, which turns the
  chip's faces, and `RoundReveal` with `useRoundReveal` (see [The cutoff,
  revealed](cutoff.md)): a day's reveal, and the season's results after the Final
  Battle's brief (see [Battles](battle.md#the-brief)).

**The season is a chip in the status zone, and it turns between two
countdowns**: today's cutoff, because the day is what scores, and the season's
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

**While a battle plays, the chip is the battle's** (`BattleChip`, see
[Battles](battle.md)): the season's countdowns move into its facts, and the season
chip is back when the battle ends.

**After the season, the chip is the champion's** (`ChampionChip`): a trophy,
"Season 0 champion", the flag and the name, from the race's `closed` once it is
the finale. It replaces the countdown as soon as the finale of the current
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

**The home page counts down to the season's end.** `src/home.ts` reads the
season with the same client, and `app/home/season.ts` fills the pill in the hero
("Season 0 · ends in") and the `#season` section (when it ends, the countdown,
the rows of the chip), both `hidden` until a season is known and again once it is
over. The page's text stays in `index.html`; the module only writes the numbers. **It imports no CSS**:
a stylesheet shared with `play.html` becomes its own file, linked before
`play-*.css`, and moves the game's cascade. The art's styles are copied into the
page's `<style>`, as `.panel` and `.button` are.

**The Hall of Fame is a trophy button in the status zone** (`app/hallOfFame/`), once a season has ended: beside
the season chip or the champion chip, or alone where they were. It opens a card (`HallOfFame`, a `Modal`) with each
season ended, newest first: the trophy (`Trophy`, the art of `SeasonFacts`) and the champion, the countries ranked 1
of its final table (`domain/hallOfFame.ts`, every country level on first place is crowned), the top 10 of that table
and the player's country under it, and the top players (`StandingsTable`). A name opens the player card by its
account id, and closes the Hall of Fame: two `Modal`s at once fight over the keyboard. `backends/hallOfFame.ts` is
the contract, `hallOfFameBackend.ts` reads `GetHallOfFame`, and `fakeHallOfFameBackend.ts` answers a sample ended
season in fake mode (`sampleEndedSeason`, a function so the build drops it). **`useHallOfFame` reads it at load, and
again when the race's closed round is a finale the hall does not hold yet**: the read is cached for a minute, so it
tries again every 20s, six times at most. So the trophy shows within about a minute of the season's end on a page
already open. The home page names the last season's champion (`app/home/champion.ts`, the `[data-champion]` pill in
the hero).

**The desktop chip writes its bottom edge on `:root` as `--status-bottom`**
(`useBottomEdge`), and on a phone the status bar does: the quiz and the bomb
news sit under it. With neither, the property is unset and they sit at the top.
