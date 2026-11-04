# Ghosts, for a quiet first visit

**While few people play, a guided page sees the map fight back near it.** From Oct 23 to Nov 3 2026, 31% of new
players left within 2 minutes, 2 of 3 arrived when fewer than 7 played, and with 12 or more online they stayed 5
times longer. A gamemaster army on the server would break the game's balance, so the fight is fake and painted on the
newcomer's page only. The newcomer must not notice it: every reading on the page agrees with it.

- **Who sees them**: a guided page (`showFront`, see [The front, for a first visit](front-guide.md)) once every tile
  is read, while the "Online" list holds fewer than 7 (`GHOSTS_BELOW` in `globe.ts`). `Viewer` reads the list's
  length from the roster's store (`onlineCount`) and passes it to `useGlobe` (`crowd`), which hands it to
  `Globe.setCrowd`; no list yet means no raid. Never in orbit, never on a frozen map. When the room fills, the
  raids stop and the ghost tiles stay until they end.
- **A raid** (`domain/ghostArmy.ts`, `GhostArmy`): within what the screen shows (70% of its shorter half around its
  middle, `GHOSTS_SIGHT`), it lands on a tile of the player's flag that touches a tile of another flag or a free one,
  3 times in 4 when there is one, else on the edge of another flag. It plays the flag next door, else the nearest
  other country, by the flag on its tiles or by its ground; never the player's flag. It taps 3 to 8 tiles 0.35s to
  1.1s apart, each touching one it took, the player's tiles first, then pauses 6s to 16s. It stops at once when the
  player takes back one of its tiles. The first raid comes 3s to 8s after the front is lit.
- **What a ghost takes**: a tile a flag holds, with no shield. A tile a ghost leaves goes back to its holder, and a
  tile going back to free would look like a bomb. At most 40 ghost tiles at once, and at most a quarter of the
  player's flag, 12 tiles at most.
- **The local state** (`domain/ghosts.ts`, `Ghosts`): each take has its own end, 1 to 3 minutes after it, so the
  tiles go back one by one, each with a click's glint, as a retake would. It is kept in `clickplanet-ghosts`
  (`ghostStore.ts`), so a reload paints the same tiles until they end, even on a page no longer guided.
- **Paint only**: a ghost is not in `MapTiles` nor in `TileOwnership`, so no version, read or rollback mixes with it.
  A read paints under a ghost (`applyChanges`, not live). Anything else the server says about the tile wins: an
  update, a bomb, a fortify of its landmass, the player's own click. A ghost on a tile the load reads as free is
  dropped once the load ends.
- **What reads them**: the tiles drawn and the front's rings (`Front` reads `shownOwnerOf`), and the board:
  `publishBoard` moves each ghost tile from its holder on the server to the ghost's flag (`Ghosts.countsOver`). So
  the board, its "+1" and "-1", the share in "Your clicks" and the under-attack moment agree with the map. The
  territory card, the flags painted in orbit and the fortify targets read the server alone.
- **A tap on a ghost tile the player's flag holds on the server is answered here**: the tile is drawn as the
  player's again, with the click's sound and glint. Nothing is sent, no click is spent, and it is no guided take. An
  empty bank refuses it as it refuses any tap. A tap on a ghost tile of another flag is a real click: the ghost is
  dropped first.
- **No hold card on a ghost tile**: the server would name the real holder. No fake chat, name or "Online" line.
- **Trying it in fake mode**: clear the page's storage, reload, and `fakeBackend.takeLandmass(39, "bg", 120)` in
  the console. A few seconds later French ghosts take Bulgarian tiles at the edge of Romania;
  `localStorage.getItem("clickplanet-ghosts")` lists them. The fake roster is mostly under 7. A reload empties the
  fake's map, but not the ghosts.
