# Fortify

A flag that takes all of a province gives each of its tiles a shield; the same flag
never twice in a row (the rule is the server's, see the backend's [Fortify](../../backend/docs/fortify.md)).
A province is a row of the board's provinces file, inside one landmass (see [Provinces](../../../docs/provinces.md)).
Fortify reads provinces whatever divides the map on screen, Countries or Provinces. Until 2026-11-18 it played on
landmasses.

- **The provinces on the page**: `loadProvinces` (`borderField.ts`) reads the board's `provinces-<hash>.bin` beside its
  borders and refuses a file cut on other borders; a board with no provinces file (`earth-1`) has its landmasses as
  provinces (`landmassProvinces`). The globe keeps each province's tiles (`provinceTiles`) and names one with
  `translator().province` (see [Translation](i18n.md)).
- **The event is `Fortification`** (`province_fortified`, through
  `UpdatesListener.listenForFortifications`). `PlanetBackend` flushes the pending tile
  updates first, so the shields go on the owners the server had.
  `MapTiles.applyFortification` adds one to each tile of the province the flag
  holds, up to `tileShields`: the same rule as the server, so no tile list travels.
  It is the one value that comes as "one more", so it is taken at once only on a
  tile read at the stream's opening or after, and waits for the next read of any
  other (see [The counts are the server's](counts.md)).
- **The wave** (`fortifyWaves.ts`, `domain/fortifyWave.ts`): each tile's shield is shown
  when a front from the closing tile reaches it, `FORTIFY_WAVE_SECONDS` to cross the
  province, and the tile shader glows steel as it passes (`fortifies`,
  `fortifyLandmass`, `fortifyReach`; the shaders' constants must match the TS ones). The glow is matched by the
  province's landmass, within the province's reach. The count is right at once; only what is drawn waits. A hidden
  tab or less motion shows the shields at once.
- **The banner** (`FortifyNews`, on the news line, see [Stores](stores.md#the-news-line)) is news only to the flag that fortified, or
  for a province of 50 tiles or more on `earth-1`, as much ground on another board (`newsFromTiles`, `isNews`: see
  [Sized by the board](board-scale.md)); so is its sound
  (`fortify`, a switch of its own). It names the province. **The first one this browser shows brings a card**
  (`FortifyCard`, which `News` shows once the wave has run, kept until "Got it", in `clickplanet-fortify-guide`):
  "France owns all of Bavaria", and "A flag can't fortify the same province twice in a row".
- **Close to fortifying** (`domain/fortifyTarget.ts`, `domain/fortifyTargets.ts`): a flag
  holding more than half of a province and `CLOSE_TILES` (25) or fewer from all of it,
  and not the flag that fortified it last. The target comes from the server's counts per province
  (`Holdings.leaderOfProvince`, `recount` when `holdings_changed` moves a province's row); the tiles it still misses
  pulse gold (bit 1 of a `pulse` attribute, a ring in the fragment shader; bit 2 is the first visit's front, see
  [The front, for a first visit](front-guide.md)), for everyone, but only a tile whose owner
  is read: a batch that lands `touch`es its tiles. The locks come from
  `GetFortresses`, by province, at load and after a gap, then from the events; nothing pulses before
  they are read. **The pulse is drawn at `AMBIENT_FRAME_MS` (20 fps), and only while a
  pulsing province faces the camera** (`drawsFrame`'s `ambient`, each province's frame from its file). It stops at its
  faintest a minute after the last input (see [Drawing only when something changed](render-on-demand.md)).
- **The territory card's foot** is the province under the pointer: its lock, its target and its last fortifier (see
  [A territory, from orbit](territory-card.md)).
- **Names**: each board's own, `static/boards/<board>/provinceNames-<hash>.json`, written by `npm run provinces`, and
  their translations (see [Provinces](../../../docs/provinces.md#names)). A province absent from them is a country's
  whole main landmass, and reads as it: the country. The landmass names (`landmassNames-<hash>.json`, `npm run
  landmassNames -- <board>`) still name each territory in Countries, the bomb and the rally.
- **In the chat**: an announcement of kind `fortify` (`{country, ground, province, landmass, tiles}`), for a province of
  50 tiles or more: "France fortified Bavaria, Germany", or "France fortified Belgium" for a province that reads as its
  country. A line kept from before 2026-11-18 names a landmass alone: "France fortified Corsica".
