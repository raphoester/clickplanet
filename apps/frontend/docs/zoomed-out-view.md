# The zoomed-out view

Zoomed out, a tile is about a pixel and a half sampling a 100px flag out of one
shared 1300×1232 atlas, so the GPU picks a mip level that is the whole atlas
averaged and every country comes out the same mud. Turning mipmaps off swaps mud
for sparkle. Neither end works, so from far enough away the globe is drawn from
something coarser than a tile.

Every tile resolves **offline** to a *landmass*: a country's tiles split into the
separate pieces of land they actually form (Natural Earth 1:50m, 239 countries →
694 landmasses). Neither borders nor tiles move, so `npm run map:generate` writes
the whole table once and nothing recomputes it at runtime. Two tiles are the same
piece when they **touch on the lattice** and carry the same country — not when
they are within a tuned radius, which reached past the neighbours in places and
joined two islands across a strait into one flag painted over the water between
them. A landmass rather than a
country because a flag belongs to a piece of ground — one frame spanning mainland
France, Corsica, Guiana and Réunion would stretch the tricolour across half the
planet and paint nothing recognisable anywhere.

**The flags are painted in their landmass's frame**, with distance measured
**along the surface**, so they bend with the globe and are cropped by their own
coastline. A player picks how, under "Country flags" in Settings (`MapView` in the
display settings): **Split**, the default, paints each tile in its own holder's
flag, so a landmass one flag holds whole is one picture and a landmass several
flags share is that many pieces; **Big** paints the whole landmass in the flag that
leads it, at its share; **Off** paints nothing (below). The choice reaches the
running globe through `Globe.setMapView` and the `splitFlags` uniform, so it never
rebuilds it. The flag that leads the landmass gets the frame fitted to it and its slot:
`BorderField` writes it from the server's row for the landmass (`setRows`, with
`Holdings.leaderOf`: see [The counts are the server's](counts.md)) into one row
per landmass of a `DataTexture` the territory layer reads. Any other flag covers the
landmass from its middle (`coverOf` in the territory fragment shader), from the
landmass's reach, which the same row keeps, and never less than about a tile of the board drawn (`leastFlagRadius`). A
landmass of fewer than 4 tiles of `earth-1` (as much ground on another board, `leastPaintedTiles`) has no leader whose
flag is fitted to its frame: see [Sized by the board](board-scale.md). A tile knows it holds the leader's flag
when its own atlas rectangle is the one in the row.

**The flags are painted by the territory layer, not by the tiles.** `territoryLayer.ts` is a sphere drawn after
the earth and before the tiles. For each pixel, its fragment shader finds the ground under it (the camera is
orthographic, so every pixel looks down one axis onto the tiles' sphere of radius 1), then the lattice point nearest
that ground: the icosahedron face the direction leaves through, the point on that face's flat grid (on a warped board,
in the lattice's own weights: two Newton steps undo the warp, `unwarped`; and the tile's centre for its effects is
warped back, `warped`), and its canonical id (`cellOf` in `domain/latticeCells.ts` does the same, and its tests check
that every tile of every shipped board falls in its own cell). The face data is the board's: its rotated corners, and
its warp in `latticeWarp`. Whether the id is a tile, and which texel holds that tile, comes from one word
per 32 ids: their bits and the count of tiles before them (`tileCells`, 221 KB on `earth-1`, 850 KB on `earth-2`,
built from each tile's canonical id as the board's runs give it). The tile's flag and landmass come
from `TileField.tileData`, two u16 per tile (1 MB on `earth-1`, 4 MB on `earth-2`), 1,024 a row, of which `TileField.uploadTiles` sends only the rows whose tiles
changed hands. So a pixel wears exactly its own tile's flag: a drawing keeps the shape of its cells, and no tile
reaches over another.

**The tiles drew it until November 2026, and that is why they overlapped.** Each disc was widened to 1.31 of the
spacing so the discs covered the ground, and each painted the flag at the ground under each of its pixels. Where
two discs of different holders overlapped, the depth test chose the one in front, and which one that is changes as
the globe turns: the edges of a drawing moved and flickered. The discs are now never widened, and a disc stays inside
its own cell but round the lattice's 12 corners, where two neighbours are as close as 0.73 of the median spacing.

**A flag is read at the pixel's own ground.** The footprint the atlas is sampled over (`textureGrad`) is how far the
flag's uv moves to the next pixel across and up, from `dFdx` and `dFdy` of the ground, which runs on smoothly from
one tile to the next: derivatives of the uv itself would jump at every edge between two flags and blur it. Without a
footprint the driver takes the top mip level and point-samples a 100px flag into a few dozen pixels.

**The layer draws a tile's effects too**: a bomb's flash, ring and scorch, and the fortify wave, from the same
`tileEffects.glsl` the tiles' vertex shader reads, at the tile's own centre, so they are drawn cell by cell as on the
tiles. The tiles' own effects fade out as the layer comes in; drawn on both, the gaps between the discs showed as dots.

**It costs one pass over the globe's pixels, at every zoom**: it paints the terrain under the flags, and alone where
`flagPaint` is 0 (see [The terrain](terrain.md)). An open-sea pixel stops after one texel and leaves the earth's sphere
to show. Until November 2026 it was off (`visible`) wherever `flagPaint` was 0, and the earth's photo showed there. **While it is solid the tiles are not drawn under it**, but for the few wearing a ring
(see [The tiles out of sight are not drawn](far-side.md)): the landing view and the idle spin cost the same whatever the
number of tiles. The widened discs it replaces were about 9 million fragments for the front half of
the globe at the last zoom they were drawn.

**A painted flag has no frame of its own.** Ground past the flag's rectangle
wears the colour the flag ends on, and the country outline is the only line
drawn. A dark keyline round the rectangle was drawn until November 2026: it ran
straight along the edge tiles of every landmass, over the outline, and across
open ground wherever a rectangle stopped short, as one capped at
`MAX_FLAG_RADIUS` does in Russia.

**The flags on screen are drawn from their SVGs, not read from the atlas.** A sprite is 100px wide and a painted
flag is as wide as its landmass's frame: zoomed out over Africa on a retina screen, Algeria, Libya and Sudan stretch
their sprite 6 to 7 times, and a crescent, a star or a coat of arms is mud. So each flag in view gets a cell of its
own in one texture, `flagSlots.ts`, rasterised from `static/countries/svg/<code>.svg` at its sprite's aspect.

*How wide a flag must be* (`flagNeedOf` in `slotNeed.ts`): the width of its frame on screen, in drawing-buffer pixels,
or its height times the flag's aspect if that is more. Each axis is foreshortened by its own slant at the frame's
centre, so a frame at the edge of the earth still needs its full height. A flag takes the largest need of the
landmasses it leads. *How much land it shows* (`LandmassSight`): up to 32 tiles of each landmass, each standing for its
share of it, counted when they face the camera and are on screen, each by its slant. Both are measured from the
camera: what is behind the earth or off screen needs nothing.

*The texture* is sized from the drawing buffer (`slotBudgetOf` in `slotCells.ts`), because the flags on screen share
the screen: one 2048×1024 block per 2 million pixels, rounded, 1 to 4 of them, stacked. That is 2048×2048 on a
retina desktop (2880×1800, 21 MB with its mip levels) and 2048×1024 on a phone (11 MB). No side passes 4096, some
players' `MAX_TEXTURE_SIZE`. A flag is never drawn wider than the screen's longer side: 2048 on a retina desktop, 1024
on a phone. The texture is allocated the first time a flag is drawn into it, and only its storage: nothing is sent
until a cell is.

*The cells* halve down from a 2048×1024 block: a 2:1 cell into two squares side by side, a square into two 2:1
cells, down to 128×64 (`CellPacking`, a buddy allocator). Every cell sits on a multiple of its own size, so a mip level
never mixes two cells until a cell is smaller than a texel. A flag sits inside a margin of 1/32 of the cell's shorter
side, its last column and row repeated out to the cell's edge, so a coarse mip level finds the flag's own colours at
its edge.

*Which flags, and how large* (`rankSlots` in `slotRanking.ts`). A flag is worth a cell only when it needs 1.25 times
its sprite's width. It earns the smallest cell that holds it as wide as it needs, but no larger than its land on
screen pays for: so many texels per pixel of land, one share for the whole view, the largest of 16, 8, 4, 2, 1, ½ and ¼
whose cells fit the texture. A frame can be far larger than its land: ranked on width alone, Chile's thin strip and
Brazil took the whole texture over South America, and Russia half of it over Europe. The room left then raises the
flags with the most land to the share above. A flag keeps its cell unless it needs clearly more (`EVICTION_MARGIN`,
1.2 times its width) or two sizes less, so a tile war or a small move of the camera redraws nothing. A flag out of
view keeps its cell while there is room and is dropped first. When the texture is still full, the flags with the
most land win and the rest drop a size. `SlotLayout` keeps a flag's cell while its size holds, and places the others
largest first; when the free cells are too broken up for one, it drops a size.

*When*: when the camera settles (`Settling`): once each time it comes to rest somewhere new, sampled every 250 ms, and
every quarter radian while it turns slowly, as in the idle spin. Never while it moves fast, and never per frame. A
recount ranks the flags again for the view they were last ranked for, never for a camera in motion. Nothing is ranked
where no flag is painted: zoomed in past the handover, or with the painted flags off.

*Drawn one at a time, off the frame*: the most land first, a flag's SVG is fetched once a session, its size set on
the text (parsing the largest flags, 100 to 400 KB, as a document held the page), rasterised into a canvas the cell's
size in an idle moment, and turned into an `ImageBitmap`, which finishes the raster off the frame. The next frame sends
that cell alone (`copyTextureToTexture`) and the GPU rebuilds the mip levels; no frame does more than one. Measured
on a retina desktop with the CPU slowed 4 times, no camera stop made a frame longer than before (17 ms); sending the
canvas itself, its raster ran inside the frame, up to 100 ms. Until a flag is drawn it stays on the atlas, and a flag
moved to a new cell shows its old one until the new one is sent or another flag is drawn over it. A flag whose SVG fails stays on the atlas and is not
asked for again.

**A slotted flag samples its slot wherever it is painted**, whatever the landmass's size and whether it leads it or
not: a data texture holds each flag's frame in the slot texture by its number (`flagNumbers` in `atlas.ts`), 0 for a
flag on the atlas, and the territory layer reads it once a pixel, the tile's own flag in Split, the leader's (kept in
its landmass's row) in Big. So one flag looks the same everywhere, and 50 slotted flags cost a pixel what 8 did. One
slot for the biggest flag in view was tried first, and a sharp Brazil beside a blurred Brazil on Peru looked worse
than neither. The slot was then only for the landmasses a flag leads, and that gave the same defect on a contested
landmass: a blurred Brazil on Argentina, led by Bulgaria. Until November 2026 eight slots went to the flags holding the
planet's biggest landmasses, whatever was on screen, and the whole 12.6 MB canvas was sent again for any change.

**Split is opaque.** It is how the season-end clips paint the map, every point in
its own tile's holder's flag, and they looked much better than Big, where a
contested landmass comes out as one faded flag over the terrain and the ground the
others hold does not show. Zoomed in, the tiles are still at 0.7 over the ground,
so a country is a little fainter once its tiles have the frame.

**Big's opacity is the leader's share, and the curve it goes through is not a free
knob.** Zoomed in, that share is already on screen as the fraction of discs
wearing the holder's flag, so the tiles show `share * 0.7` of ink no matter what;
the painted flag shows `share * 0.94`. Bend the curve and the summary becomes
fainter than the tiles it hands over to, and a country gets *brighter* as you zoom
into it — measured at 5x for Sudan with the share cubed. `borderField.test.ts`
pins this.

**One schedule owns the whole handover** — `coarseHandover` in `pointSize.ts`
drives the layer's flags fading out, the tiles fading in, and the outline moving from over
the tiles to under them. The flags stay solid through the first half
(`min(1, 2 * flagPaint)`), so the tiles come in over a whole map, and fade to the bare
terrain in the second: faded both at once, the ground showed between the discs and the map looked
washed out. `pointSize.test.ts` pins the schedule.

**A pixel picks the tile whose cell holds it**, the cell the layer paints, so a pixel between two discs is still the
territory: see [The tile under the pointer](picking.md).

**A player can turn the painted flags off** — Off under "Country flags".
Players draw pictures with the tiles, and the painted flag hides them. With it off
(`MapView` `"tiles"`), the handover in `pointSize.ts` is held at 1 at every zoom, so the
globe is drawn as it was before #52: each tile its own flag at its own size,
and the outline always under the tiles. Zoomed out that is the mud described
above, and that is the price of seeing every tile. It used to be a switch, "Big
country flags": a setting saved by it as `"flags"` now reads as the default, Split.

`npm run flagFit` decides the rest for the leader's flag: a flag that is only bands can be pulled to
the country's own shape and still say what it is, while one carrying a device is
cropped, anchored on the part that names it rather than on its middle. The
result is `static/countries/flagFit.json`.

**Every tile is in a country**, because the runs and the borders come from one Natural Earth
query: a tile exists exactly where `groundOf` answers a code, and that same answer
is what the borders blob records. The 2,191 tiles that used to fall outside every
country are what the globe drew as discs floating on open water with no outline
round them. See [`/map/README.md`](../../../map/README.md).
