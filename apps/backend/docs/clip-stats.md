# Clip stats (`internal/clips/`, in the server)

**Where the numbers of the clips are kept**: a snapshot a day of each post of the clips job on each platform, which
the job's `clips stats` sends (see [The clips job](clips-job.md), "`stats/` is what became
of the posts") and the backoffice reads and makes a table of (its [Clip stats](../../backoffice/docs/clip-stats.md)). It is in the server
because the data has to be private and kept: postgres is both, and a file in a public repository, its artifacts or
its logs is neither. Chosen 2026-10-24 over a private repository the workflow would push to, which needs a key of its
own and has no page to read it from. **It is the clips package's own module** (`clips.NewModule`), not a module of
its own: its pieces sit beside the job's, `internal/clips/internal/stats/snapshots/` (the domain of what is kept, its
postgres store and its use case), `internal/clipsv1controller/` and `internal/migrations/`. The proto package is the
clips' too: `clips.v1.AdminService`, in `proto/clips/v1/admin.proto`, beside the files the generator writes.

**`clips.v1.AdminService` alone, nothing on the public router.** The job reaches it over ssh on the console
listener, and the backoffice on the admin listener, behind the gate (see [Operator tools](admin.md)).

- **`KeepSnapshot` keeps a part of a snapshot**: its summary (`clips.v1.StatsSnapshot`: when, since when, the counts,
  each TikTok account's lane and the posts lost) and some of its rows, in one transaction. A snapshot is its
  `taken_at`; a row is its snapshot, its post and its platform, and **a row or a summary kept again replaces the one
  kept**, so the parts of one snapshot, or a run made again, are kept once. `snapshots.NewPart` refuses a row of
  another snapshot, a row with no post or no platform, and a document that is not a JSON object:
  `InvalidArgument`.
- **What it keeps is what the job wrote**: the summary as protojson writes it, and each row's `document`, the row
  as the job built it (what Zernio answered as it came, and the post's metadata), both `jsonb`. A metric Zernio adds
  is kept with no change here: the module reads no metric, and the backoffice makes the columns. `jsonb` keeps the
  values and a number as written (`0.310`), not the order of the keys or the spaces.
- **`ListSnapshots`** is the summaries of a window, the newest first (100, at most 500): what the backoffice shows,
  with whether TikTok's business lane is live. **`ListRows`** is the rows of the snapshots taken in `[since, until)`,
  oldest first, by the primary key, a page at a time (500, at most 2000): `next_page` is the key of the page's last
  row, opaque to the caller; a page this query did not give is `InvalidArgument`.
- **Its tables** (`clips` schema, migration `20261025090000_snapshots`): `snapshots(taken_at, summary)` and
  `snapshot_rows(taken_at, post_id, platform, document)`, a row deleted with its snapshot. Nothing prunes them: a
  day is about 2 MB at 4 weeks of posts read daily, compressed by postgres.
- **The pool is a closer**, not `cppg.CloseAfter`: the module runs nothing in the background, and `CloseAfter` of
  no runner closes the pool as soon as the server starts. `module_test.go` boots it and calls it on the console
  listener, which fails if it does.
