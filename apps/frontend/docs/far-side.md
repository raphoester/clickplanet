# The tiles out of sight are not drawn

**The CPU skips the pieces of the map that cannot show**, so the work on the tiles follows what is on screen, not the
size of the map. `TileField.cull` runs before each frame that is drawn.

- **Chunks.** `chunksOf` (`domain/tileChunks.ts`) cuts the tiles into runs of consecutive ids, each at most 0.1 radians
  (about 640 km) from its first tile and at most 1,024 tiles long. Tile ids follow the lattice's rows, so a chunk is a
  piece of a row. Each chunk has a cap of the sphere that holds all its tiles: a centre and an angle. Each 32
  consecutive chunks make a block, with a cap of its own. The lattice worker builds them with the positions: 12,234
  chunks on `earth-1`, 24,106 on `earth-2` and 3,815 on `siege-1`. A board in canonical order keeps them short: its
  ids run along each of the 30 edges, then each face's inside row by row.
- **In sight.** `showingChunks` skips a block out of sight whole, takes a block wholly in sight whole, and tests each
  chunk of the others. A cap is behind the earth when all of it is past the limb the display vertex shader uses below,
  and off screen when it is past the screen's edge with room for the largest sprite (a blast swells a tile up to 4
  times, a virus's pods reach 1.8 times) and a blast's throw. So the CPU skips only tiles the shader drops or the
  screen does not show, and the frame is the same.
- **Draws.** `drawnRanges` turns the chunks in sight into ranges of tiles: two pieces closer than 8,192 tiles are drawn
  as one (`LEAST_GAP`), and there are at most 16 (`MAX_DRAWS`). A tile drawn for nothing costs the GPU less than one
  more draw call costs the CPU. The ranges are the geometry's groups, and the points have an array of one material, so
  three draws each group.
- **Kept while the view moves a little.** The pieces are found with a quarter of the screen's half-height to spare
  (`CULL_SLACK`), and kept while the camera has turned and moved by less than that, the screen has not grown and the
  sprites have not widened (`ChunkSight` in `chunkSight.ts`). A drag or a flight finds them again every few frames,
  not every frame.
- **Not when the globe fills the screen.** While the screen shows more than half the globe's disc (`CULL_BELOW`), the
  zoom of the landing view and below, every tile is one draw, as before: what is out of sight then is mostly the far
  side, which the vertex shader drops as cheaply, and finding the pieces cost more than it saved.
- **Tested.** `tileChunks.test.ts` pins it on every shipped board, `earth-1` in three's order and `earth-2` and
  `siege-1` in canonical order: every tile in exactly one chunk, every cap holding its
  tiles, every block its chunks, and, from several cameras, no tile the shader would draw in a chunk left out.
  `tileField.test.ts` pins the pieces kept through small moves against every tile in sight.

**While the painted flags cover the globe, only the tiles wearing a ring are drawn.** At `flagPaint` 1, the landing
view and the idle spin, the territory layer paints every pixel and a tile draws nothing of its own but the front's and
the fortify pulse's rings (see [The zoomed-out view](zoomed-out-view.md)). `TileField` counts the ringed tiles of each
chunk and draws only the ringed chunks, wherever they are; with no ring on the map, no tile is drawn. **This is the one
change on screen.** The tiles under the paint drew no colour, but wrote their depth, and the outline drawn after them
is tested against it: where the globe slopes away, a tile's disc, flat toward the camera, stood in front of the line
and cut it. Those pixels of the outline now show. Measured against the same frames before the change, 0.04 to 0.5% of
the pixels of a view from orbit differ, all on the outline; every view where the tiles have the frame is the same to
the pixel, with HD graphics on and off.

**The vertex shader still drops the far side of the pieces it is given.** The earth's own sphere is opaque at radius
0.999, so half of every pass is behind it. The display and border-line vertex shaders drop such a point on one dot
product, before the texture fetches the painted flag costs.

What may not be dropped is what the earth's silhouette does not cover. A point at radius *r*, an angle *θ* past the
limb, projects to a screen radius of *r·cos θ*, so it still shows while *θ < acos(0.999 / r)*: about 0.045 for the
tiles at 1, and `limbOf(lift)` for each outline pass. On top of that comes half the tile's own disc, half the line's own
width, and how far a blast may throw a tile outward (`LIMB` and `THROW`, the same in `tileField.ts` and the shader).
`borderLines.test.ts` pins `limbOf` against the earth's radius.

**The outlines are cut into chunks too** (`outlineChunksOf` in `borderLines.ts`, culled by `cull` before each frame,
after the tiles'). On `earth-2` they drew more vertices than the tiles in sight: 129,431 cell edges, then 2 quads each
in the over pass and 4 in the under pass (one in every pass since the outline is drawn straight, see [The countries'
outlines](country-outlines.md)), and before this every frame drew them all.

- **Chunks.** The same `chunksOf`, over each set's pieces in their order: the cell edges, and each coarse set the over
  pass draws from orbit (see [The countries' outlines](country-outlines.md)). A chunk's cap holds both ends of every
  piece. The cell edges are 1,032 chunks on `earth-1`, 1,449 on `earth-2` and 709 on `siege-1`, built in the border
  lines' worker as the board's files land.
- **In sight.** The same `showingChunks`, with the limb of the outermost pass (`limbOf(OVER)`), the widest line (the
  highlight's) and the quad's corners, which reach √2 half-widths past a piece's ends. A coarse piece's quad is wider by
  its slant, at most twice (`LEAST_SLANT`), and its set is culled that much wider. So no pass loses a piece its vertex
  shader would draw. The cell edges are culled every frame, and the coarse set the over pass draws as well.
- **Draws.** WebGL2 draws no range of an instanced geometry's instances without an extension it does not promise. So a
  pass is up to 8 draws (`MAX_DRAWS`), each a geometry whose attributes start at its range's first piece in the one
  buffer every draw shares (an `InterleavedBufferAttribute` at an offset), with `instanceCount` the range's length.
  Ranges closer than 512 pieces are drawn as one (`LEAST_GAP`). The passes on one set draw the same geometries. Each
  draw keeps the pieces' order and the passes keep theirs (the meshes are made in drawing order, and three draws the
  transparent ones of one `renderOrder` by id), so the cell edges blend as they did, to the pixel.
- **Kept while the view moves a little** (`ChunkSight` in `chunkSight.ts`, which the tiles use as well): the same
  quarter of the screen's half-height, but at most 0.05 of the globe's radius (`MOST_SLACK`). Zoomed out, a quarter of
  the screen would keep most of the far side.
- **Also when the globe fills the screen.** Unlike the tiles, the outline is culled at the landing view: its far side
  is about half of it, and the vertex shader runs a piece's four corners before it drops them.
- **Tested** in `borderLines.test.ts` on every shipped board: every cap holds both ends of its pieces. From several
  cameras, each turned by 98% of the slack after the cull, no pass has a piece in sight of the set it draws left out. The test copies the vertex shader's rule and reads each pass's own uniforms. A line just off the screen's edge
  that reaches onto it is kept until the camera has panned by the slack.

**Measured** in headless Chrome with the CPU slowed 4 times (see [Slow devices](../../../docs/boards.md#slow-devices)):
the landing view draws no tile, and its GPU time on a test map of 1M tiles went from 8.8 to 1.9 ms; at zoom 10 a few
percent of the map is drawn. The CPU time per frame is the same as before: the pieces are found again only every few
frames of a drag.
