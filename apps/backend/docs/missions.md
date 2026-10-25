# Missions (`internal/missions/`)

**Every flag has three missions a day, the same for each of its players, and what collecting one pays.** It reads
planet's facts (`TileTaken`, `GetTerritories`, `GetGrounds`) and asks planet to grant a reward (`GrantCharge`), as
seasons asks planet what each country holds: planet never imports, calls or hears it.
Its public route, `/missions.v1.MissionService/*`, is listed in `deploy/vps/caddy/Caddyfile`, as every service's is.

```
internal/missions/internal/
  missions/                         AccountID, Calendar and Day (the cutoff), Take, Slot, Kind, Mission (one type per kind),
                                    Brief, Draw and World, Sheet (an account's day), Briefing; the Store port and its contract suite
    postgres_mission_store/         the Store over missions.mission_briefs, mission_sheets, mission_progress and mission_collections
    inmemory_mission_store/         the same port in maps, behind the testing tag
    rpc_planet_world/               what each flag holds and each ground's size, from planet.v1.InternalService
    rpc_planet_charges/             a mission's reward, through planet.v1.InternalService/GrantCharge
    usecases/record_take_usecase/  get_missions_usecase/  collect_mission_usecase/  forget_account_usecase/
  missionsv1controller/             MissionService (a bag), the session interceptor, the caller, missionmessage
    get_missions_handler/  collect_mission_handler/
  subscribers/                      tile_taken_subscriber/  account_deleted_subscriber/  log_subscriber/
  migrations/
```

- **A day ends at `missions.cutoff`** (21:00 UTC in production), the time the daily round closes, and is named by the
  date it ends on. **The rules are objects**: a `Mission` is a slot, a goal, a reward and one order, `takeFor` (tiles
  for the flag), `takeFrom` (tiles another flag held) or `takeIn` (tiles on a country's ground), each with its own
  `counts(Take)`.
  - **`Draw(day, flag, world)` is the brief**, seeded by the day and the flag: 1. take 60 tiles (one bank) for the flag,
    for 10 shields; 2. take 40 tiles from whoever holds the most ground but the flag, so every flag but the leader aims
    at the leader, for 4 spread clicks (with no other flag holding 400 tiles, 120 for the flag); 3. take 200 tiles on a
    ground of 1,000 tiles or more that is not the flag's, for a bomb (with none, 200 for the flag). **Never a refill**:
    a mission pays what multiplies or guards clicks, not clicks. The world is planet's (`GetTerritories`, and
    `GetGrounds`, `clicks.Borders.Grounds`).
  - **`Briefing.Of(day, flag)` draws once and keeps it**: the store's first `KeepBrief` wins, so two takes racing to
    draw a flag's day read one brief, and a map that moves later never changes it.
  - **A day is locked to the flag of its first take** (`Sheet.WithTake`): a take for another flag counts nothing, so a
    change of flag cannot farm a second set. Until that take, `GetMissions(country_id)` answers the asked flag's, not
    enlisted. **Progress is counted from `TileTaken`** (`missions-progress`, `tile_taken_subscriber`), up to each goal,
    in one transaction under `pg_advisory_xact_lock` per take, with nothing written once nothing changed. **Not from the
    ledger**: `ledger_events` has no time index, and an account's takes since the cutoff would read every take it ever
    made. **A take told twice counts once**: Redis delivers at least once, and the sheet keeps the ledger position after
    the last take it counted (`counted_takes_below`), so one below it counts nothing.
  - **A guest's progress counts and is kept**, so a guest who signs in that day collects it. **`CollectMission(slot)`
    pays only a signed-in account** (the click token's linked byte; `PermissionDenied` otherwise), once per slot and
    day: `mission_collections` is written in a transaction that commits only if `GrantCharge` kept something. A full
    stack (a bomb held) is `FailedPrecondition` and the mission stays to collect later; so is a mission not done or
    collected already. The answer is what fit.
  - **`GetMissions` is `Identified`, `CollectMission` `Attested`.** `auth.v1.AccountDeleted` deletes the account's
    sheets, progress and collections (`missions-accounts`). The briefs name nobody and stay.
