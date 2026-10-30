# Deploy

**A Cloudflare Worker with static assets, `clickplanet-backoffice`** (`wrangler.jsonc`), like the game's. Its
custom domain `backoffice.clickplanet.lol` is set on the Worker in the dashboard, not in `wrangler.jsonc`: a route
there would make the DNS record on the first deploy.

**Workers Builds deploys it from `main`**, on a push that changes something under `apps/backoffice/`: `npm ci && npm
run build`, then `npx wrangler deploy`, with `NODE_VERSION` 22 and `VITE_API_BASE_URL` set on the build. A change
to `proto/` alone ships nothing: the generated code is committed.

`public/_headers` is the page's security:

- **`Content-Security-Policy`**: scripts, styles and images from the page alone, calls to
  `https://api.clickplanet.lol`, the bucket (a video's upload, see [Videos](videos.md)) and the page's own `/assets/`
  (the tile coordinates) alone, and no frame (`frame-ancestors 'none'`): the pages ban, revert and repaint
  the map, and a framed page could be clicked through. **A new API host is a change here.**
- **The support bucket's host is in `img-src` and `media-src`**, and no other: a note's photos and videos load
  straight from the R2 link `ListNotes` signs (`support.attachments.bucket.endpoint` in `deploy/vps/backend.yaml`).
  **Another bucket is a change here too**, or every attachment shows "not shown here".
- `Referrer-Policy: no-referrer` and `X-Robots-Tag: noindex`.
- `/assets/*` is immutable, `/index.html` is revalidated.
