# Battles

**The game shows when the schedule plays a battle.** The backend's gamemaster keeps a
schedule of rule-set entries, and each entry has a kind (the backend's
[gamemaster](../../backend/docs/gamemaster.md)). The client never hears a rule set's
name, only the kind of the entry that plays, from `gamemaster.v1.GamemasterService/ListenForEvents`:
`battle` or `plain`, since when, when the running entry ends, the entry's id, the
battle's round and times, the battles to come, and the last debrief (a day, a battle
or a season scoring scored). A battle has a look, and so has Halloween (see [Halloween](halloween.md)).

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
  season that runs (book `season-0`), and today as the gamemaster counts
  days (day K ends at the K-th 21:00 UTC cutoff after the start). The fake pages date
  that season from the clock (`seasonRunningAt`), so a day always runs; the tests and
  `/archive` keep Season 0's real dates (`SEASON_ZERO`).
- **A battle is its entry's id** (`entryId`, empty when no entry runs): the battle's
  key and the start moment's "seen" mark. **Its round is `roundId`**, scoring's id for
  it: the board's battle column reads it, and a debrief is matched to the battle the
  page saw by it. Two battles are never told apart by their times (the owner,
  2026-10-27).
- **`description` is the entry's, the operator's words for the players**, written
  on the backoffice's Rule schedule page (at most 300 characters, the backend's
  rule). Empty when the operator wrote none, or when no entry runs.
- `app/battle/scheduleStore.ts` (`useSchedule`) holds the turn, its battle, the battles to come and the seasons, as
  the stream sends them, and the last debrief's numbers once read from scoring (`closed`). `useScheduleFeed`, which
  `PlayScreen` calls with the turn and standings backends, fills it (see [Stores](stores.md)). The end of a battle
  needs no read of its own: the stream sends the next turn within a second.
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
battle chip is the chip, as below. **The season's finale is the Final Battle**: when the
next battle has `endsSeason`, the countdown says "Final Battle in" (tag "Final"), and
"Next battle in" otherwise. The fact is the gamemaster's, set on the season's own
battle; the client never guesses it from times. A finale has no entry: its
`entryId` is empty and its round, `season-N/finale`, is its key (`battleKeyOf`).

**`finaleOf` is the season's finale while it is to come or plays**, for the season chip's details and the home
page's Final Battle panel (see [The season](season.md)): the turn while it plays, else the
listed battle with `endsSeason` that has not ended. The timeline lists only battles that have not started, so a
finale whose start passed stays, live, until its turn arrives.

## The teaser

**A battle to come is teased full screen, the Final Battle as any other** (`BattleTeaser`, `useBattleTeaser`, rules
in `domain/battleTeaser.ts`), with its rule set's cards: each a background and words over it, which the operator keeps
on the backoffice's Promo page and the backend's promo module fills with the values the set plays (see the backend's
[Promo](../../backend/docs/promo.md)). **No cards, no teaser** (the owner: "If there is no carousel let's just not
tease anything"). The Final Battle's own teaser, once per browser in the two days before it, is gone (2026-11-05).

- **When** (`battleTeaserDue`): the next battle (`nextBattleOf`), **within the 24 hours before it starts**
  (`BATTLE_TEASER_WITHIN_MS`), as a page opens and again **at most every 6 hours** (`BATTLE_TEASER_EVERY_MS`): Close
  keeps the battle's key and the time in `clickplanet-battle-teased`. A player who comes back in the evening hears of
  it again, one who reloads does not. **A first visit plays first** (`guided`, the front guide's): it waits for 30
  clicks on the page (`TEASER_AFTER_CLICKS`), half the start bank, before a full screen talks of a battle.
- **Its key** (`teaserKeyOf`) is the battle's entry id, or `season-N/finale` for a season's finale, which has no
  entry: the id the timeline names it by, and the one `promo.v1.PromoService/GetTeaser` takes.
- **Its cards are asked once due** (`PromoBackend.teaser`, `ConnectPromoBackend`: a cached GET, a minute in
  Cloudflare), through `useTeaserCards` (`app/battle/teaserCardsStore.ts`), which `useTeaserCardsFeed` serves from
  `PlayScreen`: one read per battle per page, shared with the chip's "See the rules" (below). A battle whose set has
  none, one the schedule no longer holds, a server without promo and a read that failed all answer none, and nothing
  is kept: cards written later show on the next page.
- **The screen** is the start moment's look, with the battle mood and the tape along the top: "Battle" ("Final
  Battle" for the season's finale, `battleNameOf`), "Starts in" and its clock to the second ("Ends in" for a battle
  that plays, `teaserClockLine`), when it plays in the player's time with its points' times (`battleWhenLine`: "Today
  · 22:00 – 23:00 · Points ×2"), the operator's words on the entry, then the cards, and "To the globe" once it
  settled (`TITLE_REVEAL.ready`).
- **The cards** (`TeaserCarousel`) are 3:4 portraits, as tall as the screen leaves room for, one shown at a time,
  the words on a band of shade along the top. **On a wide screen** (1024px and wider, in landscape) the screen's words
  and its button go beside the cards, so a card takes the screen's height (600px wide at most) and its words grow with
  it: the image is the screen's hero, not a thumbnail under the head. A row moved by a transform, so the card shown is always the one the
  dots say: a swipe of 40px turns it (pointer events; an upright swipe still scrolls), the arrows beside it on a
  wide screen and the dots under it are the buttons, and **it turns on its own every 5 seconds** (`TEASER_TURN_MS`)
  until the player touches it. Under `prefers-reduced-motion` it never turns on its own, and slides with no
  movement.
- It plays through the announcements queue (below).

## The rules, asked for

**A player sees when the next battle is and its rules in two presses: the chip, then "See the rules"** (the owner,
2026-11-06: "the players should definitely be able to click one or two buttons max to see when the next battle is and
which are the rules exactly"). Before, the cards showed only when the teaser came on its own, and a battle further
than a day away, or one that already played its teaser, had no rules anywhere.

- **The season chip's details open on the next battle** (`NextBattleDetails`, `app/battle/NextBattle.tsx`, see [The
  season](season.md)): "Next battle", when it plays in the player's time with its points' times (`battleWhenLine`),
  and the operator's words. The Final Battle keeps its own details (`FinaleDetails`).
- **"See the rules"** (`BattleRulesButton`) is under them, and under the Final Battle's once it is the next battle. It
  opens the battle's teaser, asked for: `{kind: "teaser", asked: true}` in the announcements queue. **No cards, no
  button**, as no cards is no teaser; the button shows once `useTeaserCards` read them.
- **The battle that plays has its rules too**, in the battle chip's facts (`BattleFacts`): the same button, for the
  turn as a battle to come (`playingOf`), counted down to its end ("Ends in"). Promo answers a battle's cards whether
  it is to come or plays.
- **Asked for, the teaser plays no entrance** (`asked` on `BattleTeaser`, `battle-teaser--asked`): no title sound,
  no slam, no shake and no flash; the screen fades in whole, and "To the globe" and Escape close it at once. It is the
  same screen as the teaser that comes on its own, so a player knows it, and it costs no new modal.
- **Closing it counts as told**: the battle's key and the time go to `clickplanet-battle-teased`, so the teaser does
  not come on its own in the next 6 hours. A battle that started keeps no mark (`dismiss` skips it): it is never
  teased again, and its mark would only hide the next battle's.

## The look

**The battle is a mood** (see [`DESIGN.md`](../DESIGN.md#moods)): `useMood` writes
`data-mood="battle"` on the root while a battle plays, and `tokens.css` redefines
the panels to charcoal and the secondary button to steel under it. The warning tape
(`--tape`) runs along the top of the dock (`ClickBudgetMeter.css`, under the mood)
and of the battle chip. **The globe's halo turns to ember**: `PlayScreen` hands
`useGlobe` a `halo` (`calm` or `war`), the globe's `setHalo` repaints the
atmosphere's colour (`app/viewer/atmosphere.ts`) and asks for a frame. The sky, the
gold and the text do not move: a coin, a header or a reward looks the same in a
battle.

## The chip

**While a battle plays, the status zone's chip is the battle's** (`BattleChip`),
in the season chip's place and shape (its `season-chip` classes, with its own
colour, glow and tape in `Battle.css`): crossed swords, "Battle ends in", and the
clock to the second. The season chip, or the champion's, is back when the battle
ends. `ScheduleChip` picks the chip from the schedule's store: the battle's, else the champion's, else the season's.
On a phone the chip opens `BattleSheet` or `SeasonSheet`, which read the same store.

- On a desktop a hover opens its facts under it, and a press holds them open
  (Escape closes a pressed one; a hover-only one closes when the pointer leaves).
  On a phone it is the right end of the status bar, "Battle" over the two largest
  units, and a press opens the "Battle" sheet (`BattleDetails`): the clock and the
  same facts.
- **The facts are the description, when the operator wrote one, and when the battle
  ends**, then "See the rules" when the battle has cards (see [The rules, asked for](#the-rules-asked-for)). The
  season's countdowns follow (today's end and the season's), so a player
  loses no clock while the chip is the battle's.
  The facts explain no rule on their own: what a battle changes is the operator's
  description, read from the server.
- The chip's name holds the clock in full, and the visible clock is hidden from a
  screen reader, as the season chip's is.

**The chip of the season's finale says "Final Battle"**: its name ("Final Battle ends in …"), its label, the
compact tag ("Final") and the sheet's clock, from the turn's `endsSeason`.

## The start moment, and the end

**The start plays once per battle per browser** (`BattleReveal`, `useBattleReveal`):
the title unlock's slam, "Battle!" ("Final Battle!" for the season's finale,
`battleNameOf`), a disc of crossed swords in the battle's
colour, the description and when it ends, then "To the globe" once it settled
(`TITLE_REVEAL.ready`). Close keeps the battle's key in `clickplanet-battle-seen`,
so a reload does not play it again; a player who arrives mid-battle sees it, since
the battle is on. It plays through the announcements queue (below).

**The start waits for the day the battle ended.** A battle that is a round starts as a day ends (the backend's
[gamemaster](../../backend/docs/gamemaster.md#a-battle-and-the-days)), and the turn reaches the page about a second
before that day's debrief. So `useBattleReveal` holds the start until the last debrief read (`closed`) ended at or after
the battle's start (`dayBriefAwaited`): the day's reveal is queued first, and the start follows it. Until 2026-11-01
the start came first, and the Final Battle's start played before the day it ended. A battle with no round waits for
nothing, and none waits more than `DAY_BRIEF_WAIT_MS` (10 s): scoring may be down, and a battle at a season's start
ends no day.

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
  3. `players` — "Top players": rank, flag, name in its colour, the title it wears now (its badge, by account, see
     [Badges](badges.md)), tiles;
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
and the store `useAnnouncements` (`app/announcements/announcementsStore.ts`, see [Stores](stores.md)) keep the
queue: a title unlocked, a battle's start, a battle's teaser, a day closed (the cutoff's reveal), a battle's
three debrief screens, the season's results, a clip that names the player (see
[In a clip](clip-appearances.md)), the flag's mission done (see [Missions](missions.md)).
`app/announcements/Moments.tsx` runs the hooks that say what is due (`useRoundReveal`, `useBattleReveal`,
`useBattleTeaser`, `useClipNotice`) and announces each; `PlayScreen` announces the title and the flag's mission.
`Moments` shows the head of the queue once the map is loaded, tells the hook it came from when the player closes
it, and the next starts. An
announcement is queued once, by its key. So a round that closes while the player
reads a battle's screens waits its turn, the season's results wait for the debrief
queued before them, and nothing is drawn over anything. The one fact queued later than it arrives is a battle's start,
held for the day it ended (above).

The round's "once per browser" and the battle's keep their own rule (`useRoundReveal`,
`useBattleReveal`): the queue only orders what is due.

## In fake mode

`startBattle(minutes = 120, description, times = 3)`, `startFinale(description, times = 3)` and `endBattle()` on the console
(see [Commands](commands.md)): the mood, the halo, the chip and the
board's battle column follow at once, and the day that runs ends a second later, so its reveal plays, then the moment. `endBattle()` leaves the battle's round
undebriefed, so "The results are coming" stays; `closeBattle()` scores it and the fake
gamemaster debriefs it, so the debrief plays; `closeFinale()` debriefs it as the Final
Battle, then the season, and the season's results follow. `scheduleFinale(inMinutes, minutes)` plans one: under
two days away, the chip wears its look. **A battle planned within a day is teased** (`scheduleBattle(90, 60)`), with
the fake promo's three cards over made-up backgrounds (`FakePromoBackend`); `fakePromo.cards = []` teases nothing,
and `localStorage.removeItem("clickplanet-battle-teased")` teases it again. **A battle planned further away**
(`scheduleBattle(26 * 60, 60)`) is not teased, and the season chip's details show it with "See the rules".
