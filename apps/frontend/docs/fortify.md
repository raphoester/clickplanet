# Fortify

A flag that takes all of a landmass gives each of its tiles a shield; the same flag
never twice in a row (the rule is the server's, see the backend's [Fortify](../../backend/docs/fortify.md)).
A landmass is a row of the borders blob's table, which `BorderField` already reads.

- **The event is `Fortification`** (`landmass_fortified`, through
  `UpdatesListener.listenForFortifications`). `PlanetBackend` flushes the pending tile
  updates first, so the shields go on the owners the server had.
  `TileShields.applyFortification` adds one to each tile of the landmass the flag
  holds, up to `tileShields`: the same rule as the server, so no tile list travels.
- **The wave** (`fortifyWaves.ts`, `domain/fortifyWave.ts`): each tile's shield is shown
  when a front from the closing tile reaches it, `FORTIFY_WAVE_SECONDS` to cross any
  landmass, and the tile shader glows steel as it passes (`fortifies`,
  `fortifyLandmass`, `fortifyReach`; the shaders' constants must match the TS ones).
  The count is right at once; only what is drawn waits. A hidden tab or less motion
  shows the shields at once.
- **The banner** (`FortifyNews`, in Moments) is news only to the flag that fortified, or
  for a landmass of `NEWS_FROM_TILES` (50) or more (`isNews`); so is its sound
  (`fortify`, a switch of its own). **The first one this browser shows brings a card**
  (`FortifyCard`, kept until "Got it", in `clickplanet-fortify-guide`).
- **Close to fortifying** (`domain/fortifyTarget.ts`, `domain/fortifyTargets.ts`): a flag
  holding more than half of a landmass and `CLOSE_TILES` (25) or fewer from all of it,
  and not the flag that fortified it last. The tiles it still misses pulse gold (a
  `pulse` attribute, a ring in the fragment shader), for everyone. The locks come from
  `GetFortresses` at load and after a gap, then from the events; nothing pulses before
  they are read. **The pulse is drawn at `AMBIENT_FRAME_MS` (20 fps), and only while a
  pulsing landmass faces the camera** (`drawsFrame`'s `ambient`).
- **Your own flag getting close** is a Moments line (`FortifyHint`, "Sicily: 23 tiles to
  fortify"), once per approach, never during the load. Moments show one at a time: the
  newest of a bomb, a fortify and this line.
- **Names** (`domain/landmassNames.ts`): `static/landmassNames.json`, written by
  `npm run landmassNames` from Natural Earth at the map's pin (`scripts/map/names/`:
  subunits, islands, island groups, admin-1 and its region, by tiles in common), one
  name per landmass, unique in its country. A country's main landmass is not listed:
  it is named after the country, and it is the capital's when the biggest has a name
  of its own (Java, not New Guinea). **Regenerate it after `npm run map:generate`**;
  a test fails when it names another borders blob.
- **In the chat**: an announcement of kind `fortify` (`{country, ground, landmass,
  tiles}`), for a landmass of 50 tiles or more.
