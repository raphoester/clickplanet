# Provinces: fortify on smaller ground

**Designed 2026-11-18, with the owner. PR 2 is built (2026-11-18): the generator, the file, the names and their
translations, the preview and the audit. PR 3 is built (2026-11-18): fortify plays on provinces, on the backend and the
page. PR 4 is built (2026-11-18): the hover and the big flags by province, the province lines, the taller texture, then
behind a setting. PR 5 is built (2026-11-19): the zoom brings the provinces in, and the setting is gone.** Each PR at
the end turns its part of this doc into "how it works" and updates the docs it makes stale:
[Fortify](../apps/backend/docs/fortify.md), the frontend's `fortify.md`, `zoomed-out-view.md`, `territory-card.md` and
`country-outlines.md`, [Map geometry](map-geometry.md) and [`map/README.md`](../map/README.md).

## Why

- **A landmass is too big to fortify on `earth-2`.** The world plays 1,004,682 tiles since 2026-11-18, and the click
  economy did not change. The lower 48 are ~54,000 tiles and Russia's landmass 112,274. Nobody holds all of that.
- **The small ones were all that was fortified.** On `earth-1` (Oct 2026), 46% of fortifies were 1-tile islands.

## Words

- **Province**: the ground fortify plays on. One Natural Earth admin-1 unit (a US state, a German Land, a French
  département), or several joined, inside one landmass.
- **Landmass**: unchanged. The Countries mode, the bomb and rally lines in the chat, the clips' `landmasses_flipped` and
  the country sizes keep it.
- Not "region": the page says it for a part of the world ("Europe"). Not "territory": the page says it for a landmass
  seen from orbit, the backend for a flag's holdings.

## How a board's provinces are cut

`npm run provinces -- <board>` does it (`scripts/generateProvinces.ts`, `scripts/map/provinces.mjs`), from
`ne_10m_admin_1_states_provinces` at the Natural Earth tag the generator pins (v5.1.2), which `naturalEarth()` caches.

1. **A tile takes the unit under its centre**, only if the unit is in the tile's own country, by the game's own country
   rule (`countryCodeOf`, with its `ABSORBED`: Somaliland and Baikonur, which Natural Earth gives no country, are
   Somalia's and Kazakhstan's; Northern Cyprus is Cyprus's). So Somaliland is a province of Somalia. Where two units
   overlap, the tile's own country's is taken.
2. **A tile never changes country.** A tile under another country's unit, or under none, takes the unit of the nearest
   tile of its own landmass that has one, by steps on the lattice. The two Natural Earth maps are drawn at different
   detail and do not agree everywhere. On `earth-2`:
   - 1,322 tiles under another country's unit only: a disputed line (Aksai Chin's Chinese tiles under India's "Ladakh"
     join Tibet), or a border drawn a little apart (4 Mexican tiles under "Texas" join Chihuahua). The prototype of the
     design counted 2,531, because it read only the first unit under a tile;
   - 13,618 under no unit: Antarctica's ice shelves, which the game counts as ground and admin-1 does not, and coasts and
     islands where the country is drawn coarser than the unit (Nunavut's Arctic coast, the Aleutians, Chile's fjords).
3. **A landmass where no tile has a unit is one province**: 72 tiles in 68 landmasses on `earth-2`, 59 on `siege-1`,
   the 1- and 2-tile islands and the seats.
4. **A piece** is the tiles of one unit that touch on the lattice, inside one landmass. So a province never crosses a
   sea or a country border, and Alaska and Hawaii stay apart from the lower 48. A strait narrower than a tile is no sea:
   Sicily and Calabria are one landmass, and one province on `earth-2`.
5. **A piece under N tiles joins a touching piece of its landmass**, the smallest first. It prefers one in the same
   Natural Earth `region` (France's 13 regions, Italy's 20), then one still under N, then the longest shared edge.
   It never crosses a UK nation (a UK unit's `geonunit`: England, Scotland, Wales, Northern Ireland), so a whole
   nation stays a province even under N (Wales is 143 tiles on `earth-2`). A landmass under N stays one province.
6. **No upper limit.** Sakha is 20,499 tiles on `earth-2`, Texas 4,675, Antarctica 93,872 (with a piece of the Ronne
   Ice Shelf apart, 213 tiles that Natural Earth's nameless unit of Antarctic islets reaches by step 2).

**N is a number per board**, `provinceFloor` in `BOARD_DEFINITIONS` (`scripts/map/board.mjs`) beside its `cols`, not a
formula: a rule by area would copy `earth-2`'s cut onto `siege-1`, and the owner wants `siege-1` to group more.

| board | N | provinces | of N tiles or more | the prototype | note |
| --- | --- | --- | --- | --- | --- |
| `earth-2` | 200 | 2,097 | 1,090 | 2,108, 1,089 | Germany in 10; most US states alone (117 US provinces, with every island) |
| `siege-1` | 20 | 782 | 516 | 787, 515 | a tile is 79 km apart, ~5,500 km²: 20 tiles are about Bulgaria; France in 3 |

The few provinces from the prototype are the overlapping units of step 1. Under N on `earth-2`: 1,006 whole landmasses
and Wales; on `siege-1`, 263 and 3 UK nations.

`earth-1` and the tests' `fixture-1` have none: `earth-1` only draws history, and its fortifies keep their landmass.
**Changing N later** is a regeneration and a deploy: no tile changes id, and the locks are dropped and settled again as at
the first switch.

**`npm run map:provinces -- <board> <N,N,...>`** draws a board's provinces at each N as PNGs of the world, Europe and the
USA (`scripts/previewProvinces.ts`, `node_modules/.cache/provinces/` or `--out <dir>`): one color a province, the country
borders and the coasts in black. It writes nothing to the board. **`npm run map:audit -- <board>`** fails on a tile in
no province, a province off its tile's landmass, a province under N that is not a whole landmass or a whole UK nation,
and a file that differs from a fresh cut.

## Names

Each province's English name, in `static/boards/<board>/provinceNames-<hash>.json`, keyed by the provinces file it names
(`{"provinces": "provinces-<hash>.bin", "names": {"<id>": "<name>"}}`):

- **One unit: its name**, Natural Earth's English (`name_en`, then `name`, the kind of division cut off as the landmass
  names cut it: "Hokkaidō", not "Hokkaidō Prefecture"). A unit whose English name another unit of its country has too
  takes its own local one: Natural Earth gives Madagascar's 22 regions the names of its 6 old provinces, and calls
  Altai Krai "Altai Republic".
- **All the units of one Natural Earth region on the landmass, in this province alone: the region's name**
  ("Hauts-de-France"). Else the name of its largest unit.
- **A whole landmass: the landmass's name**, numbered as the landmasses are ("Texas 2") when another province of its
  country has the name: a barrier island the landmass names call "Texas" (77 on `earth-2`, 4 on `siege-1`). **A
  country's whole main landmass has no line**: it reads as its country, translated by the page as the landmass is
  (121 on `earth-2`, Belgium among them; 160 on `siege-1`).
- **Overrides**, where a better name exists ("Northern Italy", "Central Germany", "Paris Basin", "Upper Peninsula"),
  and for Natural Earth's own bad names ("West Soudan" is West Darfur, "Eastern National Capital Area" Kenya's Eastern
  Province, the nameless Antarctic unit's piece the Ronne Ice Shelf). Claude wrote them, and the owner reviews them:
  `scripts/map/provinceNames/<board>.json`, 149 on `earth-2` and 109 on `siege-1`. An override names a province by the
  exact set of its units, each with its Natural Earth name so the list reads, and a point (`at`, [lon, lat]) when two
  provinces have the same units (Michigan's two peninsulas):

  ```json
  {"name": "Berlin-Brandenburg", "units": {"DEU-1572": "Brandenburg", "DEU-1599": "Berlin"}}
  ```

  **The generator fails** on an override that matches no province, a unit whose name is not the one written, two
  overrides of one province, and an override of two provinces with no point; so a change of N makes the names checked
  again. It fails too on two provinces of one country named alike, and on a province with no name, until an override
  names it.
- **In 8 languages**, `src/i18n/provinces/<board>/<tag>.json`, by the [translate skill](../apps/frontend/docs/i18n.md#translating):
  a name another board's provinces or landmasses already translate; else Natural Earth's own `name_xx`, where every
  place of that English agrees on it and it has no more words than the English ("İsfahan Eyaleti" is not "Isfahan");
  else Haiku. Czech, which Natural Earth lacks, is all Haiku's. 2026-11-18: 1,976 names a language on `earth-2`, 622 on
  `siege-1`.

## The board's file

`provinces-<hash>.bin`, a file of the board in `map/boards/<board>/`, named in the manifest (`files.provinces`; the
page's `BoardFiles`, the backend's `cpboards`): the province of each tile, and each province's landmass, frame and tile
count (see [`map/README.md`](../map/README.md#format)). **Adding it renamed the manifest only**: the board keeps its name
and positions hash, so no new board, no era, no remap (see `boards-and-eras.md`). `earth-2`'s is 2,016 KB, 113 KB with
brotli at 5; `siege-1`'s 73 KB.

- **The backend** embeds it (`make map`), reads it as it builds the board's ground and refuses a file cut on other
  borders (`embedded_board.loadProvinces`, `clicks.Provinces`, `clicks.ProvinceID`), and logs its count. Fortify reads
  it (`Provinces.ProvinceOf`, `TilesOf`).
- **The page** carries it in `static/boards/<board>/` (`npm run map`), with the names beside it (`page.provinceNames` in
  `BOARDS`), and reads it with `domain/provinces.ts`: `createGlobe` fetches it with the board's other files
  (`loadProvinces`, which refuses one cut on other borders) and the names (`provinceNamesOf`), and a board with none has
  its landmasses as provinces. Its province lines are a file of the page's own (`page.provinceLines`, see the page's
  `country-outlines.md`).
- **The backoffice** has none.

## Fortify on provinces (PR 3, built 2026-11-18)

- **Fortify, the lock and "never the same flag twice in a row" play on provinces** instead of landmasses: the
  backend's `inmemory_tile_storage` counts each province's tiles per flag and keeps its lock, kept in
  `planet.provinces (map, board, asset, id, fortified_by)`, keyed by the provinces file (see the backend's
  [Fortify](../apps/backend/docs/fortify.md)). A board with no provinces file (`earth-1`) fortifies nothing.
- **The switch**: migration `20261118171922_province_locks` drops `planet.landmasses`, and the boot settles every
  whole province to its holder with no shield, as it did for a whole landmass with no lock. No shield is taken away,
  and nobody gets a free fortify (`TestTheBootAfterTheLocksMovedToProvincesSettlesEachWholeProvinceWithNoShield`).
- **The wire**: `province_fortified{province_id, country_id, tile_id, version}` in place of `landmass_fortified`,
  `Fortress.province_id`, `GetHoldings` and `holdings_changed` with each province's row (`ProvinceHoldings`), and the
  hover card's last fortify read for the province (`ClickService/GetProvince`; `GetLandmass` answers the holders
  alone). The ledger's `fortify` names the `province` and its `provinces` file; an old event keeps its `landmass`, and
  every reader takes both. A hard switch of the wire, so the backend and the page ship together (see
  [A server restart reloads the page](reload-on-restart.md)).
- **The chat** names the province and its country: "France fortified Bavaria, Germany" (`announcement.fortifiedProvince`),
  "France fortified Belgium" for a province that reads as its country. `planet.v1.Fortified` carries the province and
  the landmass it is inside. The floor stays `fortifyNewsTiles` (200 `earth-2` tiles).
- **The page**: the shields and the wave on the province's tiles the flag owns, the gold pulse and `GetFortresses` by
  province, the "Fortified!" card ("France owns all of Bavaria", "province" in all 9 languages), and the territory
  card's foot on the province under the pointer (see the frontend's [Fortify](../apps/frontend/docs/fortify.md) and
  [A territory, from orbit](../apps/frontend/docs/territory-card.md)).
- **Unchanged**: the fortify missions (they count tiles), 0.2 XP a tile fortified, the bomb and rally lines, the clips'
  `landmasses_flipped`, the country sizes.
- **Balance**: every knob as it is. For a week after the deploy, measure the shielded share of the map (83% on
  2026-11-05) and the fortify news lines a day, then tune `charges.shield.perTile` and the floor.

## The page

- **The zoom chooses the divisions, as it does the tiles: the countries from orbit, their provinces from where a CSS
  pixel is 7 km of ground and the screen's shorter side 2,700 km** (zoom 2.77 on a phone, 4.72 on a laptop), 5% each
  side of the line so a zoom held on it does not flicker, the one framing fading into the other over a quarter of a
  second as the zoom crosses it. No setting: a new player lands on the countries and finds the provinces by zooming in. They change three things only: the territory the pointer lights and names in orbit, the frame each big
  flag stretches over (past the line, the US flag repeats over each state), and the lines seen from orbit. The video,
  the archive and the clips keep the countries at every zoom. Why, the measures and the shots that put the line there:
  [The provinces come in with the zoom](../apps/frontend/docs/zoomed-out-view.md#the-provinces-come-in-with-the-zoom).
  A setting, "Map: Countries or Provinces", came first (2026-11-18, #895): from orbit Provinces was noise, and pushed
  in Countries stretched one band of a big flag over the screen, so the owner had it follow the zoom the next day; the
  `divisions` it saved in `clickplanet-display-settings` is dropped on read. How each works is in the page's docs: the
  hover in `territory-card.md`, the frames in `zoomed-out-view.md`, the lines in `country-outlines.md`, the tile's
  province on the GPU in `tile-layout.md`, the counts in `counts.md`.
- **A big flag's leader in a province is read from the holdings per province**: `ProvinceHoldings` in `GetHoldings`
  and `HoldingsChanged`, beside the landmasses' (`Holdings.leaderOfProvince`). The least painted size,
  `leastPaintedTiles`, is per province as it is per landmass.
- **Fortify reads provinces at every zoom**: the pulse on the tiles a flag still misses, the "Fortified!" card ("France
  owns all of Bavaria"), the hover card's last fortify.
- **The lines.** From orbit, the country lines, and past the provinces' line the province lines in light grey too.
  Zoomed in, both, the province lines darkening as the tiles grow to the country lines' grey, dashed
  (2026-11-18: the owner could not see the light grey pushed in; see [The countries'
  outlines](../apps/frontend/docs/country-outlines.md)). The country lines are as they were.
- **The outlines are straight cell edges, with no B-spline** (the owner, 2026-11-18). The tiles are straight hexagons,
  so a curve round them reads worse, and it costs four pieces a cell edge pushed in and two from orbit, where straight
  costs one. Province lines add 70% to `earth-2`'s outline edges at N 200 (90,568 on 129,431); the straight edges pay
  for them. From orbit they take coarse sets of their own, as the country lines do (`coarseOutlinesOf`, see
  `country-outlines.md`), so there their 70% costs about a fifth as much.
- **The landmass texture holds 2,048 rows** and `earth-2` has 2,097 provinces, so the provinces' texture is taller: a
  row a province, 2,098 on `earth-2`, and `BorderField` refuses more than 4,096 (see the players' GPU limits).

## The PRs, in order

1. **Straight outlines** (built 2026-11-18, #889). The spline goes, and its junction clamps with it. It ships alone and
   helps the lag of 2026-11-18. Culling the outlines by chunk comes only if the GPU timer still finds them heavy.
2. **The generator** (built 2026-11-18): the provinces file of `earth-2` and `siege-1`, the names, the overrides and
   their translations, the preview, the audit, the copy in each app. Nothing reads it yet. This doc landed with it.
3. **The backend and the page together** (built 2026-11-18): fortify by province, the switch of the locks, the wire,
   the ledger, the hover card's last fortify, the chat line.
4. **The page** (built 2026-11-18): the setting, the hover and the big flags by province, the grey lines, the taller
   texture.
5. **The zoom chooses** (built 2026-11-19): the countries from orbit, the provinces from 7 km a CSS pixel and 2,700 km
   across the screen's shorter side (10 km a pixel alone on the first day, which left the countries too little of the
   zoom, then 7 km alone, which a laptop crossed at zoom 2.1), a quarter of a second's fade between them; the setting
   goes.
