# The front, for a first visit

**A first visit opens on the player's own country and flies to its fight.** Before this, a new player saw a slice of
the globe that turned (on a phone, often Asia), and a first tap from that far only flew down to a territory. Half of
the players who left in their first two minutes left after fewer than 30 clicks.

- **Who is guided** (`app/frontGuide.ts`): `main.tsx` calls `enterFrontGuide` before the first render, so before
  `useKeepCountry` stores a country (the country store reads it on import, and writes it only from the page's first
  render). A browser with no country stored is on its first visit: it writes
  `clickplanet-front-guide` = `0`, and every page is guided while that count is under `GUIDED_TAKES` (50).
  `useGuidedTakes` counts each accepted click that took a tile. A browser that played before the guide has a country
  and no count: it is never guided. A refused storage guides nothing.
- **The camera starts on the flag's own country** (`showFront` in `createGlobe`): its biggest landmass, framed as a
  click in orbit frames a territory (`territoryShotOf`), so the tiles have the frame from the first image and a tap
  takes one. A Discord ping's `?lat=&lon=&zoom=` wins over it (see [Discord pings](discord-alerts.md)).
- **Once every tile is read, it flies to the open front tile nearest the middle of that country**, at tiles of 28px
  (`flownZoomsOf`'s `most`). Open is a tile one click takes (`domain/openTiles.ts`, `openTileNear`): not the flag's,
  with no shield (`outcomeOf` in `domain/shields.ts`, read from `TileShields`). A front of shields is a wall where no
  one click changes anything: on 2026-11-12 a CrazyGames reviewer landed on Indonesia's, every tile in view a grey
  "1", and left after 2 clicks. When every front tile is shielded, it flies to the front nearest the middle, as before.
  **A flag that holds no tile** flies to the open tile nearest the middle of its country, in it or near it, and its
  first tile there makes a front; while the reach plays, a click off the land is stopped, so it stays on its country.
  Not if the player took the globe first (`Pilot.cancel`).
- **A hand taps a tile there until the first take** (`app/viewer/tapPointer.ts`). Once the flight is over, a glove
  presses on the tile it flew to, every 1.6s, until the server accepts a click that took a tile: on this page at
  once, and on the next pages by the stored count (`firstTakeAhead`, `useGlobe`'s `showTap`). It follows the tile as
  the camera moves, and hides while the tile is off the screen or behind the globe, in orbit, while a bomb is armed and
  on a frozen map. When the tile stops being open (taken, shielded), it moves to the open tile nearest the middle of
  the screen, looked for once a second. No text: the owner allows no label that explains a rule, and CrazyGames asks
  for pictures over text.
  - **A DOM element, not the GPU layer**, placed each tick from the tile's projected point, as the bonus box's pointer
    is: the loop draws a frame only when something changed (see [Drawing only when something changed](render-on-demand.md)),
    and a press drawn on the GPU would draw one every frame for a minute. As CSS, the press runs on the compositor,
    `prefers-reduced-motion` stills it, and its colors are tokens. It sits under the panels and the chat (z-index 100
    under 128) and takes no pointer, so a press goes through it to the tile.
  - **Its colors**: a `--cream` glove with an `--ink` outline and a hard ink drop it loses as it presses, as a button
    does (see [`DESIGN.md`](../DESIGN.md)), and three short cream strokes above the fingertip on the press. On the
    globe red is a mission, white the front's ring, gold a fortify: the glove is a shape, never a ring, so it names
    none of them.
- **The front is lit** (`domain/front.ts`, `Front`): every tile the flag does not hold, enemy or free, that touches one
  it holds. While the reach plays, every page lights it, and the far end of each bridge from the flag's land too (see [The
reach](reach.md)). A white ring beats on each, as the gold one does on a tile a flag misses to fortify (see
  [Fortify](fortify.md)). It follows every change of owner (`touch`, in `applyChanges`), a ghost's tiles included, and
  the flag played (`follow`, in `setCountry`). It stays lit for the whole page, even once the count is reached.
- **The map fights back** while fewer than 7 play: fake raids on the front, painted on this page only. See
  [Ghosts](ghosts.md).
- **Touching is a distance** (`domain/neighbours.ts`, `Neighbours`): the six around a tile are at most 0.0044 apart on
  the globe of radius 1, the next ring at least 0.0048, so two tiles touch within 0.0046. The tiles are sorted by a
  grid cell of twice that, and a tile's neighbours are in the two cells around it on each axis.
  `neighbours.test.ts` checks it on the shipped map: no tile has more than six.
- **The ring shares the fortify pulse's slot**: bit 1 is the gold ring, bit 2 the white one, bit 4 a mission's still
  red ring, bit 8 a capital's crown (`TileField.setPulses`, `setFront`, `setMissionMarks`,
  `setCapitalMarks`, see [Missions](missions.md) and [The capitals](capitals.md)); the shaders read each bit back with
  `floor` and `mod`, and only bits 1 and 2 beat. The ring beats at
  `AMBIENT_FRAME_MS` while a lit tile faces the camera (`pulsing`).
- **Trying it in fake mode**: every tile starts French there, so a guided Bulgaria holds nothing and flies to a French
  tile in Bulgaria. Clear the page's storage, reload, and `fakeBackend.takeLandmass(39, "bg", 120)` in the console
  lights the front around most of Romania. To see the flight skip shields, shield Bulgaria too
  (`for (const t of fakeBackend.fortresses.tilesOf(41)) fakeBackend.botShield(t, "fr")`), then switch HD graphics in
  Settings: the globe is made again and flies to the open front in Dobruja, not to the shielded Danube.
