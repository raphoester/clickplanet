# Bombs

A bonus box can hold a bomb (`BonusReward` kind `bomb`), kept until it is dropped
anywhere on the planet. It clears every tile within `radius` of where it lands —
the server's call, not this client's.

**A bomb is aimed or put away.** It is kept until it is dropped, so it cannot stay
aimed: while aimed, a click claims no tile. It is never aimed on its own, not
even when it is caught: the inventory's bomb slot aims it and puts it away
(`Globe.setArmed`), and Escape puts it away. The radius comes from the rules, so a bomb still in hand
after a reload can be aimed. The pieces:
`backends/backend.ts` declares `Bomber` and `BombDrop`, `domain/blast.ts` the
timeline every screen agrees on, `domain/holdToDrop.ts` the gesture,
`viewer/blasts.ts` the drawing, and `components/BombNews.tsx` the line at the top.

**It drops on a press held still for 0.7s, never on a click.** A bomb is precious
and a click is also what ends every drag of the globe — releasing a drag over the
planet used to drop it. `HoldToDrop` is the whole rule: a press that moves more
than 6px is a drag, a press let go early is a change of mind, and only a press
held to the end drops. The aiming ring fills in clockwise while it is held. Mouse
and touch go through the same pointer events, so there is one flow and no
two-tap variant for phones. While armed, a click claims no tile, and the
`viewer-canvas--armed` class blocks the long-press callout on iOS.

**The client sends where it aimed, not a tile.** The sea has no tiles, and whether
an aim is land is decided by the server: past one tile spacing from the nearest
tile it is the sea, the bomb is spent, and everyone sees a splash (`BombDrop.tile`
undefined, nothing cleared). The aiming ring follows the sphere under the cursor
(a ray against the unit sphere), not the tile picker, which finds nothing between
tiles or over water and made a ring that followed it blink.

**Rings are meshes, not tiles.** The aiming ring and the closing "incoming" ring
are a flat quad laid on the globe with a per-pixel ring shader
(`shaders/ring/`). Drawn out of tile discs they broke up over the sea and crawled
as they moved.

**The ground effects are in the tile shader.** The shock wave that throws tiles
outward, the flash and the scorch are computed per vertex from a few uniforms per
blast (`blasts`, `blastRadii`, up to `MAX_BLASTS` at once), so a blast costs a
uniform write per frame and nothing per tile. Each blast slot also owns a flash
sprite and 140 GPU-animated debris points, allocated once and reused.

**The effects are ready before the first bomb.** Built and first drawn when the first bomb a tab saw landed, they
froze the page for about 0.8 s, even on a fast laptop (measured 2026-11-01, after players said the Final Battle
lagged). The globe compiles the whole scene at load (`renderer.compileAsync`), then `Blasts.prime` draws every
flash, debris and ring once at zero opacity, so the graphics driver also finishes its part at load. The first bomb
now draws like any other.

**The clear waits for the explosion.** `BombDrop.cleared` arrives in one event,
and the globe holds it for `IMPACT_DELAY` so the ground goes when the bomb hits,
not when the message lands. Two things keep that honest:

- A tile update that arrives while a clear is waiting **wins its tile**: it came
  after the blast on the wire, so the tile is taken out of the waiting clear and
  the rest of the crater still goes on impact. (Flushing the whole clear early
  instead is what made craters appear before the explosion on a busy map.)
- `PlanetBackend` batches tile updates every 100ms but delivers a blast at once,
  so it **flushes the pending batch first** — otherwise a tile taken just before
  the blast would be applied after it and repaint the crater.

A hidden tab draws no frames, so there the clear is applied immediately.

The dropper's own screen shakes on impact; nobody else's does. Under
`prefers-reduced-motion` nothing moves — no shake, no debris, no displaced tiles —
and the colours stay. A blast off screen gets a red edge pointer with a drawn burst on it
(`blastMark.ts`) — the same component as the bonus box's, which carries a drawn
question mark (`questionMark.ts`). Neither is a character: a glyph is a
different picture on every platform.
