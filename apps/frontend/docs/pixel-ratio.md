# CSS pixels in, drawing-buffer pixels out

**HD graphics draws the canvas at the screen's pixel ratio, capped at 2**
(`pixelRatio()` in `scene.ts`, `?gfx=ratio` alone); without it the ratio is 1. At
1, on a phone or a laptop the browser stretches every frame over twice its
pixels and the whole globe is soft. Past 2 is more than twice the work again for
a difference nobody sees at arm's length.

## Slow frames lower the ratio

**When the globe's frames come slow, the page lowers the ratio by itself**, from 2
to 1.5 to 1 (`Sharpness` in `sharpness.ts`; the rule is `FramePace` and `slow` in
`domain/framePace.ts`). A player then gets what turning HD graphics off gives,
without finding the setting. Most players are on phones and never open Settings.
The territory layer's fragment shader runs on every pixel of the globe at every
zoom, so the pixel count is a large part of a phone's frame: ratio 1 draws a
quarter of the pixels of ratio 2.

- **Only the ratio changes.** HD graphics also picks the earth's shader and the
  tiles' compact layout, and both are fixed when the globe is built (see [HD
  graphics](hd-graphics.md)). The ratio changes live, as a `resize` does.
- **It only goes down.** The lowered ratio is kept in `localStorage`
  (`clickplanet-pixel-ratio`), so the next visit starts at it. Every storage
  access is in a `try`; when the storage fails, the lowered ratio lasts as long as
  the globe. To give a browser its sharpness back, remove the key.
- **A frame is the interval between two ticks that both drew one.** Rendering is
  on demand (see [Drawing only when something changed](render-on-demand.md)), so a
  frame is measured only while the globe moves without a break: the idle spin of
  the first minute, a drag, the glide after it, a flight.
- **The display's own interval is the tick after a tick that drew nothing.** That
  tick does no work, so it comes at the display's pace, and at the pace of a tab
  the browser holds at 30 frames a second (iOS Low Power Mode). The loop draws
  every tick while the idle spin runs on a 60 Hz screen, so the ticks before the
  loop starts count too: while the board loads, a probe reads the browser's ticks,
  for at most 10 s. **Only the last 5 are kept**: a load from the cache gave the
  probe 7 ticks in fake mode, and with no 5 the frames are never judged. A busy
  load makes a tick longer, never shorter, so a wrong one can only hold the ratio
  up.
- **The rule.** The frames are slow when the median of the last 60 is more than
  1.75 times the interval the loop aims at: the display's own (the median of the
  last 5), but never shorter than a sixtieth of a second. A 120 Hz phone at 60
  frames a second is not slow.

  | Screen | Not slow | Slow |
  |---|---|---|
  | 60 Hz | 60 a second | 30 a second |
  | 120 Hz | 60, and 40 | 30 |
  | a tab held at 30 | 30 | 15 |

  **Why 30 a second.** The idle spin at 30 was tried, and its step was visible as
  a step (see [Drawing only when something changed](render-on-demand.md)). On a
  60 Hz screen a frame comes every 16.7 ms or every 33.3 ms, so a median above
  29 ms means most frames missed the screen's. **Why 1.75 and not 2.** A display's
  frames come in steps (16.7, 25, 33.3 ms at 120 Hz), and 2 times 16.7 ms is a step
  itself: a frame a hair early or late would flip the answer. 1.75 times is 29 ms,
  between the steps of every common screen.
- **What is left out.** The frames until 3 s after the tiles are read: shaders
  compile and the tiles upload. The 3 s after a lowered ratio, while the canvas
  is sized again. A gap, a tick more than 1 s after the last: a hidden tab, a tab
  the browser stopped, a freeze; the frames before it are dropped, and the next 3 s
  are left out. One hitch is one frame of 60, and the median does not move.
- **Not lowered:** with HD graphics off (the ratio is 1 already); with `?gfx=` in
  the URL, which wins (`ratio: "screen"` in `graphics.ts`); in a clip or a player's
  video, whose director draws every frame. These keep the screen's ratio and do
  not read the kept one.
- **A lowered ratio is set after the frame**, in a `setTimeout`, as a `resize` sets
  it: sizing the canvas again clears it, and the frame just drawn would be lost.
  The next frame is drawn at the new ratio, with every size read back from the
  renderer.
- **Known limit.** A frame slowed by the CPU, not by the pixels, lowers the ratio
  too, and a lower ratio does not make it faster. On 2026-11-18, the players who
  called `earth-2` laggy were helped by HD graphics off, which changes the pixels
  first.
- **Checked** on 2026-11-18 in headless Chrome, fake mode, a phone screen at
  ratio 3, each drawn frame held 25 ms by a scratch hook (a CPU throttle of 6 did
  not slow the M2's loop): 2 to 1.5 two seconds after the frames came slow, 1.5 to
  1 five seconds later, 1 at the next load, and 2 throughout on `?gfx=all`. With
  frames at 60 a second it stayed at 2 for 40 s.

## Sizes

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
browser zoom changes it, and slow frames lower it the same way. **Read the ratio back from the renderer, never from
`window.devicePixelRatio`**: a ratio that changes with no `resize` then leaves
the frame no sharper, but every size still agrees with every other.
