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
landmass's reach, which the same row keeps. A tile knows it holds the leader's flag
when its own atlas rectangle is the one in the row.

**The flags are painted by the territory layer, not by the tiles.** `territoryLayer.ts` is a sphere drawn after
the earth and before the tiles. For each pixel, its fragment shader finds the ground under it (the camera is
orthographic, so every pixel looks down one axis onto the tiles' sphere of radius 1), then the lattice point nearest
that ground: the icosahedron face the direction leaves through, the point on that face's flat grid, and its
canonical id (`cellOf` in `domain/latticeCells.ts` does the same, and its tests check that every one of the 260,503
shipped tiles falls in its own cell). Whether the id is a tile, and which texel holds that tile, comes from one word
per 32 ids: their bits and the count of tiles before them (`tileCells`, 226 KB). The tile's flag and landmass come
from `TileField.tileData`, two u16 per tile (1 MB), which `TileField.uploadTiles` sends a row at a time as tiles
change hands. So a pixel wears exactly its own tile's flag: a drawing keeps the shape of its cells, and no tile
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

**It costs one pass over the globe's pixels while it is on**, and it is off (`visible`) wherever `flagPaint` is 0. A
sea pixel stops after one texel. The widened discs it replaces were about 9 million fragments for the front half of
the globe at the last zoom they were drawn.

**A painted flag has no frame of its own.** Ground past the flag's rectangle
wears the colour the flag ends on, and the country outline is the only line
drawn. A dark keyline round the rectangle was drawn until November 2026: it ran
straight along the edge tiles of every landmass, over the outline, and across
open ground wherever a rectangle stopped short, as one capped at
`MAX_FLAG_RADIUS` does in Russia.

**The flags holding the biggest landmasses are drawn from their SVGs, not read
from the atlas.** A sprite is 100px wide and a painted flag is as wide as its
landmass: at the last zoom where the flag is painted alone, about 135 of the
307 painted landmasses stretch their sprite past 1.5× on a plain screen, about
180 on a retina one, and the biggest — Russia, Antarctica, Brazil — past 20×,
so a motto or a device was mud. Eight slots (`SLOT_COUNT` in
`slotRanking.ts`), cells of 768×512 in one 1536×2048 canvas (`flagSlots.ts`;
12.6 MB on the GPU, 17 with its mipmaps, allocated the first time a flag is
drawn into it), hold one flag each, rasterised from
`static/countries/svg/<code>.svg` at its sprite's aspect, as large as fits
inside a 32px margin that repeats its last column and row, so a coarse mip
level finds the flag's own colours at its edge.

`SlotRanking` decides which eight, with no GPU in reach and nothing about the
camera: the flags holding the biggest painted landmasses, each by the
half-width of the largest landmass it holds (`BorderField.flagNeeds`). A flag
keeps its slot while it holds its place; a slot freed by a flag painted nowhere
any more goes to the next in line, and the weakest holder is evicted only for a
flag that needs `EVICTION_MARGIN` (1.2×) more, so a tile war over one landmass
does not redraw the texture every second. The ranking runs on every recount,
never per frame. **A slotted flag samples its slot wherever it is painted**,
whatever the landmass's size and whether it leads it or not: `FlagSlots` keeps
the atlas rectangle of each slot's flag in `slotRegions`, and the territory layer
finds the tile's flag among them and samples that slot's frame from `slotFrames`,
with the same uv and the same `textureGrad` footprint as on the atlas. So one flag looks the same everywhere,
and a flag beyond the eight stays on the atlas everywhere. One slot for the
biggest flag in view was tried first, and a sharp Brazil beside a blurred Brazil
on Peru looked worse than neither. The slot was then only for the landmasses a
flag leads, and that gave the same defect on a contested landmass: a blurred
Brazil on Argentina, led by Bulgaria. The SVG
is fetched the first time its flag earns a slot and kept for the session; the
atlas shows until it is drawn, and a flag whose SVG fails stays on the atlas
and is not asked for again. Nothing is fetched on a page where no flag is
painted.

**Split is opaque.** It is how the season-end clips paint the map, every point in
its own tile's holder's flag, and they looked much better than Big, where a
contested landmass comes out as one faded flag over the photo and the ground the
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
drives the layer fading out, the tiles fading in, and the outline moving from over
the tiles to under them. The layer stays solid through the first half
(`min(1, 2 * flagPaint)`), so the tiles come in over a whole map, and fades in the
second: faded both at once, the earth showed between the discs and the map looked
washed out. `pointSize.test.ts` pins the schedule.

**In orbit a tile is still picked wide** (`pickPointSize`, 1.73 of its size): the
picker reads tiles, not the layer, and a pixel between two discs is still the
territory.

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

**Every tile is in a country**, because both blobs come from one Natural Earth
query: a tile exists exactly where `groundOf` answers a code, and that same answer
is what the borders blob records. The 2,191 tiles that used to fall outside every
country are what the globe drew as discs floating on open water with no outline
round them. See [`/map/README.md`](../../../map/README.md).
