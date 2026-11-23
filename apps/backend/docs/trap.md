# The trap

**A trap is a charge laid on a tile of the player's flag, which only that flag sees** (the owner, 2026-11-23). It is
the mine's sibling: laid the same way, but when it goes off it takes clicks, not land. The player taps Trap, then one of
the flag's tiles. When another flag takes that tile, the trap springs: the player whose take it was loses
`charges.trap.clicks` clicks (10) from its bank, and the taker keeps the tile. The trap is gone.

**It is built and dormant**: no box and no right answer is a trap yet, as the bonuses' `kinds.trap` weighs 0 in
`default` and `gates.open.trap` is off there (migration `20261123140201_trap_offers`), and no other set holds either.
`unlocked_kind` has `trap` (`20261123140200_unlocked_trap`), and level 15 unlocks it, 38,416 XP, about a month for a
regular player, before the rally at 17 ([Levels](levels.md#the-unlocks)). The operator's `GrantCharges` (`traps`) gives some, in
the backoffice and through the connector too.

## The rules

From the owner:

- **A hand holds 3 traps at most** (`charges.trap.held`), a stack as the mines. A box gives 1 or 2 once it weighs above
  0 (the bonuses' `trap.maxPerBox`, 2).
- **A tile holds one trap and one mine at most**, on a tile of the flag the caller plays for, checked under the tiles
  lock (`Laying.Trapped`, `Laying.Mined`). The owner asked for both on 2026-11-23 ("it would be fun"): an enemy's take
  of such a tile springs the trap and sets the mine off, so it loses clicks and the tiles around it. Until then the
  brief kept one or the other.
- **Any take by another flag springs it**: a click, a spread's tile, an enclosure, a virus, a mine's blast. A click a
  shield stops takes nothing, so it springs nothing.
- **The player whose take it was loses `charges.trap.clicks`** (10, at least 1) from its own bank and, for a guest, from
  its network's guest bank (`guests:<scope>`): a new private tab would escape the debt otherwise. The network's bank
  (`scope:`), which every account behind it shares, stays as it is.
- **The clicks go to nobody.** To the trap's player, a second account on another flag could feed a bank.
- **The bank may go below 0, down to minus one bank** (−`clicks.burst` for the account's own, −60 in production).
  Players at war spend each click as it refills, so their bank sits near 0: a floor at 0 would take almost nothing.
- **Any other change of owner takes it off with no effect**: a bomb (the counterplay; a shielded tile's too), a
  knock-out's clear, a cut-off, an operator's revert. **It never expires.**

Chosen where the owner was silent:

- **A capital takes a trap, while the knock-out plays too.** The mine refuses one there, because the knock-out clears
  the flag's land before the explosion runs. A trap springs inside the take's own claim, before the knock-out, so the
  reason does not hold.
- **A deleted account's traps leave the map** (`ForgetHolder`), as its mines do. A banned player's traps stay.
- **A take with no request pays as the player kept with it**: a virus's tick, as its player used it, and a mine's
  blast, as its player laid it. Each keeps its account, its scope and whether it was signed in
  (`20261123132922_linked_payers`: `linked` on `mines` and `viruses`; a row from before reads as a guest's).

## The lay (`LayTrap`)

`ClickService/LayTrap{tile_id, country_id}`, session-gated and frozen like `LayMine`, and not throttled. `traps.Gate`
refuses before the trap leaves the hand, in this order: a banned caller (`BanRefusal`), an unknown flag or a tile off
the map (`InvalidArgument`), a flag the caller does not play for (the allegiance's refusal), and a tile not that flag's
(`TrapRefusal`, `clicks.ErrNotTrappable`). A failed read of the refusals or of the flag refuses nothing, as for a
click. Then `lay_trap_usecase` spends one trap (`Held.AfterTrap`; none held is `NotFound`) and lays it with
`clicks.Trapline.Lay`, which checks the tile again under the tiles lock (its flag's, no trap on it, a mine or not): a trap
refused there comes back to the hand (`Held.WithTrapBack`, with no cap). A lay changes no tile, so it counts no map
version and no stream of the map hears it.

## The spring is in the claim

**The tile storage keeps every trap with its tiles** (`inmemory_tile_storage`, `trapBook`): the laid ones by tile, and
the tiles whose trap changed since the last flush. The storage applies one rule of the tile, as for a mine: **every
change of owner takes the tile's trap off** (`putLocked`), and the bomb's `Clear` takes it off a struck tile too.

**A take springs it first.** `clicks.claim`, the one rule every take goes through, and a mine's blast, call
`Claimable.Trip` before they write a take: the storage sets off the tile's mine and takes its trap off the map and
answers it.
The take's `taker` then has its player pay: `clicks.Snare.Spring(payer, trap)`, and the take's `Impact` keeps the trap
(`Sprung`). So the trap leaves the map and the bank pays under the same lock of the tiles.

- **`traps.Snare` drains the bank** (`cpratelimit.Limiter.Drain` on `clicks.Buckets.Bank(payer)`: the account's own
  bucket, and `guests:<scope>` for a guest) and tells every stream (`notices.Board.PublishTrapSprung`), with what left
  the payer's own bank: 10, or less near the floor. `log_snare` logs each one.
- **The payer is the take's player**: `PayerOf(ctx)` for a click, a spread and an enclosure; the virus's kept player
  for a virus (`VirusSnapshot.Payer`), its first take too; the mine's kept player for a blast (`MineSnapshot.Payer`).

## The click's answer shows the drain

**A click that springs a trap answers with the drained bank.** `ClickResponse.budget` is read before the click
(`throttle_click`), and read again after it when the click's takes sprang one (`click_usecase.Out.Sprung`: the click's
own tile, and the spread's and the enclosure's tiles, which `ledger.Recording.Spread` and `.Enclose` answer). The
bank may read below 0: `ClickBudget.tokens` is negative while in debt, and the page shows it.

A take with no request (a virus's tick, a mine's blast) has no answer: the taker's own streams hear the `trap_sprung`
with `yours` set, and the page reads its bank again (`GetBudget`).

## The limiter's debt

`cpratelimit.Limiter.Drain(n, keys...)` takes `n` from each bucket down to minus its capacity, and answers what left
each one and what it holds after. It makes and keeps a bucket it does not know (a full one, at a new account's start),
so the debt is kept. It never raises a bucket: a smaller burst may leave a debt deeper than the new floor, which a drain
does not pay back. The refill pays the debt back as it pays any bank ([Rate limiting](rate-limiting.md#a-trap-leaves-a-debt)).
A take refuses a bank in debt (`tokens < n`), the sweep keeps it (it is not full), and a refill (`Fill`) pays the whole
debt at once.

## Who sees the traps

**Only the players of the trap's flag**: nothing public carries a trap before it springs. `GetMap` holds none.

- **`ClickService/GetTraps`** answers the traps of the flag the caller's account plays for (`traps_query`, from memory),
  with the map's version as they were read. A caller of no flag reads none. Session-read, not cached.
- **`trap_laid`** reaches only the streams whose caller plays for the trap's flag, through `notices.Board`, as
  `mine_laid` does: each stream knows its caller's flag, and a pledge moves it. It carries the map's version at the lay.
- **`trap_sprung` reaches every stream**: the tile, the trap's flag and the clicks taken, so every page that sees the
  tile shows the snap. It names no account. The taker's own streams get it with `yours` set.

## No ledger kind

**A sprung trap changes no tile**, so it has no ledger kind. The ledger is the log of the tiles' changes, which every
history, replay and count reads as takes; the take that sprang the trap is already in it, as a click, a spread, an
enclosure, a virus or a mine. What the trap does is a debt in the limiter, which lives in memory, as every bank does.
`log_snare`'s line (`trap sprung`, with the payer, the tile, the flag, the trap's player and the clicks) is its record.

## A restart

**The traps are flushed with the tiles, in the same transaction** (`planet.traps`, by map, each row naming its board:
the tile, the flag and the account of its player). The boot reads them back with the tiles (`LoadTraps`). A trap laid
in the second before a hard kill is lost, as a mine. **A bank's debt is lost on a restart**, as every bank: the
limiter is memory, and a restart fills every bank.

## Arenas, resets and remaps

As the mines:

- **An arena opens with no trap**, and the world's **come back with its map** when the arena closes.
- **A reset wipes them** with the map (`postgres_tile_store.Wipe`, `emptyLocked`).
- **A remap moves every trap with its tile** (`traps_remapped` in `scripts/map/remap.mjs`), and drops one whose tile
  the new map lost.
- **A switch of the world's board moves each trap to its tile's new tile nearest it** (`Carryover.Heir`), and loses one
  whose tile the new board draws as sea (`traps_moved` and `traps_lost` in `SwitchBoard`'s answer).

## Tests

`cpratelimit`'s `drain_test.go`: a drain from a full bank, below 0, stopped at minus one bank, a debt the refill pays
back and a take refuses, a bucket made and kept, a new account's start, each bucket at its own scale, no pace set, a
smaller burst never paid back, a fill that pays the whole debt. `clicks`' `trap_test.go`: a lay that changes no tile, one
trap and one mine a tile, a take of both that springs the trap and sets the mine off, a click that springs it and its player pays, a click a shield stops springing nothing, a
spread's, an enclosure's, a virus's and a mine's blast's take springing it with each payer, a bomb and a revert taking it
off with no spring. `inmemory_tile_storage`'s `traps_test.go`: the flush and the boot, a row gone, a failed flush, a
wipe, an arena, a deleted account, a knock-out; the switch's trap lost. `postgres_tile_store`'s: the rows upserted and
deleted, a wipe, a forgotten holder, the carry. `traps`' gate, `Snare` (a guest's two banks, a signed-in player's one,
the floor, the event), `lay_trap_usecase`; `notices`' board (a trap laid to its flag alone, a trap sprung to every stream
and `yours` to the taker's); `throttle_click`'s answer read again; `spread_click`'s and `enclose_click`'s sprung traps;
`lay_trap_handler`'s codes, `traps_query`, the sink and `GetChargeRules`. The bonuses' rules pin that `default` weighs
the trap 0 and gives 1 or 2.
