# Provinces: fortify on smaller ground

**Designed 2026-11-18, with the owner. Nothing is built yet.** Each PR at the end turns its part of this doc into "how
it works" and updates the docs it makes stale: [Fortify](../apps/backend/docs/fortify.md), the frontend's
`fortify.md`, `zoomed-out-view.md`, `territory-card.md` and `country-outlines.md`, [Map geometry](map-geometry.md) and
[`map/README.md`](../map/README.md).

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

The generator does it, from `ne_10m_admin_1_states_provinces` at the Natural Earth tag it already pins (v5.1.2), which
it already downloads for the landmass names.

1. **A tile takes the unit under its centre**, only if the unit is in the tile's own country, by the game's own country
   rule (`countryCodeOf`, with its `ABSORBED`: Somaliland and Baikonur, which Natural Earth gives no country, are
   Somalia's and Kazakhstan's; Northern Cyprus is Cyprus's). So Somaliland is a province of Somalia.
2. **A tile never changes country.** A tile under another country's unit, or under none, takes the unit of the nearest
   tile of its own landmass that has one. The two Natural Earth maps are drawn at different detail and do not agree
   everywhere. Measured on `earth-2`:
   - 2,531 tiles under another country's unit: a disputed line (Aksai Chin's Chinese tiles under India's "Ladakh" join
     Tibet; Arunachal Pradesh's Indian tiles under China's "Xizang" join Arunachal Pradesh), or a border drawn a little
     apart (4 Mexican tiles under "Texas" join Chihuahua);
   - 13,618 under no unit: Antarctica's ice shelves (10,688), which the game counts as ground and admin-1 does not, and
     coasts and islands where the country is drawn coarser than the unit (Nunavut's Arctic coast, the Aleutians, Chile's
     fjords).
3. **A landmass where no tile has a unit is one province**, named as it is today: 72 tiles in 68 landmasses, the 1- and
   2-tile islands and the seats (the Vatican's tile sits on Rome's ground, under Italy's "Lazio").
4. **A piece** is the tiles of one unit that touch on the lattice, inside one landmass. So a province never crosses a
   sea or a country border, and Alaska and Hawaii stay apart from the lower 48.
5. **A piece under N tiles joins a touching piece of its landmass**, the smallest first. It prefers one in the same
   Natural Earth `region` (France's 13 regions, Italy's 20), then one still under N, then the longest shared edge.
   It never crosses a UK nation (`geonunit` other than `admin`: England, Scotland, Wales, Northern Ireland), so a whole
   nation stays a province even under N (Wales is ~140 tiles). A landmass under N stays one province, as today.
6. **No upper limit.** Sakha is ~20,000 tiles on `earth-2`, Texas 4,675, Antarctica ~94,000.

**N is a number per board**, set by hand in the board's definition beside its `cols`, not a formula: a rule by area
would copy `earth-2`'s cut onto `siege-1`, and the owner wants `siege-1` to group more.

| board | N | provinces | of N tiles or more | note |
| --- | --- | --- | --- | --- |
| `earth-2` | 200 | 2,108 | 1,089 | Germany in ~7 to 10; most US states alone |
| `siege-1` | 20 | 787 | 515 | a tile is 79 km apart, ~5,500 km²: 20 tiles are about Bulgaria; France in 2 to 5 |

`earth-1` has none: it only draws history, and its fortifies keep their landmass. **Changing N later** is a regeneration
and a deploy: no tile changes id, and the locks are dropped and settled again as at the first switch.

## Names

- **One unit: its name.** All the units of one Natural Earth region: the region's name ("Normandie"). Else the name
  of its largest unit ("Brandenburg" for Brandenburg and Berlin).
- **Overrides**, where a better name exists ("New England", "Southern Germany"). Claude writes them in the generator's
  PR, and the owner reviews the list there. An override names a province by the exact set of units in it (their
  `adm1_code`), and **the generator fails on an override that matches no province**, so a change of N makes the names
  checked again. Natural Earth's own bad names are overrides too: an unnamed Antarctic unit (Alexander Island, 1,012
  tiles), "Eastern" for a part of Scotland. At N 200, `earth-2` has ~500 provinces of more than one unit; many only
  take a sliver of a neighbour and keep a good name.
- **In 8 languages**: Natural Earth's own `name_xx` where it has one, the translate skill (Claude Code on Haiku) for the
  overrides and the gaps, as for the landmass names.

## The board's file

A new kind of board file, `provinces-<hash>.bin`, named in the manifest (`BoardFiles` in the page's `board.ts`, and
`cpboards`): the province of each tile (uint16), and each province's landmass, frame (centre and half extents, as the
borders blob's landmasses) and tile count; its names in a file per board beside the landmass names. **Adding it renames
the manifest only**: the board keeps its name and positions hash, so no new board, no era, no remap (see
`boards-and-eras.md`). The backend embeds it and the page fetches it; the backoffice needs none. A preview command draws
a board's provinces as PNGs for a list of N, one color a province, the country borders in black.

## Fortify on provinces

- **Fortify, the lock and "never the same flag twice in a row" play on provinces** instead of landmasses.
- **The switch**: the deploy drops the landmass locks, and the boot settles every whole province to its holder with no
  shield, as it does for a whole landmass with no lock. No shield is taken away, and nobody gets a free fortify.
- **The wire**: `province_fortified` in place of `landmass_fortified`, `GetFortresses` by province, and the hover card's
  last fortify read for the province. The ledger's `fortify` names the province and its file; an old event keeps its
  landmass and borders file. A wire change, so the backend and the page ship together (see
  [A server restart reloads the page](reload-on-restart.md)).
- **The chat** names the province and its country: "France fortified Bavaria, Germany". The floor stays
  `fortifyNewsTiles` (200 `earth-2` tiles).
- **Unchanged**: the fortify missions (they count tiles), 0.2 XP a tile fortified, the bomb and rally lines, the clips'
  `landmasses_flipped`, the country sizes.
- **Balance**: every knob as it is. For a week after the deploy, measure the shielded share of the map (83% on
  2026-11-05) and the fortify news lines a day, then tune `charges.shield.perTile` and the floor.

## The page

- **A setting, "Countries / Provinces"**, beside "Country flags", kept in `clickplanet-display-settings` like the
  others, Countries by default. It changes the hover's outline and name, and the frame each big flag stretches over: in
  Provinces, the US flag repeats over each state. A big flag's leader is read from holdings per province.
- **Fortify reads provinces in both modes**: the pulse on the tiles a flag still misses, the "Fortified!" card ("France
  owns all of Bavaria"), the hover card's last fortify.
- **The lines.** From orbit, Countries draws the country lines and Provinces adds the province lines in light grey.
  Zoomed in, both, in every mode.
- **The outlines are straight cell edges, with no B-spline** (the owner, 2026-11-18). The tiles are straight hexagons,
  so a curve round them reads worse, and it costs four pieces a cell edge pushed in and two from orbit, where straight
  costs one. Province lines add 70% to `earth-2`'s outline edges at N 200 (90,834 on 129,431); the straight edges pay
  for them.
- **The landmass texture holds 2,048 rows** and `earth-2` has 2,108 provinces, so the provinces' texture is taller (4,096
  at most, see the players' GPU limits).

## The PRs, in order

1. **Straight outlines.** The spline goes, and its junction clamps with it. It ships alone and helps the lag of
   2026-11-18. Culling the outlines by chunk comes only if the GPU timer still finds them heavy.
2. **The generator**: the provinces file of `earth-2` and `siege-1`, the names, the overrides and their translations,
   the preview, the copy in each app. Nothing reads it yet. This doc lands with it.
3. **The backend and the page together**: fortify by province, the switch of the locks, the wire, the ledger, the hover
   card's last fortify, the chat line.
4. **The page**: the setting, the hover and the big flags by province, the grey lines, the taller texture.
