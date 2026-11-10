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
- **While the reach plays, a pocket needs no check of it** (see [The reach](reach.md)):
  it is walled by the flag's own tiles alone, and its flood stops at any coast, so every tile inside is the flag's
  inside. The click that closed it was checked.
- **A capital inside a pocket knocks its flag out once the whole pocket is written** (see
  [The knock-out](capitals.md#the-knock-out-capitalsknockout)): the blank never clears a tile the pocket takes after the capital, and the
  knock-out goes on the stream after the enclosure and its fortifies.
- **An enclosure is one click**, spent through `Charges.Spend(holder, Held.AfterEnclose)` only by a
  click sent with enclose on that closed a pocket, so a click that closes nothing
  (too big, open to the coast, no inside) keeps it. With enclose off, closing a
  shape takes nothing. The spend settles two clicks racing for the last one.
- **A click that closes several shapes fills them all, for one charge** (the owner, 2026-11-05: closing two
  shapes with one well-placed click is skill). Each shape is held to `maxTiles` on its own, so one click can fill up
  to three. The search joins them into one pocket (`joined`): each wall tile once, and the insides side by side,
  nearest the closing tile first, so the client fills both sides at once. Until then a click took the first shape
  only.
- **Why 25 tiles**: the wall around a 25-tile pocket costs about 20 clicks, so the
  reward matches the planning it took.

**The use case only wires three objects together.** `bonuses.Terrain` is the map as
the search sees it — who holds a tile, what touches it — and finds the pockets a
click closed. `bonuses.Charges` says whether the caller holds the charge, and
the rule set's `bonuses.enclose`, read once per click, how big a shape may be and whether it costs a charge.
`Annexer` spends the charge and hands the pocket's shape and inside to the encloser. `Execute` asks whether the
charge is held (not while enclosures are unlimited), lets the rule write, and hands the pocket to the annexer.

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
every stream at once. One event per click, with every shape it closed in it, so the client counts one charge spent.
**Each stream tells its own** `yours`: the change names who closed it (`Enclosure.By`), and the stream compares its
entrant (`bonuses.EntrantOf`). The client applies the tiles at the version, then animates the shape (see the
frontend's [Charges](../../frontend/docs/charges.md)).

`prom_enclose` wraps the encloser, so it counts exactly the clicks that closed shapes and the tiles they filled:
`bonus_enclosures_total` and `bonus_enclosed_tiles_total`.

## Unlimited enclosures (`bonuses.enclose.unlimited`)

**A rule set with `unlimited` makes enclose cost nothing**: a click sent with enclose on fills what it closes and
needs no charge and spends none (the owner, 2026-11-05, for a battle). The switch is still the player's: the server
reads `ClickRequest.enclose` as always, and the client turns it on by default while the rule plays (see the
frontend's [Charges](../../frontend/docs/charges.md)). **The enclosures a player holds are not touched**, so they are
there when the set stops playing; a battle that plays it sets `bonuses.kinds.encloseClicks` to 0, since a box of
them would be worth nothing. `GetBonusRules.enclosures_unlimited` tells the client. `default` holds `false`
(migration `20261105120000_enclose_unlimited`).

**How big a shape may be is `maxTiles`, apart from it.** Measured on the real map (2026-11-05, an M-series laptop):
a search that finds open ground stops at the first coast or lake, 0.07 ms at the median and 2 ms at worst with no
size limit, since no tile is more than 64 tiles from one. The biggest inland ring holds 11,347 tiles; its search
took 2 ms and taking its tiles 5 ms. Taken as one event (above), such a pocket cuts no stream.

**A ring bigger than that is a coastline**: every tile a flood can reach that has fewer than six neighbours (a coast,
a lake's shore, a lattice corner) must be the caller's, or the shape is open. So ringing Russia means holding its
outline, coasts and lakes included, and ringing Africa means holding the Sinai's border too.

## Giant pockets, measured (2026-11-07)

Before the Ring War battle (`unlimited-enclose`, `maxTiles` 1,000,000), a whole country and a whole continent were
closed on the real map, through every layer:

| Pocket | Tiles | Closing click (4 cores) | (1 core) | Frame |
|---|---|---|---|---|
| Russia, its outline held (2,101 tiles) | 25,895 | 58 ms over HTTP | 132 ms | 81 KB |
| Africa, its outline held (1,644 tiles) | 51,253 | 131 ms over HTTP | — | 146 KB |
| Afro-Eurasia, every coast held (6,498 tiles) | 136,032 | 0.5 s in process | 0.4–0.6 s | 400 KB |

The pocket equals an independent flood of the ring tile for tile; one in seven tiles shielded loses one shield and keeps
its flag; the stream carries one frame and no page's stream is cut; the ledger keeps one `enclose` row (800 KB to 2 MB
of payload) and the flush writes every tile; `GetMap` and `GetTileHolder` (2.6 ms on the GIN index) name the ringer;
progression's stats and scoring's tally count every tile once. Other players' clicks wait at most 66 ms (4 cores) or
142 ms (1 core) while the tiles lock is held. Taking a whole country fortifies it, and taking Africa fortifies 53
landmasses at once.

**What a giant pocket cost was behind planet, not in it, until 2026-11-07.** The relay told one `planet.v1.TileTaken` per
tile (one `ShieldStruck` per shield struck), and each subscriber handled one at a time, in a transaction of its own:
progression and scoring took in about 260 takes a second on 4 cores and 85 on one, so an Africa showed in the players'
numbers 10 minutes later. Worse, every `XADD` trims a stream to about `redis.maxLen` (100k) whether its groups read the
entries or not (Redis 7.4 has no `ACKED` trim), so pockets that added up to more than 100k takes before the slowest
group caught up dropped the oldest, silently: each player's count (stats, titles, missions, the players' board, a
battle's players), never a flag's points, since a round is scored from the map (see [Scoring](scoring.md)).

**So an enclose is one fact, and a batch is one write.** The relay tells one `planet.v1.TilesTaken` per ledger event,
with every tile it took, in parts of 4,096 tiles at most (`relay_ledger_usecase.PartTiles`): an Africa is 13 entries of
some 200 KB, not 51,253. Its subscribers read with `cpredis.SubscribeBatch` and write each account's, round's or day's
takes of a batch at once, keeping their marks (see [The composite layer](composite-layer.md#telling-other-modules-what-happened)).
Shields struck go the same way, as `planet.v1.ShieldsStruck`. Measured again on the whole stack (4 cores): every group
caught up 2.3 s after Russia's closing click and 2.9 s after Africa's, against 1.5 to 3.5 minutes before, and
progression and scoring each count exactly the tiles the ledger says the ringer took.

**To measure again**: `CP_GIANT=1 go test -tags testing ./internal/planet/internal/clicks/usecases/click_usecase/enclose_click/
-run TestGiantEnclosures -v` runs the pockets above in process. `CP_GIANT_DUMP=<dir>` also writes the Russia and Africa
rings, which `CP_GIANT_SCENARIOS=<dir> go test -tags testing ./e2e/ -run TestGiantEnclosuresOnTheWholeStack -v` plays on
the whole stack (Docker, about 8 minutes): tonight's rule set as a battle of the gamemaster, the rings clicked over HTTP
by 30 players, 25 pages listening. `taskset -c 0`, with the containers pinned to the same core, is the droplet.
