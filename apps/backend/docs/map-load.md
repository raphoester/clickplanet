# The map load

`GetMap` is an ordinary RPC, but marked `idempotency_level = NO_SIDE_EFFECTS` in the proto, so Connect sends it as an **HTTP GET** and the cache interceptor sets `Cache-Control: public, max-age=5, s-maxage=5` and an `ETag` of the batch's version on the response. `GetHoldings`, what each flag holds of the map and of each landmass, is served the same way; see [The map's version](map-version.md). A burst of visitors can therefore share one origin response; `ListenForEvents` carries everything that happens after a chunk was built, so a client starting from a slightly old map converges anyway. `MapDensity` is marked the same way.

The response never repeats a tile id. `GetMapResponse` carries `start_tile_id`, the interned `codes` table, and `tiles` — a `bytes` field holding two bytes per tile, little endian, indexing into `codes`. Tile ids are implicit in the position, which is what makes it far smaller than the deprecated `map<uint32, string>`: **524 KB against 3.6 MB** for a full 262,119-tile map.

`inmemory_tile_storage.StateBatchDense` builds it, and stamps it with the version it read under the same lock. The interned ids are copied out exactly as stored and the table travels with them, so nothing is translated on the way out and the client needs no shared country list. Protobuf does all the framing — there is no hand-rolled magic or length-prefixing on either side, and therefore no encoder and decoder that have to be edited together.

**Secondary (output):**
- `clicks/inmemory_tile_storage/` — the tile map. A preallocated `[]uint16` indexed by tile id, with country codes interned into a side table, and a `[]uint8` of each tile's shields beside it (3 bytes per tile — ~3 MB for a 1M-tile map). Fans updates out in process, and flushes the tiles that changed through its `Persistence` port. **`Storage` only holds the lock, the version and orders five objects**, each with one job: `board` (the tiles, the `codebook` they are written in, and what each country holds), `landmasses` (what each flag holds of each landmass, and the flag it is locked to), `feed` (the open streams), a `dirtySet` in each of the first two (what the next flush writes) and `recount` (the landmasses and flags the next tally tells). A change of owner goes through `Storage.moveLocked`, which moves the board and the landmass count together.
- `clicks/postgres_tile_store/` — that port, over the `planet.tiles` table. See [Durability](durability.md).
- `ledger/inmemory_ledger_storage/` — the ledger, in memory, flushed through its own `Persistence` port.
- `ledger/postgres_ledger_store/` — that port, over `planet.ledger_events`, `ledger_head` and `ledger_forgotten`, and `AnonymizeTakes` for a deleted account.
- `bonuses/postgres_charge_store/` — `bonuses.Store`, the charges each account holds, over `planet.charges`, with no copy in memory.
- `clicks.Board` (not an adapter) — validates tile IDs
- country codes are validated by `shared/cpcountries`, which chat shares — see [The composite layer](composite-layer.md)

`inmemory_tile_storage` implements `clicks.TileStorage`, including `Subscribe(ctx) (<-chan clicks.Change, error)`, one call per open stream. A `Change` is a tile update, a bomb blast, a fortify or a tally, with its version, on one channel so they keep their order — see [What a bomb does](bomb.md).
