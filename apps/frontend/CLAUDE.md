# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Multiplayer globe conquest game**: players click tiles on a 3D globe to claim them for a country. Plain
Three.js driven imperatively, not React Three Fiber, under a React UI. This file is only an index (see the root
[Docs rule](../../CLAUDE.md#docs-rule)): each line names a doc in [`docs/`](docs) and says what is in it. Read
the doc before you change its area, and read [`DESIGN.md`](DESIGN.md) before you touch a style.

## Working on it

- [Commands](docs/commands.md) — `npm run dev`, `build`, `test`, `lint`, `proto`, and the map, atlas, texture and
  clip scripts. The production API refuses `localhost`: use a local backend or `VITE_FAKE_BACKEND=1 npm run dev`,
  and the fake's console commands (`giveBomb()`, `closeDay()`, …).
- [Testing](docs/testing.md) — vitest, co-located, on node by default; jsdom only where a file asks for it.
- [Styling](docs/styling.md) — `DESIGN.md` is the design system; `ScrollArea`, the scrolling body of every panel,
  and its fade; a modal's padding; a box as wide as its word in nine languages; how to debug a mobile layout.
- [Protocol Buffers](docs/protobuf.md) — types from the shared `/proto`, generated into `src/gen/grpc`.
- [Static assets](docs/static-assets.md) — the content-addressed files in `static/`, and how each is made.
- [Translation](docs/i18n.md) — the catalogue (`src/i18n/locales/en.json` and one file per language), whole
  sentences with slots and ICU plurals, `t`/`rich`/`translator()`, `domain/` returns facts, Intl for every number and
  name, "in" a country as a phrase per language ("au Brésil"), how the language is chosen (pick, CrazyGames,
  browser), adding a string or a language.
- [Pages and routes](docs/routes.md) — the five pages Vite builds, and their routes; a flag's address,
  `clickplanet.lol/<flag>/<name>`, which the home page's first script opens on the game.
- [Search](docs/search.md) — what Google reads: the titles, the one description, the JSON-LD that names us
  "Click Planet" too, the sitemap, the CrazyGames host's noindex; the Search Console baseline of Nov 2026 and how to
  read it with Composio.

## Architecture

- [Architecture](docs/architecture.md) — the layers: `domain/`, `backends/`, `app/viewer/`, `app/play/`, `app/`; the one
  `Backends` object `main.tsx` gives the page.
- [`src/domain/`](docs/domain.md) — the logic worth testing: no GPU, no DOM, no React.
- [`src/backends/`](docs/backends.md) — the contracts and the one Connect transport; the click budget; a frozen
  map; the offers stream beside planet's, for the boxes and quizzes; connection health.
- [A server restart reloads the page](../../docs/reload-on-restart.md) — `restartInterceptor` reloads once the
  backend answers under a new boot; which answers it trusts.
- [The screen: four zones](docs/screen.md) — nothing is drawn over the globe but four zones; the phone form at
  768px wide or 600px high and under; the board and the chat folded under 1600px wide until the player opens one, and
  the player's last fold kept.
- [Data flow](docs/data-flow.md) — from `createGlobe` to the screen, step by step.
- [Stores](docs/stores.md) — a feature's state in its own Zustand store, read by the component that shows it, not
  held in `PlayScreen`; never the tiles; the stores the globe's notices go to, and the news line, one at a time.
- [Sessions](docs/sessions.md) — the click token the backend gates `Click` on.
- [Sign-in](docs/sign-in.md) — optional sign-in with Google, Discord or an emailed code; usernames; deleting an account,
  and the welcome back when a sign-in within 30 days brings it back.
- [Arrivals](docs/arrivals.md) — the first page's `?s=`, referrer host and recruiter's account (the name in a
  player's address, read with `GetPlayerByName`), kept and sent with the calls that make an account.
- [CrazyGames](docs/crazygames.md) — the same build on `crazygames.clickplanet.lol`: CrazyGames' sign-in alone, the
  session in a header, a deleted account the next start does not bring back, its settings, the world as one room
  with invites, no ad, and its word list masked in the chat and the names.
- [The counts are the server's](docs/counts.md) — every count on screen comes from `GetHoldings` and
  `holdings_changed`; one version rule for every tile value; the load and what is read again.
- [Rolling back a refused click](docs/click-rollback.md) — a click is painted first and undone if the server
  refuses it.
- [Sampling the leaderboard](docs/leaderboard-sampling.md) — how the ranks are read without a render on every
  batch.

## Features

- [Live chat](docs/chat.md) — the World and the flag's room, GIFs from KLIPY, replies, removed messages, reactions,
  announcements, delivery, and what was missed.
- [The front, for a first visit](docs/front-guide.md) — a browser's first visit opens on its own country, flies to
  its flag's nearest fight once the tiles are read, and rings the tiles it can take, until it has taken 50.
- [The reach](docs/reach.md) — a rule knob: the bridges drawn as dotted arcs, the tiles the flag can take (next to its
  land, or across a bridge from it) ringed, wider by the spread's rings while spread is on, a click elsewhere stopped
  with "Take a tile next to your land"; learnt from `GetChargeRules`.
- [The capitals](docs/capitals.md) — two rule knobs: each capital still standing marked with a crown, no shield on a
  capital, a knock-out cleared in one event, and the out window that asks for a flag still in.
- [Ghosts, for a quiet first visit](docs/ghosts.md) — while fewer than 7 play, a guided page sees fake raids near it,
  painted on that page only, kept in its storage, read by the board and the front, and taken back with no click.
- [Who is playing](docs/who-is-playing.md) — the "Online · N" list beside the chat.
- [Who holds a tile](docs/tile-holder.md) — a rest or a long press on a tile opens a card on its holder.
- [A territory, from orbit](docs/territory-card.md) — zoomed out, the same gestures open a card on the territory
  (flags, shares, fortify) and light it up; a click flies down to it.
- [Support](docs/support.md) — suggestions, bug reports and reports of a player.
- [Titles](docs/titles.md) — the ranks on each track, the medals, the unlock moment.
- [Nameplates](docs/nameplates.md) — a player's name, color and crown, read by account id from one store that batches,
  knows the roster and the page's own name at once, and holds a chat message until its names are read; the readers.
- [Badges](docs/badges.md) — a name's streak and worn title, read by account id from one store that batches,
  keeps them 5 minutes and takes the page's own changes at once; the readers.
- [Season standings](docs/season-standings.md) — the players' table of the season.
- [The season](docs/season.md) — the season chip and its clocks, the next battle and the Final Battle to come in its
  details, the Final Battle's look, and the home page's countdowns and Final Battle panel.
- [Battles](docs/battle.md) — the schedule's stream, the countdown to the next battle or the Final Battle, the
  teaser of a battle to come with its rule set's cards in a carousel, its rules a press away in the chip's details
  ("See the rules", the teaser with no entrance), and the
  battle look: the mood, the chip and its clock, the start moment, the debrief's three screens when the gamemaster
  debriefs its round, matched by the round's id, and the announcements queue that shows one full-screen moment at a
  time.
- [Nudges](docs/nudges.md) — the next battle's teaser and the asks (coffee, TikTok, YouTube) in the announcements
  queue: moments first, the teaser before any ask, one ask each 12 hours in turn, none on a first visit, in a battle
  or on CrazyGames.
- [The Hall of Fame](docs/hall-of-fame.md) — the board's crown (a gold button on a phone), and the modal's carousel of seasons: each
  ended one's winner flag, then the "?" of the one that runs; View opens a season's slides: the podium and the whole
  table, the race (each flag's points or map share round by round, drawn as a timelapse), players, the Final Battle,
  the numbers.
- [A season's map, browsed](docs/season-map.md) — `/archive`: the season or its Final Battle replayed on the globe
  with a time cursor, and the heatmap of where the tiles were fought over, a map that goes round.
- [The cutoff, revealed](docs/cutoff.md) — the full-screen reveal when a day closes, and the season's results after
  the Final Battle's debrief.
- [The flag played for](docs/flag.md) — `PledgeAllegiance`, one flag an hour, and the lock a battle's rule set puts
  on it.
- [Charges](docs/charges.md) — what a bonus box holds, off by default and one at a time, and the inventory.
- [Missions](docs/missions.md) — the Missions tab, its badge, Go, the missions drawn on the map (a ground outlined, a
  rival's tiles ringed in red), the progress line, a mission paid, and the fake.
- [Quizzes](docs/quizzes.md), [Bombs](docs/bombs.md), [Shields](docs/shields.md), [Fortify](docs/fortify.md) —
  each bonus on the client.
- [Sharing the globe](docs/sharing.md) — the photo card of the globe, and the recruit link; every share carries the
  player's address, `clickplanet.lol/<flag>/<name>`, or the flag alone for a guest.
- [Player videos](docs/player-videos.md) — "Make my video": the flag's 24 hours as an 8 s video made in the browser,
  its story, names and address, the anthem copied in, the red dots, the moments that offer it, the welcome line.
- [Clips](docs/clips.md) — `npm run clip`: a replay played back as a short video, and what it hands to the
  backend's clips job.
- [Battle teasers, as posts](docs/teasers.md) — `npm run teaser`: a battle's cards in a frame that says when it
  starts, as the stills of a TikTok carousel and a slideshow for YouTube, which the backend's promo module posts.
- [In a clip](docs/clip-appearances.md) — "You are in a clip!" with links to watch it, and the switch that keeps
  a name out of the clips.
- [Sound](docs/sound.md) — every sound built with Web Audio, no audio file; the leader's anthem.
- [Discord pings](docs/discord-alerts.md) — "Join Discord" in the chat's Discord bar, and "Get pinged" in the moment
  once a page when the flag is under attack, are a Discord sign-in, which joins the game's server; the moment's "Go
  there", for every player, flies to the attack; "Open Discord" once in; no plain invite anywhere, the plain pages
  link `/play?discord=join`; a ping's `?lat=&lon=&zoom=` starts the camera on the attack.

## The globe (`src/app/viewer/`)

- [The GPU layer](docs/viewer.md) — `globe.ts` and the files around it: marks, glints, passes.
- [Drawing only when something changed](docs/render-on-demand.md) — the loop draws a frame only when needed.
- [The far side of the globe is not drawn](docs/far-side.md) — the half behind the earth is skipped.
- [HD graphics](docs/hd-graphics.md) — the sharper globe, and `?gfx=`.
- [What a tile costs on the GPU](docs/tile-layout.md) — the plain and the compact tile layouts, and the one
  shared earth sphere.
- [CSS pixels in, drawing-buffer pixels out](docs/pixel-ratio.md) — the pixel ratio, capped at 2.
- [The light](docs/light.md) — one pass for every player: the season-end clips' light, rim and halo, on a navy
  sky.
- [The zoomed-out view](docs/zoomed-out-view.md) — the flags painted in each landmass's frame when tiles are small,
  pixel by pixel by the territory layer under the tiles: Split, Big or Off.
- [The globe's texture](docs/globe-texture.md) — the sphere under the tiles, cut from the tile field.
- [Tile positions](docs/tile-positions.md) — rebuilt in a worker from a 12 KB landmask and checked by a hash;
  the coordinates blob is only the fallback.
- [The countries' outlines](docs/country-outlines.md) — the border lines, the same width at every zoom.
- [A finger iOS took for its swipe back](docs/stray-fingers.md) — the edge swipe that never ends its pointer, the
  pinch OrbitControls was left in, and the `pointercancel` sent for iOS when the touch count says the finger is gone.
