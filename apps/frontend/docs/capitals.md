# The capitals

**Three knobs of the rules in force play on the capitals: no shield on a capital, the knock-out and the cut-off** (see
the backend's [The capitals' rules](../../backend/docs/capitals.md)). Capital Siege plays all three on its fresh map, which starts from
each country's capital (the backend's [Arena battles](../../backend/docs/arenas.md)); any set may play them on the world's
map too.

- **Where the page learns them.** `GetChargeRules.knock_out` and `capital_shields_fixed`, read into `BonusRules` and read
  again on `rules_changed`. `globe.ts` follows them in `onRules` (`followCapitals`).
- **What it reads then.** The board's `capitals-<hash>.json`, once, the first time either rule plays (`capitals.ts`;
  URL in the board's `urls.capitals`, `boardsAsset.ts`). It is a copy of [`/map`](../../../map/README.md). A file written for a map of another size is
  refused.
- **The seats.** An arena's map gives each flag with no ground of its own (a sold flag, England, Scotland and Wales
  among them) a seat, played as its capital (the backend's [The seats](../../backend/docs/capitals.md#the-seats)). Every map read says
  them (`Ownerships.seats`, from `GetMapResponse.seats`); `MapTiles` keeps the map's, a new `Map` each time a read adds
  one, forgotten with a new start. The globe's capitals are the board's file and the seats together (`withSeats`), built
  again when a read brings a seat: the crown, the fixed shields, the mine's gate and the standing read them all.
- **The capitals still standing** (`domain/capitalMarks.ts`), while either rule plays. A capital is marked while its own
  country holds it:
  - a gold crown on the top of the tile, over its flag and its shield badge, as a crest on a shield (the owner,
    2026-11-10: a ring in the middle read as a pink shield);
  - drawn last, over every ring and flash of the tile: the front's white ring runs through the top of the tile, and a
    crown under it read as a bug (the owner, 2026-11-11). Its texture is read on every fragment of the tile and kept
    inside its box: read only inside the box, the edge had no gradient and drew a grey frame around the crown;
  - drawn like the shield badge, ink outline and drop, in the badge's own texture: the cell after the last count
    (`shieldMarks.ts`, `CROWN`; `CROWN_AT` and `CROWN_SIZE` in the display fragment shader);
  - on bit 8 of the pulse slot (`TileField.setCapitalMarks`);
  - shown from 10 px across, as the mission ring, never under the painted flags.

- **A capital's badge counts every shield it holds** (the owner, 2026-11-12: its 100 shields read 10, the most a tile
  holds, so its first 90 hits changed nothing on screen). The page shows the count the server sends: the sheet of
  badges (`shieldMarks.ts`) holds a count up to the most a tile holds on the map (`TileShields.most`), `tileShields`
  at least, by tens past it (`fitMarks` in `globe.ts`). It grows when an update brings a bigger count, as a capital's
  100 in an arena, and falls back to `tileShields` once a big stack fell and no tile holds more, as when the world's
  map plays again. A hit redraws it only when it crosses one of those. Past 16 cells the sheet is laid out square in
  64 px cells: a capital's 100 weigh about 1.8 MB on the GPU, and 255 stay under 4096 px. Fortify still raises a tile
  to `tileShields`. No rule names a capital's shields but the seed's.
- **No shield on a capital**, while `capitalShieldsFixed`:
  - The shield switch on a capital sends nothing, and the slot says "No shield on a capital" (`placementOf` answers
    `"capital"`).
  - A fortify raises no shield on it, as on the server.
- **A knock-out** (the backend's [The knock-out](../../backend/docs/capitals.md#the-knock-out-capitalsknockout)): a take of a flag's
  capital clears every other tile of the flag. The stream sends it as one `knock_out`, after the take (`knockOutOf`,
  `listenForKnockOuts`); the globe flushes the earlier updates first, as for a bomb, and `MapTiles.applyKnockOut`
  clears each tile at its version. The crown goes with the capital's owner.
- **A cut-off** (the backend's [The cut-off](../../backend/docs/capitals.md#the-cut-off-capitalscutoff)): a take that
  cuts a flag's land from its capital clears the land cut off. The stream sends it as one `cut_off`, after the take
  (`cutOffOf`, `listenForCutOffs`); the globe flushes the earlier updates first, as for a knock-out, and
  `MapTiles.applyCutOff` clears each tile at its version. No flag goes out. The page does not need the rule: it reads
  no `GetChargeRules` field for it, and draws nothing more than the blank tiles.
- **The flags out**: every map read says them (`Ownerships.knockedOut`), and `MapTiles` keeps them with each knock-out
  heard. A read of the same start adds to them; a new start replaces them.
- **The flags still in reach React**: while the knock-out plays, the globe hands `onStanding` the capitals' countries
  and the flags out (`Standing`, `domain/standing.ts`) on each rule change, read and knock-out, and `undefined` while it
  does not play; it goes into `useStanding` (`app/knockout/standingStore.ts`). `stands(standing, flag)` is the server's
  rule: while the knock-out does not play every flag, while it does a flag with a capital not out. A sold flag
  stands on its seat; one with no seat (opened while the arena plays) has no
  capital, and on the world's map none has one.
- **The out window** (`app/knockout/OutModal.tsx`, in `PlayScreen`; its pick and confirm are `PickAnotherWindow`, which
  the window of a closed flag shares, with no close: see [The list of flags](flag-catalog.md#a-held-flag-closed)): while the knock-out plays and the flag played
  (held, or a newcomer's choice) does not stand, "France is out" (or "England has no capital"), "Pick a flag that is
  still in.", and the list of the flags still in. A pick pledges at once (the server lets a flag that does not stand
  change past the lock and the hour). **It can be closed, to watch** (the owner, 2026-11-10); a click out of reach opens
  it again (`useStanding.ask`). **A pick asks to be confirmed** (the owner, 2026-11-14, in a Capital Siege): the window
  opens under a player still clicking the globe, whose next click picked the flag under the finger and locked it. A
  country in the list turns the window into "Play for Germany?", its flag, and under the rules' lock that it cannot
  change again unless it is knocked out too (with no lock nothing: the forced change starts no wait, so the flag can
  change at once, and once the battle ends it changes back once at once); "Confirm" pledges it,
  "Back" returns to the list. **Confirm takes no click for 1 s after it shows** (`CONFIRM_ARM_MS`), each pick anew,
  so a double click on a country pledges nothing. A pick that failed says so on the confirm step.
- **The menu's picker** lists only the flags still in while the knock-out plays, and is open to a player whose flag does
  not stand (`lockOf` learns the standing): see [The flag played for](flag.md).
- **The fake backend plays neither rule.** Check them on the local stack.
