# The globe's texture

The sphere under the tiles is a photograph, and it used to be the third thing
that answered "sea or land" — disagreeing with both the tile set and Natural
Earth. It cannot be an authority: at 4096×2048 a one-tile island is three pixels,
so the mosaic blends it into open water however carefully the polygons are drawn.
A player must never see green with nothing to click on it, or a disc floating on
open water.

So `npm run earth` cuts it from the tile field instead. A pixel is land when the
lattice vertex nearest to it is a tile, and sea when it is not — so `cover` is each
tile's own cell, the same cell edge the countries' outline runs along, and a
one-tile lake is water across its whole cell. It fades over 0.4 of a tile spacing
across that edge. The photo is corrected toward it:

```
out = photo + (cover - opinion) * (landColour - seaColour)
```

**It is the identity wherever the photo already agrees.** `opinion` is what the
pixel's own colour says, read off the same two colours, so the correction is zero
where the two agree — most of the globe — and full on a three-pixel island. The
two colours are the photo's own local averages over the land and over the water,
in 64-pixel blocks, so they follow latitude, depth and biome and nothing is
painted in a palette somebody chose.

**The two colours are found twice.** Taken straight off `cover`, a block whose only
land is an island the photo drew as water learns that land here looks like water,
and then leaves that island exactly as it found it — the one case this exists for.
The second pass weights each pixel by whether the photo and the tile field already
agree about it, and a block with no agreement left takes its colours from a
neighbour that has some.

Where land and water are the same colour — under cloud, on an ice shelf — there is
no direction to move a pixel along, and nothing is moved. `scripts/map/recolour.mjs`
holds the rule and `recolour.test.mjs` pins every case above.

**The JPEG keeps colour at full resolution** (`chromaSubsampling: "4:4:4"`). The
default halves it, and a cell is about three pixels wide, so a strip of land one
tile wide came out blue and a lake came out green. At quality 80 it is the same
size as the halved one at 88.

`cover` used to be a disc round each tile reaching 0.83 of a spacing. Its
neighbours' discs all but filled a one-tile lake, so the lake stayed green inside
its outline.

An earlier version was a frequency separation, `mix(sea, land, cover) + (photo -
average)`. That is only the identity inside a block that is all land or all water;
along a coast it pushed the land further from the water in both directions and
moved 37% of the image.
