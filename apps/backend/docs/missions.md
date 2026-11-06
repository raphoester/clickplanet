# Missions (`internal/missions/`)

**Every player has four or five missions a day of its own, and each pays itself once done. Every flag has one more,
which all its players do together, and which gives the flag season points** (see [The flag's mission](#the-flags-mission)).
It reads planet's facts (`TileTaken`, `BonusClaimed`, `QuizAnswered`, `BombLanded`, `ShieldPlaced`, `Fortified`,
`ShieldStruck`, `GetTerritories`, `GetCountrySizes`, `GetAllegianceOf`), the gamemaster's days (`DayStarted`), player's
word on an account (`PlayerSignedIn`, `PlayerForgotten`, `GetAccounts`), and commands planet to grant a reward
(`planet.v1.GrantCharges`) and scoring to grant points (`scoring.v1.GrantPoints`), over Redis, as scoring asks planet
what each country holds: neither ever imports, calls or hears missions. It tells what it did (`MissionsPaid`,
`FlagMissionDone`) to whoever listens: progression and the chat.
Its public route, `/missions.v1.MissionService/*`, is listed in `deploy/vps/caddy/Caddyfile`, as every service's is.

```
internal/missions/internal/
  days/                             DayID, Day (a day the gamemaster told: its round's id, its start and its end), ErrNoDay;
                                    the Store port and its contract suite, TestingDayAt (season 0's days, for tests)
    postgres_day_store/             the Store over missions.mission_days
    inmemory_day_store/             the same port in a map, behind the testing tag
    usecases/keep_day_usecase/
  missions/                         AccountID, Fact (one type per kind) and Counted, MissionID,
                                    Order (one type per kind), Mission, Brief, Draw, World and Ground, Sheet (an account's day), Briefing;
                                    the Store port and its contract suite
    postgres_mission_store/         the Store over missions.mission_briefs, mission_sheets, mission_progress and mission_payments;
                                    records.go is the jsonb format of a mission, one payload per kind
    inmemory_mission_store/         the same port in maps, behind the testing tag
    rpc_planet_world/               what each flag holds and each ground's size, from planet.v1.InternalService
    rpc_planet_allegiance/          the flag an account pledged, from planet.v1.InternalService/GetAllegianceOf
    rpc_player_accounts/            whether an account is signed in, from player.v1.InternalService/GetAccounts (2s timeout)
    payroll/                        pays each mission done once: the payments and their grants, in one transaction
    planetgrant/                    a payment as the planet.v1.GrantCharges command
    redis_outbox_sender/            what the outbox relay sends with: the grants as commands, MissionsPaid and
                                    FlagMissionDone as events
    inprocess_day_feed/             wakes the account's open streams when its day changed, in this process
    usecases/record_fact_usecase/  get_missions_usecase/  pay_signed_in_usecase/  forget_account_usecase/
              listen_for_events_usecase/
  flagmissions/                     Take and Strike, Order (TakeIn, BreakShields), MissionID, Day (a flag's day), Done, Draw,
                                    Briefing; the Store port and its contract suite
    postgres_flag_day_store/        the Store over missions.flag_days and flag_missions
    inmemory_flag_day_store/        the same port in a map, behind the testing tag
    scoringgrant/                   a flag mission done as the scoring.v1.GrantPoints command
    flagmessage/                    a flag mission as missions.v1.FlagMission (GetMissions answers it too), and a
                                    flag mission done as the missions.v1.FlagMissionDone event
    usecases/record_flag_take_usecase/  record_strike_usecase/
  rules/                            Rules, the numbers missions draws with, one document the operator edits; the Store
                                    port; ErrInvalidRules
    postgres_rules_store/           the Store over missions.rules, and the flagmissions.Sizings each draw reads
    usecases/save_rules_usecase/    audit_save_rules/
  missionsv1controller/             MissionService and AdminService (bags), the session interceptors, the caller,
                                    missionmessage
    get_missions_handler/  listen_for_events_handler/
    list_flag_days_handler/flag_days_query/  get_mission_totals_handler/totals_query/
    get_rules_handler/rules_query/  save_rules_handler/
  subscribers/                      tile_taken_subscriber/  bonus_claimed_subscriber/  quiz_answered_subscriber/
                                    bomb_landed_subscriber/  shield_placed_subscriber/  fortified_subscriber/
                                    day_started_subscriber/  flag_tile_taken_subscriber/  shield_struck_subscriber/
                                    player_signed_in_subscriber/  player_forgotten_subscriber/  missions_paid_subscriber/
                                    log_subscriber/
  migrations/
```

- **A day is the gamemaster's**, never a clock of missions' own (the owner, 2026-10-29: two cutoffs that happen to
  match are brittle). The gamemaster tells each day as it starts, `gamemaster.v1.DayStarted` (`missions-days`,
  `day_started_subscriber`, `keep_day_usecase`), and missions keeps it in `mission_days` (`day_id`, the round's id such as
  `season-0/day-22`, `season` and `number`, its place among the season's days, `starts_at`, `ends_at`): a day told
  again, when a moved season end cut it, keeps its new end. **A
  fact counts in the day told that runs at its time** (`days.Store.DayAt`), and every row names its day by that id.
  **No day, no missions**: between two seasons, from a season's last cutoff to its end (the Final Battle), and while
  any battle runs (a battle takes its time from a day, see [A battle and the days](gamemaster.md#a-battle-and-the-days)),
  a fact counts nothing, `GetMissions` answers no mission and no end, and a sign-in pays nothing (the owner, 2026-10-29).
  **A fact can come before its day**: the day and the facts come on separate streams, so a take of the first second or
  so of a day may find no day yet and count nothing (the owner chose it over asking the gamemaster each fact,
  2026-10-29). Migration `20261029110000_gamemaster_days` named every row kept before by its date as season 0's day.
  **The rules are objects**: a `Mission` is an id, an `Order` and a reward, and nothing else.
  **An order is one kind of mission**, a type of its own with its own fields: `TakeFor{Flag, Goal}` (tiles for the
  flag), `TakeFrom{Flag, Goal}` (tiles another flag held), `TakeIn{Ground, Goal}` (tiles on a country's land),
  `TakeFromIn{Flag, Ground, Goal}` (tiles another flag held on a country's land), `CatchBoxes{Goal}`,
  `AnswerQuizzes{Goal}` (right answers), `BombIn{Ground, Goal}` (bombs that went off on a country's land; a bomb in the
  sea counts nothing), `ShieldTiles{Shields, Goal}` (tiles brought to `Shields` by the player's own placement: the
  placement that makes it `Shields` counts, so a tile struck back and filled again counts again; a fortify's shields do
  not count) and `FortifyTiles{Goal}` (each landmass the player made whole counts its tiles). It decides how a fact
  moves its progress (`advanced`, which reads only the facts of its kind) and when it is done (`done`). The progress of
  a mission is one counter from 0 that only its order reads. A new kind is a new type and its cases in the store and
  the encoder.
  - **`Draw(day, account, flag, world)` is the brief**, seeded by the day and the account, so each player's day has its
    own count and goals; the flag only says what is a rival and whose ground is whose.
    **Every goal is drawn from its span**, in steps of 10 tiles (`span`, `draw.goal`): 1. take 120 to 240 tiles (two
    to four banks) for the flag; 2. take 80 to 160 tiles from a rival, a flag holding 1,000 tiles (`rivalAtLeast`),
    drawn by the ground it holds (`draw.ranked`): a flag holding a third of the map is the target of a third of the
    players' days, so the leaders share the heat and a small rival gets its turn (with no rival, twice that for the flag);
    3. take 300 to 600 tiles on a ground of 3,000 tiles or more (`groundAtLeast`) that is not the flag's and of which
    1,000 tiles or more are not the flag's (`openAtLeast`), seventeen countries on today's map (with none, the same
    for the flag). **Then one or two feats** (`draw.feats`), since a day has 4 or 5 missions (`fewestMissions`,
    `mostMissions`), each kind once at most, drawn evenly: catch 3 to 6 boxes, answer 2 to 4 quizzes right, bring 1 to
    3 tiles to 10 shields (`wallShields`, planet's `bonuses.shield.perTile`), fortify 20 to 60 tiles, drop 1 or 2
    bombs on a big ground (drawn as the third mission's is, so it may be the same), and take 40 to 80 tiles from a
    rival on a ground where it holds 300 tiles or more (`raidAtLeast`, the ground drawn evenly among every rival's).
    A world with no ground or no such rival leaves the bomb or the raid out. The goals are sized for a player who
    clicks and plays the boxes: a box and a quiz come every 4 to 8 minutes, a bomb is 1 in 14 of them, a shield box
    holds 5 to 20. Until 2026-10-28 the extras were more rivals and grounds, and a day had 3 to 5 missions, all of
    tiles. The goals were a fixed 60, 40 and 200 until 2026-10-26, when a
    top player finished all three in ten minutes. Until 2026-10-27 every flag aimed at the one leader, so everyone
    attacked the same flag, and a flag could be sent to a ground it already held whole. The world is planet's
    (`GetTerritories`: what each flag holds, and how many tiles of it on each country's land; `GetCountrySizes`,
    `clicks.Borders.CountrySizes`).
  - **The rewards are drawn with the day too** (`draw.rewards`, from the `purses`): the four charges in a random
    order, each once before any twice, so a day of five pays one charge twice, each as much as its purse gives: 15 to
    30 shields, 4 to 8 spread clicks, one bomb, or 2 to 3 enclosures. A purse starts above what a box gives and tops
    at the stack planet keeps (`bonus.*.held`), since a grant past it is lost. **Never a refill**: a mission pays
    what multiplies or guards clicks, not clicks.
  - **`Briefing.Of(day, account)` draws once and keeps it**, at the account's first read or take of the day, for the
    flag it pledged then (`rpc_planet_allegiance`, planet's `GetAllegianceOf`); an account that never pledged one has
    no missions. The store's first `KeepBrief` wins, so two takes racing to draw an account's day read one brief, and a
    map or a flag that changes later never changes it. Until 2026-10-28 a day was drawn per flag, the same for each of
    its players, and locked to the flag of the account's first take: it added nothing.
  - **A mission is named by its `MissionID`**, a UUID the draw derives from the day, the account and its place in the
    brief (`drawnID`), so the same draw names it the same. Progress and payments point at it,
    and nothing caps how many missions a brief holds.
  - **A brief is one `jsonb` array per day and account** (`mission_briefs.missions`), the missions in the order drawn.
    Each is `{id, kind, reward, payload}`; the payload is the kind's own fields (`{"flag", "goal"}` for `take_for`,
    `{"goal"}` for `catch_boxes`).
    A kind with other fields, or none, needs code and no migration. A kind the store does not know is an error.
  - **`GetMissions` answers each mission in its kind's own message** (`missionmessage`, `Sheet.Statuses`): what every
    mission has is its id, its reward, `done` and `paid`; what it asks is a `oneof`, a message per kind with its own
    fields, its goal and how far it went. A client skips a kind it does not know, so a new kind
    reaches an old client as nothing. The reward is a `oneof` too (`charges` today). `country_id` and `enlisted` are
    deprecated and left empty.
  - **A fact counts on every mission it fits, whatever flag it is for** (`Sheet.WithFact`): after a change of flag the
    day's rivals and grounds still count, and its tiles for the old flag stop. A change of flag gets no second set,
    since an account has one day. The client never names the flag. **Progress is counted from planet's events**, one
    subscriber each (`missions-progress` for `TileTaken`, `missions-boxes`, `missions-answers`, `missions-bombs`,
    `missions-shields`, `missions-fortifies`), all through `record_fact_usecase`, up to each goal, in one transaction
    under `pg_advisory_xact_lock` per fact, with nothing written once nothing changed. A wrong answer, and a bomb, a
    shield or a fortify with no account, is dropped by its subscriber. **Not from the ledger**: `ledger_events` has no
    time index, and an account's takes since the day started would read every take it ever made. **A fact told twice counts
    once**: Redis delivers at least once, and the sheet keeps a mark per kind (`missions.Counted`, the
    `mission_sheets.counted_*` columns): the ledger position after the last bomb, shield and fortify it counted, the
    time of the last box and right answer, so one at or below its mark counts nothing. A take's mark is the position
    and index past the last take counted (`missions.LedgerMark`: `counted_takes_below`, `counted_takes_below_index`):
    the takes of a spread or an enclose share one position, and each has its own index. Each kind has its own
    mark because each comes on its own stream, in no order with the others. Two boxes or answers of one account in the
    same microsecond count once.
  - **A mission done is paid at once, with no claim** (`payroll.Pay`, `Sheet.WithPayments`). The take that finishes it
    writes the payment (`mission_payments`) and records the `planet.v1.GrantCharges` command in missions' outbox, in
    the transaction that counts the take: both are kept, or neither and the take is told again. `missions-outbox`
    sends it to planet, which grants it once per `grant_id` (`planetgrant.Of`: the account and the mission, so a
    payment told twice names one grant). **What does not fit the stack is lost**, as planet grants any charge: no
    reply comes back.
  - **The payment is told too**: with the grants, the payroll records `missions.v1.MissionsPaid`, how many missions the
    account was paid in all (`Store.PaidInAll`, a count of `mission_payments` under the sheet's lock), these included.
    `progression` keeps it for the Missions title track. A running total, not one event per mission: told twice or late it
    changes nothing, and the first one brings every payment before it. So one outbox holds a command and an event, and
    `redis_outbox_sender` sends each on its own kind of stream.
  - **The player hears its progress and its payments on a stream, with no read** (`MissionService.ListenForEvents`,
    `stream MissionEvent`). The stream sends the caller's day as `GetMissions` answers it (`day`, `missionmessage.Day`)
    as it opens, then again after each fact of the caller is counted, and a `heartbeat` every
    `httpServer.streamHeartbeat`. **A fact wakes the stream once its transaction is committed**: `record_fact_usecase`
    for the player's missions, `record_flag_take_usecase` and `record_strike_usecase` for the flag's, with the account
    the take or the strike names (`subscribers.AccountOrNone`: a flag's fact with no account wakes nobody). **A stream
    sends at most one day a second** (`listen_for_events_usecase.DefaultGap`): every take moves a mission, and a run of
    clicks would read the day once per take; what changes during the wait is in the day read after it. That is what
    the game's progress line reads (see the frontend's [Missions](../../frontend/docs/missions.md)). A payment wakes it
    too: missions hears its own `MissionsPaid` (`missions-streams`, `missions_paid_subscriber`), so the day is read once
    the payment is committed, and the same relay pass sends the grant to planet: the card and the charges come within
    about a second of each other, about 2 to 4 s after the act that finished the mission, whatever its kind.
    `inprocess_day_feed` wakes each open stream of the account; changes told while a stream reads its day are one more
    read, not one each. Delivery is at most once, and a stream that opens again starts from the whole day. Until
    2026-11-05 only a payment woke the stream, and progress came with the read 2 s after a run of clicks.
    Until 2026-11-03 the client read `GetMissions` after each run of clicks and 3 s later, and nothing after a box, a
    quiz, a shield or a bomb: a mission those finished showed paid only after the next run of clicks, up to a minute
    later (the owner: "it's perceived as a bug"). The stream needs an account: `NewStreamSessionReader` reads the token,
    since the unary session interceptor does not wrap streams, and a stream with none is refused `Unauthenticated`.
  - **A guest's progress counts and is kept, and its pay waits for sign-in.** `payroll` asks player whether the account
    is signed in (`rpc_player_accounts`, `player.v1.InternalService/GetAccounts`, which says what auth says: an
    account player does not answer is not signed in) only when a take finishes a mission. `player.v1.PlayerSignedIn`
    (`missions-signed-in`, `pay_signed_in_usecase`), which player tells at every sign-in and link, pays the day's
    missions done, so a guest who signs in that day gets them; a day that ended unpaid stays unpaid. A sign-in to an
    account linked long ago, or a switch of the browser to another account, pays what is due on the account signed in
    to, which is nothing unless a payment failed. Missions asks the day when it hears the sign-in, as it did when it
    heard auth's.
  - **Missions speaks player's language, not auth's.** Until 2026-11-04 it asked auth (`auth.v1.InternalService/GetAccount`,
    `rpc_auth_accounts`) and heard `auth.v1.SignedIn` (`missions-sign-ins`) and `auth.v1.AccountDeleted`
    (`missions-accounts`). `player.v1.PlayerCreated` was not enough to pay on: player tells it once per profile, and
    not when no name could be drawn, while a sign-in paid every time. Why the new groups miss no sign-in and no
    deletion is in [Player](player.md#the-deploy-that-moved-the-others-to-players-events).
  - **`GetMissions` and `ListenForEvents` are `Identified`**, and say which missions are paid. `player.v1.PlayerForgotten`,
    which player tells for every account auth deletes, deletes the account's briefs, sheets, progress and payments
    (`missions-forgotten`).

## The flag's mission

**One mission a day for the whole flag, the same for each of its players: done once, it gives the flag 3 to 10 season
points, the more the smaller the flag.** Every player of the flag counts on it. A new concept with its own lifecycle (a
day and a flag), so it has its own package and store (`flagmissions`).

- **`Draw(day, flag, world, yesterday)` is the mission**, seeded by the day and the flag: **its kind**, evenly between
  the ones the map allows: `TakeIn{Ground, Goal}`, tiles taken for the flag on a big ground (`World.GroundsOf`, the
  grounds of the player missions), or `BreakShields{Flag, Goal}`, shields of a rival broken (`World.RivalsOf`, drawn by
  the ground it holds, so the leaders share the heat). A map with neither gives a day with no mission.
- **Its numbers are the operator's** (see [The rules](#the-rules)): the goal's part of the day before, the least goal,
  and the points for the biggest flag and for a flag holding no tile, a `Sizing` that `Briefing.Of` reads at each draw
  and hands to `Draw`. The formulas stay in code; below, the numbers are those of 6 Nov 2026.
- **Its points come from the flag's size beside the biggest flag's** (`pointsOf`, the user's choice, 2026-11-05):
  most − (most − fewest) × the flag's tiles ÷ the biggest flag's tiles, rounded, from the map at the draw. With 10 and
  3, the biggest flag gets 3, a flag half its size 6, a flag under 1/14 of its size 10; on a blank map every flag gets
  10. On 5 Nov 2026 that was
  Ireland 3, Russia and Brazil 5, Czechia 6, Sweden and Bulgaria 7, Poland 8, Hungary, Palau, France and the UK 9, every
  smaller flag 10. Until then the points were drawn 3 to 10 evenly: in Season 1's first five days the flags holding 5% of the map or
  more won 57 points from missions, the flags under 0.5% won 13.
- **A ground has room for its goal** (`conquest`): its room is its tiles the flag does not hold (`Ground.Open`), and the
  ground is drawn evenly among those with room for the goal, so a busy flag is sent to Russia, Antarctica, the US, China,
  Canada, Brazil or Australia, and a quiet one anywhere big. With none, the ground with the most room, and the goal cut
  to it, rounded down to 10. Until 2026-11-02 the ground was drawn among all seventeen whatever the goal: France was
  told to take 8,000 tiles in Sudan, which has 3,536 (the owner: "this is ridiculous"). **That day was drawn again**:
  migration `20261102070000_redraw_conquests` deleted the conquests not done of the day that ran on 2 Nov at 06:00 UTC,
  so each of those flags drew again with the room at its next take or read, from 0 progress, keeping what its day had
  counted (the owner's choice: tile missions only, and tomorrow's goal sized from the whole day). A conquest done kept its
  points, and a shield mission, which the bug never touched, stayed.
- **The goal is 30% of the team's day before**, all on the one target, so a busy flag gets a big goal and a small one a
  small one (`goalOf`): 30% of all the tiles its players took and all the shields they broke, anywhere and on anyone,
  for either kind, at least 250 (`leastGoal`), rounded down to 10. A tile taken and a shield broken are a click each,
  so both kinds ask the same. The least is about ten minutes of one player's clicks, near a small flag's whole day,
  and is for a flag that did little the day before and for a season's first day. **The goal follows the team, not one
  player**: a flag of one player gets a goal of a third of one player's day. A flag puts some 5 to 10% of its day on
  one drawn ground by itself (France on Russia, 7% from 3 to 6 Nov), so 30% is done when its players go for it.
  Until 2026-11-06 it was half of the same kind only, at least 100 tiles or 30 shields: a flag that took open land
  broke few shields, so its shield goal stayed tiny, and the day after a battle every flag got the least (France: 30 of
  Sweden's shields for 9 points; the owner: "STUPIDLY EASY"). Until 2026-11-05 the goal was 50% for 3 points to
  100% for 10, at least 1,000 to 2,000 tiles or 300 to 600 shields: a flag under 0.5% of the map finished 2% of its
  140 days in Season 1's first five, its median goal 1,570 tiles while all its players took 180 to 360 a day (the user:
  "easier to reach for underdogs"). Until 2026-11-01 a mission gave 1 to 3 points (the owner: "1 to 3 is ridiculous").
  Until 2026-10-29 it was 20% to 40%, at least 300 tiles or 100 shields, and the first day after the deploy every flag
  got "break 150 shields" for 2 points (the owner: "the WHOLE TEAM works for it").
- **A flag's day counts what all its players did** (`Day`): every take for it (`takes`) and every shield it broke
  (`strikes`), for the next day's goal, and the mission's progress, one counter from 0. **`Briefing.Of(day, flag)`
  draws once and keeps it**, at the flag's first take, shield broken or read of the day, from the map then and the day
  before (`days.Store.DayBefore`, the day before it in its season, `number` − 1); a season's first day has none, so its
  goal is the least. **A battle between two days does not break the chain**: it takes its time from a day (see
  [A battle and the days](gamemaster.md#a-battle-and-the-days)), so the day before ends before the day starts. Until
  2026-11-06 the day before was the day that ended as this one started, so the day after a battle had none: on
  season 1's day 6, after a battle from 20:00 to 21:00, every flag got the least goal, 100 tiles or 30 shields (the
  owner: "STUPIDLY EASY"). Migration `20261106120000_day_numbers` numbered every day kept from its id. The store's first `KeepDay` wins. **A day kept with
  no mission is drawn again** at its next take, strike or read, and `KeepDay` gives it the first mission kept after,
  with the counts it already kept: that is how an operator's fix redraws a day, by deleting its `flag_missions` row
  (2026-11-02). Only a map with nothing to aim at draws no mission, and today's map always has a big ground the flag
  does not hold, so this costs no read of the map per take. `OfPlayer` is the day of the flag the account plays now
  (`GetAllegianceOf`), none with no flag.
- **Counted from two facts, each in its own group**: `TileTaken` (`missions-flag-takes`, `flag_tile_taken_subscriber`,
  `record_flag_take_usecase`), and `ShieldStruck` (`missions-flag-strikes`, `shield_struck_subscriber`,
  `record_strike_usecase`), every shield a click, a spread, an enclose or a bomb broke: `country` is the flag that
  broke it, `owner` the shield's. A bomb strikes its own flag's shields too, and they are no shield broken. **A fact
  told twice counts once**: each kind keeps a ledger mark (`missions.LedgerMark`: `counted_takes_below` and
  `counted_takes_below_index`, `counted_strikes_below` and `counted_strikes_below_index`), since a spread's takes and a
  bomb's strikes share one position. Until 2026-10-29 a strike was a `ClickTried` whose outcome was `STRUCK`: only
  the tile clicked counted, never a spread's, an enclose's or a bomb's (the owner found it the day it shipped).
- **The fact that finishes it gives the points, once** (`Day.WithTake`, `WithStrike`, `Done`): in the transaction
  that counts it, the use case records `scoring.v1.GrantPoints` in missions' outbox (`scoringgrant.Of`): the mission's id
  as `grant_id`, the flag, the points, and the fact's time as `granted_at`, so scoring counts them in the books open
  then. `missions-outbox` sends it with the charges.
- **And tells it, once** (`flagmessage.Done`): in the same transaction, `missions.v1.FlagMissionDone`, the flag, the
  mission as `GetMissions` shows it, done (`flagmessage.Mission`, the one encoder of a flag mission), and the fact's time
  as `done_at`. The chat hears it and announces it in the world and in the flag's room (see the chat's
  [Announcements](chat.md#announcements)); missions never learns who listens. Through the outbox because it is told
  once: a lost publish would never be told again. Migration `20261105200000_flag_mission_done` adds it to
  `outbox_event`.
- **`GetMissions` answers `flag_mission`**, the mission of the flag the caller plays now: its id, its points, `done`,
  and its kind (`take_in` with the tiles taken, `break_shields` with the shields broken). None with no flag, or no
  mission that day.
- **Kept in `flag_days`** (`day`, the day's id, `flag`, `takes`, `counted_takes_below`, `counted_takes_below_index`, `strikes`,
  `counted_strikes_below`, `counted_strikes_below_index`) and **`flag_missions`** (`mission_id`, `kind` of the enum `flag_mission_kind`, the kind's
  own fields in `payload`, `points`, `progress`, `done`): a day with no mission has no row there. They name no account, so `PlayerForgotten` leaves them.

## The rules

**The numbers missions draws with are one document in postgres, edited from the backoffice** (Ops → Mission rules) or
by the connector (`missions_save_rules`), not constants in the code (the owner, 2026-11-06: "we should be able to handle
this directly from the backoffice"). Only the flag mission's for now: `missions.v1.Rules.flag_missions`, the goal's
percent of the day before (1 to 100), the least goal (10 or more), and the points for the biggest flag (1 or more) and
for a flag holding no tile (that to 100). A player mission's spans and rewards are still code; they would join the same
document.

- **One document, no sets and no schedule**: unlike planet's and the antibot's rules, nothing switches it, since a
  battle runs between two days and no mission runs then. `missions.rules` holds one row (`singleton`), the document as
  protojson, seeded by migration `20261106180000_rules` with the numbers the code held (30, 250, 3, 10);
  `flagmissions.TestingSizing()` is the same, and `TestTheMigrationSeedsTheNumbersTheCodeHeld` compares them.
- **`GetRules`** reads it (`rules_query`). **`SaveRules`** replaces it whole (`save_rules_usecase`): `rules.RulesOf`
  checks every number with `flagmissions.SizingOf`, and a refusal names each, by its name in the proto
  (`flagMissions: … goalPercent is 0, and must be 1 to 100`), as `InvalidArgument`. `audit_save_rules` logs each save
  and refusal at Warn with the document.
- **A save plays at the next draw**: `Briefing.Of` reads the document from postgres when it draws a flag's day, which is
  once a day per flag, so most flags draw with the new numbers the next day; a day already drawn keeps its mission. A
  document postgres keeps that cannot play (edited by hand) fails the draw, and the fact is told again.

## The operator's reads

**What the game gave and how far the players took it, to tune the goals and the points from numbers, not guesses**
(the owner, 2026-11-06: "find good numbers rather than guess day after day"). `missions.v1.AdminService`, on the
console and admin listeners like every admin service, so each method is a tool of the connector too
(`missions_list_flag_days`, `missions_get_mission_totals`, see [Claude as a connector](connector.md)). Both read the
days told that **started** in the window, oldest first, and both are queries (see [Reads are queries](queries.md)).

- **`ListFlagDays(since, until)`** (`flag_days_query`): each day with every flag's day in it, the flag's mission as
  `GetMissions` shows it (`FlagMission`: target, goal, progress, points, done) and what all its players did
  (`tiles_taken`, `shields_broken`, the counts the next day's goal is drawn from). A window of 7 days at most: a day of
  every flag is about 15 KB of JSON, and a tool's answer must stay readable. The payload of each kind is read again in
  the query; `TestEveryKindTheStoreKeepsReadsAsItsPlayersSeeIt` keeps each through `postgres_flag_day_store` and asks
  for what `flagmessage.Mission` encodes.
- **`GetMissionTotals(since, until)`** (`totals_query`): each day's briefs, the accounts that finished one mission and
  all of them, and by `MissionKind` how many missions were drawn, started, done and paid, their goals and their
  progress summed. Counts, not rows: a day's briefs are about 700 missions, too many for a tool's answer. A mission is
  done once its progress reaches its goal, every kind's rule, written again in SQL;
  `TestEveryKindIsCountedDoneAndPaidAsTheSheetSaysItIs` keeps one of each kind through `postgres_mission_store` and
  asks for what `Sheet.Statuses` says. A window of 31 days at most.
- **Neither keeps when a flag mission was done**: `flag_missions` has no time. The chat's announcement has it.
