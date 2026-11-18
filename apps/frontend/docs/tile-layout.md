# What a tile costs on the GPU

Every tile hands the GPU a few numbers that each frame reads: where it is, whose
flag it wears, which landmass it is on, and whether it is hovered, shielded or
pulsing. There are 260,503 tiles, so one byte per tile is a quarter of a
megabyte; at the 1M tiles of `earth-2` (see [Boards](../../../docs/boards.md)) it is a megabyte.

**`TileField` has two layouts, and the HD graphics setting picks one**
(`compact` in `Graphics`, `?gfx=compact` on its own — see [HD
graphics](hd-graphics.md)):

| | plain, with HD graphics off | compact, with HD graphics, the default |
|---|---|---|
| position | `position` f32×3 | the same |
| flag | `flagRegion` f32×2, the flag's atlas rectangle | `ids.x` u16, a flag number |
| province | `tileMarks.x` f32 | `ids.y` u16 |
| hover, pulse, shields | `tileMarks.y`, `.z`, `.w` f32 | `marks.x`, `.y`, `.z` u8 each, `.w` unused |
| per tile | 36 bytes, 9.4 MB | 20 bytes, 5.2 MB |

**A flag is a number, not a rectangle, in the compact layout.** `ids.x` is the flag's place in
`atlas.json` plus one, and 0 is no owner. The vertex shader reads the rectangle
from `flagRectangles`, a float texture of one row per flag, and passes it on as
`vRegionVector`, so the fragment shader is the same for both
layouts. One read per tile in the vertex shader, not one per pixel in the
fragment shader. A u8 cannot hold it: the atlas has 256 flags, and 0 is taken.

**The plain layout is 32-bit floats alone, and reads no texture in its vertex shader.** The rectangle is two floats,
`x + y × 2048` and `width + height × 2048`: every atlas coordinate is a whole pixel under 2,048, so the sum is under
2^24 and exact in a float, and the shader takes it apart with `mod` and `floor`, exact for whole numbers and a power of
two. `tileField.test.ts` pins both. It was 56 bytes: four floats for the rectangle, one attribute of its own for each
mark, and three floats of picking colour, which went with the GPU picker (see [The tile under the pointer](picking.md)).

**A tile carries its province, not its landmass** (`TileField.setProvinces`; on a board with no provinces file the
province is the landmass). A board's provinces are cut landmass by landmass, in order, so a landmass's provinces are one
span of ids (`ProvinceSpans`, `domain/provinces.ts`, which refuses a file where they are not): the shaders light a
landmass, and run a fortify's wave over it, as a span (`highlight`, `fortifyProvinces[i]`, first and last), and the plain
layout's vertex shader still reads no texture. Only the territory layer turns a province into its landmass, for the
Countries frames: `provinceLandmass`, a u16 per province, 1,024 a row.

**The territory layer reads the same flag numbers from a texture**, `TileField.tileData`: each tile's flag and
province, two u16, at the texel `cellsOf` gives it (1 MB on `earth-1`, 4 MB on `earth-2`), in both layouts. See
[The zoomed-out view](zoomed-out-view.md). **It is 1,024 texels a row** (`TILE_COLUMNS`, and the territory shader's):
at 512, `earth-2`'s 1,004,682 tiles would fill 1,963 of the 2,048 rows WebGL2 guarantees, and a board a little larger
none; at 1,024 they fill 982. The other textures sized from the board: the terrain, a byte a lattice point, 2,048 a row
(`TERRAIN_COLUMNS`: `earth-2`'s 3,481,002 points fill 1,700 rows, which 1,024 a row could not hold); `tileCells`, a pair
of words per 32 points, 256 a row (425 rows on `earth-2`); the landmasses' data, a row a landmass (1,175 on `earth-2`,
of 2,048); the provinces' data, a row a province (2,098 on `earth-2`, past the 2,048 WebGL2 guarantees and under the 4,096
`BorderField` takes); the aimed outlines, 1,024 landmasses a row. Nothing else on the GPU is sized by the tile count but the
vertex buffers, which have no such limit.

**A change sends only what changed.** `TileField.uploadTiles` sends the rows of `tileData` that hold a changed texel,
a run of rows as one, at most 16; an attribute gets the changed tiles' values, two tiles apart or less as one range, at
most 64 (`domain/spans.ts`). Both used to send everything between the lowest and the highest change: two clicks at
the two ends of the map sent the whole texture (4 MB at 1M tiles) and most of each buffer.

**Small integers, read as exact floats.** `ids` is a `Uint16Array` and `marks` a
`Uint8Array`, with `normalized` off, and the shader declares them as `vec2` and
`vec4`: the GPU turns 76 into 76.0, so every test the shader makes on them is
unchanged. `#define`s at the top of the display vertex shader give both layouts
the same names, and the rest of the shader does not know which layout it reads.

**Every buffer's stride is a multiple of 4 bytes.** ANGLE draws on Metal for
Chrome and Safari on Macs and iPhones, and copies a vertex buffer whose stride is
not into a padded buffer of its own: a lone u8 per tile would then cost more than
the f32 it replaced. So the small values travel as two 4-byte vectors, and one
byte is left unused. `tileField.test.ts` pins the stride.

**The earth's sphere is built once, for both layouts.** `innerSphere()` is called
once in `createGlobe` and shared by the earth and the territory layer.
`mergeVertices` makes it 26,165 vertices with an index instead of 156,060. The earth's mesh is in the scene, so
`disposeScene` frees it.

**Measured** in headless Chrome on an M2 Max: every buffer and texture the page holds on the GPU, summed from the WebGL
calls that make them (see [Slow devices](../../../docs/boards.md#slow-devices)).

| buffers | `earth-1`, compact | `earth-1`, plain | 1M tiles, compact | 1M tiles, plain |
|---|---|---|---|---|
| before | 11.4 MB | 20.8 MB | 26.2 MB | 62.2 MB |
| after | 11.4 MB | 15.5 MB | 26.2 MB | 42.2 MB |

The page keeps the same arrays in its own memory, so it holds as much less.

**Why only with HD graphics.** The white, flickering globe on Windows Chrome with
an Intel GPU (see [HD graphics](hd-graphics.md)) could not be reproduced
anywhere else, and the compact layout reads small integer buffers that nothing else in the game uses. A player who
sees a broken globe turns HD graphics off and gets the plain layout back. **Once it is seen working on such a machine,
delete the plain layout**: `plainLayout`, the `#else` half of the display vertex shader, and the `compact` part of
`Graphics`. If one driver mishandles one integer type, that one value can go back to an f32 attribute of its own.
