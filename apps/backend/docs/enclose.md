# What an enclose does to a click

**It looks for a small inside, never for the outline.** On a sphere every loop
cuts the planet in two, and both halves are inside it. So after an accepted
click, `click/enclose_click` floods out from each neighbour of the clicked tile
that is not the caller's, over tiles that are not the caller's. A flood that
runs out before passing `enclose.maxTiles` found a pocket; one that passes it is
open ground or too big, and takes nothing. The limit is therefore also what
tells closed from open — there is no second rule.

- **A pocket is walled by the caller's tiles alone.** A flood that reaches a tile
  with fewer than six neighbours has reached the edge of the land — a coast, a
  lake — and is open. Cutting off the tip of a peninsula is not a closed shape.
  The twelve lattice corners have five neighbours and read as an edge too.
- **A shape too big takes nothing.** Taking part of it would mean choosing which
  part, and there is no good answer.
- **A triangle costs nothing**: three tiles that touch each other have no inside,
  so the flood finds open ground and no shape is spent.
- **Only a click that takes a tile closes a shape.** A click on a tile the caller
  already held changes nothing, so it closes nothing: a shape finished before the
  bonus stays as it is. A click a shield took is no wall either, since the tile
  is still the other flag's. The rule answers `Out.Outcome`, so this is read after
  the write, not guessed before it.
- **Each tile inside meets its shields**: a shielded tile loses one and is not
  taken. A bonus is never a way around the rule. `tiles_enclosed` lists it among the filled tiles, and in `struck`
  with the shields it kept.
- **An enclosure is one shape**, spent through `Charges.SpendEnclose` only by a
  click sent with enclose on that closed a pocket, so a click that closes nothing
  (too big, open to the coast, no inside) keeps it. With enclose off, closing a
  shape takes nothing. The spend settles two clicks racing for the last one. A
  click that closes two shapes takes the first.
- **Why 25 tiles**: the wall around a 25-tile pocket costs about 20 clicks, so the
  reward matches the planning it took.

**The use case only wires three objects together.** `bonuses.Terrain` is the map as
the search sees it — who holds a tile, what touches it — and finds the pockets a
click closed. `bonuses.Charges` says whether the caller holds the charge, and
the rule set's `bonuses.enclose.maxTiles`, read at each click, how big a shape may be. `Annexer` spends the charge
and hands the first pocket's shape and inside to the encloser. `Execute` asks whether the charge is held, lets the
rule write, and hands the pockets to the annexer.

It sits beside `spread_click`, against the rule and inside everything else, so
it is one click to the throttle and to `prom_click`, a click the ban refuses never
reaches it, and the tiles it takes are not reported to the antibot jury. The
annexer hands the shape to `ledger.Recording.Enclose`, which claims it through `Claiming.Enclose` and keeps the
enclosure as one ledger event. The search holds no lock across the map, so a
tile can change under it; the worst that does is fill a pocket that opened a
moment ago.

**An enclosure is one event on the stream, `tiles_enclosed`, and no `tile_update`** (2026-11-05), as a bomb is one
`bomb_dropped`. `Claiming.Enclose` claims every tile inside, each with its shields and its fortify, as
`Storage.Enclose` holds the tiles lock through all of them; the storage then publishes one `clicks.Change{Enclosure}`
at one map version, with every tile it wrote (`filled`, nearest the closing tile first) and the shields each struck
tile kept (`struck`), then the fortifies the takes earned, each a `landmass_fortified` of its own. So a pocket of any
size takes one slot in each stream's buffer (`tilesStorage.subscriberBuffer`, 1,024: a stream that falls that far
behind is cut, see [The live streams](live-streams.md)), and one frame on the wire. Until then each tile went out as
a `tile_update` and the shape again in `tiles_enclosed`, and a pocket of more than about 1,000 tiles would have cut
every stream at once. **Each stream tells its own** `yours`: the change names who closed it (`Enclosure.By`), and the
stream compares its entrant (`bonuses.EntrantOf`); an enclosure of your own is one charge spent. The client applies
the tiles at the version, then animates the shape (see the frontend's [Charges](../../frontend/docs/charges.md)).

`prom_enclose` wraps the encloser, so it counts exactly the shapes that were
closed: `bonus_enclosures_total` and `bonus_enclosed_tiles_total`.
