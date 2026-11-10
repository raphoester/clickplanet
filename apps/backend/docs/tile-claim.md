# A claim is one lock

**A tile's owner, shields and holder are one aggregate, `clicks.Tile`, and a player's claim changes them in one step.**
`Tile.Claimed(flag, by)` is the rule: a tile that wears the flag is `Unchanged`; a tile with a shield is `Shielded`,
one shield struck, the flag and the holder kept; any other tile is `Taken`, with the flag, no shield, and held by `by`.

- **Read, decide and write under one lock.** `clicks.Claiming` opens `TileStorage.Claim` (a click, a spread's
  neighbour) or `Enclose` (every tile inside), which holds the tiles' write lock (`s.mu`) for all of it: read the tile
  (`Claimable.Tile`), ask `Claimed`, write the tile it answers (`Set`, or `Click` for the tile a click named), and
  after a take ask `Fortify`. Each write publishes under that lock, so the streams still get the versions in order
  (see [The map's version](map-version.md)).
- **What it rules out.** Two clicks for one flag on one tile: the second reads the tile taken and is `Unchanged`, so
  the ledger keeps one take and the relay tells one `TilesTaken`. A shield placed while a claim decides waits for the
  write, then the claim has struck it or the placement is refused (`ErrNotYourTile`); a take never wipes a shield it
  did not see. The `previous` the ledger keeps is the owner the write replaced.
- **The adapter decides nothing.** `inmemory_tile_storage` reads a `TileSnapshot` from its board and writes the one it
  is given. A write equal to the tile is no write: nothing published, no version counted. A change of owner moves
  the landmass and flag counts (`putLocked`). A bomb's `Clear` and the operator's `Reassign` and `Restore` are one lock
  each too, with their own rule: the shields go home and nobody holds the tile.
- **A lock, not a version per tile.** Every write already takes `s.mu` alone, to publish in version order. The read
  and the rule add well under a microsecond to that hold (a claim is ~0.4 µs on a laptop). A compare-and-set on a
  version per tile still needs the lock to write and publish, and adds a retry loop and 4 bytes per tile. It wins
  only when the decision is slow or the map is shared by processes, and the map lives in one process (see
  [Durability](durability.md)).
- **Measured on production, 2026-11-10** (3.5 days of `ledger_events`, 36 minutes of Caddy's log): 128,000 takes and
  61,000 strikes a day, at most 46 a second on the whole map and 3 in the median second. The race left 9 double takes
  (2.5 a day, 0.002% of takes), 8 of them one account tapping twice. A click answers in 3.8 ms at p50 and 24.5 ms at
  p99, so a few microseconds under the lock cost nothing. The ledger keeps whole seconds
  (`inmemory_ledger_storage.seconds`), so it cannot show two takes closer than one second.

**Tests.** `tile_test.go` pins the rule. `claiming_test.go`: eight claims at once for one flag take the tile once;
a shield placed during a foreign claim is struck or refused, never lost (the old race shows there only under `-race`).
The contract suite: a shield placed while a claim decides waits for its write. `relay_ledger_usecase`: eight clicks at
once tell one `TilesTaken`.
