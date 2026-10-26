# Map geography

**Which tiles touch which.** It exists so that bonuses depending on the shape of the map can be
decided by the server: "a click spreads to the 6 adjacent tiles" cannot be computed on the client
without the client naming the tiles it gets, which is the whole of the cheat.

**It is `clicks`, not kernel**: the kernel is for what would read the same in any other program,
and this is the tile game's own map. It is then split across the two layers, on the line of *what
survives a change of input format*:

- **`clicks.Geography`** (`internal/clicks/geography.go`) — the model. `NewGeography(positions, edges)`
  builds it, and `Neighbours`, `Disc` and `Position` read it. It knows nothing about icosahedra,
  blobs or file formats; it takes an edge list in any order, with repeats, and enforces what has
  to be true of a map: tiles in range, nothing touching itself, no tile above `MaxDegree`, and
  **no asymmetric edge** — a tile that spreads onto a neighbour which would not spread back is a
  one-way street on the map.
- **`clicks/embedded_geodesic_map`** — the recovery. Everything in it exists because the
  adjacency is *not* shipped: the positions are, and the edges have to be worked back out of them.
  The blob decode, the icosahedron lattice and the position index are all knowledge of the shipped
  artifact, not of the game. **Ship a precomputed edge list one day and this package goes while
  `clicks.Geography` does not change at all.**

The blob itself is **`generated/map`**, beside `generated/proto` and for the same reason: both are
this app's committed copy of something the root owns, written by a `make` target and never edited
by hand. The vendored VPN ranges sit next to their reader instead (`shared/cpipblock/cpdata`) because
they come from a third party, not from `/`. The directory is `map` to mirror its source; the
package is `mapdata`, since `map` is a keyword.

That line also splits the checks. The domain enforces what is true of any map; the adapter enforces
what is true of *this* blob — the tile count against `gameMap.maxIndex`, and that every tile lands
on a detail-300 lattice vertex.

And it splits the tests. `domain` is tested against a hand-built patch of honeycomb — fast, no 5 MB
asset, and it says what `Geography` does rather than what the shipped blob happens to contain.
`embedded_geodesic_map` is tested against the real blob, and holds every number below.

**The tile grid is a regular honeycomb, not an arbitrary numbering.** Tile ids look like noise but
they are the land vertices of `THREE.IcosahedronGeometry(1, 300)`, deduplicated by position — a
geodesic sphere, where **every interior vertex has exactly 6 neighbours** and the 12 icosahedron
corners have 5. The parameters and the evidence for them are in [`/map/README.md`](../../../map/README.md).

## How adjacency is computed

Each icosahedron face is a triangular integer lattice: `(p, q, r)` with `p + q + r = 301`, all
non-negative, sitting at `normalize(p*A + q*B + r*C)` for the face's **un-normalised** corners —
THREE lerps across the flat triangle and normalises afterwards, so normalising the corners first
moves every vertex. The six neighbours are the ±1 exchanges between any two coordinates. Walking
all 20 faces, resolving each lattice vertex back to a tile by position, and taking the union gives
the adjacency exactly.

**Cross-face edges and the 12 corners need no special case.** A vertex on a shared edge is reached
from both faces, contributing 4 neighbours each with 2 in common; a corner is reached from all five
faces that meet there, 2 each with each shared once. 6 and 5, which is what the lattice says.

**Do not replace this with a radius-based nearest-neighbour search.** It is the obvious approach and
it fails quietly: because THREE subdivides the flat triangle, spacing varies ~25% between face
middles and corners (nearest-neighbour distances run 0.00309 to 0.00440), so no single radius works
anywhere. Measured, a tuned radius gave **2,137 tiles degrees of 7, 8 and even 10**. The lattice
walk has no threshold in it at all — the only tolerance is `matchEpsilon`, and that is f32 rounding
(~1e-7 against a 3.09e-3 gap between the closest two tiles), not a search radius.

## The off-by-one

**Wire tile id = blob array index + 1.** The blob is 0-indexed by position; ids on the wire are
1-based, which is what `clicks.Board`, `inmemory_tile_storage`'s unused slot 0 and the
frontend's `integerToColor(i + 1)` all agree on. Getting it wrong shifts every neighbourhood by one
tile, **symmetrically, with a degree histogram that still looks right** —
`TestTileIDsAreOneBasedOverTheBlob` is what catches it.

## Checked at boot, not just in the tests

`embedded_geodesic_map.Loader.LoadGeography` is the first thing the planet module loads and **fails the boot** on a blob whose
tile count is not `gameMap.maxIndex`, on any tile that does not sit on the detail-300 lattice, on a
degree above 6, or on an asymmetric edge. The tests cannot see the blob a container was actually
built with, and that is the thing that drifts; regenerating the coordinates renumbers every tile, so
a process quietly disagreeing with the frontend about what tile 42 is has no repair after the fact.
In practice the failure is unreachable in production — the blob is embedded, so a bad one fails
`make test` long before a deploy — which is what makes it cheap.

It costs ~0.3s of boot and about 12 MB resident. The boot log carries the whole result in one line:

```
map geography loaded asset=coordinates-9998a414.bin tiles=262119 edges=768288
  degrees="[225 516 1111 3796 5664 5048 245759]" took=263ms
```

Those numbers are pinned by the tests. 768,288 undirected edges, average degree 5.862, and the
degree histogram reads: 225 tiles with no neighbours, then 516, 1111, 3796, 5664, 5048, and 245,759
inland tiles with the full 6. Walking adjacency alone finds **477 landmasses**, the largest four
being 143,574 (Afro-Eurasia), 68,214 (the Americas), 22,871 (Antarctica) and 12,405 (Australia).

## The API, and what is not in it yet

All three are on `clicks.Geography`, so a rule that reads them needs no port and no adapter import.

`Neighbours(id)` hands back a window into the map's own CSR table — read it, never write it, which
is what keeps it allocation-free on a click path. **It returns nothing for the 186 single-tile
islands**, so a consumer has to have an answer for an empty neighbour set; whether the game paints
nothing or refunds the bonus there is a rule, not a fact about the map.

`Disc(id, radius)` is a breadth-first walk over that, returning `1 + 3r(r+1)` tiles inland and less
wherever the land runs out. **Discs are computed, never stored**: a radius-3 table would be ~38 MB
to save ~50µs, behind a 1 click/sec/IP throttle. The search scratch is a generation-stamped array
from a pool, one per concurrent caller, so nothing is cleared per call and two clicks cannot stamp
the same array.

`clicks.Borders` is the other half of the geography: which country's ground a tile sits on, from
`generated/map/borders-<hash>.bin`, the table the frontend's `npm run map:generate` writes to `/map`.
The operator tools read it — see [Manual bans](admin.md#manual-bans-findplayers-topplayers-banplayer-unbanplayer-revertplayer-inspectplayer) —
and so does a bomb, for the ground it landed on — see [Announcements](chat.md#announcements).

`Neighbours` is what the spread bonus reads — see [What a spread does to a click](spread.md).
`Within`, `Position`, `Nearest` and `Spacing` are what the bomb reads — see [What a bomb does](bomb.md).
`Within`, `Nearest` and `Spacing` are straight scans (a few ms over 262,119 tiles): `Spacing` runs once at
boot, `Within` and `Nearest` once per bomb. `Disc` is back behind the `testing` tag — see [Testing](testing.md).

## What the blob no longer gets wrong

Both blobs come from one Natural Earth query now, so **every tile is in a country** and the
antimeridian row carries what it should. It used to be 2,191 tiles in no country and a 36-tile
deficit at the dateline, and both left tiles with fewer than 6 neighbours — which is also what a
coastline does, so there was no telling them apart from the geometry. The 225 tiles with no
neighbours at all are genuine single-tile islands; anything reading adjacency still has to have an
answer for an empty neighbour set. See [`/map/README.md`](../../../map/README.md).
