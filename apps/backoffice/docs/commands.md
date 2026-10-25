# Commands

```bash
npm run dev        # Vite on http://localhost:5174: set it as the backend's auth.admin.origin
npm run build      # TypeScript check + Vite bundle → /dist
npm test           # vitest, once (node; a .test.tsx opens with `// @vitest-environment jsdom`)
npm run lint       # ESLint
npm run proto      # Regenerate src/gen/grpc from the shared ../../proto
npm run map        # Copy the tile coordinates from the shared ../../map into src/gen/map
```

`.github/workflows/check-backoffice.yml` runs lint, build and tests on every PR touching this app or `proto/`.
