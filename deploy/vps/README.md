# Production deploy: one DigitalOcean droplet + Cloudflare Pages

Replaces the old DigitalOcean stack (droplet + managed Redis + load balancer +
container registry, ~$50/month). The droplet was never the expensive part — the
managed add-ons were, and none of them are needed. Target cost is **~$6/month**.

| Piece | Where | Cost |
|---|---|---|
| Frontend (static bundle + textures) | Cloudflare Pages | free |
| API + WebSocket (`cmd/api`) | DigitalOcean droplet, Docker Compose | ~$6/mo |
| TLS | Caddy, automatic Let's Encrypt | free |
| Images | GitHub Container Registry | free |
| DNS | Cloudflare | free |

Prices are from late 2025 — check before you commit.

## Why DigitalOcean and not something cheaper

Hetzner is roughly 4x the resources for 60% of the price, and its API is good.
But it has no managed Postgres, Redis, Kubernetes or registry: the moment
another project needs a managed data service, that's a new vendor to stack on.
(This app needs none of them — but the next one might.)
DigitalOcean sits at the deliberate middle — enough managed services to grow
into, few enough to stay legible, an API and console that are already familiar.
The delta is about $3/month, which is noise next to the ~$44/month saved by
deleting the add-ons.

Nothing here is DO-specific, though. The stack is Docker Compose plus a
Caddyfile; it moves to any provider that rents a Linux box.

## Layout

- `docker-compose.yaml` — Caddy + backend. That is the whole stack; there is no database.
- `Caddyfile` — TLS, reverse proxy, CORS, WebSocket passthrough
- `backend.yaml` — API config; secrets come from env, not this file
- `.env.example` — copy to `.env` on the box

`deploy/docker-compose.yaml` (one level up) stays as the *local* full-stack
compose. This directory is only for the production box.

## 1. DNS

Two records on `clickplanet.lol`:

| Name | Type | Value | Proxy |
|---|---|---|---|
| `api` | A | Droplet IPv4 | **DNS only (grey cloud)** |
| `@` / `www` | CNAME | Pages target | Proxied (orange) |

Keep `api` unproxied at first — Caddy needs a direct connection on :80 to issue
its certificate. You can switch it to proxied afterwards if you want Cloudflare
in front of the API too; WebSockets work on the free plan.

## 2. The droplet

A **Basic / Regular $6 droplet** (1 vCPU, 1 GB RAM, 25 GB SSD, 1 TB transfer) is
enough: the Go API holds the whole tile grid in a few MB and there is no
database to run beside it. Images are built in CI and only pulled here, so the
box never needs build headroom. Pick the Ubuntu LTS image and a region
near your players.

`apps/backend/Dockerfile` pins `GOARCH=amd64`, so stay on a regular Intel/AMD
droplet.

```bash
ssh root@YOUR_IP
curl -fsSL https://get.docker.com | sh
adduser --disabled-password --gecos "" deploy && usermod -aG docker deploy
ufw allow OpenSSH && ufw allow 80 && ufw allow 443 && ufw enable
git clone https://github.com/raphoester/clickplanet.git /opt/clickplanet
chown -R deploy:deploy /opt/clickplanet
```

Then as `deploy`:

```bash
cd /opt/clickplanet/deploy/vps
cp .env.example .env && $EDITOR .env   # set API_DOMAIN, FRONTEND_ORIGIN
docker compose up -d
```

Nothing else to provision: no database to start, no script to load, no secret
to set. On first boot the API finds no snapshot and starts from an empty map,
logging `no tile snapshot found`.

Check it:

```bash
curl -sS https://api.clickplanet.lol/v2/rpc/map-density | head -c 200
```

## 3. Frontend on Cloudflare Pages

Create a Pages project from the GitHub repo:

| Setting | Value |
|---|---|
| Root directory | `apps/frontend` |
| Build command | `npm ci && npm run build` |
| Output directory | `dist` |
| Env var | `VITE_API_BASE_URL=https://api.clickplanet.lol` |

Generated protobuf code is committed, so the build needs no `buf`.

`npm run build` now also copies `static/` into `dist/static/`, skipping the
24 MB `coordinates.json` — that file is bundled into the JS at build time and
never fetched at runtime, so uploading it would waste a quarter of the deploy.

**Watch the bundle size.** `coordinates.json` is imported into the main chunk,
which currently builds to **23.5 MiB against Cloudflare's 25 MiB per-file
limit**. There is ~1.5 MiB of headroom. If a dependency or a denser tile map
pushes it over, Pages will reject the upload — the fix is to load the
coordinates as a binary `Float32Array` fetched at runtime (~5 MB instead of
25 MB, and no JSON parse), which means making the viewer's point setup async.

## 4. CI and the image registry

`.github/workflows/deploy-backend.yml` builds the image to GHCR and rolls the
container over SSH. GHCR rather than a dedicated registry because it adds no
account, no vendor and no bill: the code is already on GitHub, CI already runs
there, and the push authenticates with the built-in `GITHUB_TOKEN`.

Repository secrets required:

- `VPS_HOST` — the droplet's IP
- `VPS_USER` — `deploy`
- `VPS_SSH_KEY` — private key whose public half is in `deploy`'s `authorized_keys`

**One-time after the first successful build:** a new GHCR package is created
private even when the repo is public. Open
`https://github.com/users/raphoester/packages/container/clickplanet-backend/settings`,
set visibility to **Public**, and link it to the repo. The droplet can then
`docker pull` anonymously — no registry credentials on the box at all. If you'd
rather keep the package private, generate a read-only PAT with `read:packages`
and run `docker login ghcr.io` once as `deploy`.

Pages deploys itself on push; no workflow needed.

## 5. Backups

The whole game state is one snapshot file in the `tile_state` volume, written
every 30s and on every clean shutdown. A nightly cron on the box is enough:

```bash
0 4 * * * docker run --rm -v vps_tile_state:/state -v /home/deploy/backups:/out alpine \
  tar czf /out/tiles-$(date +\%F).tar.gz -C /state .
```

DigitalOcean's droplet backups (+20% of the droplet price, so ~$1.20/mo) cover
the whole disk if you would rather not think about it.

## Rollback

- **Bad backend build:** `BACKEND_IMAGE=ghcr.io/raphoester/clickplanet-backend:<sha>` in `.env`, then `docker compose up -d backend`.
- **Lost or corrupt tile state:** stop the backend, drop the newest backup's `tiles.snapshot` into the `tile_state` volume, start it again. A snapshot the API cannot parse is not fatal — it logs and starts from an empty map, so a bad file degrades to a reset rather than a crash loop.
- **In-process storage misbehaving:** there is no config switch back to Redis — that code is gone. Roll the backend image back to a pre-migration `<sha>` and restore the matching Redis stack from git history.
- **Frontend:** roll back the deployment in the Pages dashboard.
