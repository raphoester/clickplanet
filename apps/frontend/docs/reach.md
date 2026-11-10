# The reach

**While the rules in force hold `reach`, a take lands only next to the flag's land or across its waters** (see the
backend's [The reach](../../backend/docs/reach.md)). Capital Siege plays it on its fresh map; any set may play it on
the world's map too. The page rings what the player can take and stops a click elsewhere.

- **Where the page learns it.** `GetBonusRules.reach`, read into `BonusRules.reach` and read again on `rules_changed`.
  `globe.ts` follows it in `onRules` (`followReach`).
- **What it reads then.** `static/waters-<hash>.json`, once, the first time the reach plays (`waters.ts`; URL in
  `watersAsset.ts`). It is a copy of [`/map`](../../../map/README.md). A file written for a map of another size is
  refused.
- **What the player can take** (`domain/front.ts`, a `Front` with `Waters`):
  - every tile it does not hold that touches its land;
  - every coast tile of a water where it holds a coast. A tile between two waters is a coast of both.

  The front counts the flag's coasts on each water. It lights a water's whole coast when the flag takes its first
  coast there, and puts it out when the flag loses its last one. The ocean has 15,294 coast tiles, so a flag on the
  sea lights every ocean coast at once. It is the white ring of a first visit's front (see [The front](front-guide.md)),
  on every page while the reach plays. A guided page goes back to its plain front once the reach stops.
- **A click elsewhere is stopped** (`globe.ts`, right after the frozen map). Nothing is sent, the refused sound plays,
  the meter shakes, and a bubble over it says "Take a tile next to your land" (`refusedClicksStore.refuseOutOfReach`,
  `ClickBudgetMeter`). The server may refuse a click the page let through (`ReachRefusal`, read as `OutOfReachError`).
  That click is rolled back and told the same way. While the knock-out plays, it also opens the out window again (see
  [The capitals](capitals.md)).
- **The fake backend plays no reach.** Check it on the local stack.
