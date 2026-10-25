# The clips job (`internal/clips/`, `cmd/clips`)

**Every 2 hours a workflow makes the best clips of the last 12 hours and posts them on TikTok and YouTube Shorts**
(`.github/workflows/clips.yml`; the frontend's [Clips](../../frontend/docs/clips.md) has the four jobs and how a clip is made). The
frontend's generator needs a browser, so it stays there. **Everything else is here**: the fetch of the replay,
which clips, when each goes out, the client of the posting service, the retries and the run's summary. The two
meet in files: the replay, the race and the season this job saves for the generator, and `clips.v1.Shortlist` and
`clips.v1.Clip` (`proto/clips/v1/clips.proto`), which the generator writes as JSON.

```
clips fetch -out <replay.json> [-race <race.json>] [-season <season.json>] [-ssh <user@host>] [-hours 72] [-until <2026-10-12T12:00:00Z>]
clips choose -shortlist <shortlist.json> [-count 2] [-platforms tiktok,youtube] [-summary <file>]
clips schedule -clips <folder> [-review 0] [-spacing 1h] [-platforms tiktok,youtube] [-summary <file>]
clips edit -shortlist <shortlist.json> [-count 2] [-prompt]
clips stats [-ssh <user@host>] [-since 2026-10-13]
```

- **`fetch` saves a replay**: it asks the production backend's operator tool for one
  ([`GetReplay`](admin.md#getreplay)), over SSH, and writes the answer as it came. `CLICKPLANET_SSH` stands in for `-ssh`.
  With no `-hours` it is 72: the ledger keeps that much in memory, and a replay cannot reach further back.
  **With `-race` it saves the season's race too** ([`GetRace`](seasons.md)), from which the generator
  tells the day of the season closed in the replay, "FRANCE WINS DAY 5". **A race the backend does not answer is a
  warning, not a failure**: the run still has its replay, and tells no day. No race file is written then.
  **With `-season` it saves the season's calendar** ([`GetSeason`](seasons.md)) the same way: the
  generator calls each country that can still win up to the Final Battle, in its own language, "POLSKA: 14 PUNKTÓW
  STRATY". A season left out calls nobody up.
  **It is cheap enough to run 12 times a day**: on 2026-10-24 the 12 hours of the noon run were 40,820 events, the
  three files 0.8 MB zipped, and the whole step, ssh, `go run`'s build and the race and the season included, 17s. The
  ledger answers from memory. 12 hours and not 2: a story is found by the first run that sees it and skipped as told
  by the next ones, and the generator searches its last 12 hours anyway (`newsSince`).
- **`choose` prints the picks to render**, as a JSON array on the standard output, each with the arm of every
  experiment its clip is in (`[{"pick":1,"variants":{"length":"short"}}]`, or `[]` on a quiet run or a full day): the
  workflow makes one render job of each. The log goes to the standard error, with each story skipped as told and each
  one chosen with its arms. A pick is a story's place in the shortlist, from 1 (`posts.Pick`): what the generator's
  `--pick` takes, and the arms are what its `--variants` takes (below, "Experiments"). It needs `ZERNIO_API_KEY`, to
  read what was told and what each platform holds. `-platforms` is `schedule`'s, so it counts the room of the
  platforms the clips will go to. **On a run the experiment `editor` draws for the model** (below, "The editor") a
  pick also carries the model's headline (`"headline":"BRAZIL STORMS ARGENTINA"`, left out for the generator's own),
  which the render job hands the generator as `--headline`; `ANTHROPIC_API_KEY` is the model's key, and `-summary`
  adds who chose the run, and what the call cost, to that file (the workflow's step summary). Its standard output
  stays the picks alone.
- **`edit` tries the editor by hand on a saved shortlist**, outside any run: no coin, `-count` clips (2), what was
  told read from Zernio when `ZERNIO_API_KEY` is set and nothing told otherwise. It prints the instructions and the
  run's JSON as the model would get them, their size (counted by the API's free `count_tokens` with a key, else
  guessed at 4 characters a token), then what the model chose and what it cost, or why it fell back. **`-prompt`
  stops after the prompt and asks the model nothing.**
- **`schedule` sends every clip of the folder** (`clip-<n>.json` with its `clip-<n>.mp4` beside it) to the posting
  service, **as one post on every platform that has room left in its day** (`-platforms`, default all:
  `tiktok,youtube`), and adds a table to the summary: a row for each clip on each platform, its arms, its time, and
  whether it is scheduled there, or one row for a clip held because every platform has posted its day. **A clip in an
  experiment that does not run, or in an arm it has not, fails the run before anything is posted**
  (`posts.ErrUnknownVariant`): once out, it would be counted in the wrong arm. It exits 1 when a
  platform did not take a clip, when the folder holds none, or when the summary cannot be written. **Platforms are
  independent**: a platform that refuses does not stop the others, and a clip refused does not stop the next one.
  The workflow passes the repository variable `CLIPS_PLATFORMS` (Settings → Variables), so `tiktok` alone turns
  YouTube off with no commit; unset is every platform.
- **`stats` keeps a snapshot of the numbers of every post of the job**, once a day in a workflow of its own
  (`.github/workflows/clips-stats.yml`), so the experiments can be read from them: a row for each platform of each
  post, with what the post was beside it, **sent over ssh to the server's clips module**, which keeps it in
  postgres, and the backoffice makes the table (below, "`stats/` is what became of the posts"). `CLICKPLANET_SSH`
  stands in for `-ssh`, as for `fetch`. It exits 1 when a post could not be read whole, after keeping the others.
- **`cmd/clips` is three stages, and `run` only names them**: parse, work, display (see [A command's `run` is a
  table of contents](command-run.md)). `main.go` holds no stage.
  - **`flags.Parse`** (`cmd/clips/internal/flags`) answers one typed `flags.Config`: the command, its files, and
    the job's `clips.Config` with `CLICKPLANET_SSH`, `ZERNIO_API_KEY` and `ANTHROPIC_API_KEY` read into it, and `-platforms` checked by
    the domain (`clips.PlatformsOf`: each known, each once). No flag pointer leaves it.
    Flags and not `cpconfigs`: the job has a few settings and two secrets, and a workflow passes them on the
    command line.
  - **The work returns values and prints nothing**: `clips.Fetch`, `clips.Choose` (the picks, each with its arms,
    and who chose), `clips.Edit` (the editor's trial), `clips.Schedule` (the `Report`) and `clips.Stats` (the
    `StatsReport`), in `internal/clips/job.go`, which wires the objects as a `module.go` does.
    A clip or a platform the service refused is in the report, not in the error: the error is for a run that
    could not be made at all.
  - **The display owns the output and says how it went** (`cmd/clips/internal/display`): `display.Picks`
    writes the array (a clip in no experiment has `"variants":{}`, never `null`), `display.Editorial` adds who chose
    the run to the summary, `display.Trial` prints `edit`'s, `display.Report` writes the table, to the summary file when one is named, and answers
    whether every clip is scheduled on each of its platforms. It does not turn that into an error: `schedule` in `main.go` decides, and
    exits 1 on `errUnsent`. **`stats` has no display**: the backoffice shows the snapshots (its [Clip
    stats](../../backoffice/docs/clip-stats.md)), and the log says what was kept and lost (`log_take_snapshot`). `snapshot` in `main.go` exits 1 on
    `errUnread` when a post was not read whole.
  - **The dispatch acts on `fetch`, `choose`, `schedule`, `edit` and `stats`, and on nothing else**: any other command is
    `errUnknownCommand`, and nothing is fetched, printed, posted or kept.

**Experiments run on the clips, to learn what makes one do well, and the job alone decides who is in which arm**
(`internal/clips/experiments.go`, `posts/experiment.go`). The posts' numbers are set beside what each clip was (its
`record`, below), and an experiment is what makes that comparison fair: the arm is a coin, not a choice made from the
story, so a short clip is not short because its story was small.

- **`running` in `experiments.go` is the one list of the experiments that run.** Each has a name, its arms with their
  weights (`posts.EvenArms` for an even split, or `posts.Arm{Name, Weight}`: 3 to 1 is three clips in four), and its
  control, **the arm that is exactly what a clip was before the experiment**. `posts.ExperimentsOf` refuses a list
  with no name or one name twice, an experiment of fewer than two arms, an arm with no name, twice or with no weight,
  or a control that is no arm; `choose` and `schedule` both fail then. **Stopping an experiment is taking its entry
  out, and nothing else**: the generator renders as before with no arm, and a clip in it rendered by that run's
  commit is still checked by that commit.
- **`choose` flips one coin per experiment for each story chosen** (`Experiments.AssignEach`, with
  `posts.SystemDice`, in the order of the list), after the stories are chosen and apart from them: an arm never
  changes which story is told. A coin of its own per experiment makes two experiments at once a factorial design,
  every pair of arms in about the share of their weights, so each one's effect is read across the other's.
- **The one exception is an experiment on which stories are told** (`PerRun`): its coin is the run's, flipped once
  before anything is chosen (`Experiments.AssignRun`, first of the dice), and every clip of the run carries its arm
  beside its own. **Deliberately**: `editor` exists to change the choice, so a coin flipped after it would have
  nothing left to decide, and a coin per clip would let one run mix two editors' stories, each choice leaning on the
  other (a flag's share of the feed counts across the run). The arm is still a coin and not something read from the
  stories, so the runs compare: what differs between them is who chose. The run's clips share their run's news, so
  the analysis reads it per run as well as per clip.
- **An arm of weight 0 is never drawn** (`posts.Arm{Name, Weight: 0}`): it is where a clip goes when the arm it was
  drawn for could not be applied, so it is still checked and counted, apart (`editor`'s `fell-back`). An experiment
  needs two arms a coin draws.
- **The arms travel with the clip, as data**: `choose` prints them with the pick; the workflow's render job of the
  pick hands them to the generator (`--variants '{"length":"short"}'`), which applies the ones it acts on, refuses
  one it does not know, and writes them all into the clip's `ClipRecord.variants` (the frontend's
  [Clips](../../frontend/docs/clips.md)); `schedule` reads them back from `clip-<n>.json` (`posts.ClipSnapshot.Variants`), checks every clip
  against `running` before it posts any, carries them on each post (`posts.PostSnapshot.Variants`) and writes them in
  the summary. The record, and so the post's Zernio metadata, keeps them as the generator wrote them: what the
  analysis reads.
- **An arm acts where its difference is made.** At render time, in the generator (the length below); at post time,
  in the job: an adapter reads `PostSnapshot.Variants` and changes what it sends, for the platform the experiment is
  about. **None acts at post time yet**, and a test pins that a clip in any arm sends Zernio the same body as a clip
  in none, its record aside. The generator still lists an experiment it has nothing to apply for, so it refuses an
  arm nobody knows on every experiment, and carries the arms of the rest to `schedule`.
- **Adding one** is an entry in `running`, its name and arms in the generator's `src/clip/experiments.ts`, and the
  code that acts on its arms, never on its control. For the next one, TikTok's sound (the anthem against a trending
  track of TikTok's Commercial Music Library): `{Name: "sound", Arms: posts.EvenArms("anthem", "trending"), Control:
  "anthem"}`, `sound: ["anthem", "trending"]` in the generator, which renders the anthem in both, and in
  `posts/zernio` the TikTok target of a `trending` post gets `musicSoundInfo` and `videoOriginalSoundVolume: 0`
  (the operation's fields in the generated client: `oapi-codegen.yaml` may need them named). YouTube's target is
  left as it is: the arm is recorded on the whole post, and the analysis reads it per platform.
- **The first is `length`** (started 2026-10-24): `as-chosen`, the control, as long as the camera has somewhere to
  go (8.5 to 17.5s for a front, by how far its fighting moves, 10 to 19 with a day's podium, 9 to 20.5 for a tour, by
  its stops), against `short`, the shortest a story of its kind plays: 8.5s for a front, 10 with a day's podium. **A
  tour is already its shortest**, each stop its flight and its stay, so `short` leaves it as it is, and a front whose
  fighting stays in one place is already 8.5s: both still record the arm, and `durationSeconds` in the record keeps
  the real length, so the analysis compares on the length a clip had as well as on its arm. The teaser rolled when
  posting is an experiment of its own already, at random and recorded (`teaser`), and the arms do not touch it.
- **The second is `editor`** (started 2026-10-24, per run, even): `rules`, the control, is the run as before;
  `model` is Claude Haiku choosing among the stories the rules let through and writing their headlines (below);
  `fell-back` is a run drawn for the model that the rules made. Intent to treat is `model` and `fell-back` together;
  what the model's choices did is `model` alone. A run drawn for the model with nothing to decide (a quiet run, a full
  day, or a day won filling it) asks nothing and stays `model`: it is what the model's arm does then.

**The editor: a model chooses the run's stories and writes their headlines, within the rules** (`posts/editor.go`,
`posts/claude_editor/`, 2026-10-24). Step 1 of a model directing the clips: it picks and words, the renderer and its
camera are as they were, and the rules stay the baseline to beat.

- **The rules decide what may be told; the model only ranks it.** `Shortlist.Offer` is what it is offered: the
  stories `Best` could tell (no key of the last 3 days, no headline of the last week, a flag in fewer than 2 of the
  last 6 posts), the days won apart, told first whatever it says, how many clips the run makes (the budgets' count,
  as before), the posts of the last 3 days, and how many of the last 6 showed each flag. **`Shortlist.Edited` walks
  the run as `Best` does**, its next story the model's next one instead of the freshest flag: a day won first, then
  the model's order, each story still skipped when an earlier one of the run filled its flag's share, then the
  stories it left out, in the rules' order. So a run makes as many clips as the rules would, and never one they would
  not tell. The angle-once and freshest-flag orders are the rules' taste, which the model replaces. **The run's two
  clips still go out in the shortlist's order** (`Ranked`, by pick), whichever the model ranked first: the order of
  posting is left as it was, and each post's `runPosition` records it.
- **One call a run, and only when there is something to decide** (`Offer.Open`: room left after the days won, and a
  story offered). A quiet run asks nothing.
- **Its answer has to hold, or the run is the rules'**: every pick one offered, once; every headline upper case
  once written (`headlineFrom`), 60 letters at most, starting with a letter or a digit (so `--headline` never reads
  as a flag), only letters of any script, digits and `! ? ' ’ % : , . & -`; not a headline of the last week, not
  another story's, not one of the story's own teasers (the caption never repeats the clip). **No key, an API error,
  a refusal, an answer cut short or not JSON, or one that does not hold all fall back to the rules**
  (`choose_clips_usecase.Edit`): the run is never failed for it, its clips are recorded `editor: fell-back`, and the
  reason goes to the log and the summary.
- **What it is sent** (`claude_editor.PromptOf`): the instructions, then the run's offer as compact JSON: `now`,
  `slots`, `toldFirst`, each story with its pick, headline, key, countries, teasers and `facts`
  (`clips.v1.StoryFacts`, which the generator's `--shortlist` now writes; the frontend's [Clips](../../frontend/docs/clips.md)),
  `recentPosts` and `shownLately`. **The instructions are `claude_editor/instructions.md`, embedded**: prose on what
  makes a story worth telling and a headline good, from the lessons of this section and of the frontend's. **Tuning
  that file is how the model's taste is shaped**; a change ships with the next commit, and the record's `generator`
  names the commit, so the analysis can tell the versions apart.
- **Claude Haiku 5.5** (`claude-haiku-5-5`), through Anthropic's Go SDK: adaptive thinking at effort `medium`,
  `max_tokens` 8,000, the answer held to a JSON schema by structured outputs (`why`, then `stories` of `pick` and
  `headline`), 2 minutes and one retry. **A refusal is the end of it**: Haiku has no fallback model on the server,
  so `stop_reason: "refusal"` (with its category) falls back to the rules like any other failure.
- **What it costs, said every run**: the call's tokens (in, written to the cache, read from it, out) and their price
  at $0.10 and $0.50 a million in and out (5 times both past 100K tokens of prompt, a cache write 1.25 times the
  input, a read 0.1 times) go to the log and the summary (`posts.Spend`, `claude_editor.SpendOf`). Measured on the
  run of 2026-10-24 18:28 (its 25 stories and 71 posts, from its log): 8 stories offered, 32 recent posts, about
  1,700 tokens of instructions and 4,300 of run, so about 6,000 in, guessed from characters with no key at hand;
  about $0.0006 in and, with its thinking, about $0.001 out a call. A run a model's in two, 12 runs a day, is about
  $0.30 a month; every run would be $0.60. The owner's ceiling for all AI calls is $10 a month.
- **The instructions are marked for caching** (`cache_control` on the system prompt), as the owner asked. **No run
  reads it back today**: a cache entry lives 5 minutes (an hour at most) and runs are 2 hours apart, so each call
  writes it again, about $0.00004 more a call. It pays once a call follows another within minutes (`clips edit`
  tried twice, or more calls a run).
- **The key is a secret of the `choose` job alone** (`ANTHROPIC_API_KEY`), which runs no npm, as `ZERNIO_API_KEY`.
  An error never carries the request (`cpanthropic.Redacted`): the SDK's error dumps its headers, and the logs of a
  public repo are public. **The headline is the model's text**: the render job passes it through the environment,
  never into the script.
- **The client is built in `job.go` and handed to the editor** (`editorOf`), as `job.go` builds every other
  dependency: `cpanthropic.NewClient` makes it from `Config.Anthropic`, and `claude_editor.New(client, model)` never
  makes one of its own. The editor keeps only what is its own on each request: 2 minutes and one retry. **No key is
  an editor of its own, chosen there**: `claude_editor.Unkeyed` sends nothing and answers `cpanthropic.ErrNoKey`, so
  every run drawn for the model falls back and says why.
- **Tests never call the API**: the use case runs a fake `Editor`, and `claude_editor` a client pointed at a fake
  Messages API on `httptest` (`cpanthropic.Config.BaseURL`), which pins the request whole: the cached instructions,
  the schema, the prompt.

**`internal/clips/internal/replays/` is the fetch**, a concept of its own beside `posts`:

- **`Request.Window(now)`** is which stretch: `Span` back from `Until`, or from now when no end is asked for. A
  span of nothing or less is refused before anything is asked of the backend.
- **`ssh_replay_source` runs the `ssh` program**, as the operator does by hand (`deploy/vps/README.md`, "Get a
  replay for a clip"): `ssh -- <host> "cd … && docker compose exec -T backend wget … GetReplay"`. Not an SSH
  library: the program brings the agent, the config and the known hosts of whoever runs it, a laptop or the
  workflow. **`--` ends ssh's options**, so a host that starts with a dash is a host and never
  `-oProxyCommand=…`. The body is two times the adapter formats itself, so nothing a shell reads goes between
  its single quotes.
- **What comes back is checked before it is kept**: it has to read as a `planet.v1.GetReplayResponse` with its
  window, unknown fields dropped. A 502 page or an empty answer fails the fetch here, not three jobs later in a
  browser. The file holds the answer byte for byte, never a re-encoding: the generator reads the backend's JSON.
- **An ssh that fails says what it complained of** (`Permission denied (publickey)`), from its standard error.
  In the workflow the host is a secret, which Actions masks in the log.
- **The race is asked the same way, of the public listener** (`127.0.0.1:8080` in the container, `{}` for a body),
  and checked the same: it has to read as a `seasons.v1.GetRaceResponse`. A backend from before `GetRace` answers
  404, which `wget` turns into a failed ssh. `fetch_race_usecase` keeps it, and `log_fetch_race` logs the day closed
  last, or warns that the race is left out: `clips.Fetch` goes on without it. The season is asked the same way, as
  a `seasons.v1.GetSeasonResponse`, through `fetch_season_usecase` and `log_fetch_season`.
- Its test runs a fake `ssh`, a shell script that keeps its arguments. No test runs the real one.

**`internal/clips/internal/posts/` is the domain of the rest**, with no file, no HTTP and no clock of its own:

- **`Shortlist.Best(count, told, now)`** is which clips: `count` stories, all of them when there are fewer, 2 a
  run at most (`Config.PerRun`) and fewer when the platforms' day is nearly full (`Budget.Clips`, below), **never a
  story told already, and never mostly one flag**. Four rules, from the week of
  Oct 13–17, when 18 clips told "ALGERIA STRIKES BACK" three times, "PALAU IS TAKING OVER THE WORLD" twice right
  after the old one-day window, and Palau (two players on five accounts) in about half of them:
  - **A key told in the last 3 days is skipped** (`retelling`): a `StoryKey` (a flag, a continent and an angle,
    from the generator) a post holds that goes out within 72 hours or later, or that a story of the same run took.
    The keys and not the headlines: "ALGERIA STRIKES BACK" and "ALGERIA VS FRANCE" are one story. **The angle is
    what the story tells of its flag there** (`Angle`): what it did (`Deeds`, empty), how much of the place it holds
    (`Holding`, "CROATIA HOLDS 80% OF EUROPE"), or a day of the season it won (`DayWon`, "FRANCE WINS DAY 5", whose
    key names the day too, `Round`, "1.5": France winning two days is two stories), or the country called up to the
    Final Battle after a day (`CallUp`, "POLSKA: 14 PUNKTÓW STRATY", keyed by the day it follows, so a country is called
    up once a day at most). So a holding is not the story of what its flag did there. A key of a day with no day, a
    day on another angle, or an angle this job does not know is refused.
  - **A headline told in the last week is skipped** (`reheadlining`), whatever its continent: the same words read
    as the same clip. Compared in upper case, spaces at the ends cut.
  - **A flag the last 6 posts showed goes after the flags they did not** (`lately`, about 10 hours at 14 posts a day). **A
    post shows its key's flag and every country of its story** (`Countries`, as the generator writes them: the
    place, both sides, the flags it takes from): "FRANCE VS PALAU" and "VATICAN CITY VS PALAU" both show Palau. A
    story ranks by its key's flag, then by the country of it the last posts showed most, then by the generator's
    order. A story chosen counts as the newest post. A post made before the posts kept their countries shows its
    key's flag only.
  - **A flag that 2 of the last 6 posts showed waits** (`mostLately`): it is a third of the feed at most. **Both stay
    counts of posts when the job posts more often** (2026-10-24): the feed a viewer scrolls is posts, not hours, so a
    third of the last 6 is still a third of what they see. At 6 a run twice a day they covered about a day; at up to 14
    a day they cover about 10 hours, and a flag waits that much less. So a
    shortlist of one flag gives 2 clips in any 6 posts. Ranking alone was not enough at 6 a run: on Oct 15 a shortlist of
    Palau and one other flag gave 5 clips of Palau. **Only the key's flag waits, not each country of the story**:
    Palau and France defend in nearly every story of Oct 15–17, and waiting on all of them left 6 clips in those
    2.5 days, none on the night of Oct 16.

  And two from Oct 18, when the noon run skipped 8 of its 13 stories as told, nearly all one flag in Europe
  ("SWEDEN IS INVADING UKRAINE", "… LITHUANIA", "… PORTUGAL" are one key), and posted 3, and every headline told what
  a flag did:
  - **A day of the season won goes first** (`DayWon`), whatever the last posts showed: it is the day's news, and
    the flag that won a day is in most of the day's clips. It is still skipped when it was told.
  - **A run tells each angle it has once before it tells any twice**, the freshest flag first within it: so a run
    holds a holding when the shortlist has one, beside what the flags did.

  It answers a `Choice`: the stories chosen, in the shortlist's order, the ones skipped (`Retold`), and the ones not
  told whose flag 2 of the last posts showed, or that a story after them was chosen over, for a flag told less
  lately or an angle the run had not told (`Passed`), which the log shows. A story with no key is refused when the
  shortlist is read. With a day won and three holdings added to it by hand, the shortlist of Oct 18 gives 6 (a test of `Best`); what a real run gives is for its log to say.
- **2 a run, every 2 hours, within a daily budget per platform** (2026-10-24, from 6 a run twice a day). A story's
  stretch ended 10 to 20 hours before its post then: up to 12 hours waiting for the run, 4 for the review, and up to
  10 more for the sixth clip's turn. Now a story waits for the next run (an hour on average, plus GitHub starting a
  scheduled run 20 to 40 minutes late, as the runs of Oct 22–24 did), its render (10 to 35 minutes) and 5 minutes of
  lead: **2 to 3 hours**, and an hour more for the second clip of a run. Replayed over Oct 14–17, a 12-hour window held 5
  to 11 stories of different keys, so 2 every 2 hours is not short of stories; a quiet run posts nothing.
  - **`Budget` is posts in a trailing 24 hours, per platform: 14 on TikTok, 8 on YouTube** (`dailyBudget` in
    `job.go`). TikTok holds a video over 15 a trailing day on an account of its Business app, which every new Zernio
    connection is, and Zernio does not refuse it: it moves the post to when the oldest leaves the window, which
    would quietly undo the speed. 14 leaves room for one post made by hand (it shipped at 10 on 2026-10-24, and was
    raised the same day: one by hand is all the room needed). YouTube lets a channel upload about 10 a day.
    So a day of news is 8 clips on both and 6 more on TikTok alone, the rest held; 12 runs of 2 would be 24.
  - **What counts is Zernio's post list** (`Service.Posted`, a `Feed` of `Posting`s): every post on the platform in
    the trailing day, **made by hand or not**, and every one still to go out, at the platform's own time (a TikTok
    target held over the cap carries the time it will go out). A draft, a post cancelled or failed, and a target that
    failed or was unpublished count for nothing. One list of posts serves this and what was told. **A feed that
    cannot be read fails the run**, rather than post over the cap.
  - **Known effect, not handled**: the budget is spent as the news comes, so a busy morning can fill the day, and once
    it is full a slot frees when yesterday's post leaves the window, so the posting hours follow yesterday's. The run
    that gets the slot still tells the freshest story it has.
  - **`choose` makes as many clips as the platform with the most room takes** (`Budget.Clips`), so a clip goes to
    every platform that still has room, and a clip nobody has room for is never rendered. `schedule` reads the feed
    again and leaves a full platform out of the post; a clip with no platform left is held (`Report.WithHeld`): in the
    report, not a failure, and its story untold for a later run. Between the two only a post made by hand can fill a day.
  - **A full day says so**: `choose` logs `every platform has posted its day already` rather than that no story was
    worth a clip (`Choice.Room`).
- **What was told is Zernio's post list, not a table of ours**: each post keeps its key and its story's countries in
  its Zernio `metadata` (`storyFlag`, `storyContinent`, `storyCountries`, and `storyAngle` and `storyRound` when it
  has them, never published; a post with none is of what a flag did), and `choose` lists
  the posts of the last week back (`ToldSince`), on every platform: one Zernio post is one clip, whatever accounts it goes to, and the key is on the post
  (`listPosts`, `fromDate` is a date, so the domain filters to the hour). So a clip cancelled in the dashboard still
  counts as told, nothing can disagree with what Zernio holds, and a post made by hand, with no key, counts for
  nothing. **A history that cannot be read fails the run**, rather than tell a story twice.
- **Each post also keeps what it was, to be set beside its numbers**, under one key of its Zernio `metadata`,
  `record` (`recordOf` in `posts/zernio`): `clips stats` keeps it beside the post's TikTok and YouTube analytics
  every day (below, "`stats/` is what became of the posts"), to learn what makes a clip do well. **It records and
  decides nothing**: the dice are rolled in the same order as before (the arms of the experiments above are
  `choose`'s coins, not these), so
  the captions, the times and the order are what they were, and the tests pin the request body whole.
  - **It is one document, a `clips.v1.PostRecord` as protojson writes it**, not keys of Zernio's top level: its
    shape is the proto's and it grows there, with nothing to move in Zernio. `schema` names the message it is,
    `clips.v1.PostRecord`; a change that is not additive is a new message under a new name, so a reader always knows
    what it holds. `clip` is what the generator wrote (`clips.v1.ClipRecord`: its format, its commit, what every
    format has, its format's own section, its experiments' arms; the frontend's [Clips](../../frontend/docs/clips.md)), kept as written
    (`posts.ClipRecord`, which the job never reads); absent for a clip written before the generator kept one. `post`
    is what this job made of the clip: `teaser`, the name of the teaser rolled (`invades`; `unnamed` for a clip whose
    teasers have none), `callToAction`, `scheduledAt` (the time planned: the dashboard can move the post, this
    does not move), and `runPosition` and `runSize`, its place in its run from 1 and how many it has. **How fresh a
    story was** is `scheduledAt` minus the clip's `globe.storyEndedAt`, when the stretch it plays ends, which the
    generator writes.
  - **Every code in it is a word that says what it means; a number is a quantity.** The flat columns of a table are
    the reader's to make, not the document's.
  - The story's key, its countries and the call to act stay where they were, as keys of their own: `choose` reads
    them from a week of posts already out. A record that is not a `ClipRecord` fails the post before its video is
    uploaded. Zernio's `metadata` takes any JSON object and states no size, but for three keys of its own
    (`usageCounted`, `usageRefunded`, `hidden`). Posts made before the record have none, and none is filled in after.
- **`Plan.Next(clip, place, ofRun, feed, now, dice)`** is when and where: **on every platform with room left in its
  day, as soon as it can**: `Review` from now (none by default: `-review 0` and no flag are both none) and never under
  the 5-minute lead, on a whole minute, and **a `Spacing` (an hour) after the last post on each of its platforms**,
  whoever made it. False when no platform has room. It also rolls the post's words (`Dice`, `SystemDice`), so no post
  exists without them. `schedule` plans each clip once the one before it is sent, best first (`Ranked`), with the
  sent post added to the feed (`Feed.With`): a wait that moved a post later moves the next one too, and a clip
  refused takes no place, so the next one goes in it. **A platform is a `Platform`** (`tiktok`, `youtube`), and
  `PlatformsOf` is the only way from a name to one: each known, each once.
  - **Why an hour between two posts on a platform.** Without one, a run 2 hours after the last would post on top of
    the second clip of that run, which goes out an hour after its first, and two clips minutes apart split the first
    hour of reach that both TikTok and YouTube judge a Short on, among the same followers. An hour lets each have
    its first hour alone. It still fits the cadence: a run's 2 clips go out an hour apart, both before the next run
    renders, so nothing queues from run to run. 2 hours would push the second clip of every run past the next run's
    first, and 12 posts at most a day was already more than the budget.
  - **Timing**: the run of 12:17 UTC, started by GitHub at 12:34 and rendered at 12:53, posts its first clip at
    12:58 and its second at 13:58; the run of 14:17, rendered at 14:52, posts at 14:58, an hour after 13:58, then at 15:58.
- **The words of each platform are written here**, from what the generator gives (its teasers, the countries with
  their hashtags, the continent's, the music's credit), since the platform is the job's business. One teaser and
  one call per post, the same on every platform. **TikTok's caption** (`caption.go`):
  ```
  ITALY RETAKES ITS LAND FROM PORTUGAL | CLICKPLANET
  Play for free 👉 https://clickplanet.lol
  #clickplanet #rplace #wplace #pixelwar #map #geographygame #conquestgame #videogame #clickergame #viral #europe #italy #portugal #portugaltiktok🇵🇹
  ```
  A teaser and a call to act are picked at random, the call from `callsToAction`, and **the call's id is kept in
  the post's Zernio metadata** (`callToAction`), so the calls can be compared on the posts' numbers. A caption
  link cannot be clicked on TikTok, so a call is compared by how its posts do, not by visits. The music's credit,
  when its licence asks for one, comes before the tags. Then the game's tags, the continent's, each country's, and
  one TikTok community tag for the flag the clip is about (`#usatiktok🇺🇸`; an ISO private-use code such as `xk`
  has no flag emoji).
- **YouTube's words** (`youtube.go`) are a title, a description and tags, within YouTube's limits:
  ```
  ITALY RETAKES ITS LAND FROM PORTUGAL | CLICKPLANET

  Play for free 👉 https://clickplanet.lol
  #clickplanet #rplace #wplace #pixelwar #map #geographygame #conquestgame #videogame #clickergame #viral #europe #italy #portugal
  ```
  The title is the teaser and the brand, 100 characters at most: the teaser alone when both do not fit, cut with
  `…` when it does not either. The description is the call, the music's credit and the hashtags, **60 at most**:
  past that YouTube ignores every hashtag of the video. The tags are the same words with no `#`, the first that fit
  in 500 characters with a comma between two, as YouTube counts them. **No `#Shorts`**: a vertical video of 3
  minutes or less is a Short by itself. No TikTok community tag either.
- **No review, and nothing sent for a time already past** (2026-10-24). The review was 4 hours to cancel a clip in
  the dashboard before its time, and nobody did, so it is off; `-review` puts it back. **A post goes out 5 minutes
  ahead** (`leastLead`), not now: Zernio publishes a `publishNow` post, and one whose `scheduledFor` is already past,
  inside the request, answering each platform's result (a 207 when one fails), and the YouTube upload with it. 5
  minutes ahead keeps the request short and the post `scheduled`, which is what the adapter counts as taken, and is
  checked before every try: `Post.Rescheduled(now)` moves a post whose lead a wait ate to 5 minutes from then,
  later and never sooner. `Sender.Send` answers the post as it was sent, which the report and the feed read.
- **`Sender` tries again only when the service is busy**: an adapter answers a `*BusyError` for "not now" (a rate
  limit, a server error, no answer) and any other error for "no" (a refused caption, a wrong key). `Backoff` waits
  1, 5 then 15 minutes, or as long as the service said when that is longer; a service that asks for longer than
  the longest wait fails the post at once. `Pause` is the wait, so a test takes no time (`RecordedPause` moves the
  fixed clock instead).
- **`Service` is the posting service, one method**: `Schedule(ctx, post) (Scheduled, error)`. Its contract is on
  it: for the post's time and never sooner, on each of its platforms that takes it, and a post scheduled twice is
  one post there. `Scheduled` is the post's id and each platform it is not on, with why (`Refused`); the error is a
  post made on no platform.
- `Clip` and `Post` have no getter: an adapter reads `TakeSnapshot()`, and a `ClipSnapshot` the generator left a
  part out of does not `Restore()`. `Report` is what became of each clip on each platform, for the display and the
  log; `Complete()` is whether every one is a post on all its platforms.

**`posts/zernio/` is the one package that knows the service** (Zernio, chosen 2026-10-12 over upload-post: its
free tier has TikTok and the whole API, posts are unlimited, and the first two accounts are free). **No Instagram
for now** (2026-10-17): a third account is 6 USD a month. A change of service is that package, its line in `job.go`, the variable
`main.go` reads the key from, and the client below.

- **The HTTP client is generated, not written, and lives in `shared/cpzernio`** with all that makes it:
  `client.gen.go`, by oapi-codegen (pinned in the `Makefile`) from Zernio's own OpenAPI document, vendored
  beside it as `openapi.json`. `make zernio-spec` fetches the document Zernio builds its SDKs from, `make zernio`
  generates. `oapi-codegen.yaml` names the seven operations the job calls (`include-operation-ids`): four to post,
  and `getAnalytics`, `getYouTubeVideoRetention` and `listTikTokCommercialMusic` for the snapshot. So the client
  holds those and the types they reach, not the 631 paths of the document; `overlay.yaml` drops its webhooks. **An
  operation not named there is not in the client**: a new call is a line there and `make zernio`. The adapter keeps what
  no document says: which account, the upload, the idempotency key, and which answer is "busy".
- **Not Zernio's official Go SDK** (`github.com/zernio-dev/zernio-go`, looked at on 2026-10-13, v0.1.969). It is
  alive: a bot regenerates it from the same document several times a day, and its one maintainer fixed a
  critical bug in a day. But **it refuses any answer with a field it does not know** (`DisallowUnknownFields` in
  the generated models), and Zernio ships its API several times a day. Run against a fake: `ListAccounts` fails
  on one added field, and `CreatePost` fails on the 201 of the document's own example, **after the post was
  made**, the bug its issue 4 reported in August. A pinned version breaks when the API grows; ours ignores what
  it does not know, and two tests pin that (`…FieldsAddedLater…`). It is also 36 MB of Go in one package.
- **Four calls a clip**: `GET /v1/accounts`, `POST /v1/media/presign`, a `PUT` of the video to the address that
  answers (the storage, not the API: raw HTTP, and no key), then `POST /v1/posts` with `scheduledFor`, 5 minutes
  ahead at the soonest. **Never `publishNow`**: it publishes inside the request (see "No review" above). And one
  `GET /v1/posts` a run, for what each platform holds; `choose` makes it twice, for what was told too.
- **One clip is one Zernio post, with a target for each platform**, so one cancel in the dashboard stops it
  everywhere. Each target carries its own words (`customContent`) and its own settings (`platformSpecificData`), so
  a change for one platform (TikTok's sound) touches no other. Zernio publishes each target on its own: one
  platform that fails at its time leaves the post `partial`, and the next run's history says which (`tiktok: …`).
  - TikTok: public, comments, duets and stitches on.
  - YouTube: public, category Gaming (`20`), and **not made for kids**, sent every time since YouTube may block the
    views of a video with no answer. It is the user's legal choice (2026-10-17): "made for kids" turns comments
    off for good. The tags go at the root of the post, where Zernio sends them to YouTube alone. Zernio uploads a
    scheduled video to YouTube up to 15 minutes before its time, private until then: with 5 minutes of lead a clip
    is on YouTube as soon as it is scheduled, and **cancelling it there means deleting it on YouTube**.
- **It posts to the one connected account of each platform**, and refuses to guess between two: no account id is
  configured. A platform with none or two is refused (`Refused`), and the post still goes to the others; with no
  platform left there is no upload and no post. An account per country will need an id.
- **What is scheduled on a platform is what Zernio kept**: a platform the kept post has no target for is refused.
  A run made again after a platform was connected gets the first post back, without it, and says so.
- **`Idempotency-Key` is the SHA-256 of the video**, so a run made again, or a try after a timeout, is the same
  post there and not a second one. Zernio matches on the key alone, so the video uploaded again under a new
  address is still the same post.
- **A post counts only when Zernio keeps it as `scheduled`.** Any other status is a failure, `published` included:
  that would be a clip that went out with no review.
- **Busy** is a 429, a 5xx, a 409 `idempotency_conflict` (the try before is still in flight) and a request that
  got no answer, with `Retry-After` when sent. Every other answer is final, the 409 for content already posted
  too.
- **The logs of a public repo are public.** The key goes in the `Authorization` header of the calls to Zernio
  and never to the storage, and **an error never carries a URL**: the address a video is uploaded to is a
  credential for an hour, and Go's `url.Error` prints it. `unreachable` keeps the cause and drops the address;
  a test pins it.
- **A post a platform refuses at its time is seen by the next run**, about 2 hours after its own ended: the history `choose`
  reads holds each post's state and each platform's error, and `log_history` logs every post of the last week,
  a refused one as a warning with why (`a post of the last week was refused … failure="tiktok: …"`).
- The tests run it against `httptest`, with the requests and the answers of Zernio's OpenAPI document.

**`internal/clips/internal/stats/` is what became of the posts**, a concept of its own beside `replays` and `posts`,
for the experiments: what a clip was is in its post's `record`, and how it did is here, side by side. A view count read
once tells nothing of a clip's first day, and TikTok's best numbers do not wait, so it is **a snapshot a day**, kept
for good:

- **Why daily**: on a TikTok account connected through the TikTok for Business app, Zernio's `completionRate`,
  `impressionSources`, `audienceTypes`, `audienceCountries`, `retentionCurve`, `follows`, `profileViews`,
  `websiteClicks` and the watch times (`igReelsAvgWatchTime`, `igReelsVideoViewTotalTime`) come **24 to 48 hours
  after a post, and only for posts active in the last 7 days**. A day missed is lost for good. A post's age in
  hours is on every row, so the curve of a clip's first week can be drawn.
- **One row per platform per post per snapshot** (`stats.Row`): the Zernio post id, the platform, the account, the
  platform's id and address of the video, when it went out there, its age at the snapshot, the post's answer
  (`answer`: its HTTP status, a 200, a 202 while a platform still syncs, or a 424 when every platform failed; the
  post's status, its sync status and message), **the platform's entry of the answer as it came** (`analytics`,
  every metric in it, the ones no code here knows too), YouTube's retention curve (`retention`, below) and **the
  post's metadata as it came** (`metadata`: the story key, the countries, the call to act and the `record`).
  `version` is the shape of the row: a change that is not additive is a new version.
- **Which posts**: those with a story key, as `choose` reads them (`Told`), scheduled in the last 4 weeks
  (`DefaultSince`) or since `-since`, paged 500 at a time. A post out on no platform yet has nothing to measure and
  is counted, not kept; a platform it failed on is no row. 4 weeks: TikTok's business metrics stop at 7 days, and a
  Short's views have about settled by then. **The backfill is the same command** with `-since 2026-10-13`, the day
  the job started: the views and the likes are still there, the business metrics are not.
- **YouTube's retention** (`GET /v1/analytics/youtube/video-retention`, the platform's post id as `videoId`) is
  asked up to the day of the snapshot (`toDate`), the newest curve YouTube has, which it may still revise for 3
  days. **Any answer is kept as it came** (`Retention`: its status and its body), up to 100 points with
  `audienceWatchRatio` and `relativeRetentionPerformance`, or the 412 `youtube_analytics_scope_missing` that asks
  for the channel to be connected again, which the summary says plainly. YouTube has no curve for a video with few
  views or under 2 to 3 days old: an empty one is `empty`.
- **No analytics, no snapshot.** They need Zernio's Analytics add-on: `GET /v1/accounts` says
  `hasAnalyticsAccess`, and **false fails the run before any post is read, and nothing is written**
  (`stats.ErrNoAnalyticsAccess`, which says to turn it on). A 402 on a post's analytics is the same error, and
  fails the run whole. Any other post that cannot be read (a 404 for a post deleted, a service busy past the
  `Backoff`) is in the report's `Unread`: the others are kept, then the run fails.
- **Is TikTok's business lane live?** The snapshot keeps it for each TikTok account, and the backoffice shows it: `tiktokAccountType` (what
  TikTok said the account was when it was connected to Zernio: `business`, `personal`, `unknown`), how many of its
  rows of the snapshot hold a business metric (`Row.BusinessMetrics`: a completion rate, a profile view or a watch
  time above 0, or any source, country or retention point), and whether `GET /v1/accounts/{id}/tiktok/commercial-music`
  answers, which it does only for an account connected through TikTok for Business. **Live** is either, whatever the
  type says: the type is what TikTok said at the connection, and goes stale. The first snapshot (2026-10-24) found
  the account still `personal` to Zernio, the music library answering, and business metrics on 80 of its rows.
- **The store is the backend** (`ssh_stats_store`): `clips.v1.AdminService/KeepSnapshot` on the console
  listener, over ssh, as `fetch` asks for a replay (`docker compose exec -T backend wget … --post-file=/dev/stdin`).
  **The body comes on ssh's standard input, never the command line**: a part is far longer than a command may be,
  and it holds what Zernio wrote, which no shell should read. **A snapshot goes in parts of 200 rows** (about 3 KB
  a row), each with the whole snapshot's summary; the backend keeps a row again (snapshot, post, platform) and a
  summary again as one, so a run made again or a part sent twice keeps one snapshot. A part the backend kept less
  of than it was sent fails the run. The row goes as the job wrote it (`clips.v1.StatsRow.document`, the JSON of
  `stats.Row`), so a metric Zernio adds is kept with no change to the proto or the table. See
  [Clip stats](clip-stats.md) for what the backend does with it, and the backoffice's
  [Clip stats](../../backoffice/docs/clip-stats.md), for the table.
- **Nothing of it is public.** Completion rates, audience countries and profile views are not public numbers, and
  this repository, its artifacts and its logs are. The rows go to postgres, behind the console listener and the
  admin gate, and the workflow uploads no artifact and writes no summary. **The log carries counts and states, never
  a metric**: rows kept, posts not out, rows syncing, each TikTok account's lane, the posts lost with why, and why
  YouTube gave no curve (a warning for `youtube_analytics_scope_missing`, which asks for the channel to be connected
  again in Zernio).
- **The workflow** runs every day at 05:41 UTC, and by hand with `since`. It is one job: it holds the Zernio key
  and the VPS's (`VPS_*`, as `fetch`) and runs no npm, as the clips workflow's jobs do. A snapshot that lost a post
  is still kept, then the run is red.
- `take_snapshot_usecase` reads, then keeps the rows with the report, `log_take_snapshot` logs each TikTok lane and
  each post lost, and the adapter of Zernio is `posts/zernio`'s `Service` again (`stats.go`): the one package that
  knows the service. Its tests run it against `httptest` with the answers of Zernio's document, the use case's
  against a scripted source, and `ssh_stats_store`'s against a fake `ssh` that keeps its standard input.

`posts/generator_output/` reads the generator's files (`protojson`, unknown fields dropped, so the generator can
add one first), and both use cases log in a decorator
(`log_choose_clips`, `log_schedule_clips`), as do the service (`log_service`: each try the service did not take,
which is what explains a step that waited 20 minutes) and the history (`log_history`: how each post of the last week
went).
