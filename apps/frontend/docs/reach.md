# The reach

**While the rules in force hold `reach`, a take lands only next to the flag's land or at the far end of a bridge from
it** (see the backend's [The reach](../../backend/docs/reach.md)). Capital Siege plays it on its fresh map; any set may
play it on the world's map too. The page draws the bridges, rings what the player can take and stops a click elsewhere.

- **Where the page learns it.** `GetChargeRules.reach`, read into `BonusRules.reach` and read again on `rules_changed`.
  `globe.ts` follows it in `onRules` (`followReach`).
- **What it reads then.** `static/bridges-<hash>.json`, once, the first time the reach plays (`bridges.ts`; URL in
  `bridgesAsset.ts`). It is a copy of [`/map`](../../../map/README.md#capitals-and-bridges): 582 bridges, each between
  two coast tiles. A file written for a map of another size is refused.
- **The bridges, drawn** (`bridgeArcs.ts`): each a dotted arc from one end to the other, one dot about every 38 km, a
  round dot of 3.5 CSS pixels at every zoom, pale blue. An arc rises over the sea as far as it is long, up to 0.12 of
  the globe's radius, so a crossing of an ocean reads as a voyage and one of a strait as a hop. All 582 are drawn while
  the reach plays, the far side hidden by the globe, and none once it stops.
- **What the player can take** (`domain/front.ts`, a `Front` on an adjacency `acrossBridges`, `domain/bridges.ts`):
  every tile it does not hold that touches its land, the far end of a bridge counted as a tile it touches. It is the
  white ring of a first visit's front (see [The front](front-guide.md)), on every page while the reach plays. A guided
  page goes back to its plain front once the reach stops.
- **A click elsewhere is stopped** (`globe.ts`, right after the frozen map). Nothing is sent, the refused sound plays,
  the meter shakes, and a bubble over it says "Take a tile next to your land" (`refusedClicksStore.refuseOutOfReach`,
  `ClickBudgetMeter`). The server may refuse a click the page let through (`ReachRefusal`, read as `OutOfReachError`).
  That click is rolled back and told the same way. While the knock-out plays, it also opens the out window again (see
  [The capitals](capitals.md)).
- **The fake backend refuses no click out of reach**: `playReach()` in its console plays the page's side alone, the
  bridges drawn and the front rung (see [Commands](commands.md)). Check the refusal on the local stack.
