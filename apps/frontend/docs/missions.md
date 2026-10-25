# Missions

Three missions a day for the flag played, the same for each of its players: one bank for the
flag, tiles from the leader, tiles on a country's ground, each for a charge. The backend's
[Missions doc](../../backend/docs/missions.md) has the rules; the client only shows them and collects.

- **A place of its own**: the Missions tab (phone) and menu tab (desktop), `app/missions/MissionsPlace.tsx`.
  Each row is the order (`orderOf`: "Take 40 tiles from", then the flag and the name), its
  reward in its bonus color, a bar and `progress/goal`, and **Collect** once done. A guest sees
  the same rows and **Sign in to collect**, which opens the sign-in pitch: its progress is kept
  for the day. "Ends in" is the day's own end, 21:00 UTC.
- **`useMissions` lives in `Viewer`, not in the place**, so the tab's badge (`collectable`) is
  right with the tab closed. It reads `GetMissions` at load, on a change of the pledged flag (`useAllegiance`), after
  each run of clicks (`useReadsAfterClicks`) and once the day ends.
- **Collecting is the box's moment**: `collectMission` answers the kind and the amount kept;
  `Viewer` shows it as a `BonusAward` (`awardOf`) and asks the planet backend for the charges
  again (`ChargesRefresher.refreshCharges`), since the player module granted them through
  planet and no stream says so. A reward that did not fit (a bomb held) reads `Full` on the
  button and stays to collect.
- `backends/missions.ts` is the contract, `ConnectMissionsBackend` (`missions.v1.MissionService`) the real one,
  `FakeMissionsBackend` the fake: it counts this tab's takes for its first flag, and
  `finishMissions()` in the console makes every mission done.
