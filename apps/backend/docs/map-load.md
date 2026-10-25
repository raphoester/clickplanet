# The map load

`GetMap` is an ordinary RPC, but marked `idempotency_level = NO_SIDE_EFFECTS` in the proto, so Connect sends it as an **HTTP GET** and the handler sets `Cache-Control: public, max-age=5` on the response. A burst of visitors can therefore share one origin response; `ListenForEvents` carries everything that happens after a chunk was built, so a client starting from a slightly old map converges anyway. `MapDensity` is marked the same way.

The response never repeats a tile id. `GetMapResponse` carries `start_tile_id`, the interned `codes` table, and `tiles` — a `bytes` field holding two bytes per tile, little endian, indexing into `codes`. Tile ids are implicit in the position, which is what makes it far smaller than the deprecated `map<uint32, string>`: **524 KB against 3.6 MB** for a full 262,119-tile map.

`inmemory_tile_storage.StateBatchDense` builds it. The interned ids are copied out exactly as stored and the table travels with them, so nothing is translated on the way out and the client needs no shared country list. Protobuf does all the framing — there is no hand-rolled magic or length-prefixing on either side, and therefore no encoder and decoder that have to be edited together.

**Secondary (output):**
- `clicks/inmemory_tile_storage/` — the tile map. A preallocated `[]uint16` indexed by tile id, with country codes interned into a side table, and a `[]uint8` of each tile's shields beside it (3 bytes per tile — ~3 MB for a 1M-tile map). Fans updates out in process, and flushes the tiles that changed through its `Persistence` port. **`Storage` only holds the lock and orders four objects**, each with one job: `board` (the tiles, the `codebook` they are written in, and what each country holds), `landmasses` (what each flag holds of each landmass, and the flag it is locked to), `feed` (the open streams) and a `dirtySet` in each of the first two (what the next flush writes). A change of owner goes through `Storage.moveLocked`, which moves the board and the landmass count together.
- `clicks/postgres_tile_store/` — that port, over the `planet.tiles` table. See [Durability](durability.md).
- `ledger/inmemory_ledger_storage/` — the ledger, in memory, flushed through its own `Persistence` port.
- `ledger/postgres_ledger_store/` — that port, over `planet.ledger_events`, `ledger_head` and `ledger_forgotten`, and `AnonymizeTakes` for a deleted account.
- `bonuses/postgres_charge_store/` — `bonuses.Store`, the charges each account holds, over `planet.charges`, with no copy in memory.
- `clicks.Board` (not an adapter) — validates tile IDs
- country codes are validated by `shared/cpcountries`, which chat shares — see [The composite layer](composite-layer.md)

`inmemory_tile_storage` implements `clicks.TileStorage`, including `Subscribe(ctx) (<-chan clicks.Change, error)`, one call per open stream. A `Change` is a tile update or a bomb blast, on one channel so the two keep their order — see [What a bomb does](bomb.md).
