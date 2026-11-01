# A season's map, browsed

**`/archive` is a page of its own** (`archive.html`, entry `src/archive.tsx`, `app/archive/SeasonMap.tsx`): the map of
a season that ended, with no game, stream, click or sign-in. It opens on the latest season that ended, from the
gamemaster's timeline, and a switch picks another once there are several. The Hall of Fame's **Map** button, beside
View on each ended plaque, opens it (`ARCHIVE_PATH`).

- **The history is the gamemaster's `GetSeasonReplay`** (`TurnBackend.seasonReplay`): the map every 30 minutes over
  the season, or every 10 seconds over its Final Battle, and each tile's changes and shield hits (see the backend's
  [The gamemaster](../../backend/docs/gamemaster.md)). Its first read after a deploy replays planet's whole ledger, so
  the client allows it 5 minutes and shows "Rebuilding the map…".
- **`MapReel` (`domain/mapHistory.ts`) gives any frame's map**: a copy of the owners every 32 frames, a `Uint16Array`
  of country indexes, so a seek back costs a copy and a few frames.
- **Replay**: the ordinary globe, fed by `HistoryBackend` (`backends/historyBackend.ts`), which hands over the tiles
  that differ from the frame shown on each seek, as the live stream would. A time cursor, Play (24 frames a second:
  the season in about 48 s, the Final Battle in 30 s), the time in the player's zone, and the five flags holding the
  most tiles, from the globe's own leaderboard.
- **Heatmap**: a flat map (`heatmap.ts`), 4096 × 2048, each tile a sharp cell as tall as a tile and as wide as its
  latitude stretches it: dim for a tile nobody fought over, else coloured by its hits from `--war` through `--gold` to
  `--cream`. The scale is the hottest 0.5%, so a few tiles fought over all season do not leave the rest dark. Colours
  are read from the tokens at draw time. Longitude 0 is the middle (`placeOf`: north is y, longitude 0 is -x, as the
  backoffice reads it). **The map goes round**: three copies side by side, the scroll jumping a whole map at either
  edge, opened on longitude 0; a drag, the wheel or a swipe scrolls it sideways.
- **Season / Final Battle** switches the window, **Replay / Heatmap** the view.
- **In fake mode** the history is made up (`fakeHistoryOf`): it ends on each landmass held by its own country.
