# Clip stats

`src/clips/`, one page (`ClipStatsPage`). **The backend keeps the rows, the page makes the table**, as the home page
counts from the ledger: `clips.v1.AdminService` (the backend's [Clip stats](../../backend/docs/clip-stats.md)) holds a snapshot a day
of each clip on each platform, which the clips job's `clips stats` sends, each row the JSON the job wrote: what
Zernio answered, as it came, and what the clip was.

- **Whether TikTok's business lane is live**, from the last snapshot (`ListSnapshots`): each TikTok account, the type
  TikTok said it was when it was connected to Zernio, how many of its rows hold a number only the business lane gives,
  whether the commercial music library answers (its refusal on hover), and **live** or not: live is the library
  answering or a business number on a post, whatever the type says, since the type is what TikTok said when the
  account was connected to Zernio and goes stale.
- **The snapshots of the last fortnight**: when each was taken, the clips read (and how many not out yet), its rows,
  those still syncing, YouTube's retention curves, and the clips lost that day (why, on hover). A day with no line is
  a day the workflow did not keep.
- **Download CSV** (`tableFileOf` in `actions.ts`): the rows of the last snapshot (each clip once, at its latest age),
  the last 7 or 30 days, or everything, every page of `ListRows` (2000 a page), saved as
  `clip-stats-<since>-<until>.csv` from memory (`saveText`, as the replay is). **`tableOf` (`table.ts`) makes the
  columns from the rows**: the row's own first (`takenAt`, `postId`, `platform`, `ageHours`, …), then one for each
  metric (`metrics.views`, `metrics.impressionSources.forYou`), each part of the metadata (`meta.storyFlag`,
  `meta.record.clip.variants.<experiment>`: an arm is a column with no code) and YouTube's curve (`youtube.points`,
  `youtube.meanRelativeRetentionPerformance`, the mean of its points, and the curve itself). A list, and the two maps
  with a key a country or a second (`audienceCountries`, TikTok's `retentionCurve`), are one cell of JSON. **A metric
  Zernio adds is a column at once**: nothing on the way names a metric. `csvOf` quotes as RFC 4180 says.
- **The numbers are not public**, unlike the rest of the clips job's output: they live behind the admin gate, and
  nowhere in the repository or its workflows.
