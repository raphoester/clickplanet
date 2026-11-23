# A revert erases the takes it forgot

**Every history and replay reads the ledger as if a reverted player had never played** (2026-11-23). On 2026-11-23 a
preview of a clips saga showed, at 20 Nov 17:36 UTC, a swastika in the US Midwest: the season's map history
(`GetMapHistory`) read every take, forgotten or not. A clip that shows one can get the game's TikTok and YouTube banned.

## The rule: `ledger.Erasure`

`RevertPlayer` gives the tiles back (see [Operator tools](admin.md#manual-bans-findplayers-topplayers-banplayer-unbanplayer-revertplayer-inspectplayer))
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

## The readers

| read | who reads it | before 2026-11-23 | now |
| --- | --- | --- | --- |
| `GetMapHistory` (`history_query`, `ledger.History`) | the clips' sagas and paths, the `/archive` replay and heatmap | every take, forgotten or not | erased |
| `GetReplay` (memory `Replay`, `ledger.Footage`) | the clips' replay | forgotten takes skipped, a tile painted over them opened on the drawing | erased |
| `GetRecentMapHistory` (`recent_history_query`, `ledger.Rewind`) | the players' videos | the same, and the answer kept for the step | erased, read from the head, built again after a revert |
| `GetLedger` (`changes_query`) | the clips' cast, the vandal squad, the backoffice Home | every take | the forgotten events left out (`NOT forgotten`) |
| `GetSeasonReplay` (the gamemaster) | `/archive` | `GetMapHistory` kept for good, and a day in the browser | asked again each time, kept until a revert, half an hour in the browser |
| `RevertPlayer`, `FindPlayers`, `TopPlayers` (memory `Replay`) | the operator | forgotten takes skipped | erased |

The memory `Replay` erases for every reader of it. So **a revert of a player who painted over a reverted one gives back
what was under the drawing**: before, it gave back the drawing's paint, on the live map. And a take back to the paint
under a drawing counts no take in `TopPlayers` and `FindPlayers`.

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
- **A take older than the ledger's head** (72 hours): memory and the rewinds start there, and the revert itself does
  not reach further. A take forgotten that long ago, painted over since, opens on its paint in a rewind. The season's
  history reads from the reset, and sees through it.
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

## Tests

`ledger/erasure_test.go` (a take from the paint, a take back to what was under, a strike, a shield and a fortify, a
tile given back or reset, a paint painted twice, a bomb and an enclosure, a wipe and another stage), `history_test.go`
and `rewind_test.go` (a drawing in no frame and no hit), `revert_player_usecase` (the paint never back on the live map),
`replay_usecase` (the replay of a drawing painted over), the postgres suites of `history_query`, `recent_history_query`
(the drawing before the day, the step built again after a revert), `changes_query` and `forgetting`, the
`get_map_history_handler`'s `kept`, and `season_replay_query` (kept until a revert).
