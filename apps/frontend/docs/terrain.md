# The terrain

**The globe is drawn as a board game** (November 2026): each cell of land one flat colour of eight kinds, the sea
three hexagon rings of lighter blue along the coast, then one flat deep blue. It replaced a 4096×2048 satellite photo,
about 43 MB on the graphics chip and 736 KB to download, for ground the flags cover almost everywhere: on 2026-11-17 the
live map had 260,502 of its 260,503 tiles held. Players mostly see the sea, and the ground between the tile discs
zoomed in. The land shows whole only where no flag is, or with "Country flags" Off.

## The file

`terrain-<hash>.bin` is a file of the board (`/map/boards/<board>/`, named in its manifest, copied to each app as the
others are: see [`/map/README.md`](../../../map/README.md#format) for the format). It holds one kind for each point of
the lattice, by canonical id:

| kind | |
|---|---|
| 0-7 | land, under a tile: forest, tundra, boreal, arid, savanna, grass, sand, snow (`LAND` in `domain/terrain.ts`) |
| 8, 9, 10 | the sea one, two or three lattice steps from land |
| 11 | the open sea, four steps and more |

**A kind of land is exactly a tile.** The terrain is cut from the board's runs, `terrain.test.ts` checks the shipped
one, and `npm run map:audit` fails on a point astray. The territory layer reads a land kind as "this point is a tile".

**It is runs, not a byte a point.** Along the canonical order, a kind holds for long stretches: earth-1's 906,012
points are 66,392 runs, 74 KB, where a byte a point is 906 KB. A run is one byte, its kind in the high four bits and its
length in the low four, with the rest of a long run's length after it. A board of four times the points has about
twice the runs.

## `npm run terrain`

`scripts/generateTerrain.ts` reads the board and `scripts/map/earth.jpg`, the photo the game showed until November
2026, its coastline already cut to earth-1's tiles. The photo is read there and never served.

1. **Each tile's colour** is the mean of nine pixels round it, spread wider east and west towards the poles, where
   the photo stretches, in Lab.
2. **Its kind is the one whose photo colour is nearest** (`PHOTO_COLOURS` in `scripts/map/terrain.mjs`). The eight
   colours are the centres k-means (k = 8, in Lab) found over earth-1's tiles. They are pinned: a k-means run again
   numbers its kinds in another order, and the palette, which is in that order, would paint the snow green.
3. **A yellow white is sand, not snow**: a tile nearest snow with b\* past 8 takes its next nearest kind.
   **A tile where earth-1 had sea reads the photo's blue**, which is no kind of land: the photo's coast was cut to
   earth-1's cells. On another board such a tile (its point in a cell of earth-1 that is no tile) takes, wave by wave
   from the coast in, the kind most of its tiles around have (`kindsFromNeighbours`); an island earth-1 did not have
   at all takes the kind of earth-1's nearest tile. `earth-2` has 10,264 of the first and 692 of the second,
   `siege-1` 318 and 58; `earth-1` none.
4. **Three passes of a majority vote** over the lattice neighbours, a tile's own kind counting twice, turn speckle into
   patches. An island of one tile keeps its kind.
5. **The sea's bands** come from a walk out from the land, one lattice step at a time.

Against the look test the owner chose, 13 tiles of 260,503 come out another kind (the test's last k-means step).
Run again, it writes the same file byte for byte. Run it after `npm run map:generate`, which changes the tiles; then
`make map` in the backend and `npm run map` in the backoffice, since the manifest's name changes.

## How the page draws it

**The territory layer paints the land and the coast, at every zoom** (see [The zoomed-out view](zoomed-out-view.md)).
It already finds each pixel's lattice point, so it reads the point's kind from a texture of one byte a point (R8UI,
2,048 columns, 0.9 MB on `earth-1` and 3.5 MB on `earth-2`). A tile's ground is under its flag, `mix(ground, flag, paint)`, opaque, so zoomed in the ground
shows between the discs and the flags fade to it in the second half of the handover. A coast band is its colour. The
open sea is discarded, and the earth's sphere under it shows: one flat colour, with the sea's glint with HD graphics
(see [The light](light.md)). The terrain loads with the globe, beside the borders, so the clips and the player videos
have it from their first frame.

- **Each cell of land is up to 4% lighter or darker** (`SHADE_JITTER`), from a hash of its id that is the same on every
  GPU, so a patch of one kind is not one flat colour.
- **A cell's hexagonal edge is 14% darker** (`CELL_EDGE`), drawn as a cell grows from 10 to 18 pixels wide: zoomed in,
  the land between the discs is a board of hexagons. The width of a pixel on the ground is taken before the shader's
  first `discard`, since a derivative after one is undefined, and handed to the edge.
- **The colours** are in `terrain.ts`, the board-game palette the owner chose:

  | forest | tundra | boreal | arid | savanna | grass | sand | snow |
  |---|---|---|---|---|---|---|---|
  | `#2e8b47` | `#a9b79a` | `#3d7d55` | `#c98a4b` | `#b6b84e` | `#6caf43` | `#e8c77a` | `#e9eff3` |

  The sea, one, two, three steps out and the open sea: `#4a9bd6`, `#3a86c6`, `#2e72b3`, `#245f9e`. **They go into the
  shader with no sRGB conversion** (`setStyle(hex, LinearSRGBColorSpace)`): the shaders write a colour as it is, and a
  converted colour came out dark.

**The open sea is a dark blue on purpose.** In the look test a pale sea hid the light-blue flags (Greece; Palau, 8.7%
of the map); a dark one keeps them clear. Waves (a "~" mark on the open sea) were tried and dropped.

## What it costs

**Measured** in headless Chrome on an M2 Max with the fake backend, 1280×800, HD graphics at a device ratio of 2 and
plain at 1, CPU slowed 4×, three rounds of before and after in turn, medians (2026-11-17):

| | before (photo) | after (terrain) |
|---|---|---|
| download | 736 KB of JPEG | 74 KB of terrain |
| graphics memory, HD / plain | 84.2 / 88.4 MB | 40.4 / 44.6 MB |
| GPU ms a frame, landing spin, HD / plain | 3.8 / 1.2 | 3.5 / 1.1 |
| the same with "Country flags" Off | 5.5 / 1.7 | 6.2 / 1.8 |
| zoom 3 | 3.3 / 1.4 | 4.2 / 1.4 |
| zoom 10 | 1.5 / 1.2 | 2.6 / 1.7 |
| zoom 40 | 1.6 / 0.9 | 2.9 / 1.3 |

The CPU's time a frame, the JS heap and the hover pick did not move.

**Zoomed in, the layer runs where it was off before**, so a frame costs more there: still less than the landing view,
which every device already draws, and only when something changes ([Drawing only when something
changed](render-on-demand.md)). **With no flag painted, it stops at the ground**: where `flagPaint` is 0, zoomed in or
with "Country flags" Off, it returns the land's colour before it reads the tile, the flag and the effects. That took the
zoomed-in frames from 3.4-3.7 ms to 2.6-2.9 ms with HD. What is left is the walk over the twenty faces that finds each
pixel's lattice point.

## Before: the photo

Until November 2026 the sphere under the tiles was a satellite photo, cut from the tile field by `npm run earth`: a
pixel was land where its nearest lattice point was a tile, and the photo's colour was moved toward land or water to
match (`recolour.mjs`), so that a one-tile island was not drawn as open water and a disc never floated on the sea.
At 4096×2048 a cell was three pixels across, so it was a correction that never reached every island. The terrain has
nothing to correct: its land is the board's own tiles.
