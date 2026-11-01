# Season titles: the season's best players, who fought the Final Battle, and where their flag placed

**Season 0 gives ten titles** (the owner, 2026-10-31 and 2026-11-01). Seven go only to the players who fought its Final
Battle:

- `Season 0 Gold`, `Season 0 Silver` and `Season 0 Bronze`: a fighter whose flag placed 1st, 2nd or 3rd on season 0's
  table;
- `Season 0 Battle Top 1`, `Battle Top 2` and `Battle Top 3`: the fighters who took the most, the 2nd most and the 3rd
  most tiles in the battle, whatever their flag;
- `Season 0 Veteran`: every fighter, whatever its flag's place. A gold fighter holds both.

Three go to the season's best players, fighters or not:

- `Season 0 Top 1`, `Top 2` and `Top 3`: the players 1st, 2nd and 3rd on season 0's board of players, the one the
  Hall of Fame shows (see [The season's podium](#the-seasons-podium)).

**A fighter is an account that took at least one tile in the finale's round** (`season-0/finale`), under the flag it
took the most tiles for there. A battle's rule set locks the flags (see [Allegiance](allegiance.md)), so a fighter has
one flag for the whole battle: the flag worn when the battle started is the one that counts, not the season's main flag.
A player who held a flag and never clicked during the battle fought nothing. A bomb, a shield or a quiz is no take.

**The place is the flag's rank on the closed table** (scoring's `Book.Table`): ties share the rank, so two flags level
first are both gold, and no flag is then second. A flag not on the table (no points all season) has no place: its
fighters are Veterans only.

**The tiles rank is the fighter's place among all the fighters by the tiles it took in the round**, ties shared the
same way. The flags are locked, so the tiles under its flag are all the tiles it took. A guest is ranked too: a guest
first leaves `Battle Top 1` to nobody until it signs in.

## How the titles are given

1. At 23:00 the gamemaster hears scoring close season 0's book and tells `gamemaster.v1.SeasonEnded` (see [The
   gamemaster](gamemaster.md#hearing-scoring)): the season, its book and its finale's round. A season with no finale
   names no round, and nothing is recorded.
2. The `player-finale-fights` subscriber (`season_ended_subscriber`, one minute to handle it) runs
   `record_finale_usecase`, which asks scoring by those ids: `InternalService/GetTable(book_id)`, which must be closed,
   and `InternalService/GetRoundPlayers(round_id)`, every account of the round under its main flag there, with its
   tiles (see
   [Scoring](scoring.md#the-reads)). `rpc_scoring_finales` turns the answers into `finales.Place` and `finales.Fighter`.
3. `finales.Result.Fights` gives each fighter its `Fight`: the season, the flag, that flag's rank, 0 when the flag is
   not on the table, and the fighter's tiles rank. A fighter with no flag refuses the whole result.
4. The fights are kept in `player.finale_fights` (`season`, `account_id`, `country`, `flag_rank`, `tiles_rank`; one row
   per season and account, a fight recorded again takes the new answer), by `postgres_fight_store`.
5. Each fighter goes through the same award as a take (`notifying_award_titles` over `award_titles_usecase`): a player
   with the game open sees "New title!" at once. A guest earns nothing yet; its fight is kept, and the take after it
   signs in earns the titles.

**Every step can run again.** Scoring answers the same for a closed book and round, and the award grants only what is
not held, so a second delivery changes nothing, and a failure is delivered again by the stream.

**The titles read the fights through the career.** `titles.Career` holds the account's `finales.Fights` beside its
stats and account; `titles.Careers` reads auth and the fight store, for one account (`Of`, the award) or a page
(`PageOf`, the reconciliation). So `ReconcileTitles` gives the season titles to whoever fought and keeps them.
`Fights.PlacedIn(season, rank)`, `Fights.RankedByTilesIn(season, rank)` and `Fights.FoughtIn(season)` are the whole
rule each title asks.

**An account deleted** loses its fights (`player-finales-accounts`, `forget_fights_usecase`), as it loses its titles.

## A deploy after the battle still gives them

A new consumer group reads its stream from the start, and the stream's trim keeps the last entry each group handled,
so `SeasonEnded` is still there for a group's first subscribe, even days later. Scoring keeps `round_players` for good
(only a deleted account leaves it).

**The Battle Top titles came after season 0 was recorded.** Its fights had no tiles rank (`tiles_rank` 0, migration
`20261101150000_finale_tiles_rank`). The subscriber's group was renamed from `player-finales` to `player-finale-fights`,
so the new group reads season 0's `SeasonEnded` again: the record fills each fight's tiles rank, and the award gives the
Battle Top titles, live. Once that release is out, `XGROUP DESTROY events:gamemaster.v1.SeasonEnded player-finales` removes
the old group.

**The battle's tops were renamed once the season's came.** Season 0's battle tops were given as `season_0_top_N`;
migration `20261101180100_battle_top_titles` renames them, held and worn, to `season_0_battle_top_N`, and
`season_0_top_N` is the season's.

**A take scoring counts after `SeasonEnded` is handled is no fight.** Scoring counts the round's takes from
`planet.v1.TileTaken` on its own, so a take in the battle's last second could, in theory, be counted after player read
the round. The hops between the round's close and `SeasonEnded` take seconds; a deploy after the battle has no such race.

## The season's podium

**The podium is the first three of the book's board** (scoring's `GetBoard` with no flag, which `InternalService` serves
too): every signed-in player by the tiles it took for its main flag, ties sharing a rank, so two players level first are
both `Top 1`. A guest is not on that board, so it gets no place, even if it signs in later.

1. The `player-season-podiums` subscriber (`season_podium_subscriber`) hears the same `SeasonEnded`, finale or not, and
   runs `record_podium_usecase`: it asks scoring for the book's board (`rpc_scoring_podiums`), keeps its ranks 1 to 3
   (`podiums.Board.Places`), and awards each of those players.
2. The places are kept in `player.podium_places` (`season`, `account_id`, `rank` 1 to 3; one row per season and account,
   a place recorded again takes the new rank), by `postgres_podium_store`.
3. `titles.Career` holds the account's `podiums.Places`; `Places.RankedIn(season, rank)` is the whole rule of each
   `Top` title. An account deleted loses its places (`player-podiums-accounts`, `forget_podiums_usecase`).

**Season 0's podium came after it ended.** `player-season-podiums` is a new group, so it reads season 0's `SeasonEnded`
at its first start, records the podium and gives the titles, live. A player deleted since then has left the board, and
the ones below it move up.

## A later season

Each season's titles are types of their own in `titles/catalog.go`, beside `Season0Gold` and the others, listed in
`NewCatalog`. A title id may hold digits since migration `20261031130000_title_digits` (`^[a-z0-9_]+$`, on
`player.titles` and `player.worn_titles`). The client draws them in a frame
of their own (see the frontend's [Titles](../../frontend/docs/titles.md)); a later season's ids need a line in its
`SEASONS` and their icons.
