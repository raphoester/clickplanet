# The video lab: new formats on the game's own accounts

**Designed 2026-11-20, with the owner. Nothing is built yet.** The clips have one format, and the experiments so far
only tuned it. The lab runs other formats beside it, on the two accounts the game already has, a few at a time, each
post marked with the experiment it belongs to, and keeps a format only when its numbers beat the clips'. Each PR at the
end turns its part of this doc into "how it works" and updates the docs it makes stale: the backend's [The clips
job](../apps/backend/docs/clips-job.md), [Publishing](../apps/backend/docs/publishing.md), [The posts'
numbers](../apps/backend/docs/post-stats.md) and [Arrivals](../apps/backend/docs/arrivals.md), the frontend's
[Clips](../apps/frontend/docs/clips.md), and the backoffice's [Clip stats](../apps/backoffice/docs/clip-stats.md).

## Why

From the snapshot of 2026-11-19 (4 weeks, 357 posts):

- **Every post is one format**, `globe-replay`, about 10 s: 104 on TikTok, 253 on YouTube.
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

## What the owner decided (2026-11-20)

- **Experiment, at the highest rate that still reads.** Something of the lab goes out every day.
- **Nothing to film.** Every format is made by the pipeline from the game's own data, with no person in it.
- **A hobby, paid by the owner**: each Zernio account past the two free ones is $6 a month. The lab starts on the two
  accounts the game has, for nothing more.
- **TikTok matters most**: "this is where the buzz happens", and most of the audience. YouTube takes the volume.
- **Arrivals are told by platform since 2026-11-20** (#910: `clickplanet.lol/t` and `/y`). Until then an arrival with
  no tag was counted as YouTube's, so no conversion by platform before that day holds. The owner watches a few days of
  the new tags first.

## Words

- **Format**: what a video looks like, its renderer: `ClipRecord.format`, `globe-replay` today.
- **Angle**: what the editor is asked to tell, and how, over a format.
- **Experiment**: a format and an angle under a name (`far-invasion`), on for a wave. Every post it makes carries the
  name.
- **Control**: the clips as they are, `globe-replay` with the editor's own instructions. Its posts carry `control`.
- **Wave**: the experiments on at once, 2 for 10 days by default.
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
  the clips job's "Why so few on TikTok"). **2 of the 3 go to the lab, 1 to the control.** The control's 700 views a
  post are known; the lab's are what is worth learning. The share is a setting, the owner's to move.
- **About 4 of YouTube's 12 go to the lab.**
- **A lab video goes out on both platforms** when both have room, as the clips do: two readings of one video.
- **YouTube screens, TikTok decides.** A wave gives an experiment about 20 YouTube posts and 10 TikTok posts. YouTube
  shows early which format holds nobody; TikTok's numbers make the verdict, since the buzz and the audience are there.
- **Instagram Reels is the one account worth its $6** (PR 9): a third audience for every video, the clips' too. The
  owner's call, once a wave has run.

## How an experiment runs

- **The experiments are a file of the clips job**, embedded as its instructions are: one entry each, its name, its
  format, its angle (the brief the editor gets, in prose), whether it is on, and its wave's dates. A commit turns one on
  or off. **`CLIPS_LAB=off`**, a repository variable as `CLIPS_PLATFORMS` is, turns the whole lab off with no commit.
- **A coin decides which slot is the lab's, before the editor reads anything**: by the platforms' shares, then evenly
  among the experiments on. It is the design of the length and sound experiments (the clips job's "No experiment runs
  on the clips since 2026-11-05"): the arm is a coin, not a choice made from the story, so the arms compare fair.
- **The editor gets the angle for each pick the coin gave the lab**, beside its instructions. A run with no story the
  angle can tell posts the control's, recorded as `fell-back`, as the `editor` experiment did.
- **Each post records its experiment**: `post.experiment` in its `clips.v1.PostRecord`, `control` for the rest. The
  format stays in `clip.format`, and a format's own measures in a section of its own beside `globe`, as `ClipRecord`
  was made for.
- **A wave is 2 experiments for 10 days**: TikTok's 2 lab slots a day make about 10 posts each, YouTube's about 20.
  Three at once is a third fewer posts each and a wave that reads later; the owner chooses.
- **The verdict is the owner's, at the wave's end**: keep (the format joins the control's mix, at a share the owner
  sets), again (one more wave with one thing changed), or drop. It is judgement, not a threshold, as with the editor.
  The readout gives the numbers:
  - **A breakout is a keep on sight**, even one.
  - **Else the signs, against the control of the same days**: on TikTok the share held at 2 s (control 64%), the share
    that watched to the end (16%), comments per 1,000 views (1.3), shares and follows per 1,000 (about 0); on YouTube
    the views (median 935) and likes per 1,000 views (7.7).
  - **Arrivals confirm, by platform**: the `/t` and `/y` arrivals of each day beside the day's posts. They cannot name
    a post: TikTok's links cannot be clicked and the posts of a day overlap. A breakout shows as a day.
- **The lab's log**, at the end of this doc, keeps each wave: what ran, its numbers, its verdict, as the clips job's
  doc keeps the experiments of October.

## The formats, in waves

**Wave 1, two angles on today's renderer**, the cheapest to build:

- **Far invasion: "WHY IS HUNGARY IN INDONESIA?"** The lead of the best posts, made on purpose. A flag taking ground
  on a continent that is not its own; the first frame asks the question in big type over the dive, and the headline
  answers it. The editor dropped "the odd pair is the hook" on 2026-11-09 for headlines a stranger could not read
  ("BULGARIA VS GERMANY FOR SAUDI ARABIA"); this brings the pair back as a question anyone reads. Built from the
  editor's findings, which know each flag's continent, and the overlay; a check that the attacker's home is on another
  continent than the place.
- **Call to arms, in the flag's language: "POLSKA TRACI NIEMCY. POLACY, GDZIE JESTEŚCIE?"** A flag losing ground, told
  to its own people: the headline, the teasers, the TikTok caption and the YouTube title in the flag's language, its
  anthem, an end card that says to defend it, with `clickplanet.lol/t`, and the country's community hashtag as the
  clips have it. National pride argues in the comments, and the viewer has a reason to come in now. Built from the
  player video's loss story ("…IS LOSING… HELP", the frontend's [Player videos](../apps/frontend/docs/player-videos.md)),
  the anthems, and Rubik Black for the letters Luckiest Guy cannot draw. The editor writes the words in the language
  (`common.language` is in the record already); a check that the flag lost tiles in the place over the stretch.

**Wave 2, new cuts of the same globe:**

- **Satisfying**: 6 to 7 s, no headline, close on the busiest moment of bombs, a virus spreading or an enclosure
  filling, ending on the payoff so it plays again; the address small in a corner. Short clips that end on their best
  frame get watched twice. Built from the findings, which count each front's bombs, spreads and enclosures: a camera
  that holds close, and an overlay with no words.
- **Rankings: "THE 5 COUNTRIES THE INTERNET ATTACKED MOST THIS WEEK"**: the globe flies from 5th to 1st, each with its
  rank and the tiles fought over. Geography viewers argue over a ranking. Built from the map history the job already
  fetches, a ranking in the findings, and the tour's camera with ranks drawn.

**Wave 3, new tools:**

- **The war report, narrated: "DAY 9 OF THE INTERNET FIGHTING OVER A WORLD MAP"**: 20 to 25 s, a voice tells a saga
  like a war correspondent, one episode a day of the season, ending on what may come next. A series gives a reason to
  follow. The editor writes the script with the saga. The voice is **Kokoro** (open weights under Apache 2.0, runs on a
  CPU), in English, made in the render job, for nothing. Its phonemizer, espeak-ng, is GPLv3: run as a program in the
  job, never shipped. 25 s is 750 frames, 24 to 39 minutes at 1.9 to 3.1 s a frame, inside the render job's 60.
- **POV, the game itself: "POV: IT'S 3AM AND YOU'RE THE LAST BELGIAN"**: the play screen and not the cinematic globe,
  its bar and counters, a finger tapping, bombs going off. It shows a game anyone can play from the first frame. The
  play page driven by the replay in the recorder, and a drawn finger. The biggest build of the list.

**Not in a wave: AI-made video.** A 2 s AI-made hook, then the real map, is the one form of it that still shows the
game. It is paid by the second of video, more than every other AI call of the project, whose ceiling is $10 a month,
and TikTok labels AI-made content. One wave of it, once the others have read, with a cap the owner sets.

## What it costs

| What | A month |
|---|---|
| Zernio | nothing more: the two accounts there are; Instagram is $6 if taken (PR 9) |
| Rendering | nothing: Actions are free for a public repository |
| The editor | one call a run, as now; the scripts make answers longer, well under $1 more |
| The voice | nothing: Kokoro, in the render job |
| The owner | nothing to film; the readout when wanted; a verdict every 10 days |

## The PRs, in order

0. **The baseline** (no code, from 2026-11-20): a week of `/t` and `/y` arrivals before any lab post, so the lab's
   days have days to compare with. PRs 1 to 4 are built meanwhile.
1. **The lab's frame** (the clips job and `proto/clips`): the experiments' file, the coin and the platforms' shares,
   the angle handed to the editor, `post.experiment`, `CLIPS_LAB`. It ships with no experiment on, so nothing changes
   on the feeds.
2. **The readout**: auth's arrivals counted by source and by day on its `AdminService`, so the backoffice and Claude's
   connector both read them; the backoffice's Lab page, each experiment on each platform against the control of the
   same days: posts, median and best views, breakouts, the signs above, and each day's arrivals by platform.
3. **Far invasion** (wave 1).
4. **Call to arms** (wave 1). **Wave 1 starts once 1 to 4 are in and the baseline week has run**, about 2026-11-27.
5. **Satisfying** (wave 2).
6. **Rankings** (wave 2). Built while wave 1 runs, so wave 2 starts as wave 1 ends.
7. **The war report, narrated** (wave 3).
8. **POV** (wave 3).
9. **Instagram Reels**, if the owner takes it: $6 a month, `instagram` in the publishing calendar and the snapshot, and
   `clickplanet.lol/i` in the arrivals.

## The lab's log

No wave has run yet.
