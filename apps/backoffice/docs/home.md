# The home page

`src/home/`, the page the backoffice opens on. **The backend sends what happened and what is, never a chart**: the
page counts, charts and draws on its own side, so a new view of the game needs no new endpoint. It reads, each
failing alone:

- `planet.v1.AdminService/GetLedger` for the span: every change the live stream showed, as the `TileUpdate` it sent,
  with when and who (the backend's [GetLedger](../../backend/docs/admin.md#getledger-getterritories)). It is asked for whole slots, from a step back to now
  (`windowOf`, both bounds sent): the last one starts on the step before now, and is still filling. The page counts
  from the window it asked for. **It reads every page of the answer, one after the other** (`next_page`, 20,000
  events a page: 10 pages for 24 hours), each with `LONG_CALL_MS`.
- `GetTerritories` for the map now: the tiles each country holds.
- `player.v1.AdminService/GetRoster` for who plays now.
- `ListReports` (open only) and `ListNotes` (the last 24 hours) for what waits for the operator. Up to 500 of each
  are counted, and 500 reads "500+".
- **Where each tile is**, on the board the ledger names (`GetLedgerResponse.board`, the board the world plays), read
  once that answer is in and kept for the page load (`loadPlaces`, see [Boards on the page](boards.md)). A board the
  page does not carry is said where the heatmap would be, and the counts, the charts and the countries still show.

What the page does with them:

- **A span is picked above everything it scopes**: 6 hours in slots of 5 minutes, 24 hours (first) in 15, or 3 days
  in hours. **The last answer stays shown, dimmed, while the next is read** (`useLast`), so a refresh never empties
  the page.
- **A refresh reads the ledger's new changes, not the span again** (`sinceOf`, `caughtUp` in `ledger.ts`): from a
  minute before the end of the last read (`SETTLE_MS`, since a change reaches postgres a flush after it happened, later
  when a flush fails), and the page keeps what it read before that, inside the span. A new span, or a pause longer than
  the span, reads it whole. **On 2026-10-25 a tab left open read 24 hours every minute**, 7 to 8s each on the 1 GB
  droplet, until the backend swapped and every call ran its minute: clicks took 300 to 600ms for a quarter of an hour.
  The pages bound what one call holds on the server; this bounds how often the page asks.
- **A read that finds the world on another board reads the span whole again** (`addsTo`, `boardOfPages`): `GetLedger`
  answers the changes of the board the world plays alone, so after a switch what was kept is of another board's ids.
  The page drops it, reads the span from its start on the new board, and places the tiles on that board. A switch
  between two pages of one read does the same; three reads in a row that each meet a switch give up and say so.
- **The next read is `REFRESH_MS` (a minute) after the last one ended**, never while one runs: Reload waits for it
  too. None is made while the tab is hidden (`usePageVisible`); a tab shown again after its minute reads at once.
- **The counts** (`ledger.ts`): a click is a change marked `clicked` (a take or a strike); a tile changed hands when
  its flag after is not its flag before; a player is who changed one (`by`, compared and never read), an account on
  every address, or an address with no account.
- **The tiles**: who plays now (and how many signed in), the clicks and the players of the span, the open reports and
  the notes of the day. The last two go to their page.
- **Two charts of one series each** (`ActivityCharts`): the clicks a slot, and the players taking tiles a slot. Never
  one chart with two scales. A hover on either reads the same slot on both, in the line above them; the peak is
  labelled; the slot still filling is a hollow dot, off the line, since its count is short. "Table" lists every slot.
- **The heatmap** (`Heatmap`, `heat.ts`): the map seen flat, 1440×720. Each tile is drawn where it is, as wide as
  its longitude stretches far north, its width the board's spacing (`TILE_DEGREES`, 0.3° on `earth-1`, read through
  the board's scale: see [Boards on the page](boards.md#sizes-read-the-board)), on the pixel row under it and, on a board
  coarser than `earth-1`, the rows within half its width; what changed hands glows over the land around it (a box blur, three times, round the
  world from east to west), on a square root up to the hottest pixel. The colors are tokens (`--map-*`, `--heat-0` to
  `--heat-3`), read when it draws, and it draws again when the color scheme changes. The line under it names the
  degree with the most changes, or the degree under the pointer, with its tiles.
- **The countries** (`CountryTable`): what each holds now, its share of the map, and the tiles it won and lost by play
  in the span. Held, Won and Lost order them, and **the bars are a histogram of the column picked**. The ten first,
  then all.
- **Playing now** (`PlayingNow`): each signed-in player, which opens it on the player page (`OpenPlayer`), the guests
  as a count, and the flags they play for.
- **What it does not show yet**: HTTP requests. The clicks come from the ledger, which keeps 72 hours and survives a
  restart; the backend's request counters are in-process, and the metrics poller's history is files on the server.
