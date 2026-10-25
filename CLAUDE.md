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
- [Map geometry](docs/map-geometry.md) — `map/`'s tile coordinates and borders blobs, the committed copy in each
  app, and why a regeneration renumbers every tile.
- [Git hooks](docs/git-hooks.md) — `./.githooks/install`, once per clone; what pre-commit, commit-msg and pre-push
  check.
- [Local full stack](docs/local-stack.md) — `deploy/docker-compose.yaml`: backend, frontend and postgres together.
- [Independence of the apps](docs/app-independence.md) — each app builds from its own directory. No root build
  tooling.
