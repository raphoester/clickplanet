# Drawing only when something changed

**The loop draws a frame only when the one on screen has stopped being right.**
It used to draw every frame the browser offered, for as long as the tab was
open, and almost all of them were the same picture: measured at rest, before
any interaction, the drawing buffer was byte-identical across a second while
262k tiles and 98k pieces of outline were redrawn 60 times through it. On a
phone that is a flat battery for a still image, and it is why the game was
reported as a battery eater.

`drawsFrame` in `globe.ts` is the whole rule, kept out of the loop so it is
under test. Four things can ask for a frame:

- **The camera turned** — for a drag, the damping glide after one, a zoom, or
  the idle spin. The loop reads this from OrbitControls' `change` event as well
  as from its own `controls.update()`, because a wheel zoom is applied inside
  the wheel handler: `_handleMouseWheel` calls `update()` there, and that call
  is the one that moves the camera and clears the pending scale. Asking the
  loop's own `update()` afterwards gets `false`, and a zoom used to look to the
  loop like a still globe. `change` is dispatched by whichever `update()`
  actually moved the camera, so it is the signal that survives. It is cleared on
  the frame that draws it rather than the tick that reads it, so the cap below
  can hold a frame back without losing the move that asked for it.
- **A flight is on** — a click in orbit flies to a territory (see [A territory,
  from orbit](territory-card.md)). Each tick of it turns the camera and holds off the
  cap, as a drag does.
- **Something the loop drives is still moving** — every `update` that animates
  answers a boolean: `blasts`, `bonusBox`, `enclosureEffect`, `clickEffects`,
  `clickGlints` and `TileField.setHover`. **The frame an effect *ends* on counts**: it is the
  one that takes the flash, the box or the highlight off the screen, so each one
  answers `true` on the tick it stops as well as while it runs.
- **Something outside the loop touched the scene** — `invalidate()`, which
  `applyChanges` calls for every claim, batch and rollback, and which the resize
  listener, the lapsed-box branch and `capture()` call for themselves. It is
  read and cleared once per tick, and only ever set from outside the loop, so
  clearing it on a tick that then skips its frame cannot lose one.

**The idle spin is capped at `IDLE_FRAME_MS`, and it is the one animation the
cap has to be generous with.** It is the only thing that moves with nobody
touching the page, and it is the first thing anybody sees. At a turn in thirty
seconds the surface goes by at a little over a hundred pixels a second: two
pixels a frame at sixty, four at thirty. Thirty was tried and the step is
visible as a step, so the cap is sixty — which still leaves half of what a
120Hz phone or a ProMotion Mac offers. The cap is lifted while the globe is
being handled and for `INTERACTION_GRACE_MS` after — the damping glide is
looked at closely enough to be worth every frame — and it never holds back a
frame that something *changed*: a claim or a blast is drawn as it comes.

**The cap takes the display's nearest frame, not the first one past it.** A cap
in milliseconds lands between two of the display's own frames, and waiting for
the first one strictly past it leaves the answer to a fraction of a
millisecond: 16ms against a 16.7ms frame came out 60fps, then 40, then 60
again. That unevenness is seen where the rate itself is not, so `drawsFrame`
takes `sinceLastTick` and allows half a frame of slack.

**The spin turns by the clock, not by the frame.** `controls.update()` is
handed `spinStep(sinceLastTick)`, and `autoRotateSpeed` is then read as **turns
per minute** — `SPIN_TURNS_PER_MINUTE`, 2, a turn in thirty seconds. Handed
nothing, OrbitControls advances a fixed angle per call instead, which assumes
every display runs at 60: the globe went round in fifteen seconds on a 120Hz
screen and thirty on a 60Hz one, and so travelled twice as far between the
frames that were drawn — the very distance the cap exists to keep small.
`spinStep` caps one tick at `MAX_SPIN_STEP_MS`, because a hidden tab is offered
no frames at all and the whole of that wait would otherwise arrive as one step,
with the globe somewhere else by the time it is looked at again.

**The uniforms the tile pass reads are written before the render, not after
it.** They used to be written at the end of the loop, for the next frame; with
a frame skipped whenever nothing moved, a size worked out for a frame that is
never drawn is a size that never arrives, and the tiles would be left drawn for
a zoom the outline had already moved off.

`preserveDrawingBuffer` stays off (see [Sharing the
globe](sharing.md)). A skipped frame draws nothing at all, so nothing is
composited and the last frame stays on screen; a drawn frame always clears and
redraws everything, so nothing accumulates either.
