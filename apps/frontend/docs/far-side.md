# The far side of the globe is not drawn

The earth's own sphere is opaque at radius 0.999, so **half of every pass is
behind it** — and was being run through its whole vertex shader before the depth
test threw it away. The display and border-line vertex shaders now drop it on
one dot product, before the four vertex texture fetches the painted flag costs.

What may not be dropped is what the earth's silhouette does not cover. A point
at radius *r*, an angle *θ* past the limb, projects to a screen radius of
*r·cos θ*, so it still shows while *θ < acos(0.999 / r)* — about 0.045 for the
tiles at 1, and `limbOf(lift)` for each outline pass. On top of that comes half
the tile's own disc, half the line's own width, and how far a blast may throw a
tile outward. `borderLines.test.ts` pins `limbOf` against the earth's radius.

Measured against the same frame with the culling off, the limb comes out
pixel-for-pixel identical. It is worth a few percent of those two passes and no
more — the vertex shader still runs for every point and still reads every
attribute, and only its body is skipped.
