# Who holds the tiles

**Each tile keeps the account whose take holds it**, beside its owner and its shields, in
`inmemory_tile_storage` (`tileState.holder`, an index into a `holderbook` of accounts, as the flags are into
the codebook). It is a field of the tile, not a view of the ledger: it changes exactly when the owner does, under
the same lock, and is flushed in the same row. The ledger has the same fact, but reading it for a whole landmass
means every change of every one of its tiles, and the tiles a spread or an enclose took are only in JSON payloads:
thousands of index probes, or a scan of the whole ledger, for one card on the 1-vCPU box.

- **A player's take sets it**: `Set` and `Click` read the account from the context (`cpctx.GetAccount`), as
  `Fortify` already does for `Fortification.Account`. So a click, and each tile a spread or an enclose takes, is
  held by the player who made it. A click that changes no owner (its own flag, or a shield struck) keeps the holder.
- **Any other write leaves nobody**: a bomb's clear, `Reassign`, `Restore` (a revert or a paint), and a take with no
  account in its context. This is `GetTileHolder`'s rule too: an operator's write keeps no take.
- **Kept in `planet.tiles.holder`** (NULL for nobody), loaded at boot. Migration `20261026180000_tile_holders`
  filled it once from the ledger: the account of each tile's last change, while that change gave the tile the flag
  it wears.
- **A deleted account is forgotten**: `anonymize_takes_usecase` calls `ForgetHolder` after it blanks the ledger's
  rows. The tiles keep their flag and shields; the next flush writes NULL.
- **`ClickService/GetLandmass(landmass_id)`** (`NO_SIDE_EFFECTS`, cached 5s like the map) answers who holds the
  landmass now, `{account_id, country_id, tiles}` per account and flag, most tiles first, and its last fortify,
  `{account_id, country_id, fortified_at_unix_ms}`. `landmass_query` reads the fortify from `ledger_events` on the
  partial index `ledger_events_fortify` (`((payload ->> 'landmass')::integer), position` where `kind = 'fortify'`),
  and the holders through its port `Holders`, which `landmass_query/inmemory_tile_holders` answers from
  `HoldingsOf`. It knows no name: the client asks `player.v1.PlayerService/GetPlayer`, as for a tile.
- **The last fortify is the ledger's, the lock the tiles'**: a lock settled at boot or by an operator has no fortify,
  and a landmass's id in an old fortify's payload is the borders blob of its day. The client shows who fortified only
  when the fortify's flag is the lock's.
