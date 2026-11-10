# The reach: a take only next to the flag's land or across its waters

**While the rules in force hold `reach`, a take of a flag lands only next to its land or across its waters** (the
owner, 2026-11-09; a knob of its own since 2026-11-10). It is off in `default`. Capital Siege plays it on its fresh map
(see [Arena battles](arenas.md)), and any set may play it on the world's map too.

- **The rule** (`clicks.Reach`, gated on `clicks.ReachPart`): a take of a flag lands on a tile it holds, on a tile next
  to one it holds (`Geography.Neighbours`), or on a coast tile of a body of water where it holds a coast tile
  (`clicks.Waters`, from `/map`'s `waters-<hash>.json`). The board answers through `Storage.HoldsAny`, under one read
  lock; the ocean's coast is a scan of at most 15,294 tiles, only for a tile that touches none of the flag's land. A
  flag with no land reaches nothing.
- **A click** out of reach is refused by `reach_click`, outside the throttle as `allegiance_click` is, so it spends no
  token: `FailedPrecondition` with a `ReachRefusal` detail (`clicks.ErrOutOfReach`), told to the antibot as
  `CLICK_OUTCOME_OUT_OF_REACH` (an attempt, not inspected).
- **A spread** takes only the neighbours in reach (`Reach.Around`), read after the clicked tile is written. Without the
  reach, `Around` is the map's own table.
- **An enclosure** needs no check: its pocket is walled by the flag's own tiles alone, see [Enclose](enclose.md).
- **A bomb** takes no tile for the bomber, so the reach does not touch it.
- **The page** learns it from `GetBonusRules.reach` and draws the reach itself: see the frontend's
  [The reach](../../frontend/docs/reach.md).

Tests: `clicks`' `reach_test.go` pins a take next to the land, across a water with a coast held, refused across one
without, every tile in reach without the rule, and a spread past the tile taken or from a struck one; `reach_click`
refuses before the inner click; `throttle_test.go` keeps the token of three clicks out of reach over HTTP;
`click_handler` answers the `ReachRefusal`; `spread_click` takes only the neighbours in reach;
`get_bonus_rules_handler` tells the page. The shipped `waters` are pinned in `embedded_geodesic_map`'s `waters_test.go`.
