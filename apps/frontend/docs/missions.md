# Missions

Four or five missions a day of the player's own, drawn for the flag it played at its first read: a few
banks for the flag, tiles from a rival, tiles on a big country's ground, then one or two of the other kinds
(boxes, quizzes, bombs on a country, tiles at 10 shields, tiles fortified, tiles of a rival in a country), each
for a charge drawn with the day (shields, spread clicks, a bomb or enclosures). The backend's
[Missions doc](../../backend/docs/missions.md) has the rules; the client only shows them: a mission done pays
itself, with nothing to press.

- **A place of its own**: the Missions tab (phone) and menu tab (desktop), `app/missions/MissionsPlace.tsx`.
  Each row is its order's own parts (`partsOf`): its words (`orderOf`, one case per kind: "Take 40 tiles
  from", "Drop 2 bombs on", "Catch 4 boxes"), then the flag and the name of each country it names, and a bar
  with how far it went (`reachedOf`) over its goal, its reward in its bonus color, and
  **Done** once the backend says so (`done`: the client does not decide it). A guest sees
  the same rows and **Sign in to get it** on a mission done, which opens the sign-in pitch: its
  pay waits for the sign-in that day. "Ends in" is the day's own end, 21:00 UTC. With no mission (no flag
  pledged yet, or no day), a guest sees "Sign in to get missions." and a **Sign in** button to the pitch, a
  signed-in player "No missions right now.": an empty place looked broken.
- **Go**, on each mission open with a place (`GoButton`, the flag's included): the sheet closes and the camera flies
  down to the nearest tile, from where it looks, that counts on it (`Globe.flyToMission`, `nearestPlayable` in
  `domain/missionTargets.ts`): a rival's tile for a rival, a tile of the ground the flag does not hold for a ground.
- **The missions are drawn on the map** (`useMissionTargets`, `Globe.setMissions`, `domain/missionTargets.ts`): what an
  open mission names is its `MissionTarget`. **One red for a mission's place**, the outline and the rings alike (the
  owner, 2026-11-05: one colour for one concept). **A ground** (tiles or bombs in a country, a rival in a country) **is
  outlined** at every zoom (`setAimed` on the border lines, see [The countries' outlines](country-outlines.md)).
  **A rival's tiles that count** (all of them, those in its ground, or only the shielded ones for shields to break) **get
  a red ring** (`MissionMarks`, bit 4 of the tile's `pulse` slot): still, never beating, so it is not the gold fortify
  ring nor the white front, and only pushed in, since in orbit the painted flag already says whose the ground is. A
  mission done is no longer drawn. **A first visit draws none**: its front is lit already (see
  [The front, for a first visit](front-guide.md)). Each new list of targets is a scan of every tile, so the hook keeps
  its identity while the targets are the same. On 2026-11-05 the owner judged a cyan dot in a tile's corner horrible:
  the ring is the target, red and centred, and the outline went from cyan to the same red.
- **The cards beside the pointer name them**: the tile's and the territory's say which missions the red is for, in a
  strip with a small ring of the same red (`MissionsHere`, see [Who holds a tile](tile-holder.md) and
  [A territory, from orbit](territory-card.md)). The owner, 2026-11-05: the red alone did not say why it was red.
- **The progress line** (`MissionProgress`, `useMissionProgress`, `domain/missionProgress.ts`): when a day read or
  streamed moved a mission, the Moments zone shows it in one line, its words short ("Take from Algeria"), a small bar
  and the count. **One mission at a time**: the line keeps the mission it shows while it moves, and goes
  `PROGRESS_LINE_MS` (3 s) after its last move; only then can another take it, the one nearest its goal. **Never the
  bank for the flag**, which every take moves: the line would never go. A mission done takes it away at once, for
  its card. It gives way to every other Moments line (a bomb, a fortify, an attack) and to the "Mission done" card,
  and a first visit shows none. It moves while the player clicks because the stream sends the day after each fact,
  once a second at most (see the backend's [Missions](../../backend/docs/missions.md)).
- **The flag's mission comes first**, in a panel of its own (`FlagMissionRow`): "Flag mission", its order ("Break 300
  shields of", then the flag; or a ground's tiles), its bar, and the season points it gives the flag in gold. The
  whole flag counts on it, and the backend says when it is done. The badge does not count it.
- **The flag's mission done is cheered full screen**, `FlagMissionDone`: "Congratulations!", the flag on the rays as
  a round's champion is (no medal: medals are titles'), its name, the season points won and the order done. Every player of the flag sees it at the read that first says it
  is done (`useFlagMissionDone`, `newlyDoneFlagMission`: the same mission read not done before; the first read of
  the page tells nothing), so a player who stopped clicking sees it at the next run of clicks. It plays through the
  announcements queue (see [Battles](battle.md#the-announcements-queue)). Until 2026-10-29 nothing said it, and the
  owner, who finished France's, could not tell if the points were granted. **Every other player reads it in the chat**:
  the server announces each flag mission done in the World and in the flag's room (see the
  [Live chat](chat.md#announcements)).
- **`useMissions` lives in `Viewer`, not in the place**, so the badge is right with the tab closed. It reads `GetMissions` at load, on a change of the account or of the pledged flag (`useAllegiance`), after
  each run of clicks (`useReadsAfterClicks`, 2 s after the last one, for the bars), and once the day ends.
  **Progress and payments come on the missions stream** (`MissionsBackend.follow`, `MissionService.ListenForEvents`),
  followed while the account is known and again when it changes: the stream sends the whole day as it opens and
  after each fact of the player, once a second at most, whatever it was (a take, a box, a quiz, a shield, a bomb). A read that
  started before a streamed day is dropped (`streamed`), or it could put a paid mission back to unpaid and show
  its card twice. Until 2026-11-03 a second read 3 s after the clicks was what caught a mission finished: a box,
  a quiz, a shield or a bomb read nothing, and its reward showed up to a minute late.
- **The badge counts what is news**, as the chat's counts what was missed: on the phone's tab
  (`TabBar`) and the desktop menu's tab (`Menu`, `missionsNews`), "Missions, 3 new". A mission is news
  each time it moves on a stage, drawn, then done, then paid (`stageOf`), and the badge counts the ones
  past the stage last seen (`unseen`). So a new day shows 3, and a mission paid shows 1. Showing the
  place marks what it shows as seen (`seenNow`). The mark is kept on the device, in
  `localStorage` (`clickplanet-missions-seen`, the store `useSeenMissions` in `seenMissionsStore.ts`),
  with the shown day's ids alone.
- **A mission paid has its own card**, `MissionsDone`: "Mission done", or "3 missions done" for a
  burst, then each order with its flag and the reward it paid. One card for all the missions of a read,
  never a card each; it goes by itself like a box's card (`usePassing`, 2.9 s). `usePaidMissions`
  compares each day with the one before (`newlyPaid`; the first day of the page tells nothing). The
  charges come on planet's own stream (`charges_granted`, see [Charges](charges.md)), within about a
  second of the card: the client reads no charges for a mission. The charge's own card (`BonusAward`)
  is for a box or a quiz, not a mission.
- **A `Mission` is its id, its `order`, its reward, `done` and `paid`.** The order is one shape per kind
  (`MissionOrder`: `TileOrder` and `RaidOrder` count `taken`, `BombOrder`, `ShieldOrder` and `CountOrder` count
  `reached`), and a flag mission's (`FlagMissionOrder`) is a `TileOrder` or a `BreakOrder`, which counts `reached`
  too; `ConnectMissionsBackend` reads the proto's `oneof` and skips a mission whose kind or reward it does
  not know.
- `backends/missions.ts` is the contract, `ConnectMissionsBackend` (`missions.v1.MissionService`) the real one,
  `FakeMissionsBackend` the fake: it draws the day for the flag pledged at its first read (or its first
  take's), with a flag mission to break a rival's shields that this tab's takes stand for, counts every take of this tab, pays each mission done at once and hands its followers the day after each take, and `finishMissions()` in the console puts
  every mission one tile from done: the next take finishes and pays them all, as a take does on the server, and
  cheers the flag's, which it tells the fake chat once (`listenForFlagMissionsDone`). The fake has no missions before the tab's first take: take a tile, then call it.
