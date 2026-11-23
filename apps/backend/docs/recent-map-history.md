# The map's last 24 hours (`GetRecentMapHistory`)

**`ClickService/GetRecentMapHistory` is the world's map over the 24 hours up to the last :00 or :30 UTC, a frame every
30 minutes.** It is public (no token, `NO_SIDE_EFFECTS`, so a GET) and takes no argument:
the window is the same for every caller. The page makes a player's video of its flag's day from it.

## What it answers

`GetRecentMapHistoryResponse { MapHistory history = 1; }`, the message `GetMapHistory` answers (`history.proto`):

- **49 frames**, at `until - 24h`, every 30 minutes, to `until`, the last :00 or :30 UTC at or before now
  (`ledger.RecentAt`). The first frame is whole (`MapFrame.whole`, a `WholeMap`): every tile held at its time, written
  as `GetMap` writes the map, two bytes a tile from the first tile held, each an index into a table of flags
  (`planetmessage.Whole`, since 2026-11-18: with one `TileOwner` a tile the answer was 1.9 MB gzipped on `earth-2`). Each next
  frame is the tiles whose owner differs from the frame before, with `seconds_into_step` (the tile's last change of the step): a take, a tile a spread or an
  enclose took, a bomb's clear (no `country_id`). A tile changed twice in a step shows its last change; a tile changed
  back within a step is in no frame.
- **`hits`**: each tile's changes of owner and shield hits in the window, as in `GetMapHistory`, each naming the era it
  is of (`TileHits.era`): the last of the window's, the board the world plays, which the rewind keeps.
- **`eras`**: the world's eras the window covers, each with its board's name, as in `GetMapHistory` (see [Boards and
  eras](boards-and-eras.md)).
- **No account.** Until 2026-11-12 each change carried the account that made it (`TileOwner.account_id`, `reserved`
  now), for the video's end card, which named the flag's top players. A viewer knew none of them, so the end card names
  the maker alone, and nothing reads who made a change.

## How it is built: rewound from the map in memory

**No replay from the season's reset.** `recent_history_query` reads the map planet holds in memory, then every
`ledger_events` row from the window's start on, and rewinds (`ledger.Rewind`): each tile opens on what it held before
its first change at or after the start (the change's `previous`, or a payload's `owner`), every other tile as it is
now. The changes before `until` make the frames; the ones after it only rewind. `Event.Replay` and `Event.Strikes` read
every kind, so the query asks none: a spread, an enclose and a bomb list their tiles in the payload; a strike, a shield
and a fortify change no owner.

- **The map before the ledger**, as `GetReplay`: a change made between the two reads is in the rows, and rewound.
- **The rows from the ledger's head**: the head (`ledger_head`, the oldest position memory keeps, 72 hours back) bounds
  a range of the primary key. The read is `position >= head`, ordered by position; `plan_test.go` pins the primary key
  and no sort. The rows before the window are read for the rewind to see through a drawing a revert forgot (below).
  `ledger.retention` must keep at least 24.5 hours: `ledger.Config.Validate` refuses less at boot.
- **Postgres, not the ledger in memory.** Memory holds the same 72 hours, and `GetReplay` reads it. It is kept for the
  operator's tools and as the relay's outbox ([Durability](durability.md)); only the map is meant to live in memory,
  so a public read does not add a reader to it.
- **A take a revert forgot is seen through** (`ledger.Erasure`, since 2026-11-23): it is in no frame, a tile painted over
  it opens on what was under it, and a take back to that is none. Until then it was left out (`NOT forgotten`), and a
  drawing painted over before its revert opened whole on the first frame. See [A revert erases the takes it
  forgot](revert-erasure.md).
- **A reset in the window** starts the history at the first step at or after it (`Recent.Wiped`), on the blank map:
  the map before the reset is gone. A reset after `until` leaves no frame at all.
- **A switch of the world's board in the window** starts it the same way, at the first step at or after the start of
  the world's latest era (PR 6 of [Boards](../../../docs/boards.md)): the map before it is on another board, and the
  first frame is the new board's map, rewound from memory as any other. So for the 24 hours after a switch, the page's
  video starts at it. A switch after `until` leaves no frame at all.
- **It is the world's map, on the board it plays**: each row names its map and its board, and an act made on an arena's
  map, or on another board, is left out; while an arena plays it rewinds from the world's map as postgres keeps it
  (`Storage.Kept`), as `GetReplay` does. The frames stand still over a battle. See [Arena
  battles](arenas.md#the-ledgers-readers).

## What cannot be rewound

**A change that wrote no event**: an operator's `RevertPlayer`. A tile it changed inside the window opens as it was
after that change, and the change is in no frame: the revert's takes are forgotten, so the tile shows what was under
them all along. **The last second**, not flushed yet: a tile changed only in it opens as it is now.

## Built once per step

`recent_history_query.PostgresQuery` keeps the last answer with its step and what the reverts forgot
(`forgetting.Forgetting`, read at each call), and builds the next one at the first call after a step or a revert: one
build for every caller (`singleflight`; a caller that leaves does not stop it, 2 minutes at most). A
failed build keeps nothing, and the next call builds again. The answer says `Cache-Control: public, max-age=N,
s-maxage=N`, N the seconds to the next step, so a browser asks again only once the next frame exists. A restart builds
it again at the first call.

## Cost (measured 2026-11-11, 2026-11-12 and 2026-11-18, laptop)

`size_test.go` plays a synthetic day as busy as production's on each world board, `earth-1` and `earth-2` (~110,000
changes of owner by 98 accounts at the same clicks, 200 flags grown over the board's own tiles with 3% of them blank,
fronts that come back) through `Rewind`, and checks that the first frame, read as a page reads it, is the map the day
opened on, and that the frames played after it are the map now. The test fails above 1 MiB gzipped.

| the answer | `earth-1` | `earth-2` |
| --- | --- | --- |
| tiles held in the first frame | 245,798 | 942,345 |
| changes in the other 48 | 85,161 | 102,950 |
| one `TileOwner` a tile (until 2026-11-18) | 3.4 MB, 799 KB gzipped | 10.6 MB, **1,953 KB gzipped** |
| the first frame whole, two bytes a tile | 1.6 MB, **442 KB gzipped** | 3.4 MB, **589 KB gzipped** |

The changes did not grow with the board: they count clicks. With an account on each change it was 6.6 MB and 833 KB on
`earth-1`. Kept with the accounts, the answer held ~34 MB of heap; a whole frame is now one `bytes` of 2 MB on
`earth-2` in place of a million messages. Each call marshals it again: ~0.1-0.2 s on a laptop. A build over 470,000
rows from the head (330,000 of them before the window) took 0.7 s.

## Tests

`ledger/rewind_test.go` (the opening, the payloads of a spread, an enclose and a bomb, a tile changed twice and changed
back, the step a change lands in, an arena), `ledger/recent_test.go` (the window on the UTC clock, a reset in it, the
retention), `recent_history_query`'s postgres suite (the rewind from the rows and the map, a forgotten take, a drawing forgotten
before the day and painted over in it, the step built again after a revert, a reset, a
switch of the world's board in the day and after it, an arena that plays, one build per step for concurrent callers),
`planetmessage`'s `history_test.go` (a whole map's bytes, one with nothing held, a whole frame and a frame of changes),
`plan_test.go`, `size_test.go` on both boards and the handler's test (the header).
