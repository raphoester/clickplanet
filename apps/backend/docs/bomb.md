# What a bomb does

`claim_bonus_usecase` hands the bomb over with `Charges.Grant(holder, KindBomb)` and
the client draws its aiming ring at `GetBonusRules.blast_radius`, the width of
what it will clear. `DropBomb` spends it through `drop_bomb_usecase` with `Charges.SpendBomb`.

**The client names a point, never a tile.** The sea has no tiles, and whether an
aim is on land is the server's call: `Geography.Nearest` finds the closest tile,
and an aim further than one tile spacing from it is **in the sea** — the bomb is
spent, nothing is cleared, and the blast is still broadcast with tile 0 so every
screen draws a splash. That was a product decision: a bad aim costs the bomb.

On land the tiles are `Geography.Within(centre, radius)`: every tile within
`radius` of arc of the tile hit — a true circle, ~230 tiles inland, found by a
straight scan (~0.5ms, once per bomb). The radius is the rule set's `bonuses.bomb.rings` × `Geography.Spacing()`, read at each drop (`bonuses.Bombing`, whose `Blast` decides land or sea),
the mean arc between touching tiles measured at boot (0.0040 rad on the
262,119-tile map, so 0.032), rather than a number in the config that could drift
from the map; the same radius goes to clients, so the ring they draw is the clear.

**It is not `Disc`, deliberately.** Rings of neighbours on a honeycomb make a
hexagon, which showed in production as a hexagonal crater inside a round ring —
and a walk over neighbours stops at water, so an island just offshore survived a
bomb that visibly covered it. A circle has neither problem.

**A bomb hits a shielded tile like a click.** `inmemory_tile_storage.Clear` takes
one shield from each shielded tile in the blast, whoever's it is, and clears the
rest, under one lock: a shielded tile keeps its flag, is not in `cleared_tile_ids`,
and is listed in `struck` with the shields it kept (`Blast.Struck`, `Blast.Left`), so a
client sets its shields at impact. `struck_tile_ids`, the same tiles with no count, is
deprecated and still filled for the tabs loaded before. See [Shields](shields.md).

`drop_bomb_usecase` checks the country and the target **before** taking the bomb, so a
malformed request does not cost one. The drop does not move the schedule: the next
box was already due a window after the claim.

**The blast is one event, and it rides the tile feed.** `inmemory_tile_storage.Clear`
empties the tiles under one lock and publishes a single `clicks.Change{Blast}`
on the same channel as the `clicks.Change{Update}` every `Set` sends. Two
reasons it is not one `TileUpdate` per tile:

- **Order.** A bonus event travels on the registry's channel, and two channels
  merged by a `select` have no order between them. A tile retaken a moment after
  the blast could then reach a client *before* the blast and be blanked by it,
  with nothing to repair it until a reload. On one channel it cannot.
- **Timing.** The client holds the clear back until its drawing of the bomb hits
  the ground, which it can only do with the whole clear in one message. 217
  updates in a burst would also overflow a slow subscriber's buffer.

`DropBomb` is session-gated like `Click` and `ClaimBonus` — it writes the map.
It is not throttled: holding a bomb the server granted is the gate.
**Every bomb that went off is told to the other modules** as `planet.v1.BombLanded`,
by the ledger relay once the ledger flushed it, and the chat announces it — see
[Announcements](chat.md#announcements).
**And the ledger keeps it**: the drop clears through `ledger.Recording`, so every
bomb that went off is one event in `planet.ledger_events`, with the flag each tile
it cleared wore, and the flag and the shields each tile it struck kept (`Blast.StruckOwners`, from 2026-10-29: a bomb
before has no owner on its strikes), each strike told as `planet.v1.ShieldStruck` — see [Manual bans](admin.md#manual-bans-findplayers-topplayers-banplayer-unbanplayer-revertplayer-inspectplayer).
**The ban applies**: `refusal_drop_bomb` refuses a refused caller's drop before the
bomb is spent, with the same `BanRefusal` as a click (see [A ban is told](antibot.md#a-ban-is-told)),
so the bomb stays in hand. It sits inside the count so the counter can label it.
`prom_drop_bomb` counts `bonus_bombs_dropped_total{outcome=land|sea|banned|refused}`
and `bonus_bomb_tiles_cleared_total`.
