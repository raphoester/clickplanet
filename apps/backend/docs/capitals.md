# The capitals' rules: the knock-out and fixed shields

**Two knobs of a rule set play on the capitals** (`capitals`, `clicks.CapitalsPart`; knobs of their own since
2026-11-10). Both are off in `default`. Capital Siege plays both on its fresh map, which starts with each capital (see
[Arena battles](arenas.md#seeds)); any set may play them on the world's map too. The capitals are the map's
(`clicks.Capitals`): each country's tile nearest its capital city. A region flag (England, Scotland, Wales, Brittany)
has no capital.

The tile storage holds `clicks.CapitalPlay`: the capitals and the reader of the rules in force. The rules are pure
(`clicks.CapitalRules`: `KnockOutError`, `Stands`; `CapitalPlay.Fixes`, `Felled`); the storage only does the atomic work
under its lock.

## Fixed shields (`capitals.fixedShields`)

**No shield is added to a capital**, whoever holds it: `PlaceShield` is refused with `clicks.ErrCapital`
(`FailedPrecondition`, the charge given back), and a fortify raises every tile of the landmass but it
(`capitalLocked`). A capital keeps the shields it has, the seed's on a fresh map. See [Shields](shields.md).

## The knock-out (`capitals.knockOut`)

**A take of a flag's capital knocks the flag out: the taker keeps the capital, and every other tile of the flag goes
blank** (the owner, 2026-11-09). It is decided where a fortify is: under the tiles lock, right after the take, so two
takes racing for one capital knock the flag out once.

- **The rule** (`CapitalRules.KnockOutError`, pure): the knock-out plays, the tile is a capital, its flag is not out
  yet, and the taker is another flag. It reads the capital's flag, not the tile's last owner, so a capital left blank
  (a bomb) still falls to the next take. Every take asks, a click's, each tile of a spread's and an enclosure's; a bomb
  takes no tile for the bomber, so it never knocks a flag out.
- **The claim** (`clicks.Claiming`) asks `Claimable.KnockOut` after a take is written. **An enclosure asks it once its
  whole pocket is written**, so the blank never clears a tile the same pocket takes after it: a shielded tile of the out
  flag inside the pocket keeps its flag through the pocket, then goes blank.
- **The storage** (`inmemory_tile_storage`, `knock_out.go`) clears every tile the out flag holds (no shield, no holder,
  the landmass counts moved, dirty for the flush), keeps the flag out, and publishes **one** `clicks.Change{KnockOut}`
  at one map version, with every tile it cleared: right after the take's update for a click or a spread, after the
  enclosure and its fortifies for an enclosure. The stream sends it as `PlanetEvent.knock_out` (`country_id`,
  `by_country_id`, `tile_id`, `cleared_tile_ids`, `version`).
- **The flags out are kept with the tiles** (the owner, 2026-11-10): in memory beside the board, and in
  `planet.knockouts` (`map`, `flag`, `by_flag`, `knocked_out_at`), written by the map's flush in the transaction that
  writes the tiles, so a flag is never saved out apart from its blank land. A failed flush keeps them for the next one.
  The boot reads them with the map (`LoadOutFlags`); a wipe deletes them with its map (an arena's close, a season's
  reset). The world's map keeps its own since 2026-11-10. Every map read says them (`GetMapResponse.knocked_out`).
- **Which flags still stand** (`CapitalRules.Stands`, read through `Storage.Stands`): every flag while the knock-out
  does not play; while it does, a flag with a capital that is not out. A flag out on a map stays out there while the
  rule is off, and is out again if it comes back on, until a wipe. The allegiance reads it (see
  [Allegiance](allegiance.md)).
- **In the ledger**: a `knock_out` event after the take (and its fortify), on the capital, the taker's account and flag,
  the out flag as `previous`, and the cleared tiles. Its takes are each cleared tile as a bomb's (for nobody), so the
  map history sees the blank; `GetLedger` shows each as a blank tile; it strikes nothing.
- **To the other modules**: the ledger relay tells `planet.v1.FlagKnockedOut` (`country`, `by`, `tile_id`,
  `account_id`, `cleared`, `knocked_out_at`, `position`) once the ledger flushed it, in order with the takes. The chat
  announces every one ("Germany knocked out France"; the owner, 2026-11-10). No Discord ping.
- **Scoring needs nothing**: an out flag holds no tile, so it gains nothing.
- **Without the reach**, a flag that is out can still take tiles: see [Arena battles](arenas.md#known-gaps).
- **The page** shows the knock-out and asks a player whose flag is out to pick one still in: see the frontend's
  [The capitals](../../frontend/docs/capitals.md).

## Tests

`clicks`' `knock_out_test.go` pins the rule, the standing and fixed shields by the rule; `inmemory_tile_storage`'s
`capitals_test.go` a capital refused a shield while its shields are fixed (the world's map too) and taking one while
they are not, a fortify that skips a fixed capital, the map's `Arena`, and `HoldsAny`; its `knock_out_test.go` the take
of a capital that clears the flag's other tiles in one change after the take, nobody knocked out while the rule is off,
the knock-out on the world's map kept by the flush, an enclosure that knocks out once its pocket is written, the flag
out saved by the flush (again after a failed one) and read at boot, and gone with its arena; `postgres_tile_store`'s
`knockouts_test.go` the rows of each map, the world's too, kept once and deleted by a wipe; the migrations'
`knockouts_test.go`; `ledger_test.go` the event's entry, takes and scenes; the relay's `FlagKnockedOut`; the stream's
sink and `GetMap`'s out flags; `allegiances`' rule and `Book`.
