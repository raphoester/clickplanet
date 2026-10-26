# Missions (`internal/missions/`)

**Every flag has three missions a day, the same for each of its players, and each pays itself once done.** It reads
planet's facts (`TileTaken`, `GetTerritories`, `GetCountrySizes`, `GetAllegianceOf`) and commands planet to grant a reward
(`planet.v1.GrantCharges`, over Redis), as seasons asks planet what each country holds: planet never imports, calls or
hears it.
Its public route, `/missions.v1.MissionService/*`, is listed in `deploy/vps/caddy/Caddyfile`, as every service's is.

```
internal/missions/internal/
  missions/                         AccountID, Calendar and Day (the cutoff), Take, MissionID, Kind, Mission (one type per kind),
                                    Brief, Draw and World, Sheet (an account's day), Briefing; the Store port and its contract suite
    postgres_mission_store/         the Store over missions.mission_briefs, mission_sheets, mission_progress and mission_payments
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
  date it ends on. **The rules are objects**: a `Mission` is an id, a goal, a reward and one order, `takeFor` (tiles
  for the flag), `takeFrom` (tiles another flag held) or `takeIn` (tiles on a country's ground), each with its own
  `counts(Take)`.
  - **`Draw(day, flag, world)` is the brief**, seeded by the day and the flag: 1. take 60 tiles (one bank) for the flag,
    for 10 shields; 2. take 40 tiles from whoever holds the most ground but the flag, so every flag but the leader aims
    at the leader, for 4 spread clicks (with no other flag holding 400 tiles, 120 for the flag); 3. take 200 tiles on a
    ground of 1,000 tiles or more that is not the flag's, for a bomb (with none, 200 for the flag). **Never a refill**:
    a mission pays what multiplies or guards clicks, not clicks. The world is planet's (`GetTerritories`, and
    `GetCountrySizes`, `clicks.Borders.CountrySizes`).
  - **`Briefing.Of(day, flag)` draws once and keeps it**: the store's first `KeepBrief` wins, so two takes racing to
    draw a flag's day read one brief, and a map that moves later never changes it.
  - **A mission is named by its `MissionID`**, a UUID the draw derives from the day, the flag and its place in the
    brief (`drawnID`), so the same draw names it the same. Progress and payments point at it,
    and nothing caps how many missions a brief holds.
  - **A brief is one `jsonb` array per day and flag** (`mission_briefs.missions`), the missions in the order drawn.
    Each is its id, kind, goal and reward, then its kind's own fields (`target` today, `omitempty`): a kind with
    other fields, or none, needs code and no migration.
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
