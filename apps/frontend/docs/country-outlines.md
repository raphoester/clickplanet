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

**What is drawn is not that polyline but its quadratic B-spline**, four straight
pieces to a cell edge (`SAMPLES`). The lattice is a honeycomb, so the bare
outline turns 60° at every corner and reads as a staircase from a few zooms in;
the spline is the curve through the middle of every cell edge, reaching a quarter
of the way toward each corner without touching it.

That curve is not a compromise on the tile rule — it is the most a curve can be
smoothed and still obey it. The narrowest the corridor between two tiles of
different countries ever gets is at the middle of the cell edge between them, and
any line separating them has to thread that point; the spline goes through it
exactly, and at the corners, where there is half as much room again, it spends a
fraction of what it has. `borderLines.test.ts` measures the drawn line against a
lone tile's own cell and pins its near edge clear of the disc at every zoom the
fine pass is drawn at.

**Each pass carries the outline at the resolution it is looked at.** The over
pass is on screen only while the flag is painted, where a cell edge is at most
six pixels, so two pieces put it within a tenth of a pixel of the fine one and it
draws half the geometry — and it is the pass that is up whenever the whole globe
is. The fine one is only ever drawn pushed in.

**A run ends at every junction** — a corner three countries share, or where a
land border reaches the sea — which is the one corner the smoothing may not round
off. The generator cuts the runs there so the renderer clamps the curve to it,
and the three branches meeting there meet on the point rather than a fraction of
a tile apart.

**It is drawn twice, on either side of the tiles, and `flagPaint` picks.**
Zoomed in the outline sits just inside the tile shell, so the discs' own depth
hides whatever they cover and the line only ever shows in the gaps — which is
the whole width of it. Zoomed out that gap is gone: the discs are widened until
they cover the ground so the painted flag can reach it, and a line underneath
them would be invisible. So the same outline is drawn just outside the shell as
well, fading with the painted flag, and the two hand over on the one schedule that
owns the rest of the handover. **The over pass is thinner and darkens the flag
under it** (`ORBIT_WIDTH`, 0.25px and a pixel of feather, black at `ORBIT_INK`, 70%): a red flag gets a
dark red edge, as in the season-end clips, where the grey line read as a stroke
drawn across the map. Neither pass writes depth, so the stretch where they overlap
— a coast, which has no tiles on the sea side to hide the under pass — comes out a
little darker during the handover.

`npm run borderLines` writes the geometry, from the coordinates blob and the
borders blob and nothing else. It rebuilds the whole `IcosahedronGeometry(1, 300)`
the tiles were cut from, because the coordinates blob holds only the land
vertices and a coast needs the sea around it; every tile has to land on a lattice
vertex or it refuses. A cell corner is the circumcentre of a lattice triangle —
the normal of the plane through its three vertices — which makes the cells a true
Voronoi diagram of the tiles and the corners meet exactly, so there are no seams
to cover up at the joins. The edges are then chained into runs, which is what
keeps the file to one corner per edge rather than two: 53,217 edges in 331 KB.
The smoothing is not baked in — the blob is the outline on the lattice, and how
finely it is rounded off is the renderer's business and four times the size.

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
pass over the coarse pieces, which carry the pair, drawn over the tiles and only while a
territory is lit; the vertex shader drops every piece that does not border it.

It stays out of `/map`, unlike the two blobs it is built from: the backend has no
use for it. Where a tile is and who owns the ground under it are the game's rules
and are shared; how thick a line is drawn between them is this app's.
