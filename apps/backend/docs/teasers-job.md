# Battle teasers on TikTok and YouTube

**Each battle is posted on the game's TikTok and YouTube about a day before it starts** (2026-11-09): its rule set's
cards, the teaser the home page shows, as **a photo carousel on TikTok** and **a slideshow on YouTube**, in a frame
that says when the battle starts. The owner, 2026-11-09: promo "already knows when a battle's gonna happen and what is
the content associated to it", and should post it without either promo or the clips growing.

**The server decides; the job only renders** (the owner, 2026-11-09, on why the decisions were in the job at first:
"rework #723 that way"). The stills need a browser, which the server has not, so a job in GitHub Actions draws them;
everything else is the server's:

- **promo says what is due** (`ListDueTeasers`): the battles that start within 36 hours that no post teases yet.
- **the frontend draws it** (`npm run teaser`, its [Battle teasers, as posts](../../frontend/docs/teasers.md)).
- **promo posts it** (`PostTeaser`): it checks the battle is still due, writes the words, plans the slot and posts it
  through the [publishing module](publishing.md)'s `InternalService`, within what TikTok and YouTube take in a day, with
  the clips.

```
internal/promo/
  module.go                         the server: ListDueTeasers, PostTeaser beside the rest of promo
  job.go                            JobConfig, ListDue, PostTeasers: the job's (cmd/teasers)
  internal/
    teasers/                        the server's: Teaser, Card, Window, TeasedSince, Words, Post (NewPost, Key,
                                    Metadata), Media, Slot, Receipt, BusyError, ErrTaken, RefusedError, Outcome;
                                    Poster, the slot then the post, never a wait
      listed_battles/               ListTeasers' query, in the domain's words
      rpc_publishing/               publishing.v1.InternalService: the posts teased, the slot, the post
      usecases/post_teaser_usecase/ still due? then the Poster
        log_post_teaser/
    promov1controller/
      list_due_teasers_handler/due_teasers_query/   ListTeasers' within 36 hours, less the battles teased
      post_teaser_handler/          the outcome as the wire's oneof
    teaser_job/                     the job's alone, which decides nothing: Due (FolderName), Rendered, Media,
                                    Outcome, Report, BusyError, Backoff, SystemPause
      ssh_backend/                  promo's ListDueTeasers and PostTeaser, publishing's upload links, over ssh
      teaser_folder/                a run's folders: promo's teaser.json for the renderer, its stills read back
      usecases/list_due_usecase/    the battles due, each kept in its folder
      usecases/post_teasers_usecase/ each render uploaded, then handed to promo; busy waited for
cmd/teasers/                        due and post, the flags (CLICKPLANET_SSH), the run's summary
```

## What promo decides

**Which battles are due** (`ListDueTeasers`, `due_teasers_query`): `ListTeasers`' battles within **36 hours**
(`teasers.Ahead`), less every battle a post of the last 60 hours names (`teaserBattle` in its metadata, read with
publishing's `ListPosts`), **whatever became of the post**: a teaser cancelled in the dashboard was cancelled on
purpose. A battle the schedule moved after its teaser went out keeps it: its post says the old time, and a second
would be the spam the calendar is there to stop.

**Whether it still is** (`PostTeaser`, `post_teaser_usecase`): the battles and the posts are read again, as either may
have changed while the job rendered; a battle teased or dropped since is **skipped**. One photo a card and a video, or
`InvalidArgument` (`teasers.NewPost`).

**When it goes out** (`teasers.Poster`, `Teaser.Window`): from **30 hours before its battle** to **an hour before it**, at
the first slot the publishing module's calendar gives (`PlanPost` with `by`: **every platform asked takes it**, so a
teaser asked at 11:00 UTC waits for TikTok's 14:00). **The platforms, in turn**: TikTok and YouTube both; else TikTok's
carousel alone, the teaser's own form; else YouTube's slideshow alone. **None by an hour before the battle is held.**
A slot taken since it was planned is planned again, 3 times at the most. **The server never waits**: a busy posting
service is answered **busy**, with its `Retry-After`, and the job calls again. The calendar's rules are publishing's: an
hour from every other post on each platform, 3 a trailing day on TikTok from 14:00 to 20:00 UTC and 12 on YouTube, the
clips' posts and the teasers' counted together. **A teaser takes one of TikTok's 3** the day before each battle.

**What it says** (`Teaser.Words`), from the battle alone, in English, the time in UTC as the frame says it:

- the post's title, in the dashboard and in `ListPosts`, and **TikTok's title over the photos** (90 characters at most,
  no hashtag: TikTok strips them there): `BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC`, `FINAL BATTLE ON …` for a finale;
- **TikTok's caption** below them: `Battle on Saturday 14 November at 20:00 UTC ⚔️`, the schedule's words when the
  operator wrote some, `Swipe for the rules, then join the battle 👉 https://clickplanet.lol`, and the clips' tags with
  `#battle` (and `#finalbattle`);
- **YouTube's title**, `BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC | CLICKPLANET`, and its description: the same
  first lines, each card as `• headline: line`, `Join the battle 👉 https://clickplanet.lol/?s=yt`, and the tags.

**The post** (`rpc_publishing.Schedule`): one Zernio post with a target on each platform of its slot, the video its
media, **the photos TikTok's own media when TikTok is in the slot**, in the cards' order. Its metadata keeps
`teaserBattle` and `teaserStartsAt`, never published; its idempotency key is the battle and its start (`Post.Key`,
SHA-256), so the same teaser sent twice is one post on Zernio.

**What it answers**: posted (the post, its time, its platforms, each one that refused it), held, skipped, busy (with
`retry_after`) or failed, refused for good. `InvalidArgument` for no battle or media that are not the battle's;
`FailedPrecondition` while publishing is off; `Unavailable` from `ListDueTeasers` when publishing could not say what
was teased. **The connector hides `PostTeaser`** (a public post); `ListDueTeasers` is a tool.

## A run

**Every 6 hours** (`.github/workflows/teasers.yml`, at 43 past 01:00, 07:00, 13:00 and 19:00 UTC, or by hand), three
jobs:

1. **due**: `go run ./cmd/teasers due -out teasers` asks promo's `ListDueTeasers`, writes each teaser's `teaser.json`
   as promo wrote it (`promo.v1.Teaser`) in a folder of its own, and prints the folders as the render's matrix,
   `["entry-7", "season-1_finale"]`, `[]` for none. A folder is named by the battle's id with every mark that is no
   letter, digit, dot, dash or underscore an underscore (`Due.FolderName`: a finale's id is `season-1/finale`); the id
   itself is in its `teaser.json`. No npm in this job: the VPS key is in no job that installs the frontend's
   dependencies.
2. **render**, a job a teaser, side by side: `npm run teaser -- --teaser teasers/<folder>/teaser.json --out
   rendered/<folder>`, the cards as JPEGs of 1080×1920 and a slideshow of them.
3. **post**: `go run ./cmd/teasers post -teasers rendered -summary "$GITHUB_STEP_SUMMARY"` uploads each render's
   photos and video to the posting service's storage (publishing's `CreateMediaUpload`, then one `PUT`; no byte goes
   through the server), each file once a run, then calls promo's `PostTeaser` with their addresses, one after the
   other. **Busy is waited for** (promo's answer, a storage that answers 429 or 5xx, an ssh that fails, or a server
   that answers 5xx): 1, 5 then 15 minutes, or the `Retry-After`, and the call made again. **A refusal is not**: busybox's
   `wget` says `HTTP/1.1 4…`, and the teaser fails at once. A render that failed is not posted: the next run, 6 hours
   later, lists the battle again.

**A run fails when a teaser failed**, or every platform of its post refused it (no connected account); held and
skipped do not. The summary on the run's page says what promo answered of each.

## What it needs

Nothing new: the repository's `VPS_HOST`, `VPS_USER` and `VPS_SSH_KEY` secrets, which the clips' workflows hold, and on
the server the publishing module's Zernio key. The job holds no key of Zernio's. A battle whose rule set has no card is
never teased: promo leaves it out of `ListTeasers`.

## Tests

The server's: `teasers`' pin the window, the posts read back, the words of a battle and a finale and their lengths, a
post's media rules, its key and metadata, and the `Poster`: both platforms by the battle, TikTok alone then YouTube
alone, held, busy answered and never waited for, a slot taken 3 times at the most then busy, a refusal failed, any other
error returned. `rpc_publishing`'s run against a fake `InternalService` on httptest: the battles teased of any state,
the slot asked with `by`, the post's request whole with its photos for TikTok and none for YouTube alone, each answer
as a domain error, an unreachable publishing busy and a refusal passed on with its code. `listed_battles`',
`post_teaser_usecase`'s (still due, teased or dropped skipped, media refused, a busy read), the handlers' (each outcome
on the wire, `InvalidArgument`, `Unavailable`) and `due_teasers_query`'s. Publishing's module test calls its
`InternalService` on the internal listener.

The job's: `teaser_job`'s pin the folder's name of every id, the report and the backoff. `ssh_backend`'s run a fake ssh
that keeps each call: the battles due with their teaser as written, a refusal that says why and is not busy, a failed
ssh or a 5xx busy, a file uploaded once through its link, a storage that refuses or is busy, a teaser handed to promo
with its media in order, and each answer. `teaser_folder`'s write a teaser and read renders back, cards by their number
past 9, and refuse a render missing its video, a card, or its id. The use cases run on fakes, with a busy storage and a
busy promo waited for, and a backoff spent or a refusal failing that teaser alone. `job_test.go` runs `ListDue` and
`PostTeasers` against fakes of promo's and publishing's services on httptest, behind an ssh on the PATH that sends each
call there with curl and says a refusal as busybox's `wget` does: the folders written, a render uploaded and handed to
promo, what promo did not post, and a refusal failing the run. `cmd/teasers`' pin the usage, `[]` when nothing is due,
and the summary.
