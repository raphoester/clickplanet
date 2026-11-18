# Fortify

**A player's take that makes a province whole for its flag gives every tile of it a
shield**, up to the rule set's `charges.shield.perTile`. A province is a row of the board's provinces file: one
Natural Earth admin-1 unit (a US state, a German Land) or several joined, inside one landmass (see
[Provinces](../../../docs/provinces.md)). Until 2026-11-18 fortify played on landmasses; on `earth-2` the lower 48 are
~54,000 tiles, and nobody held all of that. **The same flag never fortifies a province twice in a
row**: another flag has to take all of it first, through its shields. That is the whole
defence against taking back the last tile for a free refill, and it costs no timer.

- **The rule is `clicks.FortifyError`**, pure and tested in the `clicks` root: not whole
  (`ErrNotWhole`, and so a tile in no province), or fortified last by the same flag (`ErrFortifiedAlready`).
  `clicks.SettlerOf` is the other half: a whole province nobody fortified is locked to its
  holder with no shields. Capitals are still skipped when the rules fix their shields.
- **A board with no provinces file fortifies nothing**: `earth-1`, which only draws history, and `fixture-1`. A test
  ground (`clicks.GroundOf`) makes each landmass one province, `clicks.ProvincesOf` cuts provinces of its own.
- **The lock lives with the tiles**, in `inmemory_tile_storage`'s `provinces`: a count
  per province and per flag, moved with the board by every change of owner
  (`putLocked`), and the flag that fortified each province last. The landmasses keep their own counts (`landmasses`),
  for the holdings, with no lock. A claim's `Fortify(tile, flag, most)` checks and
  raises under the lock of the take that asks it, so two takes racing for the last tile fortify once. It
  publishes one `clicks.Change{Fortification}` on the map's channel, after the updates
  that made the province whole.
- **Only a player's take fortifies.** `clicks.Claiming` asks `Fortify` after each take,
  so a click, each tile of a spread and each tile of an enclose can; the answer is on
  `Impact.Fortified`. A revert that makes a province whole locks it to its
  holder instead (`settleLocked`), and so does the boot for every whole province with no
  lock: so the launch, and an operator, can never hand out a free fortify.
- **Kept with the tiles**: `planet.provinces (map, board, asset, id, fortified_by)`, written by the
  same flush, so a lock and its shields are never saved apart. A row is keyed by the
  provinces file's name: another cut (a change of N, a regeneration) numbers its provinces again, and the boot
  reads only the rows of the file it loaded, then settles the rest. A remap migration moves no lock: the landmass a
  fortify made before provinces names is moved with its borders (see [`/map/README.md`](../../../map/README.md)).
- **On the wire**: `PlanetEvent.province_fortified{province_id, country_id, tile_id, version}`, in
  order with the tile updates. The client adds the shield to each tile of the province
  its flag holds, as the server did, rather than reading a list of 20,000 tiles.
  `ClickService/GetFortresses` (`NO_SIDE_EFFECTS`, cached 5s) is every lock now, by province, read
  through `fortresses_query`, for the pulse on the tiles a flag still misses.
  `ClickService/GetProvince` reads one province's last fortify, who and when, from the ledger (see [Who holds the
  tiles](tile-holders.md)).
- **In the ledger**: a `fortify` event after the take that earned it (see
  [Durability](durability.md)), with the province, the provinces file, its ground and size, so a replay shows each
  shield it raised. **One made before 2026-11-18 names its landmass** and no province: every reader takes both (the
  replays and the map history read the raised tiles alone, the relay tells either).
- **To the other modules**: the ledger relay tells `planet.v1.Fortified` from that event once it is flushed,
  with its ledger position (see [What may not be lost](composite-layer.md#what-may-not-be-lost-the-outbox)): its
  `province_id` and the `landmass_id` it is inside (read from the tile's board), or for an old event the
  `landmass_id` alone. The event's `account_id` is the account that took the last
  tile. The chat announces the ones of `fortifyNewsTiles` (50 of `earth-1`'s, 200 on `earth-2`) tiles or more, naming
  the province and its country. The missions and the XP count its tiles.

## The switch to provinces (2026-11-18)

Migration `20261118171922_province_locks` drops `planet.landmasses` and makes `planet.provinces`; the next boot finds
no lock and settles every province one flag holds whole to that flag, with no shield. **No shield is taken away, and
nobody gets a free fortify**: a province France held whole stays France's to defend, and its next fortify is another
flag's. `TestTheBootAfterTheLocksMovedToProvincesSettlesEachWholeProvinceWithNoShield` pins it. Its down makes an empty
`planet.landmasses` again, and the old build settles every landmass the same way. Migration
`20261118171923_ledger_events_fortify_province` swaps the partial index the last fortify is read on.

**A switch of the world's board drops every lock**: a province of the old board is no province of the new. Each
province of the new board is settled again as the boot settles them (`clicks.SettlerOf`), so one a flag holds whole is
locked to it with no shield, and the next fortify of it is another flag's. See [Boards and eras](boards-and-eras.md#the-worlds-switch-of-board-switchboard).
