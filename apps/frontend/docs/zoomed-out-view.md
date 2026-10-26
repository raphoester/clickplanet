# The zoomed-out view

Zoomed out, a tile is about a pixel and a half sampling a 200px flag out of one
shared 2600×2330 atlas, so the GPU picks a mip level that is the whole atlas
averaged and every country comes out the same mud. Turning mipmaps off swaps mud
for sparkle. Neither end works, so from far enough away the globe is drawn from
something coarser than a tile.

Every tile resolves **offline** to a *landmass*: a country's tiles split into the
separate pieces of land they actually form (Natural Earth 1:50m, 205 countries →
658 landmasses). Neither borders nor tiles move, so `npm run map:generate` writes
the whole table once and nothing recomputes it at runtime. Two tiles are the same
piece when they **touch on the lattice** and carry the same country — not when
they are within a tuned radius, which reached past the neighbours in places and
joined two islands across a strait into one flag painted over the water between
them. A landmass rather than a
country because a flag belongs to a piece of ground — one frame spanning mainland
France, Corsica, Guiana and Réunion would stretch the tricolour across half the
planet and paint nothing recognisable anywhere.

Each landmass flies the flag of whoever holds most of it, painted onto the sphere
with distance measured **along the surface**, so it bends with the globe and is
cropped by its own coastline. `BorderField` paints each landmass from the
server's row for it (`setRows`, with `Holdings.leaderOf`: see [The counts are the
server's](counts.md)) into one row per landmass of a `DataTexture` the vertex
shader reads.

**The painted flag is read under the pixel, not per tile.** It still reaches the
screen through the discs — that is what the widening is for — but what each
fragment shows is the flag at the point of ground beneath it, so overlapping
discs agree and the landmass comes out as one image at the resolution of the
screen. The vertex shader hands the fragment shader the frame rather than a
colour: where the tile's own centre falls in the flag, and how far that slides
under one screen pixel across and up. A pixel is a step on the *screen*, so the
ground step behind it is the one whose projection is a pixel — the tangent part
of the camera's axis over how much of itself the projection keeps, floored near
the limb where that divisor runs to zero. Those same two vectors are the
footprint the atlas is sampled over (`textureGrad`), which is the only reason a
mip level can be chosen at all: the frame is flat across a sprite, so without
them the driver takes the top one and point-samples a 200px flag into a few
dozen pixels. The atlas is sampled with 4× anisotropic filtering: near the limb
a screen pixel covers a long thin strip of ground, and without it the flag is
blurred to the strip's length across its width too.

**The flags are 200px wide because of this view, and not wider because of
memory.** A painted flag covers its whole landmass, and the handover to tiles
only starts once a tile is 5px wide: over Brazil that is a flag 500 to 3,000
pixels wide, drawn from a sprite. At 100px the sprite was magnified five to
thirty times and every device, Brazil's globe first, was mud. At 200px Brazil
is sharp to about zoom 2 and soft past it. 300px was tried: sharp to zoom 3,
for 69 MB on the GPU against 31, and the atlas is not the way past that in any
case, since one may not pass 4096 pixels a side (see [Static
assets](static-assets.md)). A flag much bigger than its sprite is a job for a
texture rendered at runtime from that one flag's SVG, not for the atlas.

One sample per disc was the whole flag's resolution before, and a landmass is
often only a few tiles across at the zoom where its flag is painted — six or
seven samples of Macedonia's sun or Serbia's arms, with neighbouring discs each
landing on a different one. Bands survived it; anything carrying a device came
out as pixel soup on exactly the countries that are too small to zoom past.

The lookup is skipped outright while `flagPaint` is 0, which pays for it: zoomed
in it was four vertex texture fetches and a frame per tile for a colour the
fragment shader mixed straight back out, and the fragment shader now skips the
tile's own atlas fetch at the other end, where the painted flag has the frame to
itself.

**The widest painted flag on screen is drawn from its SVG, not read from the
atlas.** A sprite is 100px wide and a painted flag is as wide as its landmass:
Brazil's is 500 to 3,000px across before the tiles take over, so from the atlas
its motto and its stars were mud at every zoom. One slot holds one flag: a
1536×1024 canvas (`flagSlotTexture.ts`; 6 MB on the GPU, 8 with its mipmaps,
allocated the first time a flag is drawn into it) with the flag rasterised from
`static/countries/svg/<code>.svg` at its sprite's aspect, as large as fits, and
the margins repeating its last column and row so a coarse mip level finds the
flag's own colours at its edge. The fragment shader samples it with the same uv
and the same `textureGrad` footprint it uses on the atlas, for the tiles of one
landmass (`slotLandmass`) and no other; `vSlotted` carries the choice from the
vertex shader, which has the landmass index.

`flagSlot.ts` decides who holds it, with no GPU in reach: among the painted
landmasses in view, the one whose flag is widest on screen (2 × halfU ×
`pixelsPerRadian`), once that width passes `MAGNIFIED_FROM` (1.5) times its
sprite's width — below that the atlas is as sharp as the slot would be. The
threshold is measured against the sprite, so the atlas may change size without
this changing. **The choice is made only once the camera has been still for
`SETTLE_SECONDS` (0.15 s) with the idle spin off**; while it turns, the last
choice stands. A pan sweeps across landmasses, and every change of flag is a
draw and a 6 MB upload, so a drag does not thrash it, and the globe spinning at
the resting zoom never takes one: the spin counts as turning even through its
damped start, where it moves the camera by less than the controls notice. A
still globe takes it at any zoom where the flag is painted, the opening one
included: there Russia's flag is four times its sprite on a 600px-high view,
and it is the biggest landmass on screen that the slot is for. The SVG is
fetched the first time its flag is chosen and kept for the session; the atlas
shows until it is drawn, a flag whose SVG fails stays on the atlas and is not
asked for again, and a page where no painted flag ever outgrows its sprite
fetches nothing. The landmass's holder is
read every tick, so a flag that changes hands is redrawn; with `flagPaint` at 0
— zoomed in, or `MapView` `"tiles"` — nothing is painted, so nothing qualifies
and the slot is let go.

**Opacity is the leader's share, and the curve it goes through is not a free
knob.** Zoomed in, that share is already on screen as the fraction of discs
wearing the holder's flag, so the tiles show `share * 0.7` of ink no matter what;
the painted flag shows `share^contrast * 0.94`. Bend the curve and the summary
becomes fainter than the tiles it hands over to, and a country gets *brighter* as
you zoom into it — measured at 5x for Sudan at contrast 3. `borderField.test.ts`
pins this.

**One schedule owns the whole handover** — `coarseHandover` in `pointSize.ts`
drives the flag fading out, the tiles fading in, the disc widening being undone,
and the outline moving from over the tiles to under them. They only work
together: the flag reaches the ground only through the discs, so while it is
painted they must cover the ground (circles on this hex lattice cover it at
1.155x the spacing), and a tile you are about to aim at must not be fattened.
Running them on separate schedules left a band where the flag was painted
through a lattice with holes in it. `pointSize.test.ts` pins that too, and those
tests fail if the two are split again.

**A player can turn the painted flags off** — "Big country flags" in
Settings (`MapView` in the display settings). Players draw pictures with the
tiles, and the painted flag hides them. With it off
(`MapView` `"tiles"`), the handover in `pointSize.ts` is held at 1 at every zoom, so the
globe is drawn as it was before #52: each tile its own flag at its own size,
and the outline always under the tiles. Zoomed out that is the mud described
above, and that is the price of seeing every tile. The switch reaches the
running globe through `Globe.setMapView`, so it never rebuilds it.

`npm run flagFit` decides the rest: a flag that is only bands can be pulled to
the country's own shape and still say what it is, while one carrying a device is
cropped, anchored on the part that names it rather than on its middle. The
result is `static/countries/flagFit.json`.

**Every tile is in a country**, because both blobs come from one Natural Earth
query: a tile exists exactly where `groundOf` answers a code, and that same answer
is what the borders blob records. The 2,191 tiles that used to fall outside every
country are what the globe drew as discs floating on open water with no outline
round them. See [`/map/README.md`](../../../map/README.md).
