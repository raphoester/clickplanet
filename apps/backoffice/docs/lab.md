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
- **The posts' list** under each table: each post, its arm, the age it was read at, its views and signs, and a link.
- **The arrivals**: TikTok's (`/t`) and YouTube's (`/y`) of each UTC day from the day picked, the newest first, a day
  with none as 0.

## Tests

`readout.test.ts` pins the reading: the row at 7 days, the posts left out, TikTok's business numbers, the medians, the
rates, the breakouts, each experiment's own control and the arrival days. `LabPage.test.tsx` renders it on
`FakeAdmin` (in `StrictMode` once): the tables, the arrivals, the default days, the arrivals refused with the posts
still shown, and a day picked. `npm run demo` draws it on ten made-up days (`demo/demoLab.ts`).
