# The battle page

`src/battle/`, for watching a rush live, the Final Battle first: bans as they fall, clicks and refused clicks a
minute, the race, the antibot, the boxes and the server. **The backend sends raw reads, the page counts**, as on [Home](home.md).
Every read refreshes every 15 seconds (`REFRESH_MS`, `usePolling`), never twice at once, each failing alone, the last
answer shown while the next is read. A span is picked above: 30 minutes, 1 hour, 2 hours (first) or 3 hours, in
minutes.

- **Bans** (`BanList`): `planet.v1.AdminService/ListConvictions` over the span, the latest first. Each ban names its
  player (`namesOf`, its flag the country it clicked for most), the watchdogs that fired with their evidence (a
  watchdog that read clear is no reason, and is left out), its offence and end, who it fell with (`alongside`), and
  its clicks. **Open player** opens it on the player page; **Unban account** and **Unban address** lift each key apart,
  since a guest the jury banned is banned on both. The antibot off says so, not "nobody".
- **Clicks a minute** come from the ledger, read on as Home reads it (`readOn` in `home/ledger.ts`, see
  [The home page](home.md)): the first read takes the span, each next one from a minute before the end of the last,
  every page of it. Then `slotsOf`, as on Home, in minutes.
- **Refused a minute** come from the process's counters (`metrics.v1.AdminService/GetMetrics`, `counters.ts`). **The
  history starts when the page opens**: the counters are in process, so the page keeps its own answers for the span
  (`keptSamples`) and subtracts them, spread over the minutes between two reads (`perSlotOf`); a restart (another
  `started_at`) counts from zero. **A reason is every counter named `<reason>_clicks`** (`reasonsOf`): throttled,
  banned, blocked, then any new one, frozen first, with no change here. The tile is the last minute's rate
  (`lastMinuteOf`) and its share of every click the counters saw.
- **The two charts** (`MinuteChart`) share one hover and one scale each: clicks alone, then one line a reason, in the
  palette's slots from the second on (`--series-2` to `--series-5`), with a legend and a table. A minute no two reads
  span has no point.
- **Race** (`RacePanel`): `seasons.v1.AdminService/GetRace`, the round's standings and the season's points, as the
  players see them.
- **Antibot**, **Boxes**, **Server** (`CounterPanels`): callers each watchdog reads at suspect and certain now
  (`antibot_opinions_standing`) and the bans each argued for (`ban_flags`); the boxes offered, caught, lapsed and
  claimed for another a minute, and the share caught within a second (`bonus_catch_seconds`); the process's CPU,
  memory, goroutines, the events between modules not read yet (`events_lag`) and those lost
  (`events_unpublished_total`, `events_dead_total`). **Every counter** lists the rest.
- **Playing now** is Home's (`PlayingNow`).
