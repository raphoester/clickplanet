# What a tile costs on the GPU

Every tile hands the GPU a few numbers that each frame reads: where it is, whose
flag it wears, which landmass it is on, and whether it is hovered, shielded or
pulsing. There are 262,119 tiles, so one byte per tile is a quarter of a
megabyte, and the plain layout spent 56 of them.

**`TileField` has two layouts, and the HD graphics setting picks one**
(`compact` in `Graphics`, `?gfx=compact` on its own — see [HD
graphics](hd-graphics.md)):

| | plain, the default | compact, with HD graphics |
|---|---|---|
| position | `position` f32×3 | the same |
| flag | `regionVector` f32×4, the flag's atlas rectangle | `ids.x` u16, a flag number |
| landmass | `landmassIndex` f32 | `ids.y` u16 |
| hover, pulse, shields | `hover`, `pulse`, `shield` f32 each | `marks.x`, `.y`, `.z` u8 each, `.w` unused |
| picking colour | `color` f32×3 | none |
| per tile | 56 bytes, 14.7 MB | 20 bytes, 5.2 MB |

**A flag is a number, not a rectangle.** `ids.x` is the flag's place in
`atlas.json` plus one, and 0 is no owner. The vertex shader reads the rectangle
from `flagRectangles`, a float texture of one row per flag, and passes it on as
`vRegionVector` exactly as before, so the fragment shader is the same for both
layouts. One read per tile in the vertex shader, not one per pixel in the
fragment shader. A u8 cannot hold it: the atlas has 256 flags, and 0 is taken.

**Small integers, read as exact floats.** `ids` is a `Uint16Array` and `marks` a
`Uint8Array`, with `normalized` off, and the shader declares them as `vec2` and
`vec4`: the GPU turns 76 into 76.0, so every test the shader makes on them is
unchanged. `#define`s at the top of the display vertex shader give them the plain
layout's names, and the rest of the shader does not know which layout it reads.

**Every buffer's stride is a multiple of 4 bytes.** ANGLE draws on Metal for
Chrome and Safari on Macs and iPhones, and copies a vertex buffer whose stride is
not into a padded buffer of its own: a lone u8 per tile would then cost more than
the f32 it replaced. So the small values travel as two 4-byte vectors, and one
byte is left unused. `tileField.test.ts` pins the stride.

**The picking colour is made, not uploaded.** Both point clouds are drawn in tile
order with no index, so `gl_VertexID` is the tile's index. The picker's vertex
shader turns `gl_VertexID + 1` into the three bytes `integerToColor` makes, and
`GpuPicker.pick` reads them back as before.

**The earth's sphere is built once, for both layouts.** `innerSphere()` is called
once in `createGlobe` and shared by the earth and the picker's occluder. It was
built twice, once for each, and without an index; `mergeVertices` makes it 26,165
vertices with an index instead of 156,060. The earth's mesh is in the scene, so
`disposeScene` frees it, and `GpuPicker.dispose` leaves it alone.

**Measured** on the fake backend, 1024×768, Chrome on an M2 Max: the sum of every
attribute's and index's `byteLength` across the three scenes, and
`renderer.info.memory` after load and one hover.

| | before | plain | compact |
|---|---|---|---|
| attributes | 29.9 MB | 21.1 MB | 11.7 MB |
| geometries / textures | 8 / 5 | 7 / 5 | 7 / 6 |

The page keeps the same arrays in its own memory, so it holds as much less.

**Why only with HD graphics.** The white, flickering globe on Windows Chrome with
an Intel GPU (see [HD graphics](hd-graphics.md)) could not be reproduced
anywhere else, and this layout reads small integer buffers that nothing else in
the game uses. A player who sees a broken globe turns HD graphics off and gets the
plain layout back. **Once it is seen working on such a machine, make it the
default** and delete the plain layout: `plainLayout`, the `#else` halves of the
display and picker vertex shaders, `pickingColors`, and the `compact` part of
`Graphics`. If one driver mishandles one integer type, that one value can go back
to an f32 attribute of its own.
