# A territory, from orbit

**In orbit the pointer asks about a territory, not a tile.** A territory is a
landmass (see [The zoomed-out view](zoomed-out-view.md)). From that far a tile is
a pixel or two: nobody aims at one, and the card on who holds it said nothing
anyone could use. So a rest of the mouse, or a long press of a finger, opens a
card on the territory under it (`TerritoryCard`), and the territory lights up on
the globe. A click or a tap flies down to it.

- **Orbit is where the painted flags have the frame to themselves**: a tile of
  `COARSE_FROM` (5px) or less (`inOrbit` in `pointSize.ts`). It is a zoom, not the
  map view: with "Big country flags" off it starts at the same zoom. Past it the
  pointer asks about tiles again, as in [Who holds a tile](tile-holder.md).
- **A click in orbit flies to the territory and takes no tile**: centred on the
  middle of its extent as the screen sees it, the north up, and as big as fits in
  `FILL` (60%) of the screen's height and width (`territoryShotOf` in
  `domain/flight.ts`). Never further out than where the tiles have the frame to
  themselves (`tilesZoomOf`), so a big territory is flown into rather than fitted, and
  never closer than tiles of `FLOWN_TILE_MOST` (28px) for a small one (`flownZoomsOf`).
  The flight takes `FLIGHT_SECONDS` (0.9s): it turns and zooms together, the zoom by
  the same ratio each step, eased at both ends (`flightAt`, on the clips' `blend`,
  `between` and `smooth`). Under less motion it lands at once. The player taking the
  globe ends it where it is (`Pilot.cancel` on the controls' `start`); until then it
  overrides what the controls did each tick, so their spin and glide die out under it,
  and it draws every frame, as a drag does.
- **The card** says the territory's name with the flag of its ground, its size and
  the tiles no flag holds; the first five flags by tiles held (`rankHoldings`, ties on
  the code), each with its tiles and its share (`sharePercent`: never 0% for a few
  tiles, never 100% one short of all), and how many more flags hold some; the
  player's own flag wears the board's white coin. Its foot is about fortifying: the
  flag that fortified it last (`FortifyTargets.lockOf`, "Never fortified" without
  one) and, when a flag is close to fortifying it, how many tiles that flag still
  misses (`targetOf`, the same rule as the gold pulse, see [Fortify](fortify.md)).
- **The flags are already on the client**: `BorderField` keeps the tiles each flag
  holds in each landmass for the painted flags, and the locks come from
  `GetFortresses`. `Globe.territoryOf` builds it, and asks no server.
- **The players are the server's**: `useTerritoryPlayers` asks `LandmassReader.landmass`
  (`ClickService/GetLandmass`) once the card opens, then `PlayerService/GetPlayer` for the
  first `TOP_PLAYERS` (3) holders and the last fortifier, as the tile card names its
  holder. "Top players" lists them with their flag and tiles, in their name's colour;
  a signed-in one's name opens its card. The fortifier and when are under "Fortified
  by", **only when the fortify's flag is the lock's**: a lock settled with no fortify
  names nobody. A guest is shown by its guest name, a deleted account as such. See the
  backend's [Who holds the tiles](../../backend/docs/tile-holders.md).
- **It follows the territory while it is open**: `useTerritoryCard` reads it again
  every `SAMPLE_MS` (500ms, as the board), and renders only when something on it
  changed (`sameTerritory`).
- **The highlight**: the territory's tiles are lifted toward white (`highlight`, a
  uniform the display shaders compare with each tile's landmass), and its outline
  is drawn again over everything in white, 2.6px (`setHighlight` on the border
  lines; the blob says which two landmasses each run is between, see [The countries'
  outlines](country-outlines.md)). The globe lights the territory under the mouse,
  else the one whose card is open (`Globe.showTerritory`), and nothing out of orbit.
- **In orbit a tile is picked as wide as the painted flag's discs** (`pickPointSize`),
  which cover the ground. At its own size a tile is about a pixel, with a gap to the
  next, and the one pixel the picker reads fell in a gap every few moves: the
  territory went off and on under a moving mouse, and its card closed and opened
  again. Past orbit a tile is picked at its own size, as before.
- **The gestures are the tile card's**: a second `TilePeek`, keyed by landmass
  rather than tile, so a mouse moving inside one territory keeps its rest. Both
  peeks see every pointer event; the orbit decides which one gets a spot.
- **It behaves as the tile card**, through the same `PointerCard`: beside the
  pointer, kept on screen, gone `LEAVE_GRACE_MS` after the pointer left both the
  territory and the card, closed by its badge, Escape or a press outside it.
