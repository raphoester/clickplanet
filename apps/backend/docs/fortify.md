# Fortify

**A player's take that makes a landmass whole for its flag gives every tile of it a
shield**, up to `bonus.shield.perTile`. A landmass is a piece of one country's ground
that touches itself on the lattice, the borders blob's own table (658 of them): Alaska
is one, the lower 48 another. **The same flag never fortifies a landmass twice in a
row**: another flag has to take all of it first, through its shields. That is the whole
defence against taking back the last tile for a free refill, and it costs no timer.

- **The rule is `clicks.FortifyError`**, pure and tested in the `clicks` root: not whole
  (`ErrNotWhole`), or fortified last by the same flag (`ErrFortifiedAlready`).
  `clicks.SettlerOf` is the other half: a whole landmass nobody fortified is locked to its
  holder with no shields.
- **The lock lives with the tiles**, in `inmemory_tile_storage`'s `landmasses`: a count
  per landmass and per flag, moved with the board by every change of owner
  (`moveLocked`), and the flag that fortified each landmass last. `Fortify(tile, flag, most)` checks and
  raises under the tiles lock, so two takes racing for the last tile fortify once. It
  publishes one `clicks.Change{Fortification}` on the map's channel, after the updates
  that made the landmass whole.
- **Only a player's take fortifies.** `clicks.Claiming` asks `Fortify` after each take,
  so a click, each tile of a spread and each tile of an enclose can; the answer is on
  `Impact.Fortified`. A reassign or a revert that makes a landmass whole locks it to its
  holder instead (`settleLocked`), and so does the boot for every whole landmass with no
  lock: so the launch, and an operator, can never hand out a free fortify.
- **Kept with the tiles**: `planet.landmasses (asset, id, fortified_by)`, written by the
  same flush, so a lock and its shields are never saved apart. A row is keyed by the
  borders blob's name: a regenerated map numbers its landmasses again, and the boot
  reads only the rows of the blob it loaded (then settles the rest).
- **On the wire**: `PlanetEvent.landmass_fortified{landmass_id, country_id, tile_id}`, in
  order with the tile updates. The client adds the shield to each tile of the landmass
  its flag holds, as the server did, rather than reading a list of 28,000 tiles.
  `ClickService/GetFortresses` (`NO_SIDE_EFFECTS`, cached 5s) is every lock now, read
  through `fortresses_query`, for the pulse on the tiles a flag still misses.
  `ClickService/GetLandmass` reads one landmass's last fortify, who and when, from the ledger (see [Who holds the
  tiles](tile-holders.md)).
- **In the ledger**: a `fortify` event after the take that earned it (see
  [Durability](durability.md)), with the landmass's ground and size, so a replay shows each shield it raised.
- **To the other modules**: the ledger relay tells `planet.v1.Fortified` from that event once it is flushed,
  with its ledger position (see [What may not be lost](composite-layer.md#what-may-not-be-lost-the-outbox)). The event's
  `account_id` is the account that took the last tile. The chat announces the ones of `fortifyNewsTiles` (50) tiles or more.
