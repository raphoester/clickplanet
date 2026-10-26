# Shields

A shield is a charge (`Charges.shields`) placed on a tile the player's flag
holds, up to `BonusRules.tileShields` on one tile. **A tile's shields are
part of the tile**, as its owner is, and every screen shows them. The server
decides what a click does; this client predicts its own. The pieces: `Shielder`
in `backends/backend.ts`, `domain/shields.ts` the count per tile and the rule,
`tileField.ts` and the display shaders the drawing.

- **The count comes with the tile.** Each map batch lists its shielded tiles
  (`GetMapResponse.shields`, read into `Ownerships.shields`), and every
  `TileUpdate` says the tile's count after it. `TileShields` keeps one number
  per tile, and `MapTiles` takes a count only when its version is newer than the
  tile's, as it does a flag (see [The counts are the server's](counts.md)).
- **The server zeroes the count on every change of owner**, so an update that
  changes the owner carries 0, and nothing here ties a count to a flag. **An
  update whose country is its previous country changes the shields alone**: a
  strike or a placement. It moves no tile on the board and plays no click
  glint.
- **A bomb strikes the shielded tiles in its blast** (`BombDrop.struck`): each
  loses one shield and keeps its flag, and is not in `cleared`; the server says
  the shields each kept. The strikes wait for the impact with the clear, and a
  tile update that arrives meanwhile wins its tile by its version, as it does
  for the clear.
- **Shield is a switch.** While it is on, a click on a tile the flag holds sends
  `PlaceShield` (session-gated, not throttled) instead of a click, and spends
  no click. A tile already at its most sends nothing and the slot says "Full".
  Any other tile gets an ordinary click.
- **A click on another flag's shielded tile is shielded** (`outcomeOf`): it costs
  a click and takes nothing, so the globe paints no flag and plays the hit; the
  update brings the count. A take is predicted only on a tile with none, so a
  paint never has a count to drop. When a prediction is wrong, the update that
  follows carries the flag the server kept, and replaces the paint as any update
  does.
- **Spread and enclose follow the same rule on the server**; their updates bring
  the counts.
- **It is drawn in the tile shader**: one more per-tile attribute (`shield`)
  and a texture of marks (`shieldMarks.ts`), drawn once on a canvas: the
  inventory slot's shield in steel and ink, then the same shield with each
  count from 1 to `tileShields` on it, in the display face. The shader picks
  the tile's cell, and shows the plain shield until the tile is 16px across
  and the count from 22px. **The count is on the shield, not on a badge in
  its corner**: a badge leaves a digit a third of the tile, unreadable
  until about 40px. The shield is 80% of the tile, so the flag still shows
  around it.
- **A shield fades out with the tiles** (`1 - flagPaint`): under the painted
  flags from orbit there is none. With the painted flags off, the tiles stay
  and so do their shields. Nothing is allocated per tile and nothing
  animates, so a shield costs no frame.
- **A count that drops glints red, one that rises glints steel**
  (`clickGlints.ts`). This player's own hit and placement play at once, and their
  echo is skipped (`ownHits`, `ownPlacements`).
