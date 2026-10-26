# The map's version, and the counts it orders

**Every write of the map counts one, and every answer and every live event says where it was.** `clicks.MapVersion`
is `{epoch, seq}`: `seq` counts the writes since the process started, and `epoch` is the second it started, since
`seq` starts from zero at every boot. On the wire it is `planet.v1.MapVersion`, on `GetMapResponse`,
`GetHoldingsResponse`, `TileUpdate`, `BombDropped`, `LandmassFortified`, `HoldingsChanged` and `Heartbeat`. A client
compares two versions of one epoch; a version of another epoch means the server restarted, and what it holds is to
be read again.

## One counter, under the tiles lock

- **`inmemory_tile_storage` keeps the counter beside the board and the landmasses**, under the same `s.mu`.
  `publishLocked` raises it and stamps the `clicks.Change` it publishes, so a write that changes nothing (a click on
  a tile already its flag, a strike with no shield left) counts nothing. A bomb, a fortify and each tile of a reassign
  or a restore count one each.
- **Every publish runs under `s.mu`**, in `put`, `Clear`, `Strike`, `Shield`, `Fortify`, `Reassign` and `Restore`. So
  every stream gets the changes in the order of their versions: two writes cannot reach a channel in the other order.
  `feed.publish` never blocks (a full channel is cut, see [The live streams](live-streams.md)), about 1-2 µs for 50 to
  100 streams, so the lock is held no longer than a click already holds it. The lock order is `s.mu` then `feed.mu`;
  nothing takes them the other way. The "cut off" Warn is the one write to stdout, so the feed only counts the cut
  streams, and the census tick logs them after it lets the lock go: at most one line a second.
- **`Subscribe` answers the version too**, read under the same lock as the subscription is made: every change after
  that version is on the channel, and none before it. The stream's opening heartbeat carries it.
- **`StateBatchDense` reads it under its read lock**, so a `GetMap` batch is the map at exactly its version.
- **No write is told twice and none is kept**: there is no replay buffer. A client that misses a change reads again.

## The census: what each flag holds

`Storage.Census()` walks `board.held` and `landmasses.held` under one read lock: every flag that holds a tile, and
every landmass with a tile held, whole (a flag a row does not list holds none of it), at the version it read.
`ClickService/GetHoldings` serves it (`get_holdings_handler/holdings_query`, a `MemoryQuery`). It is not
`HoldingsOf`, which is the accounts holding one landmass (see [Who holds the tiles](tile-holders.md)).

**`holdings_changed` carries what changed, at most once a second.** `moveLocked` marks the landmass and the two
flags of every change of owner in a `recount` (two `dirtySet`s, nothing else in memory). `Storage.Run` starts a
one-second tick, `PublishCensus`, that drains them under the write lock and publishes one `clicks.Change{Census}` with
the whole row of each landmass marked and the total of each flag marked, at the current version. It counts no write
of its own. A flag that left the map is listed with no tile, and a landmass nobody holds any more with no flag.
**Always totals, never "+1"**: a bomb, an enclosure (one put per tile, up to 75 per click) and an operator's batch
(256 puts every 50 ms) are a single event at the next tick, and a missed one is healed by the next.

Sizes, read on production on 2026-10-26: 35 flags hold tiles, 737 (landmass, flag) pairs over the 658 landmasses,
617 landmasses held by one flag and the most by 13 (Kazakhstan). The whole `GetHoldings` is about 9.5 KB before
compression, and one row is 14 to 111 bytes.

## Caching by version

`GetHoldings` and `GetMap` are GETs (`NO_SIDE_EFFECTS`) with `Cache-Control: public, max-age=5, s-maxage=5` and an
`ETag` of their version, `"<epoch>.<seq>"` (`planetv1controller.NewCacheInterceptor`). The interceptor answers an
`If-None-Match` that names the map's current version with a 304 before the handler runs, so nothing is read and no
`MapRead` is told. It compares weakly, as RFC 9110 asks: Cloudflare turns a tag weak (`W/`) when it compresses. Caddy
adds `-zstd` or `-gzip` to a strong tag it compresses and takes it off the `If-None-Match` it passes on, so the
backend sees its own tag.

## Tests

The contract suite pins that each write counts one after the subscription's version, that a no-op counts nothing,
that concurrent writes reach every subscriber in version order, and that a batch holds the map as it was at its
version. `census_test.go` pins that after a take, a bomb, an enclosure and a reassign the published rows equal a
`Census()` read right after, and list every row that changed; and that a fortify, which moves no count, publishes
none. `cache_interceptor_test.go` pins the tag, the 304 and the weak match.
