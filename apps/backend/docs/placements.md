# A shield's way from the hand to the tile

**A shield leaves the hand in postgres and lands on the map in memory, so no one transaction holds both.** A placement
(`internal/planet/internal/placements`) is the shield between the two: a row in `planet.placements` from the spend until
the map saves the shield, or until the shield goes back to the hand. So a shield is never lost, and never given twice.

Why not one step:

- **The map step cannot run in the postgres transaction.** One lock guards the whole map. Held across a database call, a
  slow database would stop every click.
- **The map step cannot come first.** Every player would see a shield that can then go away, and a click could break it
  in between.

## The steps

`place_shield_usecase` only orchestrates them:

1. **The gate** (`placements.Gate.Aim`) refuses before anything is spent: an unknown flag, a tile out of range, a tile
   of another flag or a full one (`clicks.ShieldError`), and a flag the caller does not play for. It reads the rule set
   in force once, and hands the map the most shields a tile may hold. `refusal_place_shield` refuses a refused caller
   before it, as `refusal_click` does a click.
2. **The take** (`placements.Escrow.Take`): one transaction under the account's lock spends the shield
   (`Held.AfterShield`) and keeps a placement. No shield held is `bonuses.ErrNotHeld`, and nothing is kept.
3. **The map** (`Storage.Shield`, through `ledger.Recording`) checks the tile again under the tiles lock, adds the
   shield, sends the `TileUpdate`, and keeps the placement's id beside the tile's dirty bit. It answers the shields the
   tile holds, which the ledger's `shield` event records.
4. **The flush** that saves the tile ends the placement, in the same transaction (`Escrow.Confirm`, the storage's
   `Placements` port). A failed flush keeps the ids with the dirty bits. `Wipe` and `PutBack` flush first, so the
   shields placed before a reset or an arena's end are confirmed too.
5. **A refusal by the map** (the tile changed hands between the gate and the lock) returns the shield
   (`Escrow.Return`): one transaction ends the placement and gives the hand its shield back with `Held.WithShieldBack`.
   **The return has no cap**: it grants nothing, it puts back the shield step 2 took. So a hand above a stack size that
   a battle's set made smaller gets its shield back. A grant keeps its cap (`Held.Granted`).

The answer is what the hand holds after step 2, or the map's refusal once step 5 committed. The wire did not change.

## A stop between the steps

- **A placement ends only in the transaction that saves its shield to `planet.tiles`.** So a placement left at boot is
  a shield `planet.tiles` does not hold: the map never took it, refused it, or took it and was killed before its flush.
- **The boot gives every one back** (`return_placements_usecase`), in the module's build, before any request. A failure
  is logged and leaves the rows for the next boot: none of them is a shield this boot placed.
- **It needs one server at a time**, as the map does ([Durability](durability.md)). The deploy stops the old server
  before it starts the new one.
- **Each step is safe to run again.** `Return` reads the placement under a row lock and changes only one still under
  way, so the request and then the boot give one shield. A flush run again ends nothing new.
- **What stays open**: another tab that reads `GetCharges` in the millisecond between the take and a return sees one
  shield fewer. A grant in that millisecond, on a hand at its size, leaves the hand one above the size after the return:
  a hand the rules keep ([Charges](charges.md)).

## Kept

`planet.placements` holds only the shields under way: `id`, `account`, `taken_at`. **One that ended is no row**, as an
empty hand is no row in `planet.charges`: nothing reads it, and it would keep an account id. `placements.StoreContractSuite`
runs on postgres and on `MemoryStore`.

## Not the other charges

A bomb, a spread and an enclose spend, then act on the map, but **their map step never says no**: `Clear` fails only on a
tile out of range, and a spread or an enclose strikes a shielded tile rather than refuse it. So in normal running there is
nothing to give back. Their one gap is a hard kill in the second between the spend and the flush, which every take shares
([Durability](durability.md)).
