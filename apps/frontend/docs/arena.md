# The arena

**A battle whose rule set has `arena.on` is fought on a blank map.** Each country starts from its capital, and a take
lands only next to the flag's land or across its waters (see the backend's
[Arena battles](../../backend/docs/arenas.md)). The page rings what the player can take, marks each capital still
standing, and stops a click elsewhere.

- **Where the page learns it.** Every map read says whether the map is an arena's (`GetMapResponse.arena`, read into
  `Ownerships.arena`). `MapTiles.arena` keeps what the last read that counted said. A map becomes an arena's, or the
  world's again, only with a new epoch, and a new epoch reads every range again, so the first read of the new map
  tells it. `globe.ts` follows it on every batch (`followArena`).
- **What it reads then.** `static/capitals-<hash>.json` and `static/waters-<hash>.json`, once, the first time a read
  says arena (`capitals.ts`, `waters.ts`; URLs in `capitalsAsset.ts` and `watersAsset.ts`). Both are copies of
  [`/map`](../../../map/README.md). A file written for a map of another size is refused.
- **What the player can take** (`domain/front.ts`, a `Front` with `Waters`):
  - every tile it does not hold that touches its land;
  - every coast tile of a water where it holds a coast. A tile between two waters is a coast of both.

  The front counts the flag's coasts on each water. It lights a water's whole coast when the flag takes its first
  coast there, and puts it out when the flag loses its last one. The ocean has 15,294 coast tiles, so a flag on the
  sea lights every ocean coast at once. It is the white ring of a first visit's front (see [The front](front-guide.md)),
  on every page while the arena plays. A guided page goes back to its plain front after the arena.
- **The capitals still standing** (`domain/capitalMarks.ts`). A capital is marked while its own country holds it:
  - a gold crown on the top of the tile, over its flag and its shield badge, as a crest on a shield (the owner,
    2026-11-10: a ring in the middle read as a pink shield);
  - drawn like the shield badge, ink outline and drop, in the badge's own texture: the cell after the last count
    (`shieldMarks.ts`, `CROWN`; `CROWN_AT` and `CROWN_SIZE` in the display fragment shader);
  - on bit 8 of the pulse slot (`TileField.setCapitalMarks`);
  - shown from 10 px across, as the mission ring, never under the painted flags.

  Its 100 shields show as the most the shield marks draw (`tileShields`).
- **A click elsewhere is stopped** (`globe.ts`, right after the frozen map). Nothing is sent, the refused sound plays,
  the meter shakes, and a bubble over it says "Take a tile next to your land" (`refusedClicksStore.refuseOutOfReach`,
  `ClickBudgetMeter`). The server may refuse a click the page let through (`ReachRefusal`, read as `OutOfReachError`).
  That click is rolled back and told the same way.
- **No shield on a capital.**
  - The shield switch on a capital sends nothing, and the slot says "No shield on a capital" (`placementOf` answers
    `"capital"`).
  - A fortify raises no shield on it, as on the server.
- **A knock-out** (the backend's [The knock-out](../../backend/docs/arenas.md#the-knock-out)): a take of a flag's
  capital clears every other tile of the flag. The stream sends it as one `knock_out`, after the take (`knockOutOf`,
  `listenForKnockOuts`); the globe flushes the earlier updates first, as for a bomb, and `MapTiles.applyKnockOut`
  clears each tile at its version. The crown goes with the capital's owner.
- **The flags out**: every map read says them (`Ownerships.knockedOut`), and `MapTiles` keeps them with each knock-out
  heard. A read of the same start adds to them; a new start replaces them.
- **The flags still in reach React**: the globe hands `onArenaFlags` the capitals' countries and the flags out
  (`ArenaFlags`, `domain/standing.ts`) on each read and knock-out of an arena, and `undefined` on the world's map; it
  goes into `useArena` (`app/arena/arenaStore.ts`). `stands(flags, flag)` is the server's rule: outside an arena every
  flag, in one a flag with a capital not out. A region flag (England, Scotland, Wales, Brittany) has no capital.
- **The out window** (`app/arena/OutModal.tsx`, in `PlayScreen`): while an arena plays and the flag played (held, or a
  newcomer's choice) does not stand, "France is out" (or "England has no capital"), "Pick a flag that is still in.",
  and the list of the flags still in. A pick pledges at once (the server lets a flag that does not stand change past
  the lock and the hour). **It can be closed, to watch** (the owner, 2026-11-10); a click out of reach opens it again
  (`useArena.ask`).
- **The menu's picker** lists only the flags still in while an arena plays, and is open to a player whose flag does not
  stand (`lockOf` learns the arena's flags): see [The flag played for](flag.md).
- **The fake backend has no arena.** Check it on the local stack.
