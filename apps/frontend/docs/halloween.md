# Halloween

**The game dresses for Halloween while the schedule plays an entry of kind `halloween`.** It is switched on as a
battle is: the operator adds an entry on the backoffice's Rule schedule page and picks its kind (see the backoffice's
[Ops](../../backoffice/docs/ops.md)). The client reads the kind from the turn (see [Battles](battle.md#reading-what-plays)).
A Halloween entry scores nothing, as a plain one, and has no chip, no moment and no clock: only the look.

## The look

- **The mood is `halloween`** (`domain/mood.ts`, `moodOf`): `tokens.css` turns the panels to a night purple and the
  secondary button to violet under `[data-mood="halloween"]` (see [`DESIGN.md`](../DESIGN.md#moods)). Frozen wins
  over it, as over a battle.
- **The halo turns violet** (`haunt` in `app/viewer/atmosphere.ts`), as a battle's turns to ember.
- **The props are drawn art, in `app/halloween/`**: a jack-o'-lantern, a bat, a spider web and a skeleton peeking
  over an edge, each an SVG Vite hashes. There is no emoji font in them, for the reason the chat's reactions have
  none (see [Chat](chat.md#reactions)).
- **The props sit on the zones' edges** (`Halloween.css`, CSS alone, under the mood): the pumpkin on the dock's top
  left, the skeleton peeking over its top right, a web in the menu's top right corner and in the chat's top left,
  and a bat on the chat's top edge. None takes a pointer. **Nothing is drawn on the globe's surface**: flat sprites
  there were tried, and they cut into the sphere.
- **What stands on the dock is its crown, `--dock-crown`** (`index.css`, 0 by default, 44px under the mood). What
  keeps clear of the dock keeps clear of it too: the menu's height below 1444px, the chat that sits above the dock
  from 769px to 1099px, and the phone's chat toast. So a toast pops up above the pumpkin, and the props never hide.

## The bonus box

**The box flies as a jack-o'-lantern** (`app/viewer/pumpkin.ts`): a mesh, a sphere with eight ribs, squashed, with a
stem in its dent. Its grin is carved on the front, and **the box's question mark on its sides and its back**
(`drawQuestionMark`, the box faces' own), so it still reads as a bonus. The carving is a blurred white-on-black mask,
512 × 256 round the pumpkin; its shader (`shaders/pumpkin/`) finds in it the cut, the shadow round the cut, the
flesh and the candle, which flickers (`flickerAt`).

- **The look is the box's** (`BoxLook`, `box` or `pumpkin`, `BonusBox.setLook`): `Viewer` hands `useGlobe`
  `bonusLook`, and the globe's `setBonusLook` sets it. The orbit, the fade, the burst and the timings are the box's.
- **The pumpkin tumbles on all three axes** (`pose`, `SPIN`), as the box does, so its grin and its marks come round
  in turn.
- **It is drawn after the globe's light**: a three.js group starts its own draw order, so the pumpkin's group is at
  `renderOrder` 1, as the box's. At 0 the light pass laid the air over it, and it looked see-through.
- **Its glow cycles orange, violet and green** (`HALO_CYCLES.pumpkin`), not the box's four colours.
- **A click aims at its body** (`Pumpkin.body`): the stem is too thin to aim at.
- **The pointer to a box off screen** wears the pumpkin art for a badge, with an orange arrow (`Halloween.css`).

## In fake mode

`startHalloween(minutes = 120)` on the console plays a Halloween entry at once; `endBattle()` ends it. `giveBox()`
flies a box at once (see [Commands](commands.md)).
