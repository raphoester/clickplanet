# Clips

**A clip is the real globe playing back a war, for TikTok, Shorts and Reels**: 1080×1920, 8.5 to 22s, a headline,
the map moving under it from the first frame, and the link at the end. `npm run clip` makes one from a replay and a
story: **the generator measures, the editor judges** (2026-10-27). `--findings` writes what the replay holds, every
front, tour and place with its numbers and nothing decided; a model in the backend's job writes the stories from it;
`--story` renders one of them, and chooses the camera, the look, the length and the music itself. Each comes with a
`.json` holding what to post with it, as data for the job that posts it (below), and a `.txt` with its teasers, one a
line, for a person. `--plan` prints what the clip would be and renders nothing, and `--preview 3` renders only the
first 3 seconds; every choice can be forced (`npm run clip -- --help`).

**Every 2 hours a workflow posts the best ones with no person and no laptop** (`.github/workflows/clips.yml`, at
17 past every even hour UTC, or by hand with `workflow_dispatch`), so a story reaches the feed 2 to 3 hours after it
happened rather than 10 to 20 under the two runs a day before (2026-10-24). Four jobs:

1. **fetch**: the backend's `clips fetch -hours 12 -race race.json -season season.json` over SSH, with the deploy's
   `VPS_*` secrets. The job runs no npm at all, so the key to the VPS is in no job that installs this app's
   dependencies. The replay, the season's race and its calendar go on as an artifact; all are public data. A race or
   a season that could not be fetched is no file, and the run goes on without a day to tell or a country to call up.
2. **findings** and **choose**: `npm run clip -- --findings findings.json`, then, in a job of its own (it holds
   the Zernio and Anthropic keys, and runs no npm), the backend's `clips choose` reads it and prints the stories to
   render, up to 2, and no more than the platforms have left of their day (14 posts in a trailing 24 hours on TikTok, 8
   on YouTube), each with its number in the run and the arm of every experiment it put the clip in,
   `[{"pick":1,"story":{"front":3,…},"variants":{"length":"short"}}]`. **A day won goes first**, told once; **the
   model makes the rest** (the backend's [The clips job](../../backend/docs/clips-job.md), "The editor"): which fronts
   are stories, of what kind, between whom, where, and their headlines. The job checks each against the findings, and
   against the headlines of the last week, and skips one that does not hold with why. None, or a full day: the run ends
   there, green. **No rules choose any more.** Until 2026-10-26 hand-written rules decided what a front was a story of
   and what may be told, and the model only ranked what they let through: that evening they posted Brittany taking a
   corner of Australia while France and Turkey fought the biggest battle in weeks and France took all of India, both
   skipped because a flag was "told already" on that continent or "shown lately" as a bystander in other clips.
3. **render**: one job per story, side by side, `--pick <n> --story <its JSON> --variants <its arms> --recorders 1`.
   The story goes through the environment, never into the script: it is the model's text. A runner has 4 cores and
   no GPU: Chrome draws with SwiftShader, **1.9 to 3.1s a frame, by the runner it lands on** (measured 2026-10-12: the
   same 12s clip in 11.5 minutes, then in 18.5; a 22s one would take 21 to 34), the same picture as with a GPU. More
   recorders do not help there: each one draws every frame. Chrome gets a minute to start: 20s failed the run of
   2026-10-26 18:40.
4. **schedule**: the backend's `clips schedule` sends each clip to the posting service as one post on TikTok and
   YouTube Shorts, **for 5 minutes ahead, an hour after the last post on each**. There is no review: nobody cancels
   clips in the dashboard, so the 4 hours one cost were 4 hours of the story getting old. A clip can still be
   cancelled in the service's dashboard before its time, which stops it on both.

What chooses, when a clip goes out, the client of the service and the run's summary are Go, in the backend
(its [The clips job](../../backend/docs/clips-job.md)). **This app keeps only what needs the browser**, and hands the rest over as data,
`clips.v1` in the shared `proto/`, written as JSON:

- **`--findings <path>`** writes a `Findings` (`src/clip/findings.ts`, `findingsOf`): what the replay holds, measured
  and not judged, for the editor to make the stories from. One page load, no globe, a few seconds. With nothing it is
  `{}`, not an error.
  - **`fronts`**: where tiles changed hands the most, the busiest first, 30 at most (`MOST_FRONTS`). Each is a
    stretch of time and everything within about 2,900 km of where most changed hands (`window.ts`, `front.ts`, below),
    on the whole map or on one country's ground alone (`scope`: each of the 12 countries whose ground changed hands
    the most, 400 tiles or more, as `--focus` does, so Turkey's war is a front of its own beside the Middle East's),
    with its numbers (`domain/clip/findings.ts`, `frontFactsOf`): the tiles changed and the ones taken from another
    flag (`captured`: filling empty ground is not war); its countries by tiles changed there, each with its continent,
    its size, who held the most of it when the front opened and when it ended (`holdersStart`, `holdersEnd`) and who
    took the most there (`takers`); its flags by tiles taken and lost, each with what it took, filled, lost, whom it
    took from and the solidity of what it took (below); the bombs, spreads and enclosures in sight; and the score it
    was ranked by. So a flag that held most of a country at the start and took more of it is kicking someone out, a
    flag that held much at the start and little at the end is thrown out, and two flags taking from each other are at
    war: **the editor reads that, no threshold does.**
  - **`tours`**: each flag taking land across the whole map, in its busiest stretch, with what it took on each
    continent and from whom, what the second flag across the world took in the same stretch, its solidity, and **the
    tiles of the landmasses whose painted flag switched to it, by continent** (`flippedByContinent`, from `flipsOf` and
    `flippedTo`; a landmass under 100 tiles, Belgium's size, does not count: it is a speck at a continent's framing): a
    tour's camera stays far out, where a landmass's painted flag switching is all a viewer sees. A flag far ahead of the
    next, switching landmasses on two continents or more, is taking over the world, which the camera tells as a tour
    (below); the job refuses a tour of a flag that switched none.
  - **`places`**: what the map is at the end: the flag holding the most of the world, of each continent and of each
    country of 1,000 tiles or more (49 of them), with its share, the next holders and how many tiles of the place
    changed hands to it or from it in the window, which is what its clip plays; a place where nothing did is left out,
    since there is nothing to play. **Europe is held without Russia**: Russia's ground is 71% of Natural Earth's
    Europe, so a flag holding Siberia would "hold Europe"; Russia is a place of its own. A share is of all the place's
    tiles, empty ones too.
  - **`seasonal`**: a day of the season won and each country called up after it (`round.ts`, `callup.ts`, below),
    told whole by this app, with the title, the key, the countries and the facts their clips would record
    (`StoryFacts`, `storyFactsOf`, the numbers `record()` works out, measured once by `measuredOf`).
  - **`names`**, how a clip says each code in the findings ("gb": "the UK"), and **`continents`**, each with its own
    flag when it has one (Europe's).
- **`--story <json>`** is the story to render, a `Pick` (`toldOf`): one front, tour, place or seasonal story of the
  findings by its number, from 1, and for a front what the editor made of it: its `kind` (`attack`, `invasion`,
  `comeback`, `kickout`, `rout`, `battle`), its `attacker`, its `rival` in a battle, whom it took from (`takenFrom`,
  the flag thrown out first), where it is told (`countries`, one or two, or `continent`), the continent whose flags
  strike back together (`team`), and its `headline`. A tour, a place and a seasonal story are the facts' own: the
  flag taking the world, the holder holding its share, the day won. **The page builds the `Story` from it and reads no
  rule**: a pick that names no front, a kind no front tells, no attacker or no place is refused as it is read, and
  the job checked it against the findings before the render. `--headline` overrides the story's.
- **`--race <path>`** is the season's race, a `seasons.v1.GetRaceResponse` as the backend's `GetRace` answers it,
  served to the page at `/__clip/race.json` (`src/clip/race.ts` reads its closed round). With none, that path
  answers 404, never the app's fallback page, and no day is told. **A battle's round of its own is never told as a
  day** (`Day.own`): it is numbered apart from the days, so its clip says "wins the battle" ("the final battle" for
  the finale) and calls to play, never "day 1" or "who wins day 2", and nobody is called up after it, since its
  podium scored its times.
- **`--season <path>`** is the season, a `seasons.v1.GetSeasonResponse`, served at `/__clip/season.json` the same
  way: when its Final Battle starts. With none, nobody is called up.
- **A story's key says which story it is, the headline never does** (`storyKeyOf`, `domain/clip/story.ts`): the
  flag the clip is about and the continent it is told on, `{flag: "dz", continent: "Africa"}`. **Its angle says what
  the story tells of the flag there**: none for what it did, `holding` for what it holds, `round` for a day it won,
  with the day (`round: "1.5"`, season and number; `"1.battle.2"` for a battle's round of its own), `callup` for a
  country called up after a day, with that day. The
  flag is the first of `sidesOf`: the loser of a rout, a continent's own flag when it strikes back together. The
  continent is the place's (Asia's parts as the headlines name them, `the world` for a tour), else its first
  country's. The job keeps it on each post, so the editor reads what was told, and a day is told once. **It keeps
  nothing else out**: the same key the next day is a new story when the facts say so, which the editor judges.
- **A story's countries are every country it shows** (`storyCountriesOf`): its place's, both sides and the flags it
  takes from, only the game's countries. The job counts them in what it posted: "VATICAN CITY VS PALAU" is Vatican's
  story and shows Palau. The `Clip`'s countries are the same list, with their hashtags.
- **`clip-<n>.json`** is a `Clip`: its number in the run (`--pick`), its title (the headline), its key, which the job
  keeps with the post, and **the words of its caption that need the story**: its teasers, its countries with their
  hashtags, its continent's hashtag, and the credit of its music. **The job writes the caption** from them (the
  backend's [The clips job](../../backend/docs/clips-job.md)): this app has the names, the job has the platform. The
  repo is public and so are a run's logs and artifacts: a clip can be watched there before it is out.
- **A `Clip` also says what the clip was** (`record`, a `ClipRecord`, written by `clipRecordOf` in
  `src/clip/clipRecord.ts`): the job keeps it with the post, never published, and the backend's `clips stats` sets
  it beside the post's numbers on TikTok and YouTube every day (`.github/workflows/clips-stats.yml`; the backend's
  [The clips job](../../backend/docs/clips-job.md)), to learn what makes a clip do well. **It records, and chooses nothing**:
  every value is one `record()` already worked out, read once the clip is decided, and the same replay and the same
  story give the same video, teasers and `Clip` byte for byte. **It is shaped to outlive this kind of clip**, since
  the clips will be experimented on:
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
  - **`sound`**: `anthem` or `trending`, the job's coin for each clip. Nothing to apply here: the video is the same in
    both, anthem included, and the job attaches a track trending on TikTok over it when it posts a `trending` clip,
    the anthem muted (the backend's [The clips job](../../backend/docs/clips-job.md), "Experiments"). The job's `fell-back`, a `trending` clip posted with its
    anthem when the chart could not be read, is decided after the render, so it never reaches this app, which refuses
    it like any arm the job does not draw.
  - **`length`**: `as-chosen`, the control, is exactly the length `pace.ts` gives (below); `short` is the shortest a
    story of its kind plays (`timelineOf(…, "short")`): 8.5s for a front, however far its fighting moves, 10 with a
    day's podium. **A tour is already its shortest**, each stop a flight and a stay, and is left as it is, as is a
    front whose fighting stays in one place: the arm is still recorded, and `durationSeconds` keeps the real length.
    The camera fits the play it is given either way.
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
  bomb, a spread or an enclosure as an effect. It numbers each step from its opening and counts what each flag
  holds from the borders' landmasses, as the server does (see [The counts are the server's](counts.md)), and the
  clip waits for `Globe.tilesRead` before its first frame. `ReplayBackend.cut` opens on the map at a later time.
- **The page measures, then records** (`window.clip`, `src/clip/main.ts`): `findings` is what the replay holds,
  known before any globe is built, and `record()` builds the globe for the story `--story` names and answers what
  the recorder needs. A story that names nothing in the findings fails `record()`, never `findings`.
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

**`src/domain/clip/` measures and plays; it judges nothing**, pure and under test:

- **`window.ts` finds the candidates**: for each length from 1 to 24 hours, the busiest stretch of a few places
  far apart, counting the tiles taken from another flag in each 10° cell and the eight around it. Filling empty
  ground is not war. **A clip is news**: with no window asked for, only the last 12 hours of the replay are searched
  (`newsSince`), since the busiest stretch of three days is rarely today's and would hide today's from the search;
  posted a day late, "Christmas Island is taking over the world" is about a flag already thrown out. A window asked
  for (`--since`, `--until`) is split into its places the same way, so a war next door is a story of its own. **Every
  scale is searched**: the whole map, and each of the 12 countries whose ground changed hands the most (400 tiles or
  more), on its own ground alone, as `--focus` does: on 2026-10-26 the 12-hour front of Western Europe reached Ankara,
  and Turkey's own front is how its war is seen whole.
- **`front.ts` finds the front** of a candidate: the point where most tiles changed hands, and every change within
  about 2,900 km of it, so a war in France brings in England, Spain and Germany. Two fronts within about 1,900 km on
  one scale are one, whatever window found them, the busiest kept (`sameFront`): Paris and Amsterdam, not Paris and
  Ankara. **Once the story is written, the front is found again
  from its own sides' fighting where it is told** (`fightersOf`, `inPlace`): the attacker's, both sides' of a battle,
  the loser's land going for a rout, every flag of the continent's for a team, in the country or the continent the
  story names; so a war next door (Israel in Turkey) does not pull the camera off Belgium's, and Germany and Romania
  trading Germany is told on Germany beside the war across Europe it is part of. A story whose sides fought nowhere in
  its place fails the render, and the job's log says so.
- **`findings.ts` measures a front, a tour and a place** (`frontFactsOf`, `tourFactsOf`, `placesOf`, above): who
  held what, who took what from whom, where. It holds the few numbers that say what is big enough to count (a country
  of 1,000 tiles is a place; the top 6 countries and flags of a front, the top 3 holders of a country) and none that
  says what is worth telling.
- **`story.ts` is what a clip tells, as the editor wrote it** (`Story`): its kind, its attacker, its rival, whom it
  took from, its place, and the continent striking back together (`team`). `sidesOf` is who the clip is between,
  `storyKeyOf` which story it is, `storyCountriesOf` what it shows, `fightersOf` whose fighting the camera follows.
  `src/clip/overlay.ts` words it: the headline is the editor's, and `wordsOf` writes the teasers, the call and the
  line of each kind, as before; its own headline stands in when none is given, as for a day won.
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
  DAY 5" and "DAY 5 GOES TO FRANCE". A teaser that says the editor's headline is left out, and a test checks that no
  kind ever gives its own headline back.
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
- **A holding's clip plays its flag's fighting for the place**, all over it, with the counter as the flag's share
  of it (`holdingStoryOf`, `inHolding`), and opens on all it holds there.
- **`round.ts` tells a day of the season** from the race (`--race`): **"FRANCE WINS DAY 5"**, or **"FRANCE WINS THE
  FINAL BATTLE"**, with where it leaves the winner in the season ("Leads the season with 86 points"). Only the run
  whose window holds the cutoff finds it, and the job tells it once, by its key. It plays the 3 hours before
  the cutoff, the winner's fighting (anyone's when it fought none), opens on all the winner holds, counts its tiles
  (the podium's points say who won: a day is won by the ground held at its cutoff), and ends on the podium, the places that scored with their points, then "WHO WINS DAY 6?". The ending
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
- **`score.ts` ranks the fronts**: the tiles taken from another flag, over the square root of the hours, times the
  countries they were taken in (up to 4). A short war over several countries beats a long filling of one. It orders
  the findings and caps them; the editor reads it as one number among the others.
- **`solidity.ts` measures graffiti.** For each tile the attacker took and holds at the end, the share of its 6
  neighbours it holds too: about 1 for land taken, 0.56 for names written across Canada. It is a fact of each
  flag of a front, which the editor is told plays badly; nothing is skipped for it.
- **A flag taking land across the world is on a tour of it** (`tour.ts`, for a `tour` of the findings): told over
  the whole globe as **"PORTUGAL IS TAKING OVER THE WORLD"**, its call "Fight back", with no flag. **A stop is framed
  on the landmasses whose painted flag switched to the flag** (`flippedTo`), and only what it took on them counts as a
  visit, since that is all the tour's zoom shows: on 2026-10-27 "BRAZIL IS TAKING OVER THE WORLD" spent half its length
  over France, where Brazil had taken 223 tiles under other countries' big flags, while the pieces it had switched were
  Peru and Guyana, and a player saw flashes and no Brazil. A tour with no such landmass fails the render. Among the
  visits, each continent (Asia's parts) with 5% of them is a stop, four at most, in the order it got there (a fifth of
  it taken).
  Each stop is framed where it took the most there (Angola, not all of Africa from Guinea-Bissau to Mozambique, which
  only the whole globe frames). The camera opens at the first stop, twice as close as that framing but never past
  where the painted flags start to blend into the tiles (`flagsZoomOf`): the flags changing hands are what a tour
  shows, and they change from the first second. At each stop it follows the flag's fighting there as it happens, as a
  clip follows its front, then flies to the next, rising a little over a long way (1.6 times out at most), never out
  to the globe and back in, which was brutal. It pulls back out to the globe over all of them at the end; 1.2s a
  flight and 2.6s a stop. Up close, a big country's painted flag is blurred: the flag atlas is the game's. The counter
  counts the flag across the world.
- **`look.ts` picks when the camera comes back out of the tiles.** From far, a landmass's painted flag only
  changes when its biggest holder does (`flipsOf`, over the borders blob, read 8 times along the changes, so a
  landmass taken and taken back counts too). A front too wide to frame closer than
  `TILES_ZOOM` (3) is **flags** when at least 2 landmasses changed their biggest holder and those hold 2,000 tiles
  or more: a steamroll, which comes back out halfway so its painted flags change on screen. Everything else is a
  **dive**, which comes back out at the end.
- **`camera.ts` opens down in the tiles**: the first frame is where the most tiles change hands, past the zoom the
  painted flags are gone at (`tilesZoomOf`), so the fight is seen tile by tile from the first second. Until
  2026-10-28 a clip opened on the map with its painted flags, held 0.3s and dived in 0.7s: a second with nothing
  moving up close, while 70 to 90% of TikTok viewers left before the end. It follows the densest fighting, and comes
  back out as the look says to the overview: the middle of the front's changes, zoomed until 95% of them fit but
  never closer than a continent (`overviewOf`), the map with its painted flags. **Every clip ends pulled back out**,
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
  go**: 6s for a fight in one place, however many hours it lasted (room for the close-ups and the pull back out), and 1.2s more for every screen the fighting
  crosses up close (`screensOf`, on the camera's smoothed path), up to 15s. A bomb adds nothing. The last 2.5s are the call to act, and nothing runs past 22s.
  So a front runs 8.5 to 17.5s (10 to 19 with a day's 4s podium) and a tour 6.5 to 19, by its stops; only `--seconds`
  reaches 22. The experiment `length` tries the shortest of each against it (above).

**`scripts/clip/render.mjs` is the recorder**: it starts Vite, serves the replay at `/__clip/replay.json`, opens
headless Chrome at 540×960 at 2×, waits for `window.clip.record()`, then for each frame calls `window.clip.frame(i)`,
takes a screenshot and pipes it to ffmpeg (libx264, `yuv420p`). It needs ffmpeg and Chrome (`CHROME_PATH`, which
the workflow sets: the default is the macOS path).

**The screenshot is nearly all of a clip's time, so several browsers take them in turn** (`--recorders`, one for
every 3 cores, 4 at most). The globe draws a frame in 6ms; Chrome takes 0.3s to encode it as a 2.4MB PNG, on one
core. Each recorder is a Chrome of its own that plays every frame, so all hold the same map, and takes the picture
of its own only: frames 0, 4, 8 from the first, 1, 5, 9 from the second. They reach ffmpeg in order, and they are
the frames one recorder would take, since the page's clock and dice are the recorder's (above). On 12 cores a 12s
clip takes 40s instead of 90s, and a fifth recorder adds nothing. Each one holds the whole replay, 1.8GB for 72
hours. The 14s before the first frame are the page reading the replay and measuring it, once per clip.
