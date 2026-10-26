# Missions

Three missions a day for the flag played, the same for each of its players: one bank for the
flag, tiles from the leader, tiles on a country's ground, each for a charge. The backend's
[Missions doc](../../backend/docs/missions.md) has the rules; the client only shows them: a mission done pays
itself, with nothing to press.

- **A place of its own**: the Missions tab (phone) and menu tab (desktop), `app/missions/MissionsPlace.tsx`.
  Each row is its order's own parts (`partsOf`, one case per kind; a tile order is `orderOf`: "Take 40
  tiles from", then the flag and the name, and a bar with `taken/goal`), its reward in its bonus color, and
  **Done** once the backend says so (`done`: the client does not decide it). A guest sees
  the same rows and **Sign in to get it** on a mission done, which opens the sign-in pitch: its
  pay waits for the sign-in that day. "Ends in" is the day's own end, 21:00 UTC.
- **`useMissions` lives in `Viewer`, not in the place**, so the tab's badge (`waiting`: done, not paid) is
  right with the tab closed. It reads `GetMissions` at load, on a change of the pledged flag (`useAllegiance`), after
  each run of clicks (`useReadsAfterClicks`) and once the day ends.
- **A mission paid is the box's moment**: `usePaidMissions` compares each read with the one
  before (`newlyPaid`; the first read of the page tells nothing), and `Viewer` shows each reward as
  a `BonusAward` (`awardOf`). It asks planet for the charges at once and again `GRANT_LANDS_MS`
  (2 s) later: planet grants from a command, after the read that says the mission is paid, and
  no stream says so.
- **A `Mission` is its id, its `order`, its reward, `done` and `paid`.** The order is one shape per kind
  (`MissionOrder`, `TileOrder` today); `ConnectMissionsBackend` reads the proto's `oneof` and skips a mission whose
  kind or reward it does not know.
- `backends/missions.ts` is the contract, `ConnectMissionsBackend` (`missions.v1.MissionService`) the real one,
  `FakeMissionsBackend` the fake: it counts this tab's takes for its first flag (the pledged one
  before), pays each mission done at once, and `finishMissions()` in the console makes every
  mission done.
