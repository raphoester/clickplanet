# A season's map, browsed

**`/play?archive=0` shows the map of a season that ended instead of the game** (`app/archive/SeasonMap.tsx`, entered
from `main.tsx` on `ARCHIVE_PARAM`): no stream, no click, no sign-in. The Hall of Fame's **Map** button, beside View
on each ended plaque, opens it (`archiveUrlOf`).

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
- **Heatmap**: a flat "night lights" map (`heatmap.ts`): every tile a dim dot, so the land draws itself, and the hits
  blurred and coloured from `--panel-raised` through `--war` and `--gold` to `--cream`. The scale is the hottest 0.2%,
  so a few tiles fought over all season do not leave the rest dark. Colours are read from the tokens at draw time.
- **Season / Final Battle** switches the window, **Replay / Heatmap** the view.
- **In fake mode** the history is made up (`fakeHistoryOf`): it ends on each landmass held by its own country.
