# The light

**Only with HD graphics, or `?gfx=light` or one of its three parts** — see [HD graphics](hd-graphics.md).
Without it the earth is three's standard material under an ambient light, the
tiles are unlit, and the globe reads as a flat blue disc.

**The globe is lit by one light, and everything on its surface calls the same
function for it** — `shaders/light.glsl`, included by the earth, the tiles and
the halo. Lighting only the earth would have left the flags floating flat on a
shaded ball.

**It is a studio light, not the sun.** It sits with the camera, up and to the
left, so the same side is always lit however the globe is turned: a real sun
would put half the players' countries in the dark. Half-Lambert, squared, wraps
it round the globe with no terminator, and the far limb keeps about half.

**`shadeOf` is exactly 1 at the middle of the disc**, which is what the camera
looks straight at and, zoomed in, the whole screen. A player at work on a
country sees its flags as bright as before there was a light; the lit side of
the globe seen whole comes out brighter still.

**The air is lit too.** `hazeOf` lays the halo's colour over the ground seen
edge-on, and `lit` shades it with the ground, so the rim is bright on the lit
side and fades on the far one. The halo itself is shaded by the same function,
taking the limb under it as its normal. `AIR` is the colour, and `COLOUR` in
`atmosphere.ts` is the same one for the unlit halo, until the switches go.

**Only the sea shines.** The earth adds a glint, read off the texture: the sea
is one deep blue whose blue stands clear of its red and green, and no land does
that. The photo itself is drawn as the standard material used to draw it,
`sRGB(texel · 2/π)`, so the sea is still the blue it was.

**None of it moves on its own**, so none of it costs a frame: the light turns
with the camera, and a still globe is still the same picture. See [Drawing only
when something changed](render-on-demand.md).
