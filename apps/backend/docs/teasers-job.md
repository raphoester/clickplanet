# The teasers job (`cmd/teasers`)

**Each battle is posted on the game's TikTok and YouTube about a day before it starts** (2026-11-09): its rule set's
cards, the teaser the home page shows, as **a photo carousel on TikTok** and **a slideshow on YouTube**, in a frame
that says when the battle starts. The owner, 2026-11-09: promo "already knows when a battle's gonna happen and what
is the content associated to it", and should post it without either promo or the clips growing: **promo says what is
to come** (`ListTeasers`, [Promo](promo.md)), **the frontend draws it** (`npm run teaser`, its [Battle teasers, as
posts](../../frontend/docs/teasers.md)), **the [publishing module](publishing.md) says when and posts it**, within
what TikTok and YouTube take in a day, with the clips. The job is the glue, and owns nothing: no table, no key.

**A job, not the server**, as the clips are: the stills need a browser, which the server has not, and the job runs
where the browser is, in GitHub Actions. **One package, two roots**: the job's code is promo's (`internal/promo/job.go`,
`internal/promo/internal/teasers/`), beside the module the server builds (`module.go`), and shares no process and no
config with it. What it asks of the server goes over ssh to the console listener, as an operator's call.

```
cmd/teasers/                        due and post, the flags (CLICKPLANET_SSH), the run's summary
internal/promo/
  job.go                            JobConfig, ListDue, PostTeasers
  internal/teasers/                 Teaser (Window, FolderName, Words), Due, TeasedSince, Card, Words, Post (Key,
                                    Metadata), Media, Rendered, Slot, Receipt, BusyError, ErrTaken, Report, Outcome;
                                    Sender (the post at its slot, planned again on each wait), Backoff, SystemPause
    ssh_backend/                    promo's ListTeasers and the publishing module, over ssh: the one package that posts
    teaser_folder/                  a run's folders: teaser.json written for the renderer, its stills read back
    usecases/list_due_usecase/      the battles due, each kept in its folder
      log_list_due/
    usecases/post_teasers_usecase/  the battles still due, each rendered one sent
      log_post_teasers/
```

## A run

**Every 6 hours** (`.github/workflows/teasers.yml`, at 43 past 01:00, 07:00, 13:00 and 19:00 UTC, or by hand), three
jobs:

1. **due**: `go run ./cmd/teasers due -out teasers` lists the battles starting within **36 hours** (`teasers.Ahead`,
   promo's `ListTeasers`) that no post teases yet (publishing's `ListPosts`), writes each one's `teaser.json`
   (`promo.v1.Teaser`) in a folder of its own, and prints the folders as the render's matrix, `["entry-7",
   "season-1_finale"]`, `[]` for none. A folder is named by the battle's id with every mark that is no letter, digit,
   dot, dash or underscore an underscore (`Teaser.FolderName`: a finale's id is `season-1/finale`); the id itself is in
   its `teaser.json`. No npm in this job: the VPS key is in no job that installs the frontend's dependencies.
2. **render**, a job a teaser, side by side: `npm run teaser -- --teaser teasers/<folder>/teaser.json --out
   rendered/<folder>`, the cards as JPEGs of 1080×1920 and a slideshow of them.
3. **post**: `go run ./cmd/teasers post -teasers rendered -summary "$GITHUB_STEP_SUMMARY"` **lists the battles again**,
   so one teased or dropped while the cards rendered is skipped, and posts each rendered one still due, one after the
   other, through the publishing module. A render that failed is skipped: the next run, 6 hours later, renders it again.

**When a teaser goes out** (`Teaser.Window`): from **30 hours before its battle** to **an hour before it**, at the
first slot the module's calendar gives (`PlanPost` with `by`, its `Calendar.SlotBy`: **every platform asked takes it**,
so a teaser asked at 11:00 UTC waits for TikTok's 14:00). A run sees each battle first 30 to 36 hours before it, and
the post is scheduled ahead, a reservation the calendar counts from then on: a battle at 20:00 UTC is teased at 14:00
the day before, 30 hours before it; one at 17:00 at 14:00 the day before too, 27 hours before; one at 08:00 at 14:00
the day before, 18 hours before, up to 12 hours after the run that planned it. **The platforms, in turn**
(`teasers.choices`): TikTok and YouTube both; else TikTok's carousel alone, the teaser's own form; else YouTube's
slideshow alone. **None by an hour before the battle is held**: the summary says so, the run stays green, and the next
run asks again. The calendar's rules are the module's, not the job's: an hour from every other post on each platform,
3 a trailing day on TikTok from 14:00 to 20:00 UTC and 12 on YouTube, **the clips' posts and the teasers' counted
together** ([Publishing](publishing.md)). A teaser takes one of TikTok's 3 the day before each battle.

**What each post is** (`ssh_backend.Schedule`): one Zernio post with a target on each platform of its slot. The video
is uploaded every time, the post's media; **the photos only when TikTok is in the slot**, in the cards' order, as
TikTok's own media (`tiktok_photo_urls`). Each file once a run, however many times its post is tried, straight to the
posting service's storage through a link the server signs: no byte of it passes through the server.

**Its words** (`Teaser.Words`), from the battle alone, in English, the time in UTC as the frame says it:

- the post's title, in the dashboard and in `ListPosts`, and **TikTok's title over the photos** (90 characters at most,
  no hashtag: TikTok strips them there): `BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC`, `FINAL BATTLE ON …` for a finale;
- **TikTok's caption** below them: `Battle on Saturday 14 November at 20:00 UTC ⚔️`, the schedule's words when the
  operator wrote some, `Swipe for the rules, then join the battle 👉 https://clickplanet.lol`, and the clips' tags with
  `#battle` (and `#finalbattle`);
- **YouTube's title**, `BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC | CLICKPLANET`, and its description: the same
  first lines, each card as `• headline: line`, `Join the battle 👉 https://clickplanet.lol/?s=yt`, and the tags.

**No battle is teased twice.** Each post keeps `teaserBattle` (the battle's id) and `teaserStartsAt` in its metadata,
never published; a run reads the posts of the last 60 hours (`TeasedSince`, two windows back) and leaves out every
battle one of them names, **whatever became of it**: a teaser cancelled in the dashboard was cancelled on purpose. A
battle the schedule moved after its teaser went out keeps it: its post says the old time, and a second would be the
spam the calendar is there to stop. **The idempotency key is the battle and its start** (`Post.Key`, SHA-256), so a run
made again is the same post on Zernio, whatever its slot, until the list shows it.

**Busy, taken, failed** (`Sender`, as the clips job's): a busy server or service (a 429, a 5xx, an ssh that fails) is
waited for, 1, 5 then 15 minutes, or its `Retry-After`, and the slot planned again after each wait; a slot taken since
it was planned is planned again, 3 times at the most; anything else fails the teaser. **A run fails when a teaser
failed**, or every platform of its post refused it (no connected account); held and skipped do not. The summary on the
run's page says what became of each: scheduled at when, on which platforms, which post, and why a platform refused it.

## What it needs

Nothing new: the repository's `VPS_HOST`, `VPS_USER` and `VPS_SSH_KEY` secrets, which the clips' workflows hold, and on
the server the publishing module's Zernio key. The job holds no key of Zernio's. A battle whose rule set has no card is
never teased: promo leaves it out of `ListTeasers`.

## Tests

`teasers`' pin the window, the battles due, the folder's name of every id (a finale's, a path's, an empty one), the
words of a battle and of a finale on each platform and their lengths, the post's key and metadata, the report and when
it fails, and the `Sender`: both platforms by the battle, TikTok alone then YouTube alone, held, a busy service waited
for and planned again, a backoff spent, a slot taken 3 times at the most, a refusal at once. `ssh_backend`'s run a fake
ssh that keeps each call: the battles listed with their cards, the battles teased of any state, a refused read that
says why, the slot asked with `by`, an ssh that fails as busy, the request of a post whole with its photos and video
uploaded to a fake storage, no photo for YouTube alone, each file uploaded once over two tries, each outcome, and an
upload refused with no address in the error. `teaser_folder`'s write a teaser and read renders back, cards by their
number past 9, and refuse a render missing its video, a card, or its id. The use cases run on fakes: the battles due
kept, none, a list that fails; each rendered battle sent with its media, the teased or dropped skipped, one not
rendered skipped, held and failed. `job_test.go` runs `ListDue` and `PostTeasers` against fakes of promo's and the
publishing module's services on httptest, behind an ssh on the PATH that sends each call there with curl: the folders
written, a teaser posted as a carousel and a slideshow by its battle and skipped on the next run, and TikTok closed
leaving YouTube alone with no photo. `cmd/teasers`' pin the usage, `[]` when nothing is due, and the summary.
