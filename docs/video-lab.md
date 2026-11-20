# The video lab: new formats on the game's own accounts

**Designed 2026-11-20, with the owner. PR 1 is built (2026-11-20): the lab's frame in the clips job and its first two
experiments, far invasion and call to arms.** The clips had one format, and the experiments of October only tuned it.
The lab runs other formats beside it, on the two accounts the game already has, two or three at once, each post marked
with the experiment it belongs to, and keeps a format only when its numbers beat the clips' of the same days. How the
frame works is the backend's [The clips job](../apps/backend/docs/clips-job.md), "The lab", and the hook the
frontend's [Clips](../apps/frontend/docs/clips.md). Each PR still to come turns its part of this doc into "how it works"
and updates the docs it makes stale: [Publishing](../apps/backend/docs/publishing.md), [The posts'
numbers](../apps/backend/docs/post-stats.md), [Arrivals](../apps/backend/docs/arrivals.md) and the backoffice's
[Clip stats](../apps/backoffice/docs/clip-stats.md).

## Why

From the snapshot of 2026-11-19 (4 weeks, 357 posts):

- **Every post was one format**, `globe-replay`, about 10 s: 104 on TikTok, 253 on YouTube.
- **TikTok has a ceiling.** Since the cap of 3 a day (2026-10-28) every post has 600 to 770 views, the best 1,204. That
  is the first audience TikTok tries a video on, and no post went past it. The clips hold 81% of viewers at 1 s, 64% at
  2 s, 52% at 3 s, and 16% watch to the end: TikTok stops there.
- **YouTube has a plateau**: a median of 935, and one post of 30,789 (Hungary in Indonesia, 2026-11-07).
- **What was tried only tuned the format**: its length, its sound, who edits it (the clips job's experiments, Oct 24 to
  Nov 5). Each came out flat. When every tweak is flat, the format is the ceiling.
- **The one lead is in the best posts**: 6 of the 7 best show a flag far from home. Hungary in Indonesia (30,789),
  Palestine in Antarctica (3,345), Bulgaria in South America (3,360), Canada, France and Colombia in Southeast Asia
  (3,807, 3,260, 3,378), and Poland in South Africa, TikTok's best (1,204). "Why is Hungary in Indonesia?" is read by a
  stranger in a second. Seven posts are a lead, not a finding.
- **Players, not views.** A video that does not look like a game one can join right now brings views and no players.
  Every format here shows the game, and gives a reason to come in now.

## What the owner decided

On 2026-11-20:

- **Experiment, at the highest rate that still reads.** Something of the lab goes out every day.
- **Nothing to film.** Every format is made by the pipeline from the game's own data, with no person in it.
- **A hobby, paid by the owner**: each Zernio account past the two free ones is $6 a month. The lab runs on the two
  accounts the game has, for nothing more.
- **TikTok matters most**: "this is where the buzz happens", and most of the audience. YouTube takes the volume.
- **Arrivals are told by platform since 2026-11-20** (#910: `clickplanet.lol/t` and `/y`). Until then an arrival with
  no tag was counted as YouTube's, so no conversion by platform before that day holds.

Then, the same day:

- **The experiments run in parallel, 2 or 3 at once**, not one wave after another.
- **Half of YouTube's posts stay the classic clips**, the control.
- **No baseline week**: the control of the same days is the baseline, and the lab starts as soon as it is built.
- **Views decide.** `/t` and `/y` tell a platform, never an experiment; the views, the hold and the completion of each
  post do. Arrivals by platform are a guard: a platform whose arrivals fall while the lab holds its slots says so.

## Words

- **Format**: what a video looks like, its renderer: `ClipRecord.format`, `globe-replay` today.
- **Brief**: what the editor is asked to tell, and how, for an experiment's story: prose, embedded. Not "angle": the
  code's `Angle` is what a story key says a story tells of its flag.
- **Hook**: big words over a clip's opening, read alone before the headline: a lab story's alone.
- **Experiment**: a brief, and in time a format, under a name (`far-invasion`). Every post it makes carries the name.
- **Control**: the clips as they are, `globe-replay` with the editor's own instructions. Its posts carry `control`.
- **Breakout**: a post past its platform's line at 7 days: **3,000 views on TikTok** (2.5 times the best clip, 4 times
  the ceiling), **10,000 on YouTube** (one clip of 253 did it).

## Where the lab runs: the two accounts there are

- **No second TikTok account.** 94% of the clips' TikTok views come from the For You feed and 3% from followers: 28,472
  views made 17 follows. The account's followers carry almost nothing, so a video is judged on its own whichever
  account posts it. A second account would cost $6 a month and a phone number of its own (TikTok wants one phone
  number or one email per account, and an account made with an email alone draws more checks, by secondhand sources:
  TikTok's own help does not say). It would buy a clean name and nothing measured. **Asked again when a kept format
  deserves an account of its own.**
- **No second YouTube channel**, for the same $6.
- **TikTok's 3 a day stay 3.** At 10 to 13 a day of one format its views fell from about 600 to about 15 (2026-10-19,
  the clips job's "Why so few on TikTok").
- **One pick of each run is the lab's** (built, PR 1): the first 2 times in 3, else the second, by a coin rolled before
  the editor reads anything. TikTok takes a run's first pick alone, so **2 of its 3 posts a day are the lab's**;
  YouTube takes both, so **half of its posts are the control's**. A lab video goes out on both platforms when both
  have room: two readings of one video.
- **YouTube screens, TikTok decides.** With 2 experiments on, each gets about 2 YouTube posts a day and 1 TikTok post:
  20 and 10 in 10 days. YouTube shows early which experiment holds nobody; TikTok's numbers make the verdict, since the
  buzz and the audience are there. A third experiment makes it 13 and 7.
- **Instagram Reels is the one account worth its $6**: a third audience for every video, the clips' too. The owner's
  call, once the first experiments have read.

## How an experiment runs

- **The experiments are briefs of the clips job, embedded** (`posts/experiments/`), each named in `experiments.On`; a
  commit turns one on or off. **`CLIPS_LAB=off`**, a repository variable as `CLIPS_PLATFORMS` is, turns the whole lab
  off with no commit.
- **The coin is the arm**, never a choice made from the story, so the arms compare fair: the design of the length and
  sound experiments of October. The experiment is even among those on.
- **The editor writes the lab's story apart**, to the brief, with its hook and its language; a run with no story the
  brief can tell gives its pick to the control's next story, and the log and the summary say so.
- **Each post records its experiment**: `post.experiment` in its `clips.v1.PostRecord`, `control` for the rest. The
  format stays in `clip.format`, and a format's own measures will go in a section of its own beside `globe`.
- **Rolling, not in waves**: each experiment runs 10 days from the day it is on, then gets its verdict, and another
  takes its place. Two or three are on at any time.
- **The verdict is the owner's**: keep (the format joins the control's mix, at a share the owner sets), again (10 more
  days with one thing changed), or drop (its name moves to `Off`, its brief stays). It is judgement, not a threshold,
  as with the editor. The readout gives the numbers:
  - **A breakout is a keep on sight**, even one.
  - **Else the signs, against the control of the same days**: on TikTok the share held at 2 s (control 64%), the share
    that watched to the end (16%), comments per 1,000 views (1.3), shares and follows per 1,000 (about 0); on YouTube
    the views (median 935) and likes per 1,000 views (7.7).
- **The lab's log**, at the end of this doc, keeps each experiment: when it ran, its numbers, its verdict.

## The formats, in the order they are built

**On today's renderer, with a brief and a hook** (built, PR 1):

- **Far invasion: "WHY IS HUNGARY IN INDONESIA?"** The lead of the best posts, made on purpose. A flag fighting on a
  continent that is not its own; the hook asks the question over the opening, and the headline answers it. The editor
  dropped "the odd pair is the hook" on 2026-11-09 for headlines a stranger could not read ("BULGARIA VS GERMANY FOR
  SAUDI ARABIA"); this brings the pair back as a question anyone reads. Far is the model's to judge, not a check's.
- **Call to arms, in the flag's language: "POLACY, GDZIE JESTEŚCIE?"** A flag losing a place it held, told from its
  side to its own people: the hook, the headline, the teasers (so the YouTube title) and the call in its language, its
  anthem, the end card asking to defend it. National pride argues in the comments, and the viewer has a reason to come
  in now. The words in letters Luckiest Guy cannot draw are set in Rubik Black; the caption's call to play and its
  hashtags stay English.

**New cuts of the same globe:**

- **Satisfying**: 6 to 7 s, no headline, close on the busiest moment of bombs, a virus spreading or an enclosure
  filling, ending on the payoff so it plays again; the address small in a corner. Built from the findings, which count
  each front's bombs, spreads and enclosures: a camera that holds close, and an overlay with no words. The third
  experiment on.
- **Rankings: "THE 5 COUNTRIES THE INTERNET ATTACKED MOST THIS WEEK"**: the globe flies from 5th to 1st, each with its
  rank and the tiles fought over. Built from the map history the job already fetches, a ranking in the findings, and
  the tour's camera with ranks drawn.

**New tools:**

- **The war report, narrated: "DAY 9 OF THE INTERNET FIGHTING OVER A WORLD MAP"**: 20 to 25 s, a voice tells a saga
  like a war correspondent, one episode a day of the season, ending on what may come next. The editor writes the
  script with the saga. The voice is **Kokoro** (open weights under Apache 2.0, runs on a CPU), in English, made in the
  render job, for nothing. Its phonemizer, espeak-ng, is GPLv3: run as a program in the job, never shipped. 25 s is 750
  frames, 24 to 39 minutes at 1.9 to 3.1 s a frame, inside the render job's 60.
- **POV, the game itself: "POV: IT'S 3AM AND YOU'RE THE LAST BELGIAN"**: the play screen and not the cinematic globe,
  its bar and counters, a finger tapping, bombs going off. The play page driven by the replay in the recorder, and a
  drawn finger. The biggest build of the list.

**Not built: AI-made video.** A 2 s AI-made hook, then the real map, is the one form of it that still shows the game.
It is paid by the second of video, more than every other AI call of the project, whose ceiling is $10 a month, and
TikTok labels AI-made content. One experiment of it, once the others have read, with a cap the owner sets.

## What it costs

| What | A month |
|---|---|
| Zernio | nothing more: the two accounts there are; Instagram is $6 if taken |
| Rendering | nothing: Actions are free for a public repository |
| The editor | a brief and one story more a run: about $0.05 |
| The voice | nothing: Kokoro, in the render job |
| The owner | nothing to film; the readout when wanted; a verdict per experiment, every 10 days or so |

## The PRs, in order

1. **The lab's frame, and far invasion and call to arms** (built 2026-11-20): the briefs, the coin, the lab's story
   in the editor's prompt and answer, its checks and its fallback to the control, the hook drawn over the opening,
   `experiment` from the pick to the post's record, `CLIPS_LAB`, `clips edit -experiment`. **The lab is on from its
   merge**: the workflow runs the job and the generator from the default branch, and the server keeps a post's
   metadata as it is given, so nothing is deployed.
2. **The readout**: auth's arrivals counted by source and by day on its `AdminService`, so the backoffice and Claude's
   connector both read them; the backoffice's Lab page, each experiment on each platform against the control of the
   same days: posts, median and best views, breakouts, the signs above, and each day's arrivals by platform. Until it
   is built, the rows of the snapshot hold every number, each with its `post.experiment`, and Claude reads them
   through the connector.
3. **Satisfying**, the third experiment on.
4. **Rankings.**
5. **The war report, narrated.**
6. **POV.**
7. **Instagram Reels**, if the owner takes it: $6 a month, `instagram` in the publishing calendar and the snapshot, and
   `clickplanet.lol/i` in the arrivals.

## The lab's log

- **far-invasion** and **call-to-arms**: built 2026-11-20, on from the merge of PR 1.
