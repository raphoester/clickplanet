# The countries' outlines

A thin dark-grey line runs between one country's ground and the next, and it is
the **same width at every zoom** — `borderLines.ts` builds each piece as a quad
laid out in screen pixels, not on the sphere, so pulling the view back thins
nothing. That is the only way a border survives the range this camera covers: at
0.5 the whole planet is half a screen tall, at 50 a single tile is a disc you can
aim at. Grey and not black: black carried fine at map scale but up close, where
the line is the only thing between two rows of discs, it read as a bar drawn over
the planet rather than a border on it.

**The line is not the administrative border.** It is the boundary of each
country's *tiles*. The tiles are the vertices of a geodesic sphere, so their
cells are its dual — a honeycomb — and the outline runs along the cell edges
between two tiles that belong to different countries. Natural Earth's border is
therefore moved onto the lattice, by up to half a tile, about 12 km.

That move is the point. A line on the real border crosses tiles, and a crossed
tile belongs to one side while reading as split between both. A line on the cell
edges passes *between* the discs: the tile field is 76% covered once zoomed in,
and the gap it leaves is exactly where this runs. So a tile is never cut, and
which side of a border it is on is never a matter of where the line happened to
fall across it. `borderLines.test.ts` pins the width against that gap.

**The line is drawn straight along the cell edges**: each cell edge is one piece, a quad from corner to corner, in every
pass (from orbit, where a cell edge is under a pixel, the over pass joins them; below). The outline turns 60° at every
corner, as the cells do. Until 2026-11-18 it was drawn as a quadratic B-spline through the middle of each cell edge,
four pieces to an edge pushed in and two from orbit. The owner dropped it that day: the tiles are drawn as straight
hexagons, so a curve around them reads worse than their own edges, and on `earth-2` the spline's pieces were most of the
vertices a frame drew. Straight, the outline is a quarter of the pieces pushed in and half of them in orbit, and the
code has no smoothing to keep (measured in [The outlines drawn
straight](../../../docs/boards.md#the-outlines-drawn-straight-measured)).

**A straight cell edge is the corridor's own edge.** The corridor between two tiles of different countries is narrowest
at the middle of the cell edge between them, half a spacing from each tile's centre, and the line runs through it. At a
corner, where three cells meet, the corridor is wider (a spacing over √3 from each centre), but a piece's quad runs half
a width past its end, square, straight at the third tile there. `borderLines.test.ts` measures both on a lone tile's
cell at every zoom the tiles are drawn alone: the line's solid width clear of the tile's own disc at the middle of each
side, and each piece's end clear of the neighbours' discs at the corners. The corner is the tighter of the two: it
holds `WIDTH` under about 2.2px, where the middle alone would allow 2.5px.

**The passes share their pieces.** The over, the under, the highlight and the mission pass are four materials over the
cell edges and the same draws, except from orbit, where the over and the highlight pass draw one coarse set of pieces
instead (below). A second kind of line, with its own colour, would be its own pieces and passes.

**Only the pieces in sight are drawn**, as with the tiles (see [The tiles out of sight are not drawn](far-side.md)).
Before each frame, `cull` finds the chunks of the outline's pieces that are not behind the earth or off screen, and
each pass draws them as up to 8 ranges of its pieces, in their order. Each set of pieces, the cell edges and the coarse
set the over pass draws, is culled for the outermost lift and the widest line, so no pass loses a piece it would draw.

**From orbit, the over and the highlight pass draw fewer pieces.** At the landing view of `earth-2` a cell edge is half
a drawing-buffer pixel, and the over pass drew a quad for each: 112k quads, each 2.5 pixels wide and 3 long, about six
of them over each pixel of the line. `coarseOutlinesOf` (`domain/coarseOutline.ts`) makes five coarse sets, at 0.5,
0.71, 1, 1.41 and 2 cell edges (`coarseTolerancesOf`). Each run keeps its first and last corner (every junction and
run end) and its two landmasses, and drops an inner corner while every corner dropped since the last one kept lies
within the set's tolerance of the piece that replaces them. No corner moves, and the line runs straight from corner to
corner. Before each frame `cull` picks the coarsest set whose tolerance is at most `TOLERANCE`, half a drawing-buffer
pixel, on screen. The camera is orthographic, so nothing on the globe shows longer than it is: the bound holds
everywhere up to the limb. Where a cell edge is longer, the over and the highlight pass draw the cell edges: on
`earth-2`, on a buffer 1,600 pixels high, from zoom 1.06. The under and the mission pass always draw the cell edges.

- **One coarse set would not do.** The over pass is drawn up to `tilesZoomOf`, and at `flagsZoomOf` a cell edge of
  `earth-2` is already about 4 CSS pixels long: no corner can go there. So the sets follow the zoom, each a √2 step from
  the next, so that a screen of any height finds one close to its half pixel.
- **The same ink.** A quad reaches half a width past each of its corners, so from orbit each pixel of the line was
  inked by several quads, and the 70% black stacked up to nearly solid. A long piece covers a pixel once: the line would
  come out a third lighter and thinner. So a coarse piece inks as its cell edges' quads did. Each piece carries the
  cell edges it stands for and their chord over their length (`standsFor`). The vertex shader widens its quad by that
  slant, since the cell edges run 30° to 60° across the line and their quads made it wider, and counts how many of their
  quads would have covered a pixel of it, from its length on screen. The fragment shader inks `1 − (1 − a)^k` of it, and
  half of that where two pieces' ends overlap. This is the `STACKED` shader, a second material of the over and the
  highlight pass with the first one's uniforms; the cell edges draw with the first, as they did.
- **A piece never dips under the tiles.** A chord cuts under the sphere: a piece's middle stays within a fifth of the
  over pass's lift above the tiles (`MOST_DIP`), so neither the tiles nor the earth hide it.
- **A ring keeps its corner farthest from where it starts**, so a lake of one tile is never drawn as a point.
- **Built off the page.** `loadBorderLines` hands the file to a worker (`borderLinesWorker.ts`), which decodes it and
  makes the cell edges' pieces and chunks and the five coarse sets with theirs (`outlineSetsOf` in
  `domain/outlineSets.ts`, about 50 ms in Node on `earth-2`). It runs beside the lattice's worker, as the board's files
  land, and the page builds them itself if the worker fails. The page's main thread only makes the GPU buffers. The
  coarse sets are 167,533 pieces on `earth-2` beside the 129,625 pieces of its cell edges: 4.0 MB more of buffers, each sent to the GPU
  the first time it is drawn.
- **Tested.** `coarseOutline.test.ts`, on every shipped board: no corner dropped farther than the tolerance, every
  run's ends and sides kept, every cell edge counted once, no dip past `MOST_DIP`. `outlineSets.test.ts`: each set's
  chunks hold all its pieces, and the worker hands over every buffer once. `borderLines.test.ts`: on five
  screens, at every zoom the over pass is drawn at, the farthest of its pieces from the cell edges is within `TOLERANCE`
  on screen, and the highlight draws the same pieces; the cull keeps every piece in sight of the set each pass draws,
  with the coarse quads as wide as their slant; and both shaders' ink, copied, on a zigzag and an armchair run of cell
  edges under a pixel each, puts the coarse line within 15% of the cell edges' ink, where a line inked once is a third
  lighter.
- **Measured** in [The outlines from orbit](../../../docs/boards.md#the-outlines-from-orbit-measured): at the landing
  view of `earth-2`, 23k pieces drawn instead of 112k and 0.06 ms of GPU instead of 0.30 per render, the line's ink
  within 5% of what it was.

**A run ends at every junction** — a corner three countries share, or where a land border reaches the sea. The
smoothing needed the cut to pin its curve to the point; the straight line does not, but the cut keeps each run between
one pair of landmasses, which the highlight and the mission outline read (below).

**It is drawn twice, on either side of the tiles, and `flagPaint` picks.**
Zoomed in the outline sits just inside the tile shell, so the discs' own depth
hides whatever they cover and the line only ever shows in the gaps — which is
the whole width of it. Zoomed out that gap is gone: the territory layer paints
the flags over the whole ground, and a line underneath it would be invisible. So the same outline is drawn just outside the shell as
well, fading with the painted flag, and the two hand over on the one schedule that
owns the rest of the handover. **The over pass is thinner and darkens the flag
under it** (`ORBIT_WIDTH`, 0.25px and a pixel of feather, black at `ORBIT_INK`, 70%): a red flag gets a
dark red edge, as in the season-end clips, where the grey line read as a stroke
drawn across the map. Neither pass writes depth, so the stretch where they overlap
— a coast, which has no tiles on the sea side to hide the under pass — comes out a
little darker during the handover.

`npm run borderLines -- <board>` writes the geometry of a board (of every board in `/map` with none named), from its
runs and borders and nothing else, beside the board's copy: `static/boards/<board>/borderLines-<hash>.bin`, named in
`BOARDS` (see [Static assets](static-assets.md)). The globe loads the border lines of the board it draws. It walks the whole lattice the tiles were cut from and its triangles,
in the board's order (`latticeTriangles` in `domain/lattice.ts`: three's for `earth-1`, canonical for `earth-2` and
`siege-1`, at their warped and rotated positions), because the runs hold
only the land points and a coast needs the sea around it.

**A cell corner is where the ground's cells meet**, so the line is the hexagons the ground draws, thickened where a
country ends. The territory layer paints each pixel with the nearest lattice point on its face's flat grid, and the pick
finds a tile the same way (`cellOf` in `domain/latticeCells.ts`, see [The zoomed-out view](zoomed-out-view.md)). So three
cells meet at a lattice triangle's centroid on that grid, its weights warped and projected as a lattice point's are
(`triangleOnGrid` and `gridDirection` in `domain/lattice.ts`). A cell edge on an edge of the icosahedron crosses from one
face's grid into the next, and bends there, at the middle of its lattice edge: it is two pieces, with that point as a
corner. Until 2026-11-18 a corner was the circumcentre of its lattice triangle, the true Voronoi diagram of the tiles on
the sphere. That is another tiling: on `earth-2` its corners were 0.04 of a grid step off the ground's at the median and 0.10
at worst, on an icosahedron edge. Pushed in, the line ran beside the hexagons, cutting into a lake's cell on one side and leaving
a strip of it outside on the other, plainest on a coast. The corners still meet exactly, so there are no seams to cover
up at the joins. The edges are then chained into runs, which is what
keeps the file to one corner per edge rather than two, and **each corner after a run's first is written as its
difference from the one before** (version 3 of the file, 2026-11-18, `domain/borderLinesFile.ts`; the format is in
[`/map/README.md`](../../../map/README.md#format)): a cell edge is a few dozen units of the int16 corners, a byte as a
zigzag varint. Version 2 wrote each corner as three plain int16s, which brotli packed poorly, and the file was most of
what `earth-2`'s globe waited for. Size raw, and packed with brotli at 5, on the circumcentres (edges, version 2 and
version 3) and on the ground's corners (pieces, the cell edges and their halves on the icosahedron's edges, and version
3), whose steps repeat more and pack better:

| board | edges | version 2 | version 3 | pieces | version 3 on the ground's corners |
| --- | --- | --- | --- | --- | --- |
| `earth-1` | 56,790 | 362 KB, 324 KB packed | 222 KB, 122 KB packed | 56,948 | 222 KB, 121 KB packed |
| `earth-2` | 129,431 | 810 KB, 767 KB packed | 422 KB, 220 KB packed | 129,625 | 422 KB, 179 KB packed |
| `siege-1` | 14,419 | 99 KB, 90 KB packed | 87 KB, 63 KB packed | 14,547 | 88 KB, 63 KB packed |

`borderLinesFile.test.ts` checks that each shipped board's lines, the countries' and the provinces', put every corner
where the ground's cells meet, to within the int16s (under 0.02 of a grid step; the circumcentres were 0.10 off at
worst), and that the file writes back to the same bytes. `latticeCells.test.ts` checks that the cells `cellOf` finds
meet there, at each triangle's centroid and at each lattice edge's middle.
The renderer draws the blob as it is: each two corners side by side in a run are a piece.

**The coast is the tiles' edge and nothing else.** A one-tile lake gets a ring of
its own, and a bay one tile wide is drawn as a bay. It used to give a sea vertex
its neighbours' country when four of the six agreed, which closed about 1,650 of
them: the coast came out smoother, but ran up to two tiles out over the water,
around ground nobody can click.

**Each run says which two landmasses it is between** (version 2 of the blob, the sea
being 0), so one territory's outline can be drawn on its own: the white line round a
territory in orbit (see [A territory, from orbit](territory-card.md)). A run never
changes landmass between two junctions, since two tiles of one country that touch are
one landmass; the generator checks it and refuses otherwise. The highlight is a third
pass over the pieces, each carrying its run's pair, drawn over the tiles and only while a
territory is lit; the vertex shader drops every piece that does not border it.

**A mission's ground is outlined in red**, the red of a mission's ring on a tile (`setAimed`, see [Missions](missions.md)), a fourth pass drawn over the tiles at
every zoom, `MISSION_WIDTH` (2px) wide: a ground is looked at pushed in as well as from orbit. It reads the pair of
landmasses each piece runs between, and the vertex shader reads in a texture of
`AIMED_COLUMNS` (1024) a row whether each landmass is a mission's: a piece is drawn when one side is and the other is not,
so the outline runs round the ground's coasts and its borders with every other country, and never inside it.

**The province lines** are the cell edges between two tiles of one country in different provinces (see
[Provinces](../../../docs/provinces.md)). Over the painted flags they are 0.85 grey at `PROVINCE_ORBIT_INK` (55%) and
`ORBIT_WIDTH`. **Under the tiles they darken as the tiles grow, and turn dashed**: at `WIDTH`, `PROVINCE_COLOUR` (0.6
grey) while a tile's disc is `PROVINCE_DARK_FROM` (10 CSS px) or less, the countries' 0.25 grey from
`PROVINCE_DARK_UNTIL` (20 px), the colour and the dashes coming in together in between (`provinceDarkOf`). A dash is
`PROVINCE_DASH` (8 px) and its gap `PROVINCE_GAP` (4 px). Each piece rounds that period to a whole number of periods,
with a dash centred on each of its ends, so the line is inked at every corner where it turns (`dashOf` in the fragment
shader, under `DASHED`). Until 2026-11-18 the line stayed light grey at every zoom. On the green ground, between the
flags, it could hardly be seen pushed in, where a province is the ground a fortify plays on. Dark, it reads as clearly
as a country's border; dashed, it is not taken for one. `borderLines.test.ts` checks the schedule, and the dashes on a
copy of `dashOf`: both ends of every piece inked, one gap a period. **From orbit only
the provinces' divisions draw them, zoomed in past their line (see [The provinces come in with the
zoom](zoomed-out-view.md#the-provinces-come-in-with-the-zoom)); out of orbit they always are**: the over pass shows past
the line while a flag is painted, the under pass while the tiles show, past the line or out of orbit. So the country
lines are drawn as they were at every zoom.
They are a second kind of line in `borderLines.ts`: their own cell edges and five coarse sets, of the same tolerances,
culled as the countries' are (`kindOf`), and their own over, under and highlight passes; from orbit the over and the
highlight pass draw the coarse set the countries' draw, of the province lines. `npm run borderLines` writes them beside the countries' for a board with provinces
(`earth-2`, `siege-1`): `static/boards/<board>/provinceLines-<hash>.bin`, named in `BOARDS` (`page.provinceLines`),
in `domain/provinceLinesFile.ts` (format in [`/map/README.md`](../../../map/README.md#format)). It holds:

- **the province lines**, in the countries' format, each run between two provinces of one landmass: 90,701 pieces
  on `earth-2` in 1,863 runs, 8,255 on `siege-1`;
- **the two provinces either side of each piece of the countries' lines**, in their order, as spans of the same pair
  (3,372 on `earth-2`): a province's outline is its province lines and its stretch of the coast and the country
  borders. It names the countries' file and the provinces file it was traced with, and the globe refuses another.

| board | raw | brotli at 5 |
| --- | --- | --- |
| `earth-2` | 315 KB | 135 KB |
| `siege-1` | 61 KB | 43 KB |

**The globe reads them once it is up** (`loadProvinceLines`, the same worker, `provinceOutlineOf`), not with the files it
waits for: the landing view in Countries draws none of them. **A province is lit by its whole outline**: the province
lines' highlight pass draws the pieces with the province on a side, and a second highlight pass over the countries'
cell edges reads their provinces (`PROVINCES` in the line shaders, the `provinces` attribute). That pass draws the cell
edges at every zoom, since a coarse piece of the coast can run past where a province ends; it costs only while a
province is lit. In Countries the highlight is the landmass's, as before.

It stays out of `/map`, unlike the two blobs it is built from: the backend has no
use for it. **Its handover reads the board's tiles**: `createBorderLines(data, scale)` hands the outline over at the zoom
where the board's own tiles take the frame (see [Sized by the board](board-scale.md)). Where a tile is and who owns the ground under it are the game's rules
and are shared; how thick a line is drawn between them is this app's.
