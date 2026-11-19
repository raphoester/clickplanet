# The capitals' rules: the knock-out, fixed shields and the cut-off

**Three knobs of a rule set play on the capitals** (`capitals`, `clicks.CapitalsPart`; knobs of their own since
2026-11-10, the cut-off since 2026-11-19). All are off in `default`. Capital Siege plays all three on its fresh map, which starts with each capital (see
[Arena battles](arenas.md#seeds)); any set may play them on the world's map too. The capitals are the map's
(`clicks.Capitals`): each country's tile nearest its capital city. A region flag (England, Scotland, Wales, Brittany)
has no capital.

The tile storage holds `clicks.CapitalPlay`: the capitals and the reader of the rules in force. The rules are pure
(`clicks.CapitalRules`: `KnockOutError`, `Stands`, `CutOffError`; `CapitalPlay.Fixes`, `Felled`, `Stranded`); the
storage only does the atomic work under its lock.

## Fixed shields (`capitals.fixedShields`)

**No shield is added to a capital**, whoever holds it: `PlaceShield` is refused with `clicks.ErrCapital`
(`FailedPrecondition`, the charge given back), and a fortify raises every tile of the province but it
(`capitalLocked`). A capital keeps the shields it has, the seed's on a fresh map. See [Shields](shields.md).

**No mine goes on a capital while the knock-out plays** (`CapitalPlay.KnocksOut`, `mines.Gate`): its take would clear
the flag's land, the mine with it, before the mine went off. See [The mine](mine.md).

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

## The cut-off (`capitals.cutOff`)

**A flag's land must hold to its capital at all times: a take that cuts land of a flag from its capital clears the land
cut off** (the owner, 2026-11-19, for Capital Siege). Land holds together through the flag's own tiles next to each
other (`Geography.Neighbours`) and the bridges whose both ends it holds (`clicks.Bridges.Across`): the same links as
[the reach](reach.md), so with the reach every take joins its flag's land. The taker keeps the tile it took.

- **The rule** (`CapitalPlay.Stranded`, pure, in `clicks/cut_off.go`): the cut-off plays and the flag has a capital
  (`CutOffError`), and some of its land next to the tiles it lost no longer reaches the capital. It reads only that
  land: the flag's tiles linked to a lost tile (the edge). Nothing is cut when the edge holds together by itself and the
  capital is still the flag's, which most takes at a front are; otherwise a walk from the capital, stopped once it
  reached every edge tile, and every edge tile it did not reach is cut with all the land linked to it. So land cut off
  before the rule played stays (a set switched on over the world's map clears nothing at once), and a flag that lost its
  capital loses all the land next to it.
- **Every write that takes asks** (`clicks.Claiming`, the virus's `Contagion`, the mine's `Minefield`), after its takes
  and their knock-outs: `Claimable.CutOff` once per flag the write took tiles from, with every tile it lost
  (`clicks.Losses`). One ask per flag and per write matters: two tiles of an enclosure or of two mines' blasts can cut
  together what neither cuts alone. A spread is one claim per tile, each its own write.
- **With the knock-out**, a take of a capital knocks its flag out first, and the cut-off finds no land left. **Without
  it**, the capital taken cuts the flag's whole land, and the flag still stands.
- **The storage** (`cut_off.go`) clears each tile cut off (no shield, no holder, the counts moved, a mine on it gone, a
  virus cured, dirty for the flush) and publishes **one** `clicks.Change{CutOff}` per flag at one map version, after the
  take's update and its knock-out: right after for a click, after the enclosure, the infection or the explosion and their
  fortifies and knock-outs otherwise. The impact of the flag's last tile lost carries it (`Impact.CutOff`). The stream
  sends it as `PlanetEvent.cut_off` (`country_id`, `by_country_id`, `tile_id`, `cleared_tile_ids`, `version`).
- **In the ledger**: a `cut_off` event after the take (kind added by `20261119154301_ledger_cut_off`), on the tile
  whose take cut the land, the taker's account and flag, the flag cut off as `previous`, and the cleared tiles. Its takes
  are each cleared tile as a bomb's (for nobody), so the map history and the replays see the blank, and no account gains
  XP or a take for it; `GetLedger` shows each as a blank tile. No fact goes to the other modules and the chat says
  nothing: a cut-off is too common for a line.
- **Its cost**: the walk is over the flag's land, under the tiles lock, only for the takes whose edge does not hold
  together. On `siege-1` a flag holds a few thousand tiles at most.
- **Known gaps**: a bomb clears tiles with no cut-off after it, and so does an operator's paint or revert. Capital Siege
  plays no bomb. Without the reach, a take away from the flag's land is not cut.
- **The page** clears the tiles it names: see the frontend's [The capitals](../../frontend/docs/capitals.md).

## Tests

`clicks`' `cut_off_test.go` pins the land past a take in a corridor, the tip of the land and a loop that cut nothing, a
bridge both ends held and its end taken, two tiles lost together read together, a capital lost, land cut before that
stays, the rule off, a flag with no capital, and the part's read; `inmemory_tile_storage`'s `cut_off_test.go` a take
that clears the land cut off in one change after the take (a shield on it too, and the flush), a take at the tip, the
rule off, a capital taken with the knock-out and without it, and an enclosure cut once its pocket is written;
`ledger_test.go` the event's entry, takes and scenes, and `Recording` noting it after the take; the stream's sink;
`GetLedger`'s query with every kind.

`clicks`' `knock_out_test.go` pins the rule, the standing and fixed shields by the rule; `inmemory_tile_storage`'s
`capitals_test.go` a capital refused a shield while its shields are fixed (the world's map too) and taking one while
they are not, a fortify that skips a fixed capital, the map's `Arena`, and `HoldsAny`; its `knock_out_test.go` the take
of a capital that clears the flag's other tiles in one change after the take, nobody knocked out while the rule is off,
the knock-out on the world's map kept by the flush, an enclosure that knocks out once its pocket is written, the flag
out saved by the flush (again after a failed one) and read at boot, and gone with its arena; `postgres_tile_store`'s
`knockouts_test.go` the rows of each map, the world's too, kept once and deleted by a wipe; `ledger_test.go` the event's entry, takes and scenes; the relay's `FlagKnockedOut`; the stream's
sink and `GetMap`'s out flags; `allegiances`' rule and `Book`.
