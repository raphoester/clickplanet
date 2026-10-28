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
- [Styling](docs/styling.md) — `DESIGN.md` is the design system; how to debug a mobile layout.
- [Protocol Buffers](docs/protobuf.md) — types from the shared `/proto`, generated into `src/gen/grpc`.
- [Static assets](docs/static-assets.md) — the content-addressed files in `static/`, and how each is made.
- [Pages and routes](docs/routes.md) — the two pages Vite builds, and their routes.

## Architecture

- [Architecture](docs/architecture.md) — the layers: `domain/`, `backends/`, `app/viewer/`, `app/`.
- [`src/domain/`](docs/domain.md) — the logic worth testing: no GPU, no DOM, no React.
- [`src/backends/`](docs/backends.md) — the contracts and the one Connect transport; the click budget; a frozen
  map; connection health.
- [The screen: four zones](docs/screen.md) — nothing is drawn over the globe but four zones.
- [Data flow](docs/data-flow.md) — from `createGlobe` to the screen, step by step.
- [Sessions](docs/sessions.md) — the click token the backend gates `Click` on.
- [Sign-in](docs/sign-in.md) — optional sign-in with Google, Discord or an emailed code; usernames.
- [The counts are the server's](docs/counts.md) — every count on screen comes from `GetHoldings` and
  `holdings_changed`; one version rule for every tile value; the load and what is read again.
- [Rolling back a refused click](docs/click-rollback.md) — a click is painted first and undone if the server
  refuses it.
- [Sampling the leaderboard](docs/leaderboard-sampling.md) — how the ranks are read without a render on every
  batch.

## Features

- [Live chat](docs/chat.md) — replies, removed messages, reactions, announcements, delivery, and what was missed.
- [Who is playing](docs/who-is-playing.md) — the "Online · N" list beside the chat.
- [Who holds a tile](docs/tile-holder.md) — a rest or a long press on a tile opens a card on its holder.
- [A territory, from orbit](docs/territory-card.md) — zoomed out, the same gestures open a card on the territory
  (flags, shares, fortify) and light it up; a click flies down to it.
- [Support](docs/support.md) — suggestions, bug reports and reports of a player.
- [Titles](docs/titles.md) — the ranks on each track, the medals, the unlock moment.
- [Season standings](docs/season-standings.md) — the players' table of the season.
- [The season](docs/season.md) — the season chip and its clocks, and the home page's countdowns.
- [Battles](docs/battle.md) — the schedule's stream, the countdown to the next battle or the Final Battle, and the
  battle look: the mood, the chip and its clock, the start moment, the debrief's three screens when the gamemaster
  debriefs its round, matched by the round's id, and the announcements queue that shows one full-screen moment at a
  time.
- [The cutoff, revealed](docs/cutoff.md) — the full-screen reveal when a day closes, and the season's results after
  the Final Battle's debrief.
- [The flag played for](docs/flag.md) — `PledgeAllegiance`, and one flag an hour.
- [Charges](docs/charges.md) — what a bonus box holds, off by default and one at a time, and the inventory.
- [Missions](docs/missions.md) — the Missions tab, its badge, a mission paid, and the fake.
- [Quizzes](docs/quizzes.md), [Bombs](docs/bombs.md), [Shields](docs/shields.md), [Fortify](docs/fortify.md) —
  each bonus on the client.
- [Sharing the globe](docs/sharing.md) — the photo card of the globe, and the recruit link.
- [Clips](docs/clips.md) — `npm run clip`: a replay played back as a short video, and what it hands to the
  backend's clips job.
- [In a clip](docs/clip-appearances.md) — "You are in a clip!" with links to watch it, and the switch that keeps
  a name out of the clips.
- [Sound](docs/sound.md) — every sound built with Web Audio, no audio file; the leader's anthem.

## The globe (`src/app/viewer/`)

- [The GPU layer](docs/viewer.md) — `globe.ts` and the files around it: marks, glints, passes.
- [Drawing only when something changed](docs/render-on-demand.md) — the loop draws a frame only when needed.
- [The far side of the globe is not drawn](docs/far-side.md) — the half behind the earth is skipped.
- [HD graphics](docs/hd-graphics.md) — the sharper, lit globe, and `?gfx=`.
- [What a tile costs on the GPU](docs/tile-layout.md) — the plain and the compact tile layouts, and the one
  shared earth sphere.
- [CSS pixels in, drawing-buffer pixels out](docs/pixel-ratio.md) — the pixel ratio, capped at 2.
- [The light](docs/light.md) — the earth's light, with HD graphics only.
- [The zoomed-out view](docs/zoomed-out-view.md) — how tiles sample the flag atlas when they are small.
- [The globe's texture](docs/globe-texture.md) — the sphere under the tiles, cut from the tile field.
- [Tile positions](docs/tile-positions.md) — rebuilt in a worker from a 12 KB landmask and checked by a hash;
  the coordinates blob is only the fallback.
- [The countries' outlines](docs/country-outlines.md) — the border lines, the same width at every zoom.
