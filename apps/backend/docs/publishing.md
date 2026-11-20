# The publishing module (`internal/publishing/`)

**The game's posts on its own TikTok and YouTube: when each goes out, within what each platform takes in a day,
through the one posting service, Zernio.** A module of the server since 2026-11-09, so that one place owns the
calendar everything that posts keeps: the clips job, since the same day, over ssh (its `posts/ssh_publishing`, see [The
clips job](clips-job.md)), and the promo module, which posts each battle's teaser, on the internal listener (its
`teasers/rpc_publishing`, see [Battle teasers on TikTok and YouTube](teasers-job.md)) (the owner, 2026-11-09: "let's
put publishing in the server, definitely"). Two posters that share TikTok's 3 posts a day need one arbiter, and TikTok
punished a pace of 10 a day once (see [The clips job](clips-job.md), "Why so few on TikTok"). **It keeps no table of
posts**: Zernio's post list is the record of what went out and what is to come, posts made by hand included. **It
takes a snapshot a day of every post's numbers**, in the server, and keeps those (see [The posts'
numbers](post-stats.md)).

```
internal/publishing/
  module.go                         Config (zernio, database), NewModule: off with no key, no snapshot with no database
  internal/
    calendar/                       Platform (PlatformsOf), Posting, Feed (With), PostedSince, Calendar (Left, Slot,
                                    SlotBy, Open), Budget, Allowance, Hours, Slot; Default, the game's
    posts/                          Post (NewPost from a Draft, Slot, TakeSnapshot), TikTokWords, YouTubeWords, Media
                                    (FileError), MediaUpload, Scheduled, BusyError
      zernio/                       the one package of the server that knows the service: the upload link, the post,
                                    the feed, the list, over the generated client of shared/cpzernio
      remembered_feed/              the feed, with the posts made here until Zernio lists them
      usecases/create_upload_usecase/  checks the file, asks the service where to upload it
      usecases/schedule_post_usecase/  under one lock: the slot checked against what the service holds, then the post
        log_schedule_post/          each post with its title, time and platforms; never its words or metadata
    stats/                          the snapshot of the posts' numbers (see post-stats.md)
    migrations/                     snapshots, snapshot_rows
    publishingv1controller/         AdminService, the one value mounted as both services; Off (no key), StatsOff (no
                                    database)
      publishingwire/               the wire's platforms and slot
      create_media_upload_handler/  get_room_handler/room_query/  plan_post_handler/plan_query/
      schedule_post_handler/        list_posts_handler/ (the zernio adapter answers it in the wire's words)
      list_snapshots_handler/  list_rows_handler/  get_post_links_handler/   the posts' numbers (see post-stats.md)
```

## The calendar

**`calendar.Default` is the game's** (`calendar.go`), the clips job's budget as it was on 2026-11-09:

- **An hour between two posts on a platform** (`Spacing`), whoever made them: two posts minutes apart split the first
  hour of reach that TikTok and YouTube both judge a Short on, among the same followers.
- **TikTok: 3 in a trailing 24 hours, from 14:00 to 20:00 UTC**, after school across Europe. **YouTube: 12, from
  07:00 to 23:00 UTC.** Why these numbers is the clips job's history: [The clips job](clips-job.md), "2 a run, every 2
  hours, within a daily budget per platform". The hours are UTC, so they move an hour against Europe's clocks in summer.
- **A day is a trailing 24 hours**, as Zernio counts TikTok's cap, not a calendar day.

**`Calendar.Slot(platforms, feed, notBefore, now)` is when a post goes out**: the first whole minute, 5 minutes from
now or later (`leastLead`: the service publishes a time already past at once, inside the request) and from `notBefore`
on, that is a `Spacing` from every post on its platforms, **before it or after it**, on each of them in its hours and
with room in **every trailing day that holds it**: its own, and the one that ends with each post still to come within
a day. A platform left out, for its hours or its room, no longer holds the post back, so the time is found again for
the others; none left is no slot. **A post scheduled ahead is a reservation**: a new post goes before it when the gap
and the days around it leave room, and never takes its room (the clips job's planner took one for a wall until
2026-11-09: [The clips job](clips-job.md), "A post scheduled ahead is a reservation").

**`Calendar.SlotBy(platforms, feed, notBefore, by, now)` leaves no platform out** (2026-11-09, for the battle
teasers): the first whole minute from `notBefore` to `by` at which every platform asked takes the post, by the same
rules, so a post asked at 10:00 UTC on TikTok and YouTube waits for TikTok's 14:00, or for the room of its full day.
None until `by` is no slot. It looks a week ahead at the most, a minute at a time past each clash.

**`Calendar.Open(slot, feed, now)` is whether a slot still holds**: a whole minute, the lead from now or later, and
each of its platforms in its hours, with room, a `Spacing` from every other post. **`Calendar.Left(platform, feed,
now)`** is how many more posts the platform takes now, not counting its hours.

**What counts is Zernio's post list** (`zernio.Service.Posted`), **and the posts this server just made**
(`remembered_feed`): a list read right after a post may not hold it yet, and a post missing from the feed would let the
next one crowd it, so each post `SchedulePost` makes counts until the list shows it, or for a day. Every post since a
day ago and every one still to
come, made by hand or by a job, at each platform's own time (a TikTok target held over the cap carries the time it will
go out). A draft, a post cancelled or failed, and a target that failed or was cancelled count for nothing.

**What a job wants of the platforms is the job's**: it names the platforms of each post. The clips job asks TikTok
for a run's best clip alone; the calendar knows no run.

## The calls

`publishing.v1.AdminService`, on the console listener, which a job calls over ssh as the clips job asks for a replay
(see [Operator tools](admin.md)), and on the admin listener behind the gate. **`publishing.v1.InternalService`** is
`PlanPost`, `SchedulePost` and `ListPosts` again, the same handlers in the same words, on the loopback internal
listener, for another module: promo, which posts each battle's teaser itself (`internal.proto`); and `GetPostLinks`,
each post's address as the snapshots read it, for the clips module. With no key, both are `FailedPrecondition`.

- **`CreateMediaUpload(file_name, content_type, size_bytes)`** answers where to upload a file, and the address a post
  names it by: Zernio's own storage, through `POST /v1/media/presign` with the key. **No byte of a file passes
  through the server**: the job uploads it with one `PUT` to that link, which carries no key and is a credential for an
  hour, so it is never logged. An MP4 of 1 byte to 1 GiB, or a JPEG of 1 byte to 20 MiB, a photo of a TikTok
  carousel, with a name that is no path: `InvalidArgument` for anything else.
- **`GetRoom(platforms)`** is `Calendar.Left` of each, now, from one read of the list.
- **`PlanPost(platforms, not_before, by)`** is the slot `Calendar.Slot` gives, or with `by` the one `Calendar.SlotBy`
  gives, or none. **Nothing is held**: the slot is a read, and the post at it is `SchedulePost`'s.
- **`SchedulePost(slot, media_url, idempotency_key, title, tiktok, youtube, metadata)`** posts the video at that
  slot, under one lock (`schedule_post_usecase`: the server is one process, so a mutex is enough): the list read
  again, the slot checked with `Calendar.Open`, then the post made, so two jobs never both take the last room of a
  day. **What became of it is the answer, not an error**:
  - `scheduled`: the post's id, and each platform that refused it with why (no connected account, or two; Zernio
    kept the post without it). Every platform refused is no id.
  - `busy`: Zernio asked to be tried later (a 429, a 5xx, a 409 `idempotency_conflict` while the try before is in
    flight, or no answer), with its `Retry-After` when it sent one. **The caller waits and plans again**: the
    server never waits, so no call over ssh hangs for the 20 minutes a backoff can take.
  - `taken`: the slot is no longer open (a post took its room or its spacing, or the lead passed). Plan again.
  - `failed`: Zernio refused it for good, with why (a caption it refuses, a wrong key), or kept it as anything but
    `scheduled`: a post that went out at once would be a post nobody reviewed.

  **One post on Zernio, with a target on each platform** (`zernio.Service.Schedule`), so one cancel in the dashboard
  stops it everywhere, each target with its own words: TikTok's caption, public, with comments, duets and stitches;
  YouTube's description, title, public, category Gaming, and **not made for kids**, sent every time (the owner's legal
  choice, 2026-10-17: made for kids turns the comments off for good), with the tags at the root, which Zernio sends to
  YouTube alone. **It posts to the one connected account of each platform**, and refuses to guess between two. **The
  idempotency key is the job's** (the SHA-256 of its video): the same key is the same post on Zernio, whatever the
  slot. **The metadata goes as the job gave it, never published**: what the job reads back from `ListPosts`, and sets
  the post's numbers beside. The words are the job's too: the server writes none, and checks only that each platform
  of the slot has its own (`posts.NewPost`, `InvalidArgument`).

  **On TikTok, photos can stand in for the video** (`tiktok_photo_urls`, 2026-11-09, for the battle teasers): the
  TikTok target is then a carousel of them, in their order, 35 at most, in its own media (`customMedia`), while every
  other platform gets the video. Zernio makes a photo post's text its title, 90 characters at most, its hashtags and
  links stripped, so the title is `TikTokWords.title` and the caption goes below the photos (`description`); public,
  with comments, opening on the first photo, and **with music TikTok adds** (`autoAddMusic`), as a carousel has no
  sound of its own; no duet or stitch, which a carousel does not have. Photos with no title are `InvalidArgument` when
  TikTok is in the slot. JPEG, since TikTok's photo posts take JPEG and WebP and not PNG.
- **`ListPosts(since)`** is every post Zernio holds from the day of `since` on, the latest first, every page of 500
  read (the clips job reads 4 weeks of them, about fifteen a day), by hand or not: its id, title, time, state, why each platform failed it, and its metadata as kept.
- **`ListSnapshots` and `ListRows`** are the daily snapshots of the posts' numbers: see [The posts'
  numbers](post-stats.md).

**Errors**: `InvalidArgument` for a request the domain refuses, `Unavailable` for a read Zernio is busy for, and the
[error net](error-net.md) for the rest. **No key, no posting** (`publishingv1controller.Off`): every call is
`FailedPrecondition`, the boot logs it, and the rest of the server runs. **The connector hides
`CreateMediaUpload` and `SchedulePost`** (see [Claude as a connector](connector.md)): a job's plumbing, and a public
post. `GetRoom`, `PlanPost`, `ListPosts`, `ListSnapshots` and `ListRows` are tools.

**Nothing an error says carries a URL**: an upload link is a credential, and Go's `url.Error` prints the address it
failed on, so `zernio.unreachable` keeps the cause and drops the address. The key goes in the `Authorization` header
of the calls to Zernio's API alone.

## Configuration

`publishing.zernio.apiKey`, `env://ZERNIO_API_KEY` in `deploy/vps/backend.yaml`: the repository's Actions secret,
rendered into the backend's environment by every deploy; since 2026-11-10 no other workflow reads it. Empty, the
module is off. `publishing.zernio.baseURL` is a test's fake; empty is Zernio's own API. `publishing.database`, schema
`publishing`, one connection: the snapshots; no host, no snapshot. See [Configuration](configuration.md).

## Tests

`calendar`'s pin the trailing day, a post still to come counted in the day that ends with it, a post more than a day
ahead counted nowhere today, the lead, `not_before`, the spacing on both sides and the first gap wide enough, the
hours, a platform left out no longer holding the slot back, a post scheduled ahead holding nothing before it, and when
a slot stops being open; and `SlotBy` waiting for every platform's hours and for a full day's room, from `not_before`
with its spacing, none by `by`, and a week at the most. `posts`' pin a post's every refusal at once and the words of
its own platforms alone, and the file rules. `zernio`'s run against `httptest` with the answers of Zernio's document:
the upload link, the request body whole, a platform with no account or two, busy with its wait, a refusal that says
why and is not busy, no address in an error, the feed at each platform's time, and the list with its metadata.
`schedule_post_usecase`'s pin the outcomes, nothing sent for a slot no longer open, and two posts for one slot at once
making one post. The module's test boots it alone against a fake Zernio and calls it on the console listener: an
upload link, the room, a slot, a slot that waits for TikTok's hours with `by` and leaves it out without, a post at a
slot, the same slot taken after, the list with the post's metadata, every refusal of a wrong request, and
`FailedPrecondition` with no key; and on the internal listener, a slot with `by`, a carousel posted and listed back, and
`FailedPrecondition` with no key there too.

The snapshot's tests are in [The posts' numbers](post-stats.md#tests).
