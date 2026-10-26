# Charges

A bonus box holds a **charge**, which has no clock and is never used on its own:
a refill (the click bank, filled when the player presses it), a bomb (one drop), a
stack of enclosures (one shape each, `maxTiles` at most), a pool of spread
clicks and a pool of shields. The server keeps them per account, in postgres:
a refill and a bomb at most, up to 3 enclosures, up to 8 spread clicks and up to
30 shields. A box adds a random 1 to 3 enclosures, 1 to 4 spread clicks or 5
to 20 shields, capped at the size; the reward it announces
is what was kept (`+2 spread clicks`), which is what the server answers in
`ClaimBonusResponse.amount`.

**`PlanetBackend` holds `Charges` (`domain/bonus.ts`) and nothing pushes them.**
It reads `GetCharges` at load, with the token in hand and never a fresh one, and
again when a click goes out under a new token (`followSession`): the charges are
the account's. `ClaimBonusResponse.charges` replaces them on a claim, and
`UseRefillResponse.charges` on a refill, which also brings the full budget, and
`PlaceShieldResponse.charges` on a shield placed. Otherwise
it follows its own calls: an accepted click **sent with spread on** takes a spread
click off, this player's own `tilesEnclosed` takes an enclosure off, and a drop
takes the bomb off at once and gives it back unless the server says it holds
none (a ban refusal gives it back too); a shield is taken off the same way. The click answer says nothing about charges, on purpose (see the backend's
[Charges](../../backend/docs/charges.md)). A charge spent in another tab stays on screen until the next read. It
reaches the globe through `BonusHandlers.onCharges`, and `useGlobe` hands it to the
inventory.

**The sizes are rules**: `GetBonusRules` (a GET no cache keeps) answers the
blast radius, the enclose's `maxTiles`, the spread pool's size, the enclosure
stack's size, the shields' pool size and how many can stand on one tile
(`tileShields`) as `BonusRules`, through `onRules`. It is read at load and again on each `rulesChanged` of the
planet stream, so an open page follows a new set in force (the backend's
[Rules](../../backend/docs/rules.md)).

## Off by default, one at a time

**Nothing is used until the player says so.** Spread, enclose and shield are
switches (`Switches`), off at load and never turned on by the client. Every click carries
them: `TileClicker.clickTile(tile, country, switches)` sends
`ClickRequest.spread` and `enclose`, and the server spends a charge only when its
switch is on. Shield is not sent: it changes what a click on the player's own tile
does (see [Shields](shields.md)). A switch goes off by itself when its pool runs out
(`switchesHeld`), so it never says a click does something it will not.

**One bonus at a time**, the bomb included: `switched` turns the other switches off
when one goes on, aiming the bomb switches them all off, and switching one on puts the
bomb away. `globe.ts` holds the one copy (`setSwitch`, `onSwitchesChange`). The
server refuses a click with both switches on (`INVALID_ARGUMENT`), before it
writes or spends anything.

## The inventory

`components/Inventory.tsx` is the section that shows them, the **right half of
the dock** (the meter takes it as `children`, and shows it even with no budget).
It draws no border or background of its own: the one panel is `.click-budget-dock`.
One slot per kind, always shown, each drawn with its box's
icon (`BonusIcon`) in its box's colours (the `--bonus-*` properties in
`BonusAward.css`, shared with the announcement). An empty slot is dimmed, and a
press on it does nothing but say how to get one. A pool shows its count against
its size (`5/8`, `2/3`), and a word over the icon says what the slot is doing
(`On`, `Aim`, `Full`, or `New` for a kind held and never pressed).

- **Refill** fills the bank (`Refiller.useRefill`). **On a full bank it sends
  nothing** and says "Full": a refill there would be wasted. The
  server refuses it too, `FailedPrecondition`, read as `BankFullError`, and spends
  nothing.
- **Bomb** aims it, or puts it away (`Globe.setArmed`).
- **Spread**, **Enclose** and **Shield** switch (`Globe.setSwitch`),
  `aria-pressed`.

**A slot speaks in a bubble, never a `title`.** Most of the players who came from
TikTok play on a phone, where a `title` never shows, and they asked in the chat
what enclose and shields do. So `Bubble` (`components/Bubble.tsx`, drawn in a
portal on the body like the reaction popup) says one line over a slot: what it
does while a mouse rests on it, how to get one on a press of an empty slot, and
how to use it on a press that switches it on or aims it, for its first
`LEARNING_USES` (3). The line goes after a few seconds, or as soon as the slot's
charge is spent. The same line is in the slot's `aria-describedby`.

**The globe says when a bonus did not do what the player meant**
(`GlobeOptions.onNotice`, a `BonusNotice`), and the slot says it in its bubble:

- **Shield on, a tile of another flag**: the click is an ordinary one, so the
  bubble says "Tile taken. Tap it again to shield it", or that shields go on the
  player's own tiles when a shield stopped the click.
- **Shield on, a full tile**: "Full" on the slot and a line saying so.
- **Enclose on, a click that took a tile and closed no shape**: "Shape is not
  closed or is too big (25 tiles max)". **This is guessed on the client**:
  `Click` answers nothing about bonuses (it once hid a shadow ban; see the
  backend's [Charges](../../backend/docs/charges.md)), and the server cannot tell an open shape from one too big
  either. The globe waits `ENCLOSURE_WAIT_MS` (1.5s) after the click is accepted
  for this player's own `tilesEnclosed` closed at that tile, and says it when none
  came.

**What a player has learned is kept in the browser** (`clickplanet-bonus-guide`,
`domain/bonusGuide.ts`, `useBonusGuide`): how many times each kind was switched
on (up to `LEARNING_USES`) and which kinds it has won. `Viewer` holds the one
copy and hands it to the inventory and the award. **The first box of each kind
stays up** until "Got it" or Escape (`BonusAward`'s `kept`): the passing award
lasts 2.9s and a tap anywhere closes it, which in a clicking game is before the
line under it is read. Later boxes of that kind pass as before.

**It does not fold**: five slots are one row, labelled from 1280px and icons
with their counts below that (the name stays in `aria-label`). A fold on a row
this small hid the one thing that says the next click does more than paint.
The dock glows while something is on or aimed.
