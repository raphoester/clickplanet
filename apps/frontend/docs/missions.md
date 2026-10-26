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
- **`useMissions` lives in `Viewer`, not in the place**, so the badge is right with the tab closed. It reads `GetMissions` at load, on a change of the pledged flag (`useAllegiance`), after
  each run of clicks (`useReadsAfterClicks`) and once the day ends.
- **The badge counts what is news**, as the chat's counts what was missed: on the phone's tab
  (`TabBar`) and the desktop menu's tab (`Menu`, `missionsNews`), "Missions, 3 new". A mission is news
  each time it moves on a stage, drawn, then done, then paid (`stageOf`), and the badge counts the ones
  past the stage last seen (`unseen`). So a new day shows 3, and a mission paid shows 1. Showing the
  place marks what it shows as seen (`onSeen`, `seenNow`). The mark is kept on the device, in
  `localStorage` (`clickplanet-missions-seen`, `useSeenMissions`), with the shown day's ids alone.
- **A mission paid has its own card**, `MissionsDone`: "Mission done", or "3 missions done" for a
  burst, then each order with its flag and the reward it paid. One card for all the missions of a read,
  never a card each; it goes by itself like a box's card (`usePassing`, 2.9 s). `usePaidMissions`
  compares each read with the one before (`newlyPaid`; the first read of the page tells nothing).
  `Viewer` asks planet for the charges at once and again `GRANT_LANDS_MS` (2 s) later: planet grants
  from a command, after the read that says the mission is paid, and no stream says so. The charge's
  own card (`BonusAward`) is for a box or a quiz, not a mission.
- **A `Mission` is its id, its `order`, its reward, `done` and `paid`.** The order is one shape per kind
  (`MissionOrder`, `TileOrder` today); `ConnectMissionsBackend` reads the proto's `oneof` and skips a mission whose
  kind or reward it does not know.
- `backends/missions.ts` is the contract, `ConnectMissionsBackend` (`missions.v1.MissionService`) the real one,
  `FakeMissionsBackend` the fake: it counts this tab's takes for its first flag (the pledged one
  before), pays each mission done at once, and `finishMissions()` in the console makes every
  mission done and paid; the client shows it at its next read (a click, or a change of flag).
