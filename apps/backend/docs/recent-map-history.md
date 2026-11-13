# The map's last 24 hours (`GetRecentMapHistory`)

**`ClickService/GetRecentMapHistory` is the world's map over the 24 hours up to the last :00 or :30 UTC, a frame every
30 minutes.** It is public (no token, `NO_SIDE_EFFECTS`, so a GET) and takes no argument:
the window is the same for every caller. The page makes a player's video of its flag's day from it.

## What it answers

`GetRecentMapHistoryResponse { MapHistory history = 1; }`, the message `GetMapHistory` answers (`history.proto`):

- **49 frames**, at `until - 24h`, every 30 minutes, to `until`, the last :00 or :30 UTC at or before now
  (`ledger.RecentAt`). The first frame is every tile held at its time. Each next frame is the tiles whose owner differs
  from the frame before, with `seconds_into_step` (the tile's last change of the step): a take, a tile a spread or an
  enclose took, a bomb's clear (no `country_id`). A tile changed twice in a step shows its last change; a tile changed
  back within a step is in no frame.
- **`hits`**: each tile's changes of owner and shield hits in the window, as in `GetMapHistory`.
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
- **The rows from the ledger's head**: `taken_at` has no index, and the head (`ledger_head`, the oldest position memory
  keeps, 72 hours back) bounds a range of the primary key. The read is `position >= head AND taken_at >= since`,
  ordered by position; `plan_test.go` pins the primary key and no sort. So `ledger.retention` must keep at least
  24.5 hours: `ledger.Config.Validate` refuses less at boot.
- **Postgres, not the ledger in memory.** Memory holds the same 72 hours, and `GetReplay` reads it. It is kept for the
  operator's tools and as the relay's outbox ([Durability](durability.md)); only the map is meant to live in memory,
  so a public read does not add a reader to it.
- **A take a revert forgot is left out** (`NOT forgotten`), as memory replays none of them and `GetTileHolder` skips
  them.
- **A reset in the window** starts the history at the first step at or after it (`Recent.Wiped`), on the blank map:
  the map before the reset is gone. A reset after `until` leaves no frame at all.
- **It is the world's map**: an act made while an arena played is left out, and while an arena plays it rewinds from
  the world's map as postgres keeps it (`Storage.Kept`), as `GetReplay` does. The frames stand still over a battle. See
  [Arena battles](arenas.md#the-ledgers-readers).

## What cannot be rewound

**A change that wrote no event**: an operator's `RevertPlayer`. A tile it changed inside the window opens as it was
after that change, and the change is in no frame. A revert's takes are
forgotten, so a tile a reverted account took and someone else took after opens on the reverted account's flag. **The
last second**, not flushed yet: a tile changed only in it opens as it is now.

## Built once per step

`recent_history_query.PostgresQuery` keeps the last answer with its step, and builds the next one at the first call
after a step: one build for every caller (`singleflight`; a caller that leaves does not stop it, 2 minutes at most). A
failed build keeps nothing, and the next call builds again. The answer says `Cache-Control: public, max-age=N,
s-maxage=N`, N the seconds to the next step, so a browser asks again only once the next frame exists. A restart builds
it again at the first call.

## Cost (measured 2026-11-11 and 2026-11-12, laptop)

`size_test.go` plays a synthetic day as busy as production's (~111,000 changes of owner by 98 accounts, the real map of
260,503 tiles, fronts that come back) through `Rewind`, and checks that the first frame is the map the day opened on and
that the frames played after it are the map now. The answer: 244,540 tiles in the first frame (386 KB gzipped), 85,528
changes, 3.4 MB of binary, **794 KB gzipped**; the test fails above 1 MiB. With an account on each change it was 6.6 MB
and 833 KB. Kept with the accounts, the answer held ~34 MB of heap (less now, not measured again), and each call
marshals it again: ~0.1-0.2 s on a laptop. A build over 470,000 rows from the head (330,000 of them before the window)
took 0.7 s.

## Tests

`ledger/rewind_test.go` (the opening, the payloads of a spread, an enclose and a bomb, a tile changed twice and changed
back, the step a change lands in, an arena), `ledger/recent_test.go` (the window on the UTC clock, a reset in it, the
retention), `recent_history_query`'s postgres suite (the rewind from the rows and the map, a forgotten take, a reset, an
arena that plays, one build per step for concurrent callers), `plan_test.go`, `size_test.go` and the handler's test
(the header).
