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
  taken. A bonus is never a way around the rule. `tiles_enclosed` still lists it
  among the filled tiles; no tile update says it was taken.
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
the rule set's `bonuses.enclose.maxTiles`, read at each click, how big a shape may be. `Annexer` spends the charge on the first pocket, takes its
tiles and announces them. `Execute` asks whether the charge is held, lets the
rule write, and hands the pockets to the annexer.

It sits beside `spread_click`, against the rule and inside everything else, so
it is one click to the throttle and to `prom_click`, a click the ban refuses never
reaches it, and the tiles it takes are not reported to the antibot jury. The
annexer hands the inside to `ledger.Recording.Enclose`, which claims each tile as
an ordinary `Set` and keeps the enclosure as one event. The search holds no lock across the map, so a
tile can change under it; the worst that does is fill a pocket that opened a
moment ago.

**Then it tells the planet, with `tiles_enclosed`.** The tiles already travel as
tile updates, but a patch flipping at once says nothing about why, so every
client is sent the shape — closing tile, wall, and filled tiles nearest the
closing tile first — to animate. `Registry.PublishEnclosed` sends it after the
tiles are set, to every caller. The caller who closed it gets a copy of their own
with `yours`: an enclosure is one shape, so a shape of your own is one spent.

`prom_enclose` wraps that publisher, so it counts exactly the shapes that were
closed: `bonus_enclosures_total` and `bonus_enclosed_tiles_total`.
