# Rolling back a refused click

A click is painted before the server has agreed to it, and the server refuses
plenty of them — a spent bucket, a blocked VPN, a session it would not mint. The
paint has to come back off, or a throttled player spends the rest of the session
looking at tiles nobody gave them. The leaderboard never counts it: the counts
are the server's.

`TileOwnership.applyOptimistic` paints and hands back a **claim**; `rollback`
takes that claim and returns the tile to what it would hold had the click never
happened. `globe.ts` calls it from the same `catch` that raises the dialog, and
feeds what comes back through `applyChanges` — the same path a batch or a live
update takes, so the repaint needs no special case.

What makes this more than an undo is everything that can happen to a tile while
a click is in flight, and the rule is the same in every case: **a rollback only
ever takes back what that click itself painted.**

- **The server settled the tile** — its echo, or someone else's claim, arrived
  first. `set` drops the claim, and the refusal that lands afterwards
  changes nothing. A late refusal of a click the server did in fact record
  therefore costs nothing either.
- **A later click on the same tile is still in flight.** It owns the paint, so
  rolling back the earlier one leaves the map alone. Refuse them all and the
  tile ends up where it started, whatever order the refusals arrive in — each
  claim remembers what it painted, so it falls back to the newest one left.
- **A batch landed while the click was in flight.** It cannot paint over the
  optimistic claim, but it is what the rollback restores instead of the stale
  value from before the click. `MapTiles` hands it over only when it is newer
  than what the tile last took, so it never restores a value older than one the
  stream said.

`OwnerChange.country` is `string | undefined` for this: `undefined` is a tile
going back to unowned, which `TileField` writes as a zero-sized atlas region —
what the fragment shader already draws as an unclaimed tile.

**A click predicted to be shielded paints nothing**, so it has no claim to
take back.

Rolling back on *any* failure, including a transport fault, is deliberate: if
the click did land and only the response was lost, the stream's echo repaints
it, and if the echo arrives first the rollback is already a no-op. A transport
fault rolls back with no sound and no dialog; the card from
[Connection health](backends.md#connection-health) is what says why.
