# The light

**One pass lights the globe, for every player** — `atmosphere.ts`, a triangle over
the whole frame. It is the season-end clips' light (`globe_light` in their
renderer), which looked much better than the game's: per pixel, it works out where
on the globe's disc the pixel is, from where the camera puts the disc
(`onBeforeRender`), and shades what is already drawn there:

- a Lambert light from the upper left, `0.6 + 0.55 · lambert`, held at 1;
- a rim of the air's colour, `(1 − z)³` at 35%, so the limb is hazy;
- outside the disc, the halo, `exp(−(r − 1) · 18)` at 55%, over a navy sky
  (`SKY`, `0x080b1a`, the renderer's clear colour). The stars are still drawn.

**It blends as `src + dst · src.a`**: the alpha is the shade, the colour is the air
laid over it. A blend factor stops at 1, so the light can darken but never
brighten, and the lit side is a plateau at 1. **The middle of the disc is exactly
1**, which is what the camera looks straight at and, zoomed in, the whole screen: a
player at work on a country sees its flags as they are.

**It is drawn after the ground and before the marks over it** (`renderOrder` 1.5).
The tiles and the outline are lit; the glints, the bonus effects, the highlight and
the bonus box are drawn after it (their groups are at `renderOrder` 1, which three
sorts after everything at 0), so they keep their own colours.

**A battle turns the air to ember**: `paintHalo` swaps the colour of the rim and
the halo. See [Battles](battle.md). Halloween turns it violet (`haunt`, see [Halloween](halloween.md)).

**It used to be a light in each shader, with HD graphics only.** #254 lit the
earth, the tiles and the halo, each in its own shader, and that or the pixel ratio
turned the globe white on some Intel GPUs (see [HD graphics](hd-graphics.md)). The
pass is one simple shader for every player, and nothing it draws over needs to
know about it.

**Only the sea shines, with HD graphics** (`?gfx=earth`). The earth's own shader
adds a glint, read off the texture: the sea is one deep blue whose blue stands
clear of its red and green, and no land does that. `light.glsl` holds the light's
direction, which the pass and the glint share.

**The photo is drawn nearly as it is**: a little less colour (`SATURATION`, 0.75)
and its darks lifted (`LIFT`, a gamma of 1.25), in the earth's own shader, with or
without the glint. As it is, the sea was too deep a blue. It used to be three's
standard material under an ambient light: the texel read as linear, times 2/π, then
encoded to sRGB, which lifted the deep blue sea to a pale one.

**None of it moves on its own**, so none of it costs a frame: the light turns
with the camera, and a still globe is still the same picture. See [Drawing only
when something changed](render-on-demand.md).
