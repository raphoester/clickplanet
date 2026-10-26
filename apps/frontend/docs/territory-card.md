# A territory, from orbit

**In orbit the pointer asks about a territory, not a tile.** A territory is a
landmass (see [The zoomed-out view](zoomed-out-view.md)). From that far a tile is
a pixel or two: nobody aims at one, and the card on who holds it said nothing
anyone could use. So a rest of the mouse, or a long press of a finger, opens a
card on the territory under it (`TerritoryCard`), and the territory lights up on
the globe.

- **Orbit is where the painted flags have the frame to themselves**: a tile of
  `COARSE_FROM` (5px) or less (`inOrbit` in `pointSize.ts`). It is a zoom, not the
  map view: with "Big country flags" off it starts at the same zoom. Past it the
  pointer asks about tiles again, as in [Who holds a tile](tile-holder.md). A click
  in orbit still takes the tile under it.
- **The card** says the territory's name with the flag of its ground, its size and
  the tiles no flag holds; the first five flags by tiles held (`rankHoldings`, ties on
  the code), each with its tiles and its share (`sharePercent`: never 0% for a few
  tiles, never 100% one short of all), and how many more flags hold some; the
  player's own flag wears the board's white coin. Its foot is about fortifying: the
  flag that fortified it last (`FortifyTargets.lockOf`, "Never fortified" without
  one) and, when a flag is close to fortifying it, how many tiles that flag still
  misses (`targetOf`, the same rule as the gold pulse, see [Fortify](fortify.md)).
- **Everything on it is already on the client**: `BorderField` keeps the tiles each
  flag holds in each landmass for the painted flags, and the locks come from
  `GetFortresses`. `Globe.territoryOf` builds it, and asks no server.
- **It follows the territory while it is open**: `useTerritoryCard` reads it again
  every `SAMPLE_MS` (500ms, as the board), and renders only when something on it
  changed (`sameTerritory`).
- **The highlight**: the territory's tiles are lifted toward white (`highlight`, a
  uniform the display shaders compare with each tile's landmass), and its outline
  is drawn again over everything in white, 2.6px (`setHighlight` on the border
  lines; the blob says which two landmasses each run is between, see [The countries'
  outlines](country-outlines.md)). The globe lights the territory under the mouse,
  else the one whose card is open (`Globe.showTerritory`), and nothing out of orbit.
- **The gestures are the tile card's**: a second `TilePeek`, keyed by landmass
  rather than tile, so a mouse moving inside one territory keeps its rest. Both
  peeks see every pointer event; the orbit decides which one gets a spot.
- **It behaves as the tile card**, through the same `PointerCard`: beside the
  pointer, kept on screen, gone `LEAVE_GRACE_MS` after the pointer left both the
  territory and the card, closed by its badge, Escape or a press outside it.
