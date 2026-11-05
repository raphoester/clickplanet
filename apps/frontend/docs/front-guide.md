# The front, for a first visit

**A first visit opens on the player's own country and flies to its fight.** Before this, a new player saw a slice of
the globe that turned (on a phone, often Asia), and a first tap from that far only flew down to a territory. Half of
the players who left in their first two minutes left after fewer than 30 clicks.

- **Who is guided** (`app/viewer/frontGuide.ts`): `main.tsx` calls `enterFrontGuide` before the first render, so before
  `useCountryStorage` stores a country. A browser with no country stored is on its first visit: it writes
  `clickplanet-front-guide` = `0`, and every page is guided while that count is under `GUIDED_TAKES` (50).
  `useGuidedTakes` counts each accepted click that took a tile. A browser that played before the guide has a country
  and no count: it is never guided. A refused storage guides nothing.
- **The camera starts on the flag's own country** (`showFront` in `createGlobe`): its biggest landmass, framed as a
  click in orbit frames a territory (`territoryShotOf`), so the tiles have the frame from the first image and a tap
  takes one. A Discord ping's `?lat=&lon=&zoom=` wins over it (see [Discord pings](discord-alerts.md)).
- **Once every tile is read, it flies to the front** nearest the middle of that country, at tiles of 28px
  (`flownZoomsOf`'s `most`). Not if the player took the globe first (`Pilot.cancel`), and not when the flag holds no
  tile: it stays on its country, and its first tile there makes a front.
- **The front is lit** (`domain/front.ts`, `Front`): every tile the flag does not hold, enemy or free, that touches one
  it holds. A white ring beats on each, as the gold one does on a tile a flag misses to fortify (see
  [Fortify](fortify.md)). It follows every change of owner (`touch`, in `applyChanges`), a ghost's tiles included, and
  the flag played (`follow`, in `setCountry`). It stays lit for the whole page, even once the count is reached.
- **The map fights back** while fewer than 7 play: fake raids on the front, painted on this page only. See
  [Ghosts](ghosts.md).
- **Touching is a distance** (`domain/neighbours.ts`, `Neighbours`): the six around a tile are at most 0.0044 apart on
  the globe of radius 1, the next ring at least 0.0048, so two tiles touch within 0.0046. The tiles are sorted by a
  grid cell of twice that, and a tile's neighbours are in the two cells around it on each axis.
  `neighbours.test.ts` checks it on the shipped map: no tile has more than six.
- **The ring shares the fortify pulse's slot**: bit 1 is the gold ring, bit 2 the white one, bit 4 a mission's still
  red ring (`TileField.setPulses`, `setFront`, `setMissionMarks`, see [Missions](missions.md)); the shaders read them
  back with `mod` and a threshold, gold first, and only bits 1 and 2 beat. The ring beats at
  `AMBIENT_FRAME_MS` while a lit tile faces the camera (`pulsing`).
- **Trying it in fake mode**: every tile starts French there, so a guided Bulgaria holds nothing. Clear the page's
  storage, reload, and `fakeBackend.takeLandmass(39, "bg", 120)` in the console lights the front around most of
  Romania.
