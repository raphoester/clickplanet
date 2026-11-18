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

**The line is drawn straight along the cell edges**: each cell edge is one piece, a quad from corner to corner, in
every pass. The outline turns 60° at every corner, as the cells do. Until 2026-11-18 it was drawn as a quadratic
B-spline through the middle of each cell edge, four pieces to an edge pushed in and two from orbit. The owner dropped it
that day: the tiles are drawn as straight hexagons, so a curve around them reads worse than their own edges, and on
`earth-2` the spline's pieces were most of the vertices a frame drew. Straight, the outline is a quarter of the pieces
pushed in and half of them in orbit, and the code has no smoothing to keep (measured in
[The outlines drawn straight](../../../docs/boards.md#the-outlines-drawn-straight-measured)).

**A straight cell edge is the corridor's own edge.** The corridor between two tiles of different countries is narrowest
at the middle of the cell edge between them, half a spacing from each tile's centre, and the line runs through it. At a
corner, where three cells meet, the corridor is wider (a spacing over √3 from each centre), but a piece's quad runs half
a width past its end, square, straight at the third tile there. `borderLines.test.ts` measures both on a lone tile's
cell at every zoom the tiles are drawn alone: the line's solid width clear of the tile's own disc at the middle of each
side, and each piece's end clear of the neighbours' discs at the corners. The corner is the tighter of the two: it
holds `WIDTH` under about 2.2px, where the middle alone would allow 2.5px.

**Every pass draws the same pieces.** The over, the under, the highlight and the mission pass are four materials over
one set of pieces and the same draws. A second kind of line, with its own colour, would be its own pieces and passes.

**Only the pieces in sight are drawn**, as with the tiles (see [The tiles out of sight are not drawn](far-side.md)).
Before each frame, `cull` finds the chunks of the outline's pieces that are not behind the earth or off screen, and
each pass draws them as up to 8 ranges of its pieces, in their order. Every pass shares one cull, made for the
outermost lift and the widest line, so none of them loses a piece it would draw, and the frame is the same to the pixel.

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
only the land points and a coast needs the sea around it. A cell corner is the circumcentre of a lattice triangle —
the normal of the plane through its three vertices — which makes the cells a true
Voronoi diagram of the tiles and the corners meet exactly, so there are no seams
to cover up at the joins. The edges are then chained into runs, which is what
keeps the file to one corner per edge rather than two, and **each corner after a run's first is written as its
difference from the one before** (version 3 of the file, 2026-11-18, `domain/borderLinesFile.ts`; the format is in
[`/map/README.md`](../../../map/README.md#format)): a cell edge is a few dozen units of the int16 corners, a byte as a
zigzag varint. Version 2 wrote each corner as three plain int16s, which brotli packed poorly, and the file was most of
what `earth-2`'s globe waited for. Size raw, and packed with brotli at 5:

| board | edges | version 2 | version 3 |
| --- | --- | --- | --- |
| `earth-1` | 56,790 | 362 KB, 324 KB packed | 222 KB, 122 KB packed |
| `earth-2` | 129,431 | 810 KB, 767 KB packed | 422 KB, 220 KB packed |
| `siege-1` | 14,419 | 99 KB, 90 KB packed | 87 KB, 63 KB packed |

`borderLinesFile.test.ts` pins that version 3 reads back each shipped board's lines to the bit, as version 2 held
them (a hash of the runs, the sides and the corners), and writes them back to the same bytes.
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

It stays out of `/map`, unlike the two blobs it is built from: the backend has no
use for it. **Its handover reads the board's tiles**: `createBorderLines(data, scale)` hands the outline over at the zoom
where the board's own tiles take the frame (see [Sized by the board](board-scale.md)). Where a tile is and who owns the ground under it are the game's rules
and are shared; how thick a line is drawn between them is this app's.
