# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Structure

This is a monorepo with two independent apps, each with its own `CLAUDE.md` for app-specific commands and architecture:

- [`apps/frontend/CLAUDE.md`](apps/frontend/CLAUDE.md) — React/Three.js client (npm)
- [`apps/backend/CLAUDE.md`](apps/backend/CLAUDE.md) — Go API + bookkeeper (Go modules)

Read the relevant app's `CLAUDE.md` before working inside `apps/frontend/` or `apps/backend/` — this file only covers what's shared across both.

## Shared protobuf contract

`proto/clicks/v1/clicks.proto` is the **single source of truth** for the API contract — both apps generate their own bindings from it (nothing here is hand-copied between apps). After editing it:

```bash
cd apps/backend && make proto   # regenerates apps/backend/generated/proto
cd apps/frontend && npm run proto   # regenerates apps/frontend/src/gen/grpc
```

Both apps' `buf.gen.yaml` reference `../../proto` (or `../../../proto` for the backend, whose config lives one level deeper at `apps/backend/proto/`) — do not create per-app copies of the `.proto` file again.

## Local full stack

`deploy/docker-compose.yaml` runs redis + backend + frontend together using locally built Docker images. Build each app's image first (`apps/frontend`'s `npm run dBuild`, `apps/backend`'s `make dBuild`), then `cd deploy && docker-compose up`.

## Independence of the two apps

Each app under `apps/` keeps its own dependency manifest (`package.json` / `go.mod`) and is built from its own directory as the Docker build context — nothing at the repo root is required to build or run either app in isolation. Don't introduce root-level build tooling (Nx/Turborepo/etc.) unless the apps actually start sharing more than the proto contract.
