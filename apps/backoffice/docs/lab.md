# The Lab

`src/lab/`, one page (`LabPage`, `#/lab`, under Clips). **The numbers of the video lab, for the owner's verdicts**:
each experiment on each platform against the control of the same days, and each day's arrivals by platform. What the
lab is and how a verdict is made: the root's [The video lab](../../../docs/video-lab.md). **The backend gives raw
facts and the page counts** (`readout.ts`), as the home page does:

- **The posts**: every row of the snapshots taken since the day picked (`publishing.v1.AdminService/ListRows`, every
  page, `documentsOf` of [Clip stats](clip-stats.md)), see the backend's [The posts' numbers](../../backend/docs/post-stats.md).
- **The arrivals**: `auth.v1.AdminService/CountArrivals`, the accounts made each UTC day by source, see the backend's
  [Arrivals](../../backend/docs/arrivals.md). The two reads are apart: arrivals that fail leave the posts shown.

## What it counts

- **A post is the lab's or the control's by its metadata**: `record.post.experiment` (`control`, `far-invasion`,
  `carousel`, …). A post with none (a clip from before the lab, a battle teaser, a post made by hand) is left out.
- **Each post is read at its last row at 7 days or younger** (`ageHours` ≤ 168): its views by day 7, or now if it is
  younger. The posts of both arms are of the same days, so the young ones weigh on both alike.
- **A row with no numbers yet is not read**: while Zernio syncs a post (`syncStatus` `pending`), its
  `analytics.analytics` is `null`. The post's row before it counts, or the post waits for the next snapshot, rather
  than count as 0 views. Any field of Zernio's answer can be `null`, so the page reads `null` as missing.
- **The days picked**: posts out since that UTC day, from 2026-11-20 (the lab's start, and the day TikTok's arrivals
  were first told apart) to today. Two weeks back by default, never before the lab's start. A day further back reads
  more snapshots: about 3 MB of rows each, 230 KB compressed.
- **An experiment on a platform** spans the UTC days from its first post to its last there. **Its control** is the
  control's posts on that platform on those days, so each experiment has its own control row.
- **Per experiment and control**: the posts, the median and the best views, and the **breakouts**, posts with 3,000
  views or more on TikTok and 10,000 on YouTube (`BREAKOUT`).
- **TikTok's signs**: held at 2 s (the `retentionCurve`'s second 2; none for photos), to the end (`completionRate`),
  the average watch (`igReelsAvgWatchTime`, in ms), and comments, shares, follows and saves per 1,000 views. They are
  read only on the posts whose business numbers came, which takes 24 to 48 hours, as the backend's
  `Row.BusinessMetrics` says; "too young" counts the others. **YouTube's**: likes per 1,000 views.
- **A share is the median of the posts'; a rate per 1,000 is the sum of the acts over the sum of the views**, so a rare
  act (a follow) still shows.
- **The posts' list** under each platform: each post, its arm, the age it was read at, its views and signs, and a link.
- **The arrivals**: TikTok's (`/t`) and YouTube's (`/y`) of each UTC day from the day picked, a day with none as 0;
  their table the newest first.

## The charts

Hand-made SVG, as the home page's ([The home page](home.md)), as wide as the panel at every width (`useWidth`), so a
phone at 375 px scrolls none of them. Each platform's numbers are charted first, then its table under **Table**, then
the posts' list; the arrivals' table is under their chart the same way.

- **One color per experiment**, the same on both platforms and in every chart: the `--lab-1` to `--lab-6` tokens of
  `index.css`, given by the sorted names of the experiments read (`colorOf`); the control is the grey `--lab-control`.
  Past the sixth an experiment is drawn in ink, as no seventh hue would read apart. The hues and the grey passed the
  dataviz palette check on the panels' light and dark surfaces. **One legend** at the top of each platform: the
  experiments, the control, the median, a young post and the breakout line.
- **Views**: a row per arm, each experiment's above its control's. Each post is a dot on a log axis of views, ticked at
  100, 1K, 10K…; a post under 100 is drawn at 100. Dots too near are set apart in five lanes. The tick across a row is
  its median, written at the row's end, and the dashed line is the breakout line (`BREAKOUT`).
- **Views by day**: each post a dot at the moment it went out, on the same axis of views, from the day picked to today.
- **A young post** is a hollow dot: one out under 7 days ago, whose reading still changes. Not `ageHours` < 168: with a
  snapshot a day, a post's last row at 7 days or younger is at 6 days and some hours, so every post would stay hollow.
- **A pointer near a dot** (16 px) names its post above the charts: its arm, when it went out, its views and the age
  it was read at, and a link that opens it. A tap or a click keeps it named; a tap off the dots lets it go. The same post
  is ringed in both charts.
- **The signs, as small multiples**: a panel per sign, two across on a phone, four from 720 px. In each, a bar per
  experiment with its control's under it, the value written at its end; a sign with no value ("—") draws no bar, and a
  panel's bars share its own scale. The first panel is the posts, with TikTok's "too young" beside each count. TikTok:
  held at 2 s, to the end, average watch, and comments, shares, follows and saves per 1,000 views. YouTube: likes per
  1,000.
- **Arrivals**: the UTC days, the oldest on the left, TikTok's and YouTube's columns side by side on one scale, ticked
  in whole numbers (`ticksOf` of the home page). Today's columns are hollow, as the day still fills. A pointer on a day
  names its counts in the line above; with none, the line gives the totals since the day picked.

## Tests

`readout.test.ts` pins the reading: the row at 7 days, the posts left out, a row still syncing (`null` numbers), TikTok's business numbers, the medians, the
rates, the breakouts, each experiment's own control, a young post, the arrival days and their totals. `chart.test.ts`
pins the log ticks, the colors, the lanes, the dot under the pointer, the day labels, the arrival bars and the bars'
paths; `format.test.ts` the numbers as written. `LabPage.test.tsx` renders it on `FakeAdmin` (in `StrictMode` once):
the tables, the charts' names, a post tapped, the arrivals, the default days, the arrivals refused with the posts still
shown, and a day picked. `npm run demo` draws it on ten made-up days (`demo/demoLab.ts`): three experiments, a TikTok
and a YouTube breakout, the war report too young for TikTok's signs, and a day with no arrival.
