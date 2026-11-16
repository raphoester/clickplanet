# Rally

A rally (`BonusReward` kind `rally`, `rallies` a box gave) calls the player's flag to one place. The player taps Rally,
then any point of the globe: a ring appears there for every player, and for a few minutes a click of that flag inside
the ring takes half a click from the bank. The server's rules are in the backend's [The rally](../../backend/docs/rally.md).
No box draws one yet; the operator's grant, and the fake's `giveRally()`, give some.

**It is aimed or put away, like the mine.** The inventory's Rally slot (`1/3`, against `GetChargeRules.rallies`) aims
it (`Globe.setRallyAimed`), the slot or Escape puts it away, and aiming it puts the bomb, the virus and the mines away
and turns every switch off (and the reverse). While aimed, a tap on the globe calls the rally on the point under the
finger (`Rallier.callRally(point)`, `ClickService/CallRally`) instead of clicking, and puts it away: one rally of a
player is on at a time. **A point, not a tile**: the area counts and its centre tile gets nothing more, so a tap on the
sea near a coast is a rally too, and the reach does not hold it. From orbit, a tap flies down first, as any tap. The
backend takes one rally off the hand at once and gives it back unless the server says it holds none; a `RallyRefusal`
(a rally of the caller still on) is `RallyRefusedError`, said over the slot ("Your rally is still on", the notice
`rallyOn`).

**Every rally on is known to every page.** The planet backend reads `GetRallies` as each stream opens (public, no
token) and hands the list to the globe (`BonusHandlers.onRallies`); a rally called later comes as `rally_called`
(`onRallyCalled`), and the call's own answer tells the caller's page at once. Each carries its centre (`point`) and the
time it has left, which the page turns into an end on its own clock (`performance.now()`), so a phone's wrong date
changes nothing. `Rallies` (`domain/rallies.ts`) keeps them by id.

**Each rally is marked** (`rallyMarks.ts`), at most 24, the page's own first, each gone at its end:

- **A halo, never a line** (`shaders/rallyHalo`): a soft `--rally` teal light along the ring's edge, inside and out, and
  a faint wash rising toward it, so every tile under it stays readable. Another flag's is fainter. One color for one
  concept: the flag says whose.
- **The halo's edge is a clock**: lit from the north round clockwise for the share of its time left, the rest dim, with
  a brighter head where it runs down. It moves in 360 steps (`clockStepOf`), a frame drawn at each, about every half
  second of a 3-minute rally; the whole is the rules' `rally_lasts_ms`.
- **A flag planted at its centre** (a DOM pin over the canvas): a pole, the rally's flag as its cloth (cut from the
  tiles' atlas, `clothStyleOf`), waving (still with reduced motion), and **the caller's name** above it in its color,
  from the nameplates and masked as in the chat (`RallyCallers`, which `useGlobe` builds). The page's own flag's cloth
  and name wear a teal edge. A DOM pin, not a mesh: seen from above, a pole standing on the globe would be a dot, and a
  waving cloth on the GPU would keep the loop drawing.
- **A flag behind the globe peeks over its edge**: the pin stands up the screen, so where its foot is past the horizon,
  what rises past the globe's disc shows and the rest is cut off (a `clip-path` that cuts the disc out, `peekOf`). It
  hides once all of the pole and cloth are behind, and off the screen.
- **A hover or a tap on the flag opens its card** (`.rally-pin-tip`, 4 s after a tap): who called it, for which flag,
  what a click inside costs ("Inside the ring, your clicks cost 0.5 instead of 1"), and the time left, kept current. The
  flag takes the pointer, so a tap on it is no click on the globe.
- **An arrow points at the newest ring of the page's flag only when its flag is not seen** (the box's pointer,
  `variant "rally"`, with a flag mark): off the screen, at the screen's edge as the box's; all behind the globe, just
  past the globe's edge on its side (`placeOnRim`), held inside the screen when that edge is off it. From orbit it sits
  by the globe, not at the screen's corner.

**What a click costs, as the page counts it** (`clickCostOf`): half a click, `GetChargeRules.rally_cost`, on a tile
inside a ring of the flag the page plays for, a whole one elsewhere; rings that overlap cost no less. The globe checks
the bank against that cost (`canClick(budget, at, cost)`) and hands it to `clickTile`, which counts the click in flight
at it. The server decides: its answer (`ClickBudget.spent`) anchors the bank as every answer does.

**A click a rally made cheaper says so on the globe**: its price, "−0.5", rises off its tile in teal and fades
(`rallyPops.ts`, at most 8 at once; it only fades with reduced motion). The price, never a factor: the click costs half,
it is not doubled.

**The bank shows tenths of a click** (`59.5 left`), redrawn as it refills (a tenth of a second apart with reduced
motion), so half a click shows. **"0.5 per click"** (the chip `click-budget-rally`, "Rally: inside the ring, a click
costs 0.5") shows while the last click answered took less than a whole one and a ring of its flag is still on
(`ClickBudget.rally`, `rallied`).

**The chat's line**: `rally`, "le professeur calls a rally in Tasmania", the landmass named as a fortify's, in the
flag's room (the world shows it too); "calls a rally" alone when the server names no landmass, as for a ring on open
sea. See [Live chat](chat.md).

**Known gap**: the arrow at the screen's edge can sit under the left menu, as the box's. Pins of many rallies close
together may overlap. A pin whose foot is just past the bottom of the globe's disc hides at once, as a pole standing up
the screen there would.

**In the fake**, `giveRally()` gives one, a call keeps it 3 minutes at 18 tile spacings, a click inside costs half, and
`enemyRally()` has another flag call one on a tile of the page's flag (called by a fake chatter, so its pin is named).
