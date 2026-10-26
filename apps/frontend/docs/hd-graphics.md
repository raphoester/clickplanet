# HD graphics: the sharper, lit globe, and `?gfx=`

The two sections below — the screen's pixel ratio and the light — shipped on in
#254 and turned the globe **almost white, flickering as it turned**, for players
on Windows Chrome with an Intel GPU (ANGLE on Direct3D 11). Antialiasing off
(#255) did not fix it, and both were reverted (#256). It could not be reproduced
on a Mac (Metal), on SwiftShader, or on a Windows laptop with the same GPU
(Iris Xe, ratio 1.25) in either a dev or a production build. So the cause can
only be found on the screens that have it.

**A player turns them on as "HD graphics" in Settings**, off by default
(`Rendering`, `"plain"` or `"sharp"`, in the display settings): turned on for
everyone, a player who gets the white globe would have to find the switch. **Changing it rebuilds the globe**:
`antialias` is fixed when the WebGL context is made, so `rendering` is in
`useGlobe`'s dependencies and the map loads again.

**The URL still wins over the setting**, part by part, for the bisection below
(`graphicsOf` in `graphics.ts`, read once in `createGlobe`). With `gfx` in the
query only the parts it names are on, so `?gfx=` alone is the plain globe:

| `?gfx=` | Turns on |
|---|---|
| `ratio` | the screen's pixel ratio, capped at 2, instead of 1 |
| `aa` | `antialias` on the context |
| `earth` | the earth's own shader, in the light, with the glint |
| `tiles` | the light on the tiles |
| `halo` | the light on the halo |
| `compact` | the tiles' compact layout: 20 bytes a tile on the GPU, not 56 |
| `light` | `earth`, `tiles` and `halo` |
| `all` | all of the above: everything HD graphics turns on |
| `etc1s`, `uastc` | the earth's texture as a KTX2 file the GPU keeps compressed, in that format; not part of HD graphics or `all`. See [The globe's texture](globe-texture.md#kept-compressed-on-the-gpu-a-trial) |

Words add up (`?gfx=ratio,tiles`), and sit beside the rest of the query
(`?f=fr&gfx=halo`). A word it does not know turns nothing on.

**Off is the code from before #254, not the new code multiplied by zero.** The
light is compiled out with `#ifdef LIT` (three's `defines`, which leaves out a
`false` one), so a driver that miscompiles `light.glsl` never sees it; the plain
earth is three's standard material under the ambient light again; the plain halo
has its own `colour` uniform; the plain tiles are the old attributes, with
`#ifdef COMPACT` around the new ones.

**The compact tile layout came later and is not part of #254.** It sits behind the
same setting for the same reason: it reads integer buffers that nothing else in
the game uses, so a player whose globe breaks with it turns HD graphics off. See
[What a tile costs on the GPU](tile-layout.md). What is left on every page is the ratio
arithmetic, which is a multiplication by 1.

**To use it**, send a player who has the bug the links, one per part, and ask
which come out white: `https://clickplanet.lol/play?gfx=all` first, which must
show the bug, then `ratio`, `aa`, `earth`, `tiles`, `halo`, `compact`. Ask for
`chrome://gpu` too: it names the driver. **Once the culprit is fixed, take the
URL switches out**; the setting stays.
