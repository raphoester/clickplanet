# A revert erases the takes it forgot

**Every history and replay reads the ledger as if a reverted player had never played, and so does the live map after
the revert** (2026-11-23). On 2026-11-23 a
preview of a clips saga showed, at 20 Nov 17:36 UTC, a swastika in the US Midwest. A clip that shows one can get the
game's TikTok and YouTube banned.

**What happened**: one account painted a swastika (90 tiles, `nov17-us` in the vandal squad's samples) and more around
it for Germany on `earth-1`, 16 Nov 18:45 to 17 Nov 15:35 UTC; the operator reverted it on 17 Nov, and all 133 takes are
`forgotten`. The world switched to `earth-2` on 18 Nov with the clean map. But the season's map history
(`GetMapHistory`) read every take, forgotten or not: it replayed the drawing on `earth-1`, carried it onto `earth-2` at
the cut, and nobody took those tiles of Ireland's after, so it stayed in every frame. On the live map it never
existed on `earth-2`.

## The rule: `ledger.Erasure`

`RevertPlayer` gives the tiles back ([below](#a-revert-gives-back-every-take-of-the-season), and see [Operator tools](admin.md#manual-bans-findplayers-topplayers-banplayer-unbanplayer-revertplayer-inspectplayer))
and then forgets the caller's events: memory keeps the mark (`Forget`), and the flush writes `ledger_forgotten` or
`ledger_forgotten_accounts` and sets `ledger_events.forgotten` on its events, in one transaction. Skipping those events
is not enough: a player who painted over the drawing before the revert took the tiles **from** its paint, and a rewind
opens each tile on what its first change took it from. So the readers saw the whole drawing at the start of the window.

`ledger.Erasure` takes each entry in position order (`Erasure.Read`), with `Entry.Forgotten`:

- **A forgotten event shows nothing.** Each tile it took keeps what it showed before it (`under`), and the paint it
  left (`paint`).
- **A kept event that takes a tile from that paint takes it from what was under it.** A take back to it (Ireland
  painting Ireland's tiles back over the drawing) is no change: no frame, no hit, no scene, no run.
- **A strike, a shield placed or a fortify on that paint is none**: the shields stood on the forgotten paint.
- **A bomb, an enclosure** read their tiles the same way: a cleared tile owns what was under, a tile that was nobody's
  under is not cleared, a shield struck on the paint is not struck.
- **A tile that no longer wears the paint is read as it is**: a revert giving it back and a reset leave no event, and
  the next event's previous owner names what they did. A history's `Wipe` and `Cut` empty the erasure too.
- **Each stage is apart**: an arena's tile is not the world's.

## A revert gives back every take of the season

**A revert makes the live map what every history shows** (2026-11-23, after #930): each tile back to what it would show
had the caller never played. `revert_player_usecase` reads the world's ledger from postgres, from the last reset to now,
on every board (`world_replay`, the reader `GetMapHistory` uses: each era on its board, the carry-over at each switch,
each reset). `ledger.Undoing` plays it twice to its end: with the caller's events, and as if it never made them (each
through the erasure, the caller's own events read whatever an earlier revert forgot of them). A tile of the map in play
where the two differ, and where the live map still shows the first, goes back to the second (`Restore`, a compare-and-set,
paced as before). Then `Forget` marks every event read.

- **An account reaches every take since the reset**, from every scope and on every board: a take of `earth-1` that the
  switch carried onto `earth-2` is given back on `earth-2`'s tiles that still show it.
- **A scope reaches the ledger's last 72 hours**: a take behind the head keeps no address, so nothing names it.
- **`touched`** counts the tiles of the map in play the caller's takes changed, **`held`** those that still show it.
- **Run again, a revert gives back what the one before could not reach**, and nothing more once all is given back.
- **The read is the season's whole ledger**, as `GetMapHistory`'s: a seconds-long seq scan, once per call, dry or not.

**Until then a revert read the ledger in memory, 72 hours, and forgot more than it gave back.** An account's mark
forgets every event of the account in postgres (#731), but the live map kept its takes older than 72 hours. Before #930
no history read the column, so it did not show; after it, the season's history erased takes the live map still held.
Measured on 2026-11-23 against the live map's opening of `earth-2` (clips run 45174314995, tiles no take touched on
`earth-2` since): 3,586 tiles wrong.

- **Cameroon, 2,345 tiles**: a German account took them on `earth-2` on 18 Nov, 16:00 to 18:00 UTC, and was reverted on
  23 Nov. The live map kept Germany; the history showed Sweden, what was under.
- **Ireland in the US, about 830 tiles**: Irish takes of 8 to 11 Nov on `earth-1`, forgotten by a later revert. The
  live map kept them; the history showed what was under them (Latvia, Israel, Brazil, the US).
- **German paint on Irish ground, 218 tiles**: a German drawing of 8 and 11 Nov ("LaS…" and blobs) that nobody
  reverted. The reverted Irish player had painted over it, so the history, with the Irish takes gone, showed it again.
- **84 tiles**: reverts before 2026-11-10 (#731) gave tiles back but set no `forgotten`, so every history still showed
  the reverted takes (76 tiles four alts took for Israel on 6 Nov, which a revert of their address gave back to
  Ireland).

Each one is put right by running its revert again ([Production's rows](#productions-rows)): the live map then shows
what the history shows, "LaS…" included, as if the reverted player never played.

## The readers

| read | who reads it | before 2026-11-23 | now |
| --- | --- | --- | --- |
| `GetMapHistory` (`history_query`, `ledger.History`) | the clips' sagas and paths, the `/archive` replay and heatmap | every take, forgotten or not | erased |
| `GetReplay` (memory `Replay`, `ledger.Footage`) | the clips' replay | forgotten takes skipped, a tile painted over them opened on the drawing | erased |
| `GetRecentMapHistory` (`recent_history_query`, `ledger.Rewind`) | the players' videos | the same, and the answer kept for the step | erased, read from the head, built again after a revert |
| `GetLedger` (`changes_query`) | the clips' cast, the vandal squad, the backoffice Home | every take | the forgotten events left out (`NOT forgotten`) |
| `GetSeasonReplay` (the gamemaster) | `/archive` | `GetMapHistory` kept for good, and a day in the browser | asked again each time, kept until a revert, half an hour in the browser |
| `FindPlayers`, `TopPlayers` (memory `Replay`) | the operator | forgotten takes skipped | erased |
| `RevertPlayer` (`world_replay`, `ledger.Undoing`) | the operator | the memory `Replay`, 72 hours | erased, from the reset, every board |

**A revert of a player who painted over a reverted one gives back what was under the drawing**: before #930, it gave
back the drawing's paint, on the live map. And a take back to the paint under a drawing counts no take in `TopPlayers`
and `FindPlayers`.

The Hall of Fame's race reads scoring's points and shares, and draws no map: nothing to erase.

## The caches

**What the reverts forgot is named by `forgetting.Forgetting`** (`planetv1controller/forgetting`): a hash of every mark
in `ledger_forgotten` and `ledger_forgotten_accounts`, which names no caller. A revert that forgets more changes it,
and so does the head passing a mark, 72 hours after its revert (one build more, for nothing).

- **`GetRecentMapHistory` keeps its answer by step and forgetting**: a revert builds the step again at the next call.
- **`GetMapHistory` answers `kept`** (`GetMapHistoryResponse.kept`, no history) when the caller sends the forgetting of
  the history it keeps (`kept_forgetting`) and no revert forgot more. The gamemaster asks it at each `GetSeasonReplay`,
  so an ended season's map, which a revert of an account changes too, is read again only after a revert.
- Both read the forgetting before the history, so a revert during a build makes the next call build again.

## What is left

- **The posts already made.** A clip posted before the revert is on TikTok and YouTube: take it down by hand.
- **A browser's copy**: `GetRecentMapHistory` up to its step, half an hour; `GetSeasonReplay` half an hour.
- **A revert by scope** reaches the last 72 hours alone: revert a vandal by its account.
- **A take forgotten more than 72 hours before a rewind's start, painted over inside its window**: the 24 hours and
  `GetReplay` erase from the ledger's head, so they open the tile on its paint. The season's history reads from the
  reset, and sees through it.
- **A swastika nobody reverted**: the vandal squad flags what it sees, and the operator reverts its painter.
- **A reset after a forgotten bomb** that left a tile blank, retaken after the reset, reads it taken from what was under
  the paint before the reset: a flag, never the drawing.

## Cost

The 24 hours read the ledger from the head (72 hours, about 470,000 rows) and no longer from the window's start, to see
through a drawing older than the window: about three times the rows sent, once per step. Not measured on production.
An event the erasure reads costs three more walks of it while a forgotten paint is not painted over, and none
otherwise.

## Deploy

One backend deploy. No migration and no rebuild of stored frames: no frame is stored. Each history is built at its
next read; `/archive`'s from the boot on, as after every deploy. The frontend and the backoffice only regenerated
`planet/v1/internal` bindings, which they do not call.

**The revert of the whole season** (2026-11-23): one backend deploy, no migration. The frontend and the backoffice
regenerated `planet/v1/admin` bindings for `RevertPlayer`'s description, which the connector reads too.

## Production's rows

After that deploy, through the connector, each revert first dry, its counts read, then run:

- **Every account reverted since the season's reset, run again**: it gives back on the live map what the first revert
  could not reach, and forgets what a revert before #731 never marked. The four alts of 6 Nov's revert by address, by
  their accounts.
- **The painters of the swastikas the season's history still showed, reverted**: a scan of every hourly frame with the
  vandal squad's reading found seven, on `earth-1` from 2 to 8 Nov, painted by six accounts nobody had reverted (one had
  been, before #731). Four more readings were players painting one over, in the flag of the land around, which shows
  nothing; each goes with its swastika.

## Tests

`ledger/erasure_test.go` (a take from the paint, a take back to what was under, a strike, a shield and a fortify, a
tile given back or reset, a paint painted twice, a bomb and an enclosure, a wipe and another stage), `history_test.go`
and `rewind_test.go` (a drawing in no frame and no hit, one reverted before a switch of board not carried over), `revert_player_usecase` (the paint never back on the live map),
`replay_usecase` (the replay of a drawing painted over), the postgres suites of `history_query`, `recent_history_query`
(the drawing before the day, the step built again after a revert), `changes_query` and `forgetting`, the
`get_map_history_handler`'s `kept`, and `season_replay_query` (kept until a revert).

The revert of the whole season's: `ledger/undoing_test.go` (two scopes on one flag, a bomber, a bomb of another scope, a
run back to where it started, an account across scopes, an arena's take, a revert run again, a reverted player's paint
under the caller's, a take carried over a switch of board, a reset), and `revert_player_usecase`'s (the batches, a
dry run, the interleaving, an interrupted run, an account, a bomb, an arena, a switch of board, a revert run again).
`history_query`'s postgres suite reads through `world_replay`.
