# What a spread does to a click

**The server picks the tiles, off its own map.** A client that named the tiles
a click spreads to could name any tiles it liked — that is why the spread waited
for [Map geography](map-geography.md). The client paints the tile it clicked, as it
always has, and the other tiles reach it over the stream like anyone else's.

A box or a right answer adds its clicks to the pool through planet's `InternalService/GrantCharges` (the offers', see
[The offers](bonuses.md)), which runs `Charges.Grant(holder, KindSpreadClicks, amount)`. A click with spread off never
touches the pool.

## The disc (`charges.spread.rings`)

**A spread takes every tile within `rings` steps of the tile clicked** (`clicks.Disc`: a breadth-first walk over
`Geography.Neighbours`, nearest ring first; the owner, 2026-11-12, for Capital Siege). Rings 1, 2 and 3 take 7, 19 and
37 tiles with the tile clicked, fewer at a coast. It is a walk on the map's own neighbours, not an angle like the
bomb's, so a spread never jumps a strait. `default` holds 1, today's spread (migration `20261112120000_spread_knobs`);
a set holds 1 to 3 (`charges.MostSpreadRings`).

## A spread is taken before the click, and given back on a refusal

`click/spread_click` reads it. **It sits outside the reach, the throttle and the rule**, inside the try, the ban and
the allegiance check:

1. With spread on, `Charges.TakeSpread` decides how far the click spreads before anything else plays: the rings in
   force, and a spread click spent (`Charges.Spend(holder, Held.AfterSpreadClick)`) unless the set is unlimited. With
   none held, or no account, it spreads nothing (`Rings` 0), and the click goes on as a plain click.
2. The click goes on with `click_usecase.In.Rings`. The reach checks it with its disc (below), the throttle takes one
   token, and the rule writes the tile.
3. **A click refused inside it gives its spread click back** (`Charges.GiveBackSpread`, `Held.WithSpreadClickBack`,
   on a context the caller cannot cut): out of reach, throttled, an unknown country, a tile out of range, both bonuses
   at once, or a write that failed. So a refused click spreads nothing and costs nothing, as before.
4. An accepted click spreads: `Reach.Around` names the tiles, `ledger.Recording.Spread` claims each one as an ordinary
   `Set`, and `notices.Board.PublishSpread` tells every stream.

**Why before and not after** (2026-11-12): the reach widens for a click that spreads (see [The reach](reach.md)). If the
spread were spent after the click, two clicks racing for the last spread click would both pass the wider check, and
the second one would land outside the plain reach with no spread to fill the gap. Taken first, the second click finds
none, is checked as a plain click, and is refused. **No click lands outside the plain reach unless its spread really
happens.** A store that cannot take the spread fails the click before anything is written.

- a click is **one click** to the throttle and to `prom_click`, however many tiles it took
- a click the ban or the allegiance refuses never reaches the spread, so it spends nothing
- the spread's tiles are not reported to the antibot jury — only the clicked tile is a click the caller made
- a throttled click with spread on costs two writes to `planet.charges` (the spend and the give back); an unlimited
  spread costs none

## Unlimited spreads (`charges.spread.unlimited`)

**A rule set with `unlimited` makes spread cost nothing**: a click sent with spread on spreads, needs no charge and
spends none, a guest's too (the owner, 2026-11-12, for Capital Siege), as
[Unlimited enclosures](enclose.md#unlimited-enclosures-chargesencloseunlimited) does for enclose. The switch is still
the player's: the server reads `ClickRequest.spread`, and the client turns it on by default while the rule plays (see
the frontend's [Charges](../../frontend/docs/charges.md)). **The spread clicks a player holds are not touched**, so
they are there when the set stops; a battle that plays it sets the offers' `kinds.spreadClicks` to 0 in their set of the
same name. `default` holds `false`.

## What one spread writes

`ledger.Recording.Spread` keeps the spread as one ledger event, whatever its size. **Each tile meets its shields,
exactly as a click on it would**: a shielded tile loses one and keeps its flag. A bonus is never a way around the rule.
The spread click is spent either way, since the click was accepted. Each tile claimed publishes its own `TileUpdate`,
and a tile already held is a no-op. **A lone island takes itself and nothing else**: `Neighbours` is empty there.

**Then it tells the planet, with `tiles_spread`**: the tile clicked and the tiles it took, after they are set, so every
client can animate why they flipped. A spread is one notice per click on the board's buffer (32), whatever its size.
Its tile updates are 6, 18 or 36 more entries in each stream's buffer (`tilesStorage.subscriberBuffer`, 1,024).
`tiles_spread` still lists every tile touched; the tile updates say which were taken and which lost a shield.
`Reach.Around` hands back a slice of its own, never the map's table.

**While the reach plays, a spread takes its whole disc once the tile clicked is the flag's**, and only the tiles of its
disc in reach (`reaches`: the flag holds it, a neighbour of it, or a bridge's other end) when a shield kept the tile
clicked. Both are read after the clicked tile is written. With 1 ring, this is the rule from before. **Accepted side
effect** (the owner, 2026-11-12): with 2 rings or more, a shielded tile of the first ring keeps its flag, and a tile
behind it is still taken. That tile is then an island of the flag, joined to its land by nothing.

**Measured on the whole stack** (2026-11-12, an M-series laptop, the backend, postgres and Redis local, rings 2,
unlimited, a script clicking at 1 click a second per player with spread on, random tiles, the reach off): 15 players
gave 240 tile updates a second on every stream, a click answered in 7 ms (13 ms at the 99th percentile), and the server
used 3% of one core; 60 players and 25 streams, 950 updates a second on each, 26% of one core, no stream cut, a click
in 45 ms (187 ms at the 99th). One page of the game, open through it all, kept 120 frames a second with no long task.
Production's busiest second measured so far (2026-11-10) held 46 events on the whole map. Each tile is still its own
`TileUpdate` frame: one stream message per spread, as for an enclosure, would cut that by 19 if it ever counts.

Tests: `clicks`' `reach_test.go` pins the disc for 0 to 3 rings on a patch of hexagons, its end at a coast, the whole
disc from a tile taken, the disc in reach from a struck one, the disc without the reach, and a slice of its own;
`embedded_geodesic_map`'s `load_test.go` the 6, 18 and 36 tiles around inland tiles of the shipped map; `charges`'
`spread_test.go` the spend, nothing held, unlimited, and the give back; `spread_click`'s tests the rings handed to the
click, the give back on a refusal, the pool spent to its end, unlimited, and with the real reach and `reach_click`, a
centre two tiles out taken with spread and refused without, and two clicks racing for the last spread click.
