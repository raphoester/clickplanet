# The mine

**A mine is a charge laid on a tile of the player's flag, a trap only that flag sees** (the design "Player Level and
Unlocks", the owner, 2026-11-16). The player taps Mine, then one of the flag's tiles. When another flag takes that tile,
the mine goes off: the tile and the two rings of tiles around it, about 19 tiles, go to the mine's flag, whoever held
them. A shielded tile loses one shield and keeps its flag. Every mine the blast reaches goes off too, so mines laid
together go off together.

**It is built and dormant**: no box and no right answer is a mine yet, as the bonuses' `kinds.mine` weighs 0 in `default`
and `gates.open.mine` is off there (migration `20261116135208_mine_offers`), and no other set holds either.
`unlocked_kind` has `mine` (`20261116135207_unlocked_mine`), and level 13 unlocks it ([Levels](levels.md#the-unlocks)).
The operator's `GrantCharges` (`mines`) gives some, and so does the fake on the page.

## The rules

From the design:

- **A hand holds 3 mines at most** (`charges.mine.held`), a stack as enclosures: a grant caps at it. A box gives 1 or 2
  once it weighs above 0 (the bonuses' `mine.maxPerBox`, 2).
- **A blast takes `charges.mine.rings` rings** around the mine's tile (2 in `default`, 1 to 3): `clicks.Disc`, a walk on
  the map's own neighbours as a spread's, so it never crosses water. Its own tile first, then the nearest ring.
- **A blast meets each tile as a click of the mine's flag would** (`Tile.Claimed`): a shielded tile loses one shield
  and keeps its flag, whatever its count, and a tile of the mine's flag stays as it is. Every other tile, blank or another
  flag's, goes to the mine's flag, held by the mine's player. Until 2026-11-16 a shielded tile stayed with all its
  shields; the owner asked that a blast break one, as a click, a spread, a virus and a bomb do.
- **One mine a tile**, on a tile of the flag the caller plays for.
- **A bomb takes off the mines it hits, with no blast**, a shielded tile's too.

The owner's answers (2026-11-16):

- **Every take by another flag sets a mine off**: a click, a spread's tile, an enclosure, a virus, and another mine's
  blast. **Any other change of owner takes it off with no blast**: a bomb, a knock-out's clear, an operator's revert.
- **Its blast's takes count for the mine's player, as a virus's** (below).
- **A mine never expires**: it stays until it goes off, a bomb hits it, or its tile changes hands another way.

Chosen where the design is silent:

- **Every mine a blast reaches goes off, whatever its flag**, each for its own flag, in the order the blasts reach them.
  An enemy mine in a blast goes off for the enemy, and may take back some of what the first blast took. The design's
  "a mine inside the blast explodes too" names no flag, and a blast's take of an enemy mine's tile would set it off
  anyway.
- **No mine on a capital whose take knocks its flag out** (while `capitals.knockOut` plays): the take would clear the
  flag's land, the mine with it, before the mine went off. Refused as any other tile it may not go on.
- **The enemy's take counts**: the click took the tile and paid its click, so the ledger keeps it as a take, then the
  mine's event takes it back. The page sees its click land, then the blast, so a click never looks refused.
- **A deleted account's mines leave the map with no blast** (`ForgetHolder`): a blast is recorded under its player's
  account, and a purged one has none. A banned player's mines stay, as a virus does.

## The lay (`LayMine`)

`ClickService/LayMine{tile_id, country_id}`, session-gated and frozen like `UseVirus`, and not throttled. `mines.Gate`
refuses before the mine leaves the hand, in this order: a banned caller (`BanRefusal`), an unknown flag or a tile off the
map (`InvalidArgument`), a flag the caller does not play for (the allegiance's refusal), and a tile not that flag's or a
capital the knock-out plays on (`MineRefusal`, `clicks.ErrNotMineable`). A failed read of the refusals or of the flag
refuses nothing, as for a click. Then `lay_mine_usecase` spends one mine (`Held.AfterMine`; none held is `NotFound`) and
lays it with `clicks.Minefield.Lay`, which checks the tile again under the tiles lock (its flag's, no mine on it): a
mine refused there comes back to the hand (`Held.WithMineBack`, with no cap). A lay changes no tile, so it counts no map
version and no stream of the map hears it.

## The mine is map state

**The tile storage keeps every mine with its tiles** (`inmemory_tile_storage`, `mineBook`), as it keeps the viruses:
the laid ones by tile, the ones a take set off and no explosion took yet, and the tiles whose mine changed since the
last flush. The rules are the domain's, in `clicks`: `clicks.Mine` (where it may go, what its blast does to a tile) and
`clicks.Minefield` (the lay, and the explosion under one lock). The storage applies one rule of the tile, as it does for
a virus: **every change of owner takes the tile's mine off** (`putLocked`), and the bomb's `Clear` takes it off a struck
tile too.

**A take sets it off first.** `clicks.claim`, the one rule every take goes through, calls `Claimable.Trip` before it
writes a take: the storage moves the tile's mine to the ones set off and lights the fuse (`Storage.Fuse`, a channel of
one). So the claim's own change and its events go out as before, whatever the mine does.

## The explosion

`detonate_mines_usecase`, run by `planet-mines` as soon as the fuse is lit, and every second for a mine a failure left
waiting: `ledger.MineRecording.Detonate` takes every mine set off, under one lock of the tiles (`TileStorage.Explode`).
Each blast takes or strikes its tiles nearest first, and every mine it reaches goes off after it. When none is left,
each take that made its landmass whole fortifies it, then each take that felled a capital knocks its flag out. **The
map's freeze does not hold it back**: the take that set the mines off was played before.

- **One `mines_exploded` event per explosion**, at one map version, with each blast in order (the mine's tile, its flag,
  the tiles it took, and the tiles it struck with the shields each kept). Its fortifies and knock-outs follow as their
  own events, as a virus's.
- **The fortifies wait for the last blast.** Two mines of two flags can go off in one explosion. A fortify between their
  blasts would raise a shield that the second blast then struck, but every stream hears the fortify after the whole
  explosion, so the page would raise that shield again. Judged at the end, a landmass a later blast broke earns no
  fortify, and the ledger keeps every blast before the fortifies too.
- **The enemy's take lands first**: its `tile_update` (or its enclosure, spread or virus event) is published under the
  claim's lock, and the explosion under the next one. A write that slips between the two is read by the blast as it is.
- **A mine of a flag knocked out goes with its land**: the clear takes the laid ones off, and the ones set off and still
  waiting are dropped (`dropTripped`), so no land of a flag out comes back.
- **A failure stops the explosion**: what it wrote is kept and told, the mine that failed is lost, and the ones still
  waiting go off at the next second.

## Who sees the mines

**Only the players of the mine's flag**: nothing public carries a mine before it goes off. `GetMap` holds none.

- **`ClickService/GetMines`** answers the mines of the flag the caller's account plays for (`mines_query`, from memory),
  with the map's version as they were read. A caller of no flag reads none. Session-read, not cached.
- **`mine_laid`** reaches only the streams whose caller plays for the mine's flag, through `notices.Board`. A stream
  reads its caller's flag as it opens (`allegiances.Book.FlagOf`; a failed read hears no mine until the next pledge), and
  **every pledge moves every stream of the account to its new flag** (`allegiance_streams`, one of the book's
  `allegiances.Tellers`). It carries the map's version at the lay: the page keeps the mine only while its tile has not
  changed since, so a `mine_laid` that comes after the blast that took it is dropped.
- The page takes a mine off on any change of its tile's owner, and on a bomb's struck tile, as the server does.
- **`mines_exploded` reaches every stream**: the enemy sees the blast that answered its click.

## A restart

**The mines are flushed with the tiles, in the same transaction** (`planet.mines`, by map, each row naming its board
(see [Boards and eras](boards-and-eras.md)): the tile, the flag, the
account and the scope of its player): each mine laid is upserted and each one gone is deleted, by the tiles that changed
since the last flush. The boot reads them back with the tiles (`LoadMines`). **A mine set off in the second before a
hard kill is lost**, with no blast, and a mine laid then too, as a bomb dropped: the hand is postgres, the map is memory.

## Arenas, resets and remaps

- **An arena opens with no mine**: the world's are kept with the world's map, and **come back with it** when the arena
  closes. An arena's mines go with its map.
- **A reset wipes them** with the map (`postgres_tile_store.Wipe`, `emptyLocked`).
- **A remap moves every mine with its tile** (`mines_remapped` in `scripts/map/remap.mjs`), as it moves the tile's flag,
  and drops one whose tile the new map lost. A virus lasts minutes and is ended; a mine may wait days.
- **A switch of the world's board moves each mine to its tile's new tile nearest it** (`Carryover.Heir`), on a tile of
  the same flag, and loses one whose tile the new board draws as sea (`mines_lost` in the dry run): see [Boards and eras](boards-and-eras.md#the-worlds-switch-of-board-switchboard).

## The ledger: kind `mine`, and its takes count

**Each mine gone off that took a tile or struck a shield is one `mine` event** on its tile, its payload `{taken,
struck}` as a spread's, under the mine's player's account and scope (kept with the mine, since an explosion has no
request). Every blast is recorded before the explosion's fortifies, then its knock-outs, the order they were written in. Migration
`20261116135204_ledger_mine` adds the value to `ledger_event_kind`; `20261116135205_ledger_mine_tiles` indexes its
payloads for `GetTileHolder`.

**Its takes reach `TilesTaken`**, as a virus's (the owner, 2026-11-16): stats, the XP from them, the missions that
count takes, scoring's tiles taken, the attack pings and the days played. Every reader of the ledger reads takes through
`Replay`, so a kind that took tiles and told none would break their parity with the ledger. **The difference from a
virus**: a virus lasts two minutes, a mine can wait days, so a mine that goes off on a day its player did not play counts
that day as played, and keeps a streak. A mine goes off once, so it keeps a streak a day or two at most.

`GetLedgerTotals` counts the `mine` events (`mines`). `GetLedger` reads its takes and its strikes as a spread's; the
holder query, `GetTakesByCountry` and `GetPlayDays` read its takes. **Its strikes are told as `ShieldsStruck`**, under
the mine's player, as a virus's.

## Known gaps

- **A banned player's mines stay** and go off for it, as a virus spreads on.
- **The antibot does not hear a blast's takes**: they are not the player's clicks.
- **A mine set off and not yet gone off is lost on a hard kill.**

## Tests

`clicks`' `minefield_test.go`, on the real tile storage: a lay that changes no tile, a tile not the flag's or mined
refused, a take that sets the mine off and its blast as one change nearest first, a shielded tile struck once and the
flag's own left as it is, mines that go off together whatever their flag, a spread's, an enclosure's and a virus's take
setting one off, a bomb and a revert that take it off with no blast, the blast's takes held for its player and the
fortify they earn, and a landmass a later blast broke earning none. `inmemory_tile_storage`'s `mines_test.go`: the flush and the boot, a mine gone with no row, a failed flush kept
for the next, a wipe, an arena, a deleted account, a knock-out (laid and set off). `postgres_tile_store`'s: the rows
upserted and deleted, a wipe, a forgotten holder. The ledger's `mining_test.go` and the relay's `TilesTaken`; the SQL
readers' parity tests each hold a mine. `mines`' gate, `lay_mine_usecase` (the spend, the mine back, none held, a wallet
down, the flag's streams alone told), `detonate_mines_usecase`; `notices`' board (a mine to its flag alone, a pledge that
moves the streams); the stream's flag; `lay_mine_handler`'s codes, `mines_query`, the sink and `GetChargeRules`. The
bonuses' rules pin that `default` weighs the mine 0 and gives 1 or 2.
