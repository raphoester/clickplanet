# The virus

**A virus is a charge that takes one enemy tile, then spreads on its own** (the design "Player Level and Unlocks", the
owner, 2026-11-16). The player taps Virus, then an enemy tile: the tile becomes the player's flag's and is infected.
Every 20 s each infected tile infects one enemy tile next to it, of any flag, so it doubles: 1, 2, 4, 8, 16, 32. A
defender who takes an infected tile back cures that tile, and it spreads no more.

**It is built and dormant**: no box and no right answer is a virus yet, as the bonuses' `kinds.virus` weighs 0 in
`default`, `gates.open.virus` is off there (migration `20261116122504_virus_offers`), and no other set holds either.
`unlocked_kind` has `virus` (`20261116122503_unlocked_virus`), so the bonuses can keep it unlocked. The operator's
`GrantCharges` (`virus`) gives one, and so does the fake on the page.

## The rules (the owner, 2026-11-16)

- **The cap counts every infection**, a tile taken or a shield struck, the first one included (`charges.virus.infections`,
  32 in `default`, 1 to 1,024). A virus is at most 32 "clicks", so it always ends, against shields too. The design said
  64; the owner launched it at 32.
- **A spread every `charges.virus.every`** (20 s in `default`, 1 s to 10 min), from the tiles the virus infected before
  that tick. A tile it takes at a tick spreads from the next one.
- **Each infected tile picks one enemy tile next to it, of any flag, at random** (`clicks.Contagion.target`). A shield
  absorbs the infection as it absorbs a click: one shield struck, the tile keeps its flag, and the infection counts.
- **Any enemy flag, never a blank tile**: a virus crosses borders. It spreads into the land of every flag next to its
  tiles but its own, not only the flag of the tile it was aimed at (`victim`, kept for the stream). **Why** (the owner,
  2026-11-16): the first rule, the flag it hit alone, stopped a virus at the first border. It never crosses water: it
  walks `Geography.Neighbours`.
- **It ends** at the tick it infects nothing (no enemy tile next to any tile it holds), when no tile it
  infected is still its flag's, or once it made its most infections. The end is told: every page stops drawing it.
- **The aim must be an enemy tile with no shield**, else the virus stays in the hand: `FailedPrecondition` with a
  `planet.v1.VirusRefusal` (`clicks.ErrNotInfectable`). A blank tile has no flag to spread into.
- **A hand holds `charges.virus.held` viruses** (1 in `default`), as bombs, each used alone. Until 2026-11-16 a virus
  was held or not, and a second grant replaced the first.
- The values are read at each use and each tick, so a smaller cap stops a live virus at its next tick.

## The use (`UseVirus`)

`ClickService/UseVirus{tile_id, country_id}`, session-gated and frozen like `DropBomb`, and not throttled: holding a
virus the server granted is the gate. `viruses.Gate` refuses before the virus leaves the hand, in this order: a banned
caller (`BanRefusal`), an unknown flag or a tile off the map (`InvalidArgument`), a flag the caller does not play for (as a
shield), a tile out of the reach while it plays (`ReachRefusal`), and a tile of no other flag or with a shield
(`VirusRefusal`). A failed read of the refusals or of the flag refuses nothing, as for a click. Then
`use_virus_usecase` spends the virus (`Held.AfterVirus`; none held is `NotFound`) and starts it on the map, which checks
the tile again under its lock: a shield placed meanwhile gives the virus back (`Held.WithVirusBack`).

## The virus is map state

**The tile storage keeps every live virus beside the board**, as it keeps the flags out (`inmemory_tile_storage`,
`virusBook`): a virus spreads only on the map it was used on, so it lives and dies with that map. The rules are the
domain's, in `clicks`: `clicks.Virus` (its tiles, its infections, when it is due, when it ends) and `clicks.Contagion`
(the use and each tick, under one lock of the tiles, through `TileStorage.Infect`). The storage only holds the viruses
and applies one rule of the tile, as it does for shields: **every change of owner cures the tile** (`putLocked`), a
click's, a spread's, an enclosure's, a bomb's, a knock-out's, an operator's revert. So a cure needs no event, and a tile
retaken by a plain click is not infected again.

`viruses/` holds the virus's way from the hand (`Gate`, `Outbreak`: the rules in force and the clock) and its two use
cases. It cannot hold the domain itself, since `clicks`, which keeps the map, cannot import it.

## The runner

`spread_viruses_usecase`, run every second (`planet-viruses`, logged by `log_spread_viruses`): every virus due spreads
once, through `ledger.VirusRecording`. **None spreads while the map takes no play** (the set's `frozen`, or an arena's
freeze): a virus waits through a freeze and spreads at the first second after it. A virus that left the map meanwhile is
no failure. One that fails is logged and tried again the next second; the others spread.

## A restart

**A virus is flushed with the tiles, in the same transaction** (`planet.viruses`, by map: the account, the scope, the two
flags, the tile it was used on, the tiles it infects, its infections and its next tick). The map's flush writes every
live virus of the map whenever one changed, in place of the ones kept before, so a virus that ended has no row. The boot
reads them back with the tiles (`LoadViruses`). A hard kill loses at most the last second, of the tiles and the viruses
together, so they never disagree. A virus whose tick was due during the stop spreads at the first second after the
boot. **A virus spent in the second before a hard kill is lost**, as a bomb dropped then: the hand is postgres, the map is
memory.

## Arenas, resets and remaps

- **An arena opens with no virus**: the world's viruses are saved with the world's tiles, and the arena's map has none.
  **They come back with the world's map** when the arena closes, tile for tile, and spread at once.
- **An arena's viruses go with its map** when it closes (`Wipe` on the arena's map deletes them).
- **A reset wipes them** with the map (`postgres_tile_store.Wipe`, `emptyLocked`).
- **A remap ends every live virus** (`DELETE FROM viruses` in `scripts/map/remap.mjs`): it lasts minutes, and the tiles it
  took keep their flag.

## The other rules on the map

A virus's takes are claims like a spread's (`clicks.claim`): **a take that makes a landmass whole fortifies it**, and
**a take of a capital knocks its flag out** while the knock-out plays, once every take of the tick is written, as for an
enclosure. **A capital whose shields the rules fix** keeps them: the virus strikes them like a click. **The reach** is
checked on the aim; a spread is always next to the flag's own infected tile.

## The stream and the map load

**One `virus_spread` event per use or tick**, at one map version, and no `tile_update` for its tiles (one event per server
effect, as `tiles_enclosed`): the virus, the two flags, the tiles it took (each infected now), the shields it struck with
what they kept, and `ended`. Its fortifies and knock-outs follow as their own events. A virus that ends with nothing
infected sends one with `ended` and no tile. **`GetMap` lists the infected tiles of its range** (`infected`, each with its
virus), so a page loaded mid-virus draws them. The page drops a tile's infection on any change of its owner, as the
server does.

## The ledger: kind `virus`, and its takes count

**Each use and tick that changed a tile is one `virus` event** on the tile the virus was used on, its payload
`{taken, struck}` as a spread's, under the virus's player's account and scope (kept with the virus, since a tick has no
request). Migration `20261116095722_ledger_virus` adds the value to `ledger_event_kind`;
`20261116095723_ledger_virus_tiles` indexes its payloads for `GetTileHolder` (a GIN of its own, so no index of the
table is built again on a deploy).

**Its takes reach `TilesTaken`**, as a spread's and an enclosure's do: the relay tells every take `Event.Replay` makes, and
its strikes as `ShieldsStruck`. So they count in progression's stats (and the XP from them), in the missions that count
takes, in scoring's tiles taken, in the attack pings, and in the days played. **Why**: the virus takes tiles for the
player's flag with a charge the player spent, exactly as a spread does, and every reader of the ledger (the holder card,
the replays, the map history, `GetTakesByCountry`, `GetPlayDays`, `GetLedger`, a revert) reads takes through `Replay`. A
kind that took tiles and told no `TilesTaken` would break the parity those readers and progression and scoring keep with
the ledger. **The side effect**: a virus that spreads past midnight UTC counts a day played for its player, who may not
have clicked that day. A virus lasts about two minutes, so it is rare.

`GetLedgerTotals` counts the `virus` events (`viruses`). The holder query, `GetTakesByCountry`, `GetPlayDays` and
`GetLedger` read its payload as a spread's.

## Known gaps

- **A ban or a deleted account does not stop a live virus**: it spreads to its end, for an account the antibot refused or
  the purge forgot. A virus lasts minutes, and a revert of the player undoes its takes with the rest.
- **The antibot does not hear a virus's takes** as clicks, as for a spread: they are not the caller's clicks.

## Tests

`clicks`' `virus_test.go` pins the aim, the infections, the end and the cure; `contagion_test.go`, on the real tile
storage, the use as one change, a refused aim that changes nothing, the doubling, any enemy flag's land and never a blank tile, a shield that
absorbs and counts, the cap and the end told, a cure, a virus with nowhere to go, and its takes held for its player.
`inmemory_tile_storage`'s `viruses_test.go` the flush and the boot, a virus that ended with no row, a flush with no virus
changed, a bomb's cure, a wipe and an arena; `postgres_tile_store`'s the rows of each map, replaced as a whole and
wiped with their map. `viruses`' gate, `use_virus_usecase` (the spend, the virus given back, none held, a wallet down) and
`spread_viruses_usecase` (due only, none while frozen, one failure leaves the others). The ledger's `infecting_test.go`
the replay, the strikes, the entry and the recording; the relay's `TilesTaken`; the SQL readers' parity tests each hold a
virus. `use_virus_handler`'s codes and details; the sink, `GetMap` and `GetChargeRules`. The bonuses' rules pin that
`default` weighs the virus 0.

## Its level

Level 10 unlocks it (`VirusUnlock`, [Levels](levels.md#the-unlocks)), and the bonuses' gates (#829) draw it only for an
account that unlocked it, once they are on. Left: a small weight in `kinds.virus`, in `default` and in the battles' sets
that want it.
