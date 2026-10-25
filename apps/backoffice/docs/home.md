# The home page

`src/home/`, the page the backoffice opens on. **The backend sends what happened and what is, never a chart**: the
page counts, charts and draws on its own side, so a new view of the game needs no new endpoint. It reads, each
failing alone:

- `planet.v1.AdminService/GetLedger` for the span: every change the live stream showed, as the `TileUpdate` it sent,
  with when and who (the backend's [GetLedger](../../backend/docs/admin.md#getledger-getterritories)). It is asked for whole slots, from a step back to now
  (`windowOf`, both bounds sent): the last one starts on the step before now, and is still filling. The page counts
  from the window it asked for. `LONG_CALL_MS`: three days is a
  few MB compressed.
- `GetTerritories` for the map now: the tiles each country holds.
- `player.v1.AdminService/GetRoster` for who plays now.
- `ListReports` (open only) and `ListNotes` (the last 24 hours) for what waits for the operator. Up to 500 of each
  are counted, and 500 reads "500+".
- **The tile coordinates** (`places.ts`), the map's own blob from `/map` (`npm run map` copies it into `src/gen/map`,
  and Vite bundles it under `/assets/`): where each tile is, read once a page load, about 5 MB. Its latitude and
  longitude are read as the map's generator reads a vertex: `asin(y)` and `atan2(z, -x)`. The CSP's `connect-src`
  holds `'self'` for it.

What the page does with them:

- **A span is picked above everything it scopes**: 6 hours in slots of 5 minutes, 24 hours (first) in 15, or 3 days
  in hours. Every minute the page reads it all again (`REFRESH_MS`), and Reload does now. **The last answer stays
  shown, dimmed, while the next is read** (`useLast`), so a refresh never empties the page.
- **The counts** (`ledger.ts`): a click is a change marked `clicked` (a take or a strike); a tile changed hands when
  its flag after is not its flag before; a player is who changed one (`by`, compared and never read), an account on
  every address, or an address with no account.
- **The tiles**: who plays now (and how many signed in), the clicks and the players of the span, the open reports and
  the notes of the day. The last two go to their page.
- **Two charts of one series each** (`ActivityCharts`): the clicks a slot, and the players taking tiles a slot. Never
  one chart with two scales. A hover on either reads the same slot on both, in the line above them; the peak is
  labelled; the slot still filling is a hollow dot, off the line, since its count is short. "Table" lists every slot.
- **The heatmap** (`Heatmap`, `heat.ts`): the map seen flat, 1440×720. Each tile is drawn where it is, as wide as
  its longitude stretches far north; what changed hands glows over the land around it (a box blur, twice, round the
  world from east to west), on a log scale up to the hottest pixel. The colors are tokens (`--map-*`, `--heat-0` to
  `--heat-3`), read when it draws, and it draws again when the color scheme changes. The line under it names the
  degree with the most changes, or the degree under the pointer, with its tiles.
- **The countries** (`CountryTable`): what each holds now, its share of the map, and the tiles it won and lost by play
  in the span. Held, Won and Lost order them, and **the bars are a histogram of the column picked**. The ten first,
  then all.
- **Playing now** (`PlayingNow`): each signed-in player, which opens it on the player page (`OpenPlayer`), the guests
  as a count, and the flags they play for.
- **What it does not show yet**: HTTP requests. The clicks come from the ledger, which keeps 72 hours and survives a
  restart; the backend's request counters are in-process, and the metrics poller's history is files on the server.
