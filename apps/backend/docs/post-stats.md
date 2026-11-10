# The posts' numbers (`internal/publishing/`, in the server)

**A snapshot a day of every post the game made, on each platform**: its numbers as Zernio answered them and what the
post was (its metadata), set side by side, for the experiments of the clips (see [The clips job](clips-job.md),
"Experiments") and for the operator. **The publishing module takes it, in the server** (2026-11-10): until then the
clips job's `clips stats` took it in a workflow and sent it to the clips module over ssh. The server has the key, the
posts and a clock, and a day missed is lost for good, so the snapshot moved where no workflow has to run. The
backoffice reads it and makes the table (its [Clip stats](../../backoffice/docs/clip-stats.md)).

```
internal/publishing/internal/
  stats/                            Post, Target, Answer, Entry, Retention, Row (RowsOf, BusinessMetrics), Report
                                    (ReportOf, TikTokLane), Accounts, DueSince, Backoff, BusyError, SystemPause
    zernio_source/                  the accounts and their TikTok lane, every post, each post's analytics, YouTube's
                                    retention, read from Zernio over shared/cpzernio
    postgres_snapshot_store/        Keep (the summary and every row in one transaction), Latest; SummaryOf
    usecases/take_snapshot_usecase/ the snapshot, and the Runner that takes it once a day
      log_take_snapshot/
  migrations/                       snapshots, snapshot_rows, and the copy of the clips module's
  publishingv1controller/
    list_snapshots_handler/snapshots_query/   list_rows_handler/rows_query/   get_post_links_handler/links_query/
```

## Why daily, and when

On a TikTok account connected through the TikTok for Business app, Zernio's `completionRate`, `impressionSources`,
`audienceTypes`, `audienceCountries`, `retentionCurve`, `follows`, `profileViews`, `websiteClicks` and the watch times
come **24 to 48 hours after a post, and only for posts active in the last 7 days**. A view count read once tells
nothing of a post's first day. A post's age in hours is on every row, so the curve of its first week can be drawn.

**From 08:17 UTC each day** (`stats.DueSince`): just after midnight Pacific in winter, **when YouTube's Analytics
quota starts again**, which every retention curve draws on. A run at 05:41 UTC found it spent, and every curve of Oct
25 and 26 was refused (`429`, "YouTube Analytics daily quota exhausted"). **The runner** (`take_snapshot_usecase.Runner`)
asks every 10 minutes whether the day's snapshot is due and none was taken since, so a restart catches up the same
day; a snapshot that failed is tried again an hour later. It closes the pool after it stops (`cppg.CloseAfter`).

## What a snapshot keeps

- **Which posts**: every post Zernio holds that was scheduled in the last 4 weeks (`DefaultSince`), the clips', the
  battle teasers' and those made by hand, paged 500 at a time: what each is about is in its metadata, beside its
  numbers. A post out on no platform yet has nothing to measure and is counted, not kept; a platform it failed on is
  no row. 4 weeks: TikTok's business metrics stop at 7 days, and a Short's views have about settled by then. Until
  2026-11-10 only the clips' posts were read (those with a story key).
- **One row per platform per post per snapshot** (`stats.Row`): the Zernio post id, the platform, the account, the
  platform's id and address of the video, when it went out there, its age at the snapshot, the post's answer (its
  HTTP status, a 200, a 202 while a platform still syncs, or a 424 when every platform failed; its status, sync
  status and message), **the platform's entry of the answer as it came** (`analytics`, every metric in it, the ones no
  code here knows too), YouTube's retention curve and **the post's metadata as it came** (a clip's story key,
  countries, call and `record`; a teaser's `teaserBattle`). `version` is the shape of the row: a change that is not
  additive is a new version.
- **YouTube's retention** (`GET /v1/analytics/youtube/video-retention`) is asked up to the day of the snapshot, the
  newest curve YouTube has, which it may still revise for 3 days. **Any answer is kept as it came** (`Retention`: its
  status and body), up to 100 points, or the 412 `youtube_analytics_scope_missing` that asks for the channel to be
  connected again. YouTube has no curve for a video with few views or under 2 to 3 days old: an empty one is `empty`.
  **A curve refused as busy past the `Backoff` (1, 5, 15 minutes) stops the curves of the run** (the quota spent): the
  videos after it are not asked, each in `Unread` as `not asked`, and their rows, analytics included, are kept.
- **No analytics, no snapshot.** They need Zernio's Analytics add-on: `GET /v1/accounts` says `hasAnalyticsAccess`,
  and **false fails the snapshot before any post is read, and nothing is written** (`stats.ErrNoAnalyticsAccess`). A
  402 on a post's analytics is the same. Any other post that cannot be read (a 404 for a post deleted, a service busy
  past the `Backoff`) is in the summary's `Unread`, and the others are kept.
- **Is TikTok's business lane live?** For each TikTok account, the summary keeps `tiktokAccountType` (what TikTok said
  when the account was connected: `business`, `personal`, `unknown`), how many of its rows hold a business metric
  (`Row.BusinessMetrics`), and whether `GET /v1/accounts/{id}/tiktok/commercial-music` answers, which it does only
  through TikTok for Business. **Live** is either, whatever the type says: the type goes stale. The first snapshot
  (2026-10-24) found the account still `personal` to Zernio, the music library answering, and business metrics on 80
  of its rows.
- **The summary** (`publishing.v1.StatsSnapshot`, as protojson writes it): when, since when, the counts, each TikTok
  account's lane and the posts lost with why. Counts and states, never a metric. **The log carries the same**.

## Where it is kept, and who reads it

- **Its tables** (`publishing` schema, migration `20261110090000_snapshots`): `snapshots(taken_at, summary)` and
  `snapshot_rows(taken_at, post_id, platform, document)`, both `jsonb`, a row deleted with its snapshot, and the index
  `snapshot_rows_post` for the links. **A snapshot kept again replaces the one kept**, row by row, in one
  transaction. Nothing prunes them: a day is about 2 MB at 4 weeks of posts, compressed by postgres. `jsonb` keeps
  the values and a number as written (`0.310`), not the order of the keys.
- **The snapshots the clips module kept are copied once** by that migration, when the `clips` schema has them: the
  summary is the same message `clips.v1.StatsSnapshot` was, and a row the same document. **The clips tables are left
  as they were**: they are dropped once the copy is checked, by hand, in a later change.
- **`publishing.v1.AdminService/ListSnapshots`** is the summaries of a window, the newest first (100, at most 500).
  **`ListRows`** is the rows of the snapshots taken in `[since, until)`, oldest first, by the primary key, a page at a
  time (500, at most 2000): `next_page` is the key of the page's last row, opaque; a page this query did not give is
  `InvalidArgument`. Both are tools for Claude too. **`publishing.v1.InternalService/GetPostLinks`** is each post's
  newest address on each platform a snapshot read, for the clips module, which links the posts that name a player
  (see [The players a clip names](clip-appearances.md)).
- **Nothing of it is public.** Completion rates, audience countries and profile views are not public numbers: the
  rows live in postgres, behind the console listener and the admin gate, and no workflow, artifact or log of the
  repository holds them.
- **No database host, no snapshot** (`publishing.database`): the two lists are `FailedPrecondition`, `GetPostLinks`
  answers none, no snapshot is taken, and posting still works. No Zernio key, and the whole module is off. See
  [Configuration](configuration.md).

## Tests

`stats`' pin a row's shape and the business metrics, and the day's due time. `zernio_source`'s run against
`httptest` with the answers of Zernio's document: the accounts and the TikTok lane, every post of every page, a post
made by hand with none, each answer kept as it came, the retention, and a busy service. `take_snapshot_usecase`'s run
on a scripted source and a recorded pause: what is kept, the posts lost, the curves stopped once refused as busy, no
analytics; the `Runner`'s, the snapshot taken once it is due and none since, a restart catching up, and a failure
tried again an hour later. `postgres_snapshot_store`'s run on postgres: a row read back with its numbers as written,
a snapshot kept again kept once, the newest, the pages of a window, a page refused, the summaries, the links, and the
clips module's snapshots copied once and left there. The module's test answers the lists `FailedPrecondition` and
no link with no database, while posting still works.
