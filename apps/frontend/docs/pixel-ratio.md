# CSS pixels in, drawing-buffer pixels out

**HD graphics draws the canvas at the screen's pixel ratio, capped at 2**
(`pixelRatio()` in `scene.ts`, `?gfx=ratio` alone); without it the ratio is 1. At
1, on a phone or a laptop the browser stretches every frame over twice its
pixels and the whole globe is soft. Past 2 is more than twice the work again for
a difference nobody sees at arm's length.

**No player gets antialiasing** (`?gfx=aa` turns it on, to compare). Its cost
grows with the pixels drawn, about 60 MB of GPU memory per million, and the
one-pixel feathers below already soften the edges. See [HD
graphics](hd-graphics.md).

**Every size in pixels in this viewer is a CSS pixel**, and is multiplied by the
ratio on its way to the GPU: the tile's point size, the outline's width
(`halfWidthOf`), the smallest a mark or a
ring of the bonus effects may be, the smallest debris. So are the thresholds:
`coarseHandover` and `flagPaint` are worked out in CSS pixels, or a sharper
screen would hand over at half the zoom. What is measured against
`gl_PointSize` stays in drawing-buffer pixels — `pixelsPerRadian`, the pick's
window, the one-pixel feathers that soften an edge.

The click is already in drawing-buffer pixels: `canvasPosition` scales the
pointer by `canvas.width / rect.width`. `resize` sets the ratio again, because a
browser zoom changes it. **Read the ratio back from the renderer, never from
`window.devicePixelRatio`**: a ratio that changes with no `resize` then leaves
the frame no sharper, but every size still agrees with every other.
