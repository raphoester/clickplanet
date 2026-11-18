# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Docs rule

**A `CLAUDE.md` holds only a summary and links.** The details live in a `docs/` folder beside it: [`docs/`](docs)
here for what the apps share, `apps/<app>/docs/` for each app. Read the doc of the area you touch before you
change it. When you change how something works, update its doc. When you add a topic, write a new doc and add
one line for it to the `CLAUDE.md` that indexes it. Never write the details into a `CLAUDE.md`.

## Structure

A monorepo with three independent apps. Each app's `CLAUDE.md` is the index of its docs:

- [`apps/frontend/CLAUDE.md`](apps/frontend/CLAUDE.md) — React/Three.js client (npm)
- [`apps/backend/CLAUDE.md`](apps/backend/CLAUDE.md) — Go API (Go modules)
- [`apps/backoffice/CLAUDE.md`](apps/backoffice/CLAUDE.md) — the operator's page at `backoffice.clickplanet.lol`, React (npm)

Read the relevant app's `CLAUDE.md` before working inside `apps/` — this file only covers what's shared.

## Shared docs

- [Protobuf contract](docs/proto.md) — `proto/` is the one source of the API, one package per bounded context.
  How to regenerate each app's bindings after a `.proto` edit.
- [Quiz bank](docs/quiz-bank.md) — `quiz/bank.json`, embedded in the backend only. The frontend has no copy,
  because the answers are in it.
- [Map geometry](docs/map-geometry.md) — `map/boards/`: `earth-1`, `earth-2` and `siege-1`, each board's runs, borders,
  capitals, bridges and terrain, no coordinates file, the committed copy in each app, the lakes, each board's one-tile
  seats of the countries too small for its lattice, and why a regeneration renumbers every tile.
- [Boards](docs/boards.md) — PRs 1 to 6 built (`earth-1` is a board, no coordinates file; every tile and event names
  its board and map, and the boot refuses another board; every app holds every board, the page reloads on another,
  thresholds read the board, `arena.board`, each era drawn on its board with the cut; the `fixture-1` test board;
  `earth-2` at 1,004,682 tiles and `siege-1` at 27,259, the warped and rotated lattice with its pentagons at sea, fitted
  per board, in canonical order, measured; Capital Siege on `siege-1` with `charges.spread.off` and `charges.rally.off`,
  its sets saved by the owner; the world's switch to `earth-2`: its carry-over, `SwitchBoard`, the moment, its
  rehearsal on production's map and the operator's runbook; the world plays `earth-2` since 2026-11-18), the rest
  designed: the rules at the new density, and the PRs in order.
- [Provinces](docs/provinces.md) — designed, not built: fortify on Natural Earth's admin-1 units, joined under N tiles
  (200 on `earth-2`, 20 on `siege-1`) inside one landmass; the misses, the names and their overrides, the board's new
  file, the locks' switch, the Countries / Provinces setting, straight outlines with grey province lines, the PRs.
- [Git hooks](docs/git-hooks.md) — `./.githooks/install`, once per clone; what pre-commit, commit-msg and pre-push
  check.
- [Local full stack](docs/local-stack.md) — `deploy/docker-compose.yaml`: backend, frontend and postgres together.
- [A server restart reloads the page](docs/reload-on-restart.md) — the boot id on every answer, the page that reloads
  when it changes, why a wire change can be a hard switch, and the order of the deploys.
- [Independence of the apps](docs/app-independence.md) — each app builds from its own directory. No root build
  tooling.
- [CrazyGames](docs/crazygames.md) — the game framed from `crazygames.clickplanet.lol`: why a host of its own, what
  each app and the deploy do, and what the owner sets up.
- [Chat moderation](docs/moderation.md) — the three layers (banned phrases at send, raid mode for guests, a Claude Haiku
  judge of reports alone) and the daily Claude audit; the policy the judge sends Claude and the audit reads: what stays,
  what goes, the mute steps capped at 24 hours and counted from the moderation history, usernames, reports, what goes
  to the operator, and what is not built yet.
