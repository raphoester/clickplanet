# Map geography

**Which tiles touch which.** It exists so that bonuses depending on the shape of the map can be
decided by the server: "a click spreads to the 6 adjacent tiles" cannot be computed on the client
without the client naming the tiles it gets, which is the whole of the cheat.

**It is `clicks`, not kernel**: the kernel is for what would read the same in any other program,
and this is the tile game's own map. It is then split across the two layers, on the line of *what
survives a change of input format*:

- **`clicks.Geography`** (`internal/clicks/geography.go`) — the model. `NewGeography(positions, edges)`
  builds it, and `Neighbours`, `Position`, `Within`, `Nearest` and `Spacing` read it. It knows nothing about icosahedra,
  boards or file formats; it takes an edge list in any order, with repeats, and enforces what has
  to be true of a map: tiles in range, nothing touching itself, no tile above `MaxDegree`, and
  **no asymmetric edge** — a tile that spreads onto a neighbour which would not spread back is a
  one-way street on the map.
- **`clicks/embedded_board`** — the registry of boards. For each board it reads the manifest and runs, rebuilds the
  positions, and takes the edges from the lattice; it reads the board's borders, capitals and bridges too, and makes
  them its `clicks.Ground`. The manifest, the
  runs and the lattice are knowledge of the shipped files, not of the game: **ship an edge list one day and the
  lattice walk goes while `clicks.Geography` does not change at all.**

**It loads every board the process carries** (`Registry.Carried`: each directory's name and its manifest's
`positionsHash` in 8 hex, `3fb28bdf` for `earth-1`), each the first time something needs it: the boot matches each with
planet's row of it in `planet.boards` and refuses a map whose era plays a board it does not carry (see
[Boards and eras](boards-and-eras.md)).

The boards are **`generated/map/boards`**, beside `generated/proto` and for the same reason: both are
this app's committed copy of something the root owns, written by a `make` target and never edited
by hand (`make map` copies [`/map/boards`](../../../map/README.md)). The vendored VPN ranges sit next to their reader
instead (`shared/cpipblock/cpdata`) because they come from a third party, not from `/`. The directory is `map` to
mirror its source; the package is `mapdata`, since `map` is a keyword, and `cpboards` reads it for every module. It holds
one board today, `earth-1`; PR 4 adds `earth-2` and `siege-1` beside it (see [Boards](../../../docs/boards.md)). A test
binary carries a fixture too (`cpboards.Fixture`, `fixture-1`: a coarse earth at `cols` 31, 2,998 tiles), embedded
under the `testing` tag from `internal/shared/cpboards/testdata`, which the page's `npm run map:fixture` writes.

That line also splits the checks. The domain enforces what is true of any map; the adapter enforces
what is true of *this* board — each file's tile count against its manifest's, a lattice it can walk (no warp, no
rotation, three's order), runs inside that lattice, and positions that hash to the manifest's `positionsHash`.

And it splits the tests. `domain` is tested against a hand-built patch of honeycomb — fast, no board,
and it says what `Geography` does rather than what the shipped board happens to contain.
`embedded_board` is tested against the real board, and holds every number below.

**The tile grid is a regular honeycomb, not an arbitrary numbering.** Tile ids look like noise but
they are the land points of `THREE.IcosahedronGeometry(1, 300)`, numbered in the order three walks them — a
geodesic sphere, where **every interior point has exactly 6 neighbours** and the 12 icosahedron
corners have 5. The lattice and the evidence for it are in [`/map/README.md`](../../../map/README.md).

## Where every tile is

`lattice.go` walks the lattice as the page's `lattice.ts` does: each face's triangles row by row, each point the first
time three's walk reaches it, by its canonical id. A point is a tile when its canonical id is in one of the board's
runs; a tile's id is its rank among the tiles in that walk, plus one.

**Each position is three's, to the bit.** The point lerps across the face's **un-normalised** corners
(`a + (c − a)·t`, then along the row) and is normalised after — THREE lerps the flat triangle and normalises
afterwards, so normalising the corners first moves every point. Only `+ − × ÷` and `math.Sqrt`, each rounded on its
own: **the lerp's product is converted to `float64` before the add**, because the Go spec lets the compiler fuse a
multiply into an add, and arm64 does — fused, the rebuild hashes to `5e72d311` instead of the manifest's `3fb28bdf`.
The boot refuses positions whose hash differs from the manifest's, the hash the page and the backoffice check theirs
against; `TestEveryPositionIsThePagesToTheBit` pins it.

## How adjacency is computed

Each icosahedron face is a triangular integer lattice: `(i, j)` with `i + j ≤ 301`. The six neighbours of a point are
the ±1 exchanges of its barycentric coordinates, `(i, j±1)`, `(i±1, j)` and `(i±1, j∓1)`, those on the face. Walking
all 20 faces, keeping each move between two tiles by their canonical ids, and taking the union gives the adjacency
exactly.

**Cross-face edges and the 12 corners need no special case.** A point on a shared edge is reached
from both faces, contributing 4 neighbours each with 2 in common; a corner is reached from all five
faces that meet there, 2 each with each shared once. 6 and 5, which is what the lattice says.

**There is no tolerance in it at all.** Until 2026-11-17 the backend shipped positions only and found which tile
each lattice point was by matching positions within an f32 epsilon; the canonical id makes that a lookup.

**Do not replace this with a radius-based nearest-neighbour search.** It is the obvious approach and
it fails quietly: because THREE subdivides the flat triangle, spacing varies ~25% between face
middles and corners (nearest-neighbour distances run 0.00309 to 0.00440), so no single radius works
anywhere. Measured, a tuned radius gave **2,137 tiles degrees of 7, 8 and even 10**.

## The off-by-one

**Wire tile id = rank among the board's tiles + 1.** Positions are 0-indexed; ids on the wire are
1-based, which is what `clicks.Board`, `inmemory_tile_storage`'s unused slot 0 and the
frontend's `integerToColor(i + 1)` all agree on. Getting it wrong shifts every neighbourhood by one
tile, **symmetrically, with a degree histogram that still looks right** —
`TestTileIDsAreOneBasedOverTheRuns` is what catches it.

## Checked at boot, not just in the tests

`embedded_board.Registry.Ground` builds the board the map in play plays as the planet module boots, and **fails the
boot** on a file that holds another tile count than the manifest, on runs outside the lattice, on positions that do not
hash to the manifest's, on a degree above 6, on an asymmetric edge, or on a capital off its own ground; and the
registry, before, on a lattice it cannot walk. A board an arena plays is built as the arena opens, and refused there
the same way. The tests cannot see the board a container was actually built with, and that is the thing that drifts;
regenerating a board renumbers every tile, so a process quietly disagreeing with the frontend about what tile 42 is has
no repair after the fact. In practice the failure is unreachable in production — the boards are embedded, so a bad one
fails `make test` long before a deploy — which is what makes it cheap.

It costs about 0.1 s of boot (the position matching it replaced took about 0.26 s) and about 12 MB resident. The boot
log carries the whole result in one line:

```
board loaded board=earth-1 asset=board-725d8920.json tiles=260503 edges=761422
  degrees="[250 527 1154 4072 6368 6471 241661]" borders=borders-e6851c0e.bin bridges=744 capitals=235 took=95ms
```

Those numbers are pinned by the tests. 761,422 undirected edges, average degree 5.846, and the
degree histogram reads: 250 tiles with no neighbours, then 527, 1154, 4072, 6368, 6471, and 241,661
inland tiles with the full 6. Walking adjacency alone finds **505 landmasses**, the largest four
being 142,792 (Afro-Eurasia), 67,352 (the Americas), 22,871 (Antarctica) and 12,402 (Australia).

## The API, and what is not in it yet

The reads of the tiles are on `clicks.Geography`, so a rule that reads them needs no port and no adapter import.

`Neighbours(id)` hands back a window into the map's own CSR table — read it, never write it, which
is what keeps it allocation-free on a click path. **It returns nothing for the 250 single-tile
islands**, so a consumer has to have an answer for an empty neighbour set; whether the game paints
nothing or refunds the bonus there is a rule, not a fact about the map.

`clicks.Borders` is the other half of the geography: which country's ground a tile sits on, from the board's
`borders-<hash>.bin`, the table the frontend's `npm run map:generate` writes to `/map`.
The operator tools read it — see [Manual bans](admin.md#manual-bans-findplayers-topplayers-banplayer-unbanplayer-revertplayer-inspectplayer) —
and so does a bomb, for the ground it landed on — see [Announcements](chat.md#announcements).

`clicks.Bridges` is the far end of each bridge on a tile, from the board's `bridges-<hash>.json`: 744 crossings of
the sea, each between two coast tiles (see [`/map/README.md`](../../../map/README.md#capitals-and-bridges)).
`clicks.Capitals` is each country's capital tile, from the board's `capitals-<hash>.json`: the tile of its
ground nearest its capital city. The boot refuses a capital off its own ground, and planet keeps only the capitals
of the flags a player can play for (235: `hm`, `nc`, `pm` and `sh` have ground and no flag). Both are read only in an
arena — see [The reach](reach.md).

`Neighbours` is what the spread bonus reads — see [What a spread does to a click](spread.md).
`Within`, `Position`, `Nearest` and `Spacing` are what the bomb reads — see [What a bomb does](bomb.md).
`Within`, `Nearest` and `Spacing` are straight scans (a few ms over 260,503 tiles): `Spacing` runs once at
boot, `Within` and `Nearest` once per bomb.

## What the board no longer gets wrong

The runs and the borders come from one Natural Earth query, so **every tile is in a country** and the
antimeridian row carries what it should. It used to be 2,191 tiles in no country and a 36-tile
deficit at the dateline, and both left tiles with fewer than 6 neighbours — which is also what a
coastline does, so there was no telling them apart from the geometry. The 250 tiles with no
neighbours at all are genuine single-tile islands, 26 of them the seats of island countries; anything reading adjacency still has to have an
answer for an empty neighbour set. See [`/map/README.md`](../../../map/README.md).
