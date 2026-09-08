# ClickPlanet

Monorepo for [clickplanet.lol](https://clickplanet.lol), a real-time multiplayer game where countries compete to own tiles on a 3D globe.

## Layout

```
apps/
├── frontend/   # React + Three.js client — see apps/frontend/README.md
└── backend/    # Go API — see apps/backend/README.md
proto/          # Shared protobuf contract (single source of truth for both apps)
deploy/         # Local full-stack docker-compose (frontend + backend)
```

Previously three separate repos (`clickplanet.lol-frontend`, `clickplanet.lol-backend`, `clickplanet.lol-proxy`); merged here to stop hand-syncing the protobuf contract between frontend and backend, which had drifted apart before. Full history of both apps is preserved under `apps/`.

## Getting started

Each app is self-contained and built independently — see its own README for day-to-day commands:

- [`apps/frontend`](apps/frontend/README.md)
- [`apps/backend`](apps/backend/README.md)

### Shared protobuf contract

`proto/planet/v1/planet.proto` is the single source of truth for the API contract. After editing it, regenerate both sides:

```bash
cd apps/backend && make proto
cd apps/frontend && npm run proto
```

### Full local stack

`deploy/docker-compose.yaml` runs frontend + backend together using locally built Docker images (`clickplanet-back:local`, `registry.digitalocean.com/clickplanet-frontend/frontend:latest` — build them first via each app's `dBuild` script/target):

```bash
cd deploy && docker compose up
```

There is no database: the backend holds the tile map in process and snapshots it to the `tile_state` volume.
