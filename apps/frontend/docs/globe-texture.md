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

## Kept compressed on the GPU: a trial

**The GPU holds the JPEG as raw pixels, 42.7 MB with its mipmaps.** A KTX2 file
in Basis Universal's formats stays compressed on the GPU, so `npm run earth` also
writes the same corrected pixels as two of them, with the encoder from the
`ktx2-encoder` package. `?gfx=etc1s` and `?gfx=uastc` load them through three's
`KTX2Loader` (see [HD graphics](hd-graphics.md)), and the JPEG instead when the
device has no compressed format or the load fails.

- **ETC1S** (709 KB; 5.3 MB on a phone's GPU, 10.7 MB on a desktop's) gives each
  block of 4×4 pixels one colour, only lighter or darker. A cell is 1.5 pixels
  across, so every block on a coast holds land and water and comes out grey: a
  small island becomes a grey square, and the shallow water loses its blue.
- **UASTC** (3.9 MB; 10.7 MB on the GPU) cannot be told from the JPEG, but it
  makes a first visit 3.5 MB bigger and shows the earth 3.5 s later on Fast 4G.

Both files are written upside down, because a compressed texture cannot be
flipped on upload, and marked linear, so three samples their bytes as stored, as
it does the JPEG's: both earth shaders are tuned to that. The transcoder is
three's own, imported with `?url` so that Vite serves it from `/assets/` under its
hash, and a `LoadingManager` maps the names `KTX2Loader` asks for to those URLs.
