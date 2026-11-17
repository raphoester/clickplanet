# The tile under the pointer

**The CPU finds it from the lattice.** `TilePicker.pick` (`tilePicker.ts`) turns a pixel into the ground under it, and
the ground into its cell:

- The camera is orthographic, so every pixel looks down the same axis. The pixel's ray meets the tiles' sphere, of
  radius 1, at its front: the far side is never picked.
- `tileAt` in `domain/latticeCells.ts` finds the lattice point nearest that ground with `cellOf`, as the territory
  layer's shader does, then whether it is a tile and which one: the cell's bit and count in `TileCells.words`, then
  `TileCells.tiles`, the tile at each texel.
- A ray past the globe, or a cell that is not a tile, is no tile: the sky and the sea.

`globe.ts` picks on a hover, a press and a click, with the pixel in drawing-buffer pixels (`canvasPosition`). The bomb's
and the rally's aim follow `surfacePoint`, a ray against the same sphere, not a tile.

**A pick anywhere in a tile's cell is that tile, at every zoom.** A tile's disc is 0.76 of the spacing wide
(`tilePointSize` against `tileSpacing`), so there is ground between two discs. The GPU picker found nothing there once
the tiles had the frame, and in orbit drew every disc 1.73 times wider for the pick, so that a moving mouse did not
drop the territory in the gaps. The cell is what the territory layer paints and what the hover lights, so a tap between
two discs takes the tile the hover showed. Near the limb the cell is the ground the pixel sees, where a disc, drawn flat
toward the camera, reached past its cell.

**It costs no GPU time and waits for nothing.** The GPU picker drew every tile into a small window around the pointer
and read one pixel back, so the page waited for the GPU to finish its frame on every hover tick. Measured in headless
Chrome with the CPU slowed 4 times (see [Slow devices](../../../docs/boards.md#slow-devices)): a pick took 2 to 3 ms,
1 in 10 over 3 to 4 ms, on `earth-1` and on a test map of 1M tiles; slowed 6 times, 10 ms on 1M in orbit, 1 in 10 over 20 ms.
It now takes under 0.1 ms.
