# Commands

```bash
npm run dev        # Start Vite dev server (binds to 0.0.0.0:5173)
npm run build      # CrazyGames' word list, TypeScript check, Vite bundle → /dist
npm test           # Run the vitest suite once
npm run test:watch # Re-run affected tests on change
npm run lint       # ESLint check
npm run proto      # Regenerate protobuf types from the shared ../../proto/ using buf CLI
npm run atlas      # Repack the flag sprite atlas from static/countries/png100px
npm run map        # Copy the shared /map coordinates blob into static/ (see static-assets.md)
npm run map:generate # Rewrite both shared map blobs from the ground oracle (see ../../map/README.md)
npm run map:audit  # Check the tile set, Natural Earth and the globe texture against each other
npm run quiz:generate # Rewrite the shared quiz bank (see ../../quiz/README.md). The backend embeds it; this app never does
npm run borderLines # Trace the countries' outlines onto the tile lattice (see country-outlines.md)
npm run landmask   # Write which lattice points are tiles, from the coordinates blob (see tile-positions.md)
npm run earth      # Cut the globe's texture from the tile field (see globe-texture.md)
npm run flagFit    # Work out which flags stretch, and where each one is cropped
npm run mobile     # Screenshot/inspect a URL as a phone (see styling.md)
npm run clip -- --replay replay.json --findings findings.json  # What the replay holds for the editor, as data (clips.v1.Findings), and no clip
npm run clip -- --replay replay.json --story '{"front":3,"kind":"battle","attacker":"fr","rival":"tr","countries":["tr"],"headline":"FRANCE VS TURKEY"}'  # One clip of a story of the findings, as the job's choose prints it
npm run clip -- --replay replay.json --race race.json --story '{"seasonal":1}' --plan  # With the season's race: the day it closed is a story too
npm run clip -- --replay replay.json --race race.json --season season.json --findings findings.json  # And the countries called up after it
npm run clip:anthems # Vendor the anthems only the clips play (Europe's, Palestine's) into scripts/clip/anthems
npm run clip:highlights # Measure where each anthem a clip plays is worth starting (run by both anthem scripts)
npm run regions    # Rewrite each country's continent (Asia's part of it) from Natural Earth, for the clips' headlines
npm run landmassNames # Name every landmass but each country's main one, from Natural Earth (see fortify.md)
npm run crazygames:words # Fetch CrazyGames' word list into public/crazygames/words.txt (see crazygames.md)
```

`.github/workflows/check-frontend.yml` runs lint, build and tests on every PR
touching this app.

**`npm run dev` cannot reach the production API.** `api.clickplanet.lol` sends
`access-control-allow-origin: https://clickplanet.lol` and nothing else, so the
browser blocks every request from `localhost`. Point `VITE_API_BASE_URL` at a
local backend, or run `VITE_FAKE_BACKEND=1 npm run dev` to play against
`FakeBackend` and `FakeChatBackend` — the fake serves a full map of the real tile
count, with a version on every write and the counts the server would send,
simulates other players at a few clicks a second, and reproduces every refusal the
chat can show.
The switch is gated on `import.meta.env.DEV` as well, because an unset `VITE_*`
variable is **not** folded away in a build: without the `DEV` check both fakes
ship in the production bundle.

`VITE_CRAZYGAMES=1` makes the page the CrazyGames page on any host, with the
SDK's local mode on `localhost` (see [CrazyGames](crazygames.md)). The deployed
build never sets it: `crazygames.clickplanet.lol` is the same build. `npm run dev` does not fetch the word list:
run `npm run crazygames:words` once, or every word on that page is masked.

Every tile starts French in the fake, and it keeps shields on its tiles as the
server does: in its map batches, its tile updates and its bombs' struck tiles.

In fake mode the console has a few commands: `giveBomb()` puts a bomb in the
inventory as if a box holding one had just been caught, `giveBonus("refill")` does
the same for any other bonus (the fake holds charges as the server does: a refill
and a bomb at most, a pool of 8 spread clicks, a stack of 3 enclosures and 30
shields, a box adding 1 to 4, 1 to 3 and 5 to 20 of them, spread and enclose
spent only while switched on,
both at once refused, a refill refused on a full bank), `giveQuiz()` puts a quiz
banner up at once, `giveTitle("warlord")` plays the unlock of any title (the fake
account wears nothing, so Wear it only closes it), `closeDay()` closes the
fake's day: scoring scores today's round, its countries with a little luck, the
fake gamemaster debriefs it and the next day runs, and the chat announces the
podium, so the cutoff plays (see [The cutoff, revealed](cutoff.md)). Each call
closes the next day, so the reveal plays again.
`startBattle(minutes = 120, description, times = 3)` plays a battle at once, under a
new entry id: the mood, the halo, the chip, the start moment, and the battle's round
at `times` in scoring, so the board's column is the battle's (see [Battles](battle.md)).
`scheduleBattle(inMinutes = 90, minutes = 120, description)` adds a battle to come
and answers its id, `moveBattle(id, inMinutes)` moves it and `deleteBattle(id)`
deletes it, as an operator's edit does: the chip's and the home page's "Next battle
in" follow at once. `scheduleFinale(inMinutes = 90, minutes = 120)` gives Season 0 a
finale: its end moves to after it, and the countdowns say "Final Battle in".
`startFinale(description, times = 3)` plays the finale at once, to the season's end,
under its round `season-0/finale`: the chip and the start moment say "Final Battle". `moveSeasonEnd(inMinutes)` moves Season 0's end, as an operator's
edit of the season does: every season countdown follows (the home page has these four alone).
`endBattle()` ends the turn and leaves the battle's round undebriefed, so the line says
the results are coming. `closeBattle()` scores the round with its numbers (its points at
its times, the tiles each flag took, its top players, drawn with a little luck) and the
fake gamemaster debriefs it, so the debrief plays; with no battle started it scores one
of 2 hours at ×3 that ends now, under an id the page never saw, so it plays no debrief
once the page saw a battle. `closeFinale()` debriefs it as the Final Battle, then the
season: the debrief, then the season's results, which end on the champion; the chat
says who won, and the season chip becomes the champion's. `fakeBackend.botBomb(tile, "fr")`,
`fakeBackend.botSpread(tile, "fr")` and `fakeBackend.botShield(tile, "fr")` play
somebody else's bomb, spread click or shield. `fakeBackend.takeLandmass(landmass, "es", leave)`
takes a landmass for a flag, shields and all, and leaves the last `leave` tiles to
click by hand (it answers them); the fake fortifies as the server does, and every
landmass starts locked to France. `fakeBackend.shareClicks("guests")` (or
`"network"`) reads the bucket as shared, and `fakeBackend.unlockFlag()` ends the
wait for another flag; `lockFlags()` locks the flags as a battle's rule set does, and `openFlags()` opens them (see
[The flag played for](flag.md)), and `fakeBackend.shareClicks()` as the
player's own again. `fakeBackend.closeShapes(false)` makes every enclose click
close nothing, for the bubble that says so, and `closeShapes(true)` puts it back.
`fakeBackend.ban(90)` refuses this player's clicks, bombs and shields for 90
minutes, as a ban does, and `fakeBackend.ban(0)` lifts it.
`fakeBackend.setTempo(3)` refills the bank three times as fast, as a rule set
of the server can, and `fakeBackend.setTempo(1)` puts it back.
`offerSignIn()` offers the fake guest Google, Discord and email, so the sign-in pitch and the nudge show on an
empty bank (the buttons are refused: the fake has no flow).
`signIn("Nova")` signs the fake account in, with that username (see [Sign-in](sign-in.md)). From a guest it is a
first sign-in: the fake grants a refill a second later, and the game spends it. And
`giveClip("France vs Turkey")` posts a clip that names it: `signIn()` reads it at once, so the notice
plays (see [In a clip](clip-appearances.md)). `localStorage.removeItem("clickplanet-clips-shown")`
makes the clips shown new again.
`localStorage.removeItem("clickplanet-bonus-guide")` makes the bonuses new again.
`fakeSupport.refuse("rate-limited")` (or `"rejected"`, `"no-session"`,
`"reported-already"`, `"failed"`) refuses what Settings and a player's card
send next, `fakeSupport.refuse("upload-failed")` fails each file half way up,
and `fakeSupport.refuse()` takes it all again; `fakeSupport.sent` is what it
took, with the ids of each note's files, and `fakeSupport.uploaded` the files
behind those ids (`fake-file-<n>` is its place there).
`fakeSupport.rules.maxPerNote = 0` is a server that takes no file.

A local backend is the quickest way to exercise the real chat: `cmd/api`'s
`example.yaml` runs chat (it is always on), and the Go server answers
`Access-Control-Allow-Origin: *` itself, so `VITE_API_BASE_URL=http://localhost:8080
npm run dev` works with no proxy in between.

Docker (only for the local full stack in `deploy/`; production is Cloudflare Pages):
```bash
npm run dBuild   # Build the image as clickplanet-front:local
```

The image is self-contained — `nginx.conf` is baked in and mirrors what the
deployed site does: the caching rules from `public/_headers`, the game at
`/play` and `/auth/callback`, and the fallback to the home page that
`wrangler.jsonc` sets with `not_found_handling`.
