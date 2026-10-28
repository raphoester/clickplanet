# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**The operator's page, at `backoffice.clickplanet.lol`.** It calls the backend's `AdminService`s. It is a third
app: it imports nothing from `apps/frontend`. This file is only an index (see the root [Docs rule](../../CLAUDE.md#docs-rule)):
each line names a doc in [`docs/`](docs) and says what is in it.

## Working on it

- [What this is](docs/overview.md) — the pages, by menu group, and what is left of the plan.
- [Commands](docs/commands.md) — `npm run dev` (port 5174), `build`, `test`, `lint`, `proto`, `map`.
- [How it reaches the API](docs/api.md) — `src/api.ts`, the backend's admin listener, sign-in with Google, the
  `cp_admin` cookie.
- [Running it locally](docs/running-locally.md) — the backend config it needs: auth on and an admin listener.
- [Deploy](docs/deploy.md) — the `clickplanet-backoffice` Cloudflare Worker.
- [Layout](docs/layout.md) — the `src/` tree, one line per file.

## The pages

- [The menu](docs/menu.md) — a burger drawer below 1024px, a sidebar from 1024px.
- [Every action, on every page](docs/actions.md) — every change asks first, then says what the server answered.
- [The home page](docs/home.md) — how the game is doing, counted and charted on the page from raw reads.
- [Who sent a note](docs/note-sender.md) — every note comes from an account.
- [A note's photos and videos](docs/note-media.md) — how a note's files are shown.
- [Acting on a report](docs/reports.md) — the Reports page: the list, one report, the tools to act and close it.
- [The player page](docs/player-page.md) — an account or an address, and every tool on one player.
- [The map tools](docs/map-tools.md) — reassign a country, paint random tiles, download a replay.
- [Clip stats](docs/clip-stats.md) — the daily numbers of the posted clips, and a CSV of them.
- [Ops](docs/ops.md) — `ReconcileTitles`, after a deploy; Game rules: planet's sets, made and edited; the
  antibot's rule sets, edited as JSON; the rule schedule: which set plays when, a battle's points' times, entries to
  come added and changed; the seasons: their breaks and lengths, added, moved and deleted.
