# Drawing only when something changed

**The loop draws a frame only when the one on screen has stopped being right.**
It used to draw every frame the browser offered, for as long as the tab was
open, and almost all of them were the same picture: measured at rest, before
any interaction, the drawing buffer was byte-identical across a second while
262k tiles and 98k pieces of outline were redrawn 60 times through it. On a
phone that is a flat battery for a still image, and it is why the game was
reported as a battery eater.

`drawsFrame` in `globe.ts` is the whole rule, kept out of the loop so it is
under test. Five things can ask for a frame:

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
  `clickGlints`, `fortifyWaves` and `TileField.setHover`. **The frame an effect *ends* on counts**: it is the
  one that takes the flash, the box or the highlight off the screen, so each one
  answers `true` on the tick it stops as well as while it runs.
- **Something outside the loop touched the scene** — `invalidate()`, which
  `applyChanges` calls for every claim, batch and rollback, and which the resize
  listener, the lapsed-box branch and `capture()` call for themselves. It is
  read and cleared once per tick, and only ever set from outside the loop, so
  clearing it on a tick that then skips its frame cannot lose one.
- **A pulse is on screen** — `ambient`: the front's white ring, the fortify gold ring and an infected tile's
  virus, drawn at most every `AMBIENT_FRAME_MS` (see [The front, for a first visit](front-guide.md)), and only
  while the beat has moved since the last frame drawn. The beat stops a minute after the last input, with the
  idle spin (below).

**The idle spin is capped at `IDLE_FRAME_MS`, and it is the one animation the
cap has to be generous with.** It moves with nobody touching the page, and it is
the first thing anybody sees. At a turn in thirty
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

**The idle spin and the pulses stop a minute after the last input.** `watched`
in `globe.ts` is the rule: the page is watched while the last input is less than
`WATCHED_FOR_MS` (60 s) old. An input is a `pointerdown`, `pointermove`,
`wheel`, `keydown` or `touchstart` anywhere on the page, read on `window` in the
capture phase, so a panel that stops an event still counts it. The page's load
counts as an input, so the landing view turns for its first minute. `spins`
turns the globe on its own only at or below `RESTING_ZOOM`, and only while the
page is watched. The next input turns it again, and the pulses beat again.

- **Why.** The cost of the spin is its frame rate, not its scene: a bare WebGL
  page that draws one triangle every frame costs as much. A page left open
  zoomed out drew 60 frames a second for as long as it stayed open, and most
  players are on phones. A 30fps cap for the spin was tried, and the step was
  visible. Measured on 2026-11-18 in headless Chrome on an M2 Max, renderer and
  GPU process together, 70 s after one input, with no claims coming in: 17% of
  a core before, 4.4% after, on a desktop screen and a phone screen alike. No
  frame is drawn then; the rest is the page around the globe (about:blank costs
  0.3%).
- **The loop sets `controls.autoRotate` on every tick**, from `spins`. It used
  to be set in OrbitControls' `change` listener, which cannot stop a spin: once
  the camera is still, no `change` comes. It also missed the load: the
  constructor dispatches its first `change` before `globe.ts` listens, so a
  fresh page did not turn until the first drag or wheel. With `autoRotate` off,
  the damping glides the globe to rest in about a second (55 ticks), and then
  nothing asks for a frame.
- **The pulses stop at their faintest.** The rings beat on `pulseTime`, a
  clock of their own. They cannot beat on `time`: the blasts and the fortify
  waves read it, and it has to keep running. `beatClock` runs `pulseTime` with
  the frame while the page is watched. After the minute, it runs on to the next
  trough of the beat (within 1.1 s) and holds there: the ring stays at half
  strength and the tile does not swell. That is the calmest frame of the beat,
  where a ring caught mid-beat would stay at any brightness. A pulse asks for a
  frame only while `pulseTime` differs from the frame on screen, so the last
  frame drawn is the trough. The next input beats on from there, with no jump.
  An infected tile's virus turns on `time`, so it stops wherever it is: it has
  no bright or faint point. `TROUGH_SLACK` keeps a held clock from creeping on
  to the next trough by rounding (32 of 1000 troughs would).
- **No rule on focus or blur.** On `crazygames.clickplanet.lol` the game is in
  an iframe that has no focus until the first click (see
  [CrazyGames](crazygames.md)): a blur rule would stop the spin on load for
  every visitor there. The minute covers a player who switches windows.
- **Everything else still draws when it changes**: claims and batches
  (`invalidate()`), the glint of another player's claim, blasts, the box,
  flights, a resize, `capture()`. Each glint plays for 0.7 s at the full frame
  rate, so a busy world in view still draws for much of the time: at the
  ledger's average of November 2026, about 2.2 takes and strikes a second over
  the whole map, the stopped page still drew 30 to 50 frames a second and cost
  about 12%. The minute saves the most when the world is quiet.
- **The director takes no part.** A clip or a player video aims the camera
  itself and never calls `controls.update()`, so `autoRotate` is not read. Its
  pulses always beat: nobody touches a page while it records, and a clip's
  clock is the video's (see [Clips](clips.md)), which can run past a minute.

**Only a frame that is drawn culls the tiles.** `TileField.cull` picks the
pieces of the map in sight (see [The tiles out of sight are not
drawn](far-side.md)) after `drawsFrame` has said yes, just before the render,
with the camera as it is drawn, the blast's shake on it. A tick that draws
nothing costs nothing there, and at the landing view, under the painted flags,
a drawn frame runs no tile at all.

**The uniforms the tile pass reads are written before the render, not after
it.** They used to be written at the end of the loop, for the next frame; with
a frame skipped whenever nothing moved, a size worked out for a frame that is
never drawn is a size that never arrives, and the tiles would be left drawn for
a zoom the outline had already moved off.

`preserveDrawingBuffer` stays off (see [Sharing the
globe](sharing.md)). A skipped frame draws nothing at all, so nothing is
composited and the last frame stays on screen; a drawn frame always clears and
redraws everything, so nothing accumulates either.
