# The season

`backends/season.ts` is the contract, `seasonBackend.ts` reads
`seasons.v1.SeasonService/GetSeason` once per page load (a cached GET), and
`fakeSeasonBackend.ts` answers Season 0 in fake mode. A 404 reads as no season.

- `domain/seasonClock.ts` — `seasonClock`, the time left to today's cutoff to the
  second (`13h 05m 12s`, `52m 10s`), or to the season's end once the finale runs,
  and whether it runs, nothing once over. **A day ends at the time of day the
  finale starts**, the backend's `rounds.Current`: worked out here from the
  season, so the chip needs no race. `finaleClock` and `finaleWindow` are the
  finale's countdown and its day and hours in the player's own time zone.
  `countdownsOf` is the chip's faces, made from the two clocks: today's cutoff
  and the finale's start. On the last day they are the same time, so there is
  one face. During the finale there is one, to the season's end. After it, none.
- `domain/champion.ts` — `championOf`, the countries ranked 1st once a finale
  closed, `reigningChampionOf`, that champion while no newer season is current,
  and `winsLine`, "Germany wins Season 0", for the reveal, the chip and the chat.
- `app/season/` — `useSeason`, which drops the season at its end (a page open
  across it goes back to no season), `SeasonChip`, `ChampionChip`, `SeasonDetails`,
  `SeasonFacts`, the rows both of them open on, `useRotation`, which turns the
  chip's faces, and `RoundReveal` with `useRoundReveal` (see [The cutoff,
  revealed](cutoff.md)).

**The season is a chip in the status zone, and it turns between two
countdowns**: today's cutoff, because the day is what scores, and the start of
the Final Battle. Each face shows for 6 seconds. A face is a label and its time
together, so a look never shows one countdown's time under the other's label.
The next face slides up into place; under `prefers-reduced-motion` it changes
with no movement. The mouse over the chip, or focus on it, stops the turns. With
one face the chip does not turn.

On a desktop it sits at the top centre: "Today ends in 13h 05m 12s", then "Final
Battle in 23d 04h 12m 05s", and a press opens a dropdown (Escape closes it). On a
phone it is the right end of the status bar, the two largest units alone
("13h 05m"), under a short tag ("Today", "Final") when there are two faces. A
press opens the same details as a sheet, with both countdowns at once. The chip's
name holds both countdowns in full, so a screen reader does not wait for a turn.
The faces are hidden from it, and they are not a live region. During the finale
it glows and says "Final Battle ends in"; it still opens.

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
purpose: four rows (`SeasonFacts`), the Final Battle with its day and hours, the
daily rounds ("Win the day"), the winner's trophy in the Hall of Fame, and the
titles. During the finale the first row is the power-ups instead of the date.
**Say only what is decided**: a battle full of power-ups, points each day for the
countries that held the most ground (triple in the Final Battle), a trophy for the winning country, a title for every
signed-in player and one more for the winning country's. Guests get no title, so
the row says "signed-in".

**The home page counts down to the Final Battle.** `src/home.ts` reads the
season with the same client, and `app/home/finale.ts` fills the pill in the hero
and the `#final-battle` section, both `hidden` until a season is known and again
once it is over. `finaleClock` counts to the battle's start, then to the season's
end, when both go live and glow. The page's text stays in `index.html`; the
module only writes the numbers and toggles `finale-live`. **It imports no CSS**:
a stylesheet shared with `play.html` becomes its own file, linked before
`play-*.css`, and moves the game's cascade. The art's styles are copied into the
page's `<style>`, as `.panel` and `.button` are.

**The desktop chip writes its bottom edge on `:root` as `--status-bottom`**
(`useBottomEdge`), and on a phone the status bar does: the quiz and the bomb
news sit under it. With neither, the property is unset and they sit at the top.
