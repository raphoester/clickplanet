# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Structure

This is a monorepo with three independent apps, each with its own `CLAUDE.md` for app-specific commands and architecture:

- [`apps/frontend/CLAUDE.md`](apps/frontend/CLAUDE.md) — React/Three.js client (npm)
- [`apps/backend/CLAUDE.md`](apps/backend/CLAUDE.md) — Go API (Go modules)
- [`apps/backoffice/CLAUDE.md`](apps/backoffice/CLAUDE.md) — the operator's page at `backoffice.clickplanet.lol`, React (npm)

Read the relevant app's `CLAUDE.md` before working inside `apps/` — this file only covers what's shared.

## Shared protobuf contract

`proto/` is the **single source of truth** for the API contract — every app generates its own bindings from it (nothing here is hand-copied between apps). One package per bounded context:

- `proto/planet/v1/planet.proto` — the tile game (`ClickService`)
- `proto/chat/v1/chat.proto` — the live chat (`ChatService`)
- `proto/auth/v1/auth.proto` — who a caller is (`AuthService`): the mint that gates `Click`, the identity token resumed from the cookie, the caller's account, sign-in with Google, Discord or an emailed code, sign-out and deletion
- `proto/session/v1/session.proto` — the deprecated mint (`SessionService`), served by the same auth module until no client calls it
- `proto/player/v1/player.proto` — a player's name, color, stats, titles and the countries it plays for and against, and who is playing (`PlayerService`); the colors themselves are `NameColor` in `color.proto`, and a title is `Title` in `title.proto`, which `chat.proto` and `seasons.proto` import too
- `proto/seasons/v1/seasons.proto` — when the current season ends and its finale starts, who leads it, and the countries' daily rounds (`SeasonService`)
- `proto/support/v1/support.proto` — what a player sends the operator: a feature suggestion or a bug report, with photos or videos, or a report of another player (`SupportService`)
- `proto/clips/v1/clips.proto` — what the frontend's clip generator hands the backend's job that posts its clips: the stories worth a clip with the facts its model editor judges them by, and what to post with each one. No service: both are written as JSON files in one workflow run. Beside it, `admin.proto` is the numbers of the clips the job posted, a snapshot a day (`clips.v1.AdminService`): the job's `clips stats` keeps them over ssh, the backoffice reads them and makes the table

Beside them, `internal.proto` is what one backend module asks another, or `auth`'s admin gate asks `auth` (`auth.v1`, `player.v1`, `planet.v1`), and `events.proto` is what one tells the others in process (`planet.v1.TileTaken`, `auth.v1.AccountDeleted`). Neither is on the public router; the frontend generates both without using them. `admin.proto` is what an operator calls (`AdminService`, in `auth.v1`, `chat.v1`, `clips.v1`, `planet.v1`, `player.v1` and `support.v1`): over SSH on the console listener, and from the admin page on the admin listener, which only a signed-in operator passes.

Connect derives each service's route from its proto package, so a new context gets its own path with no prefix to allocate. Every `buf.gen.yaml` input points at the whole `proto` directory, so a new package is picked up by each generator with no config change.

After editing a `.proto`:

```bash
cd apps/backend && make proto   # regenerates apps/backend/generated/proto
cd apps/frontend && npm run proto   # regenerates apps/frontend/src/gen/grpc
cd apps/backoffice && npm run proto   # regenerates apps/backoffice/src/gen/grpc
```

Every app's `buf.gen.yaml` reference `../../proto` (or `../../../proto` for the backend, whose config lives one level deeper at `apps/backend/proto/`) — do not create per-app copies of the `.proto` files again.

## Shared quiz bank

`quiz/` is the **third and last thing the root holds**, and the one that is shared with only *one*
app. `bank.json` is every question the quizzes can ask and the answer to each; `make quiz`
copies it into `apps/backend/generated/quiz/`, where it is embedded.

**The frontend deliberately has no copy**, because the answers are in it: a bank served to the page
is a bank anyone can fetch with the network tab open. The question and its three choices reach the
client one at a time from `OpenQuiz`, and which of the three is right never leaves the server.

```bash
cd apps/frontend && npm run quiz:generate   # rewrite /quiz/bank.json
cd apps/backend && make quiz                # → apps/backend/generated/quiz (embedded)
```

About 820 of the questions are **derived** from the same pinned Natural Earth snapshot `map/` is cut
from and from the tile borders themselves, so they are regenerated rather than corrected; the rest
are hand-written in `apps/frontend/scripts/quiz/extra.json` for what no dataset here can answer. See
[`quiz/README.md`](quiz/README.md), which also has the rule that keeps the hand-written half from
rotting.

## Shared map geometry

`map/` is the **second thing the frontend and the backend share**, and the only other one (the backoffice carries
the coordinates too, to draw where tiles changed hands): the tile coordinates blob,
which says where every tile is and therefore what a tile id means, and the borders blob, which says
which country's ground each tile sits on. **They are written by one command from one oracle**, so
"is this sea or land" and "which country is this" cannot disagree and a tile in no country cannot be
built. See [`map/README.md`](map/README.md).

It follows the `proto/` precedent rather than inventing a second pattern. The root holds the source;
each app carries a **committed, byte-identical copy** generated by a target, because each app's
Docker build context is its own directory and neither can read the repo root at build time:

```bash
cd apps/frontend && npm run map:generate   # rewrite both blobs in /map from Natural Earth
cd apps/backend && make map    # → apps/backend/generated/map (embedded)
cd apps/frontend && npm run map   # → apps/frontend/static (fetched at runtime)
cd apps/backoffice && npm run map   # → apps/backoffice/src/gen/map (the coordinates alone, bundled)
```

Each file is content-addressed, so all three copies carry the same name or one of them is stale.
The backend rebuilds the tile adjacency from its copy at every boot and **refuses to start** if it
does not match the lattice or `gameMap.maxIndex` — drift here silently moves every player's
territory, so it is worth a crash.

**Regenerating renumbers every tile**, since an id is an index into the coordinates blob. Positions
do not move, so `npm run map:generate -- --remap <path.up.sql>` writes the postgres migration that
follows every owned tile across; `map/README.md` has the checklist.

## Git hooks

`./.githooks/install` points git at [`.githooks/`](.githooks), once per clone.
pre-commit formats (gofumpt, via `make tidy`) and lints whichever app has staged
changes, commit-msg enforces conventional commits, and pre-push runs the
backend's tests, dead-code check, linter and format check concurrently, and checks
that the backend's copy of the quiz bank is the one in `quiz/`. All three
take `--no-verify`.

It is a script rather than a root `Makefile` target on purpose — see
[Independence of the apps](#independence-of-the-apps).

## Local full stack

`deploy/docker-compose.yaml` runs backend + frontend together using locally built Docker images. Build each app's image first (`apps/frontend`'s `npm run dBuild`, `apps/backend`'s `make dBuild`), then `cd deploy && docker compose up`.

The stack includes a **postgres**: the backend keeps the whole tile map in process and writes what changed to postgres every second. The ledger and the bonus charges each account holds go there too, the antibot keeps its evidence there in its own schema and reads and writes its bans there directly, the chat reads and writes its messages, reactions and when each account last saw it there directly, and broadcasts a write only once it is kept, the player module reads and writes profiles and stats there on every call, the seasons module keeps each season's standings and its rounds there, and the support module keeps what players send the operator there. The photos and videos a player attaches are the one thing kept elsewhere, in an S3 bucket, and they are off in this stack. See [`apps/backend/CLAUDE.md`](apps/backend/CLAUDE.md) for the durability tradeoff and the full config schema.

## Independence of the apps

Each app under `apps/` keeps its own dependency manifest (`package.json` / `go.mod`) and is built from its own directory as the Docker or Workers build context — nothing at the repo root is required to build or run any app in isolation. **`apps/backoffice` imports nothing from `apps/frontend`**, though both are React: it is a third app with its own Worker, not a page of the game. That is why `proto/`, `map/` and `quiz/` all generate a committed copy into the app that needs them instead of being read from the root at build time. Don't introduce root-level build tooling (Nx/Turborepo/etc.) unless the apps actually start sharing more than the proto contract, the map geometry and the quiz bank.
