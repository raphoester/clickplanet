# Charges

A bonus box holds a **charge**, which has no clock and is never used on its own:
a refill (the click bank, filled when the player presses it), a bomb (one drop), a
stack of enclosures (one click each, which fills every shape it closes, `maxTiles` at most a shape), a pool of spread
clicks and a pool of shields. The server keeps them per account, in postgres:
a refill and a bomb at most, up to 3 enclosures, up to 8 spread clicks and up to
30 shields. A box adds a random 1 to 3 enclosures, 1 to 4 spread clicks or 5
to 20 shields, capped at the size; the reward it announces
is what was kept (`+2 spread clicks`), which is what the server answers in
`ClaimBonusResponse.amount`. The boxes are the offers' (`bonuses.v1.BonusService`, on a stream of its own: see
[The offers stream](backends.md#the-offers-stream)), the quizzes the quizzes' ([The quizzes
stream](backends.md#the-quizzes-stream)); the charges they grant are planet's.

**`PlanetBackend` holds `Charges` (`domain/bonus.ts`), and only a grant pushes them.**
It reads `GetCharges` at load, with the token in hand and never a fresh one, again
when a click goes out under a new token (`followSession`), and when its stream comes
back after a gap: the charges are the account's. A grant from outside the player's
own clicks (a box caught, a right answer, a mission paid, an operator's) comes on the planet stream of the account,
a guest's too, as `charges_granted`: all it holds now (`grantedOf`), which replaces them with no read, and what it
added (`gainedOf`), which the grant listeners hear (a right answer's reward). `ClaimBonusResponse.charges` replaces them on a claim, and
`UseRefillResponse.charges` on a refill, which also brings the full budget, and
`PlaceShieldResponse.charges` on a shield placed. Otherwise
it follows its own calls: an accepted click **sent with spread on** takes a spread
click off (not while spreads are unlimited), this player's own `tilesEnclosed` takes an enclosure off (not while enclosures are unlimited; the event
also brings the tiles it took, see [Viewer](viewer.md)), and a drop
takes the bomb off at once and gives it back unless the server says it holds
none (a ban refusal gives it back too); a shield is taken off the same way. The click answer says nothing about charges, on purpose (see the backend's
[Charges](../../backend/docs/charges.md)). A charge spent in another tab stays on screen until the next read. It
reaches the globe through `BonusHandlers.onCharges`, and `useGlobe` puts it in the bonus store (`useBonus`, see
[Stores](stores.md)), which the inventory reads.

**The sizes are rules**: `GetChargeRules` (a GET no cache keeps) answers the
blast radius, the enclose's `maxTiles`, the spread pool's size, the enclosure
stack's size, the shields' pool size, how many can stand on one tile
(`tileShields`), whether enclosures are unlimited (`enclosuresUnlimited`) or off (`enclosuresOff`) and whether spreads
are unlimited (`spreadUnlimited`) as `BonusRules`, through `onRules`. It is read at load, again on each `rulesChanged` of the
planet stream, and again when the stream comes back after a gap, which may have hidden one, so an open page follows a
new set in force (the backend's [Rules](../../backend/docs/rules.md)). `BonusRules.signInRefill` is the offers' rule,
from `BonusService/GetGrantRules`, read at load and on the offers stream's own `rulesChanged` and comeback, and told
with planet's.

## Off by default, one at a time

**Nothing is used until the player says so.** Spread, enclose and shield are
switches (`Switches`), off at load and never turned on by the client, but for spread or enclose while it is
unlimited (below). Every click carries
them: `TileClicker.clickTile(tile, country, switches)` sends
`ClickRequest.spread` and `enclose`, and the server spends a charge only when its
switch is on. Shield is not sent: it changes what a click on the player's own tile
does (see [Shields](shields.md)). A switch goes off by itself when its pool runs out
(`switchesHeld`), so it never says a click does something it will not.

**One bonus at a time**, the bomb included: `switched` turns the other switches off
when one goes on, aiming the bomb switches them all off, and switching one on puts the
bomb away. `globe.ts` holds the one copy (`setSwitch`, `onSwitchesChange`); the bonus store holds what it told last,
and `useGlobe`'s `toggleSwitch` and `toggleBomb` ask the globe for the other state. The
server refuses a click with both switches on (`INVALID_ARGUMENT`), before it
writes or spends anything.

### A switch at rest, while spreads or enclosures are unlimited

**A rule set with unlimited spreads or enclosures** (the backend's
[Unlimited spreads](../../backend/docs/spread.md#unlimited-spreads-chargesspreadunlimited) and
[Unlimited enclosures](../../backend/docs/enclose.md#unlimited-enclosures-chargesencloseunlimited), for a battle)
**makes that switch the one at rest**, spread first when both are (`restingSwitch`): on whenever no other switch is on
and the bomb is not aimed (`atRest`, applied by `switchTo` in `globe.ts` to every change). So it is on at load and when
the set starts playing, and comes back when the other switch or shield goes off, runs out, or the bomb is dropped or
put away. **The player can still turn it off**, and it stays off until they turn it on again (`restSwitchedOff`), or
until the set changes again. When the set stops, the switch goes off, so a click never spends a spread click or an
enclosure held by surprise. Turning another switch on, or aiming the bomb, turns it off, as ever. No charge is needed
(`switchesHeld`). No "Shape is not closed" bubble is said while enclosures are unlimited: nearly every click would say
it.

### No enclose while enclosures are off

**A rule set with enclosures off** (`enclosuresOff`, the backend's
[Enclosures off](../../backend/docs/enclose.md#enclosures-off-chargesencloseoff)) **hides the Enclose slot**, held or
not, as bombs off hides the bomb. `switchesHeld` turns enclose off and never keeps it on, so the page sends no click with
enclose on, and no "Shape is not closed" bubble is said; `restingSwitch` never picks enclose. When the set starts
playing, `onRules` puts the switch off at once. The enclosures held are kept, and the slot shows them again when the set
stops. In the fake, `encloseOff()` plays it and `encloseOff(false)` stops it.

## The inventory

`bonus/Inventory.tsx` is the section that shows them, the **right half of
the dock** (the meter takes it as `children`, and shows it even with no budget).
It draws no border or background of its own: the one panel is `.click-budget-dock`.
One slot per kind, always shown (but the bomb while bombs are off, and Enclose while enclosures are off), each drawn
with its box's
icon (`BonusIcon`) in its box's colours (the `--bonus-*` properties in
`BonusAward.css`, shared with the announcement). An empty slot is dimmed, and a
press on it does nothing but say how to get one. A pool shows its count against
its size (`5/8`, `2/3`), and a word over the icon says what the slot is doing
(`On`, `Aim`, `Full`, or `New` for a kind held and never pressed).

- **Refill** fills the bank (`Refiller.useRefill`). **On a full bank it sends
  nothing** and says "Full": a refill there would be wasted. The refill a
  guest's first sign-in earns is spent by the game, through the same call (see
  [Sign-in](sign-in.md#the-refill-a-first-sign-in-earns)). The
  server refuses it too, `FailedPrecondition`, read as `BankFullError`, and spends
  nothing.
- **Bomb** aims it, or puts it away (`Globe.setArmed`).
- **Spread**, **Enclose** and **Shield** switch (`Globe.setSwitch`),
  `aria-pressed`.
- **Enclose, while enclosures are unlimited**, and **Spread, while spreads are unlimited**, count `∞`, are never empty
  and never "New" (the game gave it, not a box), and while on, a dashed outline runs round the slot without end
  (`.inventory-loop`, still under `prefers-reduced-motion`): a loop that keeps closing.

**A slot speaks in a bubble, never a `title`.** Most of the players who came from
TikTok play on a phone, where a `title` never shows, and they asked in the chat
what enclose and shields do. So `Bubble` (`components/Bubble.tsx`, drawn in a
portal on the body like the reaction popup) says one line over a slot: what it
does while a mouse rests on it, how to get one on a press of an empty slot, and
how to use it on a press that switches it on or aims it, for its first
`LEARNING_USES` (3). The line goes after a few seconds, or as soon as the slot's
charge is spent. The same line is in the slot's `aria-describedby`.

**The globe says when a bonus did not do what the player meant**
(`GlobeOptions.onNotice`, a `BonusNotice`, kept in `useBonus`, `bonus/bonusStore.ts`), and the slot says it in
its bubble:

- **Shield on, a tile of another flag**: the click is an ordinary one, so the
  bubble says "Tile taken. Tap it again to shield it", or that shields go on the
  player's own tiles when a shield stopped the click.
- **Shield on, a full tile**: "Full" on the slot and a line saying so.
- **Enclose on, a click that took a tile and closed no shape**: "Shape is not
  closed or is too big (25 tiles max)", never while enclosures are unlimited. **This is guessed on the client**:
  `Click` answers nothing about bonuses (it once hid a shadow ban; see the
  backend's [Charges](../../backend/docs/charges.md)), and the server cannot tell an open shape from one too big
  either. The globe waits `ENCLOSURE_WAIT_MS` (1.5s) after the click is accepted
  for this player's own `tilesEnclosed` closed at that tile, and says it when none
  came.

**What a player has learned is kept in the browser** (`clickplanet-bonus-guide`,
`domain/bonusGuide.ts`, the store `useBonusGuide` in `bonus/bonusGuideStore.ts`): how many
times each kind was switched on (up to `LEARNING_USES`) and which kinds it has won. The inventory
reads it and counts each use; `BonusAward` reads it for the award it shows, which it takes from `useBonus`. **The first box of each kind
stays up** until "Got it" or Escape (`BonusAward`'s `kept`): the passing award
lasts 2.9s and a tap anywhere closes it, which in a clicking game is before the
line under it is read. Later boxes of that kind pass as before.

**It does not fold**: five slots are one row, labelled from 1280px and icons
with their counts below that (the name stays in `aria-label`). A fold on a row
this small hid the one thing that says the next click does more than paint.
The dock glows while something is on or aimed.
