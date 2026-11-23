# A season's map, browsed

**`/archive` is a page of its own** (`archive.html`, entry `src/archive.tsx`, `app/archive/SeasonMap.tsx`): the map of
a season that ended, with no game, stream, click or sign-in. It opens on the latest season that ended, from the
gamemaster's timeline, and a switch picks another once there are several. The Hall of Fame's **Map** button, beside
View on each ended plaque, opens it (`ARCHIVE_PATH`).

- **The history is the gamemaster's `GetSeasonReplay`** (`TurnBackend.seasonReplay`): the map every 30 minutes over
  the season, or every 10 seconds over its Final Battle, and each tile's changes and shield hits (see the backend's
  [The gamemaster](../../backend/docs/gamemaster.md)). Its first read after a deploy replays planet's whole ledger, so
  the client allows it 5 minutes and shows "Rebuilding the map…". It is the world's map: over an arena battle it
  stands still and no tile heats (see the backend's [Arena battles](../../backend/docs/arenas.md#the-ledgers-readers)).
  A take a revert forgot is in no frame and heats no tile, and the browser reads the map again within the half hour
  (the backend's [A revert erases the takes it forgot](../../backend/docs/revert-erasure.md)).
- **`MapReel` (`domain/mapHistory.ts`) gives any frame's map**: a copy of the owners every 32 frames, a `Uint16Array`
  of country indexes, so a seek back costs a copy and a few frames.
- **Each era is drawn on its own board** (`domain/eras.ts`). The history lists the eras of the world's map it covers
  (`MapHistory.eras`), and a frame's tiles are of the board of the era at its time. `piecesOf` cuts the frames by era,
  and `ErasReel` keeps a `MapReel` per era, of its own board's tiles; the first frame of an era after the first holds
  every tile held on its board, as the window's first frame does: **the cut**, where the map is cleared and drawn whole.
  The page reads each board an era plays once (`archive.tsx`'s `geometry(board)`). With one era, the only one until the
  world's switch, it is exactly the replay of before.
- **Replay**: the ordinary globe, fed by `HistoryBackend` (`backends/historyBackend.ts`), which hands over the tiles
  that differ from the frame shown on each seek, as the live stream would. **Past a cut the globe is built again on the
  next board**, in place, with a `HistoryBackend` of that era's reel: the archive cannot reload, as the game does (see
  [The board the page draws](board-reload.md)). The time cursor carries a mark at each cut (white, inked, its tooltip
  "New map from …"), and "The world moves to a new map" shows over the globe for the 36 frames from it (a second and a
  half of play: `ErasReel.passingCut`, tested in `eras.test.ts`). **At the world's switch the first frame of the new
  board is the map the switch carried over** (the backend's [Boards and eras](../../backend/docs/boards-and-eras.md)),
  sent whole as `GetMap` sends the map, two bytes a tile, which `historyOf` reads. A time cursor, Play (24 frames a second:
  the season in about 48 s, the Final Battle in 30 s), the time in the player's zone, and the five flags holding the
  most tiles, from the globe's own leaderboard.
- **Heatmap**: a flat map (`heatmap.ts`), 4096 × 2048, each tile a sharp cell as tall as a tile (0.4° on `earth-1`, as
  much more as a board's tiles are apart: `cellDegreesOf`) and as wide as its latitude stretches it. **Each era's hits are
  laid where they were made** (`TileHits.era`, `heatOf`): the cells are the tiles of the finest board among the eras, and
  each takes the hits of the tile of each era's board its centre is on, found by `tileAt` on that board's lattice, so
  the eras add up in one geographic grid. One era is drawn as it is. Each cell is dim for a tile nobody fought over,
  else coloured by its hits from `--war` through `--gold` to `--cream`. The scale is the hottest 0.5%, so a few tiles fought over all season do not leave the rest dark. Colours
  are read from the tokens at draw time. Longitude 0 is the middle (`placeOf`: north is y, longitude 0 is -x, as the
  backoffice reads it). **The map goes round**: three copies side by side, the scroll jumping a whole map at either
  edge, opened on longitude 0; a drag, the wheel or a swipe scrolls it sideways.
- **Season / Final Battle** switches the window, **Replay / Heatmap** the view.
- **In fake mode** the history is made up (`fakeHistoryOf`): it ends on each landmass held by its own country.
