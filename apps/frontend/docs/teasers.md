# Battle teasers, as posts

**A battle's teaser goes out on TikTok and YouTube about a day before the battle** (2026-11-09): its rule set's cards,
the same the home page shows in its carousel (see [Battles](battle.md)), each in a frame that says when the battle
starts. On TikTok the cards are **a photo carousel**, the teaser's own form; on YouTube, where a Short is a video,
**a slideshow of them**. `npm run teaser` makes both from one teaser; the backend's promo module chooses the battles,
writes the words, plans the posts and posts them (its [Battle teasers on TikTok and
YouTube](../../backend/docs/teasers-job.md)). **This app keeps only what needs the browser**: drawing the frames.

```bash
npm run teaser -- --teaser teaser.json --out teaser/   # card-1.jpg … card-N.jpg, teaser.mp4, and teaser.json beside them
npm run teaser -- --teaser teaser.json --seconds 4     # 4 seconds a card in the slideshow (3 by default)
npm run teaser -- --teaser teaser.json --headed        # watch Chrome take the stills
```

- **`--teaser`** is a `promo.v1.Teaser` in JSON, as the backend's `promo.v1.AdminService/ListDueTeasers` answers it
  and its job writes it (`go run ./cmd/teasers due -out <folder>`): the battle's id, start, end, the schedule's words, its
  points' times, whether it is the Final Battle, and its cards with their words filled and their backgrounds' public
  addresses. The job reads the id back from the copy the renderer leaves beside the stills.
- **The frame is a page of its own**, `teaser.html` (`src/teaser/main.ts`, `teaser.css`), served by Vite and **left out
  of the build**, as `clip.html` is (see [Clips](clips.md)). 540×960 CSS pixels, taken at a scale of 2: 1080×1920,
  the size of a TikTok photo and of a Short. From top to bottom: the hazard tape; the battle's name ("Battle", or
  "Final Battle" smaller so it holds on one line), its day and its time **in UTC** (`domain/teaserFrame.ts`,
  `teaserFrameOf`: "Saturday 14 November", "20:00 UTC", whatever the machine's zone); **the game's own card**
  (`.teaser-card` from `app/battle/Battle.css`, 440px wide, its words scaled to it); a dot a card, the shown one in
  the war's orange; and `clickplanet.lol` in gold. Every color and font is a token of `tokens.css`, the display font
  Luckiest Guy and Rubik under it, as in the game.
- **`scripts/teaser/render.mjs` takes the stills**: Vite on 127.0.0.1:5198 serves the page and the teaser at
  `/__teaser/teaser.json`; Chrome (`CHROME_PATH`), driven over the DevTools protocol with no other dependency, opens the
  page, which preloads every background and waits for the fonts before it says it is ready (`window.teaser`); then
  each card is shown (`window.teaser.show(i)`), decoded, and taken as a JPEG of quality 92. **JPEG, not PNG**: TikTok's
  photo posts take JPEG and WebP. A background that does not load fails the page (`window.teaserFailed`), and so the
  render: that battle is not posted, and the next run tries again.
- **The slideshow** is ffmpeg's: each still for `--seconds`, H.264 in yuv420p with `faststart`, at 1080×1920, with a
  silent AAC track. TikTok adds its own music to the carousel (`autoAddMusic`, the backend's
  [Publishing](../../backend/docs/publishing.md)).

**Every 6 hours a workflow posts the battles due** (`.github/workflows/teasers.yml`, at 43 past 01:00, 07:00, 13:00
and 19:00 UTC, or by hand): `due` asks promo over SSH which battles are due and writes a folder each, in a job that runs
no npm; `render` renders each in a job of its own, side by side (Chrome and ffmpeg on the runner, a few seconds a
card); `post` uploads each render and asks promo over SSH to post it, again with no npm. The VPS key is in no job that
installs this app's dependencies, as in the clips' workflow.

## Tests

`teaserFrame.test.ts` pins the frame's words: the name of a battle and of the Final Battle, the day and the time in
UTC from a machine in Los Angeles too. The page and the script are checked by eye on a render: the run's artifacts hold
every still.
