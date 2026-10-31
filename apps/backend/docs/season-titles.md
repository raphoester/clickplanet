# Season titles: who fought the Final Battle, and where their flag placed

**Season 0 gives four titles, and only to the players who fought its Final Battle** (the owner, 2026-10-31):

- `Season 0 Gold`, `Season 0 Silver` and `Season 0 Bronze`: a fighter whose flag placed 1st, 2nd or 3rd on season 0's
  table;
- `Season 0 Veteran`: every fighter, whatever its flag's place. A gold fighter holds both.

**A fighter is an account that took at least one tile in the finale's round** (`season-0/finale`), under the flag it
took the most tiles for there. A battle's rule set locks the flags (see [Allegiance](allegiance.md)), so a fighter has
one flag for the whole battle: the flag worn when the battle started is the one that counts, not the season's main flag.
A player who held a flag and never clicked during the battle fought nothing. A bomb, a shield or a quiz is no take.

**The place is the flag's rank on the closed table** (scoring's `Book.Table`): ties share the rank, so two flags level
first are both gold, and no flag is then second. A flag not on the table (no points all season) has no place: its
fighters are Veterans only.

## How the titles are given

1. At 23:00 the gamemaster hears scoring close season 0's book and tells `gamemaster.v1.SeasonEnded` (see [The
   gamemaster](gamemaster.md#hearing-scoring)): the season, its book and its finale's round. A season with no finale
   names no round, and nothing is recorded.
2. The `player-finales` subscriber (`season_ended_subscriber`, one minute to handle it) runs
   `record_finale_usecase`, which asks scoring by those ids: `InternalService/GetTable(book_id)`, which must be closed,
   and `InternalService/GetRoundPlayers(round_id)`, every account of the round under its main flag there (see
   [Scoring](scoring.md#the-reads)). `rpc_scoring_finales` turns the answers into `finales.Place` and `finales.Fighter`.
3. `finales.Result.Fights` gives each fighter its `Fight`: the season, the flag and that flag's rank, 0 when the flag is
   not on the table. A fighter with no flag refuses the whole result.
4. The fights are kept in `player.finale_fights` (`season`, `account_id`, `country`, `flag_rank`; one row per season and
   account, the first one kept), by `postgres_fight_store`.
5. Each fighter goes through the same award as a take (`notifying_award_titles` over `award_titles_usecase`): a player
   with the game open sees "New title!" at once. A guest earns nothing yet; its fight is kept, and the take after it
   signs in earns the titles.

**Every step can run again.** The fights are inserted once and the award grants only what is not held, so a second
delivery changes nothing, and a failure is delivered again by the stream.

**The titles read the fights through the career.** `titles.Career` holds the account's `finales.Fights` beside its
stats and account; `titles.Careers` reads auth and the fight store, for one account (`Of`, the award) or a page
(`PageOf`, the reconciliation). So `ReconcileTitles` gives the season titles to whoever fought and keeps them.
`Fights.PlacedIn(season, rank)` and `Fights.FoughtIn(season)` are the whole rule each title asks.

**An account deleted** loses its fights (`player-finales-accounts`, `forget_fights_usecase`), as it loses its titles.

## A deploy after the battle still gives them

A new consumer group reads its stream from the start, and the stream's trim keeps the last entry each group handled,
so `SeasonEnded` is still there for `player-finales` when it first subscribes, even days later. Scoring keeps
`round_players` for good (only a deleted account leaves it).

**A take scoring counts after `SeasonEnded` is handled is no fight.** Scoring counts the round's takes from
`planet.v1.TileTaken` on its own, so a take in the battle's last second could, in theory, be counted after player read
the round. The hops between the round's close and `SeasonEnded` take seconds; a deploy after the battle has no such race.

## A later season

Each season's titles are types of their own in `titles/catalog.go`, beside `Season0Gold` and the others, listed in
`NewCatalog`. A title id may hold digits since migration `20261031130000_title_digits` (`^[a-z0-9_]+$`, on
`player.titles` and `player.worn_titles`). The client draws `season_0_gold`, `season_0_silver` and `season_0_bronze` as
their place number in their metal, and `season_0_veteran` as a star in platinum.
