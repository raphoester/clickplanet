# What a spread does to a click

**The server picks the tiles, off its own map.** A client that named the tiles
a click spreads to could name any tiles it liked — that is why the spread waited
for [Map geography](map-geography.md). The client paints the tile it clicked, as it
always has, and the neighbours reach it over the stream like anyone else's.

A box or a right answer adds its clicks to the pool through planet's `InternalService/GrantCharges` (the offers', see
[The offers](bonuses.md)), which runs `Charges.Grant(holder, KindSpreadClicks, amount)`. Each accepted click sent with
spread on then spends one with `Charges.Spend(holder, Held.AfterSpreadClick)`, after the rule accepted
it: a refused click spreads nothing and costs nothing, and a click with spread off
never touches the pool.

`click/spread_click` is the decorator that reads it, and **it sits right against
the rule**, inside the count, the ban and the throttle:

- a click is **one click** to the throttle and to `prom_click`, however many tiles it took
- a click the ban refuses never reaches the rule, so it spreads nothing
- a click the rule refuses (unknown country, tile out of range) spreads nothing
- the neighbours are not reported to the antibot jury — only the clicked tile is
  a click the caller made

`spread_click` hands the neighbours to `ledger.Recording.Spread`, which claims each
one as an ordinary `Set`, so it publishes its own `TileUpdate` and a tile already
held is a no-op. The ledger keeps the spread as one event. One click is at most 7 updates. **A lone island
takes itself and nothing else**: `Neighbours` is empty there, and the bonus does
not pretend otherwise.

**Each neighbour meets its shields, exactly as a click on it would**: a shielded
neighbour loses one and keeps its flag. A bonus is never a way around the rule.
The spread click is spent either way, since the click was accepted.
`tiles_spread` still lists every neighbour touched; the tile updates say which
were taken and which lost a shield.

**Then it tells the planet, with `tiles_spread`**: the tile clicked and the neighbours it took, after they are set,
so every client can animate why seven tiles flipped. `notices.Board.PublishSpread`
sends it to every stream open.

A spread is one event per click, which is why a stream's notices buffer 32
events rather than a handful.

**While the reach plays, a spread takes only the neighbours in the flag's reach** (`clicks.Reach.Around`, read after the clicked
tile is written): a click that took its tile reaches every neighbour, and one a shield took reaches only those that
touch other land of the flag or lie across one of its waters. See [The reach](reach.md).
