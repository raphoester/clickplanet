# Shields

**A shield is a charge a player places on a tile its flag holds, and each one
takes one foreign click.** A tile holds up to the rule set's `bonuses.shield.perTile` (10); a tile above a smaller one keeps its shields and takes none until it is under it. While
any stand, a click for another flag strikes one and the tile keeps its flag; with
none left, the next click takes it.

**One click is one token, always.** A strike is an accepted click: it spends from
both buckets, sets the pace, counts in `clicks_total{status="ok"}` and is reported
to the jury.

- **A tile's shields are part of the tile**: `clicks.Tile` is its owner, its shields and
  its holder, kept by `inmemory_tile_storage` in one `tileState` per tile, under one lock.
  **Every change of owner sets them to 0** — a click, a spread, an enclose, a bomb,
  `ReassignCountry`, `RevertPlayer`, `PaintRandomTiles` — so no shield ever stands on a
  tile for a flag that lost it.
- **The rule is `clicks.Tile.Claimed(flag, by)`**, pure and tested in the `clicks` root,
  over `clicks.OutcomeOf(owner, flag, shields)`: `Unchanged` when the tile wears the flag,
  `Shielded` when a shield stands (one struck, the flag and the holder kept), `Taken`
  otherwise (the flag, no shield, held by `by`).
- **`clicks.Claiming` is the one way a player's click reaches a tile**: under one lock
  of the tiles it reads the tile, asks `Claimed`, writes what it answered, and answers a
  `clicks.Impact` (the tile, its owner before, the outcome and the shields standing
  after). So a placement cannot land between the read and the write: it waits, then is
  struck by the claim or refused (`ErrNotYourTile`). See [A claim is one lock](tile-claim.md).
  `ledger.Recording` calls it for the tile clicked, each neighbour of a spread and each
  tile of an enclosure, and records what it answered. `clicks.ShieldError` is what a
  placement is refused for (`ErrNotYourTile`, `ErrTileFull`), read by the use case
  before it spends and by the storage under its lock.
- **A shielded click writes no owner**, and records no take. It publishes the tile's new
  count: a `TileUpdate` whose `Value` is its `Previous`, with `Shields` set.
- **Every `TileUpdate` carries the tile's shields after it** (0 on any change of
  owner), and every `GetMap` batch the shielded tiles in its range
  (`DenseBatch.Shields`, `GetMapResponse.shields`). So they ride the one channel
  the map does, in its order, and a client needs no read of its own.
- **Placing one is `PlaceShield(tile, country)`** (`bonuses/usecases/place_shield_usecase`).
  A tile not wearing the flag (`ErrNotYourTile`) and a full one (`ErrTileFull`) are
  `FailedPrecondition` and spend nothing; no shield held is `NotFound`; an unknown
  country or tile is `InvalidArgument`. A placement the tile refuses after the
  charge was spent (it changed hands in between) gives the shield back, with no cap,
  and a stop between the spend and the map's flush gives it back at the next boot:
  see [A shield's way from the hand to the tile](placements.md). It is
  session-gated like `DropBomb` and not throttled. `refusal_place_shield` refuses a
  refused caller before anything is spent, as `refusal_click` does a click (see
  [A ban is told](antibot.md#a-ban-is-told)). The answer is the charges left.
- **Kept with the tile**: `planet.tiles.shields`, written by the flush that writes
  the owner.
- **Told to the other modules** as `planet.v1.ShieldPlaced`, with the tile's shields after it, by the ledger relay once
  the ledger flushed the placement. A fortify's shields are not told this way.
- **Every strike is told too**, as `planet.v1.ShieldsStruck`, by the ledger relay: one fact per ledger event, with the
  flag that struck and, for each tile, the flag whose shield it was (`owner`) and its index in its event. A click strikes
  one; a spread, an enclose and a bomb (`Blast.StruckOwners`) may strike several at one position
  (`ledger.Event.Strikes`), told together.
- **`click_usecase.Out.Outcome`** says what the rule did, for the decorators around
  it: `enclose_click` closes a shape only with `Taken`, and `prom_click`
  counts `clicks_shielded_total{country_id}` by the flag clicked. **It never reaches
  the wire**. The client predicts it from the shields it holds.

What a spread, an enclose, a bomb and the antibot do with a shielded tile is in their
own sections: [spread](spread.md), [enclose](enclose.md),
[bomb](bomb.md), [antibot](antibot.md#the-parts-that-are-easy-to-get-wrong).
