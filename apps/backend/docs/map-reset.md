# The map reset

**A season after season 0 starts on a blank map** (the owner, 2026-10-28). The gamemaster tells planet when; planet
wipes the map once per reset, and every client reads the blank map again through the path it already takes after a
restart.

## Who decides, who wipes

- **The gamemaster decides when** (`season.Window.StartsOnBlankMap`: every season but season 0).
  `play_resets_usecase` runs every second (`gamemaster-resets`): once the running season is one that starts on a blank
  map, it sends `planet.v1.ResetMap{reset_id: "season-N", decided_at: its start}` on planet's command stream
  (`season/redis_map_commands`), **once per process**, and again at the next second if Redis refused it.
  `log_play_resets` logs it at Warn (`planet is told to reset the map`). A boot during the season sends it again:
  planet keeps the id and wipes nothing.
- **Planet wipes** (`commands/reset_map_handler`, `resets/usecases/reset_map_usecase`, `log_reset_map`): a command
  with no id or no time is dropped (`resets.ErrInvalidReset`); otherwise `inmemory_tile_storage.Storage.Wipe` runs.

## The wipe, once

`Storage.Wipe(ctx, keep)` holds the flush lock, then the tiles lock, and in one postgres transaction
(`postgres_tile_store.Wipe`) runs `keep`, which keeps the reset in `planet.map_resets` (`postgres_reset_store.Keep`,
`ON CONFLICT DO NOTHING`), then deletes every row of `tiles` and `landmasses`. **A reset kept already wipes nothing**,
and a failure rolls back both. Only then does it empty memory: a new board (every tile blank, no holder, no shield),
no landmass lock, a new recount.

- **No flush meets it**: `Flush` holds the same flush lock from what it takes to what it saves, so a save of the map
  before the wipe can never land after it.
- **Clicks wait** for the transaction (one `DELETE` of the map), as they wait for any write under the tiles lock.
- **A new epoch**: the version becomes `{epoch: now (or the old one + 1), seq: 0}`, and **every planet stream is cut**
  (`feed.cutAll`). A client opens again, sees another epoch, forgets every version and reads the tiles, the holdings
  and the fortresses again (see [The map's version](map-version.md)): the same path as after a restart, so the client
  has no reset code of its own.

**What stays**: accounts, allegiances, charges in hand, titles, stats, bans, chat, missions, and the ledger. Scoring's
books are the gamemaster's: season 1's book opens at the same second (see [The gamemaster](gamemaster.md)).

## Replays

The ledger keeps every take, the ones before the reset too, and `GetReplay` rewinds the map from now. So
`replay_usecase` reads the resets after its window's start (`resets.Store.After`):

- **A reset inside the window starts the replay at it**, on the blank map: the takes before it are left out, and the
  map rewound from now to the reset is blank where nothing was taken since.
- **A window that ended before a reset is refused** (`ErrBeforeReset`, `FailedPrecondition`): the map it played on is
  gone.

## Tests

`inmemory_tile_storage`'s `wipe_test.go` pins the blank map in memory and stored, no flush writing the old map back,
the fortify lock gone, the new epoch and every stream cut, a reset kept already wiping nothing, and a failed keep
leaving the map. `postgres_tile_store`'s pins the delete in the reset's transaction and its rollback;
`postgres_reset_store`'s each reset kept once and read by time; `replay_usecase`'s a replay over a reset, a window
before one, and a reset before the window; `reset_map_handler`'s a command dropped with no id or no time.
`finalnight/season_one_test.go` plays Sat 31 Oct 22:59:59 to Sun 1 Nov 15:01, second by second: the finale's set, then
`frozen` from 23:00, then `default` and one reset at 15:00 with season 1's book and day 1, nothing opened over the break,
and a boot after that tells the reset again.
