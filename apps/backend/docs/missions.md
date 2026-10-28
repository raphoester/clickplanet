# Missions (`internal/missions/`)

**Every player has four or five missions a day of its own, and each pays itself once done.** It reads
planet's facts (`TileTaken`, `BonusClaimed`, `QuizAnswered`, `BombLanded`, `ShieldPlaced`, `Fortified`, `GetTerritories`,
`GetCountrySizes`, `GetAllegianceOf`) and commands planet to grant a reward
(`planet.v1.GrantCharges`, over Redis), as scoring asks planet what each country holds: planet never imports, calls or
hears it.
Its public route, `/missions.v1.MissionService/*`, is listed in `deploy/vps/caddy/Caddyfile`, as every service's is.

```
internal/missions/internal/
  missions/                         AccountID, Calendar and Day (the cutoff), Fact (one type per kind) and Counted, MissionID,
                                    Order (one type per kind), Mission, Brief, Draw and World, Sheet (an account's day), Briefing;
                                    the Store port and its contract suite
    postgres_mission_store/         the Store over missions.mission_briefs, mission_sheets, mission_progress and mission_payments;
                                    records.go is the jsonb format of a mission, one payload per kind
    inmemory_mission_store/         the same port in maps, behind the testing tag
    rpc_planet_world/               what each flag holds and each ground's size, from planet.v1.InternalService
    rpc_planet_allegiance/          the flag an account pledged, from planet.v1.InternalService/GetAllegianceOf
    rpc_auth_accounts/              whether an account is signed in, from auth.v1.InternalService/GetAccount
    payroll/                        pays each mission done once: the payments and their grants, in one transaction
    planetgrant/                    a payment as the planet.v1.GrantCharges command
    usecases/record_fact_usecase/  get_missions_usecase/  pay_signed_in_usecase/  forget_account_usecase/
  missionsv1controller/             MissionService (a bag), the session interceptor, the caller, missionmessage
    get_missions_handler/
  subscribers/                      tile_taken_subscriber/  bonus_claimed_subscriber/  quiz_answered_subscriber/
                                    bomb_landed_subscriber/  shield_placed_subscriber/  fortified_subscriber/
                                    sign_in_subscriber/  account_deleted_subscriber/  log_subscriber/
  migrations/
```

- **A day ends at `missions.cutoff`** (21:00 UTC in production), the time the daily round closes, and is named by the
  date it ends on. **The rules are objects**: a `Mission` is an id, an `Order` and a reward, and nothing else.
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
    time index, and an account's takes since the cutoff would read every take it ever made. **A fact told twice counts
    once**: Redis delivers at least once, and the sheet keeps a mark per kind (`missions.Counted`, the
    `mission_sheets.counted_*` columns): the ledger position after the last take, bomb, shield and fortify it counted,
    and the time of the last box and right answer, so one at or below its mark counts nothing. Each kind has its own
    mark because each comes on its own stream, in no order with the others. Two boxes or answers of one account in the
    same microsecond count once.
  - **A mission done is paid at once, with no claim** (`payroll.Pay`, `Sheet.WithPayments`). The take that finishes it
    writes the payment (`mission_payments`) and records the `planet.v1.GrantCharges` command in missions' outbox, in
    the transaction that counts the take: both are kept, or neither and the take is told again. `missions-outbox`
    sends it to planet, which grants it once per `grant_id` (`planetgrant.Of`: the account and the mission, so a
    payment told twice names one grant). **What does not fit the stack is lost**, as planet grants any charge: no
    reply comes back.
  - **A guest's progress counts and is kept, and its pay waits for sign-in.** `payroll` asks auth whether the account
    is signed in (`rpc_auth_accounts`) only when a take finishes a mission. `auth.v1.SignedIn` (`missions-sign-ins`,
    `pay_signed_in_usecase`) pays the day's missions done, so a guest who signs in that day gets them; a day that
    ended unpaid stays unpaid.
  - **`GetMissions` is `Identified`**, and says which missions are paid. `auth.v1.AccountDeleted` deletes the account's
    briefs, sheets, progress and payments (`missions-accounts`).
