# Shields

**A shield is a charge a player places on a tile its flag holds, and each one
takes one foreign click.** A tile holds up to `bonus.shield.perTile` (10). While
any stand, a click for another flag strikes one and the tile keeps its flag; with
none left, the next click takes it.

**One click is one token, always.** A strike is an accepted click: it spends from
both buckets, sets the pace, counts in `clicks_total{status="ok"}` and is reported
to the jury.

- **A tile's shields are part of the tile**, kept by `inmemory_tile_storage` in a
  `[]uint8` beside the owners, under the same lock. **Every change of owner sets them
  to 0** — a click, a spread, an enclose, a bomb, `ReassignCountry`, `RevertPlayer`,
  `PaintRandomTiles` — so no shield ever stands on a tile for a flag that lost it,
  and no write path has to know about them. `Strike(tile, owner)` and
  `Shield(tile, country, most)` check the owner under that lock, so a strike or a
  placement racing a capture finds the new owner.
- **The rule is `clicks.OutcomeOf(owner, flag, shields)`**, pure and tested in the
  `clicks` root: `Unchanged` when the tile wears the flag, `Shielded` when a shield
  stands, `Taken` otherwise. `clicks.Shielding` is the rule over the storage: `Outcome`
  foresees it and spends nothing (`antibot_click`), and `Strike` spends the shield.
  A strike that lost the race for the last shield is a take.
- **`clicks.Claiming` is the one way a player's click reaches a tile**: it strikes,
  writes the owner the outcome leaves, and answers a `clicks.Impact` (the tile, its
  owner before, the outcome and the shields standing after). `ledger.Recording` calls
  it for the tile clicked, each neighbour of a spread and each tile of an enclosure,
  and records what it answered. `clicks.ShieldError` is what a placement is
  refused for (`ErrNotYourTile`, `ErrTileFull`), read by the use case before it spends
  and by the storage under its lock.
- **A shielded click writes no owner**: `OwnerAfter` is the owner, so the write is a
  no-op and records no take. The strike publishes the tile's new count instead: a
  `TileUpdate` whose `Value` is its `Previous`, with `Shields` set.
- **Every `TileUpdate` carries the tile's shields after it** (0 on any change of
  owner), and every `GetMap` batch the shielded tiles in its range
  (`DenseBatch.Shields`, `GetMapResponse.shields`). So they ride the one channel
  the map does, in its order, and a client needs no read of its own.
- **Placing one is `PlaceShield(tile, country)`** (`bonuses/usecases/place_shield_usecase`).
  A tile not wearing the flag (`ErrNotYourTile`) and a full one (`ErrTileFull`) are
  `FailedPrecondition` and spend nothing; no shield held is `NotFound`; an unknown
  country or tile is `InvalidArgument`. A placement the tile refuses after the
  charge was spent (it changed hands in between) gives the shield back. It is
  session-gated like `DropBomb` and not throttled. `antibot_place_shield` refuses a
  banned caller before anything is spent, as `ban_click` does a click (see
  [A ban is told](antibot.md#a-ban-is-told)). The answer is the charges left.
- **Kept with the tile**: `planet.tiles.shields`, written by the flush that writes
  the owner.
- **`click_usecase.Out.Outcome`** says what the rule did, for the decorators around
  it: `enclose_click` closes a shape only with `Taken`, and `prom_click`
  counts `clicks_shielded_total{country_id}` by the flag clicked. **It never reaches
  the wire**. The client predicts it from the shields it holds.

What a spread, an enclose, a bomb and the antibot do with a shielded tile is in their
own sections: [spread](spread.md), [enclose](enclose.md),
[bomb](bomb.md), [antibot](antibot.md#the-parts-that-are-easy-to-get-wrong).
