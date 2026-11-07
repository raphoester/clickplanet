# Progression (`internal/progression/`)

**How far an account went: the tiles it took, the countries it took them for and from, its daily streak, what it did
with the bonuses, the titles it earned on each track, the title it wears, the season finales it fought, and the badges
the page draws beside its name.** It makes no account and names nobody: [Player](player.md) says who an account is,
and tells progression when one becomes a player, leaves, comes back or is forgotten. Always on. Schema `progression`.
It was part of `player` until 2026-11-04.

```
internal/progression/internal/
  accounts/                         AccountID, Account (AccountOf): whether an account is linked and when it was made, as player
                                    tells it
    rpc_player_accounts/            the same of one account or a page, from player.v1.InternalService/GetAccounts (2s timeout)
  stats/                            Stats (StatsSnapshot), Streak, Day, Counted, TakeMark, and the rules on them (TakeAt, MessageAt,
                                    BombAt, FortifyAt, BonusAt, RightAnswerAt, MissionsPaid); the Store port and its contract suite
    postgres_stats_store/           the Store over progression.stats, under an advisory lock per account
    inmemory_stats_store/           the same port in maps, behind the testing tag
    usecases/record_stats_usecase/  — one change to an account's stats, then progression.v1.StatsChanged
    usecases/forget_stats_usecase/
  titles/                           Title (one type per title), Rank, Track (Conquest, Devotion, Chatter, Defense, Bombs, Loot, Quiz, Missions),
                                    Place, Standing, Catalog (NewCatalog), Career (CareerOf), Careers (Of, PageOf),
                                    Holdings, Reconciliation, Book (Shown, Unheld, Grant); the Store port and its contract suite
    postgres_title_store/           the Store over progression.titles
    inmemory_title_store/           the same port in maps, behind the testing tag
    inprocess_title_feed/           the titles each account was just granted, to its open streams, in this process
    usecases/award_titles_usecase/  forget_titles_usecase/  listen_for_titles_usecase/
      award_titles_usecase/notifying_award_titles/  publishes what was granted to the feed
    usecases/reconcile_titles_usecase/  — every account the titles the rules give it, no more; progression.v1.AdminService/ReconcileTitles
      reconcile_titles_usecase/audit_reconcile_titles/  logs every reconciliation at Warn
  wearing/                          the title a player wears: WornOf, Wearable, Showcase (ShowcaseOf), Wardrobe; the Store port and
                                    its contract suite
    postgres_worn_title_store/      the Store over progression.worn_titles
    inmemory_worn_title_store/      the same port in maps, behind the testing tag
    usecases/wear_title_usecase/  forget_worn_title_usecase/
  finales/                          Fight (FightSnapshot), Fights (FoughtIn, PlacedIn), Result, Place, Fighter: who fought a season's
                                    finale, for which flag, and where it placed; the Store port and its contract suite
    postgres_fight_store/           the Store over progression.finale_fights
    inmemory_fight_store/           the same port in maps, behind the testing tag
    rpc_scoring_finales/            the table and the round's players, from scoring.v1.InternalService
    usecases/record_finale_usecase/ — on gamemaster.v1.SeasonEnded: the fights kept, then each fighter awarded
    usecases/forget_fights_usecase/
  nameplates/                       the Store port (Keep, Named, Forget) and its contract suite: an account with a username, as
                                    player tells it
    postgres_nameplate_store/       the Store over progression.nameplates
    inmemory_nameplate_store/       the same port in maps, behind the testing tag
    usecases/keep_nameplate_usecase/  forget_nameplate_usecase/
  departures/                       Departure (Decided, Leaving), Decision (Leaves, Returns): an account its player asked to delete,
                                    as player tells it; the Store port and its contract suite
    postgres_departure_store/       the Store over progression.departures, under an advisory lock per account
    inmemory_departure_store/       the same port in maps, behind the testing tag
    usecases/decide_departure_usecase/  forget_departure_usecase/
  progressionv1controller/          ProgressionService and AdminService (bags), the session interceptor, the stream's session reader
    get_stats_handler/  get_titles_handler/  wear_title_handler/  get_fronts_handler/  get_badges_handler/
    get_progress_handler/  listen_for_events_handler/
    reconcile_titles_handler/  admin_get_progress_handler/   AdminService's own
      admin_get_progress_handler/admin_progress_query/   PostgresQuery: what progression keeps of one account, a guest's too
    get_stats_handler/stats_query/  get_titles_handler/titles_query/  get_badges_handler/badges_query/
    get_progress_handler/progress_query/
                                    PostgresQuery: the response straight from SQL
    inprocess_title_catalog/        the Titles port of three queries: shown, worn and each track, from titles.Catalog
    rpc_planet_fronts/              the countries played for and against, from planet.v1.InternalService/GetTakesByCountry:
                                    GetFronts' port, AdminService/GetProgress' too, and GetProgress' through caching_fronts/
      caching_fronts/               one answer per account for 30s
    progressionread/                what the queries share and that knows only the wire: StreakNow, Career
    caller/                         the account on the context, or Unauthenticated
    progressionmessage/             Title and Titles as progression.v1 messages, for the commands and the adapters
  subscribers/                      the edge for events, as the controller is for the wire
    tile_taken_subscriber/  message_sent_subscriber/  bomb_landed_subscriber/  fortified_subscriber/  bonus_claimed_subscriber/
    quiz_answered_subscriber/  missions_paid_subscriber/  stats_changed_subscriber/  season_ended_subscriber/
    player_created_subscriber/  player_leaving_subscriber/  player_returned_subscriber/  player_forgotten_subscriber/
    log_subscriber/
  migrations/
```

## Why a module of its own

**`player` held identity and progress together, and the two point opposite ways.** Progress hears the whole game:
planet's takes, bombs, fortifies, boxes and quizzes, chat's messages, missions' payments, the gamemaster's season ends,
scoring's finales and planet's ledger for the fronts. Identity is heard by the whole game: the chat, scoring, support,
the clips and the page ask player who a player is, and planet hears player make one. Together they made three cycles,
each through a part that is progression's now:

- **player ⇄ planet**: player asked planet for the fronts and heard its takes; planet hears `player.v1.PlayerCreated`.
- **player ⇄ scoring**: player asked scoring for a finale's table and fighters; scoring asks player who has a username.
- **player ⇄ chat**: player heard `chat.v1.MessageSent`; the chat asks player who posts.

**Split, identity is upstream of everything** (player hears auth alone), and progression is downstream: it speaks the
languages of chat, gamemaster, missions, planet, player and scoring, and nothing but the browser and the operator speaks
its own. `cmd/api/modules_test.go` pins it: a module depends on another when its code imports the other's proto
package (it calls it, hears its events or sends it commands), `TestTheModulesDependOnEachOtherWithNoCycle`,
`TestNothingButTheBrowserAndTheOperatorTalksToProgression`, and `TestProgressionKnowsPlayerAndNotAuth`.

**Progression knows player, never auth: auth is player's detail.** It hears an account's sign-in, its deletion asked,
its return and its end in player's words (`PlayerCreated`, `PlayerLeaving`, `PlayerReturned`, `PlayerForgotten`), and
asks player, not auth, whether an account is linked and when it was made (`player.v1.InternalService/GetAccounts`). The
session token is the platform's: `cpbootstrap` hands every module one verifier (`props.Sessions`), which takes auth's
key, so progression's code never names auth.

## The wire

`progression.v1.ProgressionService` is public; Caddy routes `/progression.v1.ProgressionService/*`.

- **The caller is the account in the token.** `GetStats`, `GetTitles` and `GetFronts` sit behind
  `NewSessionInterceptor` (`cpconnect.NewSessionInterceptor`, always enforcing) as `Identified`: they read only the
  caller's own, so the identity token resumed from the cookie is enough. `WearTitle` is `Attested`: it changes what
  the other players see, so it takes the click token. No token, a bad one, or one with no account is
  `Unauthenticated`. `progression_session_checks{verdict}` counts the verdicts. `GetBadges` and `GetProgress` need no
  token; `ListenForEvents` reads one (below). See [Sessions](sessions.md).
- **`GetStats`** answers `tiles_taken`, `streak_current`, `streak_best` and `streak_last_day` (YYYY-MM-DD), as of
  today (`stats_query`).
- **`GetTitles`** (`titles_query`) is the caller's showcase (the worn title, and the titles shown, which are the ones
  it can wear) and each track's steps and progress, with its stats as of today.
- **`WearTitle(title_id)`** (`wear_title_usecase`) answers the title now worn, which the page draws beside its own
  name at once. A title the caller does not show is `InvalidArgument` (`wearing.ErrNotWearable`).
- **`GetFronts`** is the caller's every country, for and against, uncached (see [The fronts](#the-fronts)).
- **`GetBadges(account_ids)`** and **`GetProgress(account_id)`** are what anybody may read of a player: see [The
  badges](#the-badges) and [The public card](#the-public-card).
- **`ListenForEvents`** is the titles the caller earns, as it earns them: see [A title earned reaches the player at
  once](#a-title-earned-reaches-the-player-at-once).

`progression.v1.AdminService` is the operator's (see [Operator tools](admin.md)), on the console and admin listeners:

- **`ReconcileTitles`** (see [The reconciliation](#the-reconciliation)). It moved from `player.v1.AdminService` with no
  change of behaviour.
- **`GetProgress(account_id)`** (`admin_get_progress_handler`, `admin_progress_query`) is what progression keeps of
  one account, a guest's too: the stats as kept (a guest's streak too, which the game never shows), `messages_sent`,
  the titles shown and the one worn, and **every** country it plays for and against, uncached (`rpc_planet_fronts`,
  not `caching_fronts`): the public card's cut to three is for the public wire. **It is zeros for an account it keeps
  nothing about, never `NotFound`**: whether the account is one is player's to say (`player.v1.AdminService/GetPlayer`).
  An id that is not an account is `InvalidArgument`; a failure to ask planet is the error net's `internal`.
  `no-store`. The back office asks both and composes the player's page (see the back office's
  [Player page](../../backoffice/docs/player-page.md)).

**The clients switched at once** (2026-11-04): player's `GetStats`, `GetTitles`, `WearTitle`, `GetFronts`,
`GetBadges` and `AdminService/ReconcileTitles` were removed, not kept for a transition, and a page reloads when the
backend restarts. `proto/player/v1/title.proto` moved to `proto/progression/v1/title.proto`, in package `progression.v1`.

## Who the others see: nameplates and departures

**Progression keeps its own copy of the two identity facts it needs**, each fed by player's events, so it never
reads player's tables (but once, at its first migration, below).

- **`nameplates`**: an account with a username. `player.v1.PlayerCreated` keeps one (`progression-nameplates`,
  `keep_nameplate_usecase`): an insert that ignores a second, so a delivery told twice changes nothing.
  `GetBadges` and `GetProgress` answer only an account with a nameplate: a guest shows no streak and wears no title
  anywhere, whatever it holds, and an account never named or unknown has nothing to show.
- **`departures`**: an account its player asked to delete. It is player's concept again, in progression's words:
  `player.v1.PlayerLeaving` decides it leaves (`progression-departures-leaving`) and `player.v1.PlayerReturned` that it
  returns (`progression-departures-returns`), both through `decide_departure_usecase`. **The latest decision wins**
  (`Departure.Decided`): the two come on two streams, in either order, and either may come twice. Player tells every
  decision, the ones its own departure no longer keeps too, so progression keeps the latest by `decided_at` whatever
  it hears first. A decision with no time is refused (`ErrNoTime`), so a broken event is buried rather than read as the
  oldest one. `GetBadges` and `GetProgress` leave a leaving account out, exactly as a deleted one: the page shows a
  player nobody can name as deleted. Nothing is deleted, so a return shows it again as it was. `GetStats`, `GetTitles`
  and `GetFronts` (the caller's own) and the operator's `GetProgress` still read it.

## The stats

- **Stats come from `planet.v1.TilesTaken`** (`player-stats`), one event per ledger event, so a spread of seven is
  seven tiles: they share a position and each has its own index. The subscriber takes a batch at once
  (`cpredis.SubscribeBatch`) and records each account's takes of it in one write (`stats.Changes`), so an enclose of a
  continent is one write, not 51,000. A strike on a shield takes no tile and is never
  published. **The streak day is UTC**: a take on the day after `streak_last_day` extends the streak, a take on the
  same day changes nothing, a gap starts it again at 1, and a late event older than the last day counts a tile and
  leaves the streak alone (`Stats.WithTake`). `GetStats` reads it as of today: a streak whose last day is before
  yesterday reads 0, and `streak_best` keeps it. The rule is `Stats.AsOf` for the commands and
  `progressionread.StreakNow`, the same `CASE`, in each query's SQL; `stats_test.go` and the queries' tests pin it
  across UTC midnight.
- **Messages sent come from `chat.v1.MessageSent`** (`player-stats-messages`), one event per message kept
  (`Stats.WithMessage`, `progression.stats.messages_sent`, player's migration `20261003140000_messages_sent`). Every message
  counts: the chat's own rate limit is the only bound. A message is no take, so it neither starts nor extends a
  streak, and a row made by a message alone has no `streak_last_day` (the column is nullable). It is not on the wire
  as a stat: the `Chatter` track's progress is where a player sees it. Counting started with that migration; messages
  sent before it are not replayed.
- **What a player did with the bonuses comes from planet's events**, one counter each in `progression.stats` (player's
  migration `20261025120000_title_tracks`): `bombs_dropped` from `planet.v1.BombLanded` (`player-stats-bombs`,
  `Stats.WithBomb`), `tiles_fortified` from `planet.v1.Fortified` (`player-stats-fortifies`), the landmass's whole size
  (`Stats.WithFortify`), so a 1-tile island taken back and forth is worth one tile each time, `bonuses_claimed` from
  `planet.v1.BonusClaimed` (`player-stats-bonuses`), a caught box (`Stats.WithBonus`), and `right_answers` from
  `planet.v1.QuizAnswered` when `correct` (`player-stats-quizzes`, `Stats.WithRightAnswer`). A bomb or a fortify with
  no account counts for nobody. None is a take, so none touches the streak, and none is on the wire as a stat: each is
  a track's progress. Counting started with that migration. Bombs and fortifies before it were brought in once from
  planet's ledger, on 2026-10-25, by an operator call since removed; boxes and quizzes before it are nowhere.
- **The missions done come from missions' events**: `missions_done` (player's migration `20261028180000_missions_done`)
  from `missions.v1.MissionsPaid` (`player-stats-missions`, `Stats.WithMissionsPaid`). The event says how many
  missions the account was paid in all, and the stats keep the highest number heard: an event told twice or late
  changes nothing, a lost one is repaired by the next, and the first one an account hears brings every mission it was
  paid before, back to the first day of missions. A guest is paid only once it signs in, so a mission a guest finished
  and never signed in for does not count. It does not touch the streak, and it is not on the wire as a stat.
- **The groups kept player's names** when they moved: a group reads its stream from the start, so a new name would
  have counted every fact in the streams again (see [Telling other modules what
  happened](composite-layer.md#telling-other-modules-what-happened)).
- **One way to change the stats.** Each subscriber turns its event into a rule on `Stats` (`stats.TakeAt(position,
  index, at)`, `stats.MessageAt(at)`, `stats.BombAt(position)`, …) and hands it to `record_stats_usecase`, which calls
  `Store.Record(account, change)` and then publishes `progression.v1.StatsChanged`. `Stats` is built from a
  `StatsSnapshot` (`Restore`) and saved from one (`TakeSnapshot`).
- **An event may come twice; it counts once.** The stats row keeps a mark per kind of fact (`stats.Counted`): the
  position and index past the last take (`stats.TakeMark`, `counted_takes_below` and `counted_takes_below_index`) and
  the position after the last bomb and fortify of the ledger it counted, and the time of the last message, box and
  right answer, to the microsecond. A fact at or before its mark was counted already, so a redelivery changes nothing
  (see [Idempotent receivers](composite-layer.md#idempotent-receivers)). A take is counted once the ledger flushed it,
  so the stats never count a take the ledger lost. Stats start the day `player`, which kept them then, was turned on:
  takes before are not replayed.
- **No memory copy: every call reads or writes postgres.** This is not the tile map's pattern on purpose. The map is
  in memory so a click never waits on the database; a take reaches this module over the event bus, so a click already
  never waits on it, and the calls are few (production is ~15 takes a second at peak). A memory copy would load every
  account that ever took a tile at boot, and cost a dirty set, a flush loop and a window a hard kill loses.
- **`Store.Record` is one transaction per fact**, under `pg_advisory_xact_lock` on the account: read the stats, apply
  the rule, upsert. The rule stays in Go, and two facts never read the same stats. An advisory lock rather than
  `FOR UPDATE`, because a first take has no row to lock; `TestFactsEachSubscriberCountsAtOnceAreAllKept`, the
  subscribers writing one account at once, fails without it.
- **If takes ever outgrow one transaction each**, the subscriber can batch them; nothing else changes.

## Titles

- **A title is an object, not a row of thresholds.** **Titles are a concept of their own** (`titles/`), beside
  `stats`, with their own `Store` port, contract suite and adapters. `titles` imports `stats`, `accounts` and `finales`,
  never the other way, so an account's stats know nothing of titles: the queries put the two together, as chat's history query
  puts a message and its author together. Each title is its own type implementing `titles.Title`: `ID()` (what the
  store keeps), `Name()` (what the card shows) and `EarnedBy(career)`, which is free to hold any rule. **Each title
  says how it was earned**, in the words the public card shows under its medal (`Standing.Feat()`): a title that stands
  alone is a `titles.Standalone`, with its own `Feat()` ("Joined before November 2026", "Played on Halloween 2026"); a
  rank's feat is its track's, at the rank's threshold (`Track.Feat`: "Took 1,000 tiles", "Dropped 1 bomb"), so the
  words of a track live in one place. A `titles.Career` is the account's `stats.Stats`, its `accounts.Account`: whether
  it is `Linked` and its `CreatedAt`, as player tells it (a guest with no date when nobody knows it), and the season
  finales it fought (`finales.Fights`, see [Season titles](season-titles.md)). `titles.Careers` reads the last two, for
  one account or a page. **A guest earns no title**: `Catalog.EarnedBy` answers nothing for a career whose account is
  not linked, whatever the titles say, so neither the worker nor the reconciliation writes a row for one. A guest has
  no username, so nobody could see its titles anyway; once it signs in, its next take earns them on its whole career,
  OG included. `catalog_test.go` pins each threshold, and that each id is unique and fits the table's `CHECK`. A rule
  that needs more than a `Career` holds widens `Career`, and whoever builds one.
- **Most titles are ranks on a track.** A `titles.Track` is an id, a name, its ranks in order, and
  `Progress(career)`, the number its ranks are measured on; a `titles.Rank` is a title with a `Threshold()`. `Conquest`
  is `Settler`, `Raider`, `Warlord`, `Conqueror` and `Warmaster` (100, 1,000, 10,000, 100,000 and 1,000,000 tiles
  taken; its progress is the tiles taken). `Devotion` is `Loyal`, `Devoted` and `Unbroken` (a best streak of 7, 30 and
  100 days, so a broken streak keeps its rank; its progress is the streak now). `Chatter` is `Talker`, `Chatterbox`,
  `Socialite` and `Icon` (10, 30, 100 and 1,000 messages sent; its progress is the messages sent). `Defense` is
  `Guard`, `Defender`, `Protector` and `Warden` (100, 1,000, 10,000 and 100,000 tiles fortified). `Bombs` is `Bomber`,
  `Blaster`, `Wrecker` and `Destroyer` (1, 10, 50 and 250 bombs dropped). `Loot` is `Looter`, `Hunter`, `Pirate` and
  `Captain` (10, 100, 500 and 2,500 boxes caught). `Quiz` is `Student`, `Expert`, `Professor` and `Genius` (10, 50, 250
  and 1,000 right answers; a guess is right one time in three, so a wrong answer counts nothing). `Missions` is
  `Rookie`, `Agent`, `Operative` and `Spymaster` (1, 25, 100 and 500 missions done; spy words, so no rank sounds like
  another track's). `OG` stands alone: an account made before 2026-11-01 UTC (a zero date is not). So do `Season 0
  Gold`, `Silver`, `Bronze`, `Battle Top 1`, `Battle Top 2`, `Battle Top 3` and `Veteran`, for the fighters of season
  0's Final Battle, and `Season 0 Top 1`, `Top 2` and `Top 3`, given once to its best three players (see [Season
  titles](season-titles.md)), and `Halloween 2026`: a tile taken on 2026-10-31 UTC, read as a streak that covers that
  day (`Streak.Covers`), so the take itself grants it, live. It is a **keepsake** (`titles.Keepsake`): a title for a
  moment that has passed, which the career stops showing once the streak breaks, so the reconciliation never revokes
  it. `titles.NewCatalog()` is the standalone titles and the tracks, in the order they are shown. **A new rank is a
  type added to its track**, a new track a type listed in the catalog: no migration, no proto change. `Conquest`'s
  names are army words, and a rank past `Warmaster` keeps to them (`Grand Warmaster`, then `Supreme Warmaster`);
  `Chatter`'s are social words; each track's names fit what it counts, and are words a player who learned English from
  games knows. The client draws a medal per id and the initial for an id it does not know, so a new title shows before
  the client has its art.
- **Only the highest rank of each track is shown.** `Catalog.Shown(held)` is the standalone titles held, then the
  highest rank held of each track, each as a `Standing`: the title and its `Place` (track, rank number, how many
  ranks). A lower rank stays held, so a stricter rule or a reconciliation never has to give one back.
- **Titles are kept** in `progression.titles` (`account_id`, `title`, `earned_at`), one row per title held.
  `titles.Book` (the store and the catalog) is what the use cases call: `Unheld` is what the career earns and the
  account does not hold, and reads nothing when the career earns nothing (a guest); `Grant` keeps them; `Shown` is
  what the catalog shows of the titles held. Each track's progress is `Catalog.Progress` (every rank, its threshold and
  whether it is held), which `titles_query` calls through `inprocess_title_catalog`.
- **Wearing a title is a concept of its own** (`wearing/`): holding a title and wearing one change apart, so the
  choice has its own port, its own table, its own adapters and its own forget. `wearing` imports `titles`, never the
  other way. A player's choice is kept in `progression.worn_titles` (`account_id`, `title`, `worn_at`), one row per
  account, by `postgres_worn_title_store`. `wearing.WornOf(shown, chosen)` is what is shown as worn: the choice when
  it is shown, the rank shown on its track when the choice is a rank of one, otherwise the first ranked title shown,
  then the first shown, and none for an account that shows no title. So a player who wore `Settler` wears `Raider` the
  moment it earns it, and nothing is rewritten when a rank is added or a title revoked. `wearing.Wardrobe` (the store,
  the titles shown through its `Titles` port, which `titles.Book` fills, and the catalog) is what the use cases call:
  `Showcase` is the worn title and the titles shown; `Wear` refuses a title that is not shown and keeps the choice.
  **The title worn is a badge**: `GetBadges` answers it by account, and the page draws it beside every name; no name
  carries it. `wearing.ShowcaseOf(catalog, held, choice)` is the same rule over what a query read: the titles shown
  and the one worn.
- **A listener grants them, not the stats' write.** `record_stats_usecase` publishes `progression.v1.StatsChanged`
  (stream `events:progression.v1.StatsChanged`) once a change is counted. The `player-titles` subscriber
  (`stats_changed_subscriber`, `award_titles_usecase`) reads the stats, then the career (`Careers.Of`: player's
  `GetAccounts` for the one account, linked and creation date; and the finales fought, one query), and grants what
  `Unheld` answers at the clock's time. It only grants: a title a rule no longer gives stays until the next
  reconciliation. A failure to ask player or to read the fights grants nothing for that fact. The same award runs for
  each fighter of a season's finale (see [Season titles](season-titles.md)). So a linked account made before November
  becomes `OG` at its first take after linking, whenever that take is; an account that never takes a tile and never
  posts gets no title. `Store.Grant` keeps the first `earned_at` of a title granted twice. An account with no stats
  (forgotten since) earns nothing. It reads the stats again rather than trusting the event, so a dropped
  `StatsChanged` only delays a title to the account's next fact.
- **A title earned reaches the player at once.** See [below](#a-title-earned-reaches-the-player-at-once).
- **`GetProgress` answers the titles shown and the one worn**, each as `{id, name, rank, feat}`
  (`progressionmessage.Titles`, `progressionmessage.Title`), `rank` set for a rank of a track: its id and name, its
  number and how many ranks the track has.
- **A title id is text, not an enum type**: a title is an object a deploy adds or drops with no migration (see
  [Durability](durability.md)).

### The reconciliation

**An operator reconciles the titles**: `progression.v1.AdminService/ReconcileTitles`, on the admin listener (see
[Operator tools](admin.md)), after a deploy that adds a title or changes a rule. Nothing runs it at boot.
`reconcile_titles_usecase` pages through `progression.stats` (the stats store's `StatsAfter`, 500 accounts at a
time, in account order); for each page it reads the careers (`Careers.PageOf`: player's `GetAccounts`, one call, and
the fights, one query) and their titles (`Store.Holdings`, one query), and `Catalog.ReconciliationOf` says what to
grant (earned, not held) and what to revoke (held, not earned: a guest's titles, a title a stricter rule no longer
gives, an id the catalog no longer has; never a keepsake). Then one `Grant` and one `Revoke` per page. It records
nothing of its own: a second run changes nothing, and a failed or interrupted one is simply run again. The answer
counts the titles `granted` and `revoked`. It runs beside the listener: a title the listener grants mid-run is one the
rules give, so the run never revokes it. `audit_reconcile_titles` logs every call at Warn. It never touches what a
player wears: a choice it makes unshowable is simply not worn, and `WornOf` falls back. A reconciliation grants
through the store, not the award, so it is never announced.

### A title earned reaches the player at once

**`ListenForEvents` is the caller's titles, as it earns them.** `notifying_award_titles` wraps the award and hands
what it granted to `inprocess_title_feed`, keyed by account. The stream sits behind `NewStreamSessionReader`
(`cpconnect.NewSessionReaderInterceptor`, `Identified`): the token is read once, from the headers that open the stream,
and the account stays on the context while it is open. **A stream with no account is refused `Unauthenticated`**
(`caller.ErrNoAccount`): all it says is the caller's own. The handler runs `listen_for_titles_usecase`:

- **A `heartbeat` at once** when the stream follows the feed, which the page reads as the stream being live, then
  every `httpServer.streamHeartbeat` (see [The live streams](live-streams.md)).
- **A `title_earned` for each title of a grant that `Catalog.Shown` keeps**: a take that earns `Raider` and `Warlord`
  together says `Warlord`.
- **At most once.** The feed drops for a stream 8 grants behind (`subscriberBuffer`), and an account with no stream
  open hears nothing: its titles are still kept.

The web client opens it only for a signed-in account, and opens it again when its token changes (see the frontend's
[Titles](../../frontend/docs/titles.md)). `TestAStreamWithAnAccountSaysItIsLiveThenSendsTheTitlesItEarns` and
`TestAStreamWithNoAccountIsRefused` pin the handler. Until 2026-11-04 the titles came on player's roster stream, to a
stream opened with a token.

## The badges

**`GetBadges(account_ids)` is what the page draws beside a page of names: each account's badges**, its streak as of
today and the title it wears (`progression.v1.Badge`). The browser keeps them by account id and draws them beside every
name it shows, wherever the name came from (the chat, the roster, a board, a card): see the frontend's
[Badges](../../frontend/docs/badges.md). It needs no token (the session interceptor does not list it: names and badges
are public), is `NO_SIDE_EFFECTS`, and answers `no-store`: the page is the cache, and an answer kept on the way could
undo what the page learns of its own badges. `get_badges_handler`, `badges_query`.

- **At most 100 ids a call** (`get_badges_handler.MaxAsked`), a repeated one counted each time; more is
  `InvalidArgument` (`ErrTooMany`) and reads nothing. An id that is not an account is `InvalidArgument`; none is an
  empty answer.
- **Only the accounts with a badge to show, once each, in the order asked.** Left out: an account with no nameplate (a
  guest, whatever it holds, one never named, an unknown or deleted one), one asked deleted (a departure leaving), and
  one with neither a streak today nor a title worn. So a caller reads a missing account as "nothing to draw" and never
  tells the reasons apart, and nothing hidden from the other players leaks.
- **The same numbers as the commands**: `progressionread.StreakNow` in SQL is `Stats.AsOf`, and the held titles and
  the choice go through the `Titles` port (`inprocess_title_catalog.Worn`, `wearing.ShowcaseOf`), the rule
  `Wardrobe.Showcase` applies. A parity test (`TestEachBadgeIsWhatTheRulesShowBesideTheName`) runs the same rows
  through both.
- **It moved from player as it was.** It spoke only of streaks and titles, by account id, so only the guest rule
  changed: a guest is an account with no nameplate, where it was one with no profile.

## The public card

**`GetProgress(account_id)` is how far a player went, on its public card**, beside player's `GetPlayer`, which says who
it is (see [Player](player.md)): its stats as of today, the titles shown and the one worn, and the top 3 countries of
each front through `caching_fronts`. The page asks both, side by side (see the frontend's [Who is
playing](../../frontend/docs/who-is-playing.md)). `get_progress_handler`, `progress_query`.

- **It needs no token** (the session interceptor does not list it), is `NO_SIDE_EFFECTS`, and answers
  `public, max-age=10`, as `GetPlayer`.
- **`NotFound`** (`progress_query.ErrNotShown`) for an account the other players do not see: one with no nameplate (a
  guest, one never named, an unknown or deleted one) or one leaving. An id that is not an account, an empty one
  included, is `InvalidArgument`. **A failure to ask planet is a real error**, the error net's `internal`.
- **It never answers the account id**: the caller holds it.

## The fronts

**The countries a player plays for and against are read from the ledger, not kept here.** The postgres ledger
(`planet.ledger_events`) keeps every take since 2026-09-16 with its account, its flag and the flag that held the tile:
after 72h only the address is blanked, and a deleted account is blanked off its takes. So progression keeps no copy:
it asks `planet.v1.InternalService/GetTakesByCountry` over the internal listener (`rpc_planet_fronts`, 5s timeout),
and planet answers from one SQL statement (`takes_query`). Both lists (`plays_for`, `plays_against`, each
`{country_id, tiles}`) come most tiles first, then by country, and a tile nobody held is against nobody.

- **`GetProgress`, which anybody may call, answers the top 3 of each**, through `caching_fronts`: one answer per
  account for 30s (`frontsKeptFor`), at most 4,096 accounts, an expired one making room. The rest is not on the public
  wire at all, so the cut is not one a page can undo.
- **`GetFronts` is the caller's every country**, uncached, behind the session interceptor as `Identified`: a read, so
  the identity token resumed from the cookie is enough. The client already asks at most every 10s while it clicks.
- **A failure to ask planet is a real error**, the error net's `internal`.
- **It costs one grouped read of the account's events**, through `ledger_events_account_kind`
  (`account, kind, country, previous`). The takes are read from the index alone, and only a spread's or an enclose's
  row is read from the table, for its payload; `takes_query/plan_test.go` pins it. Key columns and not `INCLUDE`, so
  the index deduplicates: 9.5 MB for 1.5M rows, against 63 MB with `INCLUDE`. A player with a million takes still
  makes that heavy, which is what the cache is for on the public card. If it ever is too slow, a projection fed by
  `TilesTaken` is the way back.
- **They are the total since the ledger has accounts, never a season's.** Decided on 2026-10-08, in season 0, when the
  two are the same. The card says which flags a player's tiles went to, so the flag beside its name on the Players
  board makes sense. **The per-season split is the same query bounded by the season's start and end**: `scoring`
  holds the calendar, so it would pass the interval. Until then, from season 1 the top flag on the card can differ
  from the flag on the board. Neither list adds up to `tiles_taken`, which counts from 2026-09-17 and only what the
  event bus delivered.

## An account forgotten

**`player.v1.PlayerForgotten` deletes everything progression keeps of the account**, one group per concept, each with
its own forget use case: `progression-stats-forgotten` (`forget_stats_usecase`), `progression-titles-forgotten`
(`forget_titles_usecase`), `progression-wearing-forgotten` (`forget_worn_title_usecase`),
`progression-finales-forgotten` (`forget_fights_usecase`), `progression-nameplates-forgotten`
(`forget_nameplate_usecase`) and `progression-departures-forgotten` (`forget_departure_usecase`). Each deletes what is
left, which is nothing the second time. Player tells it for every account auth deleted, even one player kept nothing
about, since progression may keep its stats (a guest's takes). A fact that arrives after, from a token minted before
the delete, makes a new stats row; the token lives an hour at most. A grant that lands after the forget, from a take
or a reconciliation page read before it, leaves rows for an account that is gone, as a late take does for the stats.
Until 2026-11-04 they heard `auth.v1.AccountDeleted` (`player-titles-accounts`, `player-wearing-accounts`,
`player-finales-accounts`, and `player-accounts` for the stats).

## The groups

Every group progression subscribes with (see [Telling other modules what
happened](composite-layer.md#telling-other-modules-what-happened)):

| group | hears | does |
|---|---|---|
| `player-stats` | `planet.v1.TilesTaken` | counts the takes, a batch at once |
| `player-stats-messages` | `chat.v1.MessageSent` | counts the message |
| `player-stats-bombs`, `player-stats-fortifies`, `player-stats-bonuses`, `player-stats-quizzes` | `planet.v1.BombLanded`, `Fortified`, `BonusClaimed`, `QuizAnswered` | counts the bomb, the tiles fortified, the box, the right answer |
| `player-stats-missions` | `missions.v1.MissionsPaid` | keeps the missions paid in all |
| `player-titles` | `progression.v1.StatsChanged` | grants the titles the stats now earn |
| `player-finale-fights` | `gamemaster.v1.SeasonEnded` | records the finale's fights, then awards each fighter ([Season titles](season-titles.md)) |
| `progression-nameplates` | `player.v1.PlayerCreated` | keeps a nameplate |
| `progression-departures-leaving`, `progression-departures-returns` | `player.v1.PlayerLeaving`, `PlayerReturned` | keeps the latest decision |
| `progression-*-forgotten` (six) | `player.v1.PlayerForgotten` | forgets what it keeps |

**Each event's write has a 5s timeout** (`subscribers.Timeout`; a season's end, a minute), so a stuck database cannot
hold a subscriber. **Errors are real errors**: absence is `ErrNoStats`, which the use cases answer as nothing to
count or grant; on a request a database error is the error net's `internal`; on an event it is counted in
`events_failed_total`, logged by `log_subscriber`, and the event tried again, then buried in `events:dead`.
**`StatsChanged` is published, not put in an outbox**: a lost one only delays a title to the account's next fact, and
a reconciliation grants anything missed.

## The tables and their move

**`stats`, `titles`, `worn_titles` and `finale_fights` moved from schema `player` to `progression`** in
progression's first migration, `20261104190000_from_player`: `ALTER TABLE player.<table> SET SCHEMA progression` for
each that exists, so rows, indexes and constraints move and nothing is copied. Then `CREATE TABLE IF NOT EXISTS` makes
them in a database with no player schema (progression's own tests).

- **Why a move, not a copy then a drop**: one statement per table, in the migration's transaction, instant, keeping
  indexes and constraint names, and no second release to drop the old tables. No downtime beyond the usual restart.
- **On a fresh full database** player's old migrations still make the four tables, and progression, built after
  player in `cmd/api`, moves them.
- **The same migration makes `nameplates(account_id)` and `departures(account_id, leaving, decided_at)`** and fills
  them once from `player.profiles` and `player.departures`. **This is the only time progression reads player's
  tables**; player's events keep both from then. A group reads its stream from the start, so a `PlayerCreated` still
  in the stream is heard again, which the insert ignores.
- **Its down migration drops the nameplates and the departures**, and leaves the four tables in `progression`.
- `migrations_test.go` pins the move with its rows and index (`TestTheTablesMoveFromPlayerWithTheirRows`), the
  backfill (`TestTheNameplatesAndTheDeparturesStartFromPlayers`) and a fresh database (`TestAFreshDatabaseMakesTheTables`).
- **No foreign key to another schema**: the schemas are each module's own, and an event is how a fact crosses.
- **The pool closes after the subscribers stop** (`cppg.CloseAfter`), so what they read before the shutdown is still
  written.
- **`stats.StoreContractSuite`** (and each concept's suite) is the port's behaviour, run on the in-memory store and on
  postgres. The use cases are tested over the in-memory one.

## The deploy that split it

**By hand, once the release is out** (see [Telling other modules what
happened](composite-layer.md#telling-other-modules-what-happened)):

```bash
XGROUP DESTROY events:auth.v1.AccountDeleted player-titles-accounts
XGROUP DESTROY events:auth.v1.AccountDeleted player-wearing-accounts
XGROUP DESTROY events:auth.v1.AccountDeleted player-finales-accounts
DEL events:player.v1.StatsChanged
```

Then run `progression.v1.AdminService/ReconcileTitles` once. **The old `StatsChanged` stream is not drained**: a
`StatsChanged` was never in an outbox (a lost one only delays a title to the account's next fact), the old server's
shutdown reads what its group has left, and the reconciliation grants anything missed. The groups that kept their names
read on from where player stopped, on the same streams.
