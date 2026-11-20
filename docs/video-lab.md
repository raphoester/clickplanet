# The video lab: new formats on the game's own accounts

**Designed 2026-11-20, with the owner. PR 1 is built (2026-11-20): the lab's frame in the clips job and its first two
experiments, far invasion and call to arms. PR 2 is built (2026-11-20): the owner's order, and the war report's bench.** The clips had one format, and the experiments of October only tuned it.
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

## The order

**The owner ranked the experiments on 2026-11-20, and they are tested in that order**, two or three at once:

1. **The war report, narrated.** It goes to production only once the owner has compared its scripts and voices and
   signed one off (below, "The war report's bench").
2. **POV, the game itself.**
3. **Far invasion** (built, on).
4. **Rankings.**
5. **Call to arms**, posted at the country's evening (built, off until its turn): the calendar's TikTok hours, 14:00
   to 20:00 UTC, are Europe's evening, not Brazil's or Indonesia's, so its turn needs hours per country.
6. **Satisfying.**

Far invasion runs alone until the war report and POV join it: the coin draws evenly among the experiments on, so it
takes every lab slot meanwhile. Call to arms ran from the merge of PR 1 to the merge of PR 2, the morning of
2026-11-20.

## The formats

- **The war report, narrated: "DAY 9 OF THE INTERNET FIGHTING OVER A WORLD MAP"**: 20 to 30 s, a voice tells one
  story's new chapter, one episode a day of the season, ending on what may come next. The editor chooses the story, as
  for any lab story; a writer of its own writes what the voice says, from that story's facts alone, the tone, and the
  episodes before it, so the series goes on from where it was. The globe plays the story under the voice, the anthem
  low beneath it. 30 s is 900 frames, 28 to 47 minutes at 1.9 to 3.1 s a frame, inside the render job's 60. The voice is
  **Kokoro** (open weights and code under Apache 2.0, its phonemizer too), run in Node with `kokoro-js`, for nothing; a
  paid voice only if every one of Kokoro's sounds made by a machine to the owner.
- **POV, the game itself: "POV: IT'S 3AM AND YOU'RE THE LAST BELGIAN"**: the play screen and not the cinematic globe,
  its bar and counters, a finger tapping, bombs going off. The play page driven by the replay in the recorder, and a
  drawn finger. The biggest build of the list.
- **Far invasion: "WHY IS HUNGARY IN INDONESIA?"** (built): the lead of the best posts, made on purpose. A flag fighting
  on a continent that is not its own; the hook asks the question over the opening, and the headline answers it. The
  editor dropped "the odd pair is the hook" on 2026-11-09 for headlines a stranger could not read ("BULGARIA VS GERMANY
  FOR SAUDI ARABIA"); this brings the pair back as a question anyone reads. Far is the model's to judge, not a check's.
- **Rankings: "THE 5 COUNTRIES THE INTERNET ATTACKED MOST THIS WEEK"**: the globe flies from 5th to 1st, each with its
  rank and the tiles fought over. Built from the map history the job already fetches, a ranking in the findings, and
  the tour's camera with ranks drawn.
- **Call to arms, in the flag's language: "POLACY, GDZIE JESTEŚCIE?"** (built, off): a flag losing a place it held,
  told from its side to its own people: the hook, the headline, the teasers (so the YouTube title) and the call in its
  language, its anthem, the end card asking to defend it. The words in letters Luckiest Guy cannot draw are set in
  Rubik Black; the caption's call to play and its hashtags stay English.
- **Satisfying**: 6 to 7 s, no headline, close on the busiest moment of bombs, a virus spreading or an enclosure
  filling, ending on the payoff so it plays again; the address small in a corner. A camera that holds close, and an
  overlay with no words.

**Not built: AI-made video.** A 2 s AI-made hook, then the real map, is the one form of it that still shows the game.
It is paid by the second of video, more than every other AI call of the project, whose ceiling is $10 a month, and
TikTok labels AI-made content. One experiment of it, once the others have read, with a cap the owner sets.

## The war report's bench

**A voice that sounds written by a machine loses the viewer in a second** (the owner, 2026-11-20): the war report's
words are tuned on real stories, in several tones, and compared by the owner, before any of it is posted.

- **Nothing posts.** A bench run is a workflow started by hand (`narration-bench.yml`): it takes a clips run's
  findings, the editor chooses that moment's best stories as it would for a run, and **the writer writes each in each
  tone, with each model asked**; Kokoro voices each script in each voice asked. The run's artifact holds the scripts,
  what each cost, and the audio.
- **On real moments, with no access to production**: the findings are what each clips run measured, kept 30 days
  from 2026-11-20 (they were kept 1), so a round can be run again on any recent run, the same stories for every
  variant. The sagas start at the world's switch to `earth-2` (2026-11-18).
- **The tones are briefs** (`narration/tones/<name>.md`), as the experiments are, and the first five are: a war
  correspondent on the ground; a sports commentator calling it; a history documentary looking back; a storyteller
  telling a friend; and deadpan, the absurd said flat. **The writer's instructions hold what every tone keeps**: written
  for the ear, short sentences, words a teen knows, numbers as people say them, only what the facts say, and none of
  the turns that give a machine away ("in a stunning turn of events", "little did they know", "buckle up", "it's not
  just X, it's Y", "only time will tell"). The bench counts each script's words, its length spoken, and any such turn
  it still has; it refuses none.
- **The models**: Claude Haiku 5.5, the editor's ($0.10 and $0.50 a million tokens in and out), Sonnet 5.5 ($2 and
  $10) and Opus 5.5 ($4 and $20). A script is a small call, one story and its tone: about $0.05 on Opus, so one episode
  a day is under $2 a month on any of them, inside the $10 ceiling. Which writes like a person is the bench's to show.
- **A round is started by hand**: GitHub's Actions, "narration bench", Run workflow, or Claude starting it. Its inputs:
  the clips run (empty is the latest that succeeded), the stories (3), the tones (every one), the models (opus), the
  voices (`am_adam`) and the season's day (0, none). Its artifact `bench`, kept 30 days, holds `bench.json` (each
  script, its words, its length, the turns that give a machine away, its cost) and an MP3 of each script in each voice.
- **The owner compares on a page**, on a phone as well: each story with its variants in a random order and no label,
  each with its audio and its words, rated 1 to 5 on "sounds like a person" and "I'd watch to the end", the best of
  each story picked, a note for any. The page keeps the ratings; Claude reads them, unblinds them, and says which
  tone, model and voice won and what the notes said. The next round changes one thing.
- **The voice is Kokoro's Adam (`am_adam`) by default**, a male American voice: the owner tried Kokoro's voices by hand
  on 2026-11-20 and Adam sounded the best, whatever its grade in Kokoro's table (F+). Another voice is a bench input.
- **The rounds**: 1. the tones, on Opus, in Adam's voice, 3 stories: 15 scripts, about 6
  minutes of listening, about $1. 2. the best two tones on the three models, in Adam's voice. 3. whole videos of the
  winner on new stories, once the renderer plays a voice. **The owner signs one off**, and only then is it an
  experiment of the lab.

## What it costs

| What | A month |
|---|---|
| Zernio | nothing more: the two accounts there are; Instagram is $6 if taken |
| Rendering | nothing: Actions are free for a public repository |
| The editor | a brief and one story more a run: about $0.05 |
| The voice | nothing: Kokoro, in the render job |
| The war report's bench | about $1 a round of 15 scripts on Opus, while it is tuned |
| The owner | nothing to film; the readout when wanted; a verdict per experiment, every 10 days or so |

## The PRs, in order

1. **The lab's frame, and far invasion and call to arms** (built 2026-11-20): the briefs, the coin, the lab's story
   in the editor's prompt and answer, its checks and its fallback to the control, the hook drawn over the opening,
   `experiment` from the pick to the post's record, `CLIPS_LAB`, `clips edit -experiment`. **The lab is on from its
   merge**: the workflow runs the job and the generator from the default branch, and the server keeps a post's
   metadata as it is given, so nothing is deployed.
2. **The owner's order and the war report's bench**: call to arms off, the findings kept 30 days, the writer, its
   tones, `clips bench`, Kokoro's voices, `narration-bench.yml`. Nothing it makes posts.
3. **The war report's rounds**, run from the bench and compared by the owner, then its renderer (the voice under the
   globe, timed to it) and its experiment, once signed off.
4. **POV.**
5. **The readout**: auth's arrivals counted by source and by day on its `AdminService`, so the backoffice and Claude's
   connector both read them; the backoffice's Lab page, each experiment on each platform against the control of the
   same days: posts, median and best views, breakouts, the signs above, and each day's arrivals by platform. Until it
   is built, the rows of the snapshot hold every number, each with its `post.experiment`, and Claude reads them
   through the connector.
6. **Rankings.**
7. **Call to arms' hours**, its country's evening, then it is on again.
8. **Satisfying.**
9. **Instagram Reels**, if the owner takes it: $6 a month, `instagram` in the publishing calendar and the snapshot, and
   `clickplanet.lol/i` in the arrivals.

## The lab's log

- **far-invasion**: built 2026-11-20, on from the merge of PR 1.
- **call-to-arms**: built 2026-11-20, on from the merge of PR 1, off from the merge of PR 2, for the owner's order:
  its turn is fifth.
