# `src/app/viewer/` — the GPU layer

- `globe.ts` — `createGlobe(options): Promise<Globe>`. Builds the scene, wires
  input and the backends to it, starts rendering. Returns `{tilesCount,
  setCountry, dispose}`. Its loop draws on demand — see [Drawing only when
  something changed](render-on-demand.md).
- `useGlobe.ts` — owns one globe for the lifetime of the component. **Its effect
  must not depend on anything that changes per render**; the selected country and
  the player's hue are pushed into the running globe through separate effects
  rather than rebuilding it.
- `useLeaderboardFeed.ts` — the one place React hears about the board, and
  **it samples rather than follows**. See [Sampling the
  leaderboard](leaderboard-sampling.md).
- `tileField.ts` — owns both point clouds and the per-tile values that change at
  runtime (flag, hover, shields, pulse), in one of two layouts. Mutates them in place
  and reports only the changed ranges. Do not replace these attributes: doing so
  makes the renderer rebuild the whole GPU buffer instead of patching it. See [What
  a tile costs on the GPU](tile-layout.md).
- `gpuPicking.ts` — `GpuPicker`. Persistent 1×1 render target and scene;
  `setViewOffset` narrows the projection to the pixel under the cursor. In orbit the
  tiles are drawn wider for it, with no gap between them (`pickPointSize`). Picks
  are resolved once per frame, not once per mousemove — each one ends in a
  synchronous GPU read that stalls the pipeline. Its occluder is the earth's own
  sphere, which it does not dispose.
- `points.ts` / `latticeWorker.ts` / `landmaskAsset.ts` — the tile positions, rebuilt in a worker from
  the landmask, or the coordinates blob when that fails. See [Tile positions](tile-positions.md).
- `capture.ts` — `readDrawingBuffer`, the frame the player is looking at. **It
  only works inside the render loop**; see [Sharing the
  globe](sharing.md).
- `viewport.ts` — `layoutViewport()`, the size the canvas is set to. **Never
  size the renderer from `window.innerWidth`**: on iOS Safari that follows the
  *visual* viewport, so a pinch fires a `resize` reporting the zoomed-in width,
  the canvas shrinks to it, and releasing the zoom fires nothing that would
  widen it again — the globe is left short of the right edge with a black band
  beside it for the rest of the session.
- `atlas.ts` / `atlasAsset.ts` — country code → sprite region, and the generated
  atlas URL and size.
- `borderField.ts` / `bordersAsset.ts` — the zoomed-out view: tile → landmass,
  who holds how much of each, and the per-landmass table the vertex shader reads.
  See [The zoomed-out view](zoomed-out-view.md).
- `borderLines.ts` / `borderLinesAsset.ts` — the grey outline between one
  country's ground and the next, at a width that does not move with the zoom, and
  the white one around a territory in orbit. See [The countries'
  outlines](country-outlines.md).
- `pointSize.ts` — how big a tile is drawn, how far apart two of them sit, the
  single schedule that hands the frame from the painted flag to the tiles,
  `inOrbit`, the zooms where the pointer asks about a territory, and `flownZoomsOf`,
  how far in a click in orbit flies.
- `zoom.ts` — how far the camera may pull back and push in. The camera is
  orthographic against a globe of radius 1, so `zoom` reads as the share of the
  viewport's height the globe fills: it opens at 1, edge to edge, and pulls back
  to 0.5, the whole sphere with sky around it. The idle spin runs at or below
  the opening zoom and stops once the view is pushed in past it.
- `stars.ts` — the sky behind the globe, drawn as a **pass of its own**. Its
  camera borrows the main camera's orientation and nothing else, so the sky
  turns with the view and holds still through a zoom. Stars in the main scene
  would do neither: that camera is orthographic, so its box frustum would clip
  them to a tube around the globe, and `camera.zoom` would fan them out across
  the screen on the way in. Its scene is not the one `disposeScene` walks, so
  `startAnimation`'s `stop()` disposes it by hand.
- `enclosureEffect.ts` — what every screen shows when somebody closes a shape
  with the enclose bonus (`tilesEnclosed` on the stream): the outline lights up
  and meets at the closing tile, the inside pours in, and a gold ring runs out
  over the ground. **The curves are TypeScript, written into attributes per
  frame**, not GLSL: a shape is a few dozen points, and that is what lets
  `enclosureEffect.test.ts` pin the timing. Marks and the ring have a minimum
  size in pixels, so a shape closed while zoomed out is still seen. A shape that
  arrives while the tab is hidden is not played — it would all start at once on
  return. **A mark is pulled toward the camera by its own size**
  (`unitsPerPixel`): a sprite has one depth, so off the middle of the globe the
  curve of the ground hid half of it. The camera is orthographic, so the pull
  moves only the depth, never the place on screen.
- `clickEffects.ts` — the same, for every click made with spread on
  (`tilesSpread`: a green burst, a spark popping onto each tile around it in
  turn, two rings). It reuses the enclosure's shaders, with normal rather than
  additive rings, which vanished on the white of a flag. A busy planet spreads a
  lot, so at most `MAX_PLAYING` run at once.
- `clickGlints.ts` — **every other click glints on its tile**: this player's at
  once, and anyone else's when its `TileUpdate` says `clicked` — the server sets
  it only on the tile a click named, never on a spread's neighbours, an
  enclosure's inside or a moderator's write. Own clicks echoed back are skipped
  through `OwnClicks`. **A glint is one soft glow and no ring**, gone in 0.7s
  and mostly gone by 0.35s: it is the most frequent thing on the map, a flash
  with a ring was too much at that rate even at its smallest, and a puff that
  held the tile for half a second got in the way of play zoomed in. Before that, a lone faint ring at 20px
  could not be seen, and a full-strength ring of at least 44px with a dark edge
  looked like a bonus. **It is never under `MIN_GLINT_PX`**, so from orbit a
  click is a spark that keeps the planet alive, and otherwise 1.8 tiles wide, so
  pushed in it stays on its tile (`glintSize`). **A hit on a shield is the
  same glint, red and shrinking as it fades** (`playHit`), and a shield placed
  is steel and grows (`playShielded`). **This player's glints are in its
  color's hue** (`hueOf`, handed down through `Globe.setClickHue`); a player
  with no color glints sky blue. Everyone else's are sky blue: a
  `TileUpdate` does not say who clicked. Not white, which vanished on the white
  of a flag. **A click out of view is not played** (`inView`): on the far side
  or off the screen it would cost frames and show nothing. With less motion a
  hit and a placement fade without changing size.
- `earth.ts` — the opaque sphere under the tiles, in the globe's light with
  `?gfx=earth`. See [The light](light.md). The sphere comes from `sphere.ts`, built
  once per globe and shared with the picker. Its texture is the JPEG, or a KTX2
  file with `?gfx=etc1s` or `?gfx=uastc`; see [The globe's texture](globe-texture.md).
- `graphics.ts` — `graphicsOf`, which parts of the sharper, lit globe are on:
  the HD graphics setting, unless the URL names them. See [HD graphics](hd-graphics.md).
- `shaders/` — GLSL for the display, picking, earth, star, enclosure and glint passes.
  `light.glsl` is not a pass but the light they share, pulled in with
  `#include ../light.glsl;` (vite-plugin-glsl's own include, not three's).
