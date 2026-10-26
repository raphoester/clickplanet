# Clips

**A clip is the real globe playing back a war, for TikTok, Shorts and Reels**: 1080×1920, 8.5 to 22s, a headline,
the map moving under it from the first frame, and the link at the end. `npm run clip` makes them from a replay
and **chooses everything itself**: the stretch of time, the place, the headline, the camera, the look and the
length. Each comes with a `.json` holding what to post with it, as data for the job that posts it (below), and a
`.txt` with its teasers, one a line, for a person. `--count 3` makes the three best stories, for a
person to pick one; `--plan` prints what they would be and renders nothing, and `--preview 3` renders only the
first 3 seconds; every choice can be forced (`npm run clip -- --help`).

**Every 2 hours a workflow posts the best ones with no person and no laptop** (`.github/workflows/clips.yml`, at
17 past every even hour UTC, or by hand with `workflow_dispatch`), so a story reaches the feed 2 to 3 hours after it
happened rather than 10 to 20 under the two runs a day before (2026-10-24). Four jobs:

1. **fetch**: the backend's `clips fetch -hours 12 -race race.json -season season.json` over SSH, with the deploy's
   `VPS_*` secrets. The job runs no npm at all, so the key to the VPS is in no job that installs this app's
   dependencies. The replay, the season's race and its calendar go on as an artifact; all are public data. A race or
   a season that could not be fetched is no file, and the run goes on without a day to tell or a country to call up.
2. **shortlist** and **choose**: `npm run clip -- --shortlist shortlist.json`, then, in a job of its own (it holds
   the Zernio key, and runs no npm), the backend's `clips choose` reads it and prints the picks to render, up to 2,
   and no more than the platforms have left of their day (14 posts in a trailing 24 hours on TikTok, 8 on YouTube),
   each with the arm of every experiment it put the clip in, `[{"pick":1,"variants":{"length":"short"}}]`.
   **It skips a story told in the last 3 days**, by its key below, and a headline told in the last week, puts a
   flag the last 6 posts showed after the others, and holds back a flag 2 of them showed, as their key or as one of
   their countries: on Oct 13–17 Algeria in Africa was told three times and Palau in half the clips. **A day won goes
   first**, whatever the posts showed, **then a run tells each angle it has once before any twice** (what changed,
   what a flag holds). None is a quiet run, or a full day: the run ends there, green. **On half the runs a model
   chooses instead, within those rules**, and writes each headline (the experiment `editor`, the backend's
   [The clips job](../../backend/docs/clips-job.md), "The editor"); the job holds its key too.
3. **render**: one job per pick, side by side, `--pick <n> --variants <its arms> --recorders 1`, and
   `--headline <the model's>` when the model wrote one. A runner has 4 cores and no GPU: Chrome
   draws with SwiftShader, **1.9 to 3.1s a frame, by the runner it lands on** (measured 2026-10-12: the same 12s
   clip in 11.5 minutes, then in 18.5; a 22s one would take 21 to 34), the same picture as with a GPU. More
   recorders do not help there: each one draws every frame.
4. **schedule**: the backend's `clips schedule` sends each clip to the posting service as one post on TikTok and
   YouTube Shorts, **for 5 minutes ahead, an hour after the last post on each**. There is no review: nobody cancels
   clips in the dashboard, so the 4 hours one cost were 4 hours of the story getting old. A clip can still be
   cancelled in the service's dashboard before its time, which stops it on both.

What chooses, when a clip goes out, the client of the service and the run's summary are Go, in the backend
(its [The clips job](../../backend/docs/clips-job.md)). **This app keeps only what needs the browser**, and hands the rest over as data,
`clips.v1` in the shared `proto/`, written as JSON:

- **`--shortlist <path>`** writes a `Shortlist` (`src/clip/shortlist.ts`): the title, the key and the countries of
  every story worth a clip, best first, so a story's place from 1 is its `--pick`: a day won first, then what changed,
  best first, then what flags hold, the biggest holding first. One page load, no globe, 6s. With no story it is `{}`, not an error.
  **Each story also carries what the job's model editor judges it by** (2026-10-24): its `teasers`, which a headline of
  the model's must not repeat, and its `facts` (`StoryFacts`, `storyFactsOf`): its kind, place, camera, sides and the
  flags it takes from, how each code is said, its line and language, and the numbers its clip would record, the span
  and when it ended, the tiles changed, the place's size, the counter at the start and the end, the effects in sight,
  the solidity, a holding's share and the generator's score. **They are the numbers `record()` already works out,
  measured once by `measuredOf`**, which the clip's record now reads too, so a story's facts and its clip's record
  cannot disagree; no camera is built for them (the look, the screens crossed and a tour's stops stay the record's).
- **`--race <path>`** is the season's race, a `seasons.v1.GetRaceResponse` as the backend's `GetRace` answers it,
  served to the page at `/__clip/race.json` (`src/clip/race.ts` reads its closed round). With none, that path
  answers 404, never the app's fallback page, and no day is told.
- **`--season <path>`** is the season, a `seasons.v1.GetSeasonResponse`, served at `/__clip/season.json` the same
  way: when its Final Battle starts. With none, nobody is called up.
- **A story's key says which story it is, the headline never does** (`storyKeyOf`, `domain/clip/story.ts`): the
  flag the clip is about and the continent it is told on, `{flag: "dz", continent: "Africa"}`. "ALGERIA STRIKES
  BACK" and "ALGERIA VS FRANCE" are one key. **Its angle says what the story tells of the flag there**: none for what
  it did, `holding` for what it holds, `round` for a day it won, with the day (`round: "1.5"`, season and number),
  `callup` for a country called up after a day, with that day. So "CROATIA HOLDS 80% OF EUROPE" is not "CROATIA IS ATTACKING EUROPE" told again, and France winning day 5 and day 6 is
  two stories. On Oct 18 a run of 13 stories skipped 8 as told already, nearly all one flag in Europe: every story was
  what changed, so a flag had one story a continent every 3 days. The flag is the first of `sidesOf`: the loser of a rout, a continent's
  own flag when it strikes back together. The continent is the place's (Asia's parts as the headlines name them,
  `the world` for a tour), else its first country's, else that country.
- **A story's countries are every country it shows** (`storyCountriesOf`): its place's, both sides and the flags it
  takes from, only the game's countries. The job counts them in what it posted: "VATICAN CITY VS PALAU" is Vatican's
  story and shows Palau. The `Clip`'s countries are the same list, with their hashtags.
- **`clip-<n>.json`** is a `Clip`: its pick, its title (the headline), its key, which the job keeps with the post,
  and **the words of its caption that need the story**: its teasers, its countries with their hashtags, its
  continent's hashtag, and the credit of its music. **The job writes the caption** from them (the backend's
  [The clips job](../../backend/docs/clips-job.md)): this app has the names, the job has the platform. The repo is public and so are a
  run's logs and artifacts: a clip can be watched there before it is out.
- **A `Clip` also says what the clip was** (`record`, a `ClipRecord`, written by `clipRecordOf` in
  `src/clip/clipRecord.ts`): the job keeps it with the post, never published, and the backend's `clips stats` sets
  it beside the post's numbers on TikTok and YouTube every day (`.github/workflows/clips-stats.yml`; the backend's
  [The clips job](../../backend/docs/clips-job.md)), to learn what makes a clip do well. **It records, and chooses nothing**:
  every value is one `record()` already worked out, read once the clip is decided, and the same replay gives the same
  video, teasers and `Clip` byte for byte as before it, `record` and `teaserNames` aside (checked on 2026-10-23
  against a render of `main`). **It is shaped to outlive this kind of clip**, since the clips will be experimented on:
  - **`format` names the kind of video**, and the section that describes it: `globe-replay`, the game's globe, in
    `globe`. A clip made another way (another layout, a video made by AI) is a new format with a section of its own,
    beside `common`, what every format has.
  - **Every code is a word that says what it means**, never a number standing for one: `camera` is `front` or
    `tour`, `look` `dive`, `flags` or `none` (a tour has none), `music` the anthem's country code or `none`,
    `storyKind` the story's kind, `language` `en` or a call-up's own (`voiceOf`). **A number is a quantity**, in the
    unit its name says: `durationSeconds`, `musicStartSeconds`, `spanMinutes` of the replay played, `tilesChanged`
    where the story is told (a bomb's cleared tiles too), `storyTilesChanged` (the fighting the camera follows),
    `placeTiles` (the place the headline names), `flagTilesStart` and `flagTilesEnd` (the counter, for the key's
    flag), `landmassesFlipped`, the effects in sight of the story as `--plan` counts its bombs (`bombs`, `spreads`,
    `enclosures`, from `ReplayBackend.effects`), `screensCrossed`, `tourStops` and `solidity`. One protojson leaves out
    is its zero.
  - **`storyEndedAt` is when the stretch it plays ends**, the newest moment of the story, as a time (the stretch's
    `until`, so the cutoff for a day or a call-up). The post's `scheduledAt` minus it is how long the story took to
    reach the feed, which is what running every 2 hours set out to cut.
  - **`generator` is the commit that made it**: `render.mjs` passes the workflow's `GITHUB_SHA`, else the checkout's
    `HEAD`, to the page as `commit`; `unknown` without either.
  - **`variants` is the arm of each experiment the clip is in**, by name (`{"length": "short"}`): a free map of
    words, so an experiment needs no change of shape. Empty for a clip in none, as one made by hand.
  - **Additive only**: a new value is a field at the end of its message. A change that is not is a new message
    under a new name, since the job names the message it writes (the backend's [The clips job](../../backend/docs/clips-job.md)).
- **`--variants <json>` is the arm of each experiment the clip is in**, as the job's `choose` assigned it
  (`{"length":"short"}`). **Which experiments run, and which arm a clip is in, is the job's alone** (the backend's
  [The clips job](../../backend/docs/clips-job.md), "Experiments"): a coin, not something this app picks from the story, or the arms would not compare.
  `src/clip/experiments.ts` lists every experiment this app knows and its arms (`EXPERIMENTS`); `variantsOf` reads
  them when the clip is recorded, before anything is built, and **an experiment or an arm it does not know fails the
  clip**, which the run then does not post. It applies the arms it acts on, and writes every arm it was given into
  `record.variants`, the ones it applied and the ones the job acts on when posting, which it lists with nothing to
  apply: so `schedule` reads them all from `clip-<n>.json`. With no `--variants` a clip is in no experiment and is
  made as before; `--seconds` with an arm of `length` is refused, since both set the length. `--plan` prints the arms.
  - **`editor`**: `rules`, `model` or `fell-back`, the job's coin for the whole run, flipped before it chooses (the
    backend's [The clips job](../../backend/docs/clips-job.md), "The editor"). Nothing to apply here: the model's headline comes as `--headline`, which
    `wordsOf` already takes, a bridge until the model writes a whole clip script. The teasers are left as they are,
    an experiment of their own.
  - **`sound`**: `anthem` or `trending`, the job's coin for each clip. Nothing to apply here: the video is the same in
    both, anthem included, and the job attaches a track trending on TikTok over it when it posts a `trending` clip,
    the anthem muted (the backend's [The clips job](../../backend/docs/clips-job.md), "Experiments"). The job's `fell-back`, a `trending` clip posted with its
    anthem when the chart could not be read, is decided after the render, so it never reaches this app, which refuses
    it like any arm the job does not draw.
  - **`length`**: `as-chosen`, the control, is exactly the length `pace.ts` gives (below); `short` is the shortest a
    story of its kind plays (`timelineOf(…, "short")`): 8.5s for a front, however far its fighting moves, 10 with a
    day's podium. **A tour is already its shortest**, each stop a flight and a stay, and is left as it is, as is a
    front whose fighting stays in one place: the arm is still recorded, and `durationSeconds` keeps the real length.
    The camera fits the play it is given either way. Checked on 2026-10-24 on a replay of three invasions: `short`
    made 10s and 9s clips 8.5s, and `as-chosen` rendered the same video and teasers as `main`, byte for byte, and the
    same `Clip` but for `record.variants`.
- **Each teaser has a name** (`Words.teaserNames`, beside `teasers`, in the same order): the name of its wording, the
  same for every flag and every language, `invades` and `under-attack-from` for an invasion, `can-anyone-stop` for a
  call-up's leader in any of its languages. The `Clip` carries them (`teaserNames`), so the job can say which wording
  a post opens with and the posts can be compared by it. A test checks every kind names each of its teasers once.

- **The replay is the backend's** (`planet.v1.AdminService/GetReplay`, see the backend's [GetReplay](../../backend/docs/admin.md#getreplay)): the map as it
  was at the start, and every act after it as the live stream sent it. The backend's job asks for the last
  72 hours over SSH (`cd ../backend && go run ./cmd/clips fetch -out ../frontend/replay.json`, with `-ssh` or
  `CLICKPLANET_SSH`: it needs no browser, so it is Go); `npm run clip:record` follows the public stream for a few
  minutes instead, to try the generator without the operator tool.
- **It is the game's own globe**, so a clip shows exactly what a player sees: `clip.html` is a page of its own,
  served by `npm run dev` and **left out of the build** (it is not in `rollupOptions.input`). `src/clip/main.ts`
  builds the globe with `createGlobe` over `backends/replayBackend.ts`, which plays the replay through the same
  decoders as `PlanetBackend` (`updateOf`, `bombOf`, `spreadOf`, `enclosureOf`): tile updates in batches, and a
  bomb, a spread or an enclosure as an effect. `ReplayBackend.cut` opens on the map at a later time.
- **The page chooses, then records** (`window.clip`, `src/clip/main.ts`): `shortlist` is the stories worth a clip,
  known before any globe is built, and `record()` builds the globe for the story asked for and answers what the
  recorder needs. A pick past the last story fails `record()`, never `shortlist`.
- **The recorder owns the clock.** `src/clip/virtualClock.ts` replaces `performance.now` and
  `requestAnimationFrame`, so a frame is drawn only when `window.clip.frame(i)` asks, and a bomb takes its second
  of video however long a frame takes to capture. **No CSS animation or transition on the page**: they run on the
  real clock. The overlay is set from JS each frame, and an animation the globe brings with it (the bomb pointer's
  pulse) is paused and set to the video's time at every frame. **It owns the dice too**: `main.ts` seeds
  `Math.random`, so a bomb's debris and its shake are the same in every browser that plays the clip.
- **The camera is the clip's** (`GlobeOptions.director`): with one, `createGlobe` turns OrbitControls off and
  aims the camera at the shot it is given every frame. The game passes none.
- **The counter counts where the headline says**: the tiles each side holds in the story's place (`inPlace`), a
  country, two, a continent or the world. Not in all that is near its front: "CYPRUS IS KICKING POLAND OUT OF
  PAKISTAN" showed Poland still holding 675 tiles, its land in India and Iran, when it held 7 in Pakistan.
- **The text stays where TikTok draws nothing** (`--safe-*` on `#clip` in `clip.css`, measured on a phone): a tall
  phone crops the sides, the tabs cover the top, the buttons run down the right from the middle, and the name and
  the caption cover the bottom. The counter sits left of the buttons, and the call to act in the top half, darkening only that half: the map
  pulled back out stays in sight under it. The call
  shows the flag it asks the viewer to fight for: the country to defend, the flag that strikes back, both sides of a
  battle, or a continent's own flag (Europe's alone, `static/countries/svg/eu.svg`, from the same set as the others).

**`src/domain/clip/` is the director**, pure and under test:

- **`window.ts` finds the candidates**: for each length from 1 to 24 hours, the busiest stretch of a few places
  far apart, counting the tiles taken from another flag in each 10° cell and the eight around it. Filling empty
  ground is not war. **A clip is news**: with no window asked for, only the last 12 hours of the replay are searched
  (`newsSince`), since the busiest stretch of three days is rarely today's and would hide today's from the search;
  posted a day late, "Christmas Island is taking over the world" is about a flag already thrown out. A window asked
  for (`--since`, `--until`) is split into its places the same way, so a war next door is a story of its own. **Every scale is searched**: the whole map, and each of the 12 countries whose
  ground changed hands the most (400 tiles or more), on its own ground alone, as `--focus` does. So Germany and
  Romania trading Germany is a story beside the war across Europe it is part of.
- **`front.ts` finds the front** of a candidate: the point where most tiles changed hands, and every change within
  about 2,900 km of it, so a war in France brings in England, Spain and Germany. Once the story is known, the front
  is found again from its own flags' fighting alone, so a war next door (Israel in Turkey) does not pull the
  camera off Belgium's; with `--country`, it is that flag's fighting from the start.
- **`story.ts` writes the story** (`storyOf`): the flag that took the most there, the flags it took from, and
  where. Nearly all in one country (90%) is **"X IS INVADING FRANCE"**; spread over several, it is **"X IS
  ATTACKING"** the continent holding 70% of it (`static/countries/regions.json`, written by `npm run regions` from
  the snapshot the map is cut from), else **"X IS INVADING EGYPT AND TURKEY"** when two countries hold 70% of it,
  else the world. Not a sub-region: "defend Western Europe" is not how anybody talks. But Asia is too big to be one
  place, and nobody calls Saudi Arabia "Asia": its parts are named as people name them, **"THE MIDDLE EAST"** (western
  Asia and Iran), South Asia, East Asia, Southeast Asia and Central Asia. A flag taking back its own ground, or its own continent from flags from elsewhere (Belgium taking Europe
  back from Palestine), is **"X STRIKES BACK"**, when they lost it at least half of what it took there: Portugal taking
  Europe from Israel first but from Germany, Poland and Spain more is attacking it; a flag that already held most of the country
  when the story starts is **"X IS KICKING Y OUT OF AUSTRALIA"**, since the opening shot shows its flag there
  already; a second flag taking 60% as much makes it **"X VS Y"**, but only when the two are at war, a quarter of
  what each took taken from the other: Israel and Belgium both taking Europe from Palestine are allies, not a battle,
  and Portugal nibbling at Romania while Romania takes the USA from Algeria is no battle either.
  `src/clip/overlay.ts` words it.
- **A story is about who leads the fighting** (`castOf`): its flags have to take 35% of everything taken around
  it. Below that, flags of one continent taking it back together, with half of it between them, are the story,
  **"EUROPE STRIKES BACK"**, under the continent's flag with one counter for them all. Otherwise nobody leads it
  and it is skipped: Germany taking its own land back while Belgium and Israel made the war around it.
- **A flag thrown out is its own story** (`routOf`): once the flag a story takes most from has lost half of what it
  held in the place (300 tiles or more), a continent's flags taking it back together become **"PALESTINE GETS
  KICKED OUT OF EUROPE"**, told from its side with its counter falling against the continent's, and one attacker
  becomes **"ISRAEL IS KICKING PALESTINE OUT OF EUROPE"**. When many flags take it and none leads, it is the story
  alone, placed where it lost its land: **"PALESTINE GETS KICKED OUT OF SOUTH AMERICA"**, with "Fight for
  Palestine"; when the flag that took most of it leads there, though the flag of the front did not, it is that flag
  kicking it out (`lossOf`, told from scratch). Only a flag thrown out makes a story of a front nobody leads: Portugal
  losing France and taking it back by evening, 1,004 of its 1,110 tiles, is no "FRANCE IS KICKING PORTUGAL OUT OF
  FRANCE", whatever the front's own story was. With `--country`, a story that flag leads nothing in is skipped. Nobody is kicked out of their own land: a flag losing its own continent **"IS LOSING AFRICA"**, its own
  country **"IS FALLING"**, and one attacker taking it is an invasion. A flag taking its own ground back still strikes
  back.
- **A flag taking land on several continents at once is on a tour of the world** (`tour.ts`): the busiest stretches
  of the whole map, for the 3 flags that took the most in each (or `--country`), when no continent holds 70% of
  what the flag took and it leads the world (`leadsTheWorld`): twice as much taken as any other flag, since no flag
  takes a third of all taken across the world, as one does on a front (Christmas Island took 28% of it on Oct 9). **"PORTUGAL IS TAKING OVER THE WORLD"**,
  and its call is "Fight back", with no flag. Each continent (Asia's parts) with 5% of what it took is a stop, four at most,
  in the order it got there (a fifth of it taken). Each stop is framed where it took the most there (Angola, not
  all of Africa from Guinea-Bissau to Mozambique, which only the whole globe frames). The camera opens on the whole
  globe over the first stop and dives into it, twice as close as that framing but never past where the painted flags
  start to blend into the tiles (`flagsZoomOf`): the flags changing hands are what a tour shows. At each stop it
  follows the flag's fighting there as it happens, as a clip follows its front, then flies to the next, rising a
  little over a long way (1.6 times out at most), never out to the globe and back in, which was brutal. It pulls back
  out to the globe over all of them at the end; 1.2s a flight and 2.6s a stop. Up close, a big country's painted flag
  is blurred: the flag atlas is the game's. The counter counts
  the flag across the world. A tour is told beside the flag's stories on each continent, not instead of them.
- **One story per flag and what it did** (`sameStory`), whatever window or scale found it, told over the most of the
  map (`widerThan`), else the best: one flag taking one place or one inside it, whoever it beat there ("Portugal is
  invading Germany" is part of "Portugal is attacking Europe"), one flag thrown out of one place or one inside it, whoever threw it out ("Israel is kicking Christmas
  Island out of Germany" is part of "Christmas Island gets kicked out of Europe"), one flag
  striking back. Portugal beating France in Africa and in India is two stories. A battle is its two sides in one
  place. `--plan` lists each one dropped with the one it repeats.
- **A country is said as people say it.** The game's names are cut to fit its board ("Christmas", "N.Zealand", "Czech
  Rep."), so a clip says the English name they were cut from, from the CLDR data the browser carries
  (`Intl.DisplayNames`): "CHRISTMAS ISLAND IS TAKING OVER THE WORLD". A name the game chose over CLDR's stays: "Turkey",
  "Ivory Coast", "UAE", "DR Congo", and its own flags'.
- **Names are said with their article** where English wants one: "the UK", "the Netherlands", "the Dominican
  Republic". Tags keep them bare.
- **The words say nothing the map says better.** Only a battle has a line under its headline, "The battle for
  France": no count of tiles and no "in 3 hours", which the counter shows and which read as written by a machine.
- **The caption never repeats the headline** (`teasers` in `wordsOf`): the clip already shows it, and the
  posts written by hand read better. Each kind of story says it two ways, which the job picks between: "PALESTINE
  IS KICKING ROMANIA OUT OF KAZAKHSTAN" is "PALESTINE TAKES KAZAKHSTAN FROM ROMANIA", an invasion that threw
  another flag out "CROATIA INVADES CHINA AND KICKS OUT FRANCE", a flag thrown out "GENERATIONAL DOWNFALL OF
  CHRISTMAS ISLAND", and a country throwing a flag out of its own land "ITALY RETAKES ITS LAND FROM PORTUGAL".
  A holding is "80% OF EUROPE FLIES THE FLAG OF CROATIA" and "CROATIA RULES EUROPE" (below half, "the biggest power
  in Europe"; the world's, "THE BIGGEST EMPIRE ON EARTH"), and asks to "TAKE EUROPE BACK"; a day won is "FRANCE TAKES
  DAY 5" and "DAY 5 GOES TO FRANCE". A test checks that no kind ever gives the headline back.
- **The hashtags are every flag of the story and the countries of its place** (`hashtagsOf`), bare
  (`#christmasisland`, `#uk`), and the place's continent. Palestine and Israel included: the user chose it on
  2026-10-14, though a flag's tag can be a political feed. A continent's own flag is not a country and has none.
- **`music.ts` picks the anthem under the clip**, since YouTube Shorts cannot add a sound to an upload: the anthem
  of whoever makes the moves, the leading flag's, else the other side's in a battle, or the continent's when a
  continent strikes back. Never a loser's, but for a flag thrown out by many flags none of which leads ("ALGERIA IS
  FALLING"): the clip names no other flag. A clip with none plays none. They are the game's own anthems (see [The
  leader's anthem](sound.md#the-leaders-anthem)), US Navy Band recordings in the public domain, and two the game does not
  play, vendored by `npm run clip:anthems` into `scripts/clip/anthems/`: the Anthem of Europe (Navy Band too) and
  Palestine's Fida'i, which the Navy Band never recorded, in an instrumental under CC BY 3.0. A recording under a
  licence carries its credit, and the caption of every clip it plays under ends with it. It fades out over the
  last 1.2s. `--silent` leaves it out, and its credit with it, for TikTok and Instagram, where a sound is added
  when posting.
- **An anthem starts on a highlight**, never on its first second: the UK's opens on 11s of drum roll, Algeria's on 7s
  of drums. `highlight.ts` hears a drum roll from a tune (a noisy spectrum, against the band's notes) and ranks the
  notes struck with the band playing on, loudest first, a phrase after a pause counting as louder.
  `npm run clip:highlights` writes what it finds for every recording into `src/clip/anthemHighlightsAsset.ts`,
  keyed by the content-addressed file, and both anthem scripts run it, since a new encode is a new file; a test
  fails when one is missing. A clip starts on the best highlight that leaves the anthem playing to its end, else
  early enough that it does.
- **`holding.ts` tells what the map is, beside what changed on it**: as the map stands at the end, the flag holding the
  most of each place, **"CROATIA HOLDS 80% OF EUROPE"**. The places are the world (from a tenth of it), each
  continent (from 40%) and each country of 1,000 tiles or more (49 of them), held by a foreign flag (from 75%). **Europe
  is held without Russia**: Russia's ground is 71% of Natural Earth's Europe, so a flag holding Siberia would "hold
  Europe"; Russia is a place of its own. A share is of all the place's tiles, empty ones too, and the headline never
  says more than it holds (79.96% is 79%). It is news only when its flag fought for it: 50 tiles of the place changed
  hands to it or from it in the window, which is what the clip plays, all over the place, with the counter as the
  flag's share of it. It is no story when it is mostly the flag's own country (Australia holding Oceania), and a flag
  is told once, where it holds the most for the place's bar: 80% of Europe over 13% of the world, Antarctica the
  continent over Antarctica the country. `--plan` lists each place's holder that fell short, and why: the bars are
  guesses made with no production data, to tune from those lines.
- **`round.ts` tells a day of the season** from the race (`--race`): **"FRANCE WINS DAY 5"**, or **"FRANCE WINS THE
  FINAL BATTLE"**, with where it leaves the winner in the season ("Leads the season with 86 points"). Only the run
  whose window holds the cutoff tells it: the 21:17 run, for the day that closed at 21:00, and the runs after it skip it
  as told. It plays the 3 hours before
  the cutoff, the winner's fighting (anyone's when it fought none), opens on all the winner holds, counts its tiles
  (not its share of the map: a day is won on its average, which the podium's points say), and ends on the podium, the places that scored with their points, then "WHO WINS DAY 6?". The ending
  lasts 4s instead of 2.5, to read it.
- **`callup.ts` calls up each country that can still win the season** (`--race` and `--season`), after the day the
  race closed last, told by the same run as the day: a country contends when winning every day left and the Final
  Battle, the leader second each time, brings it level. By pure points, 25 countries could still win on day 15, the
  UK 288 points behind; with this rule two did. A leader nobody can catch is no call-up. **It speaks the country's
  language** (`src/clip/callUp.ts`): CLDR's likely language for the country (`Intl.Locale.maximize`), when the clip
  has words for it (English, French, Spanish, Portuguese, Italian, German, Dutch, Polish, Romanian, Turkish,
  Hungarian), else English. The words never put a country's name where a language would decline it ("POLSKA: 14
  PUNKTÓW STRATY", not "behind Algeria" in Polish), names come from CLDR in that language, numbers and plurals from
  `Intl`. The Final Battle's time is the one most of the country's zones read, with the zone named when they
  disagree ("18:00 BRT"). It plays as a day does, and ends on the leader's row and the country's, "YOUR COUNTRY
  NEEDS YOU" in its language, and the Final Battle's time. **Luckiest Guy draws Latin-1 only**: a clip with any
  other letter ("Ł", "Ț", "İ") is set in Rubik Black, never in two fonts.
- **`score.ts` ranks the candidates**: the tiles taken from another flag, over the square root of the hours, times
  the countries they were taken in (up to 4). A short war over several countries beats a long filling of one. A
  story already told by a better candidate (same attacker, same front) is dropped.
- **`solidity.ts` skips graffiti.** For each tile the attacker took and holds at the end, the share of its 6
  neighbours it holds too: about 1 for land taken, 0.56 for names written across Canada. Under 0.75 the story is
  skipped, and `--plan` says so. So is a front whose story lands in "the world": its tiles are spread over several
  continents and there is no one place to show. Only a flag leading the world gets a tour of it.
- **`look.ts` picks when the camera comes back out of the tiles.** From far, a landmass's painted flag only
  changes when its biggest holder does (`flipsOf`, over the borders blob, read 8 times along the changes, so a
  landmass taken and taken back counts too). A front too wide to frame closer than
  `TILES_ZOOM` (3) is **flags** when at least 2 landmasses changed their biggest holder and those hold 2,000 tiles
  or more: a steamroll, which comes back out halfway so its painted flags change on screen. Everything else is a
  **dive**, which comes back out at the end.
- **`camera.ts` opens on the map and dives into the tiles**: every clip opens on the middle of the front's
  changes, zoomed until 95% of them fit but never closer than a continent (`openingOf`), so the first frame is the
  map with its painted flags. It holds there 0.3s (`--hold`), then dives in 0.7s past the zoom the painted flags
  are gone at (`tilesZoomOf`), to where the most tiles change hands, so the fight is seen tile by tile. It follows
  the densest fighting, and comes back out to the opening as the look says. **Every clip ends pulled back out**,
  and holds there 1s before the call to act while the last tiles change hands: close-ups are for the middle, and
  the end shows the rest of the map as it is now. **It never sits still**: where the
  fighting crosses less than 0.4 screens a second, it breathes, out to where the painted flags show and back into
  the tiles every 3s. **A bomb holds nothing still**: neither the camera nor the replay's clock stops for it, as five
  bombs in a row on Portugal froze a clip of Portugal taking Europe. It goes off while the map goes on changing, and
  the camera goes there only as the tiles it clears pull it, like any others. `--plan` counts the bombs on the front
  (`inSightOf`): not one in Alaska for a story told on the USA while the fighting is in Florida. The globe is
  always drawn with the painted flags on, so the zoom alone hands them over to the tiles, as in the game.
- **`pace.ts` spends the clip on what happens and nothing else**: the replay's clock jumps over every quiet
  stretch, so the map moves from the first frame to the last. **A clip is as long as its camera has somewhere to
  go**: 6s for a fight in one place, however many hours it lasted (room for the dive, the close-ups and the pull back
  out), and 1.2s more for every screen the fighting
  crosses up close (`screensOf`, on the camera's smoothed path), up to 15s. A bomb adds nothing. The last 2.5s are the call to act, and nothing runs past 22s.
  So a front runs 8.5 to 17.5s (10 to 19 with a day's 4s podium) and a tour 9 to 20.5, by its stops; only `--seconds`
  reaches 22. The experiment `length` tries the shortest of each against it (above).

**`scripts/clip/render.mjs` is the recorder**: it starts Vite, serves the replay at `/__clip/replay.json`, opens
headless Chrome at 540×960 at 2×, waits for `window.clip.record()`, then for each frame calls
`window.clip.frame(i)`, takes a screenshot and pipes it to ffmpeg (libx264, `yuv420p`). It needs ffmpeg and Chrome
(`CHROME_PATH`, which the workflow sets: the default is the macOS path). **A Chrome that does not come up is started
again, twice** (`LAUNCH_TRIES`, `withRetries` in `scripts/clip/retry.mjs`), each time with a line (`Chrome did not
start, starting it again: …`), from a new profile, the one that failed killed and its profile removed; the third
failure fails the run, with the error it had. On 2026-10-24 one run in 35 lost its `shortlist` job to "Chrome never
exposed a debugging target", 20s after the launch, and the next run passed: a lost run is a lost chance to post on
time. Only the launch is tried again: a page that fails still fails the clip.

**The screenshot is nearly all of a clip's time, so several browsers take them in turn** (`--recorders`, one for
every 3 cores, 4 at most). The globe draws a frame in 6ms; Chrome takes 0.3s to encode it as a 2.4MB PNG, on one
core. Each recorder is a Chrome of its own that plays every frame, so all hold the same map, and takes the picture
of its own only: frames 0, 4, 8 from the first, 1, 5, 9 from the second. They reach ffmpeg in order, and they are
the frames one recorder would take, since the page's clock and dice are the recorder's (above). On 12 cores a 12s
clip takes 40s instead of 90s, and a fifth recorder adds nothing. Each one holds the whole replay, 1.8GB for 72
hours. The 14s before the first frame are the page reading the replay and choosing the story, once per clip.
