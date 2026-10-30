# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**The operator's page, at `backoffice.clickplanet.lol`.** It calls the backend's `AdminService`s. It is a third
app: it imports nothing from `apps/frontend`. This file is only an index (see the root [Docs rule](../../CLAUDE.md#docs-rule)):
each line names a doc in [`docs/`](docs) and says what is in it.

## Working on it

- [What this is](docs/overview.md) — the pages, by menu group, and what is left of the plan.
- [Commands](docs/commands.md) — `npm run dev` (port 5174), `demo`, `build`, `test`, `lint`, `proto`, `map`.
- [How it reaches the API](docs/api.md) — `src/api.ts`, the backend's admin listener, sign-in with Google, the
  `cp_admin` cookie.
- [Running it locally](docs/running-locally.md) — `npm run demo` on a made-up backend in the page; or the backend
  config it needs: auth on and an admin listener.
- [Deploy](docs/deploy.md) — the `clickplanet-backoffice` Cloudflare Worker.
- [Layout](docs/layout.md) — the `src/` tree, one line per file.

## The pages

- [The menu](docs/menu.md) — a burger drawer below 1024px, a sidebar from 1024px; the page and the report or player
  open on it in the address bar.
- [Every action, on every page](docs/actions.md) — every change asks first, then says what the server answered.
- [The home page](docs/home.md) — how the game is doing, counted and charted on the page from raw reads.
- [Who sent a note](docs/note-sender.md) — every note comes from an account.
- [A note's photos and videos](docs/note-media.md) — how a note's files are shown.
- [Acting on a report](docs/reports.md) — the Reports inbox: the list with what was said, one report, one verdict that
  acts and closes it, then the next.
- [The player page](docs/player-page.md) — one search box, a sortable table, and one player: where it stands, every
  tool on it.
- [The map tools](docs/map-tools.md) — reassign a country, paint random tiles, download a replay.
- [Clip stats](docs/clip-stats.md) — the daily numbers of the posted clips, and a CSV of them.
- [Videos](docs/videos.md) — the season-end videos: uploaded from the Mac, downloaded with their caption on a phone.
- [Ops](docs/ops.md) — the Timeline: what the gamemaster does next, drawn from its reads; `ReconcileTitles`, after a
  deploy; Game rules: planet's sets, made and edited; the antibot's rule sets, in a form read off the proto; both show
  what a save changes; the rule schedule: which set plays when, a battle's points' times, entries to come added and
  changed, and each season's Final Battle among them; the seasons: their breaks, lengths and Final Battles, added,
  moved and deleted.
