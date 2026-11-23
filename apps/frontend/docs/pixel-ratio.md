# CSS pixels in, drawing-buffer pixels out

**HD graphics draws the canvas at the screen's pixel ratio, capped at 2**
(`pixelRatio()` in `scene.ts`, `?gfx=ratio` alone); without it the ratio is 1. At
1, on a phone or a laptop the browser stretches every frame over twice its
pixels and the whole globe is soft. Past 2 is more than twice the work again for
a difference nobody sees at arm's length.

## Slow frames lower the ratio

**A build flag switches it: `VITE_ADAPTIVE_RATIO=on`.** Unset, HD graphics keeps
the screen's ratio (`ratio: "screen"`) and runs no probe: it draws as it did
before the rule. The flag is a variable of the `clickplanet` worker's builds in
Cloudflare (Settings, Build, Variables and secrets); a change of it takes a new
build of the same commit, so the rule switches alone, with no other change beside
it. `?gfx=adaptive,earth,compact` turns it on in one browser whatever the flag, to
try it on one phone first. It is off in production since 2026-11-23 (see [The flag
is off since 2026-11-23](#the-flag-is-off-since-2026-11-23)).

**Why a flag.** On 2026-11-18, right after the rule first deployed (#883), the
owner's iPhone showed only the page's blue background in a normal tab and played
in a private one. The rule was reverted (#884) and the phone stayed blank; it
played again some minutes later with no deploy between, and its `?diag` report
held no `clickplanet-pixel-ratio`: the rule had never lowered it. The flag lets
the rule come and go alone, to see whether it changes anything.

**When the flag is on and the globe's frames come slow, the page lowers the ratio
by itself**, from 2
to 1.5 to 1 (`Sharpness` in `sharpness.ts`; the rule is `FramePace`, `slow` and
`paid` in `domain/framePace.ts`). A player then gets what turning HD graphics off
gives, without finding the setting. Most players are on phones and never open
Settings. The territory layer's fragment shader runs on every pixel of the globe at
every zoom, so the pixel count is a large part of a phone's frame: ratio 1 draws a
quarter of the pixels of ratio 2.

- **Only the ratio changes.** HD graphics also picks the earth's shader and the
  tiles' compact layout, and both are fixed when the globe is built (see [HD
  graphics](hd-graphics.md)). The ratio changes live, as a `resize` does.
- **A step must pay, or it is undone.** When the frames are slow, `FramePace`
  keeps the median that judged them and lowers the ratio one step. After the 3 s
  that follow a change and a full new window of 60 frames, it compares the new
  median with the old one. At least 15% shorter: the step stays, and the same
  window is judged again, so frames still slow go on to the next step. Not: the
  pixels were not the cost. The ratio goes back to the one before, and the rule
  lowers nothing more in that visit.
- **Why 15%.** A display's frames come in steps (16.7, 25, 33.3 ms), so a cap of
  the browser gives exactly the same median at every ratio, and a frame held by
  the CPU about the same: 0% shorter. The pixels of a frame scale with the ratio
  squared: 1.5 draws 56% of the pixels of 2, and 1 draws 44% of those of 1.5. 15%
  shorter means the pixels were at least about a third of the frame. One step of a
  120 Hz display, 33.3 to 25 ms, is 25% shorter, and pays.
- **For the visit only.** `Sharpness` holds the lowered ratio in memory, never in
  storage. The next visit starts at the screen's ratio again, so no browser can stay
  stuck at a low one. The cost: a phone that is slow on every visit pays its slow
  frames again each time before the ratio drops, 3 s after the tiles are read and a
  window of 60 frames, about 5 s at 30 frames a second. The rule before 2026-11-23
  kept the ratio in `localStorage` (`clickplanet-pixel-ratio`). The globe removes
  that key once when it is built, in a `try`, so no build can read it again.
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
  up. While every tick draws, the interval is not read again: a cap that starts
  later is not seen, and its frames read as slow. The undo is what puts that right.
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
  compile and the tiles upload. The 3 s after the ratio changes, while the canvas
  is sized again. A gap, a tick more than 1 s after the last: a hidden tab, a tab
  the browser stopped, a freeze; the frames before it are dropped, and the next 3 s
  are left out. A step waits through a gap for its window. The tick after the loop
  slept (`FramePace.pause`): it measures nothing, so 200 ms of sleep is never read as
  the display's own interval. One hitch is one frame of 60, and the median does not
  move.
- **Not lowered:** in a build without the flag; with HD graphics off (the ratio
  is 1 already); with `?gfx=` in the URL, which wins, unless it names `adaptive`;
  in a clip or a player's video, whose director draws every frame. These keep the
  screen's ratio.
- **A new ratio is set after the frame**, in a `setTimeout`, as a `resize` sets
  it: sizing the canvas again clears it, and the frame just drawn would be lost.
  The next frame is drawn at the new ratio, with every size read back from the
  renderer. A step undone sets the ratio back up the same way.
- **A slowness that is not the pixels.** A frame slowed by the CPU, or by a cap of
  the browser, is as long at every ratio. The first step does not pay, so it is
  undone: 1.5 for about 5 s, then the screen's ratio for the rest of the visit. On
  2026-11-18, the players who called `earth-2` laggy were helped by HD graphics off,
  which changes the pixels first.
- **Checked** on 2026-11-18, for the rule before the undo, in headless Chrome, fake
  mode, a phone screen at ratio 3, each drawn frame held 25 ms by a scratch hook (a
  CPU throttle of 6 did not slow the M2's loop): 2 to 1.5 two seconds after the
  frames came slow, 1.5 to 1 five seconds later, 1 at the next load, and 2
  throughout on `?gfx=all`. With frames at 60 a second it stayed at 2 for 40 s.
- **Checked** on 2026-11-23, for the undo, in headless Chrome, fake mode, a phone
  screen at ratio 3, `?gfx=adaptive,earth,compact`, the page's
  `requestAnimationFrame` wrapped after 15 s to wait two frames (a cap of 30): once
  the globe drew every tick, 1.5 a second later, 2 again five seconds after, and 2
  for the 15 s left.

## The flag is off since 2026-11-23

On 2026-11-23 the owner's iPhone 18 Pro Max drew the globe at ratio 1, and it was
pixelated. iOS caps a page's frames at 30 a second by itself in Low Power Mode
(WebKit bug 215745), and maybe when the phone is hot. The rule read the display's
own interval at the load, 16.7 ms, and not again while every tick drew. So when the
cap came mid-visit, a fast phone read as slow. A lower ratio did not lift the cap,
so the rule lowered again, to 1 within about 5 s, and kept it for every next visit.
A simulation of `FramePace` showed it: 60 s at 60 frames a second kept 2; after the
switch to the cap, the ratio was 1.5 after 1 s and 1 after 6 s.

The flag was taken out of the worker's build variables, and main was built again. A
build without the flag reads no kept ratio, so those browsers are sharp again. The
step that must pay is the fix: `framePace.test.ts` plays the same switch, and the
ratio ends at 2 after one undone step. Turning the flag back on is the owner's call.

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
