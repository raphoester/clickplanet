# Rally

A rally (`BonusReward` kind `rally`, `rallies` a box gave) calls the player's flag to one place. The player taps Rally,
then any tile: a ring appears there for every player, and for a few minutes a click of that flag inside the ring takes
half a click from the bank. The server's rules are in the backend's [The rally](../../backend/docs/rally.md). No box
draws one yet; the operator's grant, and the fake's `giveRally()`, give some.

**It is aimed or put away, like the mine.** The inventory's Rally slot (`1/3`, against `GetChargeRules.rallies`) aims
it (`Globe.setRallyAimed`), the slot or Escape puts it away, and aiming it puts the bomb, the virus and the mines away
and turns every switch off (and the reverse). While aimed, a tap on any tile calls the rally there (`Rallier.callRally`,
`ClickService/CallRally`) instead of clicking, and puts it away: one rally of a player is on at a time. The backend
takes one rally off the hand at once and gives it back unless the server says it holds none; a `RallyRefusal` (a rally
of the caller still on) is `RallyRefusedError`, said over the slot ("Your rally is still on", the notice `rallyOn`).

**Every rally on is known to every page.** The planet backend reads `GetRallies` as each stream opens (public, no
token) and hands the list to the globe (`BonusHandlers.onRallies`); a rally called later comes as `rally_called`
(`onRallyCalled`), and the call's own answer tells the caller's page at once. Each carries the time it has left, which
the page turns into an end on its own clock (`performance.now()`), so a phone's wrong date changes nothing. `Rallies`
(`domain/rallies.ts`) keeps them by id.

**The rings** (`rallyRings.ts`, the bomb's ring shader with a crisp edge): the page's flag's in `--rally` teal, the
other flags' pale, at most 24, the page's own first. A ring hides at its end. **An arrow points at the newest ring of
the page's flag** when it is off the screen or behind the globe (the box's pointer, `variant "rally"`, with a flag mark).

**What a click costs, as the page counts it** (`clickCostOf`): half a click, `GetChargeRules.rally_cost`, on a tile
inside a ring of the flag the page plays for, a whole one elsewhere; rings that overlap cost no less. The globe checks
the bank against that cost (`canClick(budget, at, cost)`) and hands it to `clickTile`, which counts the click in flight
at it. The server decides: its answer (`ClickBudget.spent`) anchors the bank as every answer does.

**The bank shows tenths of a click** (`59.5 left`), redrawn as it refills (a tenth of a second apart with reduced
motion), so half a click shows. **"2×"** (the chip `click-budget-rally`, "Rally: inside the ring, a click costs half")
shows while the last click answered took less than a whole one and a ring of its flag is still on
(`ClickBudget.rally`, `rallied`).

**The chat's line**: `rally`, "le professeur calls a rally in Tasmania", the landmass named as a fortify's, in the
flag's room (the world shows it too); "calls a rally" alone when the server names no landmass. See [Live chat](chat.md).

**Known gap**: the arrow sits on the screen's edge, as the box's, and the left menu can cover it.

**In the fake**, `giveRally()` gives one, a call keeps it 3 minutes at 18 tile spacings, a click inside costs half, and
`enemyRally()` has another flag call one on a tile of the page's flag, to see the pale ring.
