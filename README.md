# ClickPlanet

Monorepo for [clickplanet.lol](https://clickplanet.lol), a real-time multiplayer game where countries compete to own
tiles on a 3D globe.

## Layout

```
apps/
├── frontend/    # the game: React + Three.js, a Cloudflare Worker
├── backend/     # the Go API: one process, postgres and Redis
└── backoffice/  # the operator's page at backoffice.clickplanet.lol
proto/           # the API contract, one package per bounded context
map/             # the tile coordinates and borders blobs, copied into each app
quiz/            # the quiz bank, embedded in the backend
docs/            # what the apps share
deploy/          # the local full stack, and the production box (deploy/vps)
```

Each app builds from its own directory. Its `CLAUDE.md` indexes its docs:
[frontend](apps/frontend/CLAUDE.md), [backend](apps/backend/CLAUDE.md), [backoffice](apps/backoffice/CLAUDE.md).
The shared docs are indexed by the root [`CLAUDE.md`](CLAUDE.md): the [protobuf contract](docs/proto.md) and how to
regenerate each app's bindings, the [local full stack](docs/local-stack.md), the [git hooks](docs/git-hooks.md).
