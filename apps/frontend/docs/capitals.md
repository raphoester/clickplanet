# The capitals

**Two knobs of the rules in force play on the capitals: no shield on a capital, and the knock-out** (see the backend's
[The capitals' rules](../../backend/docs/capitals.md)). Capital Siege plays both on its fresh map, which starts from
each country's capital (the backend's [Arena battles](../../backend/docs/arenas.md)); any set may play them on the world's
map too.

- **Where the page learns them.** `GetChargeRules.knock_out` and `capital_shields_fixed`, read into `BonusRules` and read
  again on `rules_changed`. `globe.ts` follows them in `onRules` (`followCapitals`).
- **What it reads then.** `static/capitals-<hash>.json`, once, the first time either rule plays (`capitals.ts`; URL in
  `capitalsAsset.ts`). It is a copy of [`/map`](../../../map/README.md). A file written for a map of another size is
  refused.
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

  Its 100 shields show as the most the shield marks draw (`tileShields`).
- **No shield on a capital**, while `capitalShieldsFixed`:
  - The shield switch on a capital sends nothing, and the slot says "No shield on a capital" (`placementOf` answers
    `"capital"`).
  - A fortify raises no shield on it, as on the server.
- **A knock-out** (the backend's [The knock-out](../../backend/docs/capitals.md#the-knock-out-capitalsknockout)): a take of a flag's
  capital clears every other tile of the flag. The stream sends it as one `knock_out`, after the take (`knockOutOf`,
  `listenForKnockOuts`); the globe flushes the earlier updates first, as for a bomb, and `MapTiles.applyKnockOut`
  clears each tile at its version. The crown goes with the capital's owner.
- **The flags out**: every map read says them (`Ownerships.knockedOut`), and `MapTiles` keeps them with each knock-out
  heard. A read of the same start adds to them; a new start replaces them.
- **The flags still in reach React**: while the knock-out plays, the globe hands `onStanding` the capitals' countries
  and the flags out (`Standing`, `domain/standing.ts`) on each rule change, read and knock-out, and `undefined` while it
  does not play; it goes into `useStanding` (`app/knockout/standingStore.ts`). `stands(standing, flag)` is the server's
  rule: while the knock-out does not play every flag, while it does a flag with a capital not out. A region flag
  (England, Scotland, Wales, Brittany) has no capital.
- **The out window** (`app/knockout/OutModal.tsx`, in `PlayScreen`): while the knock-out plays and the flag played
  (held, or a newcomer's choice) does not stand, "France is out" (or "England has no capital"), "Pick a flag that is
  still in.", and the list of the flags still in. A pick pledges at once (the server lets a flag that does not stand
  change past the lock and the hour). **It can be closed, to watch** (the owner, 2026-11-10); a click out of reach opens
  it again (`useStanding.ask`).
- **The menu's picker** lists only the flags still in while the knock-out plays, and is open to a player whose flag does
  not stand (`lockOf` learns the standing): see [The flag played for](flag.md).
- **The fake backend plays neither rule.** Check them on the local stack.
