# The reach: a take only next to the flag's land or across a bridge from it

**While the rules in force hold `reach`, a take of a flag lands only next to its land or at the far end of a bridge from
it** (the owner, 2026-11-09; a knob of its own since 2026-11-10; bridges since 2026-11-11). It is off in `default`.
Capital Siege plays it on its fresh map (see [Arena battles](arenas.md)), and any set may play it on the world's map too.

- **The rule** (`clicks.Reach`, gated on `clicks.ReachPart`): a take of a flag lands on a tile it holds, on a tile next
  to one it holds (`Geography.Neighbours`), or on a tile a bridge leads to from one it holds (`clicks.Bridges.Across`,
  from `/map`'s `bridges-<hash>.json`). The board answers through `Storage.HoldsAny`, under one read lock. A flag with
  no land reaches nothing.
- **The bridges** are the map's, both ways: stepping stones that join every piece of land where the sea is narrowest,
  sea lanes placed by hand, and second ways so that no single bridge holds any land to the rest: every island group
  has two ways in at least (see [`/map/README.md`](../../../map/README.md#capitals-and-bridges)). Each end is one
  coast tile: a flag that holds and shields the end of a bridge holds the crossing.
- **Why bridges** (the owner, 2026-11-11): until then a take crossed any body of water where the flag held a coast, and
  the ocean was one body. A flag on any coast reached every coast on Earth, so 130 of the 239 capitals (Dublin, Rome,
  Lisbon, every island) were in reach of every flag from the start, and a signed-in player alone took one in about
  35 s. A distance on the water was refused: a player cannot see kilometres, and Hawaii fell out of any range that
  kept Dublin safe.
- **A click** out of reach is refused by `reach_click`, outside the throttle as `allegiance_click` is, so it spends no
  token: `FailedPrecondition` with a `ReachRefusal` detail (`clicks.ErrOutOfReach`), told to the antibot as
  `CLICK_OUTCOME_OUT_OF_REACH` (an attempt, not inspected).
- **A click that spreads is checked like any click**: its tile must be in reach. From 2026-11-12 to 2026-11-13 such a
  click was in reach when any tile of its disc was; the owner took it back as confusing (see
  [Spread](spread.md#a-spread-is-spent-after-the-click)). With 2 rings, a spread next to the land moves the front 3 tiles
  deep.
- **A spread** takes its whole disc once the tile clicked is the flag's, and only the tiles of its disc in reach when a
  shield kept it (`Reach.Around`), read after the clicked tile is written; the disc is a walk on the land and never
  crosses a bridge. Without the reach, `Around` is the whole disc.
- **An enclosure** needs no check: its pocket is walled by the flag's own tiles alone, see [Enclose](enclose.md).
- **A bomb** takes no tile for the bomber, so the reach does not touch it.
- **The page** learns it from `GetChargeRules.reach`, draws the bridges and rings what the flag can take: see the
  frontend's [The reach](../../frontend/docs/reach.md).

Tests: `clicks`' `reach_test.go` pins a take next to the land, across a bridge both ways, refused to an island no bridge
from the land leads to, every tile in reach without the rule, a spread past the tile taken or from a struck one, and a wide spread's disc;
`bridges_test.go` the far ends and a bridge off the map; `reach_click` refuses before the inner click;
`throttle_test.go` keeps the token of three clicks out of reach over HTTP; `click_handler` answers the `ReachRefusal`;
`spread_click` takes only the neighbours in reach, and with the real reach refuses a spreading click 2 tiles out and
takes the whole disc of one next to the land; `get_charge_rules_handler` tells the page. The shipped bridges are
pinned in `embedded_geodesic_map`'s `bridges_test.go`: every end on a coast, every tile reached over land and bridges,
the Faroes to Iceland, Iceland to Greenland and Hawaii to California.
