# Missions (`internal/missions/`)

**Every flag has three to five missions a day, the same for each of its players, and each pays itself once done.** It reads
planet's facts (`TileTaken`, `GetTerritories`, `GetCountrySizes`, `GetAllegianceOf`) and commands planet to grant a reward
(`planet.v1.GrantCharges`, over Redis), as seasons asks planet what each country holds: planet never imports, calls or
hears it.
Its public route, `/missions.v1.MissionService/*`, is listed in `deploy/vps/caddy/Caddyfile`, as every service's is.

```
internal/missions/internal/
  missions/                         AccountID, Calendar and Day (the cutoff), Take, MissionID, Order (one type per kind), Mission,
                                    Brief, Draw and World, Sheet (an account's day), Briefing; the Store port and its contract suite
    postgres_mission_store/         the Store over missions.mission_briefs, mission_sheets, mission_progress and mission_payments;
                                    records.go is the jsonb format of a mission, one payload per kind
    inmemory_mission_store/         the same port in maps, behind the testing tag
    rpc_planet_world/               what each flag holds and each ground's size, from planet.v1.InternalService
    rpc_planet_allegiance/          the flag an account pledged, from planet.v1.InternalService/GetAllegianceOf
    rpc_auth_accounts/              whether an account is signed in, from auth.v1.InternalService/GetAccount
    payroll/                        pays each mission done once: the payments and their grants, in one transaction
    planetgrant/                    a payment as the planet.v1.GrantCharges command
    usecases/record_take_usecase/  get_missions_usecase/  pay_signed_in_usecase/  forget_account_usecase/
  missionsv1controller/             MissionService (a bag), the session interceptor, the caller, missionmessage
    get_missions_handler/
  subscribers/                      tile_taken_subscriber/  sign_in_subscriber/  account_deleted_subscriber/  log_subscriber/
  migrations/
```

- **A day ends at `missions.cutoff`** (21:00 UTC in production), the time the daily round closes, and is named by the
  date it ends on. **The rules are objects**: a `Mission` is an id, an `Order` and a reward, and nothing else.
  **An order is one kind of mission**, a type of its own with its own fields: `TakeFor{Flag, Goal}` (tiles for the
  flag), `TakeFrom{Flag, Goal}` (tiles another flag held), `TakeIn{Ground, Goal}` (tiles on a country's land). It decides
  how a take moves its progress (`advanced`) and when it is done (`done`). The progress of a mission is one counter
  from 0 that only its order reads. A new kind is a new type and its cases in the store and the encoder.
  - **`Draw(day, flag, world)` is the brief**, seeded by the day and the flag, so a day has its own count and goals.
    **Every goal is drawn from its span**, in steps of 10 tiles (`span`, `draw.goal`): 1. take 120 to 240 tiles (two
    to four banks) for the flag; 2. take 80 to 160 tiles from a rival, a flag holding 1,000 tiles (`rivalAtLeast`),
    drawn by the ground it holds (`draw.ranked`): a flag holding a third of the map is the target of a third of the
    flags, so the leaders share the heat and a small rival gets its turn (with no rival, twice that for the flag);
    3. take 300 to 600 tiles on a ground of 3,000 tiles or more (`groundAtLeast`) that is not the flag's and of which
    1,000 tiles or more are not the flag's (`openAtLeast`), seventeen countries on today's map (with none, the same
    for the flag). **Then up to two more**, since a day has 3 to 5 missions (`fewestMissions`, `mostMissions`): each
    is another rival, with a rival's goal, or another big ground, with a ground's goal, drawn evenly from the ones
    left; a world with none left gives a shorter day. The goals were a fixed 60, 40 and 200 until 2026-10-26, when a
    top player finished all three in ten minutes. Until 2026-10-27 every flag aimed at the one leader, so everyone
    attacked the same flag, and a flag could be sent to a ground it already held whole. The world is planet's
    (`GetTerritories`: what each flag holds, and how many tiles of it on each country's land; `GetCountrySizes`,
    `clicks.Borders.CountrySizes`).
  - **The rewards are drawn with the day too** (`draw.rewards`, from the `purses`): the four charges in a random
    order, each once before any twice, so a day of five pays one charge twice, each as much as its purse gives: 15 to
    30 shields, 4 to 8 spread clicks, one bomb, or 2 to 3 enclosures. A purse starts above what a box gives and tops
    at the stack planet keeps (`bonus.*.held`), since a grant past it is lost. **Never a refill**: a mission pays
    what multiplies or guards clicks, not clicks.
  - **`Briefing.Of(day, flag)` draws once and keeps it**: the store's first `KeepBrief` wins, so two takes racing to
    draw a flag's day read one brief, and a map that moves later never changes it.
  - **A mission is named by its `MissionID`**, a UUID the draw derives from the day, the flag and its place in the
    brief (`drawnID`), so the same draw names it the same. Progress and payments point at it,
    and nothing caps how many missions a brief holds.
  - **A brief is one `jsonb` array per day and flag** (`mission_briefs.missions`), the missions in the order drawn.
    Each is `{id, kind, reward, payload}`; the payload is the kind's own fields (`{"flag", "goal"}` for `take_for`).
    A kind with other fields, or none, needs code and no migration. A kind the store does not know is an error.
  - **`GetMissions` answers each mission in its kind's own message** (`missionmessage`, `Sheet.Statuses`): what every
    mission has is its id, its reward, `done` and `paid`; what it asks is a `oneof` (`take_for`, `take_from`,
    `take_in`, each with its country, goal and tiles taken). A client skips a kind it does not know, so a new kind
    reaches an old client as nothing. The reward is a `oneof` too (`charges` today).
  - **A day is locked to the flag of its first take** (`Sheet.WithTake`): a take for another flag counts nothing, so a
    change of flag cannot farm a second set. Until that take, `GetMissions` answers the flag the caller pledged
    (`rpc_planet_allegiance`, planet's `GetAllegianceOf`), not enlisted, and no missions for an account that never
    pledged one. The client never names the flag. **Progress is counted from `TileTaken`** (`missions-progress`, `tile_taken_subscriber`), up to each goal,
    in one transaction under `pg_advisory_xact_lock` per take, with nothing written once nothing changed. **Not from the
    ledger**: `ledger_events` has no time index, and an account's takes since the cutoff would read every take it ever
    made. **A take told twice counts once**: Redis delivers at least once, and the sheet keeps the ledger position after
    the last take it counted (`counted_takes_below`), so one below it counts nothing.
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
    sheets, progress and payments (`missions-accounts`). The briefs name nobody and stay.
