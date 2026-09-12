# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Structure

This is a monorepo with two independent apps, each with its own `CLAUDE.md` for app-specific commands and architecture:

- [`apps/frontend/CLAUDE.md`](apps/frontend/CLAUDE.md) — React/Three.js client (npm)
- [`apps/backend/CLAUDE.md`](apps/backend/CLAUDE.md) — Go API (Go modules)

Read the relevant app's `CLAUDE.md` before working inside `apps/frontend/` or `apps/backend/` — this file only covers what's shared across both.

## Shared protobuf contract

`proto/` is the **single source of truth** for the API contract — both apps generate their own bindings from it (nothing here is hand-copied between apps). One package per bounded context:

- `proto/planet/v1/planet.proto` — the tile game (`ClickService`)
- `proto/chat/v1/chat.proto` — the live chat (`ChatService`)
- `proto/session/v1/session.proto` — the session mint (`SessionService`), which gates `Click`

Connect derives each service's route from its proto package, so a new context gets its own path with no prefix to allocate. Both `buf.gen.yaml` inputs point at the whole `proto` directory, so a new package is picked up by either generator with no config change.

After editing a `.proto`:

```bash
cd apps/backend && make proto   # regenerates apps/backend/generated/proto
cd apps/frontend && npm run proto   # regenerates apps/frontend/src/gen/grpc
```

Both apps' `buf.gen.yaml` reference `../../proto` (or `../../../proto` for the backend, whose config lives one level deeper at `apps/backend/proto/`) — do not create per-app copies of the `.proto` files again.

## Git hooks

`./.githooks/install` points git at [`.githooks/`](.githooks), once per clone.
pre-commit formats and lints whichever app has staged changes, commit-msg
enforces conventional commits, and pre-push runs the backend's tests, dead-code
check and linter concurrently. All three take `--no-verify`.

It is a script rather than a root `Makefile` target on purpose — see
[Independence of the two apps](#independence-of-the-two-apps).

## Local full stack

`deploy/docker-compose.yaml` runs backend + frontend together using locally built Docker images. Build each app's image first (`apps/frontend`'s `npm run dBuild`, `apps/backend`'s `make dBuild`), then `cd deploy && docker compose up`.

There is **no database**: the backend keeps the whole tile map in process and snapshots it to the `tile_state` volume, which is the only thing worth backing up. See [`apps/backend/CLAUDE.md`](apps/backend/CLAUDE.md) for the durability tradeoff and the full config schema.

## Independence of the two apps

Each app under `apps/` keeps its own dependency manifest (`package.json` / `go.mod`) and is built from its own directory as the Docker build context — nothing at the repo root is required to build or run either app in isolation. Don't introduce root-level build tooling (Nx/Turborepo/etc.) unless the apps actually start sharing more than the proto contract.
