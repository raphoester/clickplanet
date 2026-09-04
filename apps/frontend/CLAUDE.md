# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
npm run dev      # Start Vite dev server (binds to 0.0.0.0:5173)
npm run build    # TypeScript check + Vite bundle → /dist
npm run lint     # ESLint check
npm run proto    # Regenerate protobuf types from proto/ using buf CLI
```

Docker deployment:
```bash
npm run dBuild   # Build Docker image
npm run dPush    # Push to DigitalOcean registry
```

## Architecture

**Multiplayer globe conquest game** — users click tiles on a 3D globe to claim them for a country.

### Key layers

**Backend abstraction** (`src/backends/`): Three interfaces defined in `backend.ts`:
- `TileClicker` — HTTP POST to claim a tile
- `OwnershipsGetter` — Batch HTTP fetch of tile → country_code mappings
- `UpdatesListener` — WebSocket listener for real-time tile changes

`httpBackend.ts` is the production implementation (targets `https://clickplanet.lol`). `fakeBackend.ts` is a mock for development. The active backend is wired in `App.tsx`.

**3D viewer** (`src/app/viewer/`): Three.js scene rendered via React Three Fiber.
- `effect.ts` — Core scene setup, mouse interactions, and orchestration
- `scene.ts` — Three.js scene/camera/renderer initialization
- `points.ts` — Creates point geometry for all tiles on the globe
- `gpuPicking.ts` — Raycasting using WebGL render target to detect which tile is hovered/clicked
- `pickingColors.ts` — Maps tile IDs to unique RGB colors for GPU picking
- `atlas.ts` — Maps country codes to UV regions in the sprite texture
- `shaders/` — GLSL shaders for rendering tiles with country flag textures
- `leaderboard.ts` — Real-time leaderboard update logic

**UI components** (`src/app/`): Plain React with local `useState`/`useRef`. No global state library. Country selection is persisted to localStorage via `useCountryStorage.ts`.

### Data flow

1. App loads → Three.js scene initialized with all tile points
2. Tile ownerships fetched in batches via `OwnershipsGetter`
3. WebSocket opened via `UpdatesListener` for real-time updates
4. User clicks tile → `TileClicker.click()` → server confirms via WebSocket update
5. Leaderboard updated in real-time from WebSocket events

### Protocol Buffers

Types are defined in `proto/clicks.proto` and generated to `src/gen/grpc/clicks_pb.ts`. Run `npm run proto` after changing `.proto` files. Key messages: `ClickRequest`, `Ownerships`, `TileUpdate`, `OwnershipBatchRequest`.

### Static assets

`/static/coordinates.json` — tile coordinate data loaded at runtime.
`/static/countries/` — country flags and a sprite atlas used for tile rendering.

### Styling

Plain CSS files co-located with components. No CSS preprocessor or CSS-in-JS.
