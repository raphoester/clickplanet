# Missions

Three to five missions a day of the player's own, drawn for the flag it played at its first read: a few
banks for the flag, tiles from a rival, tiles on a big country's ground, then more rivals and grounds, each
for a charge drawn with the day (shields, spread clicks, a bomb or enclosures). The backend's
[Missions doc](../../backend/docs/missions.md) has the rules; the client only shows them: a mission done pays
itself, with nothing to press.

- **A place of its own**: the Missions tab (phone) and menu tab (desktop), `app/missions/MissionsPlace.tsx`.
  Each row is its order's own parts (`partsOf`, one case per kind; a tile order is `orderOf`: "Take 40
  tiles from", then the flag and the name, and a bar with `taken/goal`), its reward in its bonus color, and
  **Done** once the backend says so (`done`: the client does not decide it). A guest sees
  the same rows and **Sign in to get it** on a mission done, which opens the sign-in pitch: its
  pay waits for the sign-in that day. "Ends in" is the day's own end, 21:00 UTC.
- **The flag's mission comes first**, in a panel of its own (`FlagMissionRow`): "Flag mission", its order ("Break 300
  shields of", then the flag; or a ground's tiles), its bar, and the season points it gives the flag in gold. The
  whole flag counts on it, and the backend says when it is done. The badge does not count it.
- **`useMissions` lives in `Viewer`, not in the place**, so the badge is right with the tab closed. It reads `GetMissions` at load, on a change of the pledged flag (`useAllegiance`), after
  each run of clicks (`useReadsAfterClicks`) and once more `FOLLOW_UP_MS` (3 s) later, and once the day ends.
  The read after clicks comes 2 s after the last one, and the backend counts a take one to three seconds
  after it answered it (the ledger's flush, its relay, the missions' handler): the second read is what
  catches the take that finished a mission when the player stopped clicking, as one does who just
  spent a bank.
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
  `FakeMissionsBackend` the fake: it draws the day for the flag pledged at its first read (or its first
  take's), with a flag mission to break a rival's shields that this tab's takes stand for, counts every take of this tab, pays each mission done at once, and `finishMissions()` in the console puts
  every mission one tile from done: the next take finishes and pays them all, as a take does on the server.
